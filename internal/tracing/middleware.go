package tracing

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

type TraceRecorder interface {
	RecordSpan(ctx context.Context, spanName string, fn func(context.Context) error) error
}

func TraceMiddleware(recorder *SpanRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = GenerateRequestID()
		}

		// Set to context
		ctx := SetRequestIDToContext(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", requestID)

		// Set response header
		c.Header("X-Request-ID", requestID)

		// Record span
		startTime := time.Now()

		// Process request
		c.Next()

		duration := time.Since(startTime)

		// Create trace record
		traceRecord := &RequestTrace{
			ID:         GenerateRequestID(),
			RequestID:  requestID,
			TraceID:    requestID, // For simplicity, use request_id as trace_id
			SpanID:     GenerateRequestID()[:16],
			Operation:  "http-request",
			Status:     getStatus(c.Writer.Status()),
			StartTime:  startTime,
			EndTime:    startTime.Add(duration),
			DurationMs: duration.Milliseconds(),
			Attributes: JSONB{
				"method":      c.Request.Method,
				"path":        c.Request.URL.Path,
				"status_code": c.Writer.Status(),
				"client_ip":   c.ClientIP(),
				"user_agent":  c.Request.UserAgent(),
			},
			Events:    JSONB{},
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			CreatedAt: time.Now(),
		}

		// Get user info from context if available
		if userID, exists := c.Get("user_id"); exists {
			traceRecord.UserID = userID.(string)
		}

		// Save to database asynchronously
		if recorder != nil {
			go recorder.saveToDB(traceRecord)
		}
	}
}

func getStatus(statusCode int) string {
	if statusCode >= 200 && statusCode < 400 {
		return "success"
	}
	return "error"
}
