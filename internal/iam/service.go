package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type menuCache struct {
	menus     []Menu
	expiresAt time.Time
}

// Service implements the enterprise IAM permission engine.
type Service struct {
	repo   *Repository
	pool   *pgxpool.Pool
	logger zerolog.Logger
	cache  menuCache
	cacheMu sync.RWMutex
}

// NewService creates a new IAM service.
func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		repo:   NewRepository(pool),
		pool:   pool,
		logger: logger.With().Str("module", "iam").Logger(),
	}
}

// CanRequest holds all the context needed for a permission check.
type CanRequest struct {
	UserID       int64
	Action       string
	IsSuperAdmin bool
	IPAddress    string
	UserAgent    string
	RequestID    string
	DeviceType   string
}

// Can evaluates whether the user is allowed to perform the given action.
func (s *Service) Can(ctx context.Context, userID int64, action string) (bool, error) {
	return s.CanWithContext(ctx, CanRequest{
		UserID: userID,
		Action: action,
	})
}

// Evaluate adapts a middleware.PermissionEval-style check (used by RBAC guards).
func (s *Service) Evaluate(ctx context.Context, userID int64, isSuperAdmin bool, action, ip, ua, requestID, device string) (bool, error) {
	return s.CanWithContext(ctx, CanRequest{
		UserID:       userID,
		Action:       action,
		IsSuperAdmin: isSuperAdmin,
		IPAddress:    ip,
		UserAgent:    ua,
		RequestID:    requestID,
		DeviceType:   device,
	})
}

// CanWithContext evaluates permissions with full ABAC context.
func (s *Service) CanWithContext(ctx context.Context, req CanRequest) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx for permission check: %w", err)
	}
	defer tx.Rollback(ctx)

	evalCtx := &EvalContext{
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		RequestID:  req.RequestID,
		Now:        time.Now(),
		DeviceType: req.DeviceType,
	}

	allowed, err := s.evaluate(ctx, tx, req, evalCtx)
	if err != nil {
		return false, err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		s.logger.Error().Err(commitErr).Msg("failed to commit audit log")
	}

	return allowed, nil
}

// evaluate runs the 5-step permission evaluation chain.
func (s *Service) evaluate(ctx context.Context, tx pgx.Tx, req CanRequest, evalCtx *EvalContext) (bool, error) {
	var isActive, isSA bool
	_ = tx.QueryRow(ctx, `SELECT COALESCE(is_active, FALSE), COALESCE(is_super_admin, FALSE) FROM users WHERE id = $1`, req.UserID).
		Scan(&isActive, &isSA)
	if !isActive {
		s.audit(ctx, tx, req, "DENIED", "account_inactive")
		return false, nil
	}
	if req.IsSuperAdmin || isSA {
		if !isViewAction(req.Action) {
			s.audit(ctx, tx, req, "OVERRIDE_GRANT", "super_admin bypass")
		}
		return true, nil
	}

	// Step 2-3: user permission overrides
	override, err := s.repo.FindActiveOverride(ctx, tx, req.UserID, req.Action)
	if err != nil {
		return false, fmt.Errorf("check override: %w", err)
	}

	if override != nil {
		if override.Effect == "REVOKE" {
			s.audit(ctx, tx, req, "OVERRIDE_REVOKE", fmt.Sprintf("override REVOKE: %s", override.Reason))
			return false, nil
		}

		policies, err := s.repo.FindABACPolicies(ctx, tx, req.UserID)
		if err != nil {
			return false, fmt.Errorf("fetch ABAC policies for override: %w", err)
		}

		abacPassed := evaluateABAC(policies, evalCtx)
		decision := decisionString(abacPassed)
		reason := fmt.Sprintf("override GRANT: %s", override.Reason)
		if !abacPassed {
			reason = fmt.Sprintf("override GRANT blocked by ABAC: %s", override.Reason)
		}

		s.audit(ctx, tx, req, decision, reason)
		return abacPassed, nil
	}

	// Step 4: role-based grant
	hasGrant, err := s.repo.HasRoleGrant(ctx, tx, req.UserID, req.Action)
	if err != nil {
		return false, fmt.Errorf("check role grant: %w", err)
	}

	if hasGrant {
		policies, err := s.repo.FindABACPolicies(ctx, tx, req.UserID)
		if err != nil {
			return false, fmt.Errorf("fetch ABAC policies for role: %w", err)
		}

		abacPassed := evaluateABAC(policies, evalCtx)
		decision := decisionString(abacPassed)
		reason := "role_grant"
		if !abacPassed {
			reason = "role_grant blocked by ABAC"
		}

		if !abacPassed || !isViewAction(req.Action) {
			s.audit(ctx, tx, req, decision, reason)
		}
		return abacPassed, nil
	}

	// Step 5: default deny
	s.audit(ctx, tx, req, "DENIED", "no_matching_rule")
	return false, nil
}

