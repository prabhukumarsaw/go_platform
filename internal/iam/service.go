package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service implements the IAM permission engine.
type Service struct {
	repo   *Repository
	pool   *pgxpool.Pool
	logger zerolog.Logger
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
	TenantID     int64
	Action       string
	IsSuperAdmin bool
	IPAddress    string
	UserAgent    string
	RequestID    string
	DeviceType   string
}

// Can evaluates whether the user is allowed to perform the given action.
func (s *Service) Can(ctx context.Context, userID, tenantID int64, action string) (bool, error) {
	return s.CanWithContext(ctx, CanRequest{
		UserID:   userID,
		TenantID: tenantID,
		Action:   action,
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
	// Step 1: super_admin bypass
	if req.IsSuperAdmin {
		s.audit(ctx, tx, req, "OVERRIDE_GRANT", "super_admin bypass")
		return true, nil
	}

	// Step 2-3: user permission overrides
	override, err := s.repo.FindActiveOverride(ctx, tx, req.UserID, req.TenantID, req.Action)
	if err != nil {
		return false, fmt.Errorf("check override: %w", err)
	}

	if override != nil {
		if override.Effect == "REVOKE" {
			s.audit(ctx, tx, req, "OVERRIDE_REVOKE", fmt.Sprintf("override REVOKE: %s", override.Reason))
			return false, nil
		}

		policies, err := s.repo.FindABACPolicies(ctx, tx, req.UserID, req.TenantID)
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
	hasGrant, err := s.repo.HasRoleGrant(ctx, tx, req.UserID, req.TenantID, req.Action)
	if err != nil {
		return false, fmt.Errorf("check role grant: %w", err)
	}

	if hasGrant {
		policies, err := s.repo.FindABACPolicies(ctx, tx, req.UserID, req.TenantID)
		if err != nil {
			return false, fmt.Errorf("fetch ABAC policies for role: %w", err)
		}

		abacPassed := evaluateABAC(policies, evalCtx)
		decision := decisionString(abacPassed)
		reason := "role_grant"
		if !abacPassed {
			reason = "role_grant blocked by ABAC"
		}

		s.audit(ctx, tx, req, decision, reason)
		return abacPassed, nil
	}

	// Step 5: default deny
	s.audit(ctx, tx, req, "DENIED", "no_matching_rule")
	return false, nil
}

// audit writes a decision record to the permission audit log.
func (s *Service) audit(ctx context.Context, tx pgx.Tx, req CanRequest, decision, reason string) {
	tenantID := &req.TenantID

	entry := AuditEntry{
		UserID:     req.UserID,
		TenantID:   tenantID,
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

// ─── Admin Operations ───────────────────────────

func (s *Service) ListRoles(ctx context.Context, tx pgx.Tx, tenantID int64) ([]Role, error) {
	return s.repo.ListRoles(ctx, tx, tenantID)
}

func (s *Service) CreateRole(ctx context.Context, tx pgx.Tx, tenantID int64, name, description string) (*Role, error) {
	return s.repo.CreateRole(ctx, tx, tenantID, name, description)
}

func (s *Service) AssignRolePermissions(ctx context.Context, tx pgx.Tx, roleID int, tenantID int64, menuActionIDs []int) error {
	return s.repo.AssignRolePermissions(ctx, tx, roleID, tenantID, menuActionIDs)
}

func (s *Service) AssignUserRole(ctx context.Context, tx pgx.Tx, userID int64, tenantID int64, roleID int, assignedBy int64) error {
	return s.repo.AssignUserRole(ctx, tx, userID, tenantID, roleID, assignedBy)
}

func (s *Service) ListMenus(ctx context.Context, tx pgx.Tx) ([]Menu, error) {
	return s.repo.ListMenus(ctx, tx)
}

func (s *Service) ListMenuActions(ctx context.Context, tx pgx.Tx, menuID int) ([]MenuAction, error) {
	return s.repo.ListMenuActions(ctx, tx, menuID)
}

func (s *Service) CreateOverride(ctx context.Context, tx pgx.Tx, userID, tenantID int64, menuActionID int, effect, reason string, validFrom, validUntil *time.Time, grantedBy int64) error {
	return s.repo.CreateOverride(ctx, tx, userID, tenantID, menuActionID, effect, reason, validFrom, validUntil, grantedBy)
}

func (s *Service) CreateABACPolicy(ctx context.Context, tx pgx.Tx, userID, tenantID int64, attribute string, value json.RawMessage) error {
	return s.repo.CreateABACPolicy(ctx, tx, userID, tenantID, attribute, value)
}

func (s *Service) ListAuditLogs(ctx context.Context, tx pgx.Tx, limit, offset int) ([]AuditEntry, int64, error) {
	return s.repo.ListAuditLogs(ctx, tx, limit, offset)
}

func (s *Service) GetUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID, tenantID int64) ([]int, error) {
	return s.repo.GetUserDistrictScopes(ctx, tx, userID, tenantID)
}

func (s *Service) AssignUserDistrictScopes(ctx context.Context, tx pgx.Tx, userID, tenantID int64, districtIDs []int) error {
	return s.repo.AssignUserDistrictScopes(ctx, tx, userID, tenantID, districtIDs)
}
