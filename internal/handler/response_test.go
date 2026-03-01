package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"message": "hello"}
	Success(c, data)

	require.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

func TestSuccessWithStatus(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"id": "123"}
	SuccessWithStatus(c, http.StatusCreated, data)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusBadRequest, "invalid_request", "Invalid parameter")

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "invalid_request", resp.Error.Code)
	assert.Equal(t, "Invalid parameter", resp.Error.Message)
}

func TestErrorWithDetail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ErrorWithDetail(c, http.StatusBadGateway, "provider_error", "Provider failed", "Connection refused")

	require.Equal(t, http.StatusBadGateway, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "provider_error", resp.Error.Code)
	assert.Equal(t, "Provider failed", resp.Error.Message)
	assert.Equal(t, "Connection refused", resp.Error.Detail)
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "Missing required field")

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, ErrCodeInvalidRequest, resp.Error.Code)
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c, "API key is invalid")

	require.Equal(t, http.StatusUnauthorized, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, ErrCodeUnauthorized, resp.Error.Code)
}

func TestRateLimited(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	RateLimited(c)

	require.Equal(t, http.StatusTooManyRequests, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, ErrCodeRateLimitExceed, resp.Error.Code)
	assert.Equal(t, "Rate limit exceeded", resp.Error.Message)
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalError(c, "Database connection failed")

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, ErrCodeInternalError, resp.Error.Code)
}

func TestProviderError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ProviderError(c, "Provider request failed", "Timeout after 30s")

	require.Equal(t, http.StatusBadGateway, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, ErrCodeProviderError, resp.Error.Code)
	assert.Equal(t, "Timeout after 30s", resp.Error.Detail)
}

func TestErrorCodes(t *testing.T) {
	assert.Equal(t, "invalid_request", ErrCodeInvalidRequest)
	assert.Equal(t, "unauthorized", ErrCodeUnauthorized)
	assert.Equal(t, "rate_limit_exceeded", ErrCodeRateLimitExceed)
	assert.Equal(t, "provider_error", ErrCodeProviderError)
	assert.Equal(t, "internal_error", ErrCodeInternalError)
	assert.Equal(t, "model_not_found", ErrCodeModelNotFound)
}

func TestResponse_Structure(t *testing.T) {
	// Test Response struct
	resp := Response{
		Success: true,
		Data:    map[string]int{"count": 42},
	}
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Nil(t, resp.Error)

	// Test Response with error
	errResp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    "test_error",
			Message: "Test error message",
			Detail:  "Test detail",
		},
	}
	assert.False(t, errResp.Success)
	assert.Nil(t, errResp.Data)
	assert.NotNil(t, errResp.Error)
}

func TestErrorInfo_Structure(t *testing.T) {
	info := ErrorInfo{
		Code:    "error_code",
		Message: "Error message",
		Detail:  "Error detail",
	}

	assert.Equal(t, "error_code", info.Code)
	assert.Equal(t, "Error message", info.Message)
	assert.Equal(t, "Error detail", info.Detail)
}
