package iam

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all IAM-related database operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new IAM repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ─── Override queries ──────────────────────────

// ActiveOverride represents a user permission override that is currently valid.
type ActiveOverride struct {
	Effect       string
	MenuActionID int
	Reason       string
	ValidUntil   *time.Time
}

// FindActiveOverride checks for a valid (non-expired) permission override for a
// specific user + tenant + action combination.
func (r *Repository) FindActiveOverride(ctx context.Context, tx pgx.Tx, userID, tenantID int64, actionName string) (*ActiveOverride, error) {
	query := `
		SELECT upo.effect, upo.menu_action_id, upo.reason, upo.valid_until
		FROM user_permission_overrides upo
		JOIN menu_actions ma ON ma.id = upo.menu_action_id
		JOIN menus m ON m.id = ma.menu_id
		WHERE upo.user_id = $1
		  AND upo.tenant_id = $2
		  AND (m.name || '.' || ma.action) = $3
		  AND upo.is_active = TRUE
		  AND (upo.valid_from IS NULL OR upo.valid_from <= NOW())
		  AND (upo.valid_until IS NULL OR upo.valid_until > NOW())
		ORDER BY upo.created_at DESC
		LIMIT 1
	`

	var ov ActiveOverride
	err := tx.QueryRow(ctx, query, userID, tenantID, actionName).Scan(
		&ov.Effect, &ov.MenuActionID, &ov.Reason, &ov.ValidUntil,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ov, nil
}

// ─── Role grant queries ───────────────────────

// HasRoleGrant checks if the user's role in the given tenant grants the specified action.
func (r *Repository) HasRoleGrant(ctx context.Context, tx pgx.Tx, userID, tenantID int64, actionName string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM user_tenant_mappings utm
			JOIN role_menu_actions rma ON rma.role_id = utm.role_id AND rma.tenant_id = utm.tenant_id
			JOIN menu_actions ma ON ma.id = rma.menu_action_id
			JOIN menus m ON m.id = ma.menu_id
			WHERE utm.user_id = $1
			  AND utm.tenant_id = $2
			  AND utm.is_active = TRUE
			  AND (m.name || '.' || ma.action) = $3
		)
	`

	var exists bool
	err := tx.QueryRow(ctx, query, userID, tenantID, actionName).Scan(&exists)
	return exists, err
}

// AssignRolePermissions sets all action permissions for a role in a tenant.
func (r *Repository) AssignRolePermissions(ctx context.Context, tx pgx.Tx, roleID int, tenantID int64, menuActionIDs []int) error {
	// Remove existing grants
	_, err := tx.Exec(ctx, "DELETE FROM role_menu_actions WHERE role_id = $1 AND tenant_id = $2", roleID, tenantID)
	if err != nil {
		return err
	}

	// Insert new grants
	for _, actionID := range menuActionIDs {
		_, err := tx.Exec(ctx,
			"INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id) VALUES ($1, $2, $3)",
			roleID, actionID, tenantID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// ─── ABAC queries ──────────────────────────────

// ABACPolicy represents an attribute-based access control condition.
type ABACPolicy struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	TenantID  int             `json:"tenant_id"`
	Attribute string          `json:"attribute"`
	Value     json.RawMessage `json:"value"`
	IsActive  bool            `json:"is_active"`
}

// FindABACPolicies returns all active ABAC policies for a user in a tenant.
func (r *Repository) FindABACPolicies(ctx context.Context, tx pgx.Tx, userID, tenantID int64) ([]ABACPolicy, error) {
	query := `
		SELECT id, user_id, tenant_id, attribute, value, is_active
		FROM abac_policies
		WHERE user_id = $1 AND tenant_id = $2 AND is_active = TRUE
	`

	rows, err := tx.Query(ctx, query, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []ABACPolicy
	for rows.Next() {
		var p ABACPolicy
		if err := rows.Scan(&p.ID, &p.UserID, &p.TenantID, &p.Attribute, &p.Value, &p.IsActive); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

// CreateABACPolicy inserts a new ABAC rule.
func (r *Repository) CreateABACPolicy(ctx context.Context, tx pgx.Tx, userID, tenantID int64, attribute string, value json.RawMessage) error {
	query := `
		INSERT INTO abac_policies (user_id, tenant_id, attribute, value)
		VALUES ($1, $2, $3, $4)
	`
	_, err := tx.Exec(ctx, query, userID, tenantID, attribute, value)
	return err
}

// ─── Audit queries ──────────────────────────────

// InsertAuditLog records a permission evaluation decision.
func (r *Repository) InsertAuditLog(ctx context.Context, tx pgx.Tx, log AuditEntry) error {
	query := `
		INSERT INTO permission_audit_log
			(user_id, tenant_id, action_name, decision, reason, ip_address, user_agent, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := tx.Exec(ctx, query,
		log.UserID, log.TenantID, log.ActionName, log.Decision,
		log.Reason, log.IPAddress, log.UserAgent, log.RequestID,
	)
	return err
}

// AuditEntry represents a single row in the permission audit log.
type AuditEntry struct {
	ID         int64     `json:"id,omitempty"`
	UserID     int64     `json:"user_id"`
	TenantID   *int64    `json:"tenant_id,omitempty"`
	ActionName string    `json:"action_name"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	RequestID  string    `json:"request_id"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// ListAuditLogs returns recent audit log records.
func (r *Repository) ListAuditLogs(ctx context.Context, tx pgx.Tx, limit, offset int) ([]AuditEntry, int64, error) {
	var total int64
	_ = tx.QueryRow(ctx, "SELECT COUNT(*) FROM permission_audit_log").Scan(&total)

	query := `
		SELECT id, user_id, tenant_id, action_name, decision, COALESCE(reason, ''),
		       COALESCE(host(ip_address), ''), COALESCE(user_agent, ''), COALESCE(request_id, ''), created_at
		FROM permission_audit_log
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := tx.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []AuditEntry
	for rows.Next() {
		var l AuditEntry
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.TenantID, &l.ActionName, &l.Decision, &l.Reason,
			&l.IPAddress, &l.UserAgent, &l.RequestID, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	return logs, total, rows.Err()
}

// ─── Role CRUD ──────────────────────────────────

// Role represents a tenant-scoped role.
type Role struct {
	ID          int    `json:"id"`
	TenantID    int    `json:"tenant_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
	IsActive    bool   `json:"is_active"`
}

// ListRoles returns all roles for a tenant.
func (r *Repository) ListRoles(ctx context.Context, tx pgx.Tx, tenantID int64) ([]Role, error) {
	query := `SELECT id, tenant_id, name, COALESCE(description,''), is_system, is_active FROM roles WHERE tenant_id = $1 ORDER BY name`
	rows, err := tx.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Name, &role.Description, &role.IsSystem, &role.IsActive); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

// CreateRole creates a new role in a tenant.
func (r *Repository) CreateRole(ctx context.Context, tx pgx.Tx, tenantID int64, name, description string) (*Role, error) {
	query := `INSERT INTO roles (tenant_id, name, description) VALUES ($1, $2, $3) RETURNING id, tenant_id, name, description, is_system, is_active`
	var role Role
	err := tx.QueryRow(ctx, query, tenantID, name, description).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Description, &role.IsSystem, &role.IsActive,
	)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// ─── Menu / MenuAction queries ──────────────────

// Menu represents a navigation/feature module.
type Menu struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Label    string `json:"label"`
	ParentID *int   `json:"parent_id,omitempty"`
	Icon     string `json:"icon"`
}

// MenuAction represents a permission atom (e.g., articles.PUBLISH).
type MenuAction struct {
	ID     int    `json:"id"`
	MenuID int    `json:"menu_id"`
	Action string `json:"action"`
	Label  string `json:"label"`
}

// ListMenus returns all menus.
func (r *Repository) ListMenus(ctx context.Context, tx pgx.Tx) ([]Menu, error) {
	query := `SELECT id, name, label, parent_id, COALESCE(icon,'') FROM menus WHERE is_active = TRUE ORDER BY sort_order`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		var m Menu
		if err := rows.Scan(&m.ID, &m.Name, &m.Label, &m.ParentID, &m.Icon); err != nil {
			return nil, err
		}
		menus = append(menus, m)
	}
	return menus, rows.Err()
}

