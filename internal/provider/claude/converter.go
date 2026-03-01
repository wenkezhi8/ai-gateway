//nolint:godot,gocyclo
package claude

import (
	"encoding/json"

	"ai-gateway/internal/provider"
)

// ConvertRequest converts provider.ChatRequest to Claude MessagesRequest
func ConvertRequest(req *provider.ChatRequest) *MessagesRequest {
	messages := make([]Message, 0, len(req.Messages))
	var systemPrompt interface{}

	for i, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}

		if m.Role == "tool" {
			var content interface{} = m.Content
			if m.Content != "" {
				content = m.Content
			}
			messages = append(messages, Message{
				Role: "user",
				Content: []ContentBlock{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   content,
				}},
			})
			continue
		}

		if len(m.ToolCalls) > 0 {
			content := make([]ContentBlock, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				content[j] = ContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: json.RawMessage(tc.Function.Arguments),
				}
			}
			messages = append(messages, Message{
				Role:    convertRole(m.Role),
				Content: content,
			})
			continue
		}

		content := convertContentToBlocks(m.Content)

		messages = append(messages, Message{
			Role:    convertRole(m.Role),
			Content: content,
		})

		if i > 0 && len(messages) > 1 {
			prevMsg := messages[len(messages)-2]
			currMsg := &messages[len(messages)-1]
			if prevMsg.Role == currMsg.Role {
				prevMsg.Content = append(prevMsg.Content, currMsg.Content...)
				messages = messages[:len(messages)-1]
			}
		}
	}

	claudeReq := &MessagesRequest{
		Model:    req.Model,
		Messages: messages,
		System:   systemPrompt,
		Stream:   req.Stream,
	}

	if req.MaxTokens > 0 {
		claudeReq.MaxTokens = req.MaxTokens
	} else {
		claudeReq.MaxTokens = 4096
	}

	if req.Temperature > 0 {
		claudeReq.Temperature = req.Temperature
	}

	if len(req.Tools) > 0 {
		claudeReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			claudeReq.Tools[i] = Tool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			}
		}
	}
	if req.ToolChoice != nil {
		claudeReq.ToolChoice = req.ToolChoice
	}

	if req.Extra != nil {
		if topP, ok := req.Extra["top_p"].(float64); ok {
			claudeReq.TopP = topP
		}
		if topK, ok := req.Extra["top_k"].(int); ok {
			claudeReq.TopK = topK
		}
		if topK, ok := req.Extra["top_k"].(float64); ok {
			claudeReq.TopK = int(topK)
		}
		if stop, ok := req.Extra["stop"].([]string); ok {
			claudeReq.StopSequences = stop
		}
		if stop, ok := req.Extra["stop"].([]interface{}); ok {
			for _, s := range stop {
				if str, ok := s.(string); ok {
					claudeReq.StopSequences = append(claudeReq.StopSequences, str)
				}
			}
		}
		if userID, ok := req.Extra["user_id"].(string); ok {
			claudeReq.Metadata = &Metadata{UserID: userID}
		}
	}

	return claudeReq
}

func convertRole(role string) string {
	switch role {
	case "assistant":
		return "assistant"
	case "user", "system", "function", "tool":
		return "user"
	default:
		return role
	}
}

// ConvertResponse converts Claude MessagesResponse to provider.ChatResponse
func ConvertResponse(resp *MessagesResponse) *provider.ChatResponse {
	return ConvertToProviderResponse(resp)
}

// ConvertStreamEvent converts Claude stream event to provider stream chunk
func ConvertStreamEvent(event *StreamEvent, model string) *provider.StreamChunk {
	switch event.Type {
	case "content_block_delta":
		if event.Delta != nil && event.Delta.Type == "text_delta" {
			return &provider.StreamChunk{
				ID:      "",
				Object:  "chat.completion.chunk",
				Created: 0,
				Model:   model,
				Choices: []provider.StreamChoice{
					{
						Index: event.Index,
						Delta: &provider.StreamDelta{
							Content: event.Delta.Text,
						},
					},
				},
			}
		}
	case "message_delta":
		if event.Delta != nil && event.Delta.StopReason != "" {
			return &provider.StreamChunk{
				ID:      "",
				Object:  "chat.completion.chunk",
				Created: 0,
				Model:   model,
				Choices: []provider.StreamChoice{
					{
						Index:        0,
						FinishReason: event.Delta.StopReason,
					},
				},
				Done: true,
			}
		}
	case "message_stop":
		return &provider.StreamChunk{
			Done: true,
		}
	}
	return nil
}

// convertContentToBlocks convertsinterface{}Content to Claude ContentBlock array
func convertContentToBlocks(content interface{}) []ContentBlock {
	switch v := content.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []ContentBlock{{Type: "text", Text: v}}
	case []interface{}:
		blocks := make([]ContentBlock, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				switch m["type"] {
				case "text":
					if text, ok := m["text"].(string); ok {
						blocks = append(blocks, ContentBlock{Type: "text", Text: text})
					}
				case "image_url":
					if imgURL, ok := m["image_url"].(map[string]interface{}); ok {
						if url, ok := imgURL["url"].(string); ok {
							// Claude uses different image format
							blocks = append(blocks, ContentBlock{
								Type: "image",
								Source: &ImageSource{
									Type: "url",
									URL:  url,
								},
							})
						}
					}
				}
			}
		}
		return blocks
	case []provider.ContentPart:
		blocks := make([]ContentBlock, 0, len(v))
		for _, part := range v {
			switch part.Type {
			case "text":
				blocks = append(blocks, ContentBlock{Type: "text", Text: part.Text})
			case "image_url":
				if part.ImageURL != nil {
					blocks = append(blocks, ContentBlock{
						Type: "image",
						Source: &ImageSource{
							Type: "url",
							URL:  part.ImageURL.URL,
						},
					})
				}
			}
		}
		return blocks
	}
	return nil
}
