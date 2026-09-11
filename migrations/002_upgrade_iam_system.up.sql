-- 002_upgrade_iam_system.up.sql
-- Upgrade IAM system to Resource-Action-Scope model with advanced ABAC support
-- This migration adds new tables while keeping existing ones for backward compatibility

-- ──────────────────────────────────────────────
-- 1. NEW: Permissions Table (Resource-Action-Scope Model)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    scope VARCHAR(50) DEFAULT 'all',
    description TEXT,
    is_system BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(resource, action, scope)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource);
CREATE INDEX IF NOT EXISTS idx_permissions_action ON permissions(action);
CREATE INDEX IF NOT EXISTS idx_permissions_scope ON permissions(scope);
CREATE INDEX IF NOT EXISTS idx_permissions_resource_action ON permissions(resource, action);

-- ──────────────────────────────────────────────
-- 2. NEW: Role Permissions (replace role_menu_actions)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INT REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    conditions JSONB,
    PRIMARY KEY(role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_expires ON role_permissions(expires_at) WHERE expires_at IS NOT NULL;

-- ──────────────────────────────────────────────
-- 3. NEW: User Direct Permissions (for exceptions)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_permissions (
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    permission_id INT REFERENCES permissions(id) ON DELETE CASCADE,
    effect VARCHAR(10) DEFAULT 'GRANT',
    reason TEXT,
    granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    conditions JSONB,
    PRIMARY KEY(user_id, permission_id, effect)
);

CREATE INDEX IF NOT EXISTS idx_user_permissions_user ON user_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_permission ON user_permissions(permission_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_effect ON user_permissions(effect);
CREATE INDEX IF NOT EXISTS idx_user_permissions_valid ON user_permissions(valid_from, valid_until);

-- ──────────────────────────────────────────────
-- 4. NEW: Resource Grants (specific resource access)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS resource_grants (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    permission_id INT REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    conditions JSONB,
    UNIQUE(user_id, resource_type, resource_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_resource_grants_user ON resource_grants(user_id);
CREATE INDEX IF NOT EXISTS idx_resource_grants_resource ON resource_grants(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_resource_grants_expires ON resource_grants(expires_at) WHERE expires_at IS NOT NULL;

-- ──────────────────────────────────────────────
-- 5. ENHANCED: ABAC Policies with proper structure
-- ──────────────────────────────────────────────
-- Drop existing simple ABAC table if exists
DROP TABLE IF EXISTS abac_policies CASCADE;

CREATE TABLE IF NOT EXISTS abac_policies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    policy_type VARCHAR(20) DEFAULT 'user',
    target_id BIGINT,
    effect VARCHAR(10) DEFAULT 'allow',
    priority INT DEFAULT 0,
    conditions JSONB NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abac_policies_target ON abac_policies(target_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_abac_policies_type ON abac_policies(policy_type, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_abac_policies_priority ON abac_policies(priority DESC);

-- ──────────────────────────────────────────────
-- 6. NEW: Permission Groups (for easy management)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS permission_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permission_group_items (
    group_id INT REFERENCES permission_groups(id) ON DELETE CASCADE,
    permission_id INT REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY(group_id, permission_id)
);

CREATE TABLE IF NOT EXISTS role_permission_groups (
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    group_id INT REFERENCES permission_groups(id) ON DELETE CASCADE,
    granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY(role_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_permission_group_items_group ON permission_group_items(group_id);
CREATE INDEX IF NOT EXISTS idx_permission_group_items_permission ON permission_group_items(permission_id);
CREATE INDEX IF NOT EXISTS idx_role_permission_groups_role ON role_permission_groups(role_id);

-- ──────────────────────────────────────────────
-- 7. ENHANCE: Menus (UI navigation only, decoupled from permissions)
-- ──────────────────────────────────────────────
ALTER TABLE menus ADD COLUMN IF NOT EXISTS required_permission VARCHAR(100);
ALTER TABLE menus ADD COLUMN IF NOT EXISTS required_scope VARCHAR(50) DEFAULT 'all';

-- Update existing menus with permission mappings
UPDATE menus SET required_permission = 'news:read' WHERE name = 'articles';
UPDATE menus SET required_permission = 'media:read' WHERE name = 'media_library';
UPDATE menus SET required_permission = 'category:manage' WHERE name = 'categories';
UPDATE menus SET required_permission = 'user:read' WHERE name = 'users';
UPDATE menus SET required_permission = 'iam:manage' WHERE name = 'roles';
UPDATE menus SET required_permission = 'analytics:read' WHERE name = 'analytics';
UPDATE menus SET required_permission = 'settings:manage' WHERE name = 'settings';

-- ──────────────────────────────────────────────
-- 8. SEED: Basic Permissions
-- ──────────────────────────────────────────────
INSERT INTO permissions (resource, action, scope, description, is_system) VALUES
-- News permissions
('news', 'create', 'all', 'Create news articles', true),
('news', 'read', 'own', 'Read own news articles', true),
('news', 'read', 'all', 'Read all news articles', true),
('news', 'read', 'department', 'Read department news articles', true),
('news', 'update', 'own', 'Update own news articles', true),
('news', 'update', 'all', 'Update all news articles', true),
('news', 'delete', 'own', 'Delete own news articles', true),
('news', 'delete', 'all', 'Delete all news articles', true),
('news', 'publish', 'all', 'Publish news articles', true),
('news', 'approve', 'all', 'Approve news articles', true),
('news', 'reject', 'all', 'Reject news articles', true),
('news', 'submit', 'all', 'Submit news articles for review', true),

-- Media permissions
('media', 'upload', 'all', 'Upload media files', true),
('media', 'read', 'own', 'Read own media files', true),
('media', 'read', 'all', 'Read all media files', true),
('media', 'read', 'department', 'Read department media files', true),
('media', 'update', 'own', 'Update own media files', true),
('media', 'update', 'all', 'Update all media files', true),
('media', 'delete', 'own', 'Delete own media files', true),
('media', 'delete', 'all', 'Delete all media files', true),

-- User permissions
('user', 'read', 'all', 'Read user information', true),
('user', 'read', 'own', 'Read own user information', true),
('user', 'update', 'own', 'Update own profile', true),
('user', 'update', 'all', 'Update any user profile', true),
('user', 'delete', 'all', 'Delete users', true),
('user', 'create', 'all', 'Create users', true),

-- Category permissions
('category', 'manage', 'all', 'Manage categories', true),
('category', 'read', 'all', 'Read categories', true),

-- Tag permissions
('tag', 'manage', 'all', 'Manage tags', true),
('tag', 'read', 'all', 'Read tags', true),

-- IAM permissions
('iam', 'manage', 'all', 'Manage IAM system', true),
('iam', 'read', 'all', 'Read IAM configuration', true),
('iam', 'assign_roles', 'all', 'Assign roles to users', true),

-- Analytics permissions
('analytics', 'read', 'all', 'Read analytics data', true),
('analytics', 'read', 'own', 'Read own analytics data', true),

-- Settings permissions
('settings', 'manage', 'all', 'Manage platform settings', true),
('settings', 'read', 'all', 'Read platform settings', true),

-- Comments permissions
('comments', 'moderate', 'all', 'Moderate comments', true),
('comments', 'read', 'all', 'Read comments', true),

-- Notification permissions
('notifications', 'send', 'all', 'Send notifications', true),
('notifications', 'read', 'all', 'Read notifications', true),

-- Live blog permissions
('liveblog', 'create', 'all', 'Create live blogs', true),
('liveblog', 'update', 'own', 'Update own live blogs', true),
('liveblog', 'update', 'all', 'Update any live blog', true),

-- Web story permissions
('webstory', 'create', 'all', 'Create web stories', true),
('webstory', 'update', 'own', 'Update own web stories', true),
('webstory', 'update', 'all', 'Update any web stories', true),

-- E-paper permissions
('epaper', 'manage', 'all', 'Manage e-paper', true),
('epaper', 'read', 'all', 'Read e-paper', true)

ON CONFLICT (resource, action, scope) DO NOTHING;

-- ──────────────────────────────────────────────
-- 9. SEED: Permission Groups
-- ──────────────────────────────────────────────
INSERT INTO permission_groups (name, description, is_system) VALUES
('basic_editor', 'Basic editorial permissions', true),
('senior_editor', 'Senior editorial permissions', true),
('reporter', 'Reporter permissions', true),
('moderator', 'Content moderator permissions', true),
('admin', 'Administrative permissions', true)
ON CONFLICT (name) DO NOTHING;

-- ──────────────────────────────────────────────
-- 10. SEED: Permission Group Items
-- ──────────────────────────────────────────────
-- Basic Editor Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT 
    pg.id, 
    p.id 
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'basic_editor'
AND p.resource IN ('news', 'media', 'category', 'tag')
AND p.action IN ('create', 'read', 'update')
AND p.scope IN ('own', 'all')
ON CONFLICT DO NOTHING;

-- Senior Editor Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT 
    pg.id, 
    p.id 
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'senior_editor'
AND p.resource IN ('news', 'media', 'category', 'tag')
AND p.action IN ('create', 'read', 'update', 'publish', 'approve')
AND p.scope = 'all'
ON CONFLICT DO NOTHING;

-- Reporter Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT 
    pg.id, 
    p.id 
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'reporter'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update')
AND p.scope = 'own'
ON CONFLICT DO NOTHING;

-- Moderator Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT 
    pg.id, 
    p.id 
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'moderator'
AND p.resource IN ('comments', 'news')
AND p.action IN ('read', 'moderate', 'update')
ON CONFLICT DO NOTHING;

-- Admin Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT 
    pg.id, 
    p.id 
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'admin'
AND p.scope = 'all'
ON CONFLICT DO NOTHING;

-- ──────────────────────────────────────────────
-- 11. SEED: Sample ABAC Policies
-- ──────────────────────────────────────────────
INSERT INTO abac_policies (name, description, policy_type, effect, priority, conditions) VALUES
('business_hours_publishing', 'Only allow publishing during business hours', 'global', 'allow', 10, 
 '{"time": {"start": "09:00", "end": "18:00", "days": ["mon", "tue", "wed", "thu", "fri"]}, "resource": "news", "action": "publish"}'),

('senior_editor_approval', 'Senior editors can approve articles', 'role', 'allow', 5,
 '{"resource": "news", "action": "approve", "user_attributes": {"level": "senior"}}'),

('department_news_access', 'Users can access news from their department', 'user', 'allow', 3,
 '{"resource": "news", "action": "read", "scope": "department"}')
ON CONFLICT DO NOTHING;

-- ──────────────────────────────────────────────
-- 12. CREATE: Permission Cache Materialized View
-- ──────────────────────────────────────────────
CREATE MATERIALIZED VIEW IF NOT EXISTS user_effective_permissions_cache AS
SELECT 
    ur.user_id,
    p.resource,
    p.action,
    p.scope,
    'role' as source_type,
    r.name as source_name,
    rp.expires_at
FROM user_roles ur
JOIN role_permissions rp ON rp.role_id = ur.role_id
JOIN permissions p ON p.id = rp.permission_id
JOIN roles r ON r.id = ur.role_id
WHERE ur.is_active = true 
  AND (rp.expires_at IS NULL OR rp.expires_at > NOW())

UNION ALL

SELECT 
    up.user_id,
    p.resource,
    p.action,
    p.scope,
    'direct' as source_type,
    'user grant' as source_name,
    up.valid_until as expires_at
FROM user_permissions up
JOIN permissions p ON p.id = up.permission_id
WHERE up.effect = 'GRANT' 
  AND (up.valid_until IS NULL OR up.valid_until > NOW())
  AND (up.valid_from IS NULL OR up.valid_from <= NOW());

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_perms_cache ON user_effective_permissions_cache(user_id, resource, action, scope);

-- ──────────────────────────────────────────────
-- 13. CREATE: Refresh function for materialized view
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION refresh_user_permissions_cache()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 14. ADD: Function to check permission
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION check_user_permission(
    p_user_id BIGINT,
    p_resource VARCHAR,
    p_action VARCHAR,
    p_scope VARCHAR DEFAULT 'all'
)
RETURNS BOOLEAN AS $$
DECLARE
    has_perm BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM user_effective_permissions_cache
        WHERE user_id = p_user_id
        AND resource = p_resource
        AND action = p_action
        AND (scope = p_scope OR scope = 'all')
        AND (expires_at IS NULL OR expires_at > NOW())
    ) INTO has_perm;
    
    RETURN has_perm;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 15. ADD: Function to check resource access
-- ──────────────────────────────────────────────
CREATE OR REPLACE FUNCTION check_resource_access(
    p_user_id BIGINT,
    p_resource_type VARCHAR,
    p_resource_id VARCHAR,
    p_action VARCHAR
)
RETURNS BOOLEAN AS $$
DECLARE
    has_access BOOLEAN;
BEGIN
    -- Check resource-specific grant
    SELECT EXISTS (
        SELECT 1 FROM resource_grants rg
        JOIN permissions p ON p.id = rg.permission_id
        WHERE rg.user_id = p_user_id
        AND rg.resource_type = p_resource_type
        AND rg.resource_id = p_resource_id
        AND p.action = p_action
        AND (rg.expires_at IS NULL OR rg.expires_at > NOW())
    ) INTO has_access;
    
    IF has_access THEN
        RETURN TRUE;
    END IF;
    
    -- Check general permission
    SELECT check_user_permission(p_user_id, p_resource_type, p_action, 'all') INTO has_access;
    
    RETURN has_access;
END;
$$ LANGUAGE plpgsql;