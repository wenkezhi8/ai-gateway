//nolint:godot,gocyclo,gocritic
package openai

import (
	"ai-gateway/internal/provider"
)

// Converter handles request/response conversion between unified format and OpenAI format

// ConvertRequest converts provider.ChatRequest to OpenAI ChatRequest
func ConvertRequest(req *provider.ChatRequest) *ChatRequest {
	messages := make([]ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = ChatMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				messages[i].ToolCalls[j] = ToolCall{
					Index: tc.Index,
					ID:    tc.ID,
					Type:  tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	openaiReq := &ChatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	// Enable usage statistics for streaming requests
	if req.Stream {
		openaiReq.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			openaiReq.Tools[i] = Tool{
				Type: t.Type,
				Function: Function{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}
	if req.ToolChoice != nil {
		openaiReq.ToolChoice = req.ToolChoice
	}

	if req.Temperature > 0 {
		openaiReq.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		openaiReq.MaxTokens = req.MaxTokens
	}

	if req.Extra != nil {
		if topP, ok := req.Extra["top_p"].(float64); ok {
			openaiReq.TopP = topP
		}
		if n, ok := req.Extra["n"].(int); ok {
			openaiReq.N = n
		}
		if stop, ok := req.Extra["stop"].([]string); ok {
			openaiReq.Stop = stop
		}
		if stop, ok := req.Extra["stop"].([]interface{}); ok {
			for _, s := range stop {
				if str, ok := s.(string); ok {
					openaiReq.Stop = append(openaiReq.Stop, str)
				}
			}
		}
		if freqPenalty, ok := req.Extra["frequency_penalty"].(float64); ok {
			openaiReq.FrequencyPenalty = freqPenalty
		}
		if presPenalty, ok := req.Extra["presence_penalty"].(float64); ok {
			openaiReq.PresencePenalty = presPenalty
		}
		if user, ok := req.Extra["user"].(string); ok {
			openaiReq.User = user
		}
		if seed, ok := req.Extra["seed"].(int); ok {
			openaiReq.Seed = seed
		}
		if logprobs, ok := req.Extra["logprobs"].(bool); ok {
			openaiReq.Logprobs = logprobs
		}
		if topLogprobs, ok := req.Extra["top_logprobs"].(int); ok {
			openaiReq.TopLogprobs = topLogprobs
		}
		if maxCompletionTokens, ok := req.Extra["max_completion_tokens"].(int); ok {
			openaiReq.MaxCompletionTokens = maxCompletionTokens
		}
		if logitBias, ok := req.Extra["logit_bias"].(map[string]interface{}); ok {
			openaiReq.LogitBias = make(map[string]float64)
			for k, v := range logitBias {
				if f, ok := v.(float64); ok {
					openaiReq.LogitBias[k] = f
				}
			}
		}
	}

	return openaiReq
}

// ConvertResponse converts OpenAI ChatResponse to provider.ChatResponse
func ConvertResponse(resp *ChatResponse) *provider.ChatResponse {
	choices := make([]provider.Choice, 0, len(resp.Choices))
	for _, c := range resp.Choices {
		content := ""
		if c.Message.Content != nil {
			switch v := c.Message.Content.(type) {
			case string:
				content = v
			case []interface{}:
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						if text, ok := m["text"].(string); ok {
							content += text
						}
					}
				}
			}
		}

		var toolCalls []provider.ToolCall
		if len(c.Message.ToolCalls) > 0 {
			toolCalls = make([]provider.ToolCall, len(c.Message.ToolCalls))
			for i, tc := range c.Message.ToolCalls {
				toolCalls[i] = provider.ToolCall{
					Index: tc.Index,
					ID:    tc.ID,
					Type:  tc.Type,
					Function: provider.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choice := provider.Choice{
			Index: c.Index,
			Message: provider.ChatMessage{
				Role:      c.Message.Role,
				Content:   content,
				Name:      c.Message.Name,
				ToolCalls: toolCalls,
			},
			FinishReason: c.FinishReason,
		}
		choices = append(choices, choice)
	}

	var providerErr *provider.ProviderError
	if resp.Error != nil {
		statusCode := resp.Error.StatusCode
		if statusCode == 0 {
			switch resp.Error.Type {
			case "invalid_request_error":
				statusCode = 400
			case "authentication_error":
				statusCode = 401
			case "permission_error":
				statusCode = 403
			case "not_found_error":
				statusCode = 404
			case "rate_limit_error":
				statusCode = 429
			case "server_error":
				statusCode = 500
			}
		}

		providerErr = &provider.ProviderError{
			Code:      statusCode,
			Message:   resp.Error.Message,
			Type:      resp.Error.Type,
			Provider:  "openai",
			Retryable: isRetryableError(statusCode),
		}
	}

	return &provider.ChatResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Error: providerErr,
	}
}

// ConvertStreamChunk converts OpenAI StreamResponse to provider.StreamChunk
func ConvertStreamChunk(resp *StreamResponse, done bool) *provider.StreamChunk {
	choices := make([]provider.StreamChoice, 0, len(resp.Choices))
	for _, c := range resp.Choices {
		var delta *provider.StreamDelta
		if c.Delta != nil {
			delta = &provider.StreamDelta{
				Role:    c.Delta.Role,
				Content: c.Delta.Content,
			}
			if len(c.Delta.ToolCalls) > 0 {
				delta.ToolCalls = make([]provider.ToolCall, len(c.Delta.ToolCalls))
				for i, tc := range c.Delta.ToolCalls {
					delta.ToolCalls[i] = provider.ToolCall{
						Index: tc.Index,
						ID:    tc.ID,
						Type:  tc.Type,
						Function: provider.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
		}
		choices = append(choices, provider.StreamChoice{
			Index:        c.Index,
			Delta:        delta,
			FinishReason: c.FinishReason,
		})
	}

	var usage *provider.Usage
	if resp.Usage != nil {
		usage = &provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return &provider.StreamChunk{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
		Done:    done,
	}
}
