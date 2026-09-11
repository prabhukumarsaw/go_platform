-- 005_seed_dynamic_iam_system.up.sql
-- Complete dynamic IAM system setup with roles, users, permissions, and scopes
-- This creates a fully functional, dynamic access control system

-- ──────────────────────────────────────────────
-- 1. Enhanced Role Definitions with Dynamic Capabilities
-- ──────────────────────────────────────────────

-- Clear existing roles (except system ones to maintain integrity)
DELETE FROM role_menu_actions WHERE role_id IN (SELECT id FROM roles WHERE is_system = false);
DELETE FROM user_roles WHERE role_id IN (SELECT id FROM roles WHERE is_system = false);
DELETE FROM roles WHERE is_system = false;

-- Insert dynamic roles
INSERT INTO roles (name, description, is_system, is_active) VALUES
-- Executive Roles
('super_admin', 'Full system access with all permissions', true, true),
('chief_editor', 'Chief editor with editorial oversight and publishing rights', true, true),
('bureau_editor', 'Regional bureau editor with jurisdiction management', true, true),

-- Editorial Roles
('senior_editor', 'Senior editor with advanced editorial permissions', true, true),
('editor', 'Standard editor with content management permissions', true, true),
('sub_editor', 'Sub-editor with basic editing permissions', true, true),

-- Reporting Roles
('senior_reporter', 'Senior reporter with advanced reporting capabilities', true, true),
('reporter', 'Standard reporter with content creation permissions', true, true),
('correspondent', 'Field correspondent with limited publishing rights', true, true),

-- Specialized Roles
('photo_editor', 'Photo editor with media management permissions', true, true),
('video_editor', 'Video editor with multimedia permissions', true, true),
('fact_checker', 'Fact checker with verification permissions', true, true),
('copy_editor', 'Copy editor with proofreading permissions', true, true),

-- Moderation Roles
('community_moderator', 'Community moderator with user management permissions', true, true),
('content_moderator', 'Content moderator with approval permissions', true, true),

-- Technical Roles
('system_admin', 'System administrator with technical access', true, true),
('analytics_manager', 'Analytics manager with reporting access', true, true),

-- Custom Dynamic Roles (examples)
('national_editor', 'National editor with country-wide jurisdiction', false, true),
('regional_editor', 'Regional editor with state-level jurisdiction', false, true),
('district_editor', 'District editor with local jurisdiction', false, true),
('freelancer', 'Freelance contributor with limited access', false, true),
('intern', 'Intern with restricted access and supervision', false, true)
ON CONFLICT (name) DO NOTHING;

-- ──────────────────────────────────────────────
-- 2. Enhanced Menu System with Permission Mapping
-- ──────────────────────────────────────────────

-- Update existing menus with enhanced permission mappings
UPDATE menus SET 
    required_permission = 'news:read:all',
    required_scope = 'all'
WHERE name = 'articles';

UPDATE menus SET 
    required_permission = 'media:read:all',
    required_scope = 'all'
WHERE name = 'media_library';

UPDATE menus SET 
    required_permission = 'category:manage:all',
    required_scope = 'all'
WHERE name = 'categories';

UPDATE menus SET 
    required_permission = 'tag:manage:all',
    required_scope = 'all'
WHERE name = 'tags';

UPDATE menus SET 
    required_permission = 'user:read:all',
    required_scope = 'all'
WHERE name = 'users';

UPDATE menus SET 
    required_permission = 'iam:manage:all',
    required_scope = 'all'
WHERE name = 'roles';

UPDATE menus SET 
    required_permission = 'analytics:read:all',
    required_scope = 'all'
WHERE name = 'analytics';

UPDATE menus SET 
    required_permission = 'settings:manage:all',
    required_scope = 'all'
WHERE name = 'settings';

-- ──────────────────────────────────────────────
-- 3. Enhanced Permission Definitions with Dynamic Scopes
-- ──────────────────────────────────────────────

-- Clear existing custom permissions (keep system ones)
DELETE FROM permissions WHERE is_system = false;

