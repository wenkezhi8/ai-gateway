package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"
)

const (
	anthropicModelAuto       = "auto"
	anthropicModelLatest     = "latest"
	anthropicModelDefault    = "default"
	anthropicContentTypeText = "text"
	finishReasonStop         = "stop"
)

type AnthropicMessagesRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	System        interface{}        `json:"system,omitempty"`
	MaxTokens     int                `json:"max_tokens,omitempty"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Metadata      *AnthropicMetadata `json:"metadata,omitempty"`
	Tools         []AnthropicTool    `json:"tools,omitempty"`
	ToolChoice    interface{}        `json:"tool_choice,omitempty"`
}

type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AnthropicContentBlock struct {
	Type      string      `json:"type"`
	Text      string      `json:"text,omitempty"`
	Source    interface{} `json:"source,omitempty"`
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicMessagesResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason,omitempty"`
	StopSequence string                  `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

type AnthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

//nolint:gocyclo // Handler logic is intentionally linear for request/response flow.
func (h *ProxyHandler) AnthropicMessages(c *gin.Context) {
	startTime := time.Now()

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodySize)

	var req AnthropicMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			h.writeAnthropicError(c, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds maximum size of 10MB")
			return
		}
		h.writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "Invalid request body: "+err.Error())
		return
	}

	if req.Model == "" {
		h.writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	if len(req.Messages) == 0 {
		h.writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "messages is required and cannot be empty")
		return
	}
	sanitizeAnthropicRequestForIntentAndUpstream(&req)

	prompt := extractAnthropicPrompt(req.Messages)
	assessment := h.smartRouter.AssessDifficulty(prompt, "")
	classifierCfg := h.smartRouter.GetClassifierConfig()
	controlCfg := classifierCfg.Control
	if controlCfg.Enable && controlCfg.RiskTagEnable {
		logControlRiskSignals(assessment)
	}
	for k, v := range buildControlHeaders(controlCfg, assessment) {
		c.Header(k, v)
	}
	if shouldBlockByRisk(controlCfg, assessment) {
		h.writeAnthropicError(c, http.StatusForbidden, "permission_error", "Request blocked by control risk policy")
		return
	}
	applyControlToolGateAnthropic(&req, controlCfg, assessment)
	applyControlGenerationHintsAnthropic(&req, controlCfg, assessment)

	requestedModel := req.Model
	if req.Model == anthropicModelAuto || req.Model == anthropicModelLatest || req.Model == anthropicModelDefault {
		availableModels := make([]string, 0)
		if h.accountManager != nil {
			for _, acc := range h.accountManager.GetAllAccounts() {
				if acc.Enabled {
					availableModels = append(availableModels, acc.Provider)
				}
			}
		}

		switch req.Model {
		case anthropicModelLatest:
			requestedModel = h.smartRouter.SelectModelWithStrategy("latest", routing.StrategyQuality, prompt, availableModels)
		case anthropicModelDefault:
			config := h.smartRouter.GetConfig()
			if config.DefaultModel != "" {
				requestedModel = config.DefaultModel
			} else {
				requestedModel = h.smartRouter.SelectModelForProvider("default", "anthropic", prompt, availableModels)
			}
		default:
			requestedModel = h.smartRouter.SelectModelForProvider(anthropicModelAuto, "anthropic", prompt, availableModels)
		}
	}

	providerReq := buildProviderRequestFromAnthropic(&req, requestedModel)
	targetProvider, err := h.getProviderForRequest(c, providerReq.Model, "anthropic")
	if err != nil {
		h.recordMetrics("", "", providerReq.Model, time.Since(startTime), 0, false)
		h.writeAnthropicError(c, http.StatusServiceUnavailable, "provider_error", err.Error())
		return
	}

	if providerReq.Stream {
		h.handleAnthropicStreamResponse(c, targetProvider, providerReq, startTime)
		return
	}

	resp, err := targetProvider.Chat(c.Request.Context(), providerReq)
	if err != nil {
		h.recordMetrics("", "", providerReq.Model, time.Since(startTime), 0, false)
		if pErr, ok := err.(*provider.ProviderError); ok {
			h.writeAnthropicError(c, pErr.Code, mapProviderErrorType(pErr.Code), pErr.Message)
			return
		}
		h.writeAnthropicError(c, http.StatusBadGateway, "api_error", err.Error())
		return
	}

	if resp.Error != nil {
		h.recordMetrics("", "", providerReq.Model, time.Since(startTime), 0, false)
		h.writeAnthropicError(c, resp.Error.Code, mapProviderErrorType(resp.Error.Code), resp.Error.Message)
		return
	}

	antResp := buildAnthropicResponseFromProvider(resp)
	h.recordMetrics("", "", providerReq.Model, time.Since(startTime), resp.Usage.TotalTokens, true)
	c.JSON(http.StatusOK, antResp)
}

func applyControlToolGateAnthropic(req *AnthropicMessagesRequest, controlCfg routing.ControlConfig, assessment *routing.AssessmentResult) {
	if req == nil || !controlCfg.Enable || !controlCfg.ToolGateEnable || assessment == nil || assessment.ControlSignals == nil {
		return
	}
	if assessment.ControlSignals.ToolNeeded == nil || *assessment.ControlSignals.ToolNeeded {
		return
	}
	if len(req.Tools) == 0 {
		return
	}
	if controlCfg.ShadowOnly {
		return
	}
	req.Tools = nil
	req.ToolChoice = nil
}

func applyControlGenerationHintsAnthropic(req *AnthropicMessagesRequest, controlCfg routing.ControlConfig, assessment *routing.AssessmentResult) {
	if req == nil || !controlCfg.Enable || !controlCfg.ParameterHintEnable || assessment == nil || assessment.ControlSignals == nil {
		return
	}
	if controlCfg.ShadowOnly {
		return
	}

	if req.Temperature == nil && assessment.ControlSignals.RecommendedTemperature != nil {
		v := *assessment.ControlSignals.RecommendedTemperature
		req.Temperature = &v
	}
	if req.TopP == nil && assessment.ControlSignals.RecommendedTopP != nil {
		v := *assessment.ControlSignals.RecommendedTopP
		req.TopP = &v
	}
	if req.MaxTokens <= 0 && assessment.ControlSignals.RecommendedMaxTokens != nil {
		req.MaxTokens = *assessment.ControlSignals.RecommendedMaxTokens
	}
}

func buildProviderRequestFromAnthropic(req *AnthropicMessagesRequest, model string) *provider.ChatRequest {
	if req == nil {
		return &provider.ChatRequest{Model: model}
	}

	extra := map[string]interface{}{}
	if req.TopP != nil {
		extra["top_p"] = *req.TopP
	}
	if req.TopK != nil {
		extra["top_k"] = *req.TopK
	}
	if len(req.StopSequences) > 0 {
		extra["stop"] = req.StopSequences
	}
	if req.Metadata != nil && req.Metadata.UserID != "" {
		extra["user_id"] = req.Metadata.UserID
	}

	providerReq := &provider.ChatRequest{
		Model:     model,
		Stream:    req.Stream,
		MaxTokens: req.MaxTokens,
		Extra:     extra,
	}

	if req.Temperature != nil {
		providerReq.Temperature = *req.Temperature
	}

	if req.System != nil {
		providerReq.Messages = append(providerReq.Messages, provider.ChatMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		providerReq.Messages = append(providerReq.Messages, convertAnthropicMessage(msg)...)
	}

	if len(req.Tools) > 0 {
		providerReq.Tools = make([]provider.Tool, len(req.Tools))
		for i, t := range req.Tools {
			providerReq.Tools[i] = provider.Tool{
				Type: "function",
				Function: provider.Function{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			}
		}
	}

	if req.ToolChoice != nil {
		providerReq.ToolChoice = req.ToolChoice
	}

	return providerReq
}

//nolint:gocyclo // Multimodal conversion branches by block type.
func convertAnthropicMessage(msg AnthropicMessage) []provider.ChatMessage {
	contentBlocks, isArray := msg.Content.([]interface{})
	if !isArray {
		return []provider.ChatMessage{{
			Role:    msg.Role,
			Content: msg.Content,
		}}
	}

	contentParts := make([]interface{}, 0)
	toolCalls := make([]provider.ToolCall, 0)
	result := make([]provider.ChatMessage, 0)

	for _, raw := range contentBlocks {
		block, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		typeName := getStringFromMap(block, "type")
		if typeName == "" {
			continue
		}
		switch typeName {
		case anthropicContentTypeText:
			if text, ok := block["text"].(string); ok {
				contentParts = append(contentParts, map[string]interface{}{
					"type": anthropicContentTypeText,
					"text": text,
				})
			}
		case "image":
			source, ok := block["source"].(map[string]interface{})
			if !ok {
				continue
			}
			url, ok := source["url"].(string)
			if !ok {
				continue
			}
			if url != "" {
				contentParts = append(contentParts, map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": url,
					},
				})
			}
		case "tool_use":
			callID := getStringFromMap(block, "id")
			name := getStringFromMap(block, "name")
			input, ok := block["input"].(map[string]interface{})
			if !ok {
				input = map[string]interface{}{}
			}
			args, err := json.Marshal(input)
			if err != nil {
				continue
			}
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:   callID,
				Type: "function",
				Function: provider.FunctionCall{
					Name:      name,
					Arguments: string(args),
				},
			})
		case "tool_result":
			toolUseID := getStringFromMap(block, "tool_use_id")
			result = append(result, provider.ChatMessage{
				Role:       "tool",
				ToolCallID: toolUseID,
				Content:    block["content"],
			})
		}
	}

	mainMsg := provider.ChatMessage{Role: msg.Role}
	if len(contentParts) == 1 {
		if first, ok := contentParts[0].(map[string]interface{}); ok {
			if first["type"] == anthropicContentTypeText {
				if text, ok := first["text"].(string); ok {
					mainMsg.Content = text
				}
			}
		}
	}
	if mainMsg.Content == nil && len(contentParts) > 0 {
		mainMsg.Content = contentParts
	}
	if len(toolCalls) > 0 {
		mainMsg.ToolCalls = toolCalls
	}

	if mainMsg.Content != nil || len(mainMsg.ToolCalls) > 0 {
		result = append([]provider.ChatMessage{mainMsg}, result...)
	}

	return result
}

func buildAnthropicResponseFromProvider(resp *provider.ChatResponse) AnthropicMessagesResponse {
	message := provider.ChatMessage{}
	finishReason := ""
	if len(resp.Choices) > 0 {
		message = resp.Choices[0].Message
		finishReason = resp.Choices[0].FinishReason
	}

	content := make([]AnthropicContentBlock, 0)
	text := getTextContent(message.Content)
	if text != "" {
		content = append(content, AnthropicContentBlock{Type: anthropicContentTypeText, Text: text})
	}

	for _, call := range message.ToolCalls {
		var input interface{}
		if call.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(call.Function.Arguments), &input); err != nil {
				input = map[string]string{"raw": call.Function.Arguments}
			}
		}
		content = append(content, AnthropicContentBlock{
			Type:  "tool_use",
			ID:    call.ID,
			Name:  call.Function.Name,
			Input: input,
		})
	}

	if len(content) == 0 {
		content = append(content, AnthropicContentBlock{Type: anthropicContentTypeText, Text: ""})
	}

	return AnthropicMessagesResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Content:    content,
		Model:      resp.Model,
		StopReason: mapFinishReasonToAnthropic(finishReason),
		Usage: AnthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
}

//nolint:gocyclo // Stream event mapping needs explicit branching.
func (h *ProxyHandler) handleAnthropicStreamResponse(c *gin.Context, p provider.Provider, req *provider.ChatRequest, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	stream, err := p.StreamChat(c.Request.Context(), req)
	if err != nil {
		h.recordMetrics("", "", req.Model, time.Since(startTime), 0, false)
		h.writeAnthropicError(c, http.StatusBadGateway, "api_error", err.Error())
		return
	}

	messageID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	if err := writeAnthropicSSE(c, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            messageID,
			"type":          "message",
			"role":          "assistant",
			"content":       []interface{}{},
			"model":         req.Model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]int{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	}); err != nil {
		return
	}

	contentStarted := false
	finalStopReason := ""
	lastUsage := provider.Usage{}

	for chunk := range stream {
		if chunk.Usage != nil {
			lastUsage = *chunk.Usage
		}

		for _, choice := range chunk.Choices {
			if choice.Delta != nil && choice.Delta.Content != "" {
				if !contentStarted {
					if err := writeAnthropicSSE(c, "content_block_start", map[string]interface{}{
						"type":  "content_block_start",
						"index": 0,
						"content_block": map[string]interface{}{
							"type": anthropicContentTypeText,
							"text": "",
						},
					}); err != nil {
						return
					}
					contentStarted = true
				}

				if err := writeAnthropicSSE(c, "content_block_delta", map[string]interface{}{
					"type":  "content_block_delta",
					"index": 0,
					"delta": map[string]interface{}{
						"type": "text_delta",
						"text": choice.Delta.Content,
					},
				}); err != nil {
					return
				}
			}

			if choice.FinishReason != "" {
				finalStopReason = mapFinishReasonToAnthropic(choice.FinishReason)
			}
		}

		if chunk.Done {
			break
		}
	}

	if contentStarted {
		if err := writeAnthropicSSE(c, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": 0,
		}); err != nil {
			return
		}
	}

	if err := writeAnthropicSSE(c, "message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   finalStopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]int{
			"output_tokens": lastUsage.CompletionTokens,
		},
	}); err != nil {
		return
	}

	if err := writeAnthropicSSE(c, "message_stop", map[string]interface{}{
		"type": "message_stop",
	}); err != nil {
		return
	}

	h.recordMetrics("", "", req.Model, time.Since(startTime), lastUsage.TotalTokens, true)
}

func writeAnthropicSSE(c *gin.Context, event string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func extractAnthropicPrompt(messages []AnthropicMessage) string {
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		if text, ok := msg.Content.(string); ok {
			return text
		}
		if blocks, ok := msg.Content.([]interface{}); ok {
			for _, raw := range blocks {
				if block, ok := raw.(map[string]interface{}); ok {
					if block["type"] == anthropicContentTypeText {
						if text, ok := block["text"].(string); ok {
							return text
						}
					}
				}
			}
		}
	}
	return ""
}

func sanitizeAnthropicRequestForIntentAndUpstream(req *AnthropicMessagesRequest) {
	if req == nil {
		return
	}
	req.System = sanitizeAnthropicContent(req.System)
	for i := range req.Messages {
		req.Messages[i].Content = sanitizeAnthropicContent(req.Messages[i].Content)
	}
}

func sanitizeAnthropicContent(content interface{}) interface{} {
	switch v := content.(type) {
	case string:
		return routing.SanitizeIntentInput(v)
	case []interface{}:
		parts := make([]interface{}, len(v))
		for i, item := range v {
			m, ok := item.(map[string]interface{})
			if !ok {
				parts[i] = item
				continue
			}

			copied := make(map[string]interface{}, len(m))
			for key, value := range m {
				copied[key] = value
			}

			if copied["type"] == anthropicContentTypeText {
				if text, ok := copied["text"].(string); ok {
					copied["text"] = routing.SanitizeIntentInput(text)
				}
			}

			parts[i] = copied
		}
		return parts
	case map[string]interface{}:
		copied := make(map[string]interface{}, len(v))
		for key, value := range v {
			copied[key] = value
		}
		if copied["type"] == anthropicContentTypeText {
			if text, ok := copied["text"].(string); ok {
				copied["text"] = routing.SanitizeIntentInput(text)
			}
		}
		return copied
	default:
		return content
	}
}

func getStringFromMap(data map[string]interface{}, key string) string {
	value, ok := data[key].(string)
	if !ok {
		return ""
	}
	return value
}

func mapFinishReasonToAnthropic(finishReason string) string {
	switch finishReason {
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	case finishReasonStop, "":
		return "end_turn"
	default:
		return finishReason
	}
}

func mapProviderErrorType(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "invalid_request_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	default:
		return "api_error"
	}
}

func (h *ProxyHandler) writeAnthropicError(c *gin.Context, statusCode int, errType, message string) {
	if statusCode <= 0 {
		statusCode = http.StatusBadGateway
	}
	resp := AnthropicErrorResponse{Type: "error"}
	resp.Error.Type = errType
	resp.Error.Message = message
	c.JSON(statusCode, resp)
}
