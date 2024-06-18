package grpc_api

import (
	"context"
	"fmt"
	"strings"

	"github.com/fadedreams/gofinanceflow/business/userservice"
	"github.com/fadedreams/gofinanceflow/foundation/sdk"
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/fadedreams/gofinanceflow/infrastructure/pb"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedFinanceFlowServer
	userService *userservice.UserService
}

func NewServer(store *db.Queries, dbPool *pgxpool.Pool) *Server {
	userService := userservice.NewUserService(dbPool, store) // Create UserService instance

	return &Server{
		userService: userService,
	}
}

func (s *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	hashedPassword, err := sdk.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	params := db.CreateUserParams{
		Username:       req.Username,
		FullName:       req.FullName,
		Email:          req.Email,
		HashedPassword: hashedPassword,
	}

	user, err := s.userService.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	response := &pb.CreateUserResponse{
		User: &pb.User{
			Username:          user.Username,
			FullName:          user.FullName,
			Email:             user.Email,
			PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
			CreatedAt:         timestamppb.New(user.CreatedAt),
		},
	}

	return response, nil
}

func (s *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	user, token, refreshToken, err := s.userService.LoginUser(ctx, req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %v", err)
	}

	response := &pb.LoginUserResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: &pb.User{
			Username:          user.Username,
			FullName:          user.FullName,
			Email:             user.Email,
			PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
			CreatedAt:         timestamppb.New(user.CreatedAt),
		},
	}

	return response, nil
}

// func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
// 	user, err := s.userService.GetUser(ctx, req.Username)
// 	if err != nil {
// 		return nil, fmt.Errorf("user not found: %v", err)
// 	}
//
// 	response := &pb.GetUserResponse{
// 		User: &pb.User{
// 			Username:          user.Username,
// 			FullName:          user.FullName,
// 			Email:             user.Email,
// 			PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
// 			CreatedAt:         timestamppb.New(user.CreatedAt),
// 		},
// 	}
//
// 	return response, nil
// }

func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// Extract metadata from context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	fmt.Println("md: ", md) // md:  map[authorization:[Bearer "md)

	// Get the authorization header
	authHeader, ok := md["authorization"]
	fmt.Println("authHeader: ", authHeader)
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

	// Proceed with fetching the user information
	user, err := s.userService.GetUser(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	response := &pb.GetUserResponse{
		User: &pb.User{
			Username:          user.Username,
			FullName:          user.FullName,
			Email:             user.Email,
			PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
			CreatedAt:         timestamppb.New(user.CreatedAt),
		},
	}

	return response, nil
}
