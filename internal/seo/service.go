package seo

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service handles sitemap generation, structured data, and SEO utilities.
type Service struct {
	pool    *pgxpool.Pool
	logger  zerolog.Logger
	baseURL string
}

// NewService creates a new SEO service.
func NewService(pool *pgxpool.Pool, baseURL string, logger zerolog.Logger) *Service {
	return &Service{
		pool:    pool,
		baseURL: strings.TrimRight(baseURL, "/"),
		logger:  logger.With().Str("module", "seo").Logger(),
	}
}

// ─── Sitemap XML ────────────────────────────────

// SitemapURL represents a single URL entry in a sitemap.
type SitemapURL struct {
	XMLName    xml.Name `xml:"url"`
	Loc        string   `xml:"loc"`
	LastMod    string   `xml:"lastmod,omitempty"`
	ChangeFreq string   `xml:"changefreq,omitempty"`
	Priority   string   `xml:"priority,omitempty"`
}

// SitemapIndex represents the top-level sitemap index.
type SitemapIndex struct {
	XMLName  xml.Name        `xml:"sitemapindex"`
	XMLNS    string          `xml:"xmlns,attr"`
	Sitemaps []SitemapEntry  `xml:"sitemap"`
}

// SitemapEntry is a single sitemap reference in the index.
type SitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// GoogleNewsURLSet is the root element of a Google News sitemap file.
type GoogleNewsURLSet struct {
	XMLName xml.Name          `xml:"urlset"`
	XMLNS   string            `xml:"xmlns,attr"`
	NewsNS  string            `xml:"xmlns:news,attr"`
	URLs    []GoogleNewsEntry `xml:"url"`
}

// GoogleNewsEntry is a Google News compliant URL element.
type GoogleNewsEntry struct {
	Loc  string            `xml:"loc"`
	News GoogleNewsPayload `xml:"news:news"`
}

// GoogleNewsPayload contains Google News publication metadata.
type GoogleNewsPayload struct {
	Publication      GoogleNewsPublication `xml:"news:publication"`
	PublicationDate  string                `xml:"news:publication_date"`
	Title            string                `xml:"news:title"`
}

// GoogleNewsPublication holds publication title and language.
type GoogleNewsPublication struct {
	Name     string `xml:"news:name"`
	Language string `xml:"news:language"`
}

// URLSet is the root element of a sitemap file.
type URLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// GenerateSitemapIndex creates the top-level sitemap index listing per-tenant sitemaps.
func (s *Service) GenerateSitemapIndex(ctx context.Context) ([]byte, error) {
	query := `SELECT slug FROM tenants WHERE is_active = TRUE ORDER BY name`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fetch tenants for sitemap: %w", err)
	}
	defer rows.Close()

	index := SitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	now := time.Now().Format("2006-01-02")
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		index.Sitemaps = append(index.Sitemaps, SitemapEntry{
			Loc:     fmt.Sprintf("%s/sitemaps/%s.xml", s.baseURL, slug),
			LastMod: now,
		})
	}

	output, err := xml.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal sitemap index: %w", err)
	}
	return append([]byte(xml.Header), output...), nil
}

// GenerateTenantSitemap creates a sitemap for a specific tenant's published articles.
func (s *Service) GenerateTenantSitemap(ctx context.Context, tenantSlug string) ([]byte, error) {
	query := `
		SELECT a.slug, a.language, a.updated_at, a.is_breaking
		FROM articles a
		JOIN tenants t ON t.id = a.tenant_id
		WHERE t.slug = $1 AND a.status = 'published'
		ORDER BY a.published_at DESC
		LIMIT 50000
	`

	rows, err := s.pool.Query(ctx, query, tenantSlug)
	if err != nil {
		return nil, fmt.Errorf("fetch articles for sitemap: %w", err)
	}
	defer rows.Close()

	urlset := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for rows.Next() {
		var slug, lang string
		var updatedAt time.Time
		var isBreaking bool

		if err := rows.Scan(&slug, &lang, &updatedAt, &isBreaking); err != nil {
			return nil, err
		}

		priority := "0.6"
		changeFreq := "weekly"
		if isBreaking {
			priority = "0.9"
			changeFreq = "hourly"
		}

		urlset.URLs = append(urlset.URLs, SitemapURL{
			Loc:        fmt.Sprintf("%s/%s/%s?lang=%s", s.baseURL, tenantSlug, slug, lang),
			LastMod:    updatedAt.Format("2006-01-02"),
			ChangeFreq: changeFreq,
			Priority:   priority,
		})
	}

	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal tenant sitemap: %w", err)
	}
	return append([]byte(xml.Header), output...), nil
}

// ─── Structured Data (JSON-LD) ──────────────────

