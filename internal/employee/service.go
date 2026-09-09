package employee

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"newsplatform/api/pkg/errors"
	"github.com/rs/zerolog"
)

type Service struct {
	repo   *Repository
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		repo:   NewRepository(pool),
		pool:   pool,
		logger: logger.With().Str("module", "employee").Logger(),
	}
}

func (s *Service) ListEmployees(ctx context.Context, tx pgx.Tx, department, search string) ([]Employee, error) {
	return s.repo.ListEmployees(ctx, tx, department, search)
}

func (s *Service) GetEmployeeByID(ctx context.Context, tx pgx.Tx, employeeID int64) (*Employee, error) {
	return s.repo.GetEmployeeByID(ctx, tx, employeeID)
}

func (s *Service) OnboardEmployee(ctx context.Context, tx pgx.Tx, input OnboardEmployeeInput) (*Employee, error) {
	if input.UserID <= 0 && input.Email == "" {
		return nil, errors.BadRequest("Valid user_id or email is required", nil)
	}
	if input.DisplayName == "" && input.UserID <= 0 {
		return nil, errors.BadRequest("Full legal name is required", nil)
	}
	if input.Designation == "" {
		return nil, errors.BadRequest("Designation is required", nil)
	}

	emp, err := s.repo.OnboardEmployee(ctx, tx, input)
	if err != nil {
		return nil, fmt.Errorf("onboard employee: %w", err)
	}

	return emp, nil
}

func (s *Service) UpdateStatus(ctx context.Context, tx pgx.Tx, employeeID int64, isActive bool) error {
	return s.repo.UpdateStatus(ctx, tx, employeeID, isActive)
}

func (s *Service) UpdateEmployee(ctx context.Context, tx pgx.Tx, employeeID int64, input UpdateEmployeeInput) (*Employee, error) {
	return s.repo.UpdateEmployee(ctx, tx, employeeID, input)
}

func (s *Service) AssignRole(ctx context.Context, tx pgx.Tx, employeeID int64, roleID int) error {
	return s.repo.AssignRole(ctx, tx, employeeID, roleID)
}

func (s *Service) NextEmployeeCode(ctx context.Context, tx pgx.Tx) (string, error) {
	return s.repo.NextEmployeeCode(ctx, tx)
}

func (s *Service) DeleteEmployee(ctx context.Context, tx pgx.Tx, employeeID int64) error {
	return s.repo.Delete(ctx, tx, employeeID)
}
