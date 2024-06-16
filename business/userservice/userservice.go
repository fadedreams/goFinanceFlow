package userservice

import (
	"context"
	"fmt"

	"github.com/fadedreams/gofinanceflow/foundation/sdk"
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

func (us *UserService) LoginUser(ctx context.Context, username, password string) (*db.User, string, string, error) {
	user, err := us.store.GetUser(ctx, username)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid credentials")
	}

	// Verify hashed password
	if err := sdk.VerifyPassword(user.HashedPassword, password); err != nil {
		return nil, "", "", fmt.Errorf("invalid credentials")
	}

	// Generate JWT token
	token, err := sdk.GenerateJWTToken(user.Username, user.Role)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate JWT token")
	}

	// Generate refresh token
	refreshToken, err := sdk.GenerateRefreshToken(user.Username, user.Role)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate refresh token")
	}

	return &user, token, refreshToken, nil
}
