package iam

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// MigrationService handles the transition from old menu-based IAM to new resource-action-scope IAM
type MigrationService struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewMigrationService creates a new migration service
func NewMigrationService(pool *pgxpool.Pool, logger zerolog.Logger) *MigrationService {
	return &MigrationService{
		pool:   pool,
		logger: logger.With().Str("module", "iam_migration").Logger(),
	}
}

// MigrationResult represents the result of a migration operation
type MigrationResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	MigratedRoles int     `json:"migrated_roles"`
	MigratedPerms int     `json:"migrated_permissions"`
	Errors       []string `json:"errors,omitempty"`
	Duration     string   `json:"duration"`
}

// MigrateMenuActionsToPermissions migrates existing menu actions to new permission format
func (m *MigrationService) MigrateMenuActionsToPermissions(ctx context.Context) (*MigrationResult, error) {
	startTime := time.Now()
	result := &MigrationResult{
		Errors: []string{},
	}

	m.logger.Info().Msg("Starting migration of menu actions to permissions")

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Map menu names to resource names
	menuToResource := map[string]string{
		"articles":      "news",
		"media_library": "media",
		"users":         "user",
		"categories":    "category",
		"tags":          "tag",
		"roles":         "iam",
		"analytics":     "analytics",
		"settings":      "settings",
		"comments":      "comments",
		"notifications": "notifications",
		"live_blogs":    "liveblog",
		"web_stories":   "webstory",
		"e_paper":       "epaper",
	}

	// Migrate menu actions to permissions
	query := `
		SELECT DISTINCT 
			COALESCE(m.name, '') as menu_name,
			COALESCE(ma.action, '') as action,
			COALESCE(ma.label, '') as label
		FROM menu_actions ma
		JOIN menus m ON m.id = ma.menu_id
		WHERE m.name IS NOT NULL AND ma.action IS NOT NULL
		ORDER BY m.name, ma.action
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query menu actions: %w", err)
	}
	defer rows.Close()

	permCount := 0
	for rows.Next() {
		var menuName, action, label string
		if err := rows.Scan(&menuName, &action, &label); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to scan menu action: %v", err))
			continue
		}

		// Map menu to resource
		resource, ok := menuToResource[menuName]
		if !ok {
			resource = menuName // Use menu name as resource if no mapping exists
		}

		// Normalize action to lowercase
		normalizedAction := strings.ToLower(action)

		// Create permission
		description := label
		if description == "" {
			description = fmt.Sprintf("%s %s on %s", action, menuName, resource)
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO permissions (resource, action, scope, description, is_system)
			VALUES ($1, $2, 'all', $3, true)
			ON CONFLICT (resource, action, scope) DO NOTHING
		`, resource, normalizedAction, description)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create permission for %s:%s: %v", resource, normalizedAction, err))
			continue
		}

		permCount++
		m.logger.Debug().Str("resource", resource).Str("action", normalizedAction).Msg("Created permission")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully migrated %d permissions", permCount)
	result.MigratedPerms = permCount
	result.Duration = time.Since(startTime).String()

	m.logger.Info().Int("permissions", permCount).Str("duration", result.Duration).Msg("Migration completed")

	return result, nil
}

