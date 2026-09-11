package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

// ─── Models ───────────────────────────────────

// Role represents an enterprise newsroom IAM role.
type Role struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsSystem    bool       `json:"is_system"`
	IsActive    bool       `json:"is_active"`
	UserCount   int64      `json:"user_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// Menu represents a navigation/feature module.
type Menu struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Label     string `json:"label"`
	ParentID  *int   `json:"parent_id,omitempty"`
	Icon      string `json:"icon"`
	Path      string `json:"path"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
	GroupName string `json:"group_name"`
	APIPrefix string `json:"api_prefix"`
}

// MenuAction represents a permission atom (e.g., articles.PUBLISH).
type MenuAction struct {
	ID     int    `json:"id"`
	MenuID int    `json:"menu_id"`
	Action string `json:"action"`
	Label  string `json:"label"`
}

type MenuActionMatrixItem struct {
	ActionID int    `json:"action_id"`
	Action   string `json:"action"`
	Label    string `json:"label"`
	Granted  bool   `json:"granted"`
}

type MenuMatrixItem struct {
	MenuID    int                    `json:"menu_id"`
	Name      string                 `json:"name"`
	Label     string                 `json:"label"`
	Icon      string                 `json:"icon"`
	SortOrder int                    `json:"sort_order"`
	Actions   []MenuActionMatrixItem `json:"actions"`
}

// CategoryScope represents a regional or beat assignment for journalists.
type CategoryScope struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Path       string `json:"path"`
	Level      int    `json:"level"`
}

// EffectivePermissions summarizes all rights granted to a staff user.
type EffectivePermissions struct {
	UserID         int64           `json:"user_id"`
	IsSuperAdmin   bool            `json:"is_super_admin"`
	Roles          []string        `json:"roles"`
	Permissions    []string        `json:"permissions"`
	CategoryScopes []CategoryScope `json:"category_scopes"`
}

// ActiveOverride represents a user permission override that is currently valid.
type ActiveOverride struct {
	Effect       string
	MenuActionID int
	Reason       string
	ValidUntil   *time.Time
}

// ABACPolicy represents an attribute-based access control condition.
type ABACPolicy struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	Attribute string          `json:"attribute"`
	Value     json.RawMessage `json:"value"`
	IsActive  bool            `json:"is_active"`
}

// StaffUserRoleSummary represents a staff member with their assigned role and bureau scopes.
type StaffUserRoleSummary struct {
	UserID         int64           `json:"user_id"`
	DisplayName    string          `json:"display_name"`
	Email          string          `json:"email"`
	RoleID         *int            `json:"role_id,omitempty"`
	RoleName       *string         `json:"role_name,omitempty"`
	IsSuperAdmin   bool            `json:"is_super_admin"`
	IsActive       bool            `json:"is_active"`
	CategoryScopes []CategoryScope `json:"category_scopes"`
}

// AuditEntry represents a single row in the permission audit log.
type AuditEntry struct {
	ID         int64     `json:"id,omitempty"`
	UserID     int64     `json:"user_id"`
	UserName   string    `json:"user_name,omitempty"`
	UserEmail  string    `json:"user_email,omitempty"`
	ActionName string    `json:"action_name"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	RequestID  string    `json:"request_id"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// ─── Role CRUD Operations ──────────────────────

func (r *Repository) ListRoles(ctx context.Context, tx pgx.Tx) ([]Role, error) {
	query := `
		SELECT r.id, r.name, COALESCE(r.description,''), r.is_system, r.is_active,
		       COUNT(ur.user_id) as user_count, r.created_at, r.updated_at
		FROM roles r
		LEFT JOIN user_roles ur ON ur.role_id = r.id AND ur.is_active = TRUE
		GROUP BY r.id, r.name, r.description, r.is_system, r.is_active, r.created_at, r.updated_at
		ORDER BY r.id ASC, r.name ASC
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.IsActive,
			&role.UserCount, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *Repository) ListRolesDirect(ctx context.Context) ([]Role, error) {
	query := `
		SELECT r.id, r.name, COALESCE(r.description,''), r.is_system, r.is_active,
		       COUNT(ur.user_id) as user_count, r.created_at, r.updated_at
		FROM roles r
		LEFT JOIN user_roles ur ON ur.role_id = r.id AND ur.is_active = TRUE
		GROUP BY r.id, r.name, r.description, r.is_system, r.is_active, r.created_at, r.updated_at
		ORDER BY r.id ASC, r.name ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.IsActive,
			&role.UserCount, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *Repository) GetRole(ctx context.Context, tx pgx.Tx, roleID int) (*Role, error) {
	query := `
		SELECT r.id, r.name, COALESCE(r.description,''), r.is_system, r.is_active,
		       COUNT(ur.user_id) as user_count, r.created_at, r.updated_at
		FROM roles r
		LEFT JOIN user_roles ur ON ur.role_id = r.id AND ur.is_active = TRUE
		WHERE r.id = $1
		GROUP BY r.id, r.name, r.description, r.is_system, r.is_active, r.created_at, r.updated_at
	`
	var role Role
	err := tx.QueryRow(ctx, query, roleID).Scan(
		&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.IsActive,
		&role.UserCount, &role.CreatedAt, &role.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) CreateRole(ctx context.Context, tx pgx.Tx, name, description string) (*Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	// Case-insensitive duplicate check
	var existingID int
	err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE LOWER(TRIM(name)) = LOWER($1)", name).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("a role with the name '%s' already exists", name)
	}

	query := `
		INSERT INTO roles (name, description, is_system, is_active)
		VALUES ($1, $2, FALSE, TRUE)
		RETURNING id, name, description, is_system, is_active, created_at, updated_at
	`
	var role Role
	err = tx.QueryRow(ctx, query, name, description).Scan(
		&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.IsActive,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "roles_name_unique") {
			return nil, fmt.Errorf("a role with the name '%s' already exists", name)
		}
		return nil, err
	}
	return &role, nil
}

func (r *Repository) UpdateRole(ctx context.Context, tx pgx.Tx, id int, name, description string) (*Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	// Case-insensitive duplicate check excluding self
	var existingID int
	err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE LOWER(TRIM(name)) = LOWER($1) AND id <> $2", name, id).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("a role with the name '%s' already exists", name)
	}

	query := `
		UPDATE roles
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, description, is_system, is_active, created_at, updated_at
	`
	var role Role
	err = tx.QueryRow(ctx, query, name, description, id).Scan(
		&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.IsActive,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "roles_name_unique") {
			return nil, fmt.Errorf("a role with the name '%s' already exists", name)
		}
		return nil, err
	}
	return &role, nil
}

