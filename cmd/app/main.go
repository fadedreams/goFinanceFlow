package main

import (
	"context"
	"fmt"
	"github.com/fadedreams/gofinanceflow/business/tasks"
	"github.com/fadedreams/gofinanceflow/cmd/api"
	"github.com/fadedreams/gofinanceflow/cmd/grpc_api"
	sdk "github.com/fadedreams/gofinanceflow/foundation/sdk"
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/fadedreams/gofinanceflow/infrastructure/pb"
	"github.com/fadedreams/gosafecircuit"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type serverOptions struct {
	logger      *zap.Logger
	queries     *db.Queries
	pool        *pgxpool.Pool
	taskManager tasks.TaskManager
}

type serverOption func(*serverOptions)

func withLogger(logger *zap.Logger) serverOption {
	return func(o *serverOptions) {
		o.logger = logger
	}
}

func withQueries(queries *db.Queries, pool *pgxpool.Pool) serverOption {
	return func(o *serverOptions) {
		o.queries = queries
		o.pool = pool
	}
}

func withTaskManager(taskManager tasks.TaskManager) serverOption {
	return func(o *serverOptions) {
		o.taskManager = taskManager
	}
}
func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer logger.Sync() // flushes buffer, if any

	config, err := sdk.LoadConfig(".")
	if err != nil {
		log.Fatalf("cannot load config: %v\n", err)
	}

	// redis
	redis := asynq.RedisClientOpt{
		Addr: config.RedisAddress,
	}

	taskManager := tasks.NewTaskManager(redis)

	// Enqueue an email delivery task
	// err = taskManager.EnqueueEmailDeliveryTask(123, "welcome-template")
	// if err != nil {
	// 	log.Fatalf("could not enqueue task: %v", err)
	// }

	// Run the task manager to process tasks
	// if err := taskManager.Run(); err != nil {
	// 	log.Fatalf("could not run server: %v", err)
	// }

	// connStr := "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable"
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create a connection pool
	// pool, err := pgxpool.New(ctx, connStr)
	// pool, err := pgxpool.New(ctx, config.DBSource)
	// if err != nil {
	// 	log.Fatalf("Unable to connect to database: %v\n", err)
	// }
	// defer pool.Close()

	pool, err := connectToDatabase(ctx, &config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Create a new Queries instance
	queries := db.New(pool)
	// Define options for servers
	// options := serverOptions{
	// 	logger:      logger,
	// 	queries:     queries,
	// 	pool:        pool,
	// 	taskManager: taskManager,
	// }
	options := serverOptions{}
	for _, opt := range []serverOption{
		withLogger(logger),
		withQueries(queries, pool),
		withTaskManager(taskManager),
	} {
		opt(&options)
	}

	// Use an errgroup to manage multiple concurrent tasks
	var g errgroup.Group
	// Start task manager to process tasks
	g.Go(func() error {
		return runTaskManager(ctx, options.logger, options.taskManager)
	})

	// Start gRPC server with interceptor
	g.Go(func() error {
		return runGrpcServer(ctx, options.queries, options.pool, options.logger)
	})

	// Optionally, start an HTTP server concurrently
	g.Go(func() error {
		return runHTTPServer(ctx, options.queries, options.pool, options.taskManager, options.logger)
	})

	// Wait for all servers to exit
	if err := g.Wait(); err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
}

func runHTTPServer(ctx context.Context, queries *db.Queries, pool *pgxpool.Pool, taskManager tasks.TaskManager, logger *zap.Logger) error {
	// Since api.Server doesn't have a Handler method, assume api.NewServer returns an http.Handler directly
	server := api.NewServer(queries, pool, taskManager)

	// Start the HTTP server
	address := ":8080"
	logger.Info("Starting HTTP server on %s\n", zap.String("address", address))

	if err := server.Start(address); err != nil {
		logger.Error("Failed to start server: %v\n", zap.Error(err))
	}

	go func() {
		<-ctx.Done()
		logger.Info("Shutting down HTTP server...")
	}()
	return nil

}

func runTaskManager(ctx context.Context, logger *zap.Logger, taskManager tasks.TaskManager) error {
	// Log start of task manager
	logger.Info("Starting task manager...")

	// Run the task manager to process tasks
	err := taskManager.Run()
	if err != nil {
		logger.Error("Task manager failed to run", zap.Error(err))
	}

	// Monitor context for shutdown signal
	select {
	case <-ctx.Done():
		logger.Info("Shutting down task manager...")
	}

	return err
}

func runGrpcServer(ctx context.Context, queries *db.Queries, pool *pgxpool.Pool, logger *zap.Logger) error {
	// Initialize the gRPC server with interceptor
	grpcServer := grpc.NewServer(
		// grpc.UnaryInterceptor(authInterceptor),
		grpc.UnaryInterceptor(chainUnaryInterceptors(loggingInterceptor(logger), authInterceptor)),
	)
	server := grpc_api.NewServer(queries, pool)

	// Register your gRPC service
	pb.RegisterFinanceFlowServer(grpcServer, server)

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	// Start the gRPC server
	address := ":9090"
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	logger.Info("Starting gRPC server on %s\n", zap.String("address", address))
	go func() {
		<-ctx.Done()
		logger.Error("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(listener)
}

func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Check if the method being invoked is GetUser
	if info.FullMethod == "/pb.FinanceFlow/GetUser" {
		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, fmt.Errorf("missing metadata")
		}

		// Get the authorization header
		authHeader, ok := md["authorization"]
		if !ok || len(authHeader) == 0 {
			return nil, fmt.Errorf("missing authorization token")
		}

		// Extract the token from the "Bearer" scheme
		token := strings.TrimPrefix(authHeader[0], "Bearer ")
		if token == authHeader[0] { // If the token is not prefixed with "Bearer "
			return nil, fmt.Errorf("invalid authorization token")
		}

		// Verify the token
		_, err := sdk.VerifyToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid token: %v", err)
		}
	}

	// Call the next handler in the chain
	return handler(ctx, req)
}

