-- 010_enhanced_rbac_abac_seed.up.sql
-- Enhanced RBAC/ABAC seed with proper role hierarchy and approval workflows
-- SuperAdmin: All permissions
-- Editor: Dynamic permissions assigned by SuperAdmin
-- Author: Own content only, no delete
-- Reporter: Own content only, requires approval for publishing

-- Clear existing roles and permissions
DELETE FROM role_permissions;
DELETE FROM user_permissions;
DELETE FROM resource_grants;
DELETE FROM permissions;

-- Enhanced permissions with RBAC/ABAC support
INSERT INTO permissions (resource, action, scope, description, is_system) VALUES
-- Article permissions with RBAC/ABAC
('articles', 'read', 'all', 'Read all articles (SuperAdmin only)', true),
('articles', 'read', 'own', 'Read own articles (Author/Reporter)', true),
('articles', 'read', 'department', 'Read department articles (Editor)', true),
('articles', 'create', 'all', 'Create articles anywhere (SuperAdmin/Editor)', true),
('articles', 'create', 'own', 'Create own articles (Author/Reporter)', true),
('articles', 'update', 'all', 'Update any articles (SuperAdmin/Editor)', true),
('articles', 'update', 'own', 'Update own articles (Author/Reporter)', true),
('articles', 'delete', 'all', 'Delete any articles (SuperAdmin/Editor)', true),
('articles', 'publish', 'all', 'Publish any articles (SuperAdmin/Editor)', true),
('articles', 'publish', 'approval', 'Publish requires approval (Author/Reporter)', true),
('articles', 'approve', 'all', 'Approve articles (Editor/Reviewer)', true),
('articles', 'review', 'all', 'Review articles (Editor/Reviewer)', true),
('articles', 'reject', 'all', 'Reject articles (Editor/Reviewer)', true),
('articles', 'export', 'all', 'Export articles (SuperAdmin/Editor)', true),

-- Media permissions with RBAC/ABAC
('media', 'read', 'all', 'Read all media (SuperAdmin/Editor)', true),
('media', 'read', 'own', 'Read own media (Author/Reporter)', true),
('media', 'read', 'department', 'Read department media (Editor)', true),
('media', 'upload', 'all', 'Upload media anywhere (SuperAdmin/Editor)', true),
('media', 'upload', 'own', 'Upload own media (Author/Reporter)', true),
('media', 'update', 'all', 'Update any media (SuperAdmin/Editor)', true),
('media', 'update', 'own', 'Update own media (Author/Reporter)', true),
('media', 'delete', 'all', 'Delete any media (SuperAdmin/Editor)', true),
('media', 'publish', 'all', 'Publish any media (SuperAdmin/Editor)', true),
('media', 'publish', 'approval', 'Publish requires approval (Author/Reporter)', true),
('media', 'approve', 'all', 'Approve media (Editor/Reviewer)', true),
('media', 'review', 'all', 'Review media (Editor/Reviewer)', true),
('media', 'reject', 'all', 'Reject media (Editor/Reviewer)', true),
('media', 'export', 'all', 'Export media (SuperAdmin/Editor)', true),

-- User management (SuperAdmin only)
('users', 'read', 'all', 'Read all users (SuperAdmin only)', true),
('users', 'create', 'all', 'Create users (SuperAdmin only)', true),
('users', 'update', 'all', 'Update users (SuperAdmin only)', true),
('users', 'delete', 'all', 'Delete users (SuperAdmin only)', true),
('users', 'assign_role', 'all', 'Assign roles to users (SuperAdmin only)', true),

-- Category management
('categories', 'read', 'all', 'Read all categories', true),
('categories', 'create', 'all', 'Create categories (SuperAdmin/Editor)', true),
('categories', 'update', 'all', 'Update categories (SuperAdmin/Editor)', true),
('categories', 'delete', 'all', 'Delete categories (SuperAdmin only)', true),

-- Tag management
('tags', 'read', 'all', 'Read all tags', true),
('tags', 'create', 'all', 'Create tags (SuperAdmin/Editor)', true),
('tags', 'update', 'all', 'Update tags (SuperAdmin/Editor)', true),
('tags', 'delete', 'all', 'Delete tags (SuperAdmin/Editor)', true),

-- Analytics
('analytics', 'read', 'all', 'Read all analytics (SuperAdmin/Editor)', true),
('analytics', 'read', 'own', 'Read own analytics (Author/Reporter)', true),

-- Settings (SuperAdmin only)
('settings', 'read', 'all', 'Read all settings (SuperAdmin only)', true),
('settings', 'update', 'all', 'Update settings (SuperAdmin only)', true),

-- IAM management (SuperAdmin only)
('iam', 'read', 'all', 'Read IAM settings (SuperAdmin only)', true),
('iam', 'update', 'all', 'Update IAM settings (SuperAdmin only)', true),
('iam', 'assign_permissions', 'all', 'Assign permissions (SuperAdmin only)', true),
('iam', 'assign_approval', 'all', 'Assign approval rights (SuperAdmin only)', true)
ON CONFLICT DO NOTHING;

