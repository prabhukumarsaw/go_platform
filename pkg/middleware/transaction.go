package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"newsplatform/api/pkg/response"
)

// TransactionContext manages request-scoped database transactions for staff/authenticated requests.
// It begins a transaction, attaches it to the Fiber context (`tx`),
// and automatically commits on 2xx/3xx or rolls back on errors.
func TransactionContext(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := SessionFromCtx(c)
		if sess == nil {
			return response.Unauthorized(c, "Session required")
		}

		tx, err := pool.Begin(context.Background())
		if err != nil {
			return response.InternalError(c, "Failed to begin transaction")
		}

		c.Locals("tx", tx)

		handlerErr := c.Next()

		if handlerErr != nil || c.Response().StatusCode() >= 400 {
			_ = tx.Rollback(context.Background())
		} else {
			if commitErr := tx.Commit(context.Background()); commitErr != nil {
				return response.InternalError(c, "Failed to commit transaction")
			}
		}

		return handlerErr
	}
}

// PublicTransactionContext manages database transactions for public reader requests.
func PublicTransactionContext(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			return response.InternalError(c, "Failed to begin transaction")
		}

		c.Locals("tx", tx)

		handlerErr := c.Next()

		if handlerErr != nil || c.Response().StatusCode() >= 400 {
			_ = tx.Rollback(context.Background())
		} else {
			if commitErr := tx.Commit(context.Background()); commitErr != nil {
				return response.InternalError(c, "Failed to commit transaction")
			}
		}

		return handlerErr
	}
}
