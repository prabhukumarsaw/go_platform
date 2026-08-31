package auth

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/config"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for authentication.
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers auth routes (public and session-authenticated).
func (h *Handler) RegisterRoutes(router fiber.Router, jwtCfg config.JWTConfig) {
	auth := router.Group("/auth")

	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
	auth.Post("/refresh", h.Refresh)
	auth.Post("/logout", h.Logout)
	auth.Post("/forgot-password", h.ForgotPassword)
	auth.Post("/reset-password", h.ResetPassword)
	auth.Post("/otp/send", h.SendOTP)
	auth.Post("/otp/verify", h.VerifyOTP)
	auth.Post("/switch-tenant", h.SwitchTenant)
	auth.Post("/google/callback", h.GoogleCallback)

	// Authenticated routes
	authReq := auth.Group("", middleware.RequireAuth(jwtCfg))
	authReq.Get("/me", h.GetMe)
	authReq.Get("/sessions", h.ListSessions)
	authReq.Delete("/sessions/:id", h.RevokeSession)
	authReq.Get("/activity", h.ListActivity)
	authReq.Get("/dpdp/export", h.ExportUserData)
	authReq.Post("/dpdp/delete-account", h.DeleteAccount)
	authReq.Post("/avatar", h.UploadAvatar)
	authReq.Delete("/avatar", h.DeleteAvatar)
	authReq.Post("/totp/setup", h.SetupTOTP)
	authReq.Post("/totp/verify", h.VerifyTOTP)
}

// ─── Handlers ───────────────────────────────────

// Register creates a new user account.
func (h *Handler) Register(c *fiber.Ctx) error {
	var input RegisterInput
	if err := c.BodyParser(&input); err != nil {
		if jsonErr := json.Unmarshal(c.Body(), &input); jsonErr != nil {
			return response.BadRequest(c, "Invalid request body")
		}
	}

	if input.Email == "" && input.Phone == "" {
		return response.BadRequest(c, "Email or phone is required")
	}
	if input.Provider == "" {
		input.Provider = "email"
	}

	user, err := h.service.Register(c.Context(), input)
	if err != nil {
		return response.InternalError(c, "Registration failed: "+err.Error())
	}

	return response.Created(c, user)
}

// Login authenticates with email and password.
func (h *Handler) Login(c *fiber.Ctx) error {
	var input LoginInput
	_ = c.BodyParser(&input)
	if input.Email == "" || input.Password == "" {
		_ = json.Unmarshal(c.Body(), &input)
	}

	if input.Email == "" || input.Password == "" {
		return response.BadRequest(c, "Email and password are required")
	}

	tokens, user, err := h.service.LoginWithEmail(c.Context(), input)
	if err != nil {
		return response.Unauthorized(c, "Invalid credentials")
	}

	return response.Success(c, fiber.Map{
		"tokens": tokens,
		"user":   user,
	})
}

// RefreshRequest is the request body for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh issues a new token pair using a valid refresh token.
func (h *Handler) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.RefreshToken == "" {
		return response.BadRequest(c, "Refresh token is required")
	}

	tokens, err := h.service.RefreshTokens(c.Context(), req.RefreshToken)
	if err != nil {
		return response.Unauthorized(c, "Invalid or expired refresh token")
	}

	return response.Success(c, tokens)
}

// Logout revokes all refresh tokens for the authenticated user.
func (h *Handler) Logout(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	if err := h.service.Logout(c.Context(), sess.UserID); err != nil {
		return response.InternalError(c, "Logout failed")
	}

	return response.Success(c, fiber.Map{"message": "Logged out successfully"})
}

// SendOTPRequest is the request body for OTP send.
type SendOTPRequest struct {
	Phone string `json:"phone"`
}

// SendOTP generates and sends a phone OTP.
func (h *Handler) SendOTP(c *fiber.Ctx) error {
	var req SendOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Phone == "" {
		return response.BadRequest(c, "Phone number is required")
	}

	if err := h.service.SendOTP(c.Context(), req.Phone); err != nil {
		return response.InternalError(c, "Failed to send OTP")
	}

	return response.Success(c, fiber.Map{"message": "OTP sent successfully"})
}

// VerifyOTPRequest is the request body for OTP verification.
type VerifyOTPRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

// VerifyOTP validates a phone OTP and returns tokens.
func (h *Handler) VerifyOTP(c *fiber.Ctx) error {
	var req VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Phone == "" || req.OTP == "" {
		return response.BadRequest(c, "Phone and OTP are required")
	}

	tokens, user, err := h.service.VerifyOTP(c.Context(), req.Phone, req.OTP)
	if err != nil {
		return response.Unauthorized(c, "Invalid or expired OTP")
	}

	return response.Success(c, fiber.Map{
		"tokens": tokens,
		"user":   user,
	})
}

// SwitchTenantRequest is the payload to change the user's active tenant context.
type SwitchTenantRequest struct {
	TargetTenantID int64 `json:"target_tenant_id"`
}