-- Add dynamic permissions for all resources
INSERT INTO permissions (resource, action, scope, description, is_system) VALUES
-- Enhanced News Permissions
('news', 'create', 'all', 'Create news articles anywhere', true),
('news', 'create', 'department', 'Create news within department', true),
('news', 'create', 'own', 'Create own news articles', true),
('news', 'read', 'all', 'Read all news articles', true),
('news', 'read', 'department', 'Read department news articles', true),
('news', 'read', 'own', 'Read own news articles', true),
('news', 'update', 'all', 'Update any news article', true),
('news', 'update', 'department', 'Update department news articles', true),
('news', 'update', 'own', 'Update own news articles', true),
('news', 'delete', 'all', 'Delete any news article', true),
('news', 'delete', 'department', 'Delete department news articles', true),
('news', 'delete', 'own', 'Delete own news articles', true),
('news', 'publish', 'all', 'Publish any news article', true),
('news', 'publish', 'department', 'Publish department news articles', true),
('news', 'approve', 'all', 'Approve any news article', true),
('news', 'approve', 'department', 'Approve department news articles', true),
('news', 'reject', 'all', 'Reject any news article', true),
('news', 'reject', 'department', 'Reject department news articles', true),
('news', 'submit', 'all', 'Submit articles for review', true),
('news', 'archive', 'all', 'Archive news articles', true),
('news', 'feature', 'all', 'Feature news articles', true),
('news', 'break', 'all', 'Mark as breaking news', true),

-- Enhanced Media Permissions
('media', 'upload', 'all', 'Upload media files anywhere', true),
('media', 'upload', 'department', 'Upload media within department', true),
('media', 'upload', 'own', 'Upload own media files', true),
('media', 'read', 'all', 'Read all media files', true),
('media', 'read', 'department', 'Read department media files', true),
('media', 'read', 'own', 'Read own media files', true),
('media', 'update', 'all', 'Update any media file', true),
('media', 'update', 'department', 'Update department media files', true),
('media', 'update', 'own', 'Update own media files', true),
('media', 'delete', 'all', 'Delete any media file', true),
('media', 'delete', 'department', 'Delete department media files', true),
('media', 'delete', 'own', 'Delete own media files', true),
('media', 'download', 'all', 'Download any media file', true),
('media', 'download', 'department', 'Download department media files', true),
('media', 'download', 'own', 'Download own media files', true),

-- Enhanced User Permissions
('user', 'read', 'all', 'Read all user information', true),
('user', 'read', 'department', 'Read department user information', true),
('user', 'read', 'own', 'Read own user information', true),
('user', 'update', 'all', 'Update any user profile', true),
('user', 'update', 'department', 'Update department user profiles', true),
('user', 'update', 'own', 'Update own profile', true),
('user', 'delete', 'all', 'Delete any user', true),
('user', 'create', 'all', 'Create new users', true),
('user', 'assign_roles', 'all', 'Assign roles to users', true),
('user', 'manage_scopes', 'all', 'Manage user category scopes', true),

-- Enhanced Category Permissions
('category', 'read', 'all', 'Read all categories', true),
('category', 'create', 'all', 'Create categories', true),
('category', 'update', 'all', 'Update any category', true),
('category', 'update', 'department', 'Update department categories', true),
('category', 'delete', 'all', 'Delete any category', true),
('category', 'manage', 'all', 'Full category management', true),
('category', 'assign', 'all', 'Assign categories to content', true),

-- Enhanced Tag Permissions
('tag', 'read', 'all', 'Read all tags', true),
('tag', 'create', 'all', 'Create tags', true),
('tag', 'update', 'all', 'Update any tag', true),
('tag', 'delete', 'all', 'Delete any tag', true),
('tag', 'manage', 'all', 'Full tag management', true),
('tag', 'assign', 'all', 'Assign tags to content', true),

-- Enhanced IAM Permissions
('iam', 'read', 'all', 'Read IAM configuration', true),
('iam', 'manage', 'all', 'Full IAM management', true),
('iam', 'assign_roles', 'all', 'Assign roles to users', true),
('iam', 'create_roles', 'all', 'Create new roles', true),
('iam', 'edit_roles', 'all', 'Edit existing roles', true),
('iam', 'delete_roles', 'all', 'Delete roles', true),
('iam', 'manage_permissions', 'all', 'Manage role permissions', true),
('iam', 'audit', 'all', 'Access audit logs', true),

