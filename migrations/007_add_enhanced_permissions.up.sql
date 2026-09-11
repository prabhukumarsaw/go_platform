-- 007_add_enhanced_permissions.up.sql
-- Add enhanced permissions for user-friendly matrix
-- These permissions combine resource-action pairs without complex scope logic

-- Add enhanced read permissions (no scope complexity)
INSERT INTO permissions (resource, action, scope, description, is_system) VALUES
-- News permissions
('news', 'read_all', 'all', 'Read all news articles', true),
('news', 'read_own', 'own', 'Read own news articles only', true),
('news', 'create', 'all', 'Create news articles', true),
('news', 'update', 'all', 'Update news articles', true),
('news', 'delete', 'all', 'Delete news articles', true),
('news', 'publish', 'all', 'Publish news articles', true),
('news', 'approve', 'all', 'Approve news articles', true),
('news', 'review', 'all', 'Review news articles', true),
('news', 'reject', 'all', 'Reject news articles', true),
('news', 'search', 'all', 'Search news articles', true),

-- Media permissions
('media', 'read_all', 'all', 'Read all media files', true),
('media', 'read_own', 'own', 'Read own media files only', true),
('media', 'create', 'all', 'Upload media files', true),
('media', 'update', 'all', 'Update media files', true),
('media', 'delete', 'all', 'Delete media files', true),
('media', 'publish', 'all', 'Publish media files', true),
('media', 'approve', 'all', 'Approve media files', true),
('media', 'review', 'all', 'Review media files', true),
('media', 'reject', 'all', 'Reject media files', true),
('media', 'search', 'all', 'Search media files', true),

-- Category permissions
('category', 'read_all', 'all', 'Read all categories', true),
('category', 'create', 'all', 'Create categories', true),
('category', 'update', 'all', 'Update categories', true),
('category', 'delete', 'all', 'Delete categories', true),

-- Tag permissions
('tag', 'read_all', 'all', 'Read all tags', true),
('tag', 'create', 'all', 'Create tags', true),
('tag', 'update', 'all', 'Update tags', true),
('tag', 'delete', 'all', 'Delete tags', true),

-- User permissions
('user', 'read_all', 'all', 'Read all users', true),
('user', 'create', 'all', 'Create users', true),
('user', 'update', 'all', 'Update users', true),
('user', 'delete', 'all', 'Delete users', true),

-- Analytics permissions
('analytics', 'read_all', 'all', 'Read all analytics', true),
('analytics', 'read_own', 'own', 'Read own analytics only', true),

-- Settings permissions
('settings', 'read_all', 'all', 'Read all settings', true),
('settings', 'update', 'all', 'Update settings', true),

-- IAM permissions
('iam', 'read_all', 'all', 'Read all IAM data', true),
('iam', 'create', 'all', 'Create IAM entries', true),
('iam', 'update', 'all', 'Update IAM entries', true),
('iam', 'delete', 'all', 'Delete IAM entries', true)
ON CONFLICT DO NOTHING;

-- Refresh the permission cache
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;