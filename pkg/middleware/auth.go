package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"newsplatform/api/pkg/config"
	"newsplatform/api/pkg/response"
)

// Session represents the authenticated user context extracted from a JWT.
// It is attached to the Fiber context and used by downstream handlers and the IAM evaluator.
type Session struct {
	UserID           int64    `json:"user_id"`
	ActiveDistrictID *int64   `json:"active_district_id,omitempty"`
	Roles            []string `json:"roles"`
	IsStaff          bool     `json:"is_staff"`
	IsSuperAdmin     bool     `json:"is_super_admin"`
}

const sessionKey = "session"

// SessionFromCtx retrieves the Session from the Fiber context.
func SessionFromCtx(c *fiber.Ctx) *Session {
	s, ok := c.Locals(sessionKey).(*Session)
	if !ok {
		return nil
	}
	return s
}

// JWTClaims represents the custom JWT claims embedded in access tokens.
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID           int64    `json:"uid"`
	ActiveDistrictID *int64   `json:"did,omitempty"`
	Roles            []string `json:"roles"`
	IsStaff          bool     `json:"staff"`
	IsSuperAdmin     bool     `json:"sa"`
}

// RequireAuth is a Fiber middleware that validates the JWT access token from the
// Authorization header, extracts claims, and attaches a Session to the context.
// It also verifies that the user account is still active in the database.
func RequireAuth(cfg config.JWTConfig, pool ...*pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Unauthorized(c, "Missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return response.Unauthorized(c, "Invalid authorization header format")
		}

		tokenStr := parts[1]

		token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing method")
			}
			return []byte(cfg.Secret), nil
		})

		if err != nil {
			return response.Unauthorized(c, "Invalid or expired token")
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok || !token.Valid {
			return response.Unauthorized(c, "Invalid token claims")
		}

		// SECURITY: Check if user is still active in the database.
		// This ensures deactivated users are rejected even with a valid JWT.
		if len(pool) > 0 && pool[0] != nil {
			var isActive bool
			err := pool[0].QueryRow(c.Context(),
				"SELECT is_active FROM users WHERE id = $1", claims.UserID,
			).Scan(&isActive)
			if err != nil || !isActive {
				return response.Unauthorized(c, "Account is disabled or not found")
			}
		}

		sess := &Session{
			UserID:           claims.UserID,
			ActiveDistrictID: claims.ActiveDistrictID,
			Roles:            claims.Roles,
			IsStaff:          claims.IsStaff,
			IsSuperAdmin:     claims.IsSuperAdmin,
		}

		c.Locals(sessionKey, sess)
		return c.Next()
	}
}

// RequireStaff is a middleware that ensures the user is a staff member.
// Must be placed after RequireAuth.
func RequireStaff() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := SessionFromCtx(c)
		if sess == nil {
			return response.Unauthorized(c, "Not authenticated")
		}
		if !sess.IsStaff && !sess.IsSuperAdmin {
			return response.Forbidden(c, "Staff access required")
		}
		return c.Next()
	}
}

// OptionalAuth tries to extract a session from the JWT but does not block the
// request if no token is present. Useful for public endpoints that behave
// differently for authenticated users (e.g., showing personalized content).
func OptionalAuth(cfg config.JWTConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Next()
		}

		token, err := jwt.ParseWithClaims(parts[1], &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing method")
			}
			return []byte(cfg.Secret), nil
		})

		if err == nil {
			if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
				c.Locals(sessionKey, &Session{
					UserID:           claims.UserID,
					ActiveDistrictID: claims.ActiveDistrictID,
					Roles:            claims.Roles,
					IsStaff:          claims.IsStaff,
					IsSuperAdmin:     claims.IsSuperAdmin,
				})
			}
		}

		return c.Next()
	}
}
