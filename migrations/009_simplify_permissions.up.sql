-- 009_simplify_permissions.up.sql
-- Simplify permissions to core actions only
-- Remove complex scope logic, keep simple: View All, View Own, Create, Edit, Delete, Publish, Approve, Review, Reject

-- Clear all existing permissions
DELETE FROM role_permissions;
DELETE FROM user_permissions;

-- Add simplified core permissions
INSERT INTO permissions (resource, action, scope, description, is_system) VALUES
-- News permissions
('news', 'view_all', 'all', 'View all news articles', true),
('news', 'view_own', 'own', 'View own news articles only', true),
('news', 'create', 'all', 'Create news articles', true),
('news', 'edit', 'all', 'Edit news articles', true),
('news', 'delete', 'all', 'Delete news articles', true),
('news', 'publish', 'all', 'Publish news articles', true),
('news', 'approve', 'all', 'Approve news articles', true),
('news', 'review', 'all', 'Review news articles', true),
('news', 'reject', 'all', 'Reject news articles', true),
('news', 'export', 'all', 'Export news articles', true),

-- Media permissions
('media', 'view_all', 'all', 'View all media files', true),
('media', 'view_own', 'own', 'View own media files only', true),
('media', 'upload', 'all', 'Upload media files', true),
('media', 'edit', 'all', 'Edit media files', true),
('media', 'delete', 'all', 'Delete media files', true),
('media', 'publish', 'all', 'Publish media files', true),
('media', 'approve', 'all', 'Approve media files', true),
('media', 'review', 'all', 'Review media files', true),
('media', 'reject', 'all', 'Reject media files', true),
('media', 'export', 'all', 'Export media files', true),

-- Category permissions
('category', 'view', 'all', 'View categories', true),
('category', 'create', 'all', 'Create categories', true),
('category', 'edit', 'all', 'Edit categories', true),
('category', 'delete', 'all', 'Delete categories', true),

-- Tag permissions
('tag', 'view', 'all', 'View tags', true),
('tag', 'create', 'all', 'Create tags', true),
('tag', 'edit', 'all', 'Edit tags', true),
('tag', 'delete', 'all', 'Delete tags', true),

-- User permissions (SuperAdmin only)
('user', 'view', 'all', 'View users', true),
('user', 'create', 'all', 'Create users', true),
('user', 'edit', 'all', 'Edit users', true),
('user', 'delete', 'all', 'Delete users', true),

-- Analytics permissions
('analytics', 'view', 'all', 'View analytics', true),
('analytics', 'view_own', 'own', 'View own analytics only', true),

-- Settings permissions (SuperAdmin only)
('settings', 'view', 'all', 'View settings', true),
('settings', 'edit', 'all', 'Edit settings', true),

-- IAM permissions (SuperAdmin only)
('iam', 'view', 'all', 'View IAM settings', true),
('iam', 'edit', 'all', 'Edit IAM settings', true)
ON CONFLICT DO NOTHING;

-- Configure core roles with simplified permissions

-- SuperAdmin: All permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- Editor: Full access to news and media (except delete)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'editor'
AND p.resource IN ('news', 'media', 'category', 'tag', 'analytics')
AND p.action IN ('view_all', 'create', 'edit', 'publish', 'approve', 'review', 'export')
ON CONFLICT DO NOTHING;

-- Reporter: Own access only, can create and edit own content
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND (
  (p.resource = 'news' AND p.action IN ('view_own', 'create', 'edit')) OR
  (p.resource = 'media' AND p.action IN ('view_own', 'upload', 'edit')) OR
  (p.resource = 'category' AND p.action = 'view') OR
  (p.resource = 'tag' AND p.action = 'view') OR
  (p.resource = 'analytics' AND p.action = 'view_own')
)
ON CONFLICT DO NOTHING;

-- Refresh the permission cache
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;