package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	jpeg "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/image/draw"
	"newsplatform/api/pkg/config"
)

// Service handles media upload, storage, and retrieval using local filesystem.
type Service struct {
	pool       *pgxpool.Pool
	cfg        config.MediaConfig
	logger     zerolog.Logger
	schemaOnce sync.Once
}

// NewService creates a new media service with local filesystem storage.
func NewService(pool *pgxpool.Pool, cfg config.MediaConfig, logger zerolog.Logger) *Service {
	// Ensure upload directory exists
	os.MkdirAll(cfg.UploadDir, 0755)

	s := &Service{
		pool:   pool,
		cfg:    cfg,
		logger: logger.With().Str("module", "media").Logger(),
	}

	// Proactively verify/create schema in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.ensureSchema(ctx)
	}()

	return s
}

// EnsureSchema verifies that the media table exists with the proper UUID primary key and required columns.
func (s *Service) EnsureSchema(ctx context.Context) error {
	migrationQuery := `
		DO $$
		DECLARE
			id_type text;
		BEGIN
			SELECT data_type INTO id_type 
			FROM information_schema.columns 
			WHERE table_name = 'media' AND column_name = 'id';

			-- If table exists but id is integer/serial or uploader_id is missing, recreate table cleanly
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
	_, err := s.pool.Exec(ctx, migrationQuery)
	return err
}

func (s *Service) ensureSchema(ctx context.Context) {
	s.schemaOnce.Do(func() {
		if err := s.EnsureSchema(ctx); err != nil {
			s.logger.Warn().Err(err).Msg("failed to ensure media table schema")
		}
	})
}

// ─── Models ───────────// Media represents a media file record.
type Media struct {
	ID           uuid.UUID `json:"id"`
	UploaderID   int64     `json:"uploader_id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	Category     string    `json:"category"` // news, ads, custom, avatars
	Folder       string    `json:"folder"`   // general, banners, election2026, breaking
	FileSize     int64     `json:"file_size"`
	StoragePath  string    `json:"storage_path"`
	URL          string    `json:"url"`
	AltText      string    `json:"alt_text"`
	Caption      string    `json:"caption"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// FolderSummary represents a media folder and its file count.
type FolderSummary struct {
	Folder string `json:"folder"`
	Count  int64  `json:"count"`
}

// ─── Upload ─────────────────────────────────────

// UploadFile saves a file to local storage and records metadata in the database.
func (s *Service) UploadFile(ctx context.Context, tx pgx.Tx, uploaderID int64, filename string, mimeType string, category string, folder string, fileSize int64, reader io.Reader) (*Media, error) {
	// Validate file size
	if fileSize > s.cfg.MaxFileSize {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes", s.cfg.MaxFileSize)
	}

	// Validate MIME type
	if !isAllowedMimeType(mimeType) {
		return nil, fmt.Errorf("unsupported file type: %s", mimeType)
	}

	if category == "" {
		category = "news"
	}
	if folder == "" {
		folder = "general"
	}

	// Generate storage path: {category}/{year}/{month}/{uuid}_{filename}
	now := time.Now()
	mediaID := uuid.New()
	ext := filepath.Ext(filename)
	storedFilename := fmt.Sprintf("%s%s", mediaID.String()[:12], ext)
	relativePath := filepath.Join(
		category,
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		storedFilename,
	)

	absolutePath := filepath.Join(s.cfg.UploadDir, relativePath)

	// Create directory tree
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0755); err != nil {
		return nil, fmt.Errorf("create media directory: %w", err)
	}

	// Write file to disk
	outFile, err := os.Create(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, reader)
	if err != nil {
		os.Remove(absolutePath) // Clean up on failure
		return nil, fmt.Errorf("write file: %w", err)
	}

	// Optimize image and get dimensions if applicable
	var width, height int
	if isImageMime(mimeType) {
		w, h, optSize, optErr := optimizeImage(absolutePath, mimeType)
		if optErr == nil {
			width = w
			height = h
			if optSize > 0 {
				written = optSize
			}
		} else {
			width, height = getImageDimensions(absolutePath)
		}
	}

	// Build public URL
	publicURL := fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.BaseURL, "/"), filepath.ToSlash(relativePath))

	s.ensureSchema(ctx)

	// Insert metadata into database
	query := `
		INSERT INTO media
			(id, uploader_id, filename, original_name, mime_type,
			 category, folder, file_size, storage_path, alt_text, caption, width, height)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '', '', $10, $11)
		RETURNING created_at
	`

	var uploaderIDParam *int64
	if uploaderID > 0 {
		uploaderIDParam = &uploaderID
	}

	var createdAt time.Time
	err = tx.QueryRow(ctx, query,
		mediaID, uploaderIDParam, storedFilename, filename, mimeType,
		category, folder, written, filepath.ToSlash(relativePath), width, height,
	).Scan(&createdAt)
	if err != nil {
		os.Remove(absolutePath) // Clean up on DB failure
		return nil, fmt.Errorf("insert media record: %w", err)
	}

	return &Media{
		ID:           mediaID,
		UploaderID:   uploaderID,
		Filename:     storedFilename,
		OriginalName: filename,
		MimeType:     mimeType,
		Category:     category,
		Folder:       folder,
		FileSize:     written,
		StoragePath:  relativePath,
		URL:          publicURL,
		Width:        width,
		Height:       height,
		CreatedAt:    createdAt,
	}, nil
}

// UploadAvatar processes, crops/resizes and links a user profile avatar.
func (s *Service) UploadAvatar(ctx context.Context, userID int64, filename string, mimeType string, fileSize int64, reader io.Reader) (string, error) {
	if !isImageMime(mimeType) {
		return "", fmt.Errorf("avatar must be an image (JPEG/PNG/WebP)")
	}

	avatarID := uuid.New()
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg"
	}
	storedFilename := fmt.Sprintf("avatar_%d_%s%s", userID, avatarID.String()[:8], ext)
	relativePath := filepath.Join("avatars", storedFilename)
	absolutePath := filepath.Join(s.cfg.UploadDir, relativePath)

	_ = os.MkdirAll(filepath.Dir(absolutePath), 0755)

	outFile, err := os.Create(absolutePath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return "", err
	}

	// Optimize image
	_, _, _, _ = optimizeImage(absolutePath, mimeType)

	avatarURL := fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.BaseURL, "/"), filepath.ToSlash(relativePath))

	// Update user record
	_, err = s.pool.Exec(ctx, "UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE id = $2", avatarURL, userID)
	if err != nil {
		return "", fmt.Errorf("update user avatar: %w", err)
	}

	return avatarURL, nil
}

// ─── List / Search / Filter ─────────────────────

// ListMedia returns media files filtered by category, folder, mimeType, and search query.
func (s *Service) ListMedia(ctx context.Context, tx pgx.Tx, category string, folder string, mimeType string, search string, page, perPage int) ([]Media, int64, error) {
	s.ensureSchema(ctx)

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	whereClauses := []string{"1=1"}
	var args []interface{}
	argIdx := 1

	if category != "" && category != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}

	if folder != "" && folder != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("folder = $%d", argIdx))
		args = append(args, folder)
		argIdx++
	}

	if mimeType != "" && mimeType != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("mime_type LIKE $%d", argIdx))
		args = append(args, mimeType+"%")
		argIdx++
	}

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(original_name ILIKE $%d OR alt_text ILIKE $%d OR caption ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM media WHERE %s", whereSQL)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	query := fmt.Sprintf(`
		SELECT id, uploader_id, filename, original_name, mime_type,
			   COALESCE(category, 'news'), COALESCE(folder, 'general'),
			   file_size, storage_path, COALESCE(alt_text, ''), COALESCE(caption, ''),
			   COALESCE(width, 0), COALESCE(height, 0), created_at
		FROM media
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, perPage, offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []Media{}
	for rows.Next() {
		var m Media
		if err := rows.Scan(
			&m.ID, &m.UploaderID, &m.Filename, &m.OriginalName,
			&m.MimeType, &m.Category, &m.Folder, &m.FileSize, &m.StoragePath,
			&m.AltText, &m.Caption, &m.Width, &m.Height, &m.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		m.URL = fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.BaseURL, "/"), filepath.ToSlash(m.StoragePath))
		items = append(items, m)
	}

	return items, total, nil
}

