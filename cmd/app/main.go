package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	conn, err := pgx.Connect(context.Background(), "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var (
		username          string
		role              string
		hashedPassword    string
		fullName          string
		email             string
		isEmailVerified   bool
		passwordChangedAt string // or time.Time depending on your needs
		createdAt         string // or time.Time depending on your needs
	)

	// err = conn.QueryRow(context.Background(), "select *  from users where id=$1", 42).Scan(&name, &weight)
	err = conn.QueryRow(
		context.Background(),
		"SELECT username, role, hashed_password, full_name, email, is_email_verified, password_changed_at, created_at FROM users WHERE username=$1",
		"some_username",
	).Scan(&username, &role, &hashedPassword, &fullName, &email, &isEmailVerified, &passwordChangedAt, &createdAt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Username:", username)
	fmt.Println("Role:", role)
	fmt.Println("Hashed Password:", hashedPassword)
	fmt.Println("Full Name:", fullName)
	fmt.Println("Email:", email)
	fmt.Println("Email Verified:", isEmailVerified)
	fmt.Println("Password Changed At:", passwordChangedAt)
	fmt.Println("Created At:", createdAt)
}