func isViewAction(action string) bool {
	return strings.HasSuffix(strings.ToUpper(action), ".VIEW")
}

// audit writes a decision record to the permission audit log.
func (s *Service) audit(ctx context.Context, tx pgx.Tx, req CanRequest, decision, reason string) {
	entry := AuditEntry{
		UserID:     req.UserID,
		ActionName: req.Action,
		Decision:   decision,
		Reason:     reason,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		RequestID:  req.RequestID,
	}

	if err := s.repo.InsertAuditLog(ctx, tx, entry); err != nil {
		s.logger.Error().Err(err).Msg("failed to write audit log")
	}
}

// ─── Role Governance Operations ─────────────────

func (s *Service) ListRoles(ctx context.Context, tx pgx.Tx, _ ...int64) ([]Role, error) {
	return s.repo.ListRoles(ctx, tx)
}

func (s *Service) ListRolesDirect(ctx context.Context, _ ...int64) ([]Role, error) {
	return s.repo.ListRolesDirect(ctx)
}

func (s *Service) GetRole(ctx context.Context, tx pgx.Tx, roleID int) (*Role, error) {
	return s.repo.GetRole(ctx, tx, roleID)
}

func (s *Service) CreateRole(ctx context.Context, tx pgx.Tx, _ int64, name, description string) (*Role, error) {
	return s.repo.CreateRole(ctx, tx, name, description)
}

func (s *Service) UpdateRole(ctx context.Context, tx pgx.Tx, roleID int, name, description string) (*Role, error) {
	return s.repo.UpdateRole(ctx, tx, roleID, name, description)
}

func (s *Service) DeleteRole(ctx context.Context, tx pgx.Tx, roleID int) error {
	return s.repo.DeleteRole(ctx, tx, roleID)
}

func (s *Service) CloneRole(ctx context.Context, tx pgx.Tx, sourceRoleID int, newName, newDescription string) (*Role, error) {
	return s.repo.CloneRole(ctx, tx, sourceRoleID, newName, newDescription)
}

func (s *Service) GetRolePermissionMatrix(ctx context.Context, tx pgx.Tx, roleID int, _ ...int64) ([]MenuMatrixItem, error) {
	return s.repo.GetRolePermissionMatrix(ctx, tx, roleID)
}

func (s *Service) GetRolePermissionMatrixDirect(ctx context.Context, roleID int, _ ...int64) ([]MenuMatrixItem, error) {
	return s.repo.GetRolePermissionMatrixDirect(ctx, roleID)
}

func (s *Service) AssignRolePermissions(ctx context.Context, tx pgx.Tx, roleID int, _ int64, menuActionIDs []int) error {
	err := s.repo.AssignRolePermissions(ctx, tx, roleID, menuActionIDs)
	if err == nil {
		s.InvalidateMenuCache()
	}
	return err
}

func (s *Service) AssignUserRole(ctx context.Context, tx pgx.Tx, userID int64, _ int64, roleID int, assignedBy int64) error {
	err := s.repo.ReplaceUserRole(ctx, tx, userID, roleID, assignedBy)
	if err == nil {
		s.InvalidateMenuCache()
	}
	return err
}

// ─── Category / Bureau Scopes ───────────────────

func (s *Service) GetUserCategoryScopes(ctx context.Context, tx pgx.Tx, userID int64) ([]CategoryScope, error) {
	return s.repo.GetUserCategoryScopes(ctx, tx, userID)
}

func (s *Service) AssignUserCategoryScopes(ctx context.Context, tx pgx.Tx, userID int64, categoryIDs []int, assignedBy int64) error {
	return s.repo.AssignUserCategoryScopes(ctx, tx, userID, categoryIDs, assignedBy)
}

func (s *Service) GetUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID, _ int64) ([]int, error) {
	return s.repo.GetUserDistrictScopes(ctx, tx, userID)
}

func (s *Service) AssignUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID, _ int64, districtIDs []int) error {
	return s.repo.AssignUserDistrictScopes(ctx, tx, userID, districtIDs)
}

// ─── Effective Permissions ─────────────────────

func (s *Service) GetUserEffectivePermissions(ctx context.Context, tx pgx.Tx, userID int64) (*EffectivePermissions, error) {
	return s.repo.GetUserEffectivePermissions(ctx, tx, userID)
}

// ─── Menu & Action Exploration ──────────────────

func (s *Service) ListMenus(ctx context.Context, tx pgx.Tx) ([]Menu, error) {
	return s.repo.ListMenus(ctx, tx)
}

