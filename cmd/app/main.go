package main

import (
	"context"
	"log"

	db "github.com/fadedreams/gofinanceflow/foundation/db/sqlc"
	// "github.com/jackc/pgx/v5"
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

	// Use queries to interact with the database
	// For example, to fetch an account:
	user, err := queries.GetUser(ctx, "u1") // assuming 1 is the account ID you want to query
	if err != nil {
		log.Fatalf("GetAccount failed: %v\n", err)
	}

	log.Printf("GetUser: %+v\n", user.Username)
}

// func main() {
// 	// Define the connection URL
// 	connStr := "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable"
//
// 	// Create a context
// 	ctx := context.Background()
//
// 	// Configure the connection pool
// 	config, err := pgxpool.ParseConfig(connStr)
// 	if err != nil {
// 		log.Fatalf("Unable to parse connection string: %v\n", err)
// 	}
//
// 	// Create the connection pool
// 	pool, err := pgxpool.NewWithConfig(ctx, config)
// 	if err != nil {
// 		log.Fatalf("Unable to create connection pool: %v\n", err)
// 	}
// 	defer pool.Close()
//
// 	// Perform a query using the connection pool
// 	var username, role string
// 	err = pool.QueryRow(ctx, "SELECT username, role FROM users LIMIT 1").Scan(&username, &role)
// 	if err != nil {
// 		log.Fatalf("QueryRow failed: %v\n", err)
// 	}
//
// 	fmt.Printf("User: %s, Role: %s\n", username, role)
// }

// func main() {
// 	conn, err := pgx.Connect(context.Background(), "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable")
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
// 		os.Exit(1)
// 	}
// 	defer conn.Close(context.Background())
//
// 	var (
// 		username          string
// 		role              string
// 		hashedPassword    string
// 		fullName          string
// 		email             string
// 		isEmailVerified   bool
// 		passwordChangedAt string // or time.Time depending on your needs
// 		createdAt         string // or time.Time depending on your needs
// 	)
//
// 	// err = conn.QueryRow(context.Background(), "select *  from users where id=$1", 42).Scan(&name, &weight)
// 	err = conn.QueryRow(
// 		context.Background(),
// 		"SELECT username, role, hashed_password, full_name, email, is_email_verified, password_changed_at, created_at FROM users WHERE username=$1",
// 		"some_username",
// 	).Scan(&username, &role, &hashedPassword, &fullName, &email, &isEmailVerified, &passwordChangedAt, &createdAt)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
// 		os.Exit(1)
// 	}
//
// 	fmt.Println("Username:", username)
// 	fmt.Println("Role:", role)
// 	fmt.Println("Hashed Password:", hashedPassword)
// 	fmt.Println("Full Name:", fullName)
// 	fmt.Println("Email:", email)
// 	fmt.Println("Email Verified:", isEmailVerified)
// 	fmt.Println("Password Changed At:", passwordChangedAt)
// 	fmt.Println("Created At:", createdAt)
// }
