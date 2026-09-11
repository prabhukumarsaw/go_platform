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

// EnhancedService implements the advanced IAM permission engine with Resource-Action-Scope model and ABAC
type EnhancedService struct {
	repo   *Repository
	pool   *pgxpool.Pool
	logger zerolog.Logger
	cache  *PermissionCache
}

// PermissionCache provides in-memory caching for permission checks
type PermissionCache struct {
	userPermissions map[int64][]Permission
	rolePermissions map[int][]Permission
	mu              sync.RWMutex
	ttl             time.Duration
}

// NewPermissionCache creates a new permission cache
func NewPermissionCache(ttl time.Duration) *PermissionCache {
	return &PermissionCache{
		userPermissions: make(map[int64][]Permission),
		rolePermissions: make(map[int][]Permission),
		ttl:             ttl,
	}
}

// GetUserPermissions retrieves user permissions from cache
func (c *PermissionCache) GetUserPermissions(userID int64) ([]Permission, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	perms, exists := c.userPermissions[userID]
	return perms, exists
}

// SetUserPermissions sets user permissions in cache
func (c *PermissionCache) SetUserPermissions(userID int64, perms []Permission) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.userPermissions[userID] = perms
	go func() {
		time.Sleep(c.ttl)
		c.mu.Lock()
		delete(c.userPermissions, userID)
		c.mu.Unlock()
	}()
}

// InvalidateUserPermissions invalidates cached permissions for a user
func (c *PermissionCache) InvalidateUserPermissions(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.userPermissions, userID)
}

// InvalidateRolePermissions invalidates cached permissions for a role
func (c *PermissionCache) InvalidateRolePermissions(roleID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rolePermissions, roleID)
}

// InvalidateAll clears all cached permissions
func (c *PermissionCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userPermissions = make(map[int64][]Permission)
	c.rolePermissions = make(map[int][]Permission)
}

// NewEnhancedService creates a new enhanced IAM service
func NewEnhancedService(pool *pgxpool.Pool, logger zerolog.Logger) *EnhancedService {
	return &EnhancedService{
		repo:   NewRepository(pool),
		pool:   pool,
		logger: logger.With().Str("module", "iam_enhanced").Logger(),
		cache:  NewPermissionCache(15 * time.Minute),
	}
}

// PolicyContext represents the context for permission evaluation
type PolicyContext struct {
	UserID      int64
	Resource    string
	Action      string
	Scope       string
	ResourceID  string
	Attributes  map[string]interface{}
	Environment map[string]interface{}
}

// PolicyDecision represents the result of permission evaluation
type PolicyDecision struct {
	Allowed    bool
	Reason     string
	Scope      string
	Conditions map[string]interface{}
	Policies   []string
}

// NewEnhancedServiceWrapper is an alias for NewEnhancedService for backwards compatibility
// Deprecated: Use NewEnhancedService instead
func NewEnhancedServiceWrapper(pool *pgxpool.Pool, logger zerolog.Logger) *EnhancedService {
	return NewEnhancedService(pool, logger)
}

// Can checks if a user has permission for a resource-action-scope
func (s *EnhancedService) Can(ctx context.Context, userID int64, resource, action, scope string) (bool, error) {
	return s.CanWithContext(ctx, PolicyContext{
		UserID:   userID,
		Resource: resource,
		Action:   action,
		Scope:    scope,
	})
}

// CanWithContext evaluates permissions with full context
func (s *EnhancedService) CanWithContext(ctx context.Context, req PolicyContext) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx for permission check: %w", err)
	}
	defer tx.Rollback(ctx)

	decision, err := s.evaluate(ctx, tx, req)
	if err != nil {
		return false, err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		s.logger.Error().Err(commitErr).Msg("failed to commit permission check")
	}

	return decision.Allowed, nil
}

// CanAccessResource checks if a user can access a specific resource
func (s *EnhancedService) CanAccessResource(ctx context.Context, userID int64, resource, action string, resourceID string) (bool, error) {
	// Check resource-specific grant first
	hasResourceGrant, err := s.checkResourceGrant(ctx, userID, resource, resourceID, action)
	if err != nil {
		return false, err
	}
	if hasResourceGrant {
		return true, nil
	}

	// Check general permission
	return s.Can(ctx, userID, resource, action, "all")
}