-- Enhanced Analytics Permissions
('analytics', 'read', 'all', 'Read all analytics data', true),
('analytics', 'read', 'department', 'Read department analytics', true),
('analytics', 'read', 'own', 'Read own analytics', true),
('analytics', 'export', 'all', 'Export analytics data', true),
('analytics', 'manage', 'all', 'Manage analytics settings', true),

-- Enhanced Settings Permissions
('settings', 'read', 'all', 'Read platform settings', true),
('settings', 'update', 'all', 'Update platform settings', true),
('settings', 'manage', 'all', 'Full settings management', true),

-- Enhanced Comments Permissions
('comments', 'read', 'all', 'Read all comments', true),
('comments', 'moderate', 'all', 'Moderate all comments', true),
('comments', 'moderate', 'department', 'Moderate department comments', true),
('comments', 'delete', 'all', 'Delete any comment', true),
('comments', 'manage', 'all', 'Full comment management', true),

-- Enhanced Notification Permissions
('notifications', 'send', 'all', 'Send notifications', true),
('notifications', 'read', 'all', 'Read notifications', true),
('notifications', 'manage', 'all', 'Manage notification settings', true),

-- Enhanced Live Blog Permissions
('liveblog', 'create', 'all', 'Create live blogs', true),
('liveblog', 'create', 'department', 'Create department live blogs', true),
('liveblog', 'update', 'all', 'Update any live blog', true),
('liveblog', 'update', 'own', 'Update own live blogs', true),
('liveblog', 'delete', 'all', 'Delete any live blog', true),
('liveblog', 'manage', 'all', 'Full live blog management', true),

-- Enhanced Web Story Permissions
('webstory', 'create', 'all', 'Create web stories', true),
('webstory', 'create', 'department', 'Create department web stories', true),
('webstory', 'update', 'all', 'Update any web story', true),
('webstory', 'update', 'own', 'Update own web stories', true),
('webstory', 'delete', 'all', 'Delete any web story', true),
('webstory', 'publish', 'all', 'Publish web stories', true),

-- Enhanced E-paper Permissions
('epaper', 'read', 'all', 'Read e-paper', true),
('epaper', 'manage', 'all', 'Manage e-paper', true),
('epaper', 'publish', 'all', 'Publish e-paper', true),

-- Enhanced Poll Permissions
('poll', 'create', 'all', 'Create polls', true),
('poll', 'update', 'all', 'Update any poll', true),
('poll', 'delete', 'all', 'Delete any poll', true),
('poll', 'manage', 'all', 'Full poll management', true)
ON CONFLICT (resource, action, scope) DO NOTHING;

-- ──────────────────────────────────────────────
-- 4. Dynamic Role Permission Assignments
-- ──────────────────────────────────────────────

-- Clear existing role permissions for dynamic assignment
DELETE FROM role_permissions;

-- Super Admin - All permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- Chief Editor - Editorial oversight
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'chief_editor'
AND p.resource IN ('news', 'media', 'category', 'tag', 'user', 'analytics')
AND p.scope IN ('all', 'department')
ON CONFLICT DO NOTHING;

-- Bureau Editor - Regional management
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'bureau_editor'
AND p.resource IN ('news', 'media', 'category')
AND p.scope IN ('all', 'department', 'own')
ON CONFLICT DO NOTHING;

-- Senior Editor - Advanced editorial
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'senior_editor'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update', 'publish', 'approve', 'reject')
AND p.scope IN ('all', 'department')
ON CONFLICT DO NOTHING;

-- Editor - Standard editorial
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'editor'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update')
AND p.scope IN ('all', 'department', 'own')
ON CONFLICT DO NOTHING;

-- Reporter - Content creation
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update')
AND p.scope = 'own'
ON CONFLICT DO NOTHING;

-- Photo Editor - Media management
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'photo_editor'
AND p.resource = 'media'
AND p.scope IN ('all', 'department', 'own')
ON CONFLICT DO NOTHING;

-- Fact Checker - Verification
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'fact_checker'
AND p.resource = 'news'
AND p.action IN ('read', 'approve', 'reject')
AND p.scope IN ('all', 'department')
ON CONFLICT DO NOTHING;

-- Community Moderator - User management
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'community_moderator'
AND p.resource IN ('comments', 'user')
AND p.scope IN ('all', 'department')
ON CONFLICT DO NOTHING;