// MigrateRolePermissions migrates existing role-menu-action grants to new role-permission grants
func (m *MigrationService) MigrateRolePermissions(ctx context.Context) (*MigrationResult, error) {
	startTime := time.Now()
	result := &MigrationResult{
		Errors: []string{},
	}

	m.logger.Info().Msg("Starting migration of role permissions")

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Map menu names to resource names
	menuToResource := map[string]string{
		"articles":      "news",
		"media_library": "media",
		"users":         "user",
		"categories":    "category",
		"tags":          "tag",
		"roles":         "iam",
		"analytics":     "analytics",
		"settings":      "settings",
		"comments":      "comments",
		"notifications": "notifications",
		"live_blogs":    "liveblog",
		"web_stories":   "webstory",
		"e_paper":       "epaper",
	}

	// Get all roles with their menu action grants
	query := `
		SELECT DISTINCT 
			r.id as role_id,
			r.name as role_name,
			COALESCE(m.name, '') as menu_name,
			COALESCE(ma.action, '') as action
		FROM roles r
		JOIN role_menu_actions rma ON rma.role_id = r.id
		JOIN menu_actions ma ON ma.id = rma.menu_action_id
		JOIN menus m ON m.id = ma.menu_id
		WHERE r.name IS NOT NULL AND m.name IS NOT NULL AND ma.action IS NOT NULL
		ORDER BY r.id, m.name, ma.action
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query role permissions: %w", err)
	}
	defer rows.Close()

	rolePermCount := 0
	processedRoles := make(map[int]bool)

	for rows.Next() {
		var roleID int
		var roleName, menuName, action string
		if err := rows.Scan(&roleID, &roleName, &menuName, &action); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to scan role permission: %v", err))
			continue
		}

		processedRoles[roleID] = true

		// Map menu to resource
		resource, ok := menuToResource[menuName]
		if !ok {
			resource = menuName
		}

		// Normalize action
		normalizedAction := strings.ToLower(action)

		// Find permission ID
		var permID int
		err := tx.QueryRow(ctx, `
			SELECT id FROM permissions 
			WHERE resource = $1 AND action = $2 AND scope = 'all'
		`, resource, normalizedAction).Scan(&permID)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Permission not found for %s:%s: %v", resource, normalizedAction, err))
			continue
		}

		// Create role permission grant
		_, err = tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id, granted_by, granted_at)
			VALUES ($1, $2, 1, NOW())
			ON CONFLICT (role_id, permission_id) DO NOTHING
		`, roleID, permID)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create role permission for role %d: %v", roleID, err))
			continue
		}

		rolePermCount++
		m.logger.Debug().Int("role_id", roleID).Str("role", roleName).Str("resource", resource).Str("action", normalizedAction).Msg("Migrated role permission")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully migrated %d role permissions for %d roles", rolePermCount, len(processedRoles))
	result.MigratedRoles = len(processedRoles)
	result.MigratedPerms = rolePermCount
	result.Duration = time.Since(startTime).String()

	m.logger.Info().Int("roles", len(processedRoles)).Int("permissions", rolePermCount).Str("duration", result.Duration).Msg("Role migration completed")

	return result, nil
}