// NewsArticleLD generates schema.org NewsArticle JSON-LD for an article.
type NewsArticleLD struct {
	Context         string      `json:"@context"`
	Type            string      `json:"@type"`
	Headline        string      `json:"headline"`
	Description     string      `json:"description,omitempty"`
	Image           string      `json:"image,omitempty"`
	DatePublished   string      `json:"datePublished,omitempty"`
	DateModified    string      `json:"dateModified,omitempty"`
	Author          AuthorLD    `json:"author"`
	Publisher       PublisherLD `json:"publisher"`
	MainEntityOfPage string    `json:"mainEntityOfPage"`
	InLanguage      string      `json:"inLanguage"`
}

// AuthorLD represents schema.org Person.
type AuthorLD struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

// PublisherLD represents schema.org Organization.
type PublisherLD struct {
	Type string `json:"@type"`
	Name string `json:"name"`
	Logo LogoLD `json:"logo,omitempty"`
}

// LogoLD represents schema.org ImageObject.
type LogoLD struct {
	Type string `json:"@type"`
	URL  string `json:"url"`
}

// BuildNewsArticleLD creates structured data for a published article.
func (s *Service) BuildNewsArticleLD(title, description, image, authorName, language, slug, tenantSlug string, publishedAt, updatedAt *time.Time) NewsArticleLD {
	ld := NewsArticleLD{
		Context:          "https://schema.org",
		Type:             "NewsArticle",
		Headline:         title,
		Description:      description,
		Image:            image,
		Author:           AuthorLD{Type: "Person", Name: authorName},
		Publisher:        PublisherLD{Type: "Organization", Name: "Hybrid News Platform"},
		MainEntityOfPage: fmt.Sprintf("%s/%s/%s", s.baseURL, tenantSlug, slug),
		InLanguage:       language,
	}

	if publishedAt != nil {
		ld.DatePublished = publishedAt.Format(time.RFC3339)
	}
	if updatedAt != nil {
		ld.DateModified = updatedAt.Format(time.RFC3339)
	}

	return ld
}

// ─── RSS 2.0 Feed Generator ─────────────────────

type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author,omitempty"`
}

// GenerateRSSFeed produces standard RSS 2.0 XML for Google News and syndication.
func (s *Service) GenerateRSSFeed(ctx context.Context, tenantSlug string) ([]byte, error) {
	query := `
		SELECT a.title, a.slug, COALESCE(a.excerpt, ''), a.published_at, COALESCE(u.display_name, 'Editorial Team'), t.name
		FROM articles a
		JOIN tenants t ON t.id = a.tenant_id
		LEFT JOIN users u ON u.id = a.author_id
		WHERE a.status = 'published'
	`
	args := []interface{}{}
	if tenantSlug != "" && tenantSlug != "all" {
		query += ` AND (t.slug = $1 OR a.is_national = TRUE)`
		args = append(args, tenantSlug)
	}
	query += ` ORDER BY a.published_at DESC LIMIT 30`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RSSItem
	channelTitle := "BharatVani News Platform"
	for rows.Next() {
		var title, slug, excerpt, author, tName string
		var pubDate time.Time
		if err := rows.Scan(&title, &slug, &excerpt, &pubDate, &author, &tName); err == nil {
			link := fmt.Sprintf("%s/article/%s", s.baseURL, slug)
			items = append(items, RSSItem{
				Title:       title,
				Link:        link,
				Description: excerpt,
				PubDate:     pubDate.Format(time.RFC1123Z),
				GUID:        link,
				Author:      author,
			})
			if tenantSlug != "" {
				channelTitle = fmt.Sprintf("BharatVani News - %s Edition", tName)
			}
		}
	}

	feed := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       channelTitle,
			Link:        s.baseURL,
			Description: "Latest breaking news, politics, business, and regional updates across India",
			Language:    "hi-IN",
			Items:       items,
		},
	}

	output, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

// GenerateGoogleNewsSitemap creates a Google News compliant sitemap for stories published in last 48 hours.
func (s *Service) GenerateGoogleNewsSitemap(ctx context.Context) ([]byte, error) {
	query := `
		SELECT a.title, a.slug, a.language, a.published_at
		FROM articles a
		WHERE a.status = 'published' AND a.published_at >= NOW() - INTERVAL '48 HOURS'
		ORDER BY a.published_at DESC
		LIMIT 1000
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query google news articles: %w", err)
	}
	defer rows.Close()

	urlset := GoogleNewsURLSet{
		XMLNS:  "http://www.sitemaps.org/schemas/sitemap/0.9",
		NewsNS: "http://www.google.com/schemas/sitemap-news/0.9",
	}

	for rows.Next() {
		var title, slug, lang string
		var pubDate time.Time
		if err := rows.Scan(&title, &slug, &lang, &pubDate); err == nil {
			if lang == "" {
				lang = "en"
			}
			urlset.URLs = append(urlset.URLs, GoogleNewsEntry{
				Loc: fmt.Sprintf("%s/article/%s", s.baseURL, slug),
				News: GoogleNewsPayload{
					Publication: GoogleNewsPublication{
						Name:     "BharatVani News",
						Language: lang,
					},
					PublicationDate: pubDate.Format(time.RFC3339),
					Title:           title,
				},
			})
		}
	}

	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal google news sitemap: %w", err)
	}

	return append([]byte(xml.Header), output...), nil
}
