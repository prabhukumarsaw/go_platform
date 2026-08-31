package jobs

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RSSFeed represents an XML RSS news agency wire feed.
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title string    `xml:"title"`
	Items []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Category    string `xml:"category"`
}

// WireFeedSource defines a news agency endpoint.
type WireFeedSource struct {
	AgencyName string // PTI, ANI, Reuters
	URL        string
	TenantID   int    // default 1 = National
	Language   string // en, hi
}

// IngestWireFeeds polls configured news wire feeds and drafts stories in the database.
func (s *Scheduler) IngestWireFeeds(ctx context.Context) error {
	sources := []WireFeedSource{
		{AgencyName: "PTI", URL: "https://timesofindia.indiatimes.com/rssfeeds/-2128936835.cms", TenantID: 1, Language: "en"},
		{AgencyName: "National Wire", URL: "https://www.thehindu.com/news/national/feeder/default.rss", TenantID: 1, Language: "en"},
	}

	for _, src := range sources {
		if err := s.ingestSingleSource(ctx, src); err != nil {
			s.logger.Warn().Err(err).Str("agency", src.AgencyName).Msg("Wire feed poll skipped or failed")
		}
	}
	return nil
}

func (s *Scheduler) ingestSingleSource(ctx context.Context, src WireFeedSource) error {
	req, err := http.NewRequestWithContext(ctx, "GET", src.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "BharatVani-NewsBot/1.0")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, src.URL)
	}

	var feed RSSFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return err
	}

	ingestedCount := 0
	for _, item := range feed.Channel.Items {
		if item.Title == "" {
			continue
		}

		// Generate clean slug
		cleanTitle := strings.TrimSpace(item.Title)
		slug := strings.ToLower(cleanTitle)
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == ' ' {
				return r
			}
			return -1
		}, slug)
		slug = strings.Join(strings.Fields(slug), "-")
		if len(slug) > 180 {
			slug = slug[:180]
		}
		slug += "-" + uuid.New().String()[:6]

		// Format block body
		bodyJSON, _ := json.Marshal([]map[string]interface{}{
			{"type": "paragraph", "text": item.Description},
			{"type": "paragraph", "text": fmt.Sprintf("[Source: %s Wire Dispatch]", src.AgencyName)},
		})

		// Check if article with same title already exists
		var exists bool
		_ = s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM articles WHERE title = $1)", cleanTitle).Scan(&exists)
		if exists {
			continue
		}

		// Insert wire dispatch as draft ready for editorial desk review
		query := `
			INSERT INTO articles
				(tenant_id, language, title, slug, body, excerpt, status, author_id, is_national, meta_title, meta_description)
			VALUES ($1, $2, $3, $4, $5, $6, 'draft', 1, TRUE, $7, $8)
			ON CONFLICT DO NOTHING
		`

		metaTitle := cleanTitle
		if len(metaTitle) > 190 {
			metaTitle = metaTitle[:190]
		}

		excerpt := item.Description
		if len(excerpt) > 480 {
			excerpt = excerpt[:480]
		}

		_, err := s.pool.Exec(ctx, query,
			src.TenantID, src.Language, cleanTitle, slug, bodyJSON, excerpt,
			metaTitle, excerpt,
		)
		if err == nil {
			ingestedCount++
		}
	}

	if ingestedCount > 0 {
		s.logger.Info().Int("count", ingestedCount).Str("agency", src.AgencyName).Msg("Ingested wire agency articles into draft pool")
	}

	return nil
}
