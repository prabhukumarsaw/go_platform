package rls

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// SetTenantContext sets the PostgreSQL session variables used by Row-Level Security
// policies. Must be called within a transaction (SET LOCAL is transaction-scoped).
//
// This sets:
//   - app.tenant_id  — the active tenant for this request
//   - app.is_super_admin — bypass flag that skips all RLS policies
//
// These variables are read by RLS policies like:
//
//	USING (tenant_id = current_setting('app.tenant_id')::int
//	       OR current_setting('app.is_super_admin')::boolean)
func SetTenantContext(ctx context.Context, tx pgx.Tx, tenantID int64, isSuperAdmin bool) error {
	_, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%d'", tenantID))
	if err != nil {
		return fmt.Errorf("set app.tenant_id: %w", err)
	}

	superAdminStr := "false"
	if isSuperAdmin {
		superAdminStr = "true"
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.is_super_admin = '%s'", superAdminStr))
	if err != nil {
		return fmt.Errorf("set app.is_super_admin: %w", err)
	}

	return nil
}

// ClearTenantContext resets the session variables. Usually not needed since
// SET LOCAL is automatically reset when the transaction ends, but useful for tests.
func ClearTenantContext(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "RESET app.tenant_id; RESET app.is_super_admin")
	return err
}
