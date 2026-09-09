package employee

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s",
		hex.EncodeToString(salt), hex.EncodeToString(hash))
}

func (r *Repository) ListEmployees(ctx context.Context, tx pgx.Tx, department, search string) ([]Employee, error) {
	query := `
		SELECT DISTINCT ON (e.id) e.id, e.user_id,
		       e.employee_code, u.display_name, COALESCE(u.email, ''), COALESCE(u.phone, ''), COALESCE(u.avatar_url, ''),
		       e.department, e.designation,
		       ro.id as role_id, COALESCE(ro.name, '') as role_name,
		       COALESCE(e.address, ''), COALESCE(e.pin_code, ''), COALESCE(e.bio, ''), COALESCE(e.press_card_no, ''), COALESCE(e.x_handle, ''),
		       (e.is_active AND u.is_active) as is_active,
		       COALESCE(ac.count, 0) as article_count,
		       COALESCE(vc.views, 0) as total_views,
		       e.joined_at, e.created_at
		FROM employees e
		JOIN users u ON u.id = e.user_id
		LEFT JOIN user_roles ur ON ur.user_id = e.user_id AND ur.is_active = TRUE
		LEFT JOIN roles ro ON ro.id = ur.role_id
		LEFT JOIN (
			SELECT author_id, COUNT(*) as count FROM articles GROUP BY author_id
		) ac ON ac.author_id = e.user_id
		LEFT JOIN (
			SELECT author_id, SUM(view_count) as views FROM articles GROUP BY author_id
		) vc ON vc.author_id = e.user_id
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if department != "" && department != "all" {
		query += fmt.Sprintf(" AND e.department = $%d", argIdx)
		args = append(args, department)
		argIdx++
	}

	if search != "" && search != "all" {
		s := "%" + strings.TrimSpace(search) + "%"
		query += fmt.Sprintf(" AND (u.display_name ILIKE $%d OR u.email ILIKE $%d OR e.employee_code ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, s)
		argIdx++
	}

	query += " ORDER BY e.id ASC, ro.id ASC NULLS LAST"

	var rows pgx.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(ctx, query, args...)
	} else {
		rows, err = r.pool.Query(ctx, query, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Employee
	for rows.Next() {
		var emp Employee
		if err := rows.Scan(
			&emp.ID, &emp.UserID,
			&emp.EmployeeCode, &emp.DisplayName, &emp.Email, &emp.Phone, &emp.AvatarURL,
			&emp.Department, &emp.Designation,
			&emp.RoleID, &emp.RoleName,
			&emp.Address, &emp.PinCode, &emp.Bio, &emp.PressCardNo, &emp.XHandle,
			&emp.IsActive, &emp.ArticleCount, &emp.TotalViews,
			&emp.JoinedAt, &emp.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, emp)
	}

	if list == nil {
		list = []Employee{}
	}

	return list, rows.Err()
}

func (r *Repository) GetEmployeeByID(ctx context.Context, tx pgx.Tx, employeeID int64) (*Employee, error) {
	query := `
		SELECT e.id, e.user_id,
		       e.employee_code, u.display_name, COALESCE(u.email, ''), COALESCE(u.phone, ''), COALESCE(u.avatar_url, ''),
		       e.department, e.designation,
		       ro.id as role_id, COALESCE(ro.name, '') as role_name,
		       COALESCE(e.address, ''), COALESCE(e.pin_code, ''), COALESCE(e.bio, ''), COALESCE(e.press_card_no, ''), COALESCE(e.x_handle, ''),
		       e.is_active,
		       COALESCE(ac.count, 0) as article_count,
		       COALESCE(vc.views, 0) as total_views,
		       e.joined_at, e.created_at
		FROM employees e
		JOIN users u ON u.id = e.user_id
		LEFT JOIN user_roles ur ON ur.user_id = e.user_id AND ur.is_active = TRUE
		LEFT JOIN roles ro ON ro.id = ur.role_id
		LEFT JOIN (
			SELECT author_id, COUNT(*) as count FROM articles GROUP BY author_id
		) ac ON ac.author_id = e.user_id
		LEFT JOIN (
			SELECT author_id, SUM(view_count) as views FROM articles GROUP BY author_id
		) vc ON vc.author_id = e.user_id
		WHERE e.id = $1
	`
	var emp Employee
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, employeeID).Scan(
			&emp.ID, &emp.UserID,
			&emp.EmployeeCode, &emp.DisplayName, &emp.Email, &emp.Phone, &emp.AvatarURL,
			&emp.Department, &emp.Designation,
			&emp.RoleID, &emp.RoleName,
			&emp.Address, &emp.PinCode, &emp.Bio, &emp.PressCardNo, &emp.XHandle,
			&emp.IsActive, &emp.ArticleCount, &emp.TotalViews,
			&emp.JoinedAt, &emp.CreatedAt,
		)
	} else {
		err = r.pool.QueryRow(ctx, query, employeeID).Scan(
			&emp.ID, &emp.UserID,
			&emp.EmployeeCode, &emp.DisplayName, &emp.Email, &emp.Phone, &emp.AvatarURL,
			&emp.Department, &emp.Designation,
			&emp.RoleID, &emp.RoleName,
			&emp.Address, &emp.PinCode, &emp.Bio, &emp.PressCardNo, &emp.XHandle,
			&emp.IsActive, &emp.ArticleCount, &emp.TotalViews,
			&emp.JoinedAt, &emp.CreatedAt,
		)
	}
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

func (r *Repository) OnboardEmployee(ctx context.Context, tx pgx.Tx, input OnboardEmployeeInput) (*Employee, error) {
	// 1. Ensure user exists in users table
	if input.UserID <= 0 && input.Email != "" {
		var userID int64
		// Check if user already exists
		err := tx.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", input.Email).Scan(&userID)
		if err == nil {
			// Update existing user to staff
			_, err = tx.Exec(ctx, `
				UPDATE users SET
					display_name = $1,
					phone = COALESCE(NULLIF($2, ''), phone),
					avatar_url = COALESCE(NULLIF($3, ''), avatar_url),
					is_staff = TRUE,
					is_active = TRUE
				WHERE id = $4
			`, input.DisplayName, input.Phone, input.AvatarURL, userID)
			if err != nil {
				return nil, fmt.Errorf("update user: %w", err)
			}
			input.UserID = userID
		} else {
			// Create new user with synchronized sequence
			var pwdHash *string
			if input.Password != "" {
				h := hashPassword(input.Password)
				pwdHash = &h
			}

			_, _ = tx.Exec(ctx, "SELECT setval('users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM users) + 1, false)")

			err = tx.QueryRow(ctx, `
				INSERT INTO users (email, display_name, phone, password_hash, avatar_url, is_staff, is_active)
				VALUES ($1, $2, $3, $4, $5, TRUE, TRUE)
				RETURNING id
			`, input.Email, input.DisplayName, input.Phone, pwdHash, input.AvatarURL).Scan(&userID)
			if err != nil {
				return nil, fmt.Errorf("create user: %w", err)
			}
			input.UserID = userID
		}
	}

	if input.EmployeeCode == "" {
		code, err := r.NextEmployeeCode(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("generate employee code: %w", err)
		}
		input.EmployeeCode = code
	}

	// 2. Insert or update employee record
	_, _ = tx.Exec(ctx, "SELECT setval('employees_id_seq', (SELECT COALESCE(MAX(id), 1) FROM employees) + 1, false)")

	query := `
		INSERT INTO employees (user_id, employee_code, department, designation, address, pin_code, bio, press_card_no, x_handle)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			employee_code = EXCLUDED.employee_code,
			department = EXCLUDED.department,
			designation = EXCLUDED.designation,
			address = EXCLUDED.address,
			pin_code = EXCLUDED.pin_code,
			bio = EXCLUDED.bio,
			press_card_no = EXCLUDED.press_card_no,
			x_handle = EXCLUDED.x_handle,
			is_active = TRUE,
			updated_at = NOW()
		RETURNING id, user_id, employee_code, department, designation, address, pin_code, bio, press_card_no, x_handle, is_active, joined_at, created_at
	`

	var emp Employee
	err := tx.QueryRow(ctx, query,
		input.UserID, input.EmployeeCode, input.Department, input.Designation,
		input.Address, input.PinCode, input.Bio, input.PressCardNo, input.XHandle,
	).Scan(
		&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.Department, &emp.Designation,
		&emp.Address, &emp.PinCode, &emp.Bio, &emp.PressCardNo, &emp.XHandle,
		&emp.IsActive, &emp.JoinedAt, &emp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert employee: %w", err)
	}

	emp.DisplayName = input.DisplayName
	emp.Email = input.Email
	emp.Phone = input.Phone
	emp.AvatarURL = input.AvatarURL

	// 3. Assign staff role mapping in user_roles (single active role)
	if input.RoleID > 0 {
		_, _ = tx.Exec(ctx, `UPDATE user_roles SET is_active = FALSE WHERE user_id = $1`, input.UserID)
		_, _ = tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id, is_active, assigned_by)
			VALUES ($1, $2, TRUE, $1)
			ON CONFLICT (user_id, role_id) DO UPDATE SET is_active = TRUE
		`, input.UserID, input.RoleID)
		emp.RoleID = &input.RoleID
	}

	// 4. Set is_staff = TRUE on users table
	_, _ = tx.Exec(ctx, "UPDATE users SET is_staff = TRUE WHERE id = $1", input.UserID)

	return &emp, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, tx pgx.Tx, employeeID int64, isActive bool) error {
	// Update employee record
	var employeeErr error
	if tx != nil {
		_, employeeErr = tx.Exec(ctx, "UPDATE employees SET is_active = $1, updated_at = NOW() WHERE id = $2", isActive, employeeID)
	} else {
		_, employeeErr = r.pool.Exec(ctx, "UPDATE employees SET is_active = $1, updated_at = NOW() WHERE id = $2", isActive, employeeID)
	}
	if employeeErr != nil {
		return employeeErr
	}

	// SECURITY: also update users.is_active so the user cannot login / refresh tokens
	q := `UPDATE users SET is_active = $1 WHERE id = (
		SELECT user_id FROM employees WHERE id = $2
	)`
	if tx != nil {
		_, employeeErr = tx.Exec(ctx, q, isActive, employeeID)
	} else {
		_, employeeErr = r.pool.Exec(ctx, q, isActive, employeeID)
	}
	if employeeErr != nil {
		return employeeErr
	}

	// Kill existing sessions so a deactivated user cannot stay signed in
	revoke := `UPDATE refresh_tokens SET is_revoked = TRUE WHERE user_id = (
		SELECT user_id FROM employees WHERE id = $1
	)`
	if tx != nil {
		_, _ = tx.Exec(ctx, revoke, employeeID)
	} else {
		_, _ = r.pool.Exec(ctx, revoke, employeeID)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, tx pgx.Tx, employeeID int64) error {
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, "DELETE FROM employees WHERE id = $1", employeeID)
	} else {
		_, err = r.pool.Exec(ctx, "DELETE FROM employees WHERE id = $1", employeeID)
	}
	return err
}

