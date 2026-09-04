-- 007_seed_data.up.sql
-- Seeds: Global Platform, Hierarchical Taxonomy Tree (States > Districts > Cities + Topics),
-- Unified Enterprise Roles, Menus, Actions, Grants, and Staff Accounts.

-- ──────────────────────────────────────────────
-- 1. UNIFIED GLOBAL PLATFORM DESK
-- ──────────────────────────────────────────────
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants') THEN
        INSERT INTO tenants (id, name, slug, is_national)
        VALUES (1, 'National Newsroom Desk', 'national', TRUE)
        ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, slug = EXCLUDED.slug, is_national = TRUE;
    END IF;
END $$;

-- ──────────────────────────────────────────────
-- 2. MENUS & SECTIONS
-- ──────────────────────────────────────────────
ALTER TABLE menus ADD COLUMN IF NOT EXISTS path VARCHAR(255) DEFAULT '';

INSERT INTO menus (name, label, path, is_active) VALUES
    ('dashboard',     'Executive Newsroom & Analytics Dashboard', '/panel/dashboard', TRUE),
    ('articles',      'Article Editorial CMS & Publishing',      '/panel/articles',  TRUE),
    ('categories',    'Hierarchical Taxonomy & Regional Desks',   '/panel/categories',TRUE),
    ('tags',          'Keyword Tags & Entity Registry',          '/panel/tags',      TRUE),
    ('media_library', 'Digital Asset Management & S3 Storage',    '/panel/media',     TRUE),
    ('live_blogs',    'Real-Time Live Coverage & Updates',        '/panel/liveblog',  TRUE),
    ('web_stories',   'Visual AMP Web Stories',                  '/panel/webstories',TRUE),
    ('epaper',        'Digital Print Editions & EPapers',        '/panel/epaper',    TRUE),
    ('comments',      'Audience Engagement & Auto-Moderation',    '/panel/comments',  TRUE),
    ('roles',         'IAM Staff Roles & Role-Based Access',     '/panel/roles',     TRUE),
    ('users',         'Staff & User Identity Management',        '/panel/users',     TRUE),
    ('analytics',     'Real-Time Editorial & Traffic Metrics',   '/panel/analytics', TRUE),
    ('settings',      'Platform Identity, Brand & Caching',      '/panel/settings',  TRUE)
ON CONFLICT (name) DO UPDATE SET label = EXCLUDED.label, path = EXCLUDED.path, is_active = TRUE;

-- Remove deprecated tenant menus if any exist
DELETE FROM menus WHERE LOWER(name) IN ('tenants', 'districts') OR LOWER(path) IN ('/panel/tenants', '/panel/districts');

-- ──────────────────────────────────────────────
-- 3. MENU ACTIONS (Granular RBAC Permissions)
-- ──────────────────────────────────────────────
DO $$
DECLARE
    m RECORD;
    actions TEXT[] := ARRAY['VIEW', 'ADD', 'EDIT', 'DELETE', 'PUBLISH', 'APPROVE', 'REJECT'];
    act TEXT;
BEGIN
    FOR m IN SELECT id, name FROM menus LOOP
        FOREACH act IN ARRAY actions LOOP
            INSERT INTO menu_actions (menu_id, action)
            VALUES (m.id, act)
            ON CONFLICT DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- ──────────────────────────────────────────────
-- 4. SYSTEM ROLES
-- ──────────────────────────────────────────────
INSERT INTO roles (tenant_id, name, description, is_system) VALUES
    (1, 'super_admin', 'Full platform-wide root authority', TRUE),
    (1, 'editor',      'Editorial publishing and review authority', TRUE),
    (1, 'sub_editor',  'Content review, editing, and approval desk', TRUE),
    (1, 'reporter',    'Field journalism and draft creation desk', TRUE),
    (1, 'moderator',   'Community and comment moderation authority', TRUE)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- ──────────────────────────────────────────────
-- 5. ROLE PERMISSION GRANTS
-- ──────────────────────────────────────────────
DO $$
DECLARE
    r_super INT;
    r_editor INT;
    r_sub INT;
    r_reporter INT;
    r_mod INT;
    ma RECORD;
