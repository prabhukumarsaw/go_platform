package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service handles tenant and district operations.
type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewService creates a new tenant service.
func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "tenant").Logger(),
	}
}

// ─── Models ─────────────────────────────────────

// Tenant represents a state edition.
type Tenant struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	IsNational bool      `json:"is_national"`
	LogoURL    string    `json:"logo_url"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// District represents a district within a state tenant.
type District struct {
	ID       int    `json:"id"`
	TenantID int    `json:"tenant_id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive bool   `json:"is_active"`
}

// ─── Tenant CRUD ────────────────────────────────

// ListTenants returns all tenants.
func (s *Service) ListTenants(ctx context.Context) ([]Tenant, error) {
	query := `SELECT id, name, slug, is_national, COALESCE(logo_url, ''), is_active, created_at
			  FROM tenants ORDER BY is_national DESC, name`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.IsNational, &t.LogoURL, &t.IsActive, &t.CreatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

// GetTenantBySlug returns a tenant by slug.
func (s *Service) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	query := `SELECT id, name, slug, is_national, COALESCE(logo_url, ''), is_active, created_at
			  FROM tenants WHERE slug = $1`

	var t Tenant
	err := s.pool.QueryRow(ctx, query, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.IsNational, &t.LogoURL, &t.IsActive, &t.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return &t, nil
}

// CreateTenant creates a new tenant.
func (s *Service) CreateTenant(ctx context.Context, name, slug string) (*Tenant, error) {
	query := `INSERT INTO tenants (name, slug) VALUES ($1, $2)
			  RETURNING id, name, slug, is_national, COALESCE(logo_url, ''), is_active, created_at`

	var t Tenant
	err := s.pool.QueryRow(ctx, query, name, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.IsNational, &t.LogoURL, &t.IsActive, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return &t, nil
}

// ─── District CRUD ──────────────────────────────

// ListDistricts returns all districts for a tenant.
func (s *Service) ListDistricts(ctx context.Context, tenantID int) ([]District, error) {
	query := `SELECT id, tenant_id, name, slug, is_active FROM districts WHERE tenant_id = $1 ORDER BY name`
	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list districts: %w", err)
	}
	defer rows.Close()

	var districts []District
	for rows.Next() {
		var d District
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.Slug, &d.IsActive); err != nil {
			return nil, err
		}
		districts = append(districts, d)
	}
	return districts, rows.Err()
}

// CreateDistrict creates a new district in a tenant.
func (s *Service) CreateDistrict(ctx context.Context, tenantID int, name, slug string) (*District, error) {
	query := `INSERT INTO districts (tenant_id, name, slug) VALUES ($1, $2, $3)
			  RETURNING id, tenant_id, name, slug, is_active`

	var d District
	err := s.pool.QueryRow(ctx, query, tenantID, name, slug).Scan(
		&d.ID, &d.TenantID, &d.Name, &d.Slug, &d.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("create district: %w", err)
	}
	return &d, nil
}
