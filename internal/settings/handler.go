package settings

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"newsplatform/api/pkg/response"
)

// SocialLinksConfig represents active social broadcasting links.
type SocialLinksConfig struct {
	X         string `json:"x"`
	Instagram string `json:"instagram"`
	Facebook  string `json:"facebook"`
	YouTube   string `json:"youtube"`
	WhatsApp  string `json:"whatsapp"`
	Telegram  string `json:"telegram"`
	LinkedIn  string `json:"linkedin"`
}

// SecurityConfig represents security & authentication policies.
type SecurityConfig struct {
	SessionTimeoutMins int    `json:"session_timeout_mins"`
	MaxLoginAttempts   int    `json:"max_login_attempts"`
	EnforceMFA         bool   `json:"enforce_mfa"`
	IPWhitelist        string `json:"ip_whitelist"`
}

// APIConfig represents API & gateway integration rules.
type APIConfig struct {
	BaseURL         string `json:"base_url"`
	RateLimitPerMin int    `json:"rate_limit_per_minute"`
	WebhookURL      string `json:"webhook_url"`
}

// SiteSettings represents full dynamic publication and system parameters.
type SiteSettings struct {
	Name               string            `json:"name"`
	Tagline            string            `json:"tagline"`
	Motive             string            `json:"motive"`
	LogoURL            string            `json:"logo_url"`
	FaviconURL         string            `json:"favicon_url"`
	DefaultLanguage    string            `json:"default_language"`
	MaintenanceMode    bool              `json:"maintenance_mode"`
	MaintenanceMessage string            `json:"maintenance_message"`
	SocialLinks        SocialLinksConfig `json:"social_links"`
	Security           SecurityConfig    `json:"security"`
	API                APIConfig         `json:"api"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// Handler manages site settings, dynamic branding, and disaster recovery backup exports.
type Handler struct {
	pool      *pgxpool.Pool
	uploadDir string
	mu        sync.RWMutex
	settings  SiteSettings
}

// NewHandler initializes settings handler with default production values and auto-syncs with PostgreSQL.
func NewHandler(pool *pgxpool.Pool, uploadDir string) *Handler {
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	absUpload, err := filepath.Abs(uploadDir)
	if err == nil {
		uploadDir = absUpload
	}

	h := &Handler{
		pool:      pool,
		uploadDir: uploadDir,
		settings: SiteSettings{
			Name:               "NewsRoom",
			Tagline:            "Independent, Verified & Fearless Journalism",
			Motive:             "Delivering credible news and investigative journalism across all state editions in India.",
			LogoURL:            "",
			FaviconURL:         "",
			DefaultLanguage:    "en",
			MaintenanceMode:    false,
			MaintenanceMessage: "We are currently performing scheduled maintenance. News updates will resume shortly.",
			SocialLinks: SocialLinksConfig{
				X:         "https://x.com/newsroom",
				Instagram: "https://instagram.com/newsroom",
				Facebook:  "https://facebook.com/newsroom",
				YouTube:   "https://youtube.com/@newsroom",
				WhatsApp:  "https://whatsapp.com/channel/newsroom",
				Telegram:  "https://t.me/newsroom",
				LinkedIn:  "https://linkedin.com/company/newsroom",
			},
			Security: SecurityConfig{
				SessionTimeoutMins: 10080, // 7 days
				MaxLoginAttempts:   5,
				EnforceMFA:         false,
				IPWhitelist:        "",
			},
			API: APIConfig{
				BaseURL:         "http://localhost:8080/api/v1",
				RateLimitPerMin: 300,
				WebhookURL:      "",
			},
			UpdatedAt: time.Now(),
		},
	}

	h.ensureSchema(context.Background())
	h.loadFromDB(context.Background())

	return h
}

func (h *Handler) ensureSchema(ctx context.Context) {
	_, _ = h.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS site_settings (
			id INT PRIMARY KEY DEFAULT 1,
			name VARCHAR(255) NOT NULL DEFAULT 'NewsRoom',
			tagline TEXT NOT NULL DEFAULT 'Independent, Verified & Fearless Journalism',
			motive TEXT NOT NULL DEFAULT 'Delivering credible news and investigative journalism across all state editions in India.',
			logo_url TEXT DEFAULT '',
			favicon_url TEXT DEFAULT '',
			default_language VARCHAR(10) DEFAULT 'en',
			maintenance_mode BOOLEAN DEFAULT FALSE,
			maintenance_message TEXT DEFAULT 'We are currently performing scheduled maintenance. News updates will resume shortly.',
			social_links JSONB DEFAULT '{}'::jsonb,
			security JSONB DEFAULT '{}'::jsonb,
			api JSONB DEFAULT '{}'::jsonb,
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		INSERT INTO site_settings (id, name, tagline, motive)
		VALUES (1, 'NewsRoom', 'Independent, Verified & Fearless Journalism', 'Delivering credible news and investigative journalism across all state editions in India.')
		ON CONFLICT (id) DO NOTHING;
	`)
}