// UpdateEmployee updates editable profile fields on an existing employee.
func (r *Repository) UpdateEmployee(ctx context.Context, tx pgx.Tx, employeeID int64, input UpdateEmployeeInput) (*Employee, error) {
	// Update users table for display_name, phone, avatar_url
	userQuery := `
		UPDATE users SET
			display_name = COALESCE(NULLIF($1, ''), display_name),
			phone = COALESCE(NULLIF($2, ''), phone),
			avatar_url = COALESCE(NULLIF($3, ''), avatar_url)
		WHERE id = (SELECT user_id FROM employees WHERE id = $4)
	`
	if tx != nil {
		_, _ = tx.Exec(ctx, userQuery, input.DisplayName, input.Phone, input.AvatarURL, employeeID)
	} else {
		_, _ = r.pool.Exec(ctx, userQuery, input.DisplayName, input.Phone, input.AvatarURL, employeeID)
	}

	// Update employees table
	empQuery := `
		UPDATE employees SET
			department = COALESCE(NULLIF($1, ''), department),
			designation = COALESCE(NULLIF($2, ''), designation),
			address = COALESCE(NULLIF($3, ''), address),
			pin_code = COALESCE(NULLIF($4, ''), pin_code),
			bio = COALESCE(NULLIF($5, ''), bio),
			press_card_no = COALESCE(NULLIF($6, ''), press_card_no),
			x_handle = COALESCE(NULLIF($7, ''), x_handle),
			updated_at = NOW()
		WHERE id = $8
		RETURNING id
	`
	var returnedID int64
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, empQuery,
			input.Department, input.Designation,
			input.Address, input.PinCode, input.Bio,
			input.PressCardNo, input.XHandle, employeeID,
		).Scan(&returnedID)
	} else {
		err = r.pool.QueryRow(ctx, empQuery,
			input.Department, input.Designation,
			input.Address, input.PinCode, input.Bio,
			input.PressCardNo, input.XHandle, employeeID,
		).Scan(&returnedID)
	}
	if err != nil {
		return nil, fmt.Errorf("update employee: %w", err)
	}

	if input.RoleID != nil && *input.RoleID > 0 {
		if err := r.AssignRole(ctx, tx, employeeID, *input.RoleID); err != nil {
			return nil, fmt.Errorf("assign role: %w", err)
		}
	}

	return r.GetEmployeeByID(ctx, tx, employeeID)
}

