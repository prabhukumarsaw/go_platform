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

// Employee represents a staff journalist, editor, or bureau reporter with full profile details.
type Employee struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	EmployeeCode string     `json:"employee_code"`
	DisplayName  string     `json:"display_name"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	AvatarURL    string     `json:"avatar_url"`
	Department   Department `json:"department"`
	Designation  string     `json:"designation"`
	RoleID       *int       `json:"role_id,omitempty"`
	RoleName     string     `json:"role_name,omitempty"`
	Address      string     `json:"address"`
	PinCode      string     `json:"pin_code"`
	Bio          string     `json:"bio"`
	PressCardNo  string     `json:"press_card_no"`
	XHandle      string     `json:"x_handle"`
	IsActive     bool       `json:"is_active"`
	ArticleCount int64      `json:"article_count"`
	TotalViews   int64      `json:"total_views"`
	JoinedAt     time.Time  `json:"joined_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// OnboardEmployeeInput contains the data needed to onboard and persist newsroom staff.
type OnboardEmployeeInput struct {
	UserID       int64      `json:"user_id"`
	DisplayName  string     `json:"display_name"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	Password     string     `json:"password"`
	AvatarURL    string     `json:"avatar_url"`
	EmployeeCode string     `json:"employee_code"`
	Department   Department `json:"department"`
	Designation  string     `json:"designation"`
	RoleID       int        `json:"role_id"`
	Address      string     `json:"address"`
	PinCode      string     `json:"pin_code"`
	Bio          string     `json:"bio"`
	PressCardNo  string     `json:"press_card_no"`
	XHandle      string     `json:"x_handle"`
}