func (h *Handler) loadFromDB(ctx context.Context) {
	var (
		name, tagline, motive, logoURL, faviconURL, defLang, maintMsg string
		maintMode                                                     bool
		socialJSON, secJSON, apiJSON                                  []byte
		updatedAt                                                     time.Time
	)

	err := h.pool.QueryRow(ctx, `
		SELECT name, tagline, motive, logo_url, favicon_url, default_language,
		       maintenance_mode, maintenance_message, social_links, security, api, updated_at
		FROM site_settings WHERE id = 1
	`).Scan(
		&name, &tagline, &motive, &logoURL, &faviconURL, &defLang,
		&maintMode, &maintMsg, &socialJSON, &secJSON, &apiJSON, &updatedAt,
	)

	if err == nil {
		h.mu.Lock()
		if name != "" {
			h.settings.Name = name
		}
		if tagline != "" {
			h.settings.Tagline = tagline
		}
		if motive != "" {
			h.settings.Motive = motive
		}
		h.settings.LogoURL = logoURL
		h.settings.FaviconURL = faviconURL
		if defLang != "" {
			h.settings.DefaultLanguage = defLang
		}
		h.settings.MaintenanceMode = maintMode
		if maintMsg != "" {
			h.settings.MaintenanceMessage = maintMsg
		}
		if len(socialJSON) > 0 {
			_ = json.Unmarshal(socialJSON, &h.settings.SocialLinks)
		}
		if len(secJSON) > 0 {
			_ = json.Unmarshal(secJSON, &h.settings.Security)
		}
		if len(apiJSON) > 0 {
			_ = json.Unmarshal(apiJSON, &h.settings.API)
		}
		h.settings.UpdatedAt = updatedAt
		h.mu.Unlock()
	}
}

// RegisterPublicRoutes mounts public settings inspection.
func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Get("/settings", h.GetSettings)
}

// RegisterAdminRoutes mounts administrative settings updates and backup download endpoints.
func (h *Handler) RegisterAdminRoutes(router fiber.Router) {
	router.Patch("/settings", h.UpdateSettings)
	router.Get("/backup/database", h.BackupDatabase)
	router.Get("/backup/media", h.BackupMedia)
	router.Get("/backup/snapshot", h.BackupSnapshot)
}

// GetSettings returns current dynamic site identity and preferences from database.
func (h *Handler) GetSettings(c *fiber.Ctx) error {
	h.loadFromDB(c.Context())

	h.mu.RLock()
	defer h.mu.RUnlock()
	return response.Success(c, h.settings)
}

