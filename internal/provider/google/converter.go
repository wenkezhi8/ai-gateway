package google

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/provider"
)

type GenerateContentRequest struct {
	Contents          []Content         `json:"contents,omitempty"`
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
}

type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text,omitempty"`
}

type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type GenerateContentResponse struct {
	Candidates    []Candidate   `json:"candidates,omitempty"`
	UsageMetadata UsageMetadata `json:"usageMetadata,omitempty"`
	Error         *APIError     `json:"error,omitempty"`
}

type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason,omitempty"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status,omitempty"`
}

func ConvertRequest(req *provider.ChatRequest) *GenerateContentRequest {
	gReq := &GenerateContentRequest{}

	for _, msg := range req.Messages {
		text := extractText(msg.Content)
		if text == "" {
			continue
		}

		if msg.Role == "system" {
			gReq.SystemInstruction = &Content{Parts: []Part{{Text: text}}}
			continue
		}

		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		gReq.Contents = append(gReq.Contents, Content{
			Role:  role,
			Parts: []Part{{Text: text}},
		})
	}

	config := &GenerationConfig{}
	if req.Temperature > 0 {
		config.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		config.MaxOutputTokens = req.MaxTokens
	}
	if config.Temperature > 0 || config.MaxOutputTokens > 0 {
		gReq.GenerationConfig = config
	}

	return gReq
}

func ConvertResponse(resp *GenerateContentResponse, model string) *provider.ChatResponse {
	content := extractCandidateText(resp.Candidates)
	finish := ""
	if len(resp.Candidates) > 0 {
		finish = normalizeFinishReason(resp.Candidates[0].FinishReason)
	}

	result := &provider.ChatResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []provider.Choice{{
			Index: 0,
			Message: provider.ChatMessage{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: finish,
		}},
		Usage: provider.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}

	if resp.Error != nil {
		code := resp.Error.Code
		if code == 0 {
			code = http.StatusBadGateway
		}
		result.Error = &provider.ProviderError{
			Code:      code,
			Message:   resp.Error.Message,
			Provider:  "google",
			Retryable: isRetryableError(code),
		}
	}

	return result
}

func ConvertStreamChunk(resp *GenerateContentResponse, model string, done bool) *provider.StreamChunk {
	text := extractCandidateText(resp.Candidates)
	finish := ""
	if len(resp.Candidates) > 0 {
		finish = normalizeFinishReason(resp.Candidates[0].FinishReason)
	}

	var usage *provider.Usage
	if resp.UsageMetadata.TotalTokenCount > 0 || resp.UsageMetadata.PromptTokenCount > 0 || resp.UsageMetadata.CandidatesTokenCount > 0 {
		usage = &provider.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &provider.StreamChunk{
		ID:      fmt.Sprintf("gemini-stream-%d", time.Now().UnixNano()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []provider.StreamChoice{{
			Index: 0,
			Delta: &provider.StreamDelta{
				Role:    "assistant",
				Content: text,
			},
			FinishReason: finish,
		}},
		Usage: usage,
		Done:  done,
	}
}

func extractCandidateText(candidates []Candidate) string {
	if len(candidates) == 0 {
		return ""
	}
	parts := candidates[0].Content.Parts
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		if p.Text != "" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func extractText(content interface{}) string {
	switch v := content.(type) {
	case string:
		return strings.TrimSpace(v)
	case []provider.ContentPart:
		var b strings.Builder
		for _, p := range v {
			if p.Type == "text" && p.Text != "" {
				b.WriteString(p.Text)
			}
		}
		return strings.TrimSpace(b.String())
	case []interface{}:
		var b strings.Builder
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					b.WriteString(t)
				}
			}
		}
		return strings.TrimSpace(b.String())
	default:
		return ""
	}
}

func normalizeFinishReason(reason string) string {
	if reason == "" {
		return ""
	}
	switch strings.ToUpper(reason) {
	case "STOP", "MAX_TOKENS", "SAFETY", "RECITATION", "OTHER":
		return strings.ToLower(reason)
	default:
		return strings.ToLower(reason)
	}
}

func isRetryableError(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusInternalServerError || code == http.StatusBadGateway || code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout
}