BEGIN
    SELECT id INTO r_super FROM roles WHERE tenant_id = 1 AND name = 'super_admin';
    SELECT id INTO r_editor FROM roles WHERE tenant_id = 1 AND name = 'editor';
    SELECT id INTO r_sub FROM roles WHERE tenant_id = 1 AND name = 'sub_editor';
    SELECT id INTO r_reporter FROM roles WHERE tenant_id = 1 AND name = 'reporter';
    SELECT id INTO r_mod FROM roles WHERE tenant_id = 1 AND name = 'moderator';

    -- Super Admin gets ALL permissions
    IF r_super IS NOT NULL THEN
        FOR ma IN SELECT id FROM menu_actions LOOP
            INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
            VALUES (r_super, ma.id, 1)
            ON CONFLICT DO NOTHING;
        END LOOP;
    END IF;

    -- Editor gets full editorial & media actions
    IF r_editor IS NOT NULL THEN
        FOR ma IN
            SELECT ma2.id FROM menu_actions ma2
            JOIN menus m ON m.id = ma2.menu_id
            WHERE m.name IN ('dashboard', 'articles', 'categories', 'tags', 'media_library', 'live_blogs', 'web_stories', 'epaper', 'comments', 'analytics')
        LOOP
            INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
            VALUES (r_editor, ma.id, 1)
            ON CONFLICT DO NOTHING;
        END LOOP;
    END IF;

    -- Reporter gets VIEW/ADD/EDIT on articles and media
    IF r_reporter IS NOT NULL THEN
        FOR ma IN
            SELECT ma2.id FROM menu_actions ma2
            JOIN menus m ON m.id = ma2.menu_id
            WHERE (m.name = 'articles' AND ma2.action IN ('VIEW', 'ADD', 'EDIT'))
               OR (m.name = 'media_library' AND ma2.action IN ('VIEW', 'ADD'))
               OR (m.name IN ('categories', 'tags', 'dashboard') AND ma2.action = 'VIEW')
        LOOP
            INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
            VALUES (r_reporter, ma.id, 1)
            ON CONFLICT DO NOTHING;
        END LOOP;
    END IF;
END $$;

-- ──────────────────────────────────────────────
-- 6. HIERARCHICAL TAXONOMY (Topics & Geography)
-- ──────────────────────────────────────────────
ALTER TABLE categories ADD COLUMN IF NOT EXISTS parent_id INT;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS level INT DEFAULT 1;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS icon VARCHAR(50) DEFAULT '';
ALTER TABLE categories ADD COLUMN IF NOT EXISTS path TEXT DEFAULT '';

INSERT INTO categories (name, slug, level, icon, path, tenant_id) VALUES
    ('National Wire',        'national',     1, 'IconFlame',             'National Wire',        1),
    ('Politics & Governance','politics',     1, 'IconBuildingCommunity', 'Politics & Governance',1),
    ('Business & Economy',   'business',     1, 'IconChartBar',          'Business & Economy',   1),
    ('Tech & Science',       'technology',   1, 'IconCpu',               'Tech & Science',       1),
    ('Sports Arena',         'sports',       1, 'IconBallFootball',      'Sports Arena',         1),
    ('Entertainment & OTT',  'entertainment',1, 'IconMovie',             'Entertainment & OTT',  1),
    ('Crime & Legal',        'crime',        1, 'IconShieldAlert',       'Crime & Legal',        1),
    ('Lifestyle & Health',   'lifestyle',    1, 'IconHeartPulse',        'Lifestyle & Health',   1),
    ('States & Cities',      'states',       1, 'IconMapPin',            'States & Cities',      1)
ON CONFLICT (tenant_id, slug) DO UPDATE 
SET level = EXCLUDED.level, icon = EXCLUDED.icon, path = EXCLUDED.path;

-- ──────────────────────────────────────────────
-- 7. DEFAULT SUPERADMIN USER
-- ──────────────────────────────────────────────
INSERT INTO users (id, email, password_hash, display_name, is_staff, is_super_admin)
VALUES (
    1,
    'superadmin@newsplatform.in',
    '$2a$12$eA8j3g9Q7c1t8XG2.K3vNuqgH5oM6v1m7rQ4j8t1zX5q3K7n2o6yW',
    'Platform Chief Editor',
    TRUE,
    TRUE
)
ON CONFLICT (id) DO UPDATE 
SET is_staff = TRUE, is_super_admin = TRUE, email = EXCLUDED.email;

-- Map user to role
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_roles') THEN
        INSERT INTO user_roles (user_id, role_id) VALUES (1, 1) ON CONFLICT DO NOTHING;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_tenant_mappings') THEN
        INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id) VALUES (1, 1, 1) ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- Reset users_id_seq
SELECT setval('users_id_seq', (SELECT GREATEST(MAX(id), 1) FROM users));
