-- 008_configure_reporter_enhanced_permissions.up.sql
-- Configure Reporter role with enhanced permissions for user-friendly matrix

-- Clear existing permissions for reporter role
DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'reporter');

-- Grant enhanced permissions for reporter (own scope for read, all for create/update)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND (
  -- News permissions
  (p.resource = 'news' AND p.action IN ('read_own', 'create', 'update', 'search')) OR
  -- Media permissions  
  (p.resource = 'media' AND p.action IN ('read_own', 'create', 'update', 'search')) OR
  -- Basic category/tag read
  (p.resource = 'category' AND p.action = 'read_all') OR
  (p.resource = 'tag' AND p.action = 'read_all') OR
  -- Own analytics
  (p.resource = 'analytics' AND p.action = 'read_own')
)
ON CONFLICT DO NOTHING;

-- Refresh the permission cache
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;