package domain

import (
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
)

// LoginUserParams holds parameters for user login.
type LoginUserParams struct {
	Username string `json:"username" form:"username" query:"username" validate:"required"`
	Password string `json:"password" form:"password" query:"password" validate:"required"`
}

// LoginResponse represents the response structure for a successful login.
type LoginResponse struct {
	Token string  `json:"token"`
	User  db.User `json:"user"`
}
