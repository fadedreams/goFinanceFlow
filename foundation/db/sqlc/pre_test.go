// pre_test.go

package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testStore Store
var testQueries *Queries

// Initialize the test environment before running each test file
func TestMain(m *testing.M) {
	// Connect to the test database
	conn, err := pgxpool.New(context.Background(), "postgresql://postgres:postgres@localhost:5432/ffdb?sslmode=disable")
	if err != nil {
		panic(err) // Panic on error as we cannot continue without a valid connection
	}

	// Create test queries
	testQueries = New(conn)

	// Create test store
	testStore = NewStore(conn)

	// Run tests
	exitVal := m.Run()

	// Clean up resources if needed

	// Exit with the result of the test run
	os.Exit(exitVal)
}
