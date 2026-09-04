-- 017_streamline_menus_and_actions.up.sql
-- Streamline menu labels to clean, concise newsroom navigation names
-- Normalize menu_actions into canonical standard action set: view, create, edit, delete, publish, approve, export

-- 1. Streamline Menu Names & Labels
UPDATE menus SET label = 'Dashboard' WHERE name = 'dashboard';
UPDATE menus SET label = 'Articles' WHERE name = 'articles';
UPDATE menus SET label = 'Categories & Desks' WHERE name = 'categories';
UPDATE menus SET label = 'Tags & Topics' WHERE name = 'tags';
UPDATE menus SET label = 'Media Library' WHERE name = 'media_library';
UPDATE menus SET label = 'Live Blogs' WHERE name = 'live_blogs';
UPDATE menus SET label = 'Web Stories' WHERE name = 'web_stories';
UPDATE menus SET label = 'E-Paper' WHERE name = 'epaper';
UPDATE menus SET label = 'Comments & Community' WHERE name = 'comments';
UPDATE menus SET label = 'Roles & Permissions' WHERE name = 'roles';
UPDATE menus SET label = 'Staff & Users' WHERE name = 'users';
UPDATE menus SET label = 'Analytics' WHERE name = 'analytics';
UPDATE menus SET label = 'Settings' WHERE name = 'settings';

-- Set clean sort orders
UPDATE menus SET sort_order = 1 WHERE name = 'dashboard';
UPDATE menus SET sort_order = 2 WHERE name = 'articles';
UPDATE menus SET sort_order = 3 WHERE name = 'categories';
UPDATE menus SET sort_order = 4 WHERE name = 'tags';
UPDATE menus SET sort_order = 5 WHERE name = 'media_library';
UPDATE menus SET sort_order = 6 WHERE name = 'live_blogs';
UPDATE menus SET sort_order = 7 WHERE name = 'web_stories';
UPDATE menus SET sort_order = 8 WHERE name = 'epaper';
UPDATE menus SET sort_order = 9 WHERE name = 'comments';
UPDATE menus SET sort_order = 10 WHERE name = 'roles';
UPDATE menus SET sort_order = 11 WHERE name = 'users';
UPDATE menus SET sort_order = 12 WHERE name = 'analytics';
UPDATE menus SET sort_order = 13 WHERE name = 'settings';

-- 2. Consolidate and Normalize Menu Actions
CREATE TEMP TABLE tmp_granted_perms AS
SELECT DISTINCT rma.role_id, ma.menu_id, 
       CASE 
           WHEN LOWER(ma.action) IN ('view') THEN 'view'
           WHEN LOWER(ma.action) IN ('add', 'create') THEN 'create'
           WHEN LOWER(ma.action) IN ('edit') THEN 'edit'
           WHEN LOWER(ma.action) IN ('delete') THEN 'delete'
           WHEN LOWER(ma.action) IN ('publish') THEN 'publish'
           WHEN LOWER(ma.action) IN ('approve') THEN 'approve'
           WHEN LOWER(ma.action) IN ('export', 'reject') THEN 'export'
           ELSE 'view'
       END as action_name
FROM role_menu_actions rma
JOIN menu_actions ma ON ma.id = rma.menu_action_id;

-- Clear dependent grants and recreate canonical actions
TRUNCATE role_menu_actions;
DELETE FROM user_permission_overrides;
DELETE FROM menu_actions;

-- Populate canonical action set for every menu
INSERT INTO menu_actions (menu_id, action, label)
SELECT m.id, a.action, INITCAP(a.action)
FROM menus m
CROSS JOIN (
    VALUES ('view'), ('create'), ('edit'), ('delete'), ('publish'), ('approve'), ('export')
) a(action);

-- Restore role grants
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT DISTINCT t.role_id, ma.id
FROM tmp_granted_perms t
JOIN menu_actions ma ON ma.menu_id = t.menu_id AND ma.action = t.action_name
ON CONFLICT (role_id, menu_action_id) DO NOTHING;

DROP TABLE tmp_granted_perms;

-- 3. Ensure Super Administrator has ALL permissions
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT r.id, ma.id
FROM roles r
CROSS JOIN menu_actions ma
WHERE r.id = 1 OR LOWER(r.name) = 'super administrator'
ON CONFLICT (role_id, menu_action_id) DO NOTHING;

-- 4. Seed standard Editor permissions
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT r.id, ma.id
FROM roles r
JOIN menus m ON m.name IN ('dashboard', 'articles', 'categories', 'tags', 'media_library', 'live_blogs', 'web_stories', 'epaper', 'comments', 'analytics')
JOIN menu_actions ma ON ma.menu_id = m.id
WHERE LOWER(r.name) = 'senior editor' OR LOWER(r.name) = 'editor'
ON CONFLICT (role_id, menu_action_id) DO NOTHING;
