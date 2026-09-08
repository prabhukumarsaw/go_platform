-- 001_init_production_schema.up.sql
-- Unified Clean Production Database Schema & Essential Seeds (Single-Tenant Architecture)
-- Non-Tenant, High-Performance Newsroom & Reader Platform

-- ──────────────────────────────────────────────
-- 0. CLEAN RESET (Idempotent / Fresh Setup)
-- ──────────────────────────────────────────────
DROP TABLE IF EXISTS
    article_categories, article_tags, article_versions, live_blog_entries,
    comments, articles, tags, categories, media,
    role_menu_actions, menu_actions, user_roles, user_category_scopes,
    user_permission_overrides, permission_audit_log, abac_policies,
    menus, roles, employees, users, refresh_tokens, otp_requests,
    password_reset_tokens, user_activity_logs, web_stories, polls, poll_votes,
    newsletter_subscriptions, push_subscriptions, feedbacks, site_settings,
    activity_logs, consent_records, ad_slots, sponsored_articles CASCADE;

-- ──────────────────────────────────────────────
-- 1. EXTENSIONS
-- ──────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ──────────────────────────────────────────────
-- 2. USERS & AUTHENTICATION
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(50),
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    display_name VARCHAR(200),
    avatar_url TEXT DEFAULT '',
    provider VARCHAR(50) DEFAULT 'local',
    totp_enabled BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    is_staff BOOLEAN DEFAULT TRUE,
    is_super_admin BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS otp_requests (
    id SERIAL PRIMARY KEY,
    phone VARCHAR(50) NOT NULL,
    otp_hash VARCHAR(255) NOT NULL,
    attempts INT DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    is_used BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_activity_logs (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    event VARCHAR(100),
    ip_address VARCHAR(50),
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ──────────────────────────────────────────────
-- 3. EMPLOYEES & STAFF DIRECTORY
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS employees (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employee_code VARCHAR(100) UNIQUE NOT NULL,
    department VARCHAR(100) NOT NULL DEFAULT 'Editorial',
    designation VARCHAR(150) NOT NULL DEFAULT 'Special Correspondent',
    district_id INT,
    address TEXT DEFAULT '',
    pin_code VARCHAR(20) DEFAULT '',
    bio TEXT DEFAULT '',
    press_card_no VARCHAR(100) DEFAULT '',
    x_handle VARCHAR(100) DEFAULT '',
    facebook VARCHAR(200) DEFAULT '',
    instagram VARCHAR(200) DEFAULT '',
    youtube VARCHAR(200) DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ──────────────────────────────────────────────
-- 4. IAM RBAC & ABAC GOVERNANCE
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT TRUE,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS menus (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    label VARCHAR(150) NOT NULL,
    icon VARCHAR(100) DEFAULT '',
    path VARCHAR(255) DEFAULT '',
    parent_id INT REFERENCES menus(id) ON DELETE SET NULL,
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS menu_actions (
    id SERIAL PRIMARY KEY,
    menu_id INT REFERENCES menus(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    label VARCHAR(150) DEFAULT '',
    UNIQUE(menu_id, action)
);

CREATE TABLE IF NOT EXISTS role_menu_actions (
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    menu_action_id INT REFERENCES menu_actions(id) ON DELETE CASCADE,
    PRIMARY KEY(role_id, menu_action_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY(user_id, role_id)
);

-- ──────────────────────────────────────────────
-- 5. TAXONOMY (Categories & Tags)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    parent_id INT REFERENCES categories(id) ON DELETE SET NULL,
    level INT DEFAULT 1,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(150) UNIQUE NOT NULL,
    path TEXT,
    icon VARCHAR(50),
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    meta_title VARCHAR(255),
    meta_description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tags_slug ON tags(slug);

CREATE TABLE IF NOT EXISTS user_category_scopes (
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(id) ON DELETE CASCADE,
    assigned_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY(user_id, category_id)
);

CREATE TABLE IF NOT EXISTS user_permission_overrides (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    menu_action_id INT REFERENCES menu_actions(id) ON DELETE CASCADE,
    effect VARCHAR(10) DEFAULT 'GRANT',
    is_active BOOLEAN DEFAULT TRUE,
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permission_audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    menu_action_id INT,
    action_name VARCHAR(80),
    decision VARCHAR(20) NOT NULL,
    reason TEXT,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS abac_policies (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attribute VARCHAR(30) NOT NULL,
    value JSONB NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ──────────────────────────────────────────────
-- 6. ARTICLES & EDITORIAL CONTENT
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    story_id VARCHAR(255),
    district_id INT,
    language VARCHAR(10) DEFAULT 'hi',
    title TEXT NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    body TEXT,
    excerpt TEXT,
    summary TEXT,
    content TEXT,
    status VARCHAR(50) DEFAULT 'published',
    author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    editor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewer_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    is_breaking BOOLEAN DEFAULT FALSE,
    is_featured BOOLEAN DEFAULT FALSE,
    is_national BOOLEAN DEFAULT FALSE,
    published_at TIMESTAMPTZ DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ,
    meta_title TEXT DEFAULT '',
    meta_description TEXT DEFAULT '',
    og_image TEXT DEFAULT '',
    featured_image TEXT DEFAULT '',
    featured_image_url TEXT DEFAULT '',
    caption TEXT DEFAULT '',
    primary_category_id INT REFERENCES categories(id) ON DELETE SET NULL,
    view_count INT DEFAULT 0,
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', COALESCE(title, '') || ' ' || COALESCE(excerpt, '') || ' ' || COALESCE(summary, ''))
    ) STORED,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Essential Performance Indexes
CREATE INDEX IF NOT EXISTS idx_articles_slug ON articles(slug);
CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
CREATE INDEX IF NOT EXISTS idx_articles_category ON articles(primary_category_id);
CREATE INDEX IF NOT EXISTS idx_articles_search_vector ON articles USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_articles_title_trgm ON articles USING GIN (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_articles_published_feed ON articles (published_at DESC) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_articles_category_feed ON articles (primary_category_id, published_at DESC) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_articles_breaking_feed ON articles (published_at DESC) WHERE status = 'published' AND is_breaking = TRUE;
CREATE INDEX IF NOT EXISTS idx_articles_featured_feed ON articles (published_at DESC) WHERE status = 'published' AND is_featured = TRUE;

CREATE TABLE IF NOT EXISTS article_categories (
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY(article_id, category_id)
);

CREATE TABLE IF NOT EXISTS article_tags (
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY(article_id, tag_id)
);

CREATE TABLE IF NOT EXISTS article_versions (
    id BIGSERIAL PRIMARY KEY,
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    edited_by BIGINT NOT NULL REFERENCES users(id),
    diff JSONB NOT NULL DEFAULT '{}',
    snapshot JSONB,
    version_num INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_article_versions_article_id ON article_versions(article_id);
CREATE INDEX IF NOT EXISTS idx_article_versions_created_at ON article_versions(article_id, created_at DESC);

CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    parent_id INT REFERENCES comments(id) ON DELETE CASCADE,
    author_name VARCHAR(150),
    body TEXT,
    content TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'approved',
    is_approved BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comments_article_id ON comments(article_id);
CREATE INDEX IF NOT EXISTS idx_comments_status ON comments(status);

CREATE TABLE IF NOT EXISTS live_blog_entries (
    id SERIAL PRIMARY KEY,
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    headline VARCHAR(255) DEFAULT '',
    title VARCHAR(255) DEFAULT '',
    body JSONB DEFAULT '""'::jsonb,
    content TEXT DEFAULT '',
    author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_breaking BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_live_blog_article ON live_blog_entries(article_id, created_at DESC);

-- ──────────────────────────────────────────────
-- 7. MEDIA LIBRARY
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS media (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    uploader_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) DEFAULT '',
    mime_type VARCHAR(100) DEFAULT '',
    category VARCHAR(50) DEFAULT 'news',
    folder VARCHAR(100) DEFAULT 'general',
    file_size BIGINT DEFAULT 0,
    storage_path TEXT DEFAULT '',
    url TEXT DEFAULT '',
    alt_text TEXT DEFAULT '',
    caption TEXT DEFAULT '',
    width INT DEFAULT 0,
    height INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_media_category ON media(category);
CREATE INDEX IF NOT EXISTS idx_media_folder ON media(folder);
CREATE INDEX IF NOT EXISTS idx_media_created_at ON media(created_at DESC);

-- ──────────────────────────────────────────────
-- 8. AUDIENCE, NOTIFICATIONS & ENGAGEMENT
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS web_stories (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    language VARCHAR(10) DEFAULT 'hi',
    cover_image TEXT,
    slides JSONB,
    author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'published',
    view_count INT DEFAULT 0,
    published_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS polls (
    id SERIAL PRIMARY KEY,
    question TEXT NOT NULL,
    language VARCHAR(10) DEFAULT 'hi',
    options JSONB,
    total_votes INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS poll_votes (
    id SERIAL PRIMARY KEY,
    poll_id INT REFERENCES polls(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    option_id VARCHAR(50),
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS newsletter_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    frequency VARCHAR(50) DEFAULT 'daily',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id SERIAL PRIMARY KEY,
    district_id INT,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    endpoint TEXT UNIQUE NOT NULL,
    p256dh_key TEXT,
    auth_key TEXT,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS feedbacks (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(150),
    email VARCHAR(255),
    category VARCHAR(50) DEFAULT 'general',
    article_id UUID REFERENCES articles(id) ON DELETE SET NULL,
    message TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'open',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS site_settings (
    id SERIAL PRIMARY KEY,
    site_name VARCHAR(255) DEFAULT 'NewsRoom',
    logo_url TEXT,
    contact_email VARCHAR(255),
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS activity_logs (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100),
    details TEXT,
    ip_address VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS consent_records (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    consent_type VARCHAR(100),
    is_granted BOOLEAN DEFAULT TRUE,
    policy_version VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ad_slots (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slot_type VARCHAR(30) NOT NULL,
    ad_unit_id VARCHAR(200),
    config JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sponsored_articles (
    article_id UUID PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
    sponsor VARCHAR(200) NOT NULL,
    campaign_id VARCHAR(100),
    start_date DATE,
    end_date DATE
);

-- ──────────────────────────────────────────────
-- 9. PRODUCTION ESSENTIAL SEEDS
-- ──────────────────────────────────────────────

-- 9.1 Active Synchronized Navigation Menus (Exact 1-to-1 Match with Panel Routes)
INSERT INTO menus (name, label, icon, path, sort_order, is_active) VALUES
    ('dashboard',     'Dashboard',                 'LayoutDashboard', '/panel/dashboard',     1,  TRUE),
    ('articles',      'Articles',                  'FileText',        '/panel/articles',      2,  TRUE),
    ('categories',    'Categories',                'Tag',             '/panel/categories',    3,  TRUE),
    ('homepage',      'Home Categories & Layout',  'LayoutGrid',      '/panel/homepage',      4,  TRUE),
    ('tags',          'Tags',                      'Hash',            '/panel/tags',          5,  TRUE),
    ('media_library', 'Media',                     'Image',           '/panel/media',         6,  TRUE),
    ('live_blogs',    'Live Blog',                 'Radio',           '/panel/liveblog',      7,  TRUE),
    ('notifications', 'Broadcast & Alerts',        'Bell',            '/panel/notifications', 8,  TRUE),
    ('reports',       'Reports',                   'BarChart2',       '/panel/reports',       9,  TRUE),
    ('users',         'Team & Staff',              'Users',           '/panel/users',         10, TRUE),
    ('comments',      'Comments',                  'MessageSquare',   '/panel/comments',      11, TRUE),
    ('analytics',     'Analytics',                 'TrendingUp',      '/panel/analytics',     12, TRUE),
    ('roles',         'Roles & Permissions',       'Lock',            '/panel/roles',         13, TRUE),
    ('settings',      'Settings',                  'Settings',        '/panel/settings',      14, TRUE),
    ('audit',         'Audit Log',                 'Shield',          '/panel/audit',         15, TRUE),
    ('seo',           'SEO Management',            'Search',          '/panel/seo',           16, TRUE)
ON CONFLICT (name) DO UPDATE SET
    label      = EXCLUDED.label,
    icon       = EXCLUDED.icon,
    path       = EXCLUDED.path,
    sort_order = EXCLUDED.sort_order,
    is_active  = EXCLUDED.is_active;

-- 9.2 Menu Actions Generation
DO $$
DECLARE
    m RECORD;
    actions TEXT[] := ARRAY['VIEW', 'ADD', 'EDIT', 'DELETE', 'PUBLISH', 'APPROVE'];
    act TEXT;
BEGIN
    FOR m IN SELECT id FROM menus LOOP
        FOREACH act IN ARRAY actions LOOP
            INSERT INTO menu_actions (menu_id, action) VALUES (m.id, act) ON CONFLICT DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- 9.3 System Roles
INSERT INTO roles (name, description, is_system, is_active) VALUES
    ('super_admin', 'Full platform-wide root authority',           TRUE, TRUE),
    ('editor',      'Editorial publishing and review authority',   TRUE, TRUE),
    ('sub_editor',  'Content review and editing desk',             TRUE, TRUE),
    ('reporter',    'Field journalism and draft creation',         TRUE, TRUE),
    ('moderator',   'Community and comment moderation authority',  TRUE, TRUE)
ON CONFLICT (name) DO UPDATE SET is_active = TRUE;

-- 9.4 Role Permission Grants
-- super_admin: ALL permissions
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT r.id, ma.id FROM roles r CROSS JOIN menu_actions ma WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- editor: VIEW, ADD, EDIT, PUBLISH on all menus
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT r.id, ma.id FROM roles r
JOIN menu_actions ma ON ma.action IN ('VIEW','ADD','EDIT','PUBLISH')
WHERE r.name = 'editor'
ON CONFLICT DO NOTHING;

-- reporter: VIEW, ADD, EDIT
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT r.id, ma.id FROM roles r
JOIN menu_actions ma ON ma.action IN ('VIEW','ADD','EDIT')
WHERE r.name = 'reporter'
ON CONFLICT DO NOTHING;

-- moderator: VIEW, APPROVE, DELETE on comments and audit
INSERT INTO role_menu_actions (role_id, menu_action_id)
SELECT r.id, ma.id FROM roles r
JOIN menu_actions ma ON ma.action IN ('VIEW','APPROVE','DELETE')
JOIN menus m ON m.id = ma.menu_id AND m.name IN ('comments', 'audit')
WHERE r.name = 'moderator'
ON CONFLICT DO NOTHING;

-- 9.5 Production Superadmin Account (admin123)
INSERT INTO users (email, password_hash, first_name, last_name, display_name, is_staff, is_super_admin, is_active)
VALUES (
    'superadmin@newsplatform.in',
    '$2a$10$ivr/9lvg0ETgT1IdzgL5x.1.FEot7ndzMp/7a4A6l46h4fXx8admW',
    'Super',
    'Admin',
    'Platform Chief Editor',
    TRUE,
    TRUE,
    TRUE
)
ON CONFLICT (email) DO UPDATE SET
    password_hash = '$2a$10$ivr/9lvg0ETgT1IdzgL5x.1.FEot7ndzMp/7a4A6l46h4fXx8admW',
    is_active = TRUE;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r WHERE u.email = 'superadmin@newsplatform.in' AND r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- 9.6 Superadmin Employee Profile
INSERT INTO employees (user_id, employee_code, department, designation, bio, press_card_no, is_active)
SELECT u.id, 'EMP-0001', 'Editorial', 'Chief Editor', 'Platform Chief Editor & Root Administrator', 'PRESS-CHIEF-01', TRUE
FROM users u WHERE u.email = 'superadmin@newsplatform.in'
ON CONFLICT (user_id) DO NOTHING;

-- 9.7 Pure Hindi Category Desks & Regional Sub-Desks
INSERT INTO categories (parent_id, level, name, slug, path, sort_order) VALUES
    (NULL, 1, 'दुनिया', 'world', 'दुनिया', 1),
    (NULL, 1, 'भारत', 'national', 'भारत', 2),
    (NULL, 1, 'झारखंड', 'jharkhand', 'झारखंड', 3),
    (NULL, 1, 'बिहार', 'bihar', 'बिहार', 4),
    (NULL, 1, 'मनोरंजन', 'entertainment', 'मनोरंजन', 5),
    (NULL, 1, 'खेल', 'sports', 'खेल', 6),
    (NULL, 1, 'टेक्नोलॉजी', 'technology', 'टेक्नोलॉजी', 7),
    (NULL, 1, 'राशिफल', 'astrology', 'राशिफल', 8),
    (NULL, 1, 'शिक्षा', 'education', 'शिक्षा', 9),
    (NULL, 1, 'करियर', 'career', 'करियर', 10),
    (NULL, 1, 'राजनीति', 'politics', 'राजनीति', 11),
    (NULL, 1, 'धर्म', 'religion', 'धर्म', 12),
    (NULL, 1, 'लाइफस्टाइल', 'lifestyle', 'लाइफस्टाइल', 13),
    (NULL, 1, 'ऑटोमोबाइल्स', 'automobiles', 'ऑटोमोबाइल्स', 14)
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, parent_id = EXCLUDED.parent_id, level = EXCLUDED.level, path = EXCLUDED.path, sort_order = EXCLUDED.sort_order;

SELECT setval('categories_id_seq', (SELECT MAX(id) FROM categories));

-- Regional Sub-Desks for भारत (National)
INSERT INTO categories (parent_id, level, name, slug, path, sort_order)
SELECT c.id, 2, val.name, val.slug, val.path, val.sort_order
FROM categories c,
(VALUES
    ('दिल्ली', 'delhi', 'भारत > दिल्ली', 1),
    ('उत्तराखंड', 'uttarakhand', 'भारत > उत्तराखंड', 2),
    ('उत्तर प्रदेश', 'uttar-pradesh', 'भारत > उत्तर प्रदेश', 3),
    ('मध्य प्रदेश', 'madhya-pradesh', 'भारत > मध्य प्रदेश', 4),
    ('छत्तीसगढ़', 'chhattisgarh', 'भारत > छत्तीसगढ़', 5),
    ('हरियाणा', 'haryana', 'भारत > हरियाणा', 6),
    ('कोलकाता', 'kolkata', 'भारत > कोलकाता', 7),
    ('असम', 'assam', 'भारत > असम', 8),
    ('ओडिशा', 'odisha', 'भारत > ओडिशा', 9),
    ('पंजाब', 'punjab', 'भारत > पंजाब', 10),
    ('गुजरात', 'gujarat', 'भारत > गुजरात', 11),
    ('केरल', 'kerala', 'भारत > केरल', 12),
    ('राजस्थान', 'rajasthan', 'भारत > राजस्थान', 13)
) AS val(name, slug, path, sort_order)
WHERE c.slug = 'national'
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, parent_id = EXCLUDED.parent_id, level = EXCLUDED.level, path = EXCLUDED.path, sort_order = EXCLUDED.sort_order;

-- District Sub-Desks for झारखंड (Jharkhand)
INSERT INTO categories (parent_id, level, name, slug, path, sort_order)
SELECT c.id, 2, val.name, val.slug, val.path, val.sort_order
FROM categories c,
(VALUES
    ('रांची', 'ranchi', 'झारखंड > रांची', 1),
    ('जमशेदपुर', 'jamshedpur', 'झारखंड > जमशेदपुर', 2),
    ('धनबाद', 'dhanbad', 'झारखंड > धनबाद', 3),
    ('बोकारो', 'bokaro', 'झारखंड > बोकारो', 4),
    ('हजारीबाग', 'hazaribagh', 'झारखंड > हजारीबाग', 5),
    ('देवघर', 'deoghar', 'झारखंड > देवघर', 6),
    ('दुमका', 'dumka', 'झारखंड > दुमका', 7),
    ('गिरिडीह', 'giridih', 'झारखंड > गिरिडीह', 8),
    ('रामगढ़', 'ramgarh', 'झारखंड > रामगढ़', 9),
    ('चतरा', 'chatra', 'झारखंड > चतरा', 10),
    ('पलामू', 'palamu', 'झारखंड > पलामू', 11),
    ('गढ़वा', 'garhwa', 'झारखंड > गढ़वा', 12),
    ('लातेहार', 'latehar', 'झारखंड > लातेहार', 13),
    ('कोडरमा', 'koderma', 'झारखंड > कोडरमा', 14),
    ('गोड्डा', 'godda', 'झारखंड > गोड्डा', 15),
    ('साहिबगंज', 'sahibganj', 'झारखंड > साहिबगंज', 16),
    ('पाकुड़', 'pakur', 'झारखंड > पाकुड़', 17),
    ('जामताड़ा', 'jamtara', 'झारखंड > जामताड़ा', 18),
    ('खूंटी', 'khunti', 'झारखंड > खूंटी', 19),
    ('गुमला', 'gumla', 'झारखंड > गुमला', 20),
    ('सिमडेगा', 'simdega', 'झारखंड > सिमडेगा', 21),
    ('पश्चिम सिंहभूम', 'west-singhbhum', 'झारखंड > पश्चिम सिंहभूम', 22),
    ('सरायकेला खरसावां', 'seraikela-kharsawan', 'झारखंड > सरायकेला खरसावां', 23)
) AS val(name, slug, path, sort_order)
WHERE c.slug = 'jharkhand'
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, parent_id = EXCLUDED.parent_id, level = EXCLUDED.level, path = EXCLUDED.path, sort_order = EXCLUDED.sort_order;

-- District Sub-Desks for बिहार (Bihar)
INSERT INTO categories (parent_id, level, name, slug, path, sort_order)
SELECT c.id, 2, val.name, val.slug, val.path, val.sort_order
FROM categories c,
(VALUES
    ('मोतीहारी', 'motihari', 'बिहार > मोतीहारी', 1),
    ('सहरसा', 'saharsa', 'बिहार > सहरसा', 2),
    ('समस्तीपुर', 'samastipur', 'बिहार > समस्तीपुर', 3),
    ('सारण', 'saran', 'बिहार > सारण', 4),
    ('शेखपुरा', 'sheikhpura', 'बिहार > शेखपुरा', 5),
    ('शिवहर', 'sheohar', 'बिहार > शिवहर', 6),
    ('सीतामढ़ी', 'sitamarhi', 'बिहार > सीतामढ़ी', 7),
    ('सीवान', 'siwan', 'बिहार > सीवान', 8),
    ('सुपौल', 'supaul', 'बिहार > सुपौल', 9),
    ('वैशाली', 'vaishali', 'बिहार > वैशाली', 10),
    ('पश्चिमी चंपारण', 'west-champaran', 'बिहार > पश्चिमी चंपारण', 11),
    ('मधुबनी', 'madhubani', 'बिहार > मधुबनी', 12),
    ('नवादा', 'nawada', 'बिहार > नवादा', 13),
    ('पटना', 'patna', 'बिहार > पटना', 14),
    ('कटिहार', 'katihar', 'बिहार > कटिहार', 15),
    ('अरवल', 'arwal', 'बिहार > अरवल', 16),
    ('अररिया', 'araria', 'बिहार > अररिया', 17),
    ('औरंगाबाद', 'aurangabad', 'बिहार > औरंगाबाद', 18),
    ('बांका', 'banka', 'बिहार > बांका', 19),
    ('बेगूसराय', 'begusarai', 'बिहार > बेगूसराय', 20),
    ('भागलपुर', 'bhagalpur', 'बिहार > भागलपुर', 21),
    ('दरभंगा', 'darbhanga', 'बिहार > दरभंगा', 22),
    ('पूर्वी चंपारण', 'east-champaran', 'बिहार > पूर्वी चंपारण', 23),
    ('गया जी', 'gaya', 'बिहार > गया जी', 24),
    ('गोपालगंज', 'gopalganj', 'बिहार > गोपालगंज', 25),
    ('जमुई', 'jamui', 'बिहार > जमुई', 26),
    ('जहानाबाद', 'jehanabad', 'बिहार > जहानाबाद', 27),
    ('खगड़िया', 'khagaria', 'बिहार > खगड़िया', 28),
    ('किसनगंज', 'kishanganj', 'बिहार > किसनगंज', 29),
    ('लखीसराय', 'lakhisarai', 'बिहार > लखीसराय', 30),
    ('मुंगेर', 'munger', 'बिहार > मुंगेर', 31),
    ('मुजफ्फरपुर', 'muzaffarpur', 'बिहार > मुजफ्फरपुर', 32),
    ('पूर्णिया', 'purnia', 'बिहार > पूर्णिया', 33)
) AS val(name, slug, path, sort_order)
WHERE c.slug = 'bihar'
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, parent_id = EXCLUDED.parent_id, level = EXCLUDED.level, path = EXCLUDED.path, sort_order = EXCLUDED.sort_order;

SELECT setval('categories_id_seq', (SELECT GREATEST(MAX(id), 200) FROM categories));

-- 9.8 Production Essential Sample Hindi Articles Seed
INSERT INTO articles (title, slug, summary, excerpt, content, body, featured_image, featured_image_url, author_id, primary_category_id, status, language, view_count, published_at, is_breaking, is_featured)
SELECT
    val.title, val.slug, val.summary, val.summary, val.content, val.content, val.featured_image_url, val.featured_image_url,
    u.id, c.id, 'published', 'hi', val.view_count, NOW(), val.is_breaking, val.is_featured
FROM (VALUES
    ('संसद के मानसून सत्र में नए नीतिगत विधेयक पेश, विकास पहलों पर केंद्रित रहेगी चर्चा', 'parliament-monsoon-session-bills-development-focus', 'संसद सत्र में डिजिटल बुनियादी ढांचे और शिक्षा सुधार से जुड़े विधेयकों को पेश किया जा रहा है।', 'नई दिल्ली: संसद के मानसून सत्र की शुरुआत के साथ ही राष्ट्रीय बुनियादी ढांचे, डिजिटल कनेक्टिविटी और नई शिक्षा नीति के क्रियान्वयन से जुड़े प्रमुख विधेयकों को पटल पर रखा गया। सदन के विभिन्न सत्रों में विकास प्राथमिकताओं पर व्यापक बहस जारी है।', 'https://images.unsplash.com/photo-1541872703-74c5e44368f9?w=1200', 'national', 1250, TRUE, TRUE),
    ('ग्लोबल टेक समिट 2026: आर्टिफिशियल इंटेलिजेंस व साइबर सुरक्षा पर दिग्गजों की बड़ी घोषणाएं', 'global-tech-summit-2026-ai-innovation-cyber-security', 'तकनीकी जगत के प्रमुख दिग्गजों ने एआई प्राइवेसी और रिस्पॉन्सिबल टेक मॉडल्स की रूपरेखा प्रस्तुत की।', 'बेंगलुरु: अंतर्राष्ट्रीय टेक शिखर सम्मेलन 2026 में डेटा सुरक्षा, एआई गवर्नेंस और नेक्स्ट-जनरेशन क्लाउड इंफ्रास्ट्रक्चर पर गहन विचार-विमर्श किया गया। विशेषज्ञों ने भारत को वैश्विक सॉफ्टवेयर नवाचार का प्रमुख केंद्र बताया।', 'https://images.unsplash.com/photo-1518770660439-4636190af475?w=1200', 'technology', 1020, FALSE, TRUE),
    ('वर्ल्ड क्रिकेट शृंखला: भारतीय टीम की शानदार जीत, गेंदबाजों ने किया बेहतरीन प्रदर्शन', 'world-cricket-series-india-victory-bowling-performance', 'अंतर्राष्ट्रीय क्रिकेट मुकाबले में भारतीय गेंदबाजों की धारदार गेंदबाजी से टीम को मिली दमदार जीत।', 'मुंबई: क्रिकेट वर्ल्ड सीरीज के पहले मैच में भारतीय टीम ने ऑलराउंड खेल का प्रदर्शन करते हुए बड़ी जीत हासिल की। सलामी बल्लेबाजों की आक्रामक शुरुआत के बाद गेंदबाजों ने पूरी विपक्षी टीम को कम स्कोर पर समेट दिया।', 'https://images.unsplash.com/photo-1531415074968-036ba1b575da?w=1200', 'sports', 2340, TRUE, FALSE),
    ('ऑटो एक्सपो 2026: नई इलेक्ट्रिक कारों व स्मार्ट मोबिलिटी फीचर्स की बाजार में धूम', 'auto-expo-2026-ev-smart-cars-launch', 'प्रमुख वाहन कंपनियों ने 600 किमी रेंज वाली नई ईवी और स्मार्ट कारों को लॉन्च किया।', 'ग्रेटर नोएडा: ऑटो एक्सपो 2026 में अगली पीढ़ी की स्मार्ट कारों और हरित ऊर्जा से चलने वाले वाहनों ने ध्यान आकर्षित किया। नए बैटरी पैक और सेफ्टी फीचर्स के साथ इलेक्ट्रिक मोबिलिटी सेगमेंट में तीव्र वृद्धि देखी जा रही है।', 'https://images.unsplash.com/photo-1503376780353-7e6692767b70?w=1200', 'automobiles', 1580, FALSE, FALSE),
    ('रांची: झारखंड में नए औद्योगिक विकास प्रोजेक्ट्स को मिली मंजूरी, युवाओं के लिए रोजगार के अवसर', 'ranchi-jharkhand-industrial-projects-approved-employment', 'झारखंड सरकार ने राज्य में आधारभूत संरचना विकास और औद्योगिक निवेश को बढ़ावा देने हेतु नए प्रोजेक्ट्स को हरी झंडी दिखाई।', 'रांची: झारखंड मंत्रिमंडल की बैठक में राज्य में औद्योगिक गलियारे के निर्माण और कौशल विकास केंद्रों की स्थापना से जुड़े बड़े फैसलों पर मुहर लगाई गई। मुख्यमंत्री ने कहा कि इससे स्थानीय युवाओं को रोजगार के व्यापक अवसर प्राप्त होंगे।', 'https://images.unsplash.com/photo-1541872703-74c5e44368f9?w=1200', 'jharkhand', 1420, TRUE, TRUE),
    ('पटना: बिहार में स्मार्ट सिटी और एक्सप्रेसवे बुनियादी ढांचे में तेजी, प्रमुख जिलों का कायाकल्प', 'patna-bihar-smart-city-expressway-infrastructure', 'बिहार के कई प्रमुख शहरों में स्मार्ट सिटी मिशन के तहत नई सड़कों, ड्रेनेज व डिजिटल सुविधाओं का विकास जारी।', 'पटना: बिहार में चल रही ढांचागत विकास योजनाओं के समीक्षा बैठक में स्मार्ट सिटी परियोजनाओं की प्रगति पर संतोष जताया गया। पटना, मुजफ्फरपुर, भागलपुर और बिहारशरीफ में आधुनिक नागरिक सुविधाओं का निर्माण कार्य तेजी से आगे बढ़ रहा है।', 'https://images.unsplash.com/photo-1518770660439-4636190af475?w=1200', 'bihar', 1890, FALSE, TRUE)
) AS val(title, slug, summary, content, featured_image_url, cat_slug, view_count, is_breaking, is_featured)
LEFT JOIN categories c ON c.slug = val.cat_slug
LEFT JOIN users u ON u.email = 'superadmin@newsplatform.in'
ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, summary = EXCLUDED.summary, status = EXCLUDED.status;

-- Link articles to categories
INSERT INTO article_categories (article_id, category_id)
SELECT a.id, a.primary_category_id FROM articles a
ON CONFLICT DO NOTHING;
