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

func (r *Repository) ListMenus(ctx context.Context, tx pgx.Tx) ([]Menu, error) {
	query := `
		SELECT id, name, label, parent_id, COALESCE(icon,''), COALESCE(path,''), sort_order, COALESCE(is_active, true)
		FROM menus
		WHERE is_active IS NOT FALSE
		ORDER BY sort_order ASC, id ASC
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		var m Menu
		if err := rows.Scan(&m.ID, &m.Name, &m.Label, &m.ParentID, &m.Icon, &m.Path, &m.SortOrder, &m.IsActive); err != nil {
			return nil, err
		}
		menus = append(menus, m)
	}
	return menus, rows.Err()
}

func (r *Repository) ListMenusDirect(ctx context.Context) ([]Menu, error) {
	query := `
		SELECT id, name, label, parent_id, COALESCE(icon,''), COALESCE(path,''), sort_order, COALESCE(is_active, true)
		FROM menus
		WHERE is_active IS NOT FALSE
		ORDER BY sort_order ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		var m Menu
		if err := rows.Scan(&m.ID, &m.Name, &m.Label, &m.ParentID, &m.Icon, &m.Path, &m.SortOrder, &m.IsActive); err != nil {
			return nil, err
		}
		menus = append(menus, m)
	}
	return menus, rows.Err()
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
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
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
	for rRows.Next() {
		var roleID int
		var roleName string
		if err := rRows.Scan(&roleID, &roleName); err == nil {
			roleNames = append(roleNames, roleName)
			if roleID == 1 || strings.EqualFold(roleName, "Super Administrator") {
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

// ─── Policy & Evaluation Queries ───────────────

func (r *Repository) HasRoleGrant(ctx context.Context, tx pgx.Tx, userID int64, actionName string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM user_roles ur
			JOIN role_menu_actions rma ON rma.role_id = ur.role_id
			JOIN menu_actions ma ON ma.id = rma.menu_action_id
			JOIN menus m ON m.id = ma.menu_id
			WHERE ur.user_id = $1
			  AND ur.is_active = TRUE
			  AND (m.name || '.' || ma.action) = $2
		)
	`
	var hasGrant bool
	err := tx.QueryRow(ctx, query, userID, actionName).Scan(&hasGrant)
	return hasGrant, err
}

func (r *Repository) FindActiveOverride(ctx context.Context, tx pgx.Tx, userID int64, actionName string) (*ActiveOverride, error) {
	query := `
		SELECT upo.effect, upo.menu_action_id, upo.reason, upo.valid_until
		FROM user_permission_overrides upo
		JOIN menu_actions ma ON ma.id = upo.menu_action_id
		JOIN menus m ON m.id = ma.menu_id
		WHERE upo.user_id = $1
		  AND (m.name || '.' || ma.action) = $2
		  AND upo.is_active = TRUE
		  AND (upo.valid_from IS NULL OR upo.valid_from <= NOW())
		  AND (upo.valid_until IS NULL OR upo.valid_until > NOW())
		ORDER BY upo.created_at DESC
		LIMIT 1
	`
	var ov ActiveOverride
	err := tx.QueryRow(ctx, query, userID, actionName).Scan(
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

func (r *Repository) CreateOverride(ctx context.Context, tx pgx.Tx, userID int64, menuActionID int, effect, reason string, validFrom, validUntil *time.Time, grantedBy int64) error {
	query := `
		INSERT INTO user_permission_overrides
			(user_id, menu_action_id, effect, reason, valid_from, valid_until, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := tx.Exec(ctx, query, userID, menuActionID, effect, reason, validFrom, validUntil, grantedBy)
	return err
}

func (r *Repository) FindABACPolicies(ctx context.Context, tx pgx.Tx, userID int64) ([]ABACPolicy, error) {
	query := `
		SELECT id, user_id, attribute, value, is_active
		FROM abac_policies
		WHERE user_id = $1 AND is_active = TRUE
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
	return policies, rows.Err()
}

func (r *Repository) CreateABACPolicy(ctx context.Context, tx pgx.Tx, userID int64, attribute string, value json.RawMessage) error {
	query := `
		INSERT INTO abac_policies (user_id, attribute, value)
		VALUES ($1, $2, $3)
	`
	_, err := tx.Exec(ctx, query, userID, attribute, value)
	return err
}

// ─── Audit Log Queries ─────────────────────────

func (r *Repository) InsertAuditLog(ctx context.Context, tx pgx.Tx, log AuditEntry) error {
	query := `
		INSERT INTO permission_audit_log
			(user_id, action_name, decision, reason, ip_address, user_agent, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := tx.Exec(ctx, query,
		log.UserID, log.ActionName, log.Decision,
		log.Reason, log.IPAddress, log.UserAgent, log.RequestID,
	)
	return err
}

func (r *Repository) ListAuditLogs(ctx context.Context, tx pgx.Tx, limit, offset int) ([]AuditEntry, int64, error) {
	var total int64
	_ = tx.QueryRow(ctx, "SELECT COUNT(*) FROM permission_audit_log").Scan(&total)

	query := `
		SELECT pal.id, pal.user_id, COALESCE(u.display_name, 'System Staff'), COALESCE(u.email, ''),
		       pal.action_name, pal.decision, COALESCE(pal.reason, ''),
		       COALESCE(host(pal.ip_address), ''), COALESCE(pal.user_agent, ''), COALESCE(pal.request_id, ''), pal.created_at
		FROM permission_audit_log pal
		LEFT JOIN users u ON u.id = pal.user_id
		ORDER BY pal.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := tx.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := []AuditEntry{}
	for rows.Next() {
		var l AuditEntry
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.UserName, &l.UserEmail, &l.ActionName, &l.Decision, &l.Reason,
			&l.IPAddress, &l.UserAgent, &l.RequestID, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	return logs, total, rows.Err()
}

// ─── Staff Roles & Scoping Summary ─────────────

func (r *Repository) ListStaffWithRoles(ctx context.Context, tx pgx.Tx, search string) ([]StaffUserRoleSummary, error) {
	query := `
		SELECT u.id, COALESCE(u.display_name, 'Staff Member'), COALESCE(u.email, ''),
		       u.is_super_admin, u.is_active,
		       r.id, r.name
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id AND ur.is_active = TRUE
		LEFT JOIN roles r ON r.id = ur.role_id
		WHERE u.is_staff = TRUE OR u.is_super_admin = TRUE
	`
	var rows pgx.Rows
	var err error
	if strings.TrimSpace(search) != "" {
		query += ` AND (u.display_name ILIKE $1 OR u.email ILIKE $1)
		           ORDER BY u.is_super_admin DESC, u.id ASC`
		rows, err = tx.Query(ctx, query, "%"+strings.TrimSpace(search)+"%")
	} else {
		query += ` ORDER BY u.is_super_admin DESC, u.id ASC`
		rows, err = tx.Query(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []StaffUserRoleSummary
	for rows.Next() {
		var s StaffUserRoleSummary
		var roleID *int
		var roleName *string
		if err := rows.Scan(&s.UserID, &s.DisplayName, &s.Email, &s.IsSuperAdmin, &s.IsActive, &roleID, &roleName); err != nil {
			return nil, err
		}
		s.RoleID = roleID
		s.RoleName = roleName
		list = append(list, s)
	}

	for i := range list {
		scopes, _ := r.GetUserCategoryScopes(ctx, tx, list[i].UserID)
		if scopes == nil {
			scopes = []CategoryScope{}
		}
		list[i].CategoryScopes = scopes
	}

	return list, nil
}

func (r *Repository) ListStaffWithRolesDirect(ctx context.Context, search string) ([]StaffUserRoleSummary, error) {
	query := `
		SELECT u.id, COALESCE(u.display_name, 'Staff Member'), COALESCE(u.email, ''),
		       u.is_super_admin, u.is_active,
		       r.id, r.name
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id AND ur.is_active = TRUE
		LEFT JOIN roles r ON r.id = ur.role_id
		WHERE u.is_staff = TRUE OR u.is_super_admin = TRUE
	`
	var rows pgx.Rows
	var err error
	if strings.TrimSpace(search) != "" {
		query += ` AND (u.display_name ILIKE $1 OR u.email ILIKE $1)
		           ORDER BY u.is_super_admin DESC, u.id ASC`
		rows, err = r.pool.Query(ctx, query, "%"+strings.TrimSpace(search)+"%")
	} else {
		query += ` ORDER BY u.is_super_admin DESC, u.id ASC`
		rows, err = r.pool.Query(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []StaffUserRoleSummary
	for rows.Next() {
		var s StaffUserRoleSummary
		var roleID *int
		var roleName *string
		if err := rows.Scan(&s.UserID, &s.DisplayName, &s.Email, &s.IsSuperAdmin, &s.IsActive, &roleID, &roleName); err != nil {
			return nil, err
		}
		s.RoleID = roleID
		s.RoleName = roleName
		list = append(list, s)
	}

	for i := range list {
		catQuery := `
			SELECT c.id, c.name, c.slug, COALESCE(c.path, c.name), c.level
			FROM categories c
			JOIN user_category_scopes ucs ON ucs.category_id = c.id
			WHERE ucs.user_id = $1
			ORDER BY c.level ASC, c.name ASC
		`
		cRows, err := r.pool.Query(ctx, catQuery, list[i].UserID)
		var scopes []CategoryScope
		if err == nil {
			for cRows.Next() {
				var cs CategoryScope
				if err := cRows.Scan(&cs.CategoryID, &cs.Name, &cs.Slug, &cs.Path, &cs.Level); err == nil {
					scopes = append(scopes, cs)
				}
			}
			cRows.Close()
		}
		list[i].CategoryScopes = scopes
	}

	return list, nil
}
