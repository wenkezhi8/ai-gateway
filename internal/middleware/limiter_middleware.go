//nolint:godot // Legacy comments are kept terse in this file.
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ai-gateway/internal/limiter"
)

// LimiterMiddleware provides rate limiting middleware
type LimiterMiddleware struct {
	manager *limiter.AccountManager
	tracker *limiter.UsageTracker
	logger  *logrus.Logger
}

// NewLimiterMiddleware creates a new limiter middleware
func NewLimiterMiddleware(manager *limiter.AccountManager, tracker *limiter.UsageTracker, logger *logrus.Logger) *LimiterMiddleware {
	return &LimiterMiddleware{
		manager: manager,
		tracker: tracker,
		logger:  logger,
	}
}

// RateLimit middleware checks and enforces rate limits
func (m *LimiterMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get provider from context (set by upstream middleware)
		provider, exists := c.Get("provider")
		if !exists {
			c.Next()
			return
		}

		providerStr, ok := provider.(string)
		if !ok {
			c.Next()
			return
		}

		// Get estimated cost from context (can be set by request parsing)
		estimatedCost := int64(1000) // Default estimate
		if cost, exists := c.Get("estimated_tokens"); exists {
			if costInt, ok := cost.(int64); ok {
				estimatedCost = costInt
			}
		}

		// Check and get active account
		account, err := m.manager.CheckAndSwitch(c.Request.Context(), providerStr, estimatedCost)
		if err != nil {
			m.logger.WithError(err).WithField("provider", providerStr).Error("Rate limit check failed")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "All accounts have exceeded their limits",
			})
			c.Abort()
			return
		}

		// Store active account in context for use by handlers
		c.Set("active_account", account)

		c.Next()

		// After request completes, record actual usage
		if tokens, exists := c.Get("tokens_used"); exists {
			if tokensInt, ok := tokens.(int64); ok {
				if err := m.manager.ConsumeUsage(c.Request.Context(), account.ID, tokensInt); err != nil {
					m.logger.WithError(err).WithField("account_id", account.ID).Error("Failed to consume usage")
				}
			}
		}
	}
}

// UsageTracking middleware tracks request usage
func (m *LimiterMiddleware) UsageTracking() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// Record request duration
		duration := time.Since(start)

		// Log slow requests
		if duration > 5*time.Second {
			m.logger.WithFields(logrus.Fields{
				"method":   c.Request.Method,
				"path":     c.Request.URL.Path,
				"duration": duration.String(),
			}).Warn("Slow request")
		}

		// Update metrics
		if account, exists := c.Get("active_account"); exists {
			if acc, ok := account.(*limiter.AccountConfig); ok {
				m.logger.WithFields(logrus.Fields{
					"account_id": acc.ID,
					"provider":   acc.Provider,
					"status":     c.Writer.Status(),
					"duration":   duration.String(),
				}).Debug("Request completed")
			}
		}
	}
}

// AccountInfo middleware adds account info to response headers
func (m *LimiterMiddleware) AccountInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Add account info to response headers
		if account, exists := c.Get("active_account"); exists {
			if acc, ok := account.(*limiter.AccountConfig); ok {
				c.Header("X-Account-ID", acc.ID)
				c.Header("X-Account-Provider", acc.Provider)
			}
		}

		// Add usage info if available
		if tokens, exists := c.Get("tokens_used"); exists {
			if tokensInt, ok := tokens.(int64); ok {
				c.Header("X-Tokens-Used", string(rune(tokensInt)))
			}
		}
	}
}

// RequireAccount middleware ensures an account is available
func (m *LimiterMiddleware) RequireAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")
		if provider == "" {
			provider = c.GetHeader("X-Provider")
		}
		if provider == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "provider_required",
				"message": "Provider must be specified",
			})
			c.Abort()
			return
		}

		// Set provider in context
		c.Set("provider", provider)

		// Get active account
		account, err := m.manager.GetActiveAccount(provider)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "no_available_account",
				"message": "No available account for this provider",
			})
			c.Abort()
			return
		}

		c.Set("active_account", account)
		c.Next()
	}
}

// AlertHandler handles usage alerts
func (m *LimiterMiddleware) AlertHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Subscribe to alerts
		alertChan := m.tracker.Subscribe()
		defer m.tracker.Unsubscribe(alertChan)

		// Set SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// Stream alerts
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case alert := <-alertChan:
				c.SSEvent("alert", gin.H{
					"type":       alert.Type,
					"account_id": alert.AccountID,
					"limit_type": alert.LimitType,
					"percent":    alert.PercentUsed,
					"message":    alert.Message,
					"timestamp":  alert.Timestamp,
				})
				c.Writer.Flush()
			}
		}
	}
}

// GetUsageReport returns current usage report
func (m *LimiterMiddleware) GetUsageReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		report, err := m.tracker.GetReport(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "report_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, report)
	}
}

// GetAccountStatus returns status for a specific account
func (m *LimiterMiddleware) GetAccountStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Param("account_id")
		if accountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "account_id_required",
				"message": "Account ID is required",
			})
			return
		}

		status, err := m.manager.GetAccountStatus(accountID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "account_not_found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, status)
	}
}

// ForceSwitch forces a switch to a specific account
func (m *LimiterMiddleware) ForceSwitch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Provider  string `json:"provider" binding:"required"`
			AccountID string `json:"account_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": err.Error(),
			})
			return
		}

		if err := m.manager.ForceSwitch(req.Provider, req.AccountID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "switch_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Account switched successfully",
		})
	}
}

// GetSwitchHistory returns the account switch history
func (m *LimiterMiddleware) GetSwitchHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50
		if l := c.Query("limit"); l != "" {
			parsedLimit, err := strconv.Atoi(l)
			if err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		history := m.manager.GetSwitchHistory(limit)
		c.JSON(http.StatusOK, gin.H{
			"history": history,
		})
	}
}