func (s *Service) ListMenusDirect(ctx context.Context) ([]Menu, error) {
	s.cacheMu.RLock()
	if time.Now().Before(s.cache.expiresAt) && s.cache.menus != nil {
		menus := s.cache.menus
		s.cacheMu.RUnlock()
		return menus, nil
	}
	s.cacheMu.RUnlock()

	menus, err := s.repo.ListMenusDirect(ctx)
	if err != nil {
		return nil, err
	}
	s.cacheMu.Lock()
	s.cache = menuCache{menus: menus, expiresAt: time.Now().Add(15 * time.Second)}
	s.cacheMu.Unlock()
	return menus, nil
}

func (s *Service) InvalidateMenuCache() {
	s.cacheMu.Lock()
	s.cache = menuCache{}
	s.cacheMu.Unlock()
}

func (s *Service) ListMenusForUser(ctx context.Context, tx pgx.Tx, userID int64, isSuperAdmin bool) ([]Menu, error) {
	return s.repo.ListMenusForUser(ctx, tx, userID, isSuperAdmin)
}

func (s *Service) ListMenuPrefixes(ctx context.Context) ([]Menu, error) {
	return s.repo.ListMenusDirect(ctx)
}

func (s *Service) ListMenuActions(ctx context.Context, tx pgx.Tx, menuID int) ([]MenuAction, error) {
	return s.repo.ListMenuActions(ctx, tx, menuID)
}

// ─── Overrides & ABAC ───────────────────────────

func (s *Service) CreateOverride(ctx context.Context, tx pgx.Tx, userID, _ int64, menuActionID int, effect, reason string, validFrom, validUntil *time.Time, grantedBy int64) error {
	return s.repo.CreateOverride(ctx, tx, userID, menuActionID, effect, reason, validFrom, validUntil, grantedBy)
}

func (s *Service) CreateABACPolicy(ctx context.Context, tx pgx.Tx, userID, _ int64, attribute string, value json.RawMessage) error {
	return s.repo.CreateABACPolicy(ctx, tx, userID, attribute, value)
}

func (s *Service) ListAuditLogs(ctx context.Context, tx pgx.Tx, limit, offset int) ([]AuditEntry, int64, error) {
	return s.repo.ListAuditLogs(ctx, tx, limit, offset)
}

// ─── Staff Governance ───────────────────────────

func (s *Service) ListStaffWithRoles(ctx context.Context, tx pgx.Tx, search string) ([]StaffUserRoleSummary, error) {
	return s.repo.ListStaffWithRoles(ctx, tx, search)
}

func (s *Service) ListStaffWithRolesDirect(ctx context.Context, search string) ([]StaffUserRoleSummary, error) {
	return s.repo.ListStaffWithRolesDirect(ctx, search)
}

// ApplyRoleTemplate applies a pre-configured newsroom permission preset to a role.
func (s *Service) ApplyRoleTemplate(ctx context.Context, tx pgx.Tx, roleID int, templateName string) error {
	var actionIDs []int
	var query string

	switch templateName {
	case "editor":
		// Full editorial access
		query = `
			SELECT ma.id FROM menu_actions ma
			JOIN menus m ON m.id = ma.menu_id
			WHERE m.name IN ('dashboard', 'articles', 'categories', 'tags', 'media_library', 'live_blogs', 'web_stories', 'epaper', 'comments', 'analytics')
		`
	case "reporter":
		// Draft, edit, view
		query = `
			SELECT ma.id FROM menu_actions ma
			JOIN menus m ON m.id = ma.menu_id
			WHERE (m.name = 'articles' AND ma.action IN ('VIEW', 'ADD', 'EDIT', 'view', 'create', 'edit'))
			   OR (m.name = 'media_library' AND ma.action IN ('VIEW', 'ADD', 'view', 'create'))
			   OR (m.name IN ('categories', 'tags', 'dashboard') AND ma.action IN ('VIEW', 'view'))
		`
	case "fact_checker":
		// Review & verify
		query = `
			SELECT ma.id FROM menu_actions ma
			JOIN menus m ON m.id = ma.menu_id
			WHERE (m.name = 'articles' AND ma.action IN ('VIEW', 'EDIT', 'APPROVE', 'REJECT', 'view', 'edit'))
			   OR (m.name IN ('categories', 'tags', 'dashboard') AND ma.action IN ('VIEW', 'view'))
		`
	case "moderator":
		// Comment moderation
		query = `
			SELECT ma.id FROM menu_actions ma
			JOIN menus m ON m.id = ma.menu_id
			WHERE m.name = 'comments'
			   OR (m.name = 'dashboard' AND ma.action IN ('VIEW', 'view'))
		`
	default:
		return fmt.Errorf("unknown template preset: %s", templateName)
	}

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			actionIDs = append(actionIDs, id)
		}
	}

	return s.AssignRolePermissions(ctx, tx, roleID, 1, actionIDs)
}