func (r *Repository) DeleteRole(ctx context.Context, tx pgx.Tx, id int) error {
	var isSystem bool
	var userCount int64
	err := tx.QueryRow(ctx, `
		SELECT r.is_system, COUNT(ur.user_id)
		FROM roles r
		LEFT JOIN user_roles ur ON ur.role_id = r.id
		WHERE r.id = $1
		GROUP BY r.is_system
	`, id).Scan(&isSystem, &userCount)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	if isSystem {
		return fmt.Errorf("cannot delete core system role")
	}
	if userCount > 0 {
		return fmt.Errorf("cannot delete role assigned to %d active users", userCount)
	}

	_, err = tx.Exec(ctx, "DELETE FROM roles WHERE id = $1", id)
	return err
}

func (r *Repository) CloneRole(ctx context.Context, tx pgx.Tx, sourceRoleID int, newName, newDescription string) (*Role, error) {
	var src Role
	err := tx.QueryRow(ctx, "SELECT id, name, description FROM roles WHERE id = $1", sourceRoleID).Scan(
		&src.ID, &src.Name, &src.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("source role not found: %w", err)
	}

	if newDescription == "" {
		newDescription = fmt.Sprintf("Cloned from %s", src.Name)
	}

	newRole, err := r.CreateRole(ctx, tx, newName, newDescription)
	if err != nil {
		return nil, err
	}

	// Copy all granted actions from source role
	_, err = tx.Exec(ctx, `
		INSERT INTO role_menu_actions (role_id, menu_action_id)
		SELECT $1, menu_action_id
		FROM role_menu_actions
		WHERE role_id = $2
		ON CONFLICT (role_id, menu_action_id) DO NOTHING
	`, newRole.ID, sourceRoleID)
	if err != nil {
		return nil, fmt.Errorf("copy role permissions: %w", err)
	}

	return newRole, nil
}

// ─── Permission Matrix ──────────────────────────

func (r *Repository) GetRolePermissionMatrix(ctx context.Context, tx pgx.Tx, roleID int) ([]MenuMatrixItem, error) {
	menusQuery := `SELECT id, name, label, COALESCE(icon,''), sort_order FROM menus WHERE is_active IS NOT FALSE ORDER BY sort_order ASC, id ASC`
	mRows, err := tx.Query(ctx, menusQuery)
	if err != nil {
		return nil, err
	}
	defer mRows.Close()

	var menus []MenuMatrixItem
	menuMap := make(map[int]int)
	for mRows.Next() {
		var m MenuMatrixItem
		if err := mRows.Scan(&m.MenuID, &m.Name, &m.Label, &m.Icon, &m.SortOrder); err != nil {
			return nil, err
		}
		m.Actions = []MenuActionMatrixItem{}
		menuMap[m.MenuID] = len(menus)
		menus = append(menus, m)
	}

	actionsQuery := `
		SELECT ma.id, ma.menu_id, ma.action, COALESCE(ma.label, ''),
		       EXISTS(
		           SELECT 1 FROM role_menu_actions rma
		           WHERE rma.menu_action_id = ma.id AND rma.role_id = $1
		       ) as granted
		FROM menu_actions ma
		ORDER BY ma.menu_id, ma.id
	`
	aRows, err := tx.Query(ctx, actionsQuery, roleID)
	if err != nil {
		return nil, err
	}
	defer aRows.Close()

	for aRows.Next() {
		var ma MenuActionMatrixItem
		var menuID int
		if err := aRows.Scan(&ma.ActionID, &menuID, &ma.Action, &ma.Label, &ma.Granted); err != nil {
			return nil, err
		}
		if idx, ok := menuMap[menuID]; ok {
			menus[idx].Actions = append(menus[idx].Actions, ma)
		}
	}

	return menus, nil
}

func (r *Repository) GetRolePermissionMatrixDirect(ctx context.Context, roleID int) ([]MenuMatrixItem, error) {
	menusQuery := `SELECT id, name, label, COALESCE(icon,''), sort_order FROM menus WHERE is_active IS NOT FALSE ORDER BY sort_order ASC, id ASC`
	mRows, err := r.pool.Query(ctx, menusQuery)
	if err != nil {
		return nil, err
	}
	defer mRows.Close()

	var menus []MenuMatrixItem
	menuMap := make(map[int]int)
	for mRows.Next() {
		var m MenuMatrixItem
		if err := mRows.Scan(&m.MenuID, &m.Name, &m.Label, &m.Icon, &m.SortOrder); err != nil {
			return nil, err
		}
		m.Actions = []MenuActionMatrixItem{}
		menuMap[m.MenuID] = len(menus)
		menus = append(menus, m)
	}

	actionsQuery := `
		SELECT ma.id, ma.menu_id, ma.action, COALESCE(ma.label, ''),
		       EXISTS(
		           SELECT 1 FROM role_menu_actions rma
		           WHERE rma.menu_action_id = ma.id AND rma.role_id = $1
		       ) as granted
		FROM menu_actions ma
		ORDER BY ma.menu_id, ma.id
	`
	aRows, err := r.pool.Query(ctx, actionsQuery, roleID)
	if err != nil {
		return nil, err
	}
	defer aRows.Close()

	for aRows.Next() {
		var ma MenuActionMatrixItem
		var menuID int
		if err := aRows.Scan(&ma.ActionID, &menuID, &ma.Action, &ma.Label, &ma.Granted); err != nil {
			return nil, err
		}
		if idx, ok := menuMap[menuID]; ok {
			menus[idx].Actions = append(menus[idx].Actions, ma)
		}
	}

	return menus, nil
}

