-- 001_init_production_schema.up.sql
-- Complete Production Database Schema & Essential Production Seeds

-- Clean reset of all tables
DROP TABLE IF EXISTS article_categories, article_tags, live_blog_entries, comments, articles, tags, categories, media, role_menu_actions, menu_actions, user_roles, user_session_contexts, user_category_scopes, user_permission_overrides, menus, roles, users, refresh_tokens, otp_requests, password_reset_tokens, user_activity_logs, web_stories, polls, poll_votes, newsletter_subscriptions, push_subscriptions, feedbacks, site_settings, activity_logs, consent_records CASCADE;

-- ──────────────────────────────────────────────
-- 1. EXTENSIONS & UUID
-- ──────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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
    avatar_url TEXT,
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
-- 3. IAM RBAC PERMISSIONS & ROLES
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS menus (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    label VARCHAR(150) NOT NULL,
    path VARCHAR(255) DEFAULT '',
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS menu_actions (
    id SERIAL PRIMARY KEY,
    menu_id INT REFERENCES menus(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    UNIQUE(menu_id, action)
);

CREATE TABLE IF NOT EXISTS role_menu_actions (
    id SERIAL PRIMARY KEY,
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    menu_action_id INT REFERENCES menu_actions(id) ON DELETE CASCADE,
    UNIQUE(role_id, menu_action_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE,
    PRIMARY KEY(user_id, role_id)
);

CREATE TABLE IF NOT EXISTS user_session_contexts (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    active_district_id INT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ──────────────────────────────────────────────
-- 4. TAXONOMY & CATEGORIES
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    parent_id INT REFERENCES categories(id) ON DELETE CASCADE,
    level INT DEFAULT 1,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(150) UNIQUE NOT NULL,
    path TEXT DEFAULT '',
    icon VARCHAR(50) DEFAULT '',
    sort_order INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_level ON categories(level);
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

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

-- ──────────────────────────────────────────────
-- 5. ARTICLES & EDITORIAL CONTENT
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
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_articles_slug ON articles(slug);
CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
CREATE INDEX IF NOT EXISTS idx_articles_category ON articles(primary_category_id);

CREATE TABLE IF NOT EXISTS article_categories (
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY(article_id, category_id)
);

CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS article_tags (
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY(article_id, tag_id)
);

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

-- ──────────────────────────────────────────────
-- 6. MEDIA LIBRARY & INTERACTIVE FEATURES
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
    site_name VARCHAR(255) DEFAULT 'Naxatra News',
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

-- ──────────────────────────────────────────────
-- 7. INITIAL PRODUCTION ESSENTIAL SEEDS
-- ──────────────────────────────────────────────

-- 7.1 Single-word label Menus
INSERT INTO menus (name, label, path, sort_order, is_active) VALUES
    ('dashboard',     'Dashboard',  '/panel/dashboard', 1, TRUE),
    ('articles',      'Articles',   '/panel/articles',  2, TRUE),
    ('categories',    'Categories', '/panel/categories',3, TRUE),
    ('tags',          'Tags',       '/panel/tags',      4, TRUE),
    ('media_library', 'Media',      '/panel/media',     5, TRUE),
    ('live_blogs',    'Liveblogs',  '/panel/liveblog',  6, TRUE),
    ('comments',      'Comments',   '/panel/comments',  7, TRUE),
    ('roles',         'Roles',      '/panel/roles',     8, TRUE),
    ('users',         'Users',      '/panel/users',     9, TRUE),
    ('settings',      'Settings',   '/panel/settings', 10, TRUE)
ON CONFLICT (name) DO UPDATE SET label = EXCLUDED.label, path = EXCLUDED.path;

-- 7.2 Menu Actions Generation
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

-- 7.3 System Roles
INSERT INTO roles (name, description, is_system) VALUES
    ('super_admin', 'Full platform-wide root authority', TRUE),
    ('editor',      'Editorial publishing and review authority', TRUE),
    ('sub_editor',  'Content review and editing desk', TRUE),
    ('reporter',    'Field journalism and draft creation', TRUE),
    ('moderator',   'Community and comment moderation authority', TRUE)
ON CONFLICT (name) DO NOTHING;

-- 7.4 Role Permission Grants
DO $$
DECLARE
    r_super INT;
    r_editor INT;
    ma RECORD;
BEGIN
    SELECT id INTO r_super FROM roles WHERE name = 'super_admin';
    SELECT id INTO r_editor FROM roles WHERE name = 'editor';

    IF r_super IS NOT NULL THEN
        FOR ma IN SELECT id FROM menu_actions LOOP
            INSERT INTO role_menu_actions (role_id, menu_action_id) VALUES (r_super, ma.id) ON CONFLICT DO NOTHING;
        END LOOP;
    END IF;

    IF r_editor IS NOT NULL THEN
        FOR ma IN SELECT id FROM menu_actions LOOP
            INSERT INTO role_menu_actions (role_id, menu_action_id) VALUES (r_editor, ma.id) ON CONFLICT DO NOTHING;
        END LOOP;
    END IF;
END $$;

-- 7.5 Production Superadmin Account (admin123)
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

-- 7.6 Pure Hindi Category Desks & Regional Sub-Desks
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

-- 7.7 Production Essential Sample Hindi Articles Seed
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

