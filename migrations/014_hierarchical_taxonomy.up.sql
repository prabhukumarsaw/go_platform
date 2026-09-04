-- 014_hierarchical_taxonomy.up.sql
-- Transform categories into an Advanced Hierarchical Taxonomy Tree (Parent -> Child -> Sub-Child)

ALTER TABLE categories ADD COLUMN IF NOT EXISTS parent_id INT REFERENCES categories(id) ON DELETE CASCADE;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS level INT DEFAULT 1;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS icon VARCHAR(50) DEFAULT '';
ALTER TABLE categories ADD COLUMN IF NOT EXISTS path TEXT DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_level ON categories(level);
CREATE INDEX IF NOT EXISTS idx_categories_path ON categories(path);

-- Seed Hierarchical Master Taxonomy
DO $$
DECLARE
    -- Top Level Topics
    c_national INT;
    c_politics INT;
    c_biz INT;
    c_tech INT;
    c_sports INT;
    c_ent INT;
    c_crime INT;
    c_lifestyle INT;
    c_states INT;

    -- State Nodes
    s_mh INT;
    s_br INT;
    s_jh INT;
    s_up INT;
    s_dl INT;
    s_ka INT;
    s_wb INT;
    s_tn INT;
    s_ts INT;
    s_ap INT;
    s_gj INT;
    s_rj INT;
