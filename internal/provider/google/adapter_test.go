package google

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "google",
		APIKey:  "test-key",
		Enabled: true,
	}

	p := Factory(cfg)
	require.NotNil(t, p)
	assert.Equal(t, "google", p.Name())
	assert.NotEmpty(t, p.Models())
}

func TestConvertRequest(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "gemini-3.1-pro-preview",
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   512,
	}

	gReq := ConvertRequest(req)
	require.NotNil(t, gReq.SystemInstruction)
	assert.Equal(t, "You are helpful", gReq.SystemInstruction.Parts[0].Text)
	assert.Len(t, gReq.Contents, 1)
	assert.Equal(t, "user", gReq.Contents[0].Role)
	assert.Equal(t, "Hello", gReq.Contents[0].Parts[0].Text)
	require.NotNil(t, gReq.GenerationConfig)
	assert.Equal(t, 0.7, gReq.GenerationConfig.Temperature)
	assert.Equal(t, 512, gReq.GenerationConfig.MaxOutputTokens)
}

func TestConvertResponse(t *testing.T) {
	raw := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: Content{Parts: []Part{{Text: "Hi there"}}},
			},
		},
		UsageMetadata: UsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 20,
			TotalTokenCount:      30,
		},
	}

	resp := ConvertResponse(raw, "gemini-3.1-pro-preview")
	require.NotNil(t, resp)
	assert.Equal(t, "gemini-3.1-pro-preview", resp.Model)
	assert.Equal(t, 1, len(resp.Choices))
	assert.Equal(t, "Hi there", resp.Choices[0].Message.Content)
	assert.Equal(t, 30, resp.Usage.TotalTokens)
}

func TestAdapter_Disabled(t *testing.T) {
	a := NewAdapter(&provider.ProviderConfig{Name: "google", APIKey: "k", Enabled: false})
	_, err := a.Chat(context.Background(), &provider.ChatRequest{Model: "gemini-3.1-pro-preview"})
	require.Error(t, err)
	provErr, ok := err.(*provider.ProviderError)
	require.True(t, ok)
	assert.Equal(t, 503, provErr.Code)
}

func TestAdapter_StreamChat_PassesThroughSSEError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-3.1-pro-preview:streamGenerateContent" {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(`{"error":{"code":404,"message":"not found"}}`))
			require.NoError(t, err)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "event: error\n")
		_, _ = fmt.Fprint(w, "data: {\"error\":{\"code\":429,\"message\":\"quota exceeded\",\"status\":\"RESOURCE_EXHAUSTED\"}}\n\n")
	}))
	defer server.Close()

	a := NewAdapter(&provider.ProviderConfig{
		Name:    "google",
		APIKey:  "test-key",
		BaseURL: server.URL + "/v1beta",
		Models:  []string{"gemini-3.1-pro-preview"},
		Enabled: true,
	})

	stream, err := a.StreamChat(context.Background(), &provider.ChatRequest{
		Model: "gemini-3.1-pro-preview",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "hello"},
		},
		Stream: true,
	})
	require.NoError(t, err)

	chunk, ok := <-stream
	require.True(t, ok)
	require.NotNil(t, chunk)
	require.NotNil(t, chunk.Error)
	assert.Equal(t, 429, chunk.Error.Code)
	assert.Equal(t, "quota exceeded", chunk.Error.Message)
	assert.True(t, chunk.Done)
}
