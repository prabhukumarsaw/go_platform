package employee

import (
	"time"
)

// Department represents a newsroom division.
type Department string

const (
	DeptEditorial   Department = "Editorial"
	DeptVideo       Department = "Video"
	DeptDesk        Department = "Desk"
	DeptBureau      Department = "Bureau"
	DeptFactCheck   Department = "FactCheck"
	DeptSocialMedia Department = "SocialMedia"
)

// Employee represents a staff journalist, editor, or bureau reporter.
type Employee struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	TenantID     int        `json:"tenant_id"`
	TenantName   string     `json:"tenant_name,omitempty"`
	EmployeeCode string     `json:"employee_code"`
	DisplayName  string     `json:"display_name"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	Department   Department `json:"department"`
	Designation  string     `json:"designation"`
	DistrictID   *int       `json:"district_id,omitempty"`
	DistrictName string     `json:"district_name,omitempty"`
	RoleName     string     `json:"role_name,omitempty"`
	IsActive     bool       `json:"is_active"`
	ArticleCount int64      `json:"article_count"`
	TotalViews   int64      `json:"total_views"`
	JoinedAt     time.Time  `json:"joined_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// OnboardEmployeeInput contains the data needed to onboard newsroom staff.
type OnboardEmployeeInput struct {
	UserID       int64      `json:"user_id"`
	TenantID     int        `json:"tenant_id"`
	EmployeeCode string     `json:"employee_code"`
	Department   Department `json:"department"`
	Designation  string     `json:"designation"`
	DistrictID   *int       `json:"district_id"`
	RoleID       int        `json:"role_id"`
}
