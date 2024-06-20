package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/hibiken/asynq"
)

const (
	TypeEmailDelivery = "email:deliver"
)

// EventEmitter defines the methods for an event emitter.
type EventEmitter interface {
	On(event string, listener func(ctx context.Context, payload []byte) error)
	Emit(event string, ctx context.Context, payload []byte) error
}

// SimpleEventEmitter is a simple implementation of EventEmitter.
type SimpleEventEmitter struct {
	listeners map[string][]func(ctx context.Context, payload []byte) error
	mu        sync.RWMutex
}

// NewSimpleEventEmitter creates a new SimpleEventEmitter.
func NewSimpleEventEmitter() *SimpleEventEmitter {
	return &SimpleEventEmitter{
		listeners: make(map[string][]func(ctx context.Context, payload []byte) error),
	}
}

// On registers a listener for a specific event.
func (e *SimpleEventEmitter) On(event string, listener func(ctx context.Context, payload []byte) error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listeners[event] = append(e.listeners[event], listener)
}

// Emit emits an event to all registered listeners.
func (e *SimpleEventEmitter) Emit(event string, ctx context.Context, payload []byte) error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, listener := range e.listeners[event] {
		if err := listener(ctx, payload); err != nil {
			return err
		}
	}
	return nil
}

// TaskManager defines the methods for managing tasks.
type TaskManager interface {
	EnqueueEmailDeliveryTask(userID int, tmplID string) error
	Run() error
}

// taskManager is a concrete implementation of TaskManager.
type taskManager struct {
	client       *asynq.Client
	server       *asynq.Server
	eventEmitter EventEmitter
}

// NewTaskManager creates a new TaskManager with the given Redis options.
func NewTaskManager(redisOpt asynq.RedisClientOpt) TaskManager {
	client := asynq.NewClient(redisOpt)
	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
	})

	eventEmitter := NewSimpleEventEmitter()
	tm := &taskManager{
		client:       client,
		server:       server,
		eventEmitter: eventEmitter,
	}

	// Register default task handlers
	tm.eventEmitter.On(TypeEmailDelivery, HandleEmailDeliveryTask)

	return tm
}

// EmailDeliveryPayload defines the payload for email delivery tasks.
type EmailDeliveryPayload struct {
	UserID     int
	TemplateID string
}

// EnqueueEmailDeliveryTask enqueues a task to deliver an email.
func (tm *taskManager) EnqueueEmailDeliveryTask(userID int, tmplID string) error {
	payload, err := json.Marshal(EmailDeliveryPayload{UserID: userID, TemplateID: tmplID})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeEmailDelivery, payload)
	_, err = tm.client.Enqueue(task)
	return err
}

// HandleEmailDeliveryTask handles the email delivery task.
func HandleEmailDeliveryTask(ctx context.Context, payload []byte) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	log.Printf("Sending Email to User: user_id=%d, template_id=%s", p.UserID, p.TemplateID)
	// Email delivery code ...
	return nil
}

// Run starts the asynq server to process tasks.
func (tm *taskManager) Run() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeEmailDelivery, func(ctx context.Context, t *asynq.Task) error {
		return tm.eventEmitter.Emit(TypeEmailDelivery, ctx, t.Payload())
	})
	return tm.server.Run(mux)
}
