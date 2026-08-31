-- 004_content.up.sql
-- Content model: stories, articles, categories, tags, versions, live blogs, media, comments.

-- Story groups link translation variants of the same real-world event.
CREATE TABLE stories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    slug        VARCHAR(200),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stories_tenant ON stories(tenant_id);

-- Articles — the core content unit.
CREATE TABLE articles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id        UUID REFERENCES stories(id) ON DELETE SET NULL,
    tenant_id       INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    district_id     INT REFERENCES districts(id),
    language        VARCHAR(10) NOT NULL DEFAULT 'en' REFERENCES languages(code),
    title           TEXT NOT NULL,
    slug            VARCHAR(300) NOT NULL,
    body            JSONB NOT NULL DEFAULT '[]',
    excerpt         TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','review','approved','published','archived')),
    author_id       BIGINT NOT NULL REFERENCES users(id),
    editor_id       BIGINT REFERENCES users(id),
    reviewer_id     BIGINT REFERENCES users(id),
    is_breaking     BOOLEAN DEFAULT FALSE,
    is_featured     BOOLEAN DEFAULT FALSE,
    is_national     BOOLEAN DEFAULT FALSE,
    published_at    TIMESTAMPTZ,
    scheduled_at    TIMESTAMPTZ,
    meta_title      VARCHAR(200),
    meta_description VARCHAR(500),
    og_image        TEXT,
    featured_image  TEXT,
    view_count      BIGINT DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, slug, language)
);

COMMENT ON COLUMN articles.body IS 'Block-based content (Notion/Gutenberg-style JSON), not raw HTML. Renders cleanly on web, app, and AMP.';
COMMENT ON COLUMN articles.is_national IS 'If TRUE, this article is visible across all state tenants (exempted from RLS tenant filter).';

CREATE INDEX idx_articles_tenant ON articles(tenant_id);
CREATE INDEX idx_articles_status ON articles(tenant_id, status);
CREATE INDEX idx_articles_author ON articles(author_id);
CREATE INDEX idx_articles_published ON articles(tenant_id, published_at DESC) WHERE status = 'published';
CREATE INDEX idx_articles_story ON articles(story_id) WHERE story_id IS NOT NULL;
CREATE INDEX idx_articles_breaking ON articles(tenant_id) WHERE is_breaking = TRUE AND status = 'published';
CREATE INDEX idx_articles_featured ON articles(tenant_id) WHERE is_featured = TRUE AND status = 'published';

-- Categories (tenant-scoped, hierarchical)
CREATE TABLE categories (
    id          SERIAL PRIMARY KEY,
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id   INT REFERENCES categories(id),
    sort_order  INT DEFAULT 0,
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

-- Tags (global, shared across tenants)
CREATE TABLE tags (
    id      SERIAL PRIMARY KEY,
    name    VARCHAR(100) NOT NULL UNIQUE,
    slug    VARCHAR(100) NOT NULL UNIQUE
);

-- Article ↔ Category junction
CREATE TABLE article_categories (
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    category_id INT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, category_id)
);

-- Article ↔ Tag junction
CREATE TABLE article_tags (
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag_id     INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, tag_id)
);

-- Article version history (editorial accountability)
CREATE TABLE article_versions (
    id          BIGSERIAL PRIMARY KEY,
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    edited_by   BIGINT NOT NULL REFERENCES users(id),
    diff        JSONB NOT NULL,
    snapshot    JSONB,
    version_num INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_versions_article ON article_versions(article_id, created_at DESC);

-- Live blog entries (append-only, distinct from article body)
CREATE TABLE live_blog_entries (
    id          BIGSERIAL PRIMARY KEY,
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    body        JSONB NOT NULL,
    author_id   BIGINT NOT NULL REFERENCES users(id),
    is_pinned   BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_liveblog_article ON live_blog_entries(article_id, created_at DESC);

-- Media library (local filesystem storage)
CREATE TABLE media (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    uploader_id BIGINT NOT NULL REFERENCES users(id),
    filename    VARCHAR(500) NOT NULL,
    original_name VARCHAR(500) NOT NULL,
    mime_type   VARCHAR(100) NOT NULL,
    file_size   BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    alt_text    TEXT,
    caption     TEXT,
    width       INT,
    height      INT,
    variants    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN media.storage_path IS 'Relative path within the local uploads directory (e.g., "tenant_1/2024/03/image.webp").';
COMMENT ON COLUMN media.variants IS 'Generated size variants: {"thumb": "path", "medium": "path", "large": "path"}';

CREATE INDEX idx_media_tenant ON media(tenant_id, created_at DESC);

-- Comments (viewer-facing, moderated)
CREATE TABLE comments (
    id          BIGSERIAL PRIMARY KEY,
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    parent_id   BIGINT REFERENCES comments(id) ON DELETE CASCADE,
    body        TEXT NOT NULL,
    status      VARCHAR(20) DEFAULT 'pending'
                CHECK (status IN ('pending','approved','rejected','spam')),
    moderated_by BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_comments_article ON comments(article_id, created_at DESC) WHERE status = 'approved';
CREATE INDEX idx_comments_pending ON comments(status, created_at DESC) WHERE status = 'pending';
