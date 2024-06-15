package api

import (
	"fmt"
	"net/http"

	"github.com/fadedreams/gofinanceflow/business/domain"
	"github.com/fadedreams/gofinanceflow/business/userservice" // Import UserService package
	"github.com/fadedreams/gofinanceflow/foundation/sdk"
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	userService *userservice.UserService
	router      *echo.Echo
}

// NewServer creates a new HTTP server and sets up routing.
func NewServer(store *db.Queries) *Server {
	userService := userservice.NewUserService(store) // Create UserService instance

	server := &Server{
		userService: userService,
		router:      echo.New(),
	}

	server.router.Use(middleware.Logger())
	server.router.Use(middleware.Recover())

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	s.router.GET("/users/login", s.loginUser)
	s.router.GET("/users/:username", s.getUser)
	s.router.POST("/users", s.createUser)
	s.router.PUT("/users/:username", s.updateUser)
}

func (s *Server) Start(address string) error {
	return s.router.Start(address)
}

func (s *Server) getUser(c echo.Context) error {
	username := c.Param("username")
	user, err := s.userService.GetUser(c.Request().Context(), username)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User %s not found", username))
	}
	return c.JSON(http.StatusOK, user)
}

func (s *Server) createUser(c echo.Context) error {
	var params db.CreateUserParams
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request payload")
	}

	// Hash the password using SDK function
	hashedPassword, err := sdk.HashPassword(params.HashedPassword)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to hash password: %s", err.Error()))
	}

	// Update params with hashed password
	params.HashedPassword = hashedPassword

	user, err := s.userService.CreateUser(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create user: %s", err.Error()))
	}
	return c.JSON(http.StatusCreated, user)
}

func (s *Server) updateUser(c echo.Context) error {
	username := c.Param("username")
	var params db.UpdateUserParams
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request payload")
	}

	user, err := s.userService.UpdateUser(c.Request().Context(), username, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update user: %s", err.Error()))
	}
	return c.JSON(http.StatusOK, user)
}

func (s *Server) loginUser(c echo.Context) error {
	var params domain.LoginUserParams
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request payload")
	}

	// Authenticate user
	user, token, err := s.userService.LoginUser(c.Request().Context(), params.Username, params.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Return login response
	response := domain.LoginResponse{
		Token: token,
		User:  *user,
	}
	return c.JSON(http.StatusOK, response)
}
