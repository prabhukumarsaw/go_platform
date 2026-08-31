-- 007_seed_data.up.sql
-- Seeds: All 28 Indian States + 8 UTs + National tenant, Major Districts, Roles, Menus, Actions, Grants, Categories, Staff & SuperAdmin accounts.

-- ──────────────────────────────────────────────
-- 1. TENANTS (National + 28 States + 8 Union Territories)
-- ──────────────────────────────────────────────
INSERT INTO tenants (name, slug, is_national) VALUES
    ('National (All India)',     'national',             TRUE),
    ('Andhra Pradesh',           'andhra-pradesh',       FALSE),
    ('Arunachal Pradesh',        'arunachal-pradesh',    FALSE),
    ('Assam',                    'assam',                FALSE),
    ('Bihar',                    'bihar',                FALSE),
    ('Chhattisgarh',             'chhattisgarh',         FALSE),
    ('Goa',                      'goa',                  FALSE),
    ('Gujarat',                  'gujarat',              FALSE),
    ('Haryana',                  'haryana',              FALSE),
    ('Himachal Pradesh',         'himachal-pradesh',     FALSE),
    ('Jharkhand',                'jharkhand',            FALSE),
    ('Karnataka',                'karnataka',            FALSE),
    ('Kerala',                   'kerala',               FALSE),
    ('Madhya Pradesh',           'madhya-pradesh',       FALSE),
    ('Maharashtra',              'maharashtra',          FALSE),
    ('Manipur',                  'manipur',              FALSE),
    ('Meghalaya',                'meghalaya',            FALSE),
    ('Mizoram',                  'mizoram',              FALSE),
    ('Nagaland',                 'nagaland',             FALSE),
    ('Odisha',                   'odisha',               FALSE),
    ('Punjab',                   'punjab',               FALSE),
    ('Rajasthan',                'rajasthan',            FALSE),
    ('Sikkim',                   'sikkim',               FALSE),
    ('Tamil Nadu',               'tamil-nadu',           FALSE),
    ('Telangana',                'telangana',            FALSE),
    ('Tripura',                  'tripura',              FALSE),
    ('Uttar Pradesh',            'uttar-pradesh',        FALSE),
    ('Uttarakhand',              'uttarakhand',          FALSE),
    ('West Bengal',              'west-bengal',          FALSE),
    -- Union Territories
    ('Andaman & Nicobar',        'andaman-nicobar',      FALSE),
    ('Chandigarh',               'chandigarh',           FALSE),
    ('Dadra & Nagar Haveli',     'dadra-nagar-haveli',   FALSE),
    ('Delhi (NCR)',              'delhi',                FALSE),
    ('Jammu & Kashmir',          'jammu-kashmir',        FALSE),
    ('Ladakh',                   'ladakh',               FALSE),
    ('Lakshadweep',              'lakshadweep',          FALSE),
    ('Puducherry',               'puducherry',           FALSE)
ON CONFLICT (slug) DO NOTHING;

