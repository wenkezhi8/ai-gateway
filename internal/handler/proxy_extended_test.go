package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-gateway/internal/provider"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestChatCompletions_InvalidJSON tests invalid JSON handling
func TestChatCompletions_InvalidJSON(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	tests := []struct {
		name       string
		body       string
		expectCode int
	}{
		{
			name:       "empty body",
			body:       "",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       "{invalid}",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "missing model",
			body:       `{"messages": [{"role": "user", "content": "test"}]}`,
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "missing messages",
			body:       `{"model": "gpt-4"}`,
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "empty messages",
			body:       `{"model": "gpt-4", "messages": []}`,
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			h.ChatCompletions(c)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

// TestChatCompletions_MultimodalContent tests multimodal content handling
func TestChatCompletions_MultimodalContent(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	// Register mock provider
	mockP := &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com", []string{"gpt-4o"}, true),
	}
	provider.RegisterProvider("openai", mockP)

	// Test with multimodal content array
	body := `{
		"model": "gpt-4o",
		"messages": [{
			"role": "user",
			"content": [
				{"type": "text", "text": "What is in this image?"},
				{"type": "image_url", "image_url": {"url": "https://example.com/image.jpg"}}
			]
		}]
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	// Should accept multimodal format
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable)
}

// TestChatCompletions_Stream tests streaming response
func TestChatCompletions_Stream(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	mockP := &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com", []string{"gpt-4"}, true),
	}
	provider.RegisterProvider("openai", mockP)

	body := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"stream": true
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	// Streaming request should be handled
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable)
}

// TestGetTextContent tests content extraction helper
func TestGetTextContent(t *testing.T) {
	tests := []struct {
		name     string
		content  interface{}
		expected string
	}{
		{
			name:     "string content",
			content:  "hello world",
			expected: "hello world",
		},
		{
			name: "content array with text",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "describe this"},
				map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "http://..."}},
			},
			expected: "describe this",
		},
		{
			name: "content array without text",
			content: []interface{}{
				map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "http://..."}},
			},
			expected: "",
		},
		{
			name:     "nil content",
			content:  nil,
			expected: "",
		},
		{
			name:     "number content",
			content:  123,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTextContent(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestProxyHandler_WithAccountManager tests with account manager
func TestProxyHandler_WithAccountManager(t *testing.T) {
	cfg := testConfig()

	// AccountManager needs RedisStore and logger
	// In test mode, we can pass nil for the account manager
	h := NewProxyHandler(cfg, nil, nil)
	assert.NotNil(t, h)
}

// TestChatCompletions_TooLargeBody tests request body size limit
func TestChatCompletions_TooLargeBody(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	// Create a very large body
	largeContent := strings.Repeat("x", 20*1024*1024) // 20MB
	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "` + largeContent + `"}]}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	// Should reject large bodies
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusRequestEntityTooLarge)
}

// TestListModels tests model listing
func TestListModels(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/v1/models", nil)
	c.Request = req

	h.ListModels(c)

	// Should return 200 even without providers registered
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable)
}

// TestChatCompletions_WithContextCancellation tests context cancellation
func TestChatCompletions_WithContextCancellation(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	// Request should fail due to cancelled context
	assert.True(t, w.Code >= 400 || w.Code == 0)
}

// TestChatCompletions_WithTemperature tests temperature parameter
func TestChatCompletions_WithTemperature(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	mockP := &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com", []string{"gpt-4"}, true),
	}
	provider.RegisterProvider("openai", mockP)

	tests := []struct {
		name        string
		temperature float64
		expectValid bool
	}{
		{"zero temperature", 0, true},
		{"normal temperature", 0.7, true},
		{"max temperature", 2.0, true},
		{"negative temperature", -0.1, false},
		{"too high temperature", 2.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"model":       "gpt-4",
				"messages":    []map[string]string{{"role": "user", "content": "Hello"}},
				"temperature": tt.temperature,
			}

			bodyBytes, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("POST", "/api/v1/chat/completions", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			h.ChatCompletions(c)

			if tt.expectValid {
				assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable)
			}
		})
	}
}

// TestRecordMetrics tests metrics recording
func TestRecordMetrics(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	// This should not panic
	h.recordMetrics("openai", "gpt-4", "gpt-4", 100*time.Millisecond, 100, true)
	h.recordMetrics("", "", "", 0, 0, false)
}

// TestChatCompletions_Concurrent tests concurrent requests
func TestChatCompletions_Concurrent(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	mockP := &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com", []string{"gpt-4"}, true),
	}
	provider.RegisterProvider("openai", mockP)

	// Run multiple concurrent requests
	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			h.ChatCompletions(c)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent request timeout")
		}
	}
}

// TestValidateChatRequest tests request validation
func TestValidateChatRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     ChatCompletionRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: ChatCompletionRequest{
				Model:    "gpt-4",
				Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: ChatCompletionRequest{
				Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
			},
			wantErr: true,
		},
		{
			name: "missing messages",
			req: ChatCompletionRequest{
				Model: "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "empty messages",
			req: ChatCompletionRequest{
				Model:    "gpt-4",
				Messages: []ChatMessage{},
			},
			wantErr: true,
		},
		{
			name: "message without role",
			req: ChatCompletionRequest{
				Model:    "gpt-4",
				Messages: []ChatMessage{{Content: "Hello"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Mock io.Reader for testing read errors
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// TestChatCompletions_ReadError tests handling of read errors
func TestChatCompletions_ReadError(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", &errorReader{})
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
