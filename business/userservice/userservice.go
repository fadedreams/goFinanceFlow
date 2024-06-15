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
	user, err := us.store.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (us *UserService) CreateUser(ctx context.Context, params db.CreateUserParams) (*db.User, error) {
	user, err := us.store.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (us *UserService) UpdateUser(ctx context.Context, username string, params db.UpdateUserParams) (*db.User, error) {
	params.Username = username // Ensure the username is set correctly in the params
	user, err := us.store.UpdateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
