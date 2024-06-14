// user_test.go

package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func createTestQueries(t *testing.T) *Queries {
	// Connect to the test database
	conn, err := pgxpool.New(context.Background(), "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable")
	require.NoError(t, err)

	return New(conn)
}

func TestGetUser(t *testing.T) {
	q := createTestQueries(t)

	// Create a test user
	username := "testuser"
	hashedPassword := "hashedpassword"
	fullName := "Test User"
	email := "test@example.com"
	_, err := q.CreateUser(context.Background(), CreateUserParams{
		Username:       username,
		HashedPassword: hashedPassword,
		FullName:       fullName,
		Email:          email,
	})
	require.NoError(t, err)

	// Test the GetUser function
	user, err := q.GetUser(context.Background(), username)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	// Verify the returned user details
	require.Equal(t, username, user.Username)
	require.Equal(t, hashedPassword, user.HashedPassword)
	require.Equal(t, fullName, user.FullName)
	require.Equal(t, email, user.Email)
	require.False(t, user.IsEmailVerified) // Assuming default is false
	require.WithinDuration(t, time.Now(), user.CreatedAt, time.Second)

	// Clean up by deleting the test user
	_, err = q.db.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username)
	require.NoError(t, err)
}