-- Configure SuperAdmin: All permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- Configure Editor: Dynamic permissions (default + ability to be assigned more)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'editor'
AND (
  -- Full access to articles and media (by default)
  (p.resource = 'articles' AND p.action IN ('read', 'update', 'approve', 'review', 'reject', 'export')) OR
  (p.resource = 'media' AND p.action IN ('read', 'update', 'approve', 'review', 'reject', 'export')) OR
  -- Can create and publish without approval
  (p.resource = 'articles' AND p.action IN ('create', 'publish')) OR
  (p.resource = 'media' AND p.action IN ('upload', 'publish')) OR
  -- Basic category/tag management
  (p.resource = 'categories' AND p.action IN ('read', 'create', 'update')) OR
  (p.resource = 'tags' AND p.action IN ('read', 'create', 'update')) OR
  -- Analytics access
  (p.resource = 'analytics' AND p.action = 'read')
)
ON CONFLICT DO NOTHING;

-- Configure Author: Own content only, no delete, approval workflow
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'author'
AND (
  -- Read and create own content
  (p.resource = 'articles' AND p.action = 'read' AND p.scope = 'own') OR
  (p.resource = 'articles' AND p.action = 'create' AND p.scope = 'own') OR
  (p.resource = 'articles' AND p.action = 'update' AND p.scope = 'own') OR
  (p.resource = 'articles' AND p.action = 'publish' AND p.scope = 'approval') OR
  -- Media similar to articles
  (p.resource = 'media' AND p.action = 'read' AND p.scope = 'own') OR
  (p.resource = 'media' AND p.action = 'upload' AND p.scope = 'own') OR
  (p.resource = 'media' AND p.action = 'update' AND p.scope = 'own') OR
  (p.resource = 'media' AND p.action = 'publish' AND p.scope = 'approval') OR
  -- Basic read access to categories/tags
  (p.resource = 'categories' AND p.action = 'read') OR
  (p.resource = 'tags' AND p.action = 'read') OR
  -- Own analytics
  (p.resource = 'analytics' AND p.action = 'read' AND p.scope = 'own')
)
ON CONFLICT DO NOTHING;

-- Configure Reporter: Same as Author (own content, approval workflow)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND (
  -- Read and create own content
  (p.resource = 'articles' AND p.action = 'read' AND p.scope = 'own') OR
  (p.resource = 'articles' AND p.action = 'create' AND p.scope = 'own') OR
  (p.resource = 'articles' AND p.action = 'update' AND p.scope = 'own') OR
  (p.resource = 'articles' AND p.action = 'publish' AND p.scope = 'approval') OR
  -- Media similar to articles
  (p.resource = 'media' AND p.action = 'read' AND p.scope = 'own') OR
  (p.resource = 'media' AND p.action = 'upload' AND p.scope = 'own') OR
  (p.resource = 'media' AND p.action = 'update' AND p.scope = 'own') OR
  (p.resource = 'media' AND p.action = 'publish' AND p.scope = 'approval') OR
  -- Basic read access to categories/tags
  (p.resource = 'categories' AND p.action = 'read') OR
  (p.resource = 'tags' AND p.action = 'read') OR
  -- Own analytics
  (p.resource = 'analytics' AND p.action = 'read' AND p.scope = 'own')
)
ON CONFLICT DO NOTHING;

-- Configure Citizen: Read-only public content
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'citizen'
AND (
  -- Only read access to published content
  (p.resource = 'articles' AND p.action = 'read' AND p.scope = 'all') OR
  (p.resource = 'media' AND p.action = 'read' AND p.scope = 'all')
)
ON CONFLICT DO NOTHING;

-- Create approval workflow tables
CREATE TABLE IF NOT EXISTS approval_workflows (
    id SERIAL PRIMARY KEY,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT NOT NULL,
    requested_by BIGINT NOT NULL,
    requested_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'pending', -- pending, approved, rejected
    approved_by BIGINT,
    approved_at TIMESTAMP,
    rejection_reason TEXT,
    comments TEXT
);

CREATE INDEX IF NOT EXISTS idx_approval_workflows_resource ON approval_workflows(resource_type, resource_id, status);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_requested_by ON approval_workflows(requested_by, status);

-- Create dynamic permission assignment table (for SuperAdmin to assign extra permissions)
CREATE TABLE IF NOT EXISTS dynamic_permission_assignments (
    id SERIAL PRIMARY KEY,
    role_id INT NOT NULL,
    permission_id INT NOT NULL,
    assigned_by BIGINT NOT NULL,
    assigned_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_dynamic_assignments_role ON dynamic_permission_assignments(role_id, is_active);
CREATE INDEX IF NOT EXISTS idx_dynamic_assignments_permission ON dynamic_permission_assignments(permission_id, is_active);

-- Create approval assignment table (for assigning approval rights)
CREATE TABLE IF NOT EXISTS approval_assignments (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    resource_type VARCHAR(50) NOT NULL, -- articles, media, etc.
    can_approve BOOLEAN DEFAULT FALSE,
    assigned_by BIGINT NOT NULL,
    assigned_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_approval_assignments_user ON approval_assignments(user_id, resource_type, is_active);

-- Refresh the permission cache
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;