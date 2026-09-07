package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := "postgres://postgres:sawraj@localhost:5432/platform_db?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	sql := `
		-- 1. Ensure live_blog_entries schema
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
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS headline VARCHAR(255) DEFAULT '';
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS body JSONB DEFAULT '""'::jsonb;
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS author_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN DEFAULT FALSE;
		ALTER TABLE live_blog_entries ADD COLUMN IF NOT EXISTS is_breaking BOOLEAN DEFAULT FALSE;
		ALTER TABLE live_blog_entries ALTER COLUMN content DROP NOT NULL;
		ALTER TABLE live_blog_entries ALTER COLUMN content SET DEFAULT '';
		ALTER TABLE live_blog_entries ALTER COLUMN title DROP NOT NULL;
		ALTER TABLE live_blog_entries ALTER COLUMN title SET DEFAULT '';

		-- 2. Ensure media schema
		DO $$
		DECLARE
			id_type text;
		BEGIN
			SELECT data_type INTO id_type 
			FROM information_schema.columns 
			WHERE table_name = 'media' AND column_name = 'id';

			IF id_type IS NOT NULL AND id_type != 'uuid' THEN
				DROP TABLE IF EXISTS media CASCADE;
			END IF;
		END $$;

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

		ALTER TABLE media ADD COLUMN IF NOT EXISTS uploader_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
		ALTER TABLE media ADD COLUMN IF NOT EXISTS original_name VARCHAR(255) DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS file_size BIGINT DEFAULT 0;
		ALTER TABLE media ADD COLUMN IF NOT EXISTS storage_path TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS url TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS alt_text TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS caption TEXT DEFAULT '';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS width INT DEFAULT 0;
		ALTER TABLE media ADD COLUMN IF NOT EXISTS height INT DEFAULT 0;

		CREATE INDEX IF NOT EXISTS idx_media_category ON media(category);
		CREATE INDEX IF NOT EXISTS idx_media_folder ON media(folder);
		CREATE INDEX IF NOT EXISTS idx_media_created_at ON media(created_at DESC);
	`

	_, err = pool.Exec(ctx, sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SUCCESS: media and live_blog_entries database schema updated successfully!")
}
