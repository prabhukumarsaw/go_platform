package middleware

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"newsplatform/api/pkg/rls"
	"newsplatform/api/pkg/response"
)

// TenantContext is a Fiber middleware that:
//  1. Reads the active tenant from the session (set by RequireAuth)
//  2. Begins a Postgres transaction
//  3. Sets RLS session variables (app.tenant_id, app.is_super_admin) via SET LOCAL
//  4. Stores the transaction in the Fiber context for downstream handlers
//  5. Commits or rolls back the transaction after the handler returns
//
// This ensures that every database query within the request is automatically
// scoped to the correct tenant by PostgreSQL RLS policies.
func TenantContext(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := SessionFromCtx(c)
		if sess == nil {
			return response.Unauthorized(c, "Session required for tenant context")
		}

		ctx := c.Context()

		tx, err := pool.Begin(context.Background())
		if err != nil {
			return response.InternalError(c, "Failed to begin transaction")
		}

		// Set RLS session variables for this transaction
		if err := rls.SetTenantContext(ctx, tx, sess.ActiveTenantID, sess.IsSuperAdmin); err != nil {
			tx.Rollback(context.Background())
			return response.InternalError(c, "Failed to set tenant context")
		}

		// Store the transaction in context for handlers to use
		c.Locals("tx", tx)

		// Execute the handler
		handlerErr := c.Next()

		// Commit or rollback based on response status
		if handlerErr != nil || c.Response().StatusCode() >= 400 {
			tx.Rollback(context.Background())
		} else {
			if commitErr := tx.Commit(context.Background()); commitErr != nil {
				return response.InternalError(c, "Failed to commit transaction")
			}
		}

		return handlerErr
	}
}

// TenantContextPublic is like TenantContext but determines the tenant from the
// request (subdomain or query param) instead of requiring an authenticated session.
// Used for public reader endpoints where the tenant is derived from the URL.
func TenantContextPublic(pool *pgxpool.Pool, resolveTenant func(*fiber.Ctx) (int64, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID, err := resolveTenant(c)
		if err != nil {
			return response.BadRequest(c, fmt.Sprintf("Cannot resolve tenant: %v", err))
		}

		ctx := c.Context()

		tx, err := pool.Begin(context.Background())
		if err != nil {
			return response.InternalError(c, "Failed to begin transaction")
		}

		if err := rls.SetTenantContext(ctx, tx, tenantID, false); err != nil {
			tx.Rollback(context.Background())
			return response.InternalError(c, "Failed to set tenant context")
		}

		c.Locals("tx", tx)
		c.Locals("tenant_id", tenantID)

		handlerErr := c.Next()

		if handlerErr != nil || c.Response().StatusCode() >= 400 {
			tx.Rollback(context.Background())
		} else {
			if commitErr := tx.Commit(context.Background()); commitErr != nil {
				return response.InternalError(c, "Failed to commit transaction")
			}
		}

		return handlerErr
	}
}
