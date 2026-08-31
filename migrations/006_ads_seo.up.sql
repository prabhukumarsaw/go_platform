-- 006_ads_seo.up.sql
-- Ad slots, sponsored content tracking, and DPDP Act consent.

CREATE TABLE ad_slots (
    id          SERIAL PRIMARY KEY,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    slot_type   VARCHAR(30) NOT NULL CHECK (slot_type IN ('header', 'in_article', 'sidebar', 'native', 'footer', 'interstitial')),
    ad_unit_id  VARCHAR(200),
    config      JSONB DEFAULT '{}',
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

ALTER TABLE ad_slots ENABLE ROW LEVEL SECURITY;
ALTER TABLE ad_slots FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON ad_slots
    USING (
        tenant_id = current_setting('app.tenant_id', true)::int
        OR current_setting('app.is_super_admin', true)::boolean = true
    );

-- Sponsored content flag
CREATE TABLE sponsored_articles (
    article_id  UUID PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
    sponsor     VARCHAR(200) NOT NULL,
    campaign_id VARCHAR(100),
    start_date  DATE,
    end_date    DATE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- DPDP Act consent tracking (mandatory for Indian platforms)
CREATE TABLE consent_records (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    session_id      VARCHAR(100),
    consent_type    VARCHAR(50) NOT NULL,
    is_granted      BOOLEAN NOT NULL,
    policy_version  VARCHAR(20) NOT NULL,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE consent_records IS 'DPDP Act compliance: tracks what each user consented to, and which policy version was shown.';

CREATE INDEX idx_consent_user ON consent_records(user_id, consent_type);

-- Newsletter subscriptions
CREATE TABLE newsletter_subscriptions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    tenant_id   INT NOT NULL REFERENCES tenants(id),
    frequency   VARCHAR(20) DEFAULT 'daily' CHECK (frequency IN ('daily', 'weekly', 'breaking_only')),
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(email, tenant_id)
);
