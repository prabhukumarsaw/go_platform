-- 020_add_scheduled_status_and_workflow.up.sql
-- Add 'scheduled' status to articles and optimize scheduled publishing queries

-- 1. Update articles status constraint to include 'scheduled'
ALTER TABLE articles DROP CONSTRAINT IF EXISTS articles_status_check;
ALTER TABLE articles ADD CONSTRAINT articles_status_check 
    CHECK (status IN ('draft', 'review', 'approved', 'scheduled', 'published', 'archived'));

-- 2. Index for background scheduler job to fast-query pending scheduled articles
CREATE INDEX IF NOT EXISTS idx_articles_scheduled_queue 
    ON articles(scheduled_at) 
    WHERE status = 'scheduled' AND scheduled_at IS NOT NULL;
