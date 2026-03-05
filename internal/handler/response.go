//nolint:godot
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Standard error codes
const (
	ErrCodeInvalidRequest  = "invalid_request"
	ErrCodeUnauthorized    = "unauthorized"
	ErrCodeRateLimitExceed = "rate_limit_exceeded"
	ErrCodeProviderError   = "provider_error"
	ErrCodeInternalError   = "internal_error"
	ErrCodeModelNotFound   = "model_not_found"
	ErrCodeModelNotReg     = "model_not_registered"
)

// Success returns a successful response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithStatus returns a successful response with custom status
func SuccessWithStatus(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
	})
}

// Error returns an error response
func Error(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetail returns an error response with detail
func ErrorWithDetail(c *gin.Context, statusCode int, code, message, detail string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
	})
}

// BadRequest returns a 400 error
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, ErrCodeInvalidRequest, message)
}

// Unauthorized returns a 401 error
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// RateLimited returns a 429 error
func RateLimited(c *gin.Context) {
	Error(c, http.StatusTooManyRequests, ErrCodeRateLimitExceed, "Rate limit exceeded")
}

// InternalError returns a 500 error
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, ErrCodeInternalError, message)
}

// ProviderError returns a provider-related error
func ProviderError(c *gin.Context, message, detail string) {
	ErrorWithDetail(c, http.StatusBadGateway, ErrCodeProviderError, message, detail)
}