func (r *Repository) NextEmployeeCode(ctx context.Context, tx pgx.Tx) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("EMP-%d-", year)
	var next int
	q := `
		SELECT COALESCE(MAX(
			NULLIF(regexp_replace(employee_code, '^EMP-[0-9]{4}-', ''), '')::int
		), 0) + 1
		FROM employees
		WHERE employee_code LIKE $1
	`
	like := prefix + "%"
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, q, like).Scan(&next)
	} else {
		err = r.pool.QueryRow(ctx, q, like).Scan(&next)
	}
	if err != nil {
		next = 1
	}
	return fmt.Sprintf("EMP-%d-%04d", year, next), nil
}

// AssignRole assigns a new IAM role to an employee (replaces existing role).
func (r *Repository) AssignRole(ctx context.Context, tx pgx.Tx, employeeID int64, roleID int) error {
	// First get the user_id for this employee
	var userID int64
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, "SELECT user_id FROM employees WHERE id = $1", employeeID).Scan(&userID)
	} else {
		err = r.pool.QueryRow(ctx, "SELECT user_id FROM employees WHERE id = $1", employeeID).Scan(&userID)
	}
	if err != nil {
		return fmt.Errorf("employee not found: %w", err)
	}

	// Deactivate previous roles
	if tx != nil {
		_, err = tx.Exec(ctx, "UPDATE user_roles SET is_active = FALSE WHERE user_id = $1", userID)
	} else {
		_, err = r.pool.Exec(ctx, "UPDATE user_roles SET is_active = FALSE WHERE user_id = $1", userID)
	}
	if err != nil {
		return err
	}

	// Assign new role
	q := `INSERT INTO user_roles (user_id, role_id, is_active, assigned_by)
		VALUES ($1, $2, TRUE, 1)
		ON CONFLICT (user_id, role_id) DO UPDATE SET is_active = TRUE, assigned_by = 1`
	if tx != nil {
		_, err = tx.Exec(ctx, q, userID, roleID)
	} else {
		_, err = r.pool.Exec(ctx, q, userID, roleID)
	}
	return err
}