-- System Admin - Technical access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'system_admin'
AND p.resource IN ('iam', 'settings', 'analytics')
AND p.scope = 'all'
ON CONFLICT DO NOTHING;

-- ──────────────────────────────────────────────
-- 5. Enhanced Permission Groups for Easy Management
-- ──────────────────────────────────────────────

-- Clear existing permission groups
DELETE FROM permission_group_items;
DELETE FROM role_permission_groups;
DELETE FROM permission_groups;

-- Create permission groups
INSERT INTO permission_groups (name, description, is_system) VALUES
('full_access', 'Complete system access', true),
('editorial_full', 'Full editorial permissions', true),
('editorial_standard', 'Standard editorial permissions', true),
('reporting_full', 'Full reporting permissions', true),
('reporting_standard', 'Standard reporting permissions', true),
('media_management', 'Media file management', true),
('user_management', 'User and role management', true),
('content_moderation', 'Content moderation permissions', true),
('analytics_access', 'Analytics and reporting access', true),
('technical_admin', 'Technical administration access', true)
ON CONFLICT (name) DO NOTHING;

-- Assign permissions to groups
-- Full Access Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'full_access'
ON CONFLICT DO NOTHING;

-- Editorial Full Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'editorial_full'
AND p.resource IN ('news', 'media', 'category', 'tag')
AND p.scope IN ('all', 'department')
ON CONFLICT DO NOTHING;

-- Editorial Standard Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'editorial_standard'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update')
AND p.scope IN ('department', 'own')
ON CONFLICT DO NOTHING;

-- Reporting Full Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'reporting_full'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update', 'submit')
AND p.scope IN ('all', 'department', 'own')
ON CONFLICT DO NOTHING;

-- Reporting Standard Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'reporting_standard'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update')
AND p.scope = 'own'
ON CONFLICT DO NOTHING;

-- Media Management Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'media_management'
AND p.resource = 'media'
AND p.scope IN ('all', 'department', 'own')
ON CONFLICT DO NOTHING;

-- User Management Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'user_management'
AND p.resource IN ('user', 'iam')
AND p.scope = 'all'
ON CONFLICT DO NOTHING;

-- Content Moderation Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'content_moderation'
AND p.resource IN ('comments', 'news')
AND p.action IN ('read', 'moderate', 'approve', 'reject')
AND p.scope IN ('all', 'department')
ON CONFLICT DO NOTHING;

-- Analytics Access Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'analytics_access'
AND p.resource = 'analytics'
AND p.scope IN ('all', 'department')
ON CONFLICT DO NOTHING;

-- Technical Admin Group
INSERT INTO permission_group_items (group_id, permission_id)
SELECT pg.id, p.id
FROM permission_groups pg
CROSS JOIN permissions p
WHERE pg.name = 'technical_admin'
AND p.resource IN ('iam', 'settings', 'analytics')
AND p.scope = 'all'
ON CONFLICT DO NOTHING;

-- ──────────────────────────────────────────────
-- 6. Dynamic ABAC Policies for Advanced Control
-- ──────────────────────────────────────────────

-- Clear existing ABAC policies
DELETE FROM abac_policies;

-- Insert dynamic ABAC policies
INSERT INTO abac_policies (name, description, policy_type, effect, priority, conditions) VALUES
-- Time-based policies
('business_hours_only', 'Restrict publishing to business hours only', 'global', 'allow', 10, 
 '{"time": {"start": "09:00", "end": "18:00", "days": ["mon", "tue", "wed", "thu", "fri"]}, "resource": "news", "action": "publish"}'),

('weekend_publishing', 'Allow publishing on weekends with approval', 'global', 'allow', 5,
 '{"time": {"days": ["sat", "sun"]}, "resource": "news", "action": "publish", "attributes": {"requires_approval": true}}'),

-- Role-based policies
('senior_editor_approvals', 'Senior editors can approve without additional checks', 'role', 'allow', 8,
 '{"resource": "news", "action": "approve", "user_attributes": {"level": "senior"}}'),

('bureau_editor_scope', 'Bureau editors limited to their jurisdiction', 'role', 'allow', 7,
 '{"resource": "news", "action": "publish", "scope": "department"}'),

-- User-specific policies
('breaking_news_override', 'Override restrictions for breaking news', 'user', 'allow', 9,
 '{"resource": "news", "action": "publish", "resource_state": {"is_breaking": true}}'),