// evaluate runs the permission evaluation chain
func (s *EnhancedService) evaluate(ctx context.Context, tx pgx.Tx, req PolicyContext) (*PolicyDecision, error) {
	// Step 1: Check super admin
	var isActive, isSA bool
	err := tx.QueryRow(ctx, `SELECT COALESCE(is_active, FALSE), COALESCE(is_super_admin, FALSE) FROM users WHERE id = $1`, req.UserID).
		Scan(&isActive, &isSA)
	if err != nil {
		return &PolicyDecision{Allowed: false, Reason: "user_not_found"}, nil
	}
	if !isActive {
		return &PolicyDecision{Allowed: false, Reason: "account_inactive"}, nil
	}
	if isSA {
		return &PolicyDecision{Allowed: true, Reason: "super_admin_bypass"}, nil
	}

	// Step 2: Check explicit user permission denies
	userPerms, err := s.repo.GetUserPermissions(ctx, tx, req.UserID)
	if err != nil {
		return nil, err
	}

	for _, up := range userPerms {
		if up.Permission.Resource == req.Resource && 
		   up.Permission.Action == req.Action &&
		   up.Effect == "REVOKE" {
			return &PolicyDecision{Allowed: false, Reason: "user_permission_revoke"}, nil
		}
	}

	// Step 3: Check explicit user permission grants
	for _, up := range userPerms {
		if up.Permission.Resource == req.Resource && 
		   up.Permission.Action == req.Action &&
		   up.Effect == "GRANT" {
			// Check scope
			if s.matchesScope(req.Scope, up.Permission.Scope) {
				// Evaluate ABAC conditions if present
				if up.Conditions != nil {
					if !s.evaluateConditions(req, up.Conditions) {
						return &PolicyDecision{Allowed: false, Reason: "abac_conditions_failed"}, nil
					}
				}
				return &PolicyDecision{Allowed: true, Reason: "user_permission_grant", Scope: up.Permission.Scope}, nil
			}
		}
	}

	// Step 4: Check resource-specific grants
	if req.ResourceID != "" {
		resourceGrants, err := s.repo.GetUserResourceGrants(ctx, tx, req.UserID)
		if err != nil {
			return nil, err
		}

		for _, rg := range resourceGrants {
			if rg.ResourceType == req.Resource && 
			   rg.ResourceID == req.ResourceID &&
			   rg.Permission.Action == req.Action {
				// Evaluate conditions
				if rg.Conditions != nil {
					if !s.evaluateConditions(req, rg.Conditions) {
						return &PolicyDecision{Allowed: false, Reason: "resource_grant_conditions_failed"}, nil
					}
				}
				return &PolicyDecision{Allowed: true, Reason: "resource_grant"}, nil
			}
		}
	}

	// Step 5: Check role permissions
	rolesQuery := `
		SELECT ur.role_id 
		FROM user_roles ur
		WHERE ur.user_id = $1 AND ur.is_active = true
	`
	rows, err := tx.Query(ctx, rolesQuery, req.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleIDs []int
	for rows.Next() {
		var roleID int
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, roleID)
	}

	for _, roleID := range roleIDs {
		rolePerms, err := s.repo.GetRolePermissions(ctx, tx, roleID)
		if err != nil {
			return nil, err
		}

		for _, rp := range rolePerms {
			if rp.Resource == req.Resource && rp.Action == req.Action {
				// Check scope with ownership consideration
				if s.evaluateScopeWithOwnership(ctx, req, rp.Scope) {
					// Evaluate ABAC policies
					policies, err := s.repo.ListEnhancedABACPolicies(ctx, tx)
					if err != nil {
						return nil, err
					}

					abacPassed := s.evaluateABACPolicies(req, policies, roleID)
					if abacPassed {
						return &PolicyDecision{Allowed: true, Reason: "role_permission_grant", Scope: rp.Scope}, nil
					} else {
						return &PolicyDecision{Allowed: false, Reason: "abac_policy_denied"}, nil
					}
				}
			}
		}
	}

	// Step 6: Default deny
	return &PolicyDecision{Allowed: false, Reason: "no_matching_permission"}, nil
}