func (r *Repository) AssignRolePermissions(ctx context.Context, tx pgx.Tx, roleID int, menuActionIDs []int) error {
	// Remove existing grants
	_, err := tx.Exec(ctx, "DELETE FROM role_menu_actions WHERE role_id = $1", roleID)
	if err != nil {
		return err
	}

	// Insert new grants
	for _, actionID := range menuActionIDs {
		_, err := tx.Exec(ctx,
			"INSERT INTO role_menu_actions (role_id, menu_action_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			roleID, actionID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// ─── Menu Queries ──────────────────────────────

const menusSelect = `
		SELECT id, name, label, COALESCE(icon,''), COALESCE(path,''), sort_order, COALESCE(is_active, true),
		       COALESCE(group_name, 'content'), COALESCE(api_prefix, '')
		FROM menus
		WHERE is_active IS NOT FALSE
		ORDER BY sort_order ASC, id ASC
`

func scanMenus(rows pgx.Rows) ([]Menu, error) {
	defer rows.Close()
	var menus []Menu
	for rows.Next() {
		var m Menu
		if err := rows.Scan(&m.ID, &m.Name, &m.Label, &m.Icon, &m.Path, &m.SortOrder, &m.IsActive, &m.GroupName, &m.APIPrefix); err != nil {
			return nil, err
		}
		menus = append(menus, m)
	}
	if menus == nil {
		menus = []Menu{}
	}
	return menus, rows.Err()
}

func (r *Repository) ListMenus(ctx context.Context, tx pgx.Tx) ([]Menu, error) {
	rows, err := tx.Query(ctx, menusSelect)
	if err != nil {
		return nil, err
	}
	return scanMenus(rows)
}

func (r *Repository) ListMenusDirect(ctx context.Context) ([]Menu, error) {
	rows, err := r.pool.Query(ctx, menusSelect)
	if err != nil {
		return nil, err
	}
	return scanMenus(rows)
}

func (r *Repository) ListMenusForUser(ctx context.Context, tx pgx.Tx, userID int64, isSuperAdmin bool) ([]Menu, error) {
	if isSuperAdmin {
		if tx != nil {
			return r.ListMenus(ctx, tx)
		}
		return r.ListMenusDirect(ctx)
	}

	query := `
		SELECT DISTINCT m.id, m.name, m.label, COALESCE(m.icon,''), COALESCE(m.path,''), m.sort_order,
		       COALESCE(m.is_active, true), COALESCE(m.group_name, 'content'), COALESCE(m.api_prefix, '')
		FROM menus m
		JOIN menu_actions ma ON ma.menu_id = m.id AND UPPER(ma.action) = 'VIEW'
		WHERE m.is_active IS NOT FALSE
		  AND (
		    EXISTS (
		      SELECT 1 FROM users u WHERE u.id = $1 AND u.is_super_admin = TRUE AND u.is_active = TRUE
		    )
		    OR EXISTS (
		      SELECT 1
		      FROM user_roles ur
		      JOIN role_menu_actions rma ON rma.role_id = ur.role_id
		      WHERE ur.user_id = $1 AND ur.is_active = TRUE AND rma.menu_action_id = ma.id
		    )
		    OR EXISTS (
		      SELECT 1 FROM user_permission_overrides upo
		      WHERE upo.user_id = $1 AND upo.menu_action_id = ma.id AND upo.is_active = TRUE
		        AND upo.effect = 'GRANT'
		        AND (upo.valid_from IS NULL OR upo.valid_from <= NOW())
		        AND (upo.valid_until IS NULL OR upo.valid_until > NOW())
		    )
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM user_permission_overrides upo
		    WHERE upo.user_id = $1 AND upo.menu_action_id = ma.id AND upo.is_active = TRUE
		      AND upo.effect = 'REVOKE'
		      AND (upo.valid_from IS NULL OR upo.valid_from <= NOW())
		      AND (upo.valid_until IS NULL OR upo.valid_until > NOW())
		  )
		ORDER BY m.sort_order ASC, m.id ASC
	`
	var rows pgx.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(ctx, query, userID)
	} else {
		rows, err = r.pool.Query(ctx, query, userID)
	}
	if err != nil {
		return nil, err
	}
	return scanMenus(rows)
}

func (r *Repository) ListMenuActions(ctx context.Context, tx pgx.Tx, menuID int) ([]MenuAction, error) {
	query := `SELECT id, menu_id, action, COALESCE(label,'') FROM menu_actions WHERE menu_id = $1 ORDER BY id ASC`
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

// ─── User Role Mapping ────────────────────────

func (r *Repository) AssignUserRole(ctx context.Context, tx pgx.Tx, userID int64, roleID int, assignedBy int64) error {
	return r.ReplaceUserRole(ctx, tx, userID, roleID, assignedBy)
}

func (r *Repository) ReplaceUserRole(ctx context.Context, tx pgx.Tx, userID int64, roleID int, assignedBy int64) error {
	if _, err := tx.Exec(ctx, `UPDATE user_roles SET is_active = FALSE WHERE user_id = $1`, userID); err != nil {
		return err
	}
	query := `
		INSERT INTO user_roles (user_id, role_id, is_active, assigned_by)
		VALUES ($1, $2, TRUE, $3)
		ON CONFLICT (user_id, role_id)
		DO UPDATE SET assigned_by = EXCLUDED.assigned_by, is_active = TRUE
	`
	_, err := tx.Exec(ctx, query, userID, roleID, assignedBy)
	return err
}

// ─── Category / Bureau Scoping ─────────────────

func (r *Repository) GetUserCategoryScopes(ctx context.Context, tx pgx.Tx, userID int64) ([]CategoryScope, error) {
	query := `
		SELECT c.id, c.name, c.slug, COALESCE(c.path, c.name), c.level
		FROM categories c
		JOIN user_category_scopes ucs ON ucs.category_id = c.id
		WHERE ucs.user_id = $1
		ORDER BY c.level ASC, c.name ASC
	`
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scopes []CategoryScope
	for rows.Next() {
		var cs CategoryScope
		if err := rows.Scan(&cs.CategoryID, &cs.Name, &cs.Slug, &cs.Path, &cs.Level); err != nil {
			return nil, err
		}
		scopes = append(scopes, cs)
	}
	return scopes, rows.Err()
}

func (r *Repository) AssignUserCategoryScopes(ctx context.Context, tx pgx.Tx, userID int64, categoryIDs []int, assignedBy int64) error {
	_, err := tx.Exec(ctx, "DELETE FROM user_category_scopes WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	for _, catID := range categoryIDs {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_category_scopes (user_id, category_id, assigned_by)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, category_id) DO NOTHING
		`, userID, catID, assignedBy)
		if err != nil {
			return err
		}
	}
	return nil
}

// ─── Resource Ownership Operations ──────────────────────

func (r *Repository) GetResourceOwner(ctx context.Context, tx pgx.Tx, resourceType, resourceID string) (int64, error) {
	var ownerID int64
	var query string
	
	switch resourceType {
	case "articles":
		query = "SELECT owner_id FROM articles WHERE id::text = $1"
	case "media":
		query = "SELECT owner_id FROM media WHERE id::text = $1"
	case "live_blog_entries":
		query = "SELECT owner_id FROM live_blog_entries WHERE id::text = $1"
	case "web_stories":
		query = "SELECT owner_id FROM web_stories WHERE id::text = $1"
	default:
		return 0, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	
	err := tx.QueryRow(ctx, query, resourceID).Scan(&ownerID)
	if err != nil {
		return 0, err
	}
	
	return ownerID, nil
}

// Backward-compatibility aliases for legacy frontend routes
func (r *Repository) GetUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID int64) ([]int, error) {
	scopes, err := r.GetUserCategoryScopes(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(scopes))
	for _, s := range scopes {
		ids = append(ids, s.CategoryID)
	}
	return ids, nil
}

func (r *Repository) CheckResourceOwnership(ctx context.Context, tx pgx.Tx, userID int64, resourceType, resourceID string) (bool, error) {
	ownerID, err := r.GetResourceOwner(ctx, tx, resourceType, resourceID)
	if err != nil {
		return false, err
	}
	
	return ownerID == userID, nil
}

// ─── NEW: Permission Operations (Resource-Action-Scope Model) ──────────────────────

// Permission represents a resource-action-scope permission
type Permission struct {
	ID          int        `json:"id"`
	Resource    string     `json:"resource"`
	Action      string     `json:"action"`
	Scope       string     `json:"scope"`
	Description string     `json:"description"`
	IsSystem    bool       `json:"is_system"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PermissionGroup represents a collection of permissions
type PermissionGroup struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsSystem    bool       `json:"is_system"`
	CreatedAt   time.Time  `json:"created_at"`
}

// UserPermission represents a direct user permission grant
type UserPermission struct {
	UserID       int64       `json:"user_id"`
	PermissionID int         `json:"permission_id"`
	Permission   Permission  `json:"permission"`
	Effect       string      `json:"effect"`
	Reason       string      `json:"reason"`
	GrantedBy    int64       `json:"granted_by"`
	ValidFrom    time.Time   `json:"valid_from"`
	ValidUntil   *time.Time  `json:"valid_until,omitempty"`
	Conditions   interface{} `json:"conditions,omitempty"`
}

// ResourceGrant represents a specific resource access grant
type ResourceGrant struct {
	ID           int64       `json:"id"`
	UserID       int64       `json:"user_id"`
	ResourceType string      `json:"resource_type"`
	ResourceID   string      `json:"resource_id"`
	Permission   Permission  `json:"permission"`
	GrantedBy    int64       `json:"granted_by"`
	GrantedAt    time.Time   `json:"granted_at"`
	ExpiresAt    *time.Time  `json:"expires_at,omitempty"`
	Conditions   interface{} `json:"conditions,omitempty"`
}

// EnhancedABACPolicy represents an advanced ABAC policy
type EnhancedABACPolicy struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	PolicyType  string      `json:"policy_type"`
	TargetID    *int64      `json:"target_id,omitempty"`
	Effect      string      `json:"effect"`
	Priority    int         `json:"priority"`
	Conditions  interface{} `json:"conditions"`
	IsActive    bool        `json:"is_active"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ListPermissions returns all permissions
func (r *Repository) ListPermissions(ctx context.Context, tx pgx.Tx) ([]Permission, error) {
	query := `
		SELECT id, resource, action, scope, COALESCE(description,''), is_system, created_at
		FROM permissions
		ORDER BY resource, action, scope
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Resource, &p.Action, &p.Scope, &p.Description, &p.IsSystem, &p.CreatedAt); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

// GetPermission returns a specific permission
func (r *Repository) GetPermission(ctx context.Context, tx pgx.Tx, permissionID int) (*Permission, error) {
	query := `
		SELECT id, resource, action, scope, COALESCE(description,''), is_system, created_at
		FROM permissions WHERE id = $1
	`
	var p Permission
	err := tx.QueryRow(ctx, query, permissionID).Scan(
		&p.ID, &p.Resource, &p.Action, &p.Scope, &p.Description, &p.IsSystem, &p.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePermission creates a new permission
func (r *Repository) CreatePermission(ctx context.Context, tx pgx.Tx, resource, action, scope, description string, isSystem bool) (*Permission, error) {
	query := `
		INSERT INTO permissions (resource, action, scope, description, is_system)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, resource, action, scope, description, is_system, created_at
	`
	var p Permission
	err := tx.QueryRow(ctx, query, resource, action, scope, description, isSystem).Scan(
		&p.ID, &p.Resource, &p.Action, &p.Scope, &p.Description, &p.IsSystem, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPermissionGroups returns all permission groups
func (r *Repository) ListPermissionGroups(ctx context.Context, tx pgx.Tx) ([]PermissionGroup, error) {
	query := `
		SELECT id, name, COALESCE(description,''), is_system, created_at
		FROM permission_groups
		ORDER BY name
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []PermissionGroup
	for rows.Next() {
		var g PermissionGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.IsSystem, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// GetRolePermissions returns permissions for a specific role
func (r *Repository) GetRolePermissions(ctx context.Context, tx pgx.Tx, roleID int) ([]Permission, error) {
	query := `
		SELECT p.id, p.resource, p.action, p.scope, COALESCE(p.description,''), p.is_system, p.created_at
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		  AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
		ORDER BY p.resource, p.action, p.scope
	`
	rows, err := tx.Query(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Resource, &p.Action, &p.Scope, &p.Description, &p.IsSystem, &p.CreatedAt); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

// HasEnhancedPermission checks if a role has a specific enhanced permission
func (r *Repository) HasEnhancedPermission(ctx context.Context, tx pgx.Tx, roleID int, resource, action string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		  AND p.resource = $2
		  AND p.action = $3
		  AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
	`
	err := tx.QueryRow(ctx, query, roleID, resource, action).Scan(&count)
	return count > 0, err
}

// AssignRolePermissions assigns permissions to a role (new system)
func (r *Repository) AssignRolePermissionsNew(ctx context.Context, tx pgx.Tx, roleID int, permissionIDs []int, grantedBy int64) error {
	// Remove existing grants
	_, err := tx.Exec(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return err
	}

	// Insert new grants
	for _, permID := range permissionIDs {
		_, err := tx.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_id, granted_by)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (role_id, permission_id) 
			 DO UPDATE SET granted_by = EXCLUDED.granted_by`,
			roleID, permID, grantedBy,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUserPermissions returns direct permissions for a user
func (r *Repository) GetUserPermissions(ctx context.Context, tx pgx.Tx, userID int64) ([]UserPermission, error) {
	query := `
		SELECT 
			up.user_id, up.permission_id, up.effect, COALESCE(up.reason,''),
			COALESCE(up.granted_by, 0), up.valid_from, up.valid_until, up.conditions,
			p.id, p.resource, p.action, p.scope, COALESCE(p.description,''), p.is_system, p.created_at
		FROM user_permissions up
		JOIN permissions p ON p.id = up.permission_id
		WHERE up.user_id = $1
		  AND (up.valid_until IS NULL OR up.valid_until > NOW())
		  AND (up.valid_from IS NULL OR up.valid_from <= NOW())
		ORDER BY p.resource, p.action, p.scope
	`
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userPerms []UserPermission
	for rows.Next() {
		var up UserPermission
		var p Permission
		var conditions json.RawMessage
		if err := rows.Scan(
			&up.UserID, &up.PermissionID, &up.Effect, &up.Reason,
			&up.GrantedBy, &up.ValidFrom, &up.ValidUntil, &conditions,
			&p.ID, &p.Resource, &p.Action, &p.Scope, &p.Description, &p.IsSystem, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		up.Permission = p
		if conditions != nil {
			up.Conditions = conditions
		}
		userPerms = append(userPerms, up)
	}
	return userPerms, nil
}

// GrantUserPermission grants a direct permission to a user
func (r *Repository) GrantUserPermission(ctx context.Context, tx pgx.Tx, userID int64, permissionID int, effect, reason string, validFrom, validUntil *time.Time, grantedBy int64) error {
	query := `
		INSERT INTO user_permissions (user_id, permission_id, effect, reason, valid_from, valid_until, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, permission_id, effect)
		DO UPDATE SET 
			reason = EXCLUDED.reason,
			valid_from = EXCLUDED.valid_from,
			valid_until = EXCLUDED.valid_until,
			granted_by = EXCLUDED.granted_by
	`
	_, err := tx.Exec(ctx, query, userID, permissionID, effect, reason, validFrom, validUntil, grantedBy)
	return err
}

// RevokeUserPermission revokes a direct permission from a user
func (r *Repository) RevokeUserPermission(ctx context.Context, tx pgx.Tx, userID int64, permissionID int) error {
	_, err := tx.Exec(ctx, "DELETE FROM user_permissions WHERE user_id = $1 AND permission_id = $2", userID, permissionID)
	return err
}

// GetUserResourceGrants returns resource-specific grants for a user
func (r *Repository) GetUserResourceGrants(ctx context.Context, tx pgx.Tx, userID int64) ([]ResourceGrant, error) {
	query := `
		SELECT 
			rg.id, rg.user_id, rg.resource_type, rg.resource_id,
			COALESCE(rg.granted_by, 0), rg.granted_at, rg.expires_at, rg.conditions,
			p.id, p.resource, p.action, p.scope, COALESCE(p.description,''), p.is_system, p.created_at
		FROM resource_grants rg
		JOIN permissions p ON p.id = rg.permission_id
		WHERE rg.user_id = $1
		  AND (rg.expires_at IS NULL OR rg.expires_at > NOW())
		ORDER BY rg.resource_type, rg.resource_id
	`
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []ResourceGrant
	for rows.Next() {
		var rg ResourceGrant
		var p Permission
		var conditions json.RawMessage
		if err := rows.Scan(
			&rg.ID, &rg.UserID, &rg.ResourceType, &rg.ResourceID,
			&rg.GrantedBy, &rg.GrantedAt, &rg.ExpiresAt, &conditions,
			&p.ID, &p.Resource, &p.Action, &p.Scope, &p.Description, &p.IsSystem, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		rg.Permission = p
		if conditions != nil {
			rg.Conditions = conditions
		}
		grants = append(grants, rg)
	}
	return grants, nil
}

// CreateResourceGrant creates a resource-specific access grant
func (r *Repository) CreateResourceGrant(ctx context.Context, tx pgx.Tx, userID int64, resourceType, resourceID string, permissionID int, grantedBy int64, expiresAt *time.Time, conditions json.RawMessage) (*ResourceGrant, error) {
	query := `
		INSERT INTO resource_grants (user_id, resource_type, resource_id, permission_id, granted_by, expires_at, conditions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, resource_type, resource_id, granted_by, granted_at, expires_at, conditions
	`
	var rg ResourceGrant
	err := tx.QueryRow(ctx, query, userID, resourceType, resourceID, permissionID, grantedBy, expiresAt, conditions).Scan(
		&rg.ID, &rg.UserID, &rg.ResourceType, &rg.ResourceID,
		&rg.GrantedBy, &rg.GrantedAt, &rg.ExpiresAt, &rg.Conditions,
	)
	if err != nil {
		return nil, err
	}

	// Fetch permission details
	p, err := r.GetPermission(ctx, tx, permissionID)
	if err != nil {
		return nil, err
	}
	rg.Permission = *p

	return &rg, nil
}

// RevokeResourceGrant revokes a resource-specific grant
func (r *Repository) RevokeResourceGrant(ctx context.Context, tx pgx.Tx, grantID int64) error {
	_, err := tx.Exec(ctx, "DELETE FROM resource_grants WHERE id = $1", grantID)
	return err
}

// ListEnhancedABACPolicies returns all ABAC policies
func (r *Repository) ListEnhancedABACPolicies(ctx context.Context, tx pgx.Tx) ([]EnhancedABACPolicy, error) {
	query := `
		SELECT id, name, COALESCE(description,''), policy_type, target_id, effect, priority, conditions, is_active, created_at, updated_at
		FROM abac_policies
		WHERE is_active = true
		ORDER BY priority DESC, id
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []EnhancedABACPolicy
	for rows.Next() {
		var p EnhancedABACPolicy
		var conditions json.RawMessage
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.PolicyType, &p.TargetID,
			&p.Effect, &p.Priority, &conditions, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if conditions != nil {
			p.Conditions = conditions
		}
		policies = append(policies, p)
	}
	return policies, nil
}

// CreateEnhancedABACPolicy creates a new ABAC policy
func (r *Repository) CreateEnhancedABACPolicy(ctx context.Context, tx pgx.Tx, name, description, policyType string, targetID *int64, effect string, priority int, conditions json.RawMessage) (*EnhancedABACPolicy, error) {
	query := `
		INSERT INTO abac_policies (name, description, policy_type, target_id, effect, priority, conditions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, description, policy_type, target_id, effect, priority, conditions, is_active, created_at, updated_at
	`
	var p EnhancedABACPolicy
	err := tx.QueryRow(ctx, query, name, description, policyType, targetID, effect, priority, conditions).Scan(
		&p.ID, &p.Name, &p.Description, &p.PolicyType, &p.TargetID,
		&p.Effect, &p.Priority, &p.Conditions, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateEnhancedABACPolicy updates an existing ABAC policy
func (r *Repository) UpdateEnhancedABACPolicy(ctx context.Context, tx pgx.Tx, policyID int64, name, description string, effect string, priority int, conditions json.RawMessage) (*EnhancedABACPolicy, error) {
	query := `
		UPDATE abac_policies
		SET name = $1, description = $2, effect = $3, priority = $4, conditions = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING id, name, description, policy_type, target_id, effect, priority, conditions, is_active, created_at, updated_at
	`
	var p EnhancedABACPolicy
	err := tx.QueryRow(ctx, query, name, description, effect, priority, conditions, policyID).Scan(
		&p.ID, &p.Name, &p.Description, &p.PolicyType, &p.TargetID,
		&p.Effect, &p.Priority, &p.Conditions, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeleteEnhancedABACPolicy deletes an ABAC policy
func (r *Repository) DeleteEnhancedABACPolicy(ctx context.Context, tx pgx.Tx, policyID int64) error {
	_, err := tx.Exec(ctx, "DELETE FROM abac_policies WHERE id = $1", policyID)
	return err
}

// RefreshPermissionCache refreshes the materialized view
func (r *Repository) RefreshPermissionCache(ctx context.Context) error {
	// Try to refresh the materialized view concurrently
	_, err := r.pool.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache")
	if err != nil {
		// If concurrent refresh fails, try regular refresh
		_, err = r.pool.Exec(ctx, "REFRESH MATERIALIZED VIEW user_effective_permissions_cache")
		if err != nil {
			// If materialized view doesn't exist, log but don't fail
			// This allows the system to work without the cache
			return nil
		}
	}
	return nil
}

// GetUserEffectivePermissionsNew returns all effective permissions for a user (new system)
func (r *Repository) GetUserEffectivePermissionsNew(ctx context.Context, tx pgx.Tx, userID int64) (*EffectivePermissions, error) {
	// Get user roles
	rolesQuery := `
		SELECT r.name 
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND ur.is_active = true
	`
	rows, err := tx.Query(ctx, rolesQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, err
		}
		roles = append(roles, roleName)
	}

	// Get effective permissions from cache
	permsQuery := `
		SELECT DISTINCT resource, action, scope
		FROM user_effective_permissions_cache
		WHERE user_id = $1
		ORDER BY resource, action, scope
	`
	permRows, err := tx.Query(ctx, permsQuery, userID)
	if err != nil {
		return nil, err
	}
	defer permRows.Close()

	var permissions []string
	for permRows.Next() {
		var resource, action, scope string
		if err := permRows.Scan(&resource, &action, &scope); err != nil {
			return nil, err
		}
		permStr := fmt.Sprintf("%s:%s:%s", resource, action, scope)
		permissions = append(permissions, permStr)
	}

	// Get category scopes
	categoryScopes, err := r.GetUserCategoryScopes(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	// Check if super admin
	var isSuperAdmin bool
	err = tx.QueryRow(ctx, "SELECT COALESCE(is_super_admin, false) FROM users WHERE id = $1", userID).Scan(&isSuperAdmin)
	if err != nil {
		return nil, err
	}

	return &EffectivePermissions{
		UserID:         userID,
		IsSuperAdmin:   isSuperAdmin,
		Roles:          roles,
		Permissions:    permissions,
		CategoryScopes: categoryScopes,
	}, nil
}

// ─── LEGACY: Maintain backward compatibility methods ─────────────────────────

// HasRoleGrant checks if a user has a specific role grant (legacy)
func (r *Repository) HasRoleGrant(ctx context.Context, tx pgx.Tx, userID int64, action string) (bool, error) {
	// Try new system first
	query := `
		SELECT EXISTS (
			SELECT 1 FROM user_effective_permissions_cache
			WHERE user_id = $1
			AND action = $2
			AND (expires_at IS NULL OR expires_at > NOW())
		)
	`
	var exists bool
	err := tx.QueryRow(ctx, query, userID, action).Scan(&exists)
	if err == nil {
		return exists, nil
	}

	// Fallback to old system
	query = `
		SELECT EXISTS (
			SELECT 1 FROM user_roles ur
			JOIN role_menu_actions rma ON rma.role_id = ur.role_id
			JOIN menu_actions ma ON ma.id = rma.menu_action_id
			WHERE ur.user_id = $1 AND ur.is_active = true AND ma.action = $2
		)
	`
	err = tx.QueryRow(ctx, query, userID, action).Scan(&exists)
	return exists, err
}

// FindActiveOverride finds an active override for a user (legacy)
func (r *Repository) FindActiveOverride(ctx context.Context, tx pgx.Tx, userID int64, action string) (*ActiveOverride, error) {
	query := `
		SELECT upo.effect, upo.menu_action_id, COALESCE(upo.reason,''), upo.valid_until
		FROM user_permission_overrides upo
		WHERE upo.user_id = $1 AND upo.is_active = true
		  AND (upo.valid_from IS NULL OR upo.valid_from <= NOW())
		  AND (upo.valid_until IS NULL OR upo.valid_until > NOW())
		LIMIT 1
	`
	var override ActiveOverride
	err := tx.QueryRow(ctx, query, userID).Scan(&override.Effect, &override.MenuActionID, &override.Reason, &override.ValidUntil)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &override, nil
}

// FindABACPolicies finds ABAC policies for a user (legacy)
func (r *Repository) FindABACPolicies(ctx context.Context, tx pgx.Tx, userID int64) ([]ABACPolicy, error) {
	query := `
		SELECT id, user_id, attribute, value, is_active
		FROM abac_policies
		WHERE user_id = $1 AND is_active = true
	`
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []ABACPolicy
	for rows.Next() {
		var p ABACPolicy
		if err := rows.Scan(&p.ID, &p.UserID, &p.Attribute, &p.Value, &p.IsActive); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

// InsertAuditLog inserts an audit log entry
func (r *Repository) InsertAuditLog(ctx context.Context, tx pgx.Tx, entry AuditEntry) error {
	query := `
		INSERT INTO permission_audit_log (user_id, menu_action_id, permission_string, action_name, decision, reason, ip_address, user_agent, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	// Try to parse menu_action_id from action_name if it's numeric
	var menuActionID *int
	if entry.ActionName != "" {
		var id int
		if _, err := fmt.Sscanf(entry.ActionName, "%d", &id); err == nil {
			menuActionID = &id
		}
	}
	
	_, err := tx.Exec(ctx, query,
		entry.UserID, 
		menuActionID, 
		entry.ActionName, // Use action_name as permission_string for new system
		entry.ActionName, 
		entry.Decision,
		entry.Reason, 
		entry.IPAddress, 
		entry.UserAgent, 
		entry.RequestID,
	)
	return err
}

// ListAuditLogs lists audit log entries
func (r *Repository) ListAuditLogs(ctx context.Context, tx pgx.Tx, limit, offset int) ([]AuditEntry, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM permission_audit_log`
	err := tx.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated logs
	query := `
		SELECT 
			al.id, al.user_id, COALESCE(u.display_name, u.email, ''), COALESCE(u.email, ''),
			al.action_name, al.decision, COALESCE(al.reason,''), 
			COALESCE(al.ip_address::text, ''), COALESCE(al.user_agent, ''), 
			COALESCE(al.request_id, ''), al.created_at
		FROM permission_audit_log al
		LEFT JOIN users u ON u.id = al.user_id
		ORDER BY al.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := tx.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []AuditEntry
	for rows.Next() {
		var log AuditEntry
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.UserName, &log.UserEmail,
			&log.ActionName, &log.Decision, &log.Reason,
			&log.IPAddress, &log.UserAgent, &log.RequestID, &log.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// ListStaffWithRoles lists staff with their roles
func (r *Repository) ListStaffWithRoles(ctx context.Context, tx pgx.Tx, search string) ([]StaffUserRoleSummary, error) {
	query := `
		SELECT 
			u.id, COALESCE(u.display_name, u.email, ''), u.email,
			ur.role_id, r.name, COALESCE(u.is_super_admin, false), COALESCE(u.is_active, true)
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id AND ur.is_active = true
		LEFT JOIN roles r ON r.id = ur.role_id
		WHERE u.is_staff = true
		  AND ($1 = '' OR u.email ILIKE $2 OR u.display_name ILIKE $2)
		ORDER BY u.email
	`
	searchPattern := "%" + search + "%"
	rows, err := tx.Query(ctx, query, search, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staff []StaffUserRoleSummary
	for rows.Next() {
		var s StaffUserRoleSummary
		if err := rows.Scan(
			&s.UserID, &s.DisplayName, &s.Email,
			&s.RoleID, &s.RoleName, &s.IsSuperAdmin, &s.IsActive,
		); err != nil {
			return nil, err
		}
		staff = append(staff, s)
	}
	return staff, nil
}

// ListStaffWithRolesDirect lists staff with their roles (direct pool access)
func (r *Repository) ListStaffWithRolesDirect(ctx context.Context, search string) ([]StaffUserRoleSummary, error) {
	query := `
		SELECT 
			u.id, COALESCE(u.display_name, u.email, ''), u.email,
			ur.role_id, r.name, COALESCE(u.is_super_admin, false), COALESCE(u.is_active, true)
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id AND ur.is_active = true
		LEFT JOIN roles r ON r.id = ur.role_id
		WHERE u.is_staff = true
		  AND ($1 = '' OR u.email ILIKE $2 OR u.display_name ILIKE $2)
		ORDER BY u.email
	`
	searchPattern := "%" + search + "%"
	rows, err := r.pool.Query(ctx, query, search, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staff []StaffUserRoleSummary
	for rows.Next() {
		var s StaffUserRoleSummary
		if err := rows.Scan(
			&s.UserID, &s.DisplayName, &s.Email,
			&s.RoleID, &s.RoleName, &s.IsSuperAdmin, &s.IsActive,
		); err != nil {
			return nil, err
		}
		staff = append(staff, s)
	}
	return staff, nil
}

// CreateOverride creates a permission override (legacy)
func (r *Repository) CreateOverride(ctx context.Context, tx pgx.Tx, userID int64, menuActionID int, effect, reason string, validFrom, validUntil *time.Time, grantedBy int64) error {
	query := `
		INSERT INTO user_permission_overrides (user_id, menu_action_id, effect, reason, valid_from, valid_until, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, menu_action_id)
		DO UPDATE SET 
			effect = EXCLUDED.effect,
			reason = EXCLUDED.reason,
			valid_from = EXCLUDED.valid_from,
			valid_until = EXCLUDED.valid_until,
			granted_by = EXCLUDED.granted_by
	`
	_, err := tx.Exec(ctx, query, userID, menuActionID, effect, reason, validFrom, validUntil, grantedBy)
	return err
}

// ─── Enhanced RBAC/ABAC Repository Methods ───────────────────────────

// ListAllPermissions lists all permissions in the system
func (r *Repository) ListAllPermissions(ctx context.Context, tx pgx.Tx) ([]Permission, error) {
	query := `
		SELECT id, resource, action, scope, description, is_system
		FROM permissions
		ORDER BY resource, action, scope
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(
			&p.ID, &p.Resource, &p.Action, &p.Scope, &p.Description, &p.IsSystem,
		); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

// RevokeRolePermissionsBulk revokes permissions from a role (bulk)
func (r *Repository) RevokeRolePermissionsBulk(ctx context.Context, tx pgx.Tx, roleID int, permissionIDs []int, revokedBy int64) error {
	query := `
		DELETE FROM role_permissions
		WHERE role_id = $1 AND permission_id = ANY($2)
	`
	_, err := tx.Exec(ctx, query, roleID, permissionIDs)
	return err
}

// AssignDynamicPermissions assigns temporary dynamic permissions to a role
func (r *Repository) AssignDynamicPermissions(ctx context.Context, tx pgx.Tx, roleID int, permissionIDs []int, assignedBy int64, expiresAt *time.Time, reason string) error {
	query := `
		INSERT INTO dynamic_permission_assignments (role_id, permission_id, assigned_by, expires_at, reason)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (role_id, permission_id) DO UPDATE SET
			expires_at = EXCLUDED.expires_at,
			reason = EXCLUDED.reason,
			is_active = true
	`
	for _, permID := range permissionIDs {
		_, err := tx.Exec(ctx, query, roleID, permID, assignedBy, expiresAt, reason)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetDynamicPermissions gets dynamic permissions for a role
func (r *Repository) GetDynamicPermissions(ctx context.Context, tx pgx.Tx, roleID int) ([]DynamicPermissionAssignment, error) {
	query := `
		SELECT id, role_id, permission_id, assigned_by, assigned_at, expires_at, is_active, reason
		FROM dynamic_permission_assignments
		WHERE role_id = $1 AND is_active = true
		ORDER BY assigned_at DESC
	`
	rows, err := tx.Query(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []DynamicPermissionAssignment
	for rows.Next() {
		var a DynamicPermissionAssignment
		if err := rows.Scan(
			&a.ID, &a.RoleID, &a.PermissionID, &a.AssignedBy, &a.AssignedAt, &a.ExpiresAt, &a.IsActive, &a.Reason,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, nil
}

// RevokeDynamicPermission revokes a dynamic permission assignment
func (r *Repository) RevokeDynamicPermission(ctx context.Context, tx pgx.Tx, assignmentID int, revokedBy int64) error {
	query := `
		UPDATE dynamic_permission_assignments
		SET is_active = false
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, assignmentID)
	return err
}

// ─── Approval Workflow Repository Methods ───────────────────────────

// CreateApprovalRequest creates a new approval request
func (r *Repository) CreateApprovalRequest(ctx context.Context, tx pgx.Tx, resourceType string, resourceID int64, requestedBy int64, comments string) (int, error) {
	query := `
		INSERT INTO approval_workflows (resource_type, resource_id, requested_by, comments)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id int
	err := tx.QueryRow(ctx, query, resourceType, resourceID, requestedBy, comments).Scan(&id)
	return id, err
}

// ListApprovalRequests lists approval requests for a user
func (r *Repository) ListApprovalRequests(ctx context.Context, tx pgx.Tx, userID int64) ([]ApprovalRequest, error) {
	query := `
		SELECT id, resource_type, resource_id, requested_by, requested_at, status, approved_by, approved_at, rejection_reason, comments
		FROM approval_workflows
		WHERE requested_by = $1
		ORDER BY requested_at DESC
	`
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []ApprovalRequest
	for rows.Next() {
		var req ApprovalRequest
		if err := rows.Scan(
			&req.ID, &req.ResourceType, &req.ResourceID, &req.RequestedBy, &req.RequestedAt, &req.Status,
			&req.ApprovedBy, &req.ApprovedAt, &req.RejectionReason, &req.Comments,
		); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// GetPendingApprovals gets pending approval requests for a user
func (r *Repository) GetPendingApprovals(ctx context.Context, tx pgx.Tx, userID int64) ([]ApprovalRequest, error) {
	query := `
		SELECT aw.id, aw.resource_type, aw.resource_id, aw.requested_by, aw.requested_at, aw.status, aw.approved_by, aw.approved_at, aw.rejection_reason, aw.comments
		FROM approval_workflows aw
		JOIN approval_assignments aa ON aa.user_id = $1 AND aa.resource_type = aw.resource_type AND aa.can_approve = true AND aa.is_active = true
		WHERE aw.status = 'pending'
		ORDER BY aw.requested_at DESC
	`
	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []ApprovalRequest
	for rows.Next() {
		var req ApprovalRequest
		if err := rows.Scan(
			&req.ID, &req.ResourceType, &req.ResourceID, &req.RequestedBy, &req.RequestedAt, &req.Status,
			&req.ApprovedBy, &req.ApprovedAt, &req.RejectionReason, &req.Comments,
		); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// ApproveRequest approves an approval request
func (r *Repository) ApproveRequest(ctx context.Context, tx pgx.Tx, requestID int, approvedBy int64, comments string) error {
	query := `
		UPDATE approval_workflows
		SET status = 'approved', approved_by = $1, approved_at = NOW(), comments = COALESCE($2, comments)
		WHERE id = $3 AND status = 'pending'
	`
	_, err := tx.Exec(ctx, query, approvedBy, comments, requestID)
	return err
}

// RejectRequest rejects an approval request
func (r *Repository) RejectRequest(ctx context.Context, tx pgx.Tx, requestID int, rejectedBy int64, reason string) error {
	query := `
		UPDATE approval_workflows
		SET status = 'rejected', approved_by = $1, approved_at = NOW(), rejection_reason = $2
		WHERE id = $3 AND status = 'pending'
	`
	_, err := tx.Exec(ctx, query, rejectedBy, reason, requestID)
	return err
}

// ─── Approval Assignment Repository Methods ───────────────────────────

// AssignApprovalRights assigns approval rights to a user
func (r *Repository) AssignApprovalRights(ctx context.Context, tx pgx.Tx, userID int64, resourceType string, canApprove bool, assignedBy int64, expiresAt *time.Time) error {
	query := `
		INSERT INTO approval_assignments (user_id, resource_type, can_approve, assigned_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, resource_type) DO UPDATE SET
			can_approve = EXCLUDED.can_approve,
			expires_at = EXCLUDED.expires_at,
			is_active = true
	`
	_, err := tx.Exec(ctx, query, userID, resourceType, canApprove, assignedBy, expiresAt)
	return err
}

// GetApprovalAssignments gets all approval assignments
func (r *Repository) GetApprovalAssignments(ctx context.Context, tx pgx.Tx) ([]ApprovalAssignment, error) {
	query := `
		SELECT id, user_id, resource_type, can_approve, assigned_by, assigned_at, expires_at, is_active
		FROM approval_assignments
		WHERE is_active = true
		ORDER BY assigned_at DESC
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []ApprovalAssignment
	for rows.Next() {
		var a ApprovalAssignment
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.ResourceType, &a.CanApprove, &a.AssignedBy, &a.AssignedAt, &a.ExpiresAt, &a.IsActive,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, nil
}

// RevokeApprovalRights revokes approval rights
func (r *Repository) RevokeApprovalRights(ctx context.Context, tx pgx.Tx, assignmentID int, revokedBy int64) error {
	query := `
		UPDATE approval_assignments
		SET is_active = false
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, assignmentID)
	return err
}

// CreateABACPolicy creates an ABAC policy (legacy)
func (r *Repository) CreateABACPolicy(ctx context.Context, tx pgx.Tx, userID int64, attribute string, value json.RawMessage) error {
	query := `
		INSERT INTO abac_policies (user_id, attribute, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, attribute)
		DO UPDATE SET value = EXCLUDED.value
	`
	_, err := tx.Exec(ctx, query, userID, attribute, value)
	return err
}

func (r *Repository) AssignUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID int64, districtIDs []int) error {
	return r.AssignUserCategoryScopes(ctx, tx, userID, districtIDs, 1)
}

// ─── Effective Permissions Resolution ──────────

func (r *Repository) GetUserEffectivePermissions(ctx context.Context, tx pgx.Tx, userID int64) (*EffectivePermissions, error) {
	// 1. Fetch user roles
	rolesQuery := `
		SELECT r.id, r.name
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND ur.is_active = TRUE
		ORDER BY r.id ASC
	`
	rRows, err := tx.Query(ctx, rolesQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rRows.Close()

	var roleNames []string
	var isSuperAdmin bool
	_ = tx.QueryRow(ctx, `SELECT COALESCE(is_super_admin, FALSE) FROM users WHERE id = $1`, userID).Scan(&isSuperAdmin)
	for rRows.Next() {
		var roleID int
		var roleName string
		if err := rRows.Scan(&roleID, &roleName); err == nil {
			roleNames = append(roleNames, roleName)
			if strings.EqualFold(roleName, "super_admin") || strings.EqualFold(roleName, "Super Administrator") {
				isSuperAdmin = true
			}
		}
	}

	// 2. Fetch distinct granted menu action strings
	permsQuery := `
		SELECT DISTINCT (m.name || '.' || ma.action) as perm
		FROM role_menu_actions rma
		JOIN user_roles ur ON ur.role_id = rma.role_id AND ur.is_active = TRUE
		JOIN menu_actions ma ON ma.id = rma.menu_action_id
		JOIN menus m ON m.id = ma.menu_id
		WHERE ur.user_id = $1
	`
	pRows, err := tx.Query(ctx, permsQuery, userID)
	if err != nil {
		return nil, err
	}
	defer pRows.Close()

	permMap := make(map[string]bool)
	for pRows.Next() {
		var p string
		if err := pRows.Scan(&p); err == nil {
			permMap[p] = true
		}
	}

	// 3. Apply active user permission overrides
	overridesQuery := `
		SELECT (m.name || '.' || ma.action) as perm, upo.effect
		FROM user_permission_overrides upo
		JOIN menu_actions ma ON ma.id = upo.menu_action_id
		JOIN menus m ON m.id = ma.menu_id
		WHERE upo.user_id = $1 AND upo.is_active = TRUE
		  AND (upo.valid_from IS NULL OR upo.valid_from <= NOW())
		  AND (upo.valid_until IS NULL OR upo.valid_until > NOW())
	`
	ovRows, err := tx.Query(ctx, overridesQuery, userID)
	if err == nil {
		defer ovRows.Close()
		for ovRows.Next() {
			var perm, effect string
			if err := ovRows.Scan(&perm, &effect); err == nil {
				if effect == "GRANT" {
					permMap[perm] = true
				} else if effect == "REVOKE" {
					delete(permMap, perm)
				}
			}
		}
	}

	permsList := make([]string, 0, len(permMap))
	for p := range permMap {
		permsList = append(permsList, p)
	}

	if isSuperAdmin {
		allRows, err := tx.Query(ctx, `SELECT DISTINCT (m.name || '.' || ma.action) FROM menus m JOIN menu_actions ma ON ma.menu_id = m.id WHERE m.is_active IS NOT FALSE`)
		if err == nil {
			defer allRows.Close()
			permMap = map[string]bool{}
			for allRows.Next() {
				var p string
				if allRows.Scan(&p) == nil {
					permMap[p] = true
				}
			}
			permsList = permsList[:0]
			for p := range permMap {
				permsList = append(permsList, p)
			}
		}
	}

	// 4. Fetch Category Scopes
	catScopes, _ := r.GetUserCategoryScopes(ctx, tx, userID)

	return &EffectivePermissions{
		UserID:         userID,
		IsSuperAdmin:   isSuperAdmin,
		Roles:          roleNames,
		Permissions:    permsList,
		CategoryScopes: catScopes,
	}, nil
}
