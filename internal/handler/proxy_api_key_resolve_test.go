package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeAPIKeyValidator struct {
	validKeys map[string]bool
	calls     []string
}

func (f *fakeAPIKeyValidator) ValidateAPIKey(apiKey string) bool {
	f.calls = append(f.calls, apiKey)
	return f.validKeys[apiKey]
}

func TestResolveAPIKeyFromRequestWithHandler_ShouldValidateXAPIKeyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := &fakeAPIKeyValidator{validKeys: map[string]bool{"sk-test": true}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", http.NoBody)
	req.Header.Set("X-API-Key", "sk-test")
	c.Request = req

	resolved := resolveAPIKeyFromRequestWithHandler(c, validator)

	assert.Equal(t, "sk-test", resolved)
	assert.Equal(t, []string{"sk-test"}, validator.calls)
}

func TestResolveAPIKeyFromRequestWithHandler_ShouldFallbackToAuthorizationBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := &fakeAPIKeyValidator{validKeys: map[string]bool{"sk-auth": true}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", http.NoBody)
	req.Header.Set("Authorization", "Bearer sk-auth")
	c.Request = req

	resolved := resolveAPIKeyFromRequestWithHandler(c, validator)

	assert.Equal(t, "sk-auth", resolved)
	assert.Equal(t, []string{"sk-auth"}, validator.calls)
}

func TestResolveAPIKeyFromRequestWithHandler_ShouldSkipValidationWhenContextAlreadyHasAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := &fakeAPIKeyValidator{validKeys: map[string]bool{"sk-context": true}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", http.NoBody)
	c.Request = req
	c.Set("api_key", "sk-context")

	resolved := resolveAPIKeyFromRequestWithHandler(c, validator)

	assert.Equal(t, "sk-context", resolved)
	assert.Empty(t, validator.calls)
}

func TestResolveAPIKeyFromRequestWithHandler_ShouldReturnEmptyWhenValidationFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := &fakeAPIKeyValidator{validKeys: map[string]bool{"sk-other": true}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", http.NoBody)
	req.Header.Set("X-API-Key", "sk-invalid")
	c.Request = req

	resolved := resolveAPIKeyFromRequestWithHandler(c, validator)

	assert.Equal(t, "", resolved)
	assert.Equal(t, []string{"sk-invalid"}, validator.calls)
}
