// cmd/api/server.go

package api

import (
	"fmt"
	"net/http"

	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	store  *db.Queries
	router *echo.Echo
}

// NewServer creates a new HTTP server and sets up routing.
func NewServer(store *db.Queries) *Server {
	server := &Server{
		store:  store,
		router: echo.New(),
	}

	server.router.Use(middleware.Logger())
	server.router.Use(middleware.Recover())

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	s.router.GET("/users/:username", s.getUser)
}

func (s *Server) Start(address string) error {
	return s.router.Start(address)
}

func (s *Server) getUser(c echo.Context) error {
	username := c.Param("username")
	fmt.Println(username)
	user, err := s.store.GetUser(c.Request().Context(), username)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User %s not found", username))
	}
	return c.JSON(http.StatusOK, user)
}