BEGIN
    -- 1. Root Topical Categories (Level 1)
    INSERT INTO categories (name, slug, level, icon, path, tenant_id)
    VALUES 
        ('National Wire', 'national', 1, 'IconFlame', 'National Wire', 1),
        ('Politics & Governance', 'politics', 1, 'IconBuildingCommunity', 'Politics & Governance', 1),
        ('Business & Economy', 'business', 1, 'IconChartBar', 'Business & Economy', 1),
        ('Tech & Science', 'technology', 1, 'IconCpu', 'Tech & Science', 1),
        ('Sports Arena', 'sports', 1, 'IconBallFootball', 'Sports Arena', 1),
        ('Entertainment & OTT', 'entertainment', 1, 'IconMovie', 'Entertainment & OTT', 1),
        ('Crime & Legal', 'crime', 1, 'IconShieldAlert', 'Crime & Legal', 1),
        ('Lifestyle & Health', 'lifestyle', 1, 'IconHeartPulse', 'Lifestyle & Health', 1),
        ('States & Cities', 'states', 1, 'IconMapPin', 'States & Cities', 1)
    ON CONFLICT (tenant_id, slug) DO UPDATE 
    SET level = EXCLUDED.level, icon = EXCLUDED.icon, path = EXCLUDED.path;

    SELECT id INTO c_national FROM categories WHERE slug = 'national';
    SELECT id INTO c_politics FROM categories WHERE slug = 'politics';
    SELECT id INTO c_biz FROM categories WHERE slug = 'business';
    SELECT id INTO c_tech FROM categories WHERE slug = 'technology';
    SELECT id INTO c_sports FROM categories WHERE slug = 'sports';
    SELECT id INTO c_ent FROM categories WHERE slug = 'entertainment';
    SELECT id INTO c_crime FROM categories WHERE slug = 'crime';
    SELECT id INTO c_lifestyle FROM categories WHERE slug = 'lifestyle';
    SELECT id INTO c_states FROM categories WHERE slug = 'states';

    -- 2. Level 2 Topical Sub-Categories
    -- Sports sub-categories
    INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
        ('Cricket & IPL', 'cricket', c_sports, 2, 'Sports Arena > Cricket & IPL', 1),
        ('Football', 'football', c_sports, 2, 'Sports Arena > Football', 1),
        ('Badminton & Tennis', 'badminton-tennis', c_sports, 2, 'Sports Arena > Badminton & Tennis', 1),
        ('Motorsport', 'motorsport', c_sports, 2, 'Sports Arena > Motorsport', 1)
    ON CONFLICT (tenant_id, slug) DO NOTHING;

    -- Politics sub-categories
    INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
        ('Parliament & Lok Sabha', 'parliament', c_politics, 2, 'Politics & Governance > Parliament & Lok Sabha', 1),
        ('Elections & Polls', 'elections', c_politics, 2, 'Politics & Governance > Elections & Polls', 1),
        ('Foreign Affairs & Diplomacy', 'diplomacy', c_politics, 2, 'Politics & Governance > Foreign Affairs & Diplomacy', 1)
    ON CONFLICT (tenant_id, slug) DO NOTHING;

    -- Tech sub-categories
    INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
        ('Artificial Intelligence', 'ai', c_tech, 2, 'Tech & Science > Artificial Intelligence', 1),
        ('Smartphones & Gadgets', 'gadgets', c_tech, 2, 'Tech & Science > Smartphones & Gadgets', 1),
        ('Space & ISRO', 'isro-space', c_tech, 2, 'Tech & Science > Space & ISRO', 1),
        ('Cybersecurity', 'cybersecurity', c_tech, 2, 'Tech & Science > Cybersecurity', 1)
    ON CONFLICT (tenant_id, slug) DO NOTHING;

    -- Business sub-categories
    INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
        ('Stock Markets & Sensex', 'markets', c_biz, 2, 'Business & Economy > Stock Markets & Sensex', 1),
        ('Startups & Unicorns', 'startups', c_biz, 2, 'Business & Economy > Startups & Unicorns', 1),
        ('Banking & RBI', 'banking', c_biz, 2, 'Business & Economy > Banking & RBI', 1),
        ('Real Estate & Infra', 'real-estate', c_biz, 2, 'Business & Economy > Real Estate & Infra', 1)
    ON CONFLICT (tenant_id, slug) DO NOTHING;

    -- 3. Level 2 Geographic States (Under "States & Cities")
    INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
        ('Maharashtra', 'maharashtra', c_states, 2, 'States & Cities > Maharashtra', 1),
        ('Bihar', 'bihar', c_states, 2, 'States & Cities > Bihar', 1),
        ('Jharkhand', 'jharkhand', c_states, 2, 'States & Cities > Jharkhand', 1),
        ('Uttar Pradesh', 'uttar-pradesh', c_states, 2, 'States & Cities > Uttar Pradesh', 1),
        ('Delhi NCR', 'delhi-ncr', c_states, 2, 'States & Cities > Delhi NCR', 1),
        ('Karnataka', 'karnataka', c_states, 2, 'States & Cities > Karnataka', 1),
        ('West Bengal', 'west-bengal', c_states, 2, 'States & Cities > West Bengal', 1),
        ('Tamil Nadu', 'tamil-nadu', c_states, 2, 'States & Cities > Tamil Nadu', 1),
        ('Telangana', 'telangana', c_states, 2, 'States & Cities > Telangana', 1),
        ('Andhra Pradesh', 'andhra-pradesh', c_states, 2, 'States & Cities > Andhra Pradesh', 1),
        ('Gujarat', 'gujarat', c_states, 2, 'States & Cities > Gujarat', 1),
        ('Rajasthan', 'rajasthan', c_states, 2, 'States & Cities > Rajasthan', 1)
    ON CONFLICT (tenant_id, slug) DO UPDATE 
    SET parent_id = EXCLUDED.parent_id, level = EXCLUDED.level, path = EXCLUDED.path;

    SELECT id INTO s_mh FROM categories WHERE slug = 'maharashtra';
    SELECT id INTO s_br FROM categories WHERE slug = 'bihar';
    SELECT id INTO s_jh FROM categories WHERE slug = 'jharkhand';
    SELECT id INTO s_up FROM categories WHERE slug = 'uttar-pradesh';
    SELECT id INTO s_dl FROM categories WHERE slug = 'delhi-ncr';
    SELECT id INTO s_ka FROM categories WHERE slug = 'karnataka';
    SELECT id INTO s_wb FROM categories WHERE slug = 'west-bengal';
    SELECT id INTO s_ts FROM categories WHERE slug = 'telangana';
    SELECT id INTO s_ap FROM categories WHERE slug = 'andhra-pradesh';

    -- 4. Level 3 Geographic Districts / Cities
    -- Maharashtra Cities
    IF s_mh IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Mumbai', 'mumbai', s_mh, 3, 'States & Cities > Maharashtra > Mumbai', 1),
            ('Pune', 'pune', s_mh, 3, 'States & Cities > Maharashtra > Pune', 1),
            ('Nagpur', 'nagpur', s_mh, 3, 'States & Cities > Maharashtra > Nagpur', 1),
            ('Nashik', 'nashik', s_mh, 3, 'States & Cities > Maharashtra > Nashik', 1),
            ('Thane', 'thane', s_mh, 3, 'States & Cities > Maharashtra > Thane', 1),
            ('Chhatrapati Sambhajinagar', 'aurangabad', s_mh, 3, 'States & Cities > Maharashtra > Chhatrapati Sambhajinagar', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Bihar Districts & Cities
    IF s_br IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Patna', 'patna', s_br, 3, 'States & Cities > Bihar > Patna', 1),
            ('Gaya', 'gaya', s_br, 3, 'States & Cities > Bihar > Gaya', 1),
            ('Muzaffarpur', 'muzaffarpur', s_br, 3, 'States & Cities > Bihar > Muzaffarpur', 1),
            ('Bhagalpur', 'bhagalpur', s_br, 3, 'States & Cities > Bihar > Bhagalpur', 1),
            ('Darbhanga', 'darbhanga', s_br, 3, 'States & Cities > Bihar > Darbhanga', 1),
            ('Purnia', 'purnia', s_br, 3, 'States & Cities > Bihar > Purnia', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Jharkhand Districts & Cities
    IF s_jh IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Ranchi', 'ranchi', s_jh, 3, 'States & Cities > Jharkhand > Ranchi', 1),
            ('Jamshedpur', 'jamshedpur', s_jh, 3, 'States & Cities > Jharkhand > Jamshedpur', 1),
            ('Dhanbad', 'dhanbad', s_jh, 3, 'States & Cities > Jharkhand > Dhanbad', 1),
            ('Bokaro', 'bokaro', s_jh, 3, 'States & Cities > Jharkhand > Bokaro', 1),
            ('Deoghar', 'deoghar', s_jh, 3, 'States & Cities > Jharkhand > Deoghar', 1),
            ('Hazaribagh', 'hazaribagh', s_jh, 3, 'States & Cities > Jharkhand > Hazaribagh', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Uttar Pradesh Cities
    IF s_up IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Lucknow', 'lucknow', s_up, 3, 'States & Cities > Uttar Pradesh > Lucknow', 1),
            ('Noida & Greater Noida', 'noida', s_up, 3, 'States & Cities > Uttar Pradesh > Noida & Greater Noida', 1),
            ('Varanasi', 'varanasi', s_up, 3, 'States & Cities > Uttar Pradesh > Varanasi', 1),
            ('Kanpur', 'kanpur', s_up, 3, 'States & Cities > Uttar Pradesh > Kanpur', 1),
            ('Ayodhya', 'ayodhya', s_up, 3, 'States & Cities > Uttar Pradesh > Ayodhya', 1),
            ('Prayagraj', 'prayagraj', s_up, 3, 'States & Cities > Uttar Pradesh > Prayagraj', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Delhi NCR Localities
    IF s_dl IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Central Delhi', 'central-delhi', s_dl, 3, 'States & Cities > Delhi NCR > Central Delhi', 1),
            ('Gurugram', 'gurugram', s_dl, 3, 'States & Cities > Delhi NCR > Gurugram', 1),
            ('Faridabad', 'faridabad', s_dl, 3, 'States & Cities > Delhi NCR > Faridabad', 1),
            ('Ghaziabad', 'ghaziabad', s_dl, 3, 'States & Cities > Delhi NCR > Ghaziabad', 1),
            ('Dwarka & West Delhi', 'dwarka', s_dl, 3, 'States & Cities > Delhi NCR > Dwarka & West Delhi', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Karnataka Cities
    IF s_ka IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Bengaluru', 'bengaluru', s_ka, 3, 'States & Cities > Karnataka > Bengaluru', 1),
            ('Mysuru', 'mysuru', s_ka, 3, 'States & Cities > Karnataka > Mysuru', 1),
            ('Hubballi-Dharwad', 'hubballi', s_ka, 3, 'States & Cities > Karnataka > Hubballi-Dharwad', 1),
            ('Mangaluru', 'mangaluru', s_ka, 3, 'States & Cities > Karnataka > Mangaluru', 1),
            ('Belagavi', 'belagavi', s_ka, 3, 'States & Cities > Karnataka > Belagavi', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- West Bengal Cities
    IF s_wb IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Kolkata', 'kolkata', s_wb, 3, 'States & Cities > West Bengal > Kolkata', 1),
            ('Howrah', 'howrah', s_wb, 3, 'States & Cities > West Bengal > Howrah', 1),
            ('Siliguri', 'siliguri', s_wb, 3, 'States & Cities > West Bengal > Siliguri', 1),
            ('Durgapur', 'durgapur', s_wb, 3, 'States & Cities > West Bengal > Durgapur', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    -- Telangana & Andhra Pradesh Cities
    IF s_ts IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Hyderabad', 'hyderabad', s_ts, 3, 'States & Cities > Telangana > Hyderabad', 1),
            ('Warangal', 'warangal', s_ts, 3, 'States & Cities > Telangana > Warangal', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

    IF s_ap IS NOT NULL THEN
        INSERT INTO categories (name, slug, parent_id, level, path, tenant_id) VALUES
            ('Visakhapatnam', 'visakhapatnam', s_ap, 3, 'States & Cities > Andhra Pradesh > Visakhapatnam', 1),
            ('Vijayawada', 'vijayawada', s_ap, 3, 'States & Cities > Andhra Pradesh > Vijayawada', 1),
            ('Tirupati', 'tirupati', s_ap, 3, 'States & Cities > Andhra Pradesh > Tirupati', 1)
        ON CONFLICT (tenant_id, slug) DO NOTHING;
    END IF;

END $$;
