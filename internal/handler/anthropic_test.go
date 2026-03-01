package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"
)

func boolPtrAnthropic(v bool) *bool {
	return &v
}

func TestConvertAnthropicMessage_Multimodal(t *testing.T) {
	msg := AnthropicMessage{
		Role: "user",
		Content: []interface{}{
			map[string]interface{}{"type": "text", "text": "describe image"},
			map[string]interface{}{
				"type": "image",
				"source": map[string]interface{}{
					"type": "url",
					"url":  "https://example.com/a.png",
				},
			},
		},
	}

	converted := convertAnthropicMessage(msg)
	require.Len(t, converted, 1)
	assert.Equal(t, "user", converted[0].Role)

	parts, ok := converted[0].Content.([]interface{})
	require.True(t, ok)
	require.Len(t, parts, 2)

	first, ok := parts[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "text", first["type"])
	assert.Equal(t, "describe image", first["text"])

	second, ok := parts[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "image_url", second["type"])
}

func TestConvertAnthropicMessage_ToolUse(t *testing.T) {
	msg := AnthropicMessage{
		Role: "assistant",
		Content: []interface{}{
			map[string]interface{}{
				"type": "tool_use",
				"id":   "toolu_123",
				"name": "get_weather",
				"input": map[string]interface{}{
					"city": "Shanghai",
				},
			},
		},
	}

	converted := convertAnthropicMessage(msg)
	require.Len(t, converted, 1)
	require.Len(t, converted[0].ToolCalls, 1)
	assert.Equal(t, "toolu_123", converted[0].ToolCalls[0].ID)
	assert.Equal(t, "get_weather", converted[0].ToolCalls[0].Function.Name)
}

func TestConvertAnthropicMessage_ToolResult(t *testing.T) {
	msg := AnthropicMessage{
		Role: "user",
		Content: []interface{}{
			map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": "toolu_123",
				"content":     "25C",
			},
		},
	}

	converted := convertAnthropicMessage(msg)
	require.Len(t, converted, 1)
	assert.Equal(t, "tool", converted[0].Role)
	assert.Equal(t, "toolu_123", converted[0].ToolCallID)
	assert.Equal(t, "25C", converted[0].Content)
}

func TestBuildAnthropicResponseFromProvider(t *testing.T) {
	resp := &provider.ChatResponse{
		ID:    "chatcmpl-123",
		Model: "claude-3-5-sonnet-20241022",
		Choices: []provider.Choice{
			{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: "tool result",
					ToolCalls: []provider.ToolCall{
						{
							ID:   "toolu_123",
							Type: "function",
							Function: provider.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"city":"Shanghai"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: provider.Usage{PromptTokens: 10, CompletionTokens: 7},
	}

	antResp := buildAnthropicResponseFromProvider(resp)
	assert.Equal(t, "message", antResp.Type)
	assert.Equal(t, "assistant", antResp.Role)
	assert.Equal(t, "tool_use", antResp.StopReason)
	assert.Equal(t, 10, antResp.Usage.InputTokens)
	assert.Equal(t, 7, antResp.Usage.OutputTokens)
	require.GreaterOrEqual(t, len(antResp.Content), 2)
	assert.Equal(t, "text", antResp.Content[0].Type)
	assert.Equal(t, "tool_use", antResp.Content[1].Type)
}

func TestApplyControlToolGateAnthropic(t *testing.T) {
	req := &AnthropicMessagesRequest{
		Tools: []AnthropicTool{{Name: "get_weather"}},
		ToolChoice: map[string]interface{}{
			"type": "tool",
		},
	}

	cfg := routing.ControlConfig{Enable: true, ToolGateEnable: true}
	assessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{ToolNeeded: boolPtrAnthropic(false)}}

	applyControlToolGateAnthropic(req, cfg, assessment)
	assert.Len(t, req.Tools, 0)
	assert.Nil(t, req.ToolChoice)

	shadowReq := &AnthropicMessagesRequest{Tools: []AnthropicTool{{Name: "get_weather"}}}
	shadowCfg := routing.ControlConfig{Enable: true, ToolGateEnable: true, ShadowOnly: true}
	applyControlToolGateAnthropic(shadowReq, shadowCfg, assessment)
	assert.Len(t, shadowReq.Tools, 1)

	allowReq := &AnthropicMessagesRequest{Tools: []AnthropicTool{{Name: "get_weather"}}}
	allowAssessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{ToolNeeded: boolPtrAnthropic(true)}}
	applyControlToolGateAnthropic(allowReq, cfg, allowAssessment)
	assert.Len(t, allowReq.Tools, 1)
}

func TestApplyControlGenerationHintsAnthropic(t *testing.T) {
	temp := 0.25
	topP := 0.88
	maxTokens := 1200

	req := &AnthropicMessagesRequest{}
	cfg := routing.ControlConfig{Enable: true, ParameterHintEnable: true}
	assessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{
		RecommendedTemperature: &temp,
		RecommendedTopP:        &topP,
		RecommendedMaxTokens:   &maxTokens,
	}}

	applyControlGenerationHintsAnthropic(req, cfg, assessment)
	if assert.NotNil(t, req.Temperature) {
		assert.Equal(t, temp, *req.Temperature)
	}
	if assert.NotNil(t, req.TopP) {
		assert.Equal(t, topP, *req.TopP)
	}
	assert.Equal(t, maxTokens, req.MaxTokens)

	shadowReq := &AnthropicMessagesRequest{}
	shadowCfg := routing.ControlConfig{Enable: true, ParameterHintEnable: true, ShadowOnly: true}
	applyControlGenerationHintsAnthropic(shadowReq, shadowCfg, assessment)
	assert.Nil(t, shadowReq.Temperature)
	assert.Nil(t, shadowReq.TopP)
	assert.Equal(t, 0, shadowReq.MaxTokens)
}
