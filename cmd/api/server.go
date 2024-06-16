package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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
	// s.router.POST("/users/login", s.loginUser)
	// s.router.POST("/users/refresh", s.refreshToken)
	// s.router.GET("/users/:username", s.getUser, JWTAuthMiddleware)
	//
	// // s.router.GET("/users/:username", s.getUser)
	// s.router.POST("/users", s.createUser)
	// s.router.PUT("/users/:username", s.getUser, JWTAuthMiddleware)

	s.router.POST("/users/login", s.loginUser)
	s.router.POST("/users", s.createUser)
	s.router.POST("/users/refresh", s.refreshToken)

	protected := s.router.Group("/users")
	protected.Use(JWTAuthMiddleware)
	// protected.Use(AdminRoleCheckMiddleware)
	protected.GET("/:username", s.getUser)
	protected.PUT("/:username", s.updateUser)

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
	user, token, refreshToken, err := s.userService.LoginUser(c.Request().Context(), params.Username, params.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Set refresh token as an HTTP-only cookie
	refreshTokenCookie := new(http.Cookie)
	refreshTokenCookie.Name = "refresh_token"
	refreshTokenCookie.Value = refreshToken
	refreshTokenCookie.HttpOnly = true
	refreshTokenCookie.Path = "/"
	refreshTokenCookie.Expires = time.Now().Add(30 * 24 * time.Hour) // Set cookie expiration time (e.g., 30 days)
	c.SetCookie(refreshTokenCookie)

	// Return login response
	response := domain.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         *user,
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) refreshToken(c echo.Context) error {
	refreshTokenCookie, err := c.Cookie("refresh_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid refresh token")
	}

	claims, err := sdk.VerifyToken(refreshTokenCookie.Value)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
	}

	username := claims["username"].(string)
	role := claims["role"].(string)
	newToken, err := sdk.GenerateJWTToken(username, role)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate new JWT token")
	}

	response := map[string]string{"token": newToken}
	return c.JSON(http.StatusOK, response)
}

func JWTAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check for Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			// Extract token from Authorization header
			token := strings.Split(authHeader, "Bearer ")[1]
			claims, err := sdk.VerifyToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			// Store user information from token claims in the context
			c.Set("username", claims["username"])
			c.Set("role", claims["role"])
			return next(c)
		}

		// Check for refresh token in cookies
		refreshTokenCookie, err := c.Cookie("refresh_token")
		if err == nil {
			claims, err := sdk.VerifyToken(refreshTokenCookie.Value)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
			}

			// Store user information from token claims in the context
			// fmt.Println(claims["username"])
			c.Set("role", claims["role"])
			c.Set("username", claims["username"])
			return next(c)
		}

		return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid token")
	}

}

func AdminRoleCheckMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		role, ok := c.Get("role").(string)
		// println(role)
		if !ok || role != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, "access forbidden: insufficient role")
		}
		return next(c)
	}
}
