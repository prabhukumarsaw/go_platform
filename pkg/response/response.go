package response

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	appErrors "newsplatform/api/pkg/errors"
)

// Standard API response envelope.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// Success sends a 200 JSON response with data.
func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}

// Created sends a 201 JSON response with data.
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}

// Paginated sends a 200 JSON response with data and pagination metadata.
func Paginated(c *fiber.Ctx, data interface{}, page, perPage int, total int64) error {
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	return c.Status(fiber.StatusOK).JSON(APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Error sends an error JSON response.
func Error(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, appErrors.CodeBadRequest, message)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnauthorized, appErrors.CodeUnauthorized, message)
}

func Forbidden(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusForbidden, appErrors.CodeForbidden, message)
}

func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusNotFound, appErrors.CodeNotFound, message)
}

func InternalError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusInternalServerError, appErrors.CodeInternal, message)
}

// GlobalErrorHandler handles all unhandled errors, domain AppErrors, and Fiber errors uniformly.
func GlobalErrorHandler(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	// 1. Check if it's our domain AppError
	if appErr, ok := appErrors.As(err); ok {
		return c.Status(appErr.StatusCode).JSON(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			},
		})
	}

	// 2. Check if it's a Fiber built-in error
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "HTTP_ERROR",
				Message: fiberErr.Message,
			},
		})
	}

	// 3. Fallback to generic 500
	return c.Status(fiber.StatusInternalServerError).JSON(APIResponse{
		Success: false,
		Error: &APIError{
			Code:    appErrors.CodeInternal,
			Message: "Internal server error occurred",
		},
	})
}