// MigrateUserOverrides migrates user permission overrides to new user permissions
func (m *MigrationService) MigrateUserOverrides(ctx context.Context) (*MigrationResult, error) {
	startTime := time.Now()
	result := &MigrationResult{
		Errors: []string{},
	}

	m.logger.Info().Msg("Starting migration of user overrides")

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Map menu names to resource names
	menuToResource := map[string]string{
		"articles":      "news",
		"media_library": "media",
		"users":         "user",
		"categories":    "category",
		"tags":          "tag",
		"roles":         "iam",
		"analytics":     "analytics",
		"settings":      "settings",
		"comments":      "comments",
		"notifications": "notifications",
		"live_blogs":    "liveblog",
		"web_stories":   "webstory",
		"e_paper":       "epaper",
	}

	// Get all user overrides
	query := `
		SELECT DISTINCT 
			upo.user_id,
			COALESCE(m.name, '') as menu_name,
			COALESCE(ma.action, '') as action,
			upo.effect,
			COALESCE(upo.reason, ''),
			upo.valid_from,
			upo.valid_until,
			COALESCE(upo.granted_by, 0)
		FROM user_permission_overrides upo
		JOIN menu_actions ma ON ma.id = upo.menu_action_id
		JOIN menus m ON m.id = ma.menu_id
		WHERE upo.is_active = true
		  AND (upo.valid_from IS NULL OR upo.valid_from <= NOW())
		  AND (upo.valid_until IS NULL OR upo.valid_until > NOW())
		ORDER BY upo.user_id, m.name, ma.action
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query user overrides: %w", err)
	}
	defer rows.Close()

	overrideCount := 0
	processedUsers := make(map[int64]bool)

	for rows.Next() {
		var userID int64
		var menuName, action, effect, reason string
		var validFrom, validUntil *time.Time
		var grantedBy int64

		if err := rows.Scan(&userID, &menuName, &action, &effect, &reason, &validFrom, &validUntil, &grantedBy); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to scan user override: %v", err))
			continue
		}

		processedUsers[userID] = true

		// Map menu to resource
		resource, ok := menuToResource[menuName]
		if !ok {
			resource = menuName
		}

		// Normalize action
		normalizedAction := strings.ToLower(action)

		// Find permission ID
		var permID int
		err := tx.QueryRow(ctx, `
			SELECT id FROM permissions 
			WHERE resource = $1 AND action = $2 AND scope = 'all'
		`, resource, normalizedAction).Scan(&permID)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Permission not found for %s:%s: %v", resource, normalizedAction, err))
			continue
		}

		// Create user permission
		_, err = tx.Exec(ctx, `
			INSERT INTO user_permissions (user_id, permission_id, effect, reason, valid_from, valid_until, granted_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (user_id, permission_id, effect) 
			DO UPDATE SET 
				reason = EXCLUDED.reason,
				valid_from = EXCLUDED.valid_from,
				valid_until = EXCLUDED.valid_until,
				granted_by = EXCLUDED.granted_by
		`, userID, permID, effect, reason, validFrom, validUntil, grantedBy)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create user permission for user %d: %v", userID, err))
			continue
		}

		overrideCount++
		m.logger.Debug().Int64("user_id", userID).Str("resource", resource).Str("action", normalizedAction).Str("effect", effect).Msg("Migrated user override")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully migrated %d user overrides for %d users", overrideCount, len(processedUsers))
	result.MigratedRoles = len(processedUsers)
	result.MigratedPerms = overrideCount
	result.Duration = time.Since(startTime).String()

	m.logger.Info().Int("users", len(processedUsers)).Int("overrides", overrideCount).Str("duration", result.Duration).Msg("User override migration completed")

	return result, nil
}

// RunFullMigration runs the complete migration process
func (m *MigrationService) RunFullMigration(ctx context.Context) (*MigrationResult, error) {
	m.logger.Info().Msg("Starting full IAM migration")

	// Step 1: Migrate menu actions to permissions
	permResult, err := m.MigrateMenuActionsToPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("menu actions migration failed: %w", err)
	}

	// Step 2: Migrate role permissions
	roleResult, err := m.MigrateRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("role permissions migration failed: %w", err)
	}

	// Step 3: Migrate user overrides
	userResult, err := m.MigrateUserOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("user overrides migration failed: %w", err)
	}

	// Step 4: Refresh permission cache
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction for cache refresh: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache")
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to refresh permission cache")
	} else {
		if err := tx.Commit(ctx); err != nil {
			m.logger.Warn().Err(err).Msg("Failed to commit cache refresh")
		}
	}

	// Combine results
	finalResult := &MigrationResult{
		Success: true,
		Message: "Full IAM migration completed successfully",
		MigratedRoles: roleResult.MigratedRoles + userResult.MigratedRoles,
		MigratedPerms: permResult.MigratedPerms + roleResult.MigratedPerms + userResult.MigratedPerms,
		Errors: append(append(permResult.Errors, roleResult.Errors...), userResult.Errors...),
		Duration: time.Since(time.Now()).String(),
	}

	m.logger.Info().Str("result", finalResult.Message).Int("total_roles", finalResult.MigratedRoles).Int("total_perms", finalResult.MigratedPerms).Msg("Full migration completed")

	return finalResult, nil
}

// ValidateMigration checks if the migration was successful
func (m *MigrationService) ValidateMigration(ctx context.Context) (*MigrationResult, error) {
	result := &MigrationResult{
		Errors: []string{},
	}

	m.logger.Info().Msg("Validating IAM migration")

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if permissions table has data
	var permCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM permissions").Scan(&permCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count permissions: %w", err)
	}

	// Check if role_permissions table has data
	var rolePermCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM role_permissions").Scan(&rolePermCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count role permissions: %w", err)
	}

	// Check if user_permissions table has data
	var userPermCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM user_permissions").Scan(&userPermCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count user permissions: %w", err)
	}

	// Check if materialized view exists and has data
	var cacheCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM user_effective_permissions_cache").Scan(&cacheCount)
	if err != nil {
		result.Errors = append(result.Errors, "Materialized view may not exist or be empty")
	}

	result.Success = permCount > 0 && rolePermCount > 0
	result.Message = fmt.Sprintf("Migration validation: %d permissions, %d role permissions, %d user permissions, %d cache entries",
		permCount, rolePermCount, userPermCount, cacheCount)
	result.MigratedPerms = permCount + rolePermCount + userPermCount

	m.logger.Info().Int("permissions", permCount).Int("role_permissions", rolePermCount).Int("user_permissions", userPermCount).Int("cache", cacheCount).Msg("Migration validation completed")

	return result, nil
}