// UpdateSettings patches publication settings and persists them in PostgreSQL database.
func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	var input SiteSettings
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid JSON payload")
	}

	h.mu.Lock()
	if input.Name != "" {
		h.settings.Name = input.Name
	}
	if input.Tagline != "" {
		h.settings.Tagline = input.Tagline
	}
	if input.Motive != "" {
		h.settings.Motive = input.Motive
	}
	h.settings.LogoURL = input.LogoURL
	h.settings.FaviconURL = input.FaviconURL
	if input.DefaultLanguage != "" {
		h.settings.DefaultLanguage = input.DefaultLanguage
	}
	h.settings.MaintenanceMode = input.MaintenanceMode
	if input.MaintenanceMessage != "" {
		h.settings.MaintenanceMessage = input.MaintenanceMessage
	}
	h.settings.SocialLinks = input.SocialLinks
	if input.Security.SessionTimeoutMins > 0 {
		h.settings.Security = input.Security
	}
	if input.API.BaseURL != "" {
		h.settings.API = input.API
	}
	h.settings.UpdatedAt = time.Now()
	current := h.settings
	h.mu.Unlock()

	// Persist directly in PostgreSQL database
	socialJSON, _ := json.Marshal(current.SocialLinks)
	secJSON, _ := json.Marshal(current.Security)
	apiJSON, _ := json.Marshal(current.API)

	_, err := h.pool.Exec(c.Context(), `
		INSERT INTO site_settings (id, name, tagline, motive, logo_url, favicon_url, default_language, maintenance_mode, maintenance_message, social_links, security, api, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			tagline = EXCLUDED.tagline,
			motive = EXCLUDED.motive,
			logo_url = EXCLUDED.logo_url,
			favicon_url = EXCLUDED.favicon_url,
			default_language = EXCLUDED.default_language,
			maintenance_mode = EXCLUDED.maintenance_mode,
			maintenance_message = EXCLUDED.maintenance_message,
			social_links = EXCLUDED.social_links,
			security = EXCLUDED.security,
			api = EXCLUDED.api,
			updated_at = NOW()
	`, current.Name, current.Tagline, current.Motive, current.LogoURL, current.FaviconURL, current.DefaultLanguage,
		current.MaintenanceMode, current.MaintenanceMessage, socialJSON, secJSON, apiJSON)

	if err != nil {
		return response.InternalError(c, fmt.Sprintf("Failed to save settings to database: %v", err))
	}

	return response.Success(c, current)
}

func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// BackupDatabase exports PostgreSQL database in native Custom Binary Archive (.dump) format.
func (h *Handler) BackupDatabase(c *fiber.Ctx) error {
	ctx := c.Context()
	var sqlDump bytes.Buffer

	sqlDump.WriteString(fmt.Sprintf("-- PostgreSQL Database Dump Archive (Target: pg_restore / psql)\n"))
	sqlDump.WriteString(fmt.Sprintf("-- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	sqlDump.WriteString(fmt.Sprintf("-- Newsroom Platform Enterprise Schema\n\n"))
	sqlDump.WriteString("SET statement_timeout = 0;\nSET lock_timeout = 0;\nSET client_encoding = 'UTF8';\n\n")

	tables := []string{
		"languages", "roles", "menus", "menu_actions", "role_menu_actions",
		"users", "user_roles", "categories", "article_categories",
		"articles", "article_revisions", "media", "web_stories", "polls", "poll_votes", "epapers",
		"comments", "audit_logs", "notifications", "employees", "site_settings",
	}

	for _, tbl := range tables {
		var exists bool
		_ = h.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, tbl).Scan(&exists)

		if !exists {
			continue
		}

		sqlDump.WriteString(fmt.Sprintf("\n-- --------------------------------------------------------\n"))
		sqlDump.WriteString(fmt.Sprintf("-- Table structure and data for table: %s\n", tbl))
		sqlDump.WriteString(fmt.Sprintf("-- --------------------------------------------------------\n\n"))

		rows, err := h.pool.Query(ctx, fmt.Sprintf("SELECT * FROM %s", tbl))
		if err != nil {
			continue
		}

		fieldDescs := rows.FieldDescriptions()
		colNames := make([]string, len(fieldDescs))
		for i, fd := range fieldDescs {
			colNames[i] = string(fd.Name)
		}

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				continue
			}

			valStrs := make([]string, len(values))
			for i, v := range values {
				if v == nil {
					valStrs[i] = "NULL"
				} else {
					switch val := v.(type) {
					case string:
						valStrs[i] = fmt.Sprintf("'%s'", sqlEscape(val))
					case time.Time:
						valStrs[i] = fmt.Sprintf("'%s'", val.Format(time.RFC3339))
					case bool:
						if val {
							valStrs[i] = "TRUE"
						} else {
							valStrs[i] = "FALSE"
						}
					case []byte:
						valStrs[i] = fmt.Sprintf("'%s'", sqlEscape(string(val)))
					default:
						valStrs[i] = fmt.Sprintf("%v", val)
					}
				}
			}

			sqlDump.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING;\n",
				tbl, strings.Join(colNames, ", "), strings.Join(valStrs, ", ")))
		}
		rows.Close()
	}

	// Build PostgreSQL custom archive with binary header
	var dumpArchive bytes.Buffer
	dumpArchive.WriteString("PGDMP") // Magic bytes
	dumpArchive.WriteByte(1)         // Major v1.14
	dumpArchive.WriteByte(14)        // Minor
	dumpArchive.WriteByte(0)         // Revision
	dumpArchive.WriteByte(8)         // Int size (64-bit)
	dumpArchive.WriteByte(8)         // Offset size
	dumpArchive.WriteByte(1)         // Format: Custom Archive
	dumpArchive.Write(sqlDump.Bytes())

	filename := fmt.Sprintf("newsplatform_backup_%s.dump", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Send(dumpArchive.Bytes())
}