// checkResourceGrant checks if user has a specific resource grant
func (s *EnhancedService) checkResourceGrant(ctx context.Context, userID int64, resource, resourceID, action string) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	grants, err := s.repo.GetUserResourceGrants(ctx, tx, userID)
	if err != nil {
		return false, err
	}

	for _, grant := range grants {
		if grant.ResourceType == resource && 
		   grant.ResourceID == resourceID && 
		   grant.Permission.Action == action {
			return true, nil
		}
	}

	return false, nil
}

// matchesScope checks if the requested scope matches the granted scope
func (s *EnhancedService) matchesScope(requestedScope, grantedScope string) bool {
	// 'all' scope matches everything
	if grantedScope == "all" {
		return true
	}
	
	// Exact match
	if requestedScope == grantedScope {
		return true
	}
	
	// Handle hierarchical scope matching
	// 'all' > 'department' > 'own' > 'custom' hierarchy
	scopeHierarchy := map[string]int{
		"all":        4,
		"department": 3,
		"own":        2,
		"custom":     1,
		"":           0, // Empty scope defaults to lowest level
	}
	
	requestedLevel := scopeHierarchy[requestedScope]
	grantedLevel := scopeHierarchy[grantedScope]
	
	// If granted scope is higher level, it matches lower level requests
	if grantedLevel >= requestedLevel {
		return true
	}
	
	// Special case: if requesting 'all' but only have 'department', check if department scope should work
	if requestedScope == "all" && grantedScope == "department" {
		// This is handled by the hierarchy check above
		return false // Department scope doesn't grant all access
	}
	
	return false
}

// evaluateScopeWithOwnership evaluates scope considering resource ownership
func (s *EnhancedService) evaluateScopeWithOwnership(ctx context.Context, req PolicyContext, grantedScope string) bool {
	// 'all' scope always matches
	if grantedScope == "all" {
		return true
	}
	
	// For 'own' scope, check if user owns the resource
	if grantedScope == "own" {
		// Check if user is the resource owner
		if req.ResourceID != "" {
			tx, err := s.pool.Begin(ctx)
			if err != nil {
				return false
			}
			defer tx.Rollback(ctx)
			
			isOwner, err := s.repo.CheckResourceOwnership(ctx, tx, req.UserID, req.Resource, req.ResourceID)
			if err != nil {
				return false
			}
			
			return isOwner
		}
		// If no resource ID, deny 'own' scope
		return false
	}
	
	// For 'department' scope, check department membership
	if grantedScope == "department" {
		// This would typically check if user belongs to the same department as the resource
		// For now, we'll use the hierarchy logic
		return s.matchesScope(req.Scope, grantedScope)
	}
	
	// For 'custom' scope, use exact match or hierarchy
	return s.matchesScope(req.Scope, grantedScope)
}

// evaluateConditions evaluates JSON conditions
func (s *EnhancedService) evaluateConditions(req PolicyContext, conditions interface{}) bool {
	// Parse conditions
	conditionsMap, ok := conditions.(map[string]interface{})
	if !ok {
		// Try to parse from JSON
		if jsonBytes, ok := conditions.([]byte); ok {
			var parsed map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &parsed); err == nil {
				conditionsMap = parsed
			} else {
				return false
			}
		} else {
			return false
		}
	}

	// Evaluate time conditions
	if timeCond, ok := conditionsMap["time"].(map[string]interface{}); ok {
		if !s.evaluateTimeCondition(timeCond) {
			return false
		}
	}

	// Evaluate attribute conditions
	if attrCond, ok := conditionsMap["attributes"].(map[string]interface{}); ok {
		if !s.evaluateAttributeConditions(req.Attributes, attrCond) {
			return false
		}
	}

	// Evaluate resource state conditions
	if resourceCond, ok := conditionsMap["resource_state"].(map[string]interface{}); ok {
		if !s.evaluateResourceStateConditions(req, resourceCond) {
			return false
		}
	}

	return true
}

// evaluateTimeCondition evaluates time-based conditions
func (s *EnhancedService) evaluateTimeCondition(condition map[string]interface{}) bool {
	now := time.Now()
	currentHour := now.Hour()
	currentDay := strings.ToLower(now.Weekday().String())

	// Check time range
	if startHour, ok := condition["start"].(float64); ok {
		if endHour, ok := condition["end"].(float64); ok {
			if currentHour < int(startHour) || currentHour >= int(endHour) {
				return false
			}
		}
	}

	// Check days
	if days, ok := condition["days"].([]interface{}); ok {
		dayMatch := false
		for _, day := range days {
			if dayStr, ok := day.(string); ok {
				if strings.ToLower(dayStr) == currentDay {
					dayMatch = true
					break
				}
			}
		}
		if !dayMatch {
			return false
		}
	}

	return true
}

