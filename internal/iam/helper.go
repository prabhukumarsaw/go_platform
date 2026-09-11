package iam

import (
	"context"
	"fmt"
	"strings"
)

// Helper provides convenience methods for permission checking across handlers
type Helper struct {
	enhancedService *EnhancedService
}

// NewHelper creates a new IAM helper
func NewHelper(enhancedService *EnhancedService) *Helper {
	return &Helper{
		enhancedService: enhancedService,
	}
}

// CheckPermission checks if a user has a specific permission
func (h *Helper) CheckPermission(ctx context.Context, userID int64, resource, action, scope string) (bool, error) {
	return h.enhancedService.Can(ctx, userID, resource, action, scope)
}

// CheckResourceAccess checks if a user can access a specific resource
func (h *Helper) CheckResourceAccess(ctx context.Context, userID int64, resource, action, resourceID string) (bool, error) {
	return h.enhancedService.CanAccessResource(ctx, userID, resource, action, resourceID)
}

// CheckOwnResource checks if a user can access their own resource
func (h *Helper) CheckOwnResource(ctx context.Context, userID int64, resource, action string, resourceOwnerID int64) (bool, error) {
	// If user is the owner, check for 'own' scope
	if userID == resourceOwnerID {
		allowed, err := h.enhancedService.Can(ctx, userID, resource, action, "own")
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
		// Fall through to check 'all' scope as well
	}
	
	// Otherwise, check for 'all' scope
	return h.enhancedService.Can(ctx, userID, resource, action, "all")
}

// CheckResourceOwnership checks if a user owns a specific resource
func (h *Helper) CheckResourceOwnership(ctx context.Context, userID int64, resourceType, resourceID string) (bool, error) {
	tx, err := h.enhancedService.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	
	return h.enhancedService.repo.CheckResourceOwnership(ctx, tx, userID, resourceType, resourceID)
}

// CheckPermissionWithOwnership checks permission considering resource ownership
func (h *Helper) CheckPermissionWithOwnership(ctx context.Context, userID int64, resource, action, scope string, resourceType, resourceID string) (bool, error) {
	// Get resource owner
	isOwner, err := h.CheckResourceOwnership(ctx, userID, resourceType, resourceID)
	if err != nil {
		// If ownership check fails, fall back to regular permission check
		return h.enhancedService.Can(ctx, userID, resource, action, scope)
	}
	
	// If requesting "own" scope and user is not owner, deny
	if scope == "own" && !isOwner {
		return false, nil
	}
	
	// If requesting "own" scope and user is owner, check for "own" or "all" permission
	if scope == "own" && isOwner {
		// Check for "own" permission
		allowed, err := h.enhancedService.Can(ctx, userID, resource, action, "own")
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
		
		// Check for "all" permission as fallback
		return h.enhancedService.Can(ctx, userID, resource, action, "all")
	}
	
	// For other scopes, use regular permission check
	return h.enhancedService.Can(ctx, userID, resource, action, scope)
}

// CheckAnyScope checks if user has permission with any scope (own, all, department)
func (h *Helper) CheckAnyScope(ctx context.Context, userID int64, resource, action string) (bool, string, error) {
	// Try 'all' scope first (most permissive)
	allowed, err := h.enhancedService.Can(ctx, userID, resource, action, "all")
	if err != nil {
		return false, "", err
	}
	if allowed {
		return true, "all", nil
	}
	
	// Try 'department' scope
	allowed, err = h.enhancedService.Can(ctx, userID, resource, action, "department")
	if err != nil {
		return false, "", err
	}
	if allowed {
		return true, "department", nil
	}
	
	// Try 'own' scope (most restrictive)
	allowed, err = h.enhancedService.Can(ctx, userID, resource, action, "own")
	if err != nil {
		return false, "", err
	}
	if allowed {
		return true, "own", nil
	}
	
	return false, "", nil
}

// GetPermissionString returns the permission string in resource:action:scope format
func (h *Helper) GetPermissionString(resource, action, scope string) string {
	if scope == "" {
		scope = "all"
	}
	return fmt.Sprintf("%s:%s:%s", resource, action, scope)
}

// ParsePermissionString parses a permission string into components
func (h *Helper) ParsePermissionString(permString string) (resource, action, scope string, err error) {
	parts := []string{}
	for i, part := range strings.Split(permString, ":") {
		if i >= 3 {
			break // Only take first 3 parts
		}
		parts = append(parts, part)
	}
	
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid permission format: %s", permString)
	}
	
	resource = parts[0]
	action = parts[1]
	
	if len(parts) >= 3 {
		scope = parts[2]
	} else {
		scope = "all"
	}
	
	return resource, action, scope, nil
}

// Common permission constants for easy reference
const (
	// News permissions
	PermNewsCreateAll      = "news:create:all"
	PermNewsReadOwn        = "news:read:own"
	PermNewsReadAll        = "news:read:all"
	PermNewsUpdateOwn      = "news:update:own"
	PermNewsUpdateAll      = "news:update:all"
	PermNewsDeleteOwn      = "news:delete:own"
	PermNewsDeleteAll      = "news:delete:all"
	PermNewsPublishAll     = "news:publish:all"
	PermNewsApproveAll     = "news:approve:all"
	
	// Media permissions
	PermMediaUploadAll     = "media:upload:all"
	PermMediaReadOwn       = "media:read:own"
	PermMediaReadAll       = "media:read:all"
	PermMediaDeleteOwn     = "media:delete:own"
	PermMediaDeleteAll     = "media:delete:all"
	
	// User permissions
	PermUserReadAll        = "user:read:all"
	PermUserUpdateOwn      = "user:update:own"
	PermUserUpdateAll      = "user:update:all"
	
	// Category permissions
	PermCategoryManageAll  = "category:manage:all"
	PermCategoryReadAll    = "category:read:all"
	
	// IAM permissions
	PermIAMManageAll       = "iam:manage:all"
	PermIAMReadAll         = "iam:read:all"
	PermIAMAssignRolesAll  = "iam:assign_roles:all"
)