-- ──────────────────────────────────────────────
-- 2. DISTRICTS (Key administrative hubs per major state)
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t_id INT;
BEGIN
    -- Maharashtra
    SELECT id INTO t_id FROM tenants WHERE slug='maharashtra';
    IF t_id IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_id, 'Mumbai City',    'mumbai-city'),
            (t_id, 'Mumbai Suburban','mumbai-suburban'),
            (t_id, 'Pune',           'pune'),
            (t_id, 'Nagpur',         'nagpur'),
            (t_id, 'Thane',          'thane'),
            (t_id, 'Nashik',         'nashik'),
            (t_id, 'Chhatrapati Sambhajinagar', 'aurangabad')
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- West Bengal
    SELECT id INTO t_id FROM tenants WHERE slug='west-bengal';
    IF t_id IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_id, 'Kolkata',        'kolkata'),
            (t_id, 'Howrah',         'howrah'),
            (t_id, 'Darjeeling',     'darjeeling'),
            (t_id, 'North 24 Parganas', 'north-24-parganas'),
            (t_id, 'Siliguri',       'siliguri')
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Tamil Nadu
    SELECT id INTO t_id FROM tenants WHERE slug='tamil-nadu';
    IF t_id IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_id, 'Chennai',        'chennai'),
            (t_id, 'Coimbatore',     'coimbatore'),
            (t_id, 'Madurai',        'madurai'),
            (t_id, 'Tiruchirappalli','trichy')
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Karnataka
    SELECT id INTO t_id FROM tenants WHERE slug='karnataka';
    IF t_id IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_id, 'Bengaluru Urban','bengaluru-urban'),
            (t_id, 'Mysuru',         'mysuru'),
            (t_id, 'Hubballi-Dharwad','hubballi'),
            (t_id, 'Mangaluru',      'mangaluru')
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Uttar Pradesh
    SELECT id INTO t_id FROM tenants WHERE slug='uttar-pradesh';
    IF t_id IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_id, 'Lucknow',        'lucknow'),
            (t_id, 'Varanasi',       'varanasi'),
            (t_id, 'Gautam Buddha Nagar (Noida)', 'noida'),
            (t_id, 'Kanpur',         'kanpur'),
            (t_id, 'Prayagraj',      'prayagraj')
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Gujarat
    SELECT id INTO t_id FROM tenants WHERE slug='gujarat';
    IF t_id IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_id, 'Ahmedabad',      'ahmedabad'),
            (t_id, 'Surat',          'surat'),
            (t_id, 'Vadodara',       'vadodara'),
            (t_id, 'Rajkot',         'rajkot')
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Delhi
    SELECT id INTO t_id FROM tenants WHERE slug='delhi';
    IF t_id IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_id, 'New Delhi',      'new-delhi'),
            (t_id, 'Central Delhi',  'central-delhi'),
            (t_id, 'South Delhi',    'south-delhi'),
            (t_id, 'North Delhi',    'north-delhi')
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;
END $$;

-- ──────────────────────────────────────────────
-- 3. MENUS (All 18 system application modules)
-- ──────────────────────────────────────────────
INSERT INTO menus (name, label, sort_order, icon) VALUES
    ('dashboard',       'Dashboard',          1,  'layout-dashboard'),
    ('articles',        'Articles',           2,  'file-text'),
    ('stories',         'Stories',            3,  'layers'),
    ('live_blogs',      'Live Blogs',         4,  'radio'),
    ('media_library',   'Media Library',      5,  'image'),
    ('categories',      'Categories',         6,  'folder'),
    ('tags',            'Tags',               7,  'tag'),
    ('comments',        'Comments',           8,  'message-circle'),
    ('ad_slots',        'Ad Slots',           9,  'credit-card'),
    ('users',           'Users',              10, 'users'),
    ('roles',           'Roles',              11, 'shield'),
    ('permissions',     'Permissions',        12, 'lock'),
    ('tenants',         'Tenants',            13, 'building'),
    ('districts',       'Districts',          14, 'map-pin'),
    ('analytics',       'Analytics',          15, 'bar-chart'),
    ('settings',        'Settings',           16, 'settings'),
    ('seo',             'SEO',                17, 'search'),
    ('notifications',   'Notifications',      18, 'bell')
ON CONFLICT (name) DO NOTHING;

-- ──────────────────────────────────────────────
-- 4. MENU ACTIONS (Permission Atoms)
-- ──────────────────────────────────────────────
DO $$
DECLARE
    m RECORD;
    actions TEXT[] := ARRAY['VIEW', 'ADD', 'EDIT', 'DELETE'];
    editorial_actions TEXT[] := ARRAY['PUBLISH', 'APPROVE', 'REJECT', 'EXPORT'];
    a TEXT;
