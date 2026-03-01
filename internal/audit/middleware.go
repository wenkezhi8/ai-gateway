package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//nolint:revive // Kept for package API compatibility.
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

		if am.shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		requestBody := readRequestBody(c)

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		latency := time.Since(start)

		if c.Writer.Status() >= 400 || am.shouldAudit(c) {
			am.logger.Log(am.buildAuditLogEntry(c, requestBody, latency))
		}
	}
}

func (am *AuditMiddleware) shouldSkipPath(path string) bool {
	for skipPath := range am.skipPaths {
		if strings.HasSuffix(skipPath, "*") {
			prefix := strings.TrimSuffix(skipPath, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
			continue
		}

		if path == skipPath {
			return true
		}
	}

	return false
}

func readRequestBody(c *gin.Context) []byte {
	if c.Request.Body == nil || !shouldReadBody(c.Request.Method) {
		return nil
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body
}

func shouldReadBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func (am *AuditMiddleware) buildAuditLogEntry(c *gin.Context, requestBody []byte, latency time.Duration) LogEntry {
	userID, username, _ := getUserFromContext(c)

	return LogEntry{
		Timestamp:  time.Now(),
		UserID:     userID,
		Username:   username,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Action:     am.detectAction(c),
		Resource:   am.detectResource(c),
		ResourceID: am.detectResourceID(c),
		Detail:     am.buildDetail(c, latency),
		OldData:    nil,
		NewData:    am.extractNewData(requestBody),
		Status:     requestStatus(c.Writer.Status()),
		Error:      c.Errors.String(),
	}
}

func requestStatus(statusCode int) string {
	if statusCode >= http.StatusBadRequest {
		return "failed"
	}
	return "success"
}

func (am *AuditMiddleware) extractNewData(requestBody []byte) interface{} {
	if len(requestBody) == 0 {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(requestBody, &data); err != nil {
		return nil
	}

	return am.sanitizeData(data)
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
		if id, ok := v.(string); ok {
			userID = id
		}
	}
	if v, exists := c.Get("username"); exists {
		if name, ok := v.(string); ok {
			username = name
		}
	}
	if v, exists := c.Get("role"); exists {
		if value, ok := v.(string); ok {
			role = value
		}
	}
	return
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func containsMiddle(s, substr string) bool {
	return strings.Contains(s, substr)
}

//nolint:revive // Kept for package API compatibility.
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
