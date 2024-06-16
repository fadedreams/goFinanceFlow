package userservice

import (
	"context"
	"fmt"
	"log"

	"github.com/fadedreams/gofinanceflow/business/domain"
	"github.com/fadedreams/gofinanceflow/foundation/sdk"
	db "github.com/fadedreams/gofinanceflow/infrastructure/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	connPool *pgxpool.Pool
	store    *db.Queries
}

//	func NewUserService(store *db.Queries) *UserService {
//		return &UserService{
//			store: store,
//		}
//	}
func NewUserService(dbPool *pgxpool.Pool, store *db.Queries) *UserService {
	return &UserService{
		connPool: dbPool,
		store:    store,
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

func (us *UserService) HandleFundsTransfer(ctx context.Context, arg domain.HandleFundsTransferParams) (domain.HandleFundsTransferResult, error) {
	var result domain.HandleFundsTransferResult

	err := us.ExecuteInTransaction(ctx, func(q *db.Queries) error {
		// Check if 'from' account has sufficient balance
		fromAccount, err := q.GetAccount(ctx, arg.FromAccountID)
		if err != nil {
			return fmt.Errorf("failed to get 'from' account: %v", err)
		}
		if fromAccount.Balance < arg.Amount {
			return fmt.Errorf("insufficient funds in 'from' account")
		}

		// Create the transfer record
		result.Transfer, err = q.CreateTransfer(ctx, db.CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("failed to create transfer: %v", err)
		}

		// Create account transactions for both accounts
		result.FromEntry, err = q.CreateAccountTransactions(ctx, db.CreateAccountTransactionsParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("failed to create 'from' account transaction: %v", err)
		}

		result.ToEntry, err = q.CreateAccountTransactions(ctx, db.CreateAccountTransactionsParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("failed to create 'to' account transaction: %v", err)
		}

		// Adjust account balances
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}
		if err != nil {
			return fmt.Errorf("failed to adjust account balances: %v", err)
		}

		return nil
	})

	return result, err
}

func (us *UserService) ExecuteInTransaction(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := us.connPool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			// If a panic occurs, rollback the transaction and then re-panic
			rbErr := tx.Rollback(ctx)
			if rbErr != nil {
				log.Printf("Failed to rollback transaction: %v", rbErr)
			}
			panic(p) // Re-panic to propagate the panic further
		}
	}()

	q := db.New(tx) // Create a new db.Queries instance bound to the transaction

	err = fn(q) // Execute the provided function with the db.Queries instance
	if err != nil {
		rbErr := tx.Rollback(ctx)
		if rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func addMoney(
	ctx context.Context,
	q *db.Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 db.Account, account2 db.Account, err error) {
	account1, err = q.AddAccountBalance(ctx, db.AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, db.AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}