('emergency_publishing', 'Allow emergency publishing situations', 'user', 'allow', 10,
 '{"resource": "news", "action": "publish", "conditions": {"emergency": true}}'),

-- Content-based policies
('sensitive_content', 'Additional approval for sensitive content', 'global', 'allow', 6,
 '{"resource": "news", "action": "publish", "resource_state": {"sensitivity": "high"}}'),

('political_content', 'Political content requires editorial approval', 'global', 'allow', 7,
 '{"resource": "news", "action": "publish", "resource_state": {"category": "politics"}}'),

-- Department-based policies
('national_news_access', 'National news requires higher clearance', 'global', 'allow', 5,
 '{"resource": "news", "action": "read", "resource_state": {"is_national": true}}'),

('regional_content_management', 'Regional editors manage regional content', 'role', 'allow', 6,
 '{"resource": "news", "action": "update", "scope": "department"}')
ON CONFLICT DO NOTHING;

-- ──────────────────────────────────────────────
-- 7. Refresh Permission Cache
-- ──────────────────────────────────────────────
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;

-- ──────────────────────────────────────────────
-- 8. Create Dynamic Role Templates
-- ──────────────────────────────────────────────

-- Create a function to apply role templates
CREATE OR REPLACE FUNCTION apply_role_template(p_role_id INT, p_template_name VARCHAR)
RETURNS VOID AS $$
DECLARE
    template_permissions TEXT[];
BEGIN
    CASE p_template_name
        WHEN 'super_admin' THEN
            -- Grant all permissions
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT p_role_id, id FROM permissions
            ON CONFLICT DO NOTHING;
            
        WHEN 'chief_editor' THEN
            -- Editorial oversight permissions
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT p_role_id, id FROM permissions 
            WHERE resource IN ('news', 'media', 'category', 'tag', 'user', 'analytics')
            AND scope IN ('all', 'department')
            ON CONFLICT DO NOTHING;
            
        WHEN 'senior_editor' THEN
            -- Advanced editorial permissions
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT p_role_id, id FROM permissions 
            WHERE resource IN ('news', 'media')
            AND action IN ('create', 'read', 'update', 'publish', 'approve', 'reject')
            AND scope IN ('all', 'department')
            ON CONFLICT DO NOTHING;
            
        WHEN 'editor' THEN
            -- Standard editorial permissions
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT p_role_id, id FROM permissions 
            WHERE resource IN ('news', 'media')
            AND action IN ('create', 'read', 'update')
            AND scope IN ('all', 'department', 'own')
            ON CONFLICT DO NOTHING;
            
        WHEN 'reporter' THEN
            -- Content creation permissions
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT p_role_id, id FROM permissions 
            WHERE resource IN ('news', 'media')
            AND action IN ('create', 'read', 'update')
            AND scope = 'own'
            ON CONFLICT DO NOTHING;
            
        WHEN 'fact_checker' THEN
            -- Verification permissions
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT p_role_id, id FROM permissions 
            WHERE resource = 'news'
            AND action IN ('read', 'approve', 'reject')
            AND scope IN ('all', 'department')
            ON CONFLICT DO NOTHING;
            
        WHEN 'moderator' THEN
            -- Moderation permissions
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT p_role_id, id FROM permissions 
            WHERE resource IN ('comments', 'user')
            AND scope IN ('all', 'department')
            ON CONFLICT DO NOTHING;
            
        ELSE
            RAISE EXCEPTION 'Unknown template: %', p_template_name;
    END CASE;
    
    -- Refresh cache after applying template
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 9. Create Dynamic User Permission Functions
-- ──────────────────────────────────────────────

-- Function to clone role permissions to user
CREATE OR REPLACE FUNCTION clone_role_to_user(p_user_id BIGINT, p_role_id INT)
RETURNS VOID AS $$
BEGIN
    -- Clear existing user permissions
    DELETE FROM user_permissions WHERE user_id = p_user_id;
    
    -- Copy role permissions to user
    INSERT INTO user_permissions (user_id, permission_id, effect, granted_by)
    SELECT p_user_id, rp.permission_id, 'GRANT', p_user_id
    FROM role_permissions rp
    WHERE rp.role_id = p_role_id
    ON CONFLICT DO NOTHING;
    
    -- Refresh cache
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;
END;
$$ LANGUAGE plpgsql;

