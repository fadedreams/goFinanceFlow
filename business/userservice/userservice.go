package userservice

import (
	"context"

	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
)

type UserService struct {
	store *db.Queries
}

func NewUserService(store *db.Queries) *UserService {
	return &UserService{
		store: store,
	}
}

func (us *UserService) GetUser(ctx context.Context, username string) (*db.User, error) {
	// Call the appropriate method from db.Queries to fetch user data
	user, err := us.store.GetUser(ctx, username)
	if err != nil {
		// Handle database errors or any other specific errors
		return nil, err
	}
	return &user, nil
}
