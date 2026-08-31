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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"newsplatform/api/pkg/config"
	"github.com/rs/zerolog"
)

// Service handles media upload, storage, and retrieval using local filesystem.
type Service struct {
	pool      *pgxpool.Pool
	cfg       config.MediaConfig
	logger    zerolog.Logger
}

// NewService creates a new media service with local filesystem storage.
func NewService(pool *pgxpool.Pool, cfg config.MediaConfig, logger zerolog.Logger) *Service {
	// Ensure upload directory exists
	os.MkdirAll(cfg.UploadDir, 0755)

	return &Service{
		pool:   pool,
		cfg:    cfg,
		logger: logger.With().Str("module", "media").Logger(),
	}
}

// ─── Models ───────────// Media represents a media file record.
type Media struct {
	ID           uuid.UUID `json:"id"`
	TenantID     int       `json:"tenant_id"`
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

// ─── Upload ─────────────────────────────────────

// UploadFile saves a file to local storage and records metadata in the database.
func (s *Service) UploadFile(ctx context.Context, tx pgx.Tx, tenantID int, uploaderID int64, filename string, mimeType string, category string, folder string, fileSize int64, reader io.Reader) (*Media, error) {
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

	// Generate storage path: tenant_{id}/{category}/{year}/{month}/{uuid}_{filename}
	now := time.Now()
	mediaID := uuid.New()
	ext := filepath.Ext(filename)
	storedFilename := fmt.Sprintf("%s%s", mediaID.String()[:12], ext)
	relativePath := filepath.Join(
		fmt.Sprintf("tenant_%d", tenantID),
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

	// Insert metadata into database
	query := `
		INSERT INTO media
			(id, tenant_id, uploader_id, filename, original_name, mime_type,
			 category, folder, file_size, storage_path, alt_text, caption, width, height)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, '', '', $11, $12)
		RETURNING created_at
	`

	var createdAt time.Time
	err = tx.QueryRow(ctx, query,
		mediaID, tenantID, uploaderID, storedFilename, filename, mimeType,
		category, folder, written, relativePath, width, height,
	).Scan(&createdAt)
	if err != nil {
		os.Remove(absolutePath) // Clean up on DB failure
		return nil, fmt.Errorf("insert media record: %w", err)
	}

	return &Media{
		ID:           mediaID,
		TenantID:     tenantID,
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

// ListMedia returns media files filtered by category, mimeType, and search query.
func (s *Service) ListMedia(ctx context.Context, tx pgx.Tx, tenantID int, category string, mimeType string, search string, page, perPage int) ([]Media, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	whereClauses := []string{"(tenant_id = $1 OR $1 = 1)"}
	args := []interface{}{tenantID}
	argIdx := 2

	if category != "" && category != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
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
		SELECT id, tenant_id, uploader_id, filename, original_name, mime_type,
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
			&m.ID, &m.TenantID, &m.UploaderID, &m.Filename, &m.OriginalName,
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

// optimizeImage decodes, compresses at 82% quality, and re-writes the image file.
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

	// If JPEG or PNG, compress and optimize
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		var buf bytes.Buffer
		opts := &jpeg.Options{Quality: 82} // Standard editorial compression
		if err := jpeg.Encode(&buf, img, opts); err == nil {
			// Write optimized buffer back to file
			if err := os.WriteFile(path, buf.Bytes(), 0644); err == nil {
				return width, height, int64(buf.Len()), nil
			}
		}
	}

	return width, height, 0, nil
}