// ListFolders aggregates all distinct media folders and returns their item counts.
func (s *Service) ListFolders(ctx context.Context, tx pgx.Tx) ([]FolderSummary, error) {
	s.ensureSchema(ctx)

	query := `
		SELECT COALESCE(NULLIF(folder, ''), 'general') AS folder_name, COUNT(*) AS total
		FROM media
		GROUP BY 1
		ORDER BY total DESC, folder_name ASC
	`
	var rows pgx.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(ctx, query)
	} else {
		rows, err = s.pool.Query(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []FolderSummary
	for rows.Next() {
		var f FolderSummary
		if err := rows.Scan(&f.Folder, &f.Count); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	if folders == nil {
		folders = []FolderSummary{}
	}
	return folders, rows.Err()
}

// UpdateMediaMetadata updates alt_text, caption, category, and folder for a media asset.
func (s *Service) UpdateMediaMetadata(ctx context.Context, tx pgx.Tx, mediaID uuid.UUID, altText, caption, category, folder string) error {
	query := `
		UPDATE media
		SET alt_text = $1, caption = $2, category = $3, folder = $4
		WHERE id = $5
	`
	_, err := tx.Exec(ctx, query, altText, caption, category, folder, mediaID)
	return err
}

// DeleteMedia removes a media file from disk and database.
func (s *Service) DeleteMedia(ctx context.Context, tx pgx.Tx, mediaID uuid.UUID) error {
	var storagePath string
	err := tx.QueryRow(ctx, "SELECT storage_path FROM media WHERE id = $1", mediaID).Scan(&storagePath)
	if err != nil {
		return fmt.Errorf("find media: %w", err)
	}

	// Delete from database
	_, err = tx.Exec(ctx, "DELETE FROM media WHERE id = $1", mediaID)
	if err != nil {
		return fmt.Errorf("delete media record: %w", err)
	}

	// Delete from disk
	absolutePath := filepath.Join(s.cfg.UploadDir, storagePath)
	if err := os.Remove(absolutePath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn().Err(err).Str("path", absolutePath).Msg("failed to delete media file from disk")
	}

	return nil
}

// ─── Helpers ────────────────────────────────────

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/svg+xml":   true,
	"video/mp4":       true,
	"video/webm":      true,
	"application/pdf": true,
	"audio/mpeg":      true,
	"audio/ogg":       true,
}

func isAllowedMimeType(mime string) bool {
	return allowedMimeTypes[mime]
}

func isImageMime(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}

func getImageDimensions(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// optimizeImage scales oversized images down to standard Full HD (max width 1920px, max height 1080px),
// compresses at 82% quality, and re-writes the optimized file to conserve mobile reader bandwidth.
func optimizeImage(path string, mimeType string) (int, int, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return 0, 0, 0, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Full HD maximum bounds for news editorial photos
	const maxW = 1920
	const maxH = 1080

	targetW := width
	targetH := height

	// Downscale oversized images using high-fidelity CatmullRom resampling
	if width > maxW || height > maxH {
		ratioW := float64(maxW) / float64(width)
		ratioH := float64(maxH) / float64(height)
		scaleRatio := ratioW
		if ratioH < scaleRatio {
			scaleRatio = ratioH
		}
		targetW = int(float64(width) * scaleRatio)
		targetH = int(float64(height) * scaleRatio)
		if targetW < 1 {
			targetW = 1
		}
		if targetH < 1 {
			targetH = 1
		}

		dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		img = dst
		width = targetW
		height = targetH
	}

	// Compress JPEG or PNG with 82% quality
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		var buf bytes.Buffer
		opts := &jpeg.Options{Quality: 82} // Standard editorial compression
		if err := jpeg.Encode(&buf, img, opts); err == nil {
			if err := os.WriteFile(path, buf.Bytes(), 0644); err == nil {
				return width, height, int64(buf.Len()), nil
			}
		}
	}

	return width, height, 0, nil
}