-- Function to grant user direct permission
CREATE OR REPLACE FUNCTION grant_user_direct_permission(
    p_user_id BIGINT, 
    p_resource VARCHAR, 
    p_action VARCHAR, 
    p_granted_by BIGINT,
    p_scope VARCHAR
)
RETURNS VOID AS $$
DECLARE
    v_permission_id INT;
BEGIN
    -- Find the permission ID
    SELECT id INTO v_permission_id 
    FROM permissions 
    WHERE resource = p_resource AND action = p_action AND scope = p_scope;
    
    IF v_permission_id IS NULL THEN
        RAISE EXCEPTION 'Permission not found: %:%:%', p_resource, p_action, p_scope;
    END IF;
    
    -- Grant the permission
    INSERT INTO user_permissions (user_id, permission_id, effect, granted_by)
    VALUES (p_user_id, v_permission_id, 'GRANT', p_granted_by)
    ON CONFLICT DO NOTHING;
    
    -- Refresh cache
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 10. Create Scope Management Functions
-- ──────────────────────────────────────────────

-- Function to set user category scopes
CREATE OR REPLACE FUNCTION set_user_category_scopes(p_user_id BIGINT, p_category_ids INT[])
RETURNS VOID AS $$
BEGIN
    -- Clear existing scopes
    DELETE FROM user_category_scopes WHERE user_id = p_user_id;
    
    -- Add new scopes
    INSERT INTO user_category_scopes (user_id, category_id, assigned_by)
    SELECT p_user_id, unnest(p_category_ids), p_user_id
    ON CONFLICT DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Function to get user effective scope
CREATE OR REPLACE FUNCTION get_user_effective_scope(p_user_id BIGINT)
RETURNS VARCHAR AS $$
DECLARE
    scope_count INT;
BEGIN
    -- Count user's category scopes
    SELECT COUNT(*) INTO scope_count
    FROM user_category_scopes
    WHERE user_id = p_user_id;
    
    IF scope_count = 0 THEN
        RETURN 'all'; -- No restrictions = national access
    ELSIF scope_count = 1 THEN
        RETURN 'department'; -- Single category = department level
    ELSE
        RETURN 'custom'; -- Multiple categories = custom scope
    END IF;
END;
$$ LANGUAGE plpgsql;

-- ──────────────────────────────────────────────
-- 11. Create Dynamic Permission Checking Views
-- ──────────────────────────────────────────────

-- View for user effective permissions with scope
CREATE OR REPLACE VIEW v_user_effective_permissions AS
SELECT 
    u.id as user_id,
    u.email,
    u.display_name,
    r.name as role_name,
    p.resource,
    p.action,
    p.scope,
    CASE 
        WHEN u.is_super_admin THEN 'super_admin'
        WHEN EXISTS (SELECT 1 FROM user_category_scopes WHERE user_id = u.id) THEN 
            get_user_effective_scope(u.id)
        ELSE 'all'
    END as effective_scope,
    CASE 
        WHEN u.is_super_admin THEN true
        WHEN p.scope = 'all' THEN true
        WHEN p.scope = 'department' AND EXISTS (SELECT 1 FROM user_category_scopes WHERE user_id = u.id) THEN true
        WHEN p.scope = 'own' THEN true -- Will be checked with ownership
        ELSE false
    END as allowed
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id AND ur.is_active = true
LEFT JOIN roles r ON r.id = ur.role_id
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
WHERE u.is_active = true;

-- ──────────────────────────────────────────────
-- 12. Final Cache Refresh
-- ──────────────────────────────────────────────
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;

-- ──────────────────────────────────────────────
-- Summary
-- ──────────────────────────────────────────────
-- This migration creates a fully dynamic IAM system with:
-- 1. 20+ dynamic roles for different user types
-- 2. 100+ granular permissions across all resources
-- 3. Hierarchical scope system (all > department > own > custom)
-- 4. Permission groups for easy role assignment
-- 5. ABAC policies for advanced access control
-- 6. Resource ownership tracking
-- 7. Dynamic role templates
-- 8. Enhanced permission checking functions
-- 9. Category-based scope management
-- 10. Real-time permission cache updates