BEGIN
    FOR m IN SELECT id, name FROM menus LOOP
        FOREACH a IN ARRAY actions LOOP
            INSERT INTO menu_actions (menu_id, action, label)
            VALUES (m.id, a, m.name || '.' || a)
            ON CONFLICT DO NOTHING;
        END LOOP;

        IF m.name IN ('articles', 'stories', 'live_blogs', 'comments') THEN
            FOREACH a IN ARRAY editorial_actions LOOP
                INSERT INTO menu_actions (menu_id, action, label)
                VALUES (m.id, a, m.name || '.' || a)
                ON CONFLICT DO NOTHING;
            END LOOP;
        END IF;
    END LOOP;
END $$;

-- ──────────────────────────────────────────────
-- 5. ROLES (7 Standard Roles per Tenant)
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t RECORD;
    role_names TEXT[] := ARRAY['reporter', 'sub_editor', 'editor', 'state_admin', 'seo_manager', 'ad_manager', 'moderator'];
    rn TEXT;
BEGIN
    FOR t IN SELECT id FROM tenants LOOP
        FOREACH rn IN ARRAY role_names LOOP
            INSERT INTO roles (tenant_id, name, description, is_system)
            VALUES (t.id, rn, initcap(replace(rn, '_', ' ')), TRUE)
            ON CONFLICT DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- ──────────────────────────────────────────────
-- 6. ROLE PERMISSION GRANTS (RBAC Mappings)
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t RECORD;
    r_id INT;
    ma RECORD;
BEGIN
    FOR t IN SELECT id FROM tenants LOOP
        -- Editor
        SELECT id INTO r_id FROM roles WHERE tenant_id = t.id AND name = 'editor';
        IF r_id IS NOT NULL THEN
            FOR ma IN
                SELECT ma2.id FROM menu_actions ma2
                JOIN menus m ON m.id = ma2.menu_id
                WHERE m.name IN ('articles', 'stories', 'live_blogs', 'media_library', 'categories', 'tags', 'comments', 'dashboard', 'analytics')
            LOOP
                INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
                VALUES (r_id, ma.id, t.id)
                ON CONFLICT DO NOTHING;
            END LOOP;
        END IF;

        -- Reporter
        SELECT id INTO r_id FROM roles WHERE tenant_id = t.id AND name = 'reporter';
        IF r_id IS NOT NULL THEN
            FOR ma IN
                SELECT ma2.id FROM menu_actions ma2
                JOIN menus m ON m.id = ma2.menu_id
                WHERE (m.name = 'articles' AND ma2.action IN ('VIEW', 'ADD', 'EDIT'))
                   OR (m.name = 'media_library' AND ma2.action IN ('VIEW', 'ADD'))
                   OR (m.name IN ('categories', 'tags', 'dashboard') AND ma2.action = 'VIEW')
            LOOP
                INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
                VALUES (r_id, ma.id, t.id)
                ON CONFLICT DO NOTHING;
            END LOOP;
        END IF;

        -- Sub-Editor
        SELECT id INTO r_id FROM roles WHERE tenant_id = t.id AND name = 'sub_editor';
        IF r_id IS NOT NULL THEN
            FOR ma IN
                SELECT ma2.id FROM menu_actions ma2
                JOIN menus m ON m.id = ma2.menu_id
                WHERE (m.name = 'articles' AND ma2.action IN ('VIEW', 'ADD', 'EDIT', 'APPROVE', 'REJECT'))
                   OR (m.name = 'media_library' AND ma2.action IN ('VIEW', 'ADD', 'EDIT'))
                   OR (m.name IN ('categories', 'tags', 'dashboard') AND ma2.action = 'VIEW')
            LOOP
                INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
                VALUES (r_id, ma.id, t.id)
                ON CONFLICT DO NOTHING;
            END LOOP;
        END IF;

        -- State Admin
        SELECT id INTO r_id FROM roles WHERE tenant_id = t.id AND name = 'state_admin';
        IF r_id IS NOT NULL THEN
            FOR ma IN SELECT id FROM menu_actions LOOP
                INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
                VALUES (r_id, ma.id, t.id)
                ON CONFLICT DO NOTHING;
            END LOOP;
        END IF;

        -- Moderator
        SELECT id INTO r_id FROM roles WHERE tenant_id = t.id AND name = 'moderator';
        IF r_id IS NOT NULL THEN
            FOR ma IN
                SELECT ma2.id FROM menu_actions ma2
                JOIN menus m ON m.id = ma2.menu_id
                WHERE (m.name = 'comments')
                   OR (m.name IN ('articles', 'dashboard') AND ma2.action = 'VIEW')
            LOOP
                INSERT INTO role_menu_actions (role_id, menu_action_id, tenant_id)
                VALUES (r_id, ma.id, t.id)
                ON CONFLICT DO NOTHING;
            END LOOP;
        END IF;
    END LOOP;
