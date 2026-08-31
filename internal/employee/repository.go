package employee

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListEmployees(ctx context.Context, tx pgx.Tx, tenantID int, department string) ([]Employee, error) {
	query := `
		SELECT e.id, e.user_id, e.tenant_id, COALESCE(t.name, ''),
		       e.employee_code, u.display_name, COALESCE(u.email, ''), COALESCE(u.phone, ''),
		       e.department, e.designation, e.district_id, COALESCE(d.name, ''),
		       COALESCE(ro.name, ''), e.is_active,
		       COALESCE(ac.count, 0) as article_count,
		       COALESCE(vc.views, 0) as total_views,
		       e.joined_at, e.created_at
		FROM employees e
		JOIN users u ON u.id = e.user_id
		JOIN tenants t ON t.id = e.tenant_id
		LEFT JOIN districts d ON d.id = e.district_id
		LEFT JOIN user_tenant_mappings utm ON utm.user_id = e.user_id AND utm.tenant_id = e.tenant_id
		LEFT JOIN roles ro ON ro.id = utm.role_id
		LEFT JOIN (
			SELECT author_id, COUNT(*) as count FROM articles GROUP BY author_id
		) ac ON ac.author_id = e.user_id
		LEFT JOIN (
			SELECT author_id, SUM(view_count) as views FROM articles GROUP BY author_id
		) vc ON vc.author_id = e.user_id
		WHERE (e.tenant_id = $1 OR $1 = 1)
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if department != "" {
		query += fmt.Sprintf(" AND e.department = $%d", argIdx)
		args = append(args, department)
		argIdx++
	}

	query += " ORDER BY e.created_at DESC"

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Employee
	for rows.Next() {
		var emp Employee
		if err := rows.Scan(
			&emp.ID, &emp.UserID, &emp.TenantID, &emp.TenantName,
			&emp.EmployeeCode, &emp.DisplayName, &emp.Email, &emp.Phone,
			&emp.Department, &emp.Designation, &emp.DistrictID, &emp.DistrictName,
			&emp.RoleName, &emp.IsActive, &emp.ArticleCount, &emp.TotalViews,
			&emp.JoinedAt, &emp.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, emp)
	}

	return list, rows.Err()
}

func (r *Repository) OnboardEmployee(ctx context.Context, tx pgx.Tx, input OnboardEmployeeInput) (*Employee, error) {
	query := `
		INSERT INTO employees (user_id, tenant_id, employee_code, department, designation, district_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, tenant_id, employee_code, department, designation, district_id, is_active, joined_at, created_at
	`

	var emp Employee
	err := tx.QueryRow(ctx, query, input.UserID, input.TenantID, input.EmployeeCode, input.Department, input.Designation, input.DistrictID).
		Scan(&emp.ID, &emp.UserID, &emp.TenantID, &emp.EmployeeCode, &emp.Department, &emp.Designation, &emp.DistrictID, &emp.IsActive, &emp.JoinedAt, &emp.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Assign tenant role mapping
	if input.RoleID > 0 {
		_, _ = tx.Exec(ctx, `
			INSERT INTO user_tenant_mappings (user_id, tenant_id, role_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, tenant_id) DO UPDATE SET role_id = EXCLUDED.role_id
		`, input.UserID, input.TenantID, input.RoleID)
	}

	// Set is_staff = TRUE on users table
	_, _ = tx.Exec(ctx, "UPDATE users SET is_staff = TRUE WHERE id = $1", input.UserID)

	return &emp, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, tx pgx.Tx, employeeID int64, isActive bool) error {
	_, err := tx.Exec(ctx, "UPDATE employees SET is_active = $1, updated_at = NOW() WHERE id = $2", isActive, employeeID)
	return err
}
