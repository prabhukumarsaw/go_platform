-- 006_configure_reporter_permissions.up.sql
-- Configure Reporter role with own-scope permissions for stable dynamic access
-- This allows reporters to only see and manage their own articles and media

-- Clear existing permissions for reporter role
DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'reporter');

-- Grant own-scope permissions for reporter (news and media)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update')
AND p.scope = 'own'
ON CONFLICT DO NOTHING;

-- Grant basic system permissions (categories and tags for all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND p.resource IN ('category', 'tag')
AND p.action = 'read'
AND p.scope = 'all'
ON CONFLICT DO NOTHING;

-- Grant basic analytics read permissions (own analytics only)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND p.resource = 'analytics'
AND p.action = 'read'
AND p.scope = 'own'
ON CONFLICT DO NOTHING;

-- Refresh the permission cache
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;