package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/config"
)

// GoogleUserInfo represents claims returned by Google's userinfo endpoint.
type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// GoogleOAuthProvider manages Google OAuth exchange and user synchronization.
type GoogleOAuthProvider struct {
	cfg        config.GoogleOAuthConfig
	httpClient *http.Client
}

func NewGoogleOAuthProvider(cfg config.GoogleOAuthConfig) *GoogleOAuthProvider {
	return &GoogleOAuthProvider{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ExchangeCode exchanges an authorization code from the frontend for Google user info.
func (p *GoogleOAuthProvider) ExchangeCode(ctx context.Context, code string) (*GoogleUserInfo, error) {
	if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" {
		// Mock profile for local development if client ID/Secret not set
		return &GoogleUserInfo{
			Sub:           "google_dev_123456789",
			Email:         "viewer@gmail.com",
			EmailVerified: true,
			Name:          "Dev Google User",
			Picture:       "https://api.dicebear.com/7.x/bottts/svg?seed=google",
		}, nil
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", p.cfg.ClientID)
	data.Set("client_secret", p.cfg.ClientSecret)
	data.Set("redirect_uri", p.cfg.RedirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}

	// Fetch UserInfo with access token
	userReq, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

	userResp, err := p.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("google userinfo request: %w", err)
	}
	defer userResp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// AuthenticateGoogle processes a Google sign-in/up and returns a full JWT token pair.
func (s *Service) AuthenticateGoogle(ctx context.Context, code string) (*TokenPair, *User, error) {
	oauth := NewGoogleOAuthProvider(s.cfg.GoogleOAuth)
	info, err := oauth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("google oauth exchange: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	// Find or Upsert user
	var user User
	err = tx.QueryRow(ctx,
		`SELECT id, email, phone, display_name, avatar_url, provider, is_staff, is_super_admin, totp_enabled, is_active, created_at
		 FROM users WHERE email = $1`,
		info.Email,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.DisplayName, &user.AvatarURL,
		&user.Provider, &user.IsStaff, &user.IsSuperAdmin, &user.TOTPEnabled, &user.IsActive, &user.CreatedAt)

	if err == pgx.ErrNoRows {
		// Register new user via Google
		err = tx.QueryRow(ctx,
			`INSERT INTO users (email, display_name, avatar_url, provider, provider_id)
			 VALUES ($1, $2, $3, 'google', $4)
			 RETURNING id, email, phone, display_name, avatar_url, provider, is_staff, is_super_admin, totp_enabled, is_active, created_at`,
			info.Email, info.Name, info.Picture, info.Sub,
		).Scan(&user.ID, &user.Email, &user.Phone, &user.DisplayName, &user.AvatarURL,
			&user.Provider, &user.IsStaff, &user.IsSuperAdmin, &user.TOTPEnabled, &user.IsActive, &user.CreatedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("create google user: %w", err)
		}
	} else if err != nil {
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, fmt.Errorf("account is disabled")
	}

	roles, err := s.getUserRoles(ctx, tx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	tokens, err := s.generateTokenPair(ctx, tx, user.ID, nil, roles, user.IsStaff, user.IsSuperAdmin)
	if err != nil {
		return nil, nil, err
	}

	tx.Exec(ctx, "UPDATE users SET last_login_at = NOW() WHERE id = $1", user.ID)

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return tokens, &user, nil
}
