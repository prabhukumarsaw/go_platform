-- 008_features_production.up.sql
-- Adds: Web Stories (Google Discover/AMP format), Opinion Polls & Voting, E-Paper Editions, and Fact-Checking badges.

-- ──────────────────────────────────────────────
-- 1. WEB STORIES / VISUAL SHORTS (Google Discover format)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS web_stories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    slug            VARCHAR(300) NOT NULL,
    language        VARCHAR(10) NOT NULL DEFAULT 'hi',
    cover_image     TEXT NOT NULL,
    slides          JSONB NOT NULL DEFAULT '[]', -- Array of slide objects (image, video, text, link)
    author_id       BIGINT NOT NULL REFERENCES users(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'published' CHECK (status IN ('draft', 'published', 'archived')),
    view_count      BIGINT DEFAULT 0,
    published_at    TIMESTAMPTZ DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, slug, language)
);

CREATE INDEX IF NOT EXISTS idx_web_stories_tenant ON web_stories(tenant_id, published_at DESC) WHERE status = 'published';

-- ──────────────────────────────────────────────
-- 2. OPINION POLLS & AUDIENCE ENGAGEMENT
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS polls (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    question        TEXT NOT NULL,
    language        VARCHAR(10) NOT NULL DEFAULT 'hi',
    options         JSONB NOT NULL, -- [{"id": 1, "text": "Option A", "votes": 120}, ...]
    total_votes     BIGINT DEFAULT 0,
    is_active       BOOLEAN DEFAULT TRUE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS poll_votes (
    id              BIGSERIAL PRIMARY KEY,
    poll_id         UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    user_id         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    option_id       INT NOT NULL,
    ip_address      INET,
    voted_at        TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(poll_id, ip_address),
    UNIQUE(poll_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_polls_active ON polls(tenant_id, is_active);

-- ──────────────────────────────────────────────
-- 3. E-PAPER DIGITAL PRINT EDITIONS (Prabhat Khabar / Dainik Jagran style)
-- ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS epapers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    district_id     INT REFERENCES districts(id) ON DELETE SET NULL,
    edition_date    DATE NOT NULL,
    title           VARCHAR(255) NOT NULL,
    pdf_url         TEXT NOT NULL,
    thumbnail_url   TEXT,
    page_count      INT DEFAULT 1,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, district_id, edition_date)
);

CREATE INDEX IF NOT EXISTS idx_epapers_date ON epapers(tenant_id, edition_date DESC);

-- ──────────────────────────────────────────────
-- 4. FACT-CHECKING METADATA EXTENSION FOR ARTICLES
-- ──────────────────────────────────────────────
ALTER TABLE articles ADD COLUMN IF NOT EXISTS is_fact_check BOOLEAN DEFAULT FALSE;
ALTER TABLE articles ADD COLUMN IF NOT EXISTS fact_check_verdict VARCHAR(50); -- 'TRUE', 'FALSE', 'MISLEADING', 'EXAGGERATED'
ALTER TABLE articles ADD COLUMN IF NOT EXISTS reading_time_minutes INT DEFAULT 3;
ALTER TABLE articles ADD COLUMN IF NOT EXISTS audio_narration_url TEXT;

-- ──────────────────────────────────────────────
-- 5. INITIAL SEED FOR WEB STORIES, POLLS & EPAPERS
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t_national INT;
    t_mh INT;
    t_jh INT;
    p_id UUID;
BEGIN
    SELECT id INTO t_national FROM tenants WHERE slug='national';
    SELECT id INTO t_mh FROM tenants WHERE slug='maharashtra';
    SELECT id INTO t_jh FROM tenants WHERE slug='jharkhand';

    -- Seed 1: Visual Web Story
    IF t_national IS NOT NULL THEN
        INSERT INTO web_stories (tenant_id, title, slug, language, cover_image, slides, author_id)
        VALUES (
            t_national,
            'Top 5 Indian Space Exploration Milestones in 2026',
            'top-5-indian-space-exploration-milestones-2026',
            'hi',
            'https://images.unsplash.com/photo-1517976487050-73fb78b4618e?q=80&w=800',
            '[
                {"title": "Chandrayaan 4 Rover", "image": "https://images.unsplash.com/photo-1614728894747-a83421e2b9c9?q=80&w=800", "description": "New sample return mission successfully initiates drilling operations."},
                {"title": "Gaganyaan Crew Orbit", "image": "https://images.unsplash.com/photo-1451187580459-43490279c0fa?q=80&w=800", "description": "Indian vyomanauts successfully complete 3-day microgravity orbit tests."},
                {"title": "Shukrayaan Venus Probe", "image": "https://images.unsplash.com/photo-1541185933-ef5d8ed016c2?q=80&w=800", "description": "Atmospheric sensor satellite launched towards the Venusian orbit."}
            ]',
            1
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

    -- Seed 2: Opinion Poll
    IF t_national IS NOT NULL THEN
        INSERT INTO polls (id, tenant_id, question, language, options, total_votes)
        VALUES (
            'a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d',
            t_national,
            'क्या भारतीय रेलवे का नया ''वंदे भारत स्लीपर'' नेटवर्क लंबी दूरी की यात्रा को पूरी तरह बदल देगा?',
            'hi',
            '[
                {"id": 1, "text": "हाँ, यात्रा समय और आराम में क्रांतिकारी बदलाव आएगा", "votes": 420},
                {"id": 2, "text": "नहीं, टिकट दरें सामान्य यात्रियों के लिए अधिक हैं", "votes": 150},
                {"id": 3, "text": "कह नहीं सकते / समीक्षा आवश्यक है", "votes": 35}
            ]',
            605
        ) ON CONFLICT (id) DO NOTHING;
    END IF;

    -- Seed 3: E-Paper
    IF t_jh IS NOT NULL THEN
        INSERT INTO epapers (tenant_id, edition_date, title, pdf_url, thumbnail_url, page_count)
        VALUES (
            t_jh,
            CURRENT_DATE,
            'Ranchi Prabhat Edition - Daily Newspaper',
            'https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf',
            'https://images.unsplash.com/photo-1585829365295-ab7cd400c167?q=80&w=600',
            16
        ) ON CONFLICT (tenant_id, district_id, edition_date) DO NOTHING;
    END IF;
END $$;