// func loggingInterceptor(
// 	ctx context.Context,
// 	req interface{},
// 	info *grpc.UnaryServerInfo,
// 	handler grpc.UnaryHandler,
// ) (interface{}, error) {
// 	start := time.Now()
// 	h, err := handler(ctx, req)
// 	log.Printf("Method: %s, Duration: %s, Error: %v", info.FullMethod, time.Since(start), err)
// 	return h, err
// }

func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		h, err := handler(ctx, req)
		duration := time.Since(start)
		if err != nil {
			logger.Error("gRPC call failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			logger.Info("gRPC call succeeded",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
			)
		}
		return h, err
	}
}

// chainUnaryInterceptors chains multiple unary interceptors into a single interceptor
func chainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		chain := func(currentInter grpc.UnaryServerInterceptor, currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return currentInter(currentCtx, currentReq, info, currentHandler)
			}
		}
		h := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			h = chain(interceptors[i], h)
		}
		return h(ctx, req)
	}
}

func connectToDatabase(ctx context.Context, config *sdk.Config) (*pgxpool.Pool, error) {
	// Create a new circuit breaker with maxFailures=3, timeout=5 seconds, pauseTime=1 second, and maxConsecutiveSuccesses=2
	cb := gosafecircuit.NewCircuitBreaker(3, 5*time.Second, 1*time.Second, 2)

	// Set up the callbacks
	cb.SetOnOpen(func() {
		fmt.Println("Circuit breaker opened!")
	})

	cb.SetOnClose(func() {
		fmt.Println("Circuit breaker closed!")
	})

	cb.SetOnHalfOpen(func() {
		fmt.Println("Circuit breaker is half-open, trying again...")
	})

	// Execute the connection logic using the circuit breaker
	var pool *pgxpool.Pool
	var err error
	err = cb.Execute(func() error {
		pool, err = pgxpool.New(ctx, config.DBSource)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return pool, nil
}

////for all
// func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
// 	// Extract metadata from context
// 	md, ok := metadata.FromIncomingContext(ctx)
// 	if !ok {
// 		return nil, fmt.Errorf("missing metadata")
// 	}
//
// 	// Get the authorization header
// 	authHeader, ok := md["authorization"]
// 	if !ok || len(authHeader) == 0 {
// 		return nil, fmt.Errorf("missing authorization token")
// 	}
//
// 	// Extract the token from the "Bearer" scheme
// 	token := strings.TrimPrefix(authHeader[0], "Bearer ")
// 	if token == authHeader[0] { // If the token is not prefixed with "Bearer "
// 		return nil, fmt.Errorf("invalid authorization token")
// 	}
//
// 	// Verify the token
// 	_, err := sdk.VerifyToken(token)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid token: %v", err)
// 	}
//
// 	// Call the next handler in the chain
// 	return handler(ctx, req)
// }
