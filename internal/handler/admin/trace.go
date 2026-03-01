package admin

import (
	"ai-gateway/internal/tracing"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type TraceHandler struct {
	db *sql.DB
}

func NewTraceHandler(db *sql.DB) *TraceHandler {
	return &TraceHandler{db: db}
}

// GetTraces returns list of traces
// GET /api/admin/traces
func (h *TraceHandler) GetTraces(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	operation := c.Query("operation")
	status := c.Query("status")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	query := `
		SELECT id, request_id, trace_id, span_id, parent_span_id, operation, status,
		       start_time, end_time, duration_ms, attributes, events, user_id, method, path, model, provider, error, created_at
		FROM request_traces
		WHERE 1=1
	`
	args := []interface{}{}

	if operation != "" {
		query += " AND operation = ?"
		args = append(args, operation)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if startTime != "" {
		query += " AND created_at >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		query += " AND created_at <= ?"
		args = append(args, endTime)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "query_failed", "message": err.Error()},
		})
		return
	}
	defer rows.Close()

	traces := []*tracing.RequestTrace{}
	for rows.Next() {
		trace := &tracing.RequestTrace{}
		var attrsJSON, eventsJSON string
		var startTimeStr, endTimeStr, createdAtStr string

		err := rows.Scan(
			&trace.ID, &trace.RequestID, &trace.TraceID, &trace.SpanID, &trace.ParentSpanID,
			&trace.Operation, &trace.Status, &startTimeStr, &endTimeStr, &trace.DurationMs,
			&attrsJSON, &eventsJSON, &trace.UserID, &trace.Method, &trace.Path, &trace.Model, &trace.Provider, &trace.Error, &createdAtStr,
		)
		if err != nil {
			continue
		}

		trace.StartTime, _ = time.Parse(time.RFC3339, startTimeStr)
		trace.EndTime, _ = time.Parse(time.RFC3339, endTimeStr)
		trace.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		traces = append(traces, trace)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    traces,
	})
}

// GetTraceDetail returns detail of a trace
// GET /api/admin/traces/:request_id
func (h *TraceHandler) GetTraceDetail(c *gin.Context) {
	requestID := c.Param("request_id")

	query := `
		SELECT id, request_id, trace_id, span_id, parent_span_id, operation, status,
		       start_time, end_time, duration_ms, attributes, events, user_id, method, path, model, provider, error, created_at
		FROM request_traces
		WHERE request_id = ?
		ORDER BY start_time ASC
	`

	rows, err := h.db.Query(query, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "query_failed", "message": err.Error()},
		})
		return
	}
	defer rows.Close()

	traces := []*tracing.RequestTrace{}
	for rows.Next() {
		trace := &tracing.RequestTrace{}
		var attrsJSON, eventsJSON string
		var startTimeStr, endTimeStr, createdAtStr string

		err := rows.Scan(
			&trace.ID, &trace.RequestID, &trace.TraceID, &trace.SpanID, &trace.ParentSpanID,
			&trace.Operation, &trace.Status, &startTimeStr, &endTimeStr, &trace.DurationMs,
			&attrsJSON, &eventsJSON, &trace.UserID, &trace.Method, &trace.Path, &trace.Model, &trace.Provider, &trace.Error, &createdAtStr,
		)
		if err != nil {
			continue
		}

		trace.StartTime, _ = time.Parse(time.RFC3339, startTimeStr)
		trace.EndTime, _ = time.Parse(time.RFC3339, endTimeStr)
		trace.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		traces = append(traces, trace)
	}

	if len(traces) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   gin.H{"code": "not_found", "message": "Trace not found"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    traces,
	})
}
