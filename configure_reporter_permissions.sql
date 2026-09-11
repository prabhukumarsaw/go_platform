-- Configure Reporter role with own-scope permissions
-- This will allow reporters to only see and manage their own articles and media

-- First, let's check the current reporter role ID
SELECT id, name FROM roles WHERE name = 'reporter';

-- Then grant appropriate permissions to the reporter role
-- For a reporter, we want: create, read, update (own scope) for news and media

-- Clear existing permissions for reporter role
DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'reporter');

-- Grant own-scope permissions for reporter
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND p.resource IN ('news', 'media')
AND p.action IN ('create', 'read', 'update')
AND p.scope = 'own'
ON CONFLICT DO NOTHING;

-- Grant basic system permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'reporter'
AND p.resource IN ('category', 'tag')
AND p.action = 'read'
AND p.scope = 'all'
ON CONFLICT DO NOTHING;

-- Refresh the permission cache
REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_permissions_cache;

-- Verify the permissions
SELECT 
    r.name as role_name,
    p.resource,
    p.action,
    p.scope,
    p.description
FROM roles r
JOIN role_permissions rp ON rp.role_id = r.id
JOIN permissions p ON p.id = rp.permission_id
WHERE r.name = 'reporter'
ORDER BY p.resource, p.action, p.scope;