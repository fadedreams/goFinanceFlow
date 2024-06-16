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
	Token        string  `json:"token"`
	RefreshToken string  `json:"refresh_token"`
	User         db.User `json:"user"`
}

type HandleFundsTransferParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// HandleFundsTransferResult is the result of the transfer transaction
type HandleFundsTransferResult struct {
	Transfer    db.Transfer           `json:"transfer"`
	FromAccount db.Account            `json:"from_account"`
	ToAccount   db.Account            `json:"to_account"`
	FromEntry   db.AccountTransaction `json:"from_entry"`
	ToEntry     db.AccountTransaction `json:"to_entry"`
}
