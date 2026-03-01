package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/constants"
)

func (h *ProxyHandler) APIv1Root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":  "AI Gateway OpenAI-compatible API",
		"base_url": constants.ApiV1Prefix,
		"endpoints": gin.H{
			"chat_completions": constants.ChatCompletions,
			"completions":      constants.Completions,
			"embeddings":       constants.Embeddings,
			"models":           constants.Models,
			"providers":        constants.Providers,
		},
		"tip": "POST to /api/v1 or /api/v1/chat/completions with OpenAI chat payload",
	})
}

func (h *ProxyHandler) APIv1ChatCompletionsInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":  "OpenAI chat completions endpoint",
		"endpoint": constants.ChatCompletions,
		"method":   "POST",
		"tip":      "Send OpenAI-compatible chat payload to this URL",
	})
}

func (h *ProxyHandler) AnthropicRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":  "AI Gateway Anthropic-compatible API",
		"base_url": constants.ApiAnthropicBasePrefix,
		"endpoints": gin.H{
			"messages": constants.ApiAnthropicPrefix + "/messages",
		},
		"tip": "POST to /api/anthropic or /api/anthropic/v1/messages with Anthropic messages payload",
	})
}

func (h *ProxyHandler) AnthropicMessagesInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":  "Anthropic messages endpoint",
		"endpoint": constants.ApiAnthropicPrefix + "/messages",
		"method":   "POST",
		"tip":      "Send Anthropic-compatible messages payload to this URL",
	})
}