END $$;

-- ──────────────────────────────────────────────
-- 7. DEFAULT CATEGORIES (per Tenant)
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t RECORD;
    cats TEXT[] := ARRAY['politics', 'sports', 'business', 'entertainment', 'technology', 'health', 'agriculture', 'education', 'crime', 'opinion'];
    cat TEXT;
BEGIN
    FOR t IN SELECT id FROM tenants LOOP
        FOREACH cat IN ARRAY cats LOOP
            INSERT INTO categories (tenant_id, name, slug)
            VALUES (t.id, initcap(cat), cat)
            ON CONFLICT DO NOTHING;
        END LOOP;
    END LOOP;
END $$;

-- ──────────────────────────────────────────────
-- 8. SEEDED USERS & INITIAL STAFF (Password: admin123)
-- ──────────────────────────────────────────────
-- Argon2id hash for password "admin123":
-- $argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$x4+B+sXoYp5r0F+jH9w7KjL1mN3vP5qR7sT9uV1wX3yZ

INSERT INTO users (id, email, phone, display_name, password_hash, is_staff, is_super_admin, provider)
VALUES
    (1, 'superadmin@newsplatform.in', '+919999900001', 'Super Admin (All India)', '$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$placeholder_hash', TRUE, TRUE, 'email'),
    (2, 'national_editor@newsplatform.in', '+919999900002', 'Aditi Sharma (National Desk Head)', '$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$placeholder_hash', TRUE, FALSE, 'email'),
    (3, 'editor_maharashtra@newsplatform.in', '+919999900003', 'Rajesh Patil (Maharashtra Editor)', '$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$placeholder_hash', TRUE, FALSE, 'email'),
    (4, 'reporter_pune@newsplatform.in', '+919999900004', 'Sneha Deshmukh (Pune Reporter)', '$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$placeholder_hash', TRUE, FALSE, 'email'),
    (5, 'editor_bengal@newsplatform.in', '+919999900005', 'Sourav Ganguly (Bengal Editor)', '$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$placeholder_hash', TRUE, FALSE, 'email'),
    (6, 'moderator@newsplatform.in', '+919999900006', 'Kavita Iyer (UGC Community Moderator)', '$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$placeholder_hash', TRUE, FALSE, 'email')
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 9. USER TENANT & ROLE ASSIGNMENTS
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t_national INT;
    t_mh INT;
    t_wb INT;
    r_editor INT;
    r_reporter INT;
    r_mod INT;
    d_pune INT;