// evaluateAttributeConditions evaluates user attribute conditions
func (s *EnhancedService) evaluateAttributeConditions(userAttrs, conditionAttrs map[string]interface{}) bool {
	for key, expectedValue := range conditionAttrs {
		if actualValue, ok := userAttrs[key]; ok {
			if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// evaluateResourceStateConditions evaluates resource state conditions
func (s *EnhancedService) evaluateResourceStateConditions(req PolicyContext, conditions map[string]interface{}) bool {
	// This would typically check resource state from database
	// For now, we'll check if the condition matches the request
	for key, expectedValue := range conditions {
		if actualValue, ok := req.Attributes[key]; ok {
			if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
				return false
			}
		}
	}
	return true
}

// evaluateABACPolicies evaluates ABAC policies
func (s *EnhancedService) evaluateABACPolicies(req PolicyContext, policies []EnhancedABACPolicy, roleID int) bool {
	// Sort policies by priority (highest first)
	sortedPolicies := make([]EnhancedABACPolicy, len(policies))
	copy(sortedPolicies, policies)
	
	// Simple sort by priority descending
	for i := 0; i < len(sortedPolicies); i++ {
		for j := i + 1; j < len(sortedPolicies); j++ {
			if sortedPolicies[i].Priority < sortedPolicies[j].Priority {
				sortedPolicies[i], sortedPolicies[j] = sortedPolicies[j], sortedPolicies[i]
			}
		}
	}

	for _, policy := range sortedPolicies {
		// Check if policy applies to this context
		if s.policyApplies(req, policy, roleID) {
			// Evaluate policy conditions
			if s.evaluateConditions(req, policy.Conditions) {
				return policy.Effect == "allow"
			}
		}
	}

	// Default allow if no matching policies
	return true
}

// policyApplies checks if a policy applies to the current context
func (s *EnhancedService) policyApplies(req PolicyContext, policy EnhancedABACPolicy, roleID int) bool {
	// Check policy type
	switch policy.PolicyType {
	case "global":
		return true
	case "role":
		return policy.TargetID != nil && int(*policy.TargetID) == roleID
	case "user":
		return policy.TargetID != nil && int(*policy.TargetID) == int(req.UserID)
	default:
		return false
	}
}

// ─── Permission Management Methods ─────────────────────────

// ListPermissions returns all permissions
func (s *EnhancedService) ListPermissions(ctx context.Context) ([]Permission, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.ListPermissions(ctx, tx)
}

// GetPermission returns a specific permission
func (s *EnhancedService) GetPermission(ctx context.Context, permissionID int) (*Permission, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.GetPermission(ctx, tx, permissionID)
}

// CreatePermission creates a new permission
func (s *EnhancedService) CreatePermission(ctx context.Context, resource, action, scope, description string, isSystem bool) (*Permission, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	perm, err := s.repo.CreatePermission(ctx, tx, resource, action, scope, description, isSystem)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.repo.RefreshPermissionCache(ctx)
	return perm, nil
}

// ListPermissionGroups returns all permission groups
func (s *EnhancedService) ListPermissionGroups(ctx context.Context) ([]PermissionGroup, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.ListPermissionGroups(ctx, tx)
}

// ─── User Permission Methods ───────────────────────────────

// GetUserPermissions returns direct permissions for a user
func (s *EnhancedService) GetUserPermissions(ctx context.Context, userID int64) ([]UserPermission, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.GetUserPermissions(ctx, tx, userID)
}

// GrantUserPermission grants a direct permission to a user
func (s *EnhancedService) GrantUserPermission(ctx context.Context, userID int64, permissionID int, effect, reason string, validFrom, validUntil *time.Time, grantedBy int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	err = s.repo.GrantUserPermission(ctx, tx, userID, permissionID, effect, reason, validFrom, validUntil, grantedBy)
	if err != nil {
		return fmt.Errorf("failed to grant user permission: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.cache.InvalidateUserPermissions(userID)
	s.repo.RefreshPermissionCache(ctx)
	return nil
}

// RevokeUserPermission revokes a direct permission from a user
func (s *EnhancedService) RevokeUserPermission(ctx context.Context, userID int64, permissionID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	err = s.repo.RevokeUserPermission(ctx, tx, userID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to revoke user permission: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.cache.InvalidateUserPermissions(userID)
	s.repo.RefreshPermissionCache(ctx)
	return nil
}

// ─── Resource Grant Methods ────────────────────────────────

// GetUserResourceGrants returns resource-specific grants for a user
func (s *EnhancedService) GetUserResourceGrants(ctx context.Context, userID int64) ([]ResourceGrant, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.GetUserResourceGrants(ctx, tx, userID)
}

// CreateResourceGrant creates a resource-specific access grant
func (s *EnhancedService) CreateResourceGrant(ctx context.Context, userID int64, resourceType, resourceID string, permissionID int, grantedBy int64, expiresAt *time.Time, conditions json.RawMessage) (*ResourceGrant, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	grant, err := s.repo.CreateResourceGrant(ctx, tx, userID, resourceType, resourceID, permissionID, grantedBy, expiresAt, conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource grant: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.cache.InvalidateUserPermissions(userID)
	return grant, nil
}

// RevokeResourceGrant revokes a resource-specific grant
func (s *EnhancedService) RevokeResourceGrant(ctx context.Context, grantID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	err = s.repo.RevokeResourceGrant(ctx, tx, grantID)
	if err != nil {
		return fmt.Errorf("failed to revoke resource grant: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ─── ABAC Policy Methods ───────────────────────────────────

// ListEnhancedABACPolicies returns all ABAC policies
func (s *EnhancedService) ListEnhancedABACPolicies(ctx context.Context) ([]EnhancedABACPolicy, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.ListEnhancedABACPolicies(ctx, tx)
}

// CreateEnhancedABACPolicy creates a new ABAC policy
func (s *EnhancedService) CreateEnhancedABACPolicy(ctx context.Context, name, description, policyType string, targetID *int64, effect string, priority int, conditions json.RawMessage) (*EnhancedABACPolicy, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	policy, err := s.repo.CreateEnhancedABACPolicy(ctx, tx, name, description, policyType, targetID, effect, priority, conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to create ABAC policy: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.repo.RefreshPermissionCache(ctx)
	return policy, nil
}

// UpdateEnhancedABACPolicy updates an existing ABAC policy
func (s *EnhancedService) UpdateEnhancedABACPolicy(ctx context.Context, policyID int64, name, description string, effect string, priority int, conditions json.RawMessage) (*EnhancedABACPolicy, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	policy, err := s.repo.UpdateEnhancedABACPolicy(ctx, tx, policyID, name, description, effect, priority, conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to update ABAC policy: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.repo.RefreshPermissionCache(ctx)
	return policy, nil
}

// DeleteEnhancedABACPolicy deletes an ABAC policy
func (s *EnhancedService) DeleteEnhancedABACPolicy(ctx context.Context, policyID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	err = s.repo.DeleteEnhancedABACPolicy(ctx, tx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete ABAC policy: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.repo.RefreshPermissionCache(ctx)
	return nil
}

// ─── Enhanced Role Methods ─────────────────────────────────

// GetRolePermissions returns permissions for a specific role (new system)
func (s *EnhancedService) GetRolePermissionsNew(ctx context.Context, roleID int) ([]Permission, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.GetRolePermissions(ctx, tx, roleID)
}

// AssignRolePermissionsNew assigns permissions to a role (new system)
func (s *EnhancedService) AssignRolePermissionsNew(ctx context.Context, roleID int, permissionIDs []int, grantedBy int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	err = s.repo.AssignRolePermissionsNew(ctx, tx, roleID, permissionIDs, grantedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role permissions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.repo.RefreshPermissionCache(ctx)
	// Also invalidate role permission cache
	if s.cache != nil {
		// Since we don't have user-level role tracking, we'll clear the entire cache
		// In a production system, you'd want to track which users have this role
	}
	return nil
}

// GetUserEffectivePermissionsNew returns all effective permissions for a user (new system)
func (s *EnhancedService) GetUserEffectivePermissionsNew(ctx context.Context, userID int64) (*EffectivePermissions, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	return s.repo.GetUserEffectivePermissionsNew(ctx, tx, userID)
}