// ListMenuActions returns all menu_actions for a given menu.
func (r *Repository) ListMenuActions(ctx context.Context, tx pgx.Tx, menuID int) ([]MenuAction, error) {
	query := `SELECT id, menu_id, action, COALESCE(label,'') FROM menu_actions WHERE menu_id = $1`
	rows, err := tx.Query(ctx, query, menuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []MenuAction
	for rows.Next() {
		var a MenuAction
		if err := rows.Scan(&a.ID, &a.MenuID, &a.Action, &a.Label); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// ─── User-Tenant mapping ────────────────────────

// UserTenantMapping represents a user's role assignment within a tenant.
type UserTenantMapping struct {
	UserID   int64  `json:"user_id"`
	TenantID int    `json:"tenant_id"`
	RoleID   int    `json:"role_id"`
	RoleName string `json:"role_name"`
	IsActive bool   `json:"is_active"`
}

// AssignUserRole assigns a role to a user for a specific tenant.
func (r *Repository) AssignUserRole(ctx context.Context, tx pgx.Tx, userID int64, tenantID int64, roleID int, assignedBy int64) error {
	query := `
		INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id, assigned_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, tenant_id)
		DO UPDATE SET role_id = EXCLUDED.role_id, assigned_by = EXCLUDED.assigned_by, updated_at = NOW()
	`
	_, err := tx.Exec(ctx, query, userID, tenantID, roleID, assignedBy)
	return err
}

// ─── Permission overrides CRUD ──────────────────

// CreateOverride creates a new permission override for a user.
func (r *Repository) CreateOverride(ctx context.Context, tx pgx.Tx, userID, tenantID int64, menuActionID int, effect, reason string, validFrom, validUntil *time.Time, grantedBy int64) error {
	query := `
		INSERT INTO user_permission_overrides
			(user_id, tenant_id, menu_action_id, effect, reason, valid_from, valid_until, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := tx.Exec(ctx, query, userID, tenantID, menuActionID, effect, reason, validFrom, validUntil, grantedBy)
	return err
}

// ─── District scope queries ─────────────────────

// GetUserDistrictScopes returns all district IDs a user can access within a tenant.
func (r *Repository) GetUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID, tenantID int64) ([]int, error) {
	query := `SELECT district_id FROM user_district_scopes WHERE user_id = $1 AND tenant_id = $2`
	rows, err := tx.Query(ctx, query, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// AssignUserDistrictScopes sets the assigned districts for a user in a tenant.
func (r *Repository) AssignUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID, tenantID int64, districtIDs []int) error {
	_, err := tx.Exec(ctx, "DELETE FROM user_district_scopes WHERE user_id = $1 AND tenant_id = $2", userID, tenantID)
	if err != nil {
		return err
	}

	for _, dID := range districtIDs {
		_, err := tx.Exec(ctx, "INSERT INTO user_district_scopes (user_id, tenant_id, district_id) VALUES ($1, $2, $3)", userID, tenantID, dID)
		if err != nil {
			return err
		}
	}
	return nil
}
