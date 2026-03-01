//nolint:godot,gocyclo,dupl
package volcengine

import (
	"ai-gateway/internal/provider"
)

// ConvertRequest converts provider.ChatRequest to Volcengine ChatRequest
func ConvertRequest(req *provider.ChatRequest) *ChatRequest {
	messages := make([]ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = ChatMessage{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		}
		if len(m.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				messages[i].ToolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	volcReq := &ChatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if len(req.Tools) > 0 {
		volcReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			volcReq.Tools[i] = Tool{
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
		volcReq.ToolChoice = req.ToolChoice
	}

	if req.Temperature > 0 {
		volcReq.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		volcReq.MaxTokens = req.MaxTokens
	}

	if req.Extra != nil {
		if topP, ok := req.Extra["top_p"].(float64); ok {
			volcReq.TopP = topP
		}
		if stop, ok := req.Extra["stop"].([]string); ok {
			volcReq.Stop = stop
		}
		if stop, ok := req.Extra["stop"].([]interface{}); ok {
			for _, s := range stop {
				if str, ok := s.(string); ok {
					volcReq.Stop = append(volcReq.Stop, str)
				}
			}
		}
		if freqPenalty, ok := req.Extra["frequency_penalty"].(float64); ok {
			volcReq.FrequencyPenalty = freqPenalty
		}
		if presPenalty, ok := req.Extra["presence_penalty"].(float64); ok {
			volcReq.PresencePenalty = presPenalty
		}
		if user, ok := req.Extra["user"].(string); ok {
			volcReq.User = user
		}
		if n, ok := req.Extra["n"].(int); ok {
			volcReq.N = n
		}
		if logprobs, ok := req.Extra["logprobs"].(bool); ok {
			volcReq.Logprobs = logprobs
		}
		if topLogprobs, ok := req.Extra["top_logprobs"].(int); ok {
			volcReq.TopLogprobs = topLogprobs
		}
	}

	return volcReq
}

// ConvertResponse converts Volcengine ChatResponse to provider.ChatResponse
func ConvertResponse(resp *ChatResponse) *provider.ChatResponse {
	choices := make([]provider.Choice, 0, len(resp.Choices))
	for _, c := range resp.Choices {
		var toolCalls []provider.ToolCall
		if len(c.Message.ToolCalls) > 0 {
			toolCalls = make([]provider.ToolCall, len(c.Message.ToolCalls))
			for i, tc := range c.Message.ToolCalls {
				toolCalls[i] = provider.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
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
				Content:   c.Message.Content,
				Name:      c.Message.Name,
				ToolCalls: toolCalls,
			},
			FinishReason: c.FinishReason,
		}
		choices = append(choices, choice)
	}

	var providerErr *provider.ProviderError
	if resp.Error != nil {
		providerErr = &provider.ProviderError{
			Code:      resp.Error.Code,
			Message:   resp.Error.Message,
			Type:      resp.Error.Type,
			Provider:  "volcengine",
			Retryable: isRetryableError(resp.Error.Code),
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

// ConvertStreamChunk converts Volcengine StreamResponse to provider.StreamChunk
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
						ID:   tc.ID,
						Type: tc.Type,
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

	return &provider.StreamChunk{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Done:    done,
	}
}

func isRetryableError(code int) bool {
	return code >= 500 || code == 429 || code == 408
}
