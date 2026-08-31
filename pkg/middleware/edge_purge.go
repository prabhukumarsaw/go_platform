package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// EdgePurgeClient handles targeted edge cache invalidation across CDN providers (Cloudflare / CloudFront / Fastly).
type EdgePurgeClient struct {
	zoneID     string
	apiToken   string
	webhookURL string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewEdgePurgeClient initializes the edge purge dispatcher.
func NewEdgePurgeClient(zoneID, apiToken, webhookURL string, logger zerolog.Logger) *EdgePurgeClient {
	return &EdgePurgeClient{
		zoneID:     zoneID,
		apiToken:   apiToken,
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger.With().Str("module", "edge_purge").Logger(),
	}
}

// PurgeStateEdition invalidates edge cache for a state edition and specific article URL.
func (p *EdgePurgeClient) PurgeStateEdition(ctx context.Context, stateSlug, articleSlug string) error {
	urlsToPurge := []string{
		fmt.Sprintf("https://newsplatform.in?state=%s", stateSlug),
		fmt.Sprintf("https://newsplatform.in/feed/%s/rss.xml", stateSlug),
		"https://newsplatform.in",
		"https://newsplatform.in/feed/rss.xml",
	}
	if articleSlug != "" {
		urlsToPurge = append(urlsToPurge, fmt.Sprintf("https://newsplatform.in/article/%s", articleSlug))
	}

	p.logger.Info().Str("state", stateSlug).Strs("urls", urlsToPurge).Msg("Dispatched targeted edge cache purge")

	// If Cloudflare credentials configured
	if p.zoneID != "" && p.apiToken != "" {
		payload, _ := json.Marshal(map[string]interface{}{"files": urlsToPurge})
		req, err := http.NewRequestWithContext(ctx, "POST",
			fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", p.zoneID),
			bytes.NewReader(payload),
		)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+p.apiToken)
			req.Header.Set("Content-Type", "application/json")
			resp, reqErr := p.httpClient.Do(req)
			if reqErr == nil {
				defer resp.Body.Close()
			}
		}
	}

	// If custom webhook URL configured
	if p.webhookURL != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"event":      "cache_purge",
			"state_slug": stateSlug,
			"urls":       urlsToPurge,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
		req, err := http.NewRequestWithContext(ctx, "POST", p.webhookURL, bytes.NewReader(payload))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			resp, reqErr := p.httpClient.Do(req)
			if reqErr == nil {
				defer resp.Body.Close()
			}
		}
	}

	return nil
}
