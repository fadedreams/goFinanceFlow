// cmd/app/main.go

package main

import (
	"context"
	"log"

	"github.com/fadedreams/gofinanceflow/cmd/api"
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	connStr := "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable"
	ctx := context.Background()

	// Create a connection pool
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Create a new Queries instance
	queries := db.New(pool)

	// Initialize the Echo server with both queries and pool
	server := api.NewServer(queries, pool)

	// Start the server
	address := ":8080"
	log.Printf("Starting server on %s\n", address)
	if err := server.Start(address); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
