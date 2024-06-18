package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"

	"github.com/fadedreams/gofinanceflow/cmd/api"
	"github.com/fadedreams/gofinanceflow/cmd/grpc_api"
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/fadedreams/gofinanceflow/infrastructure/pb"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	connStr := "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable"
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create a connection pool
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Create a new Queries instance
	queries := db.New(pool)

	// Use an errgroup to manage multiple concurrent tasks
	var g errgroup.Group

	// Start gRPC server
	g.Go(func() error {
		return runGrpcServer(ctx, queries, pool)
	})

	// Optionally, start an HTTP server concurrently
	// Example: Uncomment and implement as needed
	// g.Go(func() error {
	// 	return runHTTPServer(ctx, queries, pool)
	// })

	// Wait for all servers to exit
	if err := g.Wait(); err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
}

func runGrpcServer(ctx context.Context, queries *db.Queries, pool *pgxpool.Pool) error {
	// Initialize the gRPC server
	grpcServer := grpc.NewServer()
	server := grpc_api.NewServer(queries, pool)

	// Register your gRPC service
	// Assuming pb.RegisterFinanceFlowServer registers your service
	pb.RegisterFinanceFlowServer(grpcServer, server)

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	// Start the gRPC server
	address := ":9090"
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	log.Printf("Starting gRPC server on %s\n", address)
	go func() {
		<-ctx.Done()
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(listener)
}

func runHTTPServer(ctx context.Context, queries *db.Queries, pool *pgxpool.Pool) error {
	// Since api.Server doesn't have a Handler method, assume api.NewServer returns an http.Handler directly
	server := api.NewServer(queries, pool)

	// Start the HTTP server
	address := ":8080"
	log.Printf("Starting HTTP server on %s\n", address)

	if err := server.Start(address); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}

	go func() {
		<-ctx.Done()
		log.Println("Shutting down HTTP server...")
	}()
	return nil

}