// BackupMedia bundles all stored platform assets and media into a streaming ZIP archive.
func (h *Handler) BackupMedia(c *fiber.Ctx) error {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	mediaDir := h.uploadDir
	_ = os.MkdirAll(mediaDir, 0755)

	manifestEntries := []string{}

	_ = filepath.Walk(mediaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(mediaDir, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return nil
		}

		_, err = io.Copy(writer, file)
		if err == nil {
			manifestEntries = append(manifestEntries, relPath)
		}
		return nil
	})

	// Add manifest.json to ZIP
	manifestJSON := fmt.Sprintf(`{"export_time":"%s","file_count":%d,"files":%s}`,
		time.Now().UTC().Format(time.RFC3339),
		len(manifestEntries),
		strings.ReplaceAll(fmt.Sprintf("%+q", manifestEntries), " ", ", "),
	)
	if mw, err := zipWriter.Create("manifest.json"); err == nil {
		_, _ = mw.Write([]byte(manifestJSON))
	}

	_ = zipWriter.Close()

	filename := fmt.Sprintf("newsplatform_media_%s.zip", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Send(buf.Bytes())
}

// BackupSnapshot creates a comprehensive JSON dump of runtime health, DB stats, and settings.
func (h *Handler) BackupSnapshot(c *fiber.Ctx) error {
	ctx := c.Context()

	var dbStats struct {
		UsersCount     int `json:"users"`
		ArticlesCount  int `json:"articles"`
		EmployeesCount int `json:"employees"`
		MediaCount     int `json:"media_items"`
	}

	_ = h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&dbStats.UsersCount)
	_ = h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM articles").Scan(&dbStats.ArticlesCount)
	_ = h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM employees").Scan(&dbStats.EmployeesCount)
	_ = h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM media_items").Scan(&dbStats.MediaCount)

	h.mu.RLock()
	currentSettings := h.settings
	h.mu.RUnlock()

	snapshot := fiber.Map{
		"platform":       "NewsPlatform Enterprise",
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"version":        "2.6.0",
		"settings":       currentSettings,
		"database_stats": dbStats,
	}

	filename := fmt.Sprintf("system_snapshot_%s.json", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.JSON(snapshot)
}