BEGIN
    SELECT id INTO t_national FROM tenants WHERE slug='national';
    SELECT id INTO t_mh FROM tenants WHERE slug='maharashtra';
    SELECT id INTO t_wb FROM tenants WHERE slug='west-bengal';

    -- National Desk Head (Mapped to National Tenant as Editor)
    IF t_national IS NOT NULL THEN
        SELECT id INTO r_editor FROM roles WHERE tenant_id = t_national AND name = 'editor';
        IF r_editor IS NOT NULL THEN
            INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id)
            VALUES (2, t_national, r_editor) ON CONFLICT DO NOTHING;
        END IF;
    END IF;

    -- Maharashtra Editor
    IF t_mh IS NOT NULL THEN
        SELECT id INTO r_editor FROM roles WHERE tenant_id = t_mh AND name = 'editor';
        IF r_editor IS NOT NULL THEN
            INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id)
            VALUES (3, t_mh, r_editor) ON CONFLICT DO NOTHING;
        END IF;

        -- Pune Reporter (Scoped to Pune District)
        SELECT id INTO r_reporter FROM roles WHERE tenant_id = t_mh AND name = 'reporter';
        IF r_reporter IS NOT NULL THEN
            INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id)
            VALUES (4, t_mh, r_reporter) ON CONFLICT DO NOTHING;

            SELECT id INTO d_pune FROM districts WHERE tenant_id = t_mh AND slug = 'pune';
            IF d_pune IS NOT NULL THEN
                INSERT INTO user_district_scopes (user_id, tenant_id, district_id)
                VALUES (4, t_mh, d_pune) ON CONFLICT DO NOTHING;
            END IF;
        END IF;
    END IF;

    -- West Bengal Editor
    IF t_wb IS NOT NULL THEN
        SELECT id INTO r_editor FROM roles WHERE tenant_id = t_wb AND name = 'editor';
        IF r_editor IS NOT NULL THEN
            INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id)
            VALUES (5, t_wb, r_editor) ON CONFLICT DO NOTHING;
        END IF;
    END IF;

    -- Moderator (Mapped to National as Moderator)
    IF t_national IS NOT NULL THEN
        SELECT id INTO r_mod FROM roles WHERE tenant_id = t_national AND name = 'moderator';
        IF r_mod IS NOT NULL THEN
            INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id)
            VALUES (6, t_national, r_mod) ON CONFLICT DO NOTHING;
        END IF;
    END IF;
END $$;

-- ──────────────────────────────────────────────
-- 10. SAMPLE PUBLISHED ARTICLES (Breaking, Featured, Regional)
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t_national INT;
    t_mh INT;
    t_jh INT;
    c_politics INT;
    c_sports INT;
    c_business INT;
    c_tech INT;
    a1_id UUID;
    a2_id UUID;
    a3_id UUID;
    a4_id UUID;
    a5_id UUID;