// SwitchTenant generates a fresh JWT with the new active tenant claim.
func (h *Handler) SwitchTenant(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	var req SwitchTenantRequest
	if err := c.BodyParser(&req); err != nil || req.TargetTenantID == 0 {
		return response.BadRequest(c, "Valid target_tenant_id is required")
	}

	tokens, err := h.service.SwitchTenant(c.Context(), sess.UserID, req.TargetTenantID)
	if err != nil {
		return response.Forbidden(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Tenant switched successfully",
		"tokens":  tokens,
	})
}

// SetupTOTP initiates TOTP generation for employee accounts.
func (h *Handler) SetupTOTP(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	res, err := h.service.SetupTOTP(c.Context(), sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to setup TOTP: "+err.Error())
	}

	return response.Success(c, res)
}

// VerifyTOTP validates a 6-digit code to enable TOTP MFA.
func (h *Handler) VerifyTOTP(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	var body struct {
		Passcode string `json:"passcode"`
	}
	if err := c.BodyParser(&body); err != nil || body.Passcode == "" {
		return response.BadRequest(c, "Passcode is required")
	}

	valid, err := h.service.VerifyTOTP(c.Context(), sess.UserID, body.Passcode)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	if !valid {
		return response.BadRequest(c, "Invalid TOTP passcode")
	}

	return response.Success(c, fiber.Map{"message": "TOTP MFA enabled successfully"})
}

// GoogleCallback processes OAuth authorization codes from Google.
func (h *Handler) GoogleCallback(c *fiber.Ctx) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil || body.Code == "" {
		return response.BadRequest(c, "Authorization code is required")
	}

	tokens, user, err := h.service.AuthenticateGoogle(c.Context(), body.Code)
	if err != nil {
		return response.Unauthorized(c, "Google authentication failed: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"tokens": tokens,
		"user":   user,
	})
}

// ForgotPassword handles password reset email requests.
func (h *Handler) ForgotPassword(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil || body.Email == "" {
		return response.BadRequest(c, "Valid email address is required")
	}

	_ = h.service.ForgotPassword(c.Context(), body.Email, c.IP(), c.Get("User-Agent"))
	return response.Success(c, fiber.Map{
		"message": "If the email is registered, a password reset link has been dispatched.",
	})
}

// ResetPassword applies the new password using the reset token.
func (h *Handler) ResetPassword(c *fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil || body.Token == "" || len(body.NewPassword) < 6 {
		return response.BadRequest(c, "Token and new password (min 6 characters) are required")
	}

	if err := h.service.ResetPassword(c.Context(), body.Token, body.NewPassword, c.IP(), c.Get("User-Agent")); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Password reset successfully. You can now login."})
}

// GetMe returns the authenticated user profile and newsroom employee details.
func (h *Handler) GetMe(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	user, err := h.service.GetUserByID(c.Context(), sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to load user profile")
	}

	return response.Success(c, fiber.Map{
		"user":    user,
		"session": sess,
	})
}

// ListSessions returns all active device sessions for the authenticated user.
func (h *Handler) ListSessions(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	sessions, err := h.service.ListSessions(c.Context(), sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to load sessions: "+err.Error())
	}

	return response.Success(c, sessions)
}

// RevokeSession deletes a specific active device session.
func (h *Handler) RevokeSession(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid session ID")
	}

	if err := h.service.RevokeSession(c.Context(), sess.UserID, id); err != nil {
		return response.InternalError(c, "Failed to revoke session")
	}

	return response.Success(c, fiber.Map{"message": "Session revoked"})
}

// ListActivity returns security and authentication audit logs for the user.
func (h *Handler) ListActivity(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	logs, err := h.service.ListActivityLogs(c.Context(), sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to load activity logs: "+err.Error())
	}

	return response.Success(c, logs)
}

// ExportUserData downloads a full machine-readable JSON archive of the user's data (DPDP Act).
func (h *Handler) ExportUserData(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	data, err := h.service.ExportUserData(c.Context(), sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to export user data: "+err.Error())
	}

	return response.Success(c, data)
}

// DeleteAccount processes user data erasure under DPDP Act right to be forgotten.
func (h *Handler) DeleteAccount(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil || body.Password == "" {
		return response.BadRequest(c, "Current password is required to confirm account deletion")
	}

	if err := h.service.DeleteAccount(c.Context(), sess.UserID, body.Password); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Account personal data anonymized and sessions terminated successfully."})
}

// UploadAvatar handles profile image uploads, cropping/compressing and linking to the user account.
func (h *Handler) UploadAvatar(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return response.BadRequest(c, "No avatar image provided (form field 'avatar')")
	}

	src, err := file.Open()
	if err != nil {
		return response.InternalError(c, "Failed to read avatar file")
	}
	defer src.Close()

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	avatarURL, err := h.service.UploadAvatar(c.Context(), sess.UserID, file.Filename, mimeType, src)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message":    "Avatar updated successfully",
		"avatar_url": avatarURL,
	})
}

// DeleteAvatar resets the user's avatar to default.
func (h *Handler) DeleteAvatar(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	if err := h.service.DeleteAvatar(c.Context(), sess.UserID); err != nil {
		return response.InternalError(c, "Failed to remove avatar")
	}

	return response.Success(c, fiber.Map{"message": "Avatar removed"})
}

