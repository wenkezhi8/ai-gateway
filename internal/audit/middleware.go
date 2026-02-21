package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuditMiddleware struct {
	logger          *Logger
	skipPaths       map[string]bool
	sensitiveFields []string
}

func NewAuditMiddleware(logger *Logger) *AuditMiddleware {
	return &AuditMiddleware{
		logger: logger,
		skipPaths: map[string]bool{
			"/health":    true,
			"/metrics":   true,
			"/swagger/*": true,
		},
		sensitiveFields: []string{
			"password",
			"api_key",
			"secret",
			"token",
		},
	}
}

func (am *AuditMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if am.logger == nil {
			c.Next()
			return
		}

		for path := range am.skipPaths {
			if len(path) > 0 && path[len(path)-1] == '*' {
				prefix := path[:len(path)-1]
				if len(c.Request.URL.Path) >= len(prefix) && c.Request.URL.Path[:len(prefix)] == prefix {
					c.Next()
					return
				}
			}
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		start := time.Now()

		var requestBody []byte
		if c.Request.Body != nil && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH") {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		latency := time.Since(start)

		if c.Writer.Status() >= 400 || am.shouldAudit(c) {
			userID, username, _ := getUserFromContext(c)

			action := am.detectAction(c)
			resource := am.detectResource(c)
			resourceID := am.detectResourceID(c)

			detail := am.buildDetail(c, latency)

			var oldData, newData interface{}
			if len(requestBody) > 0 {
				var data map[string]interface{}
				if json.Unmarshal(requestBody, &data) == nil {
					newData = am.sanitizeData(data)
				}
			}

			status := "success"
			if c.Writer.Status() >= 400 {
				status = "failed"
			}

			am.logger.Log(LogEntry{
				Timestamp:  time.Now(),
				UserID:     userID,
				Username:   username,
				IP:         c.ClientIP(),
				UserAgent:  c.Request.UserAgent(),
				Action:     action,
				Resource:   resource,
				ResourceID: resourceID,
				Detail:     detail,
				OldData:    oldData,
				NewData:    newData,
				Status:     status,
				Error:      c.Errors.String(),
			})
		}
	}
}

func (am *AuditMiddleware) shouldAudit(c *gin.Context) bool {
	path := c.Request.URL.Path

	auditPaths := []string{
		"/api/admin/accounts",
		"/api/admin/providers",
		"/api/admin/routing",
		"/api/admin/cache",
		"/api/admin/config",
		"/api/auth",
	}

	for _, prefix := range auditPaths {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	return c.Request.Method != "GET"
}

func (am *AuditMiddleware) detectAction(c *gin.Context) ActionType {
	method := c.Request.Method
	path := c.Request.URL.Path

	if contains(path, "/login") {
		return ActionLogin
	}
	if contains(path, "/logout") {
		return ActionLogout
	}
	if contains(path, "/switch") {
		return ActionForceSwitch
	}
	if contains(path, "/test") {
		return ActionTestConnect
	}
	if contains(path, "/clear") || (method == "DELETE" && contains(path, "/cache")) {
		return ActionCacheClear
	}
	if contains(path, "/config") {
		return ActionConfig
	}

	switch method {
	case "POST":
		return ActionCreate
	case "PUT", "PATCH":
		return ActionUpdate
	case "DELETE":
		return ActionDelete
	default:
		return ActionCreate
	}
}

func (am *AuditMiddleware) detectResource(c *gin.Context) ResourceType {
	path := c.Request.URL.Path

	if contains(path, "/accounts") {
		return ResourceAccount
	}
	if contains(path, "/providers") {
		return ResourceProvider
	}
	if contains(path, "/routing") {
		return ResourceRouting
	}
	if contains(path, "/cache") {
		return ResourceCache
	}
	if contains(path, "/config") {
		return ResourceConfig
	}
	if contains(path, "/auth") || contains(path, "/login") {
		return ResourceAuth
	}

	return ResourceSystem
}

func (am *AuditMiddleware) detectResourceID(c *gin.Context) string {
	return c.Param("id")
}

func (am *AuditMiddleware) buildDetail(c *gin.Context, latency time.Duration) string {
	return c.Request.Method + " " + c.Request.URL.Path + " (" + latency.String() + ")"
}

func (am *AuditMiddleware) sanitizeData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		if am.isSensitive(k) {
			result[k] = "***"
		} else {
			result[k] = v
		}
	}
	return result
}

func (am *AuditMiddleware) isSensitive(field string) bool {
	for _, sf := range am.sensitiveFields {
		if contains(field, sf) {
			return true
		}
	}
	return false
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func getUserFromContext(c *gin.Context) (userID, username, role string) {
	if v, exists := c.Get("user_id"); exists {
		userID = v.(string)
	}
	if v, exists := c.Get("username"); exists {
		username = v.(string)
	}
	if v, exists := c.Get("role"); exists {
		role = v.(string)
	}
	return
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || containsMiddle(s, substr))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func AuditHandler(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Limit  int                    `json:"limit"`
			Offset int                    `json:"offset"`
			Filter map[string]interface{} `json:"filter"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			req.Limit = 100
			req.Offset = 0
		}

		if req.Limit <= 0 || req.Limit > 1000 {
			req.Limit = 100
		}

		logs := logger.GetLogs(req.Limit, req.Offset, req.Filter)
		total := logger.Count(req.Filter)

		c.JSON(http.StatusOK, gin.H{
			"data":   logs,
			"total":  total,
			"limit":  req.Limit,
			"offset": req.Offset,
		})
	}
}