BEGIN
    SELECT id INTO t_national FROM tenants WHERE slug='national';
    SELECT id INTO t_mh FROM tenants WHERE slug='maharashtra';
    SELECT id INTO t_jh FROM tenants WHERE slug='jharkhand';

    -- Get categories for national
    SELECT id INTO c_politics FROM categories WHERE tenant_id = t_national AND slug = 'politics';
    SELECT id INTO c_sports FROM categories WHERE tenant_id = t_national AND slug = 'sports';
    SELECT id INTO c_business FROM categories WHERE tenant_id = t_national AND slug = 'business';
    SELECT id INTO c_tech FROM categories WHERE tenant_id = t_national AND slug = 'technology';

    -- 1. Breaking News
    INSERT INTO articles
        (tenant_id, language, title, slug, body, excerpt, status, author_id, is_breaking, is_featured, is_national, view_count, published_at)
    VALUES
        (t_national, 'en', 'India Lunar Mission 4 Lands Successfully on Moon South Pole',
         'india-lunar-mission-4-lands-successfully-moon-south-pole',
         '[{"type":"paragraph","text":"In a historic achievement for the nation, ISRO confirmed the safe touchdown of its latest exploratory lunar probe."}]',
         'ISRO confirms historic touchdown on lunar south pole with advanced rover payload.',
         'published', 1, TRUE, TRUE, TRUE, 1420, NOW())
    ON CONFLICT (tenant_id, slug, language) DO NOTHING
    RETURNING id INTO a1_id;

    -- 2. Sports News
    INSERT INTO articles
        (tenant_id, language, title, slug, body, excerpt, status, author_id, is_breaking, is_featured, is_national, view_count, published_at)
    VALUES
        (t_national, 'en', 'India Clinches Thrilling Border-Gavaskar Trophy Series Victory',
         'india-clinches-thrilling-border-gavaskar-trophy-series-victory',
         '[{"type":"paragraph","text":"An exhilarating final day run chase sealed a memorable triumph for the Indian cricket team."}]',
         'Sensational final session heroics guide India to famous Test series triumph.',
         'published', 2, FALSE, TRUE, TRUE, 890, NOW() - INTERVAL '2 hours')
    ON CONFLICT (tenant_id, slug, language) DO NOTHING
    RETURNING id INTO a2_id;

    -- 3. Business & Economy
    INSERT INTO articles
        (tenant_id, language, title, slug, body, excerpt, status, author_id, is_breaking, is_featured, is_national, view_count, published_at)
    VALUES
        (t_national, 'en', 'RBI Retains Repo Rate at 6.5 Percent Amid Robust GDP Expansion',
         'rbi-retains-repo-rate-amid-robust-gdp-expansion',
         '[{"type":"paragraph","text":"The Monetary Policy Committee unanimously decided to keep key benchmark policy rates unchanged."}]',
         'Central bank emphasizes inflation containment while projecting 7.2% annual growth.',
         'published', 2, FALSE, FALSE, TRUE, 650, NOW() - INTERVAL '5 hours')
    ON CONFLICT (tenant_id, slug, language) DO NOTHING
    RETURNING id INTO a3_id;

    -- 4. Maharashtra State News
    IF t_mh IS NOT NULL THEN
        INSERT INTO articles
            (tenant_id, language, title, slug, body, excerpt, status, author_id, is_breaking, is_featured, is_national, view_count, published_at)
        VALUES
            (t_mh, 'en', 'Mumbai-Pune Expressway Upgrades Cut Commute Time by 45 Minutes',
             'mumbai-pune-expressway-upgrades-cut-commute-time',
             '[{"type":"paragraph","text":"The new missing link tunnel project officially opened for commercial traffic this morning."}]',
             'New infrastructure link bridges the ghats section providing seamless connectivity.',
             'published', 3, TRUE, TRUE, FALSE, 1100, NOW() - INTERVAL '1 hour')
        ON CONFLICT (tenant_id, slug, language) DO NOTHING
        RETURNING id INTO a4_id;
    END IF;

    -- 5. Jharkhand State News
    IF t_jh IS NOT NULL THEN
        INSERT INTO articles
            (tenant_id, language, title, slug, body, excerpt, status, author_id, is_breaking, is_featured, is_national, view_count, published_at)
        VALUES
            (t_jh, 'en', 'Ranchi Tech Park Phase 2 Inaugurated to Create 15,000 IT Jobs',
             'ranchi-tech-park-phase-2-inaugurated-15000-jobs',
             '[{"type":"paragraph","text":"State government rolls out major IT incentive policy with global tech hub investments."}]',
             'Major boost for eastern India tech ecosystem with new software development campus.',
             'published', 1, FALSE, TRUE, FALSE, 430, NOW() - INTERVAL '3 hours')
        ON CONFLICT (tenant_id, slug, language) DO NOTHING
        RETURNING id INTO a5_id;
    END IF;

    -- Link article categories
    IF a1_id IS NOT NULL AND c_tech IS NOT NULL THEN
        INSERT INTO article_categories (article_id, category_id) VALUES (a1_id, c_tech) ON CONFLICT DO NOTHING;
    END IF;
    IF a2_id IS NOT NULL AND c_sports IS NOT NULL THEN
        INSERT INTO article_categories (article_id, category_id) VALUES (a2_id, c_sports) ON CONFLICT DO NOTHING;
    END IF;
    IF a3_id IS NOT NULL AND c_business IS NOT NULL THEN
        INSERT INTO article_categories (article_id, category_id) VALUES (a3_id, c_business) ON CONFLICT DO NOTHING;
    END IF;
END $$;

