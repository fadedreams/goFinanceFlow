package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fadedreams/gofinanceflow/cmd/grpc_api"
	"github.com/fadedreams/gofinanceflow/foundation/sdk"
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/fadedreams/gofinanceflow/infrastructure/pb"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

	// Start gRPC server with interceptor
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
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer logger.Sync() // flushes buffer, if any
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

	log.Printf("Starting gRPC server on %s\n", address)
	go func() {
		<-ctx.Done()
		log.Println("Shutting down gRPC server...")
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
