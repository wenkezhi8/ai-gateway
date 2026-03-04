package admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ai-gateway/internal/tracing"

	"github.com/gin-gonic/gin"
)

type TraceHandler struct {
	db *sql.DB
}

type traceSummary struct {
	RequestID    string `json:"request_id"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Status       string `json:"status"`
	DurationMs   int64  `json:"duration_ms"`
	CreatedAt    string `json:"created_at"`
	StepCount    int    `json:"step_count"`
	AnswerSource string `json:"answer_source"`
	TaskType     string `json:"task_type"`
	Model        string `json:"model"` // 新增
}

func NewTraceHandler(db *sql.DB) *TraceHandler {
	return &TraceHandler{db: db}
}

// GET /api/admin/traces.
func (h *TraceHandler) GetTraces(c *gin.Context) {
	limit := 100
	if parsedLimit, err := strconv.Atoi(c.DefaultQuery("limit", "100")); err == nil && parsedLimit > 0 {
		limit = parsedLimit
	}
	offset := 0
	if parsedOffset, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && parsedOffset >= 0 {
		offset = parsedOffset
	}
	operation := c.Query("operation")
	status := c.Query("status")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	totalQuery := `
		WITH request_candidates AS (
			SELECT DISTINCT request_id
			FROM request_traces
			WHERE (? = '' OR operation = ?)
			  AND (? = '' OR created_at >= ?)
			  AND (? = '' OR created_at <= ?)
		),
		request_agg AS (
			SELECT
				rt.request_id,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.entry' THEN rt.method END), MAX(rt.method), '') AS method,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.entry' THEN rt.path END), MAX(rt.path), '') AS path,
				CASE WHEN SUM(CASE WHEN rt.status = 'error' THEN 1 ELSE 0 END) > 0 THEN 'error' ELSE 'success' END AS request_status,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.response' THEN rt.duration_ms END), MAX(rt.duration_ms), 0) AS duration_ms,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.entry' THEN rt.created_at END), MAX(rt.created_at), '') AS created_at,
				COUNT(*) AS step_count,
				COALESCE(
					MAX(CASE WHEN rt.operation = 'classifier.assess' AND json_valid(rt.attributes) = 1 THEN NULLIF(json_extract(rt.attributes, '$.task_type'), '') END),
					MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 THEN NULLIF(json_extract(rt.attributes, '$.task_type'), '') END),
					''
				) AS task_type,
				CASE
					WHEN MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'cache_v2'
					WHEN MAX(CASE WHEN rt.operation = 'cache.read-semantic' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'cache_semantic'
					WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'cache_exact'
					WHEN MAX(CASE WHEN rt.operation = 'provider.chat' AND rt.status = 'success' THEN 1 ELSE 0 END) = 1 THEN 'provider_chat'
					ELSE 'provider_chat'
				END AS answer_source,
				COALESCE(MAX(CASE WHEN rt.operation='http.response' THEN rt.model END), MAX(rt.model), '') AS model
			FROM request_traces rt
			INNER JOIN request_candidates rc ON rc.request_id = rt.request_id
			GROUP BY rt.request_id
		)
		SELECT COUNT(*) FROM request_agg WHERE (? = '' OR request_status = ?)
	`
	var total int
	totalArgs := []interface{}{operation, operation, startTime, startTime, endTime, endTime, status, status}
	if err := h.db.QueryRow(totalQuery, totalArgs...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "query_failed", "message": err.Error()},
		})
		return
	}

	listQuery := `
		WITH request_candidates AS (
			SELECT DISTINCT request_id
			FROM request_traces
			WHERE (? = '' OR operation = ?)
			  AND (? = '' OR created_at >= ?)
			  AND (? = '' OR created_at <= ?)
		),
		request_agg AS (
			SELECT
				rt.request_id,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.entry' THEN rt.method END), MAX(rt.method), '') AS method,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.entry' THEN rt.path END), MAX(rt.path), '') AS path,
				CASE WHEN SUM(CASE WHEN rt.status = 'error' THEN 1 ELSE 0 END) > 0 THEN 'error' ELSE 'success' END AS request_status,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.response' THEN rt.duration_ms END), MAX(rt.duration_ms), 0) AS duration_ms,
				COALESCE(MAX(CASE WHEN rt.operation = 'http.entry' THEN rt.created_at END), MAX(rt.created_at), '') AS created_at,
				COUNT(*) AS step_count,
				COALESCE(
					MAX(CASE WHEN rt.operation = 'classifier.assess' AND json_valid(rt.attributes) = 1 THEN NULLIF(json_extract(rt.attributes, '$.task_type'), '') END),
					MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 THEN NULLIF(json_extract(rt.attributes, '$.task_type'), '') END),
					''
				) AS task_type,
				CASE
					WHEN MAX(CASE WHEN rt.operation = 'cache.read-v2' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'cache_v2'
					WHEN MAX(CASE WHEN rt.operation = 'cache.read-semantic' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'cache_semantic'
					WHEN MAX(CASE WHEN rt.operation = 'cache.read-exact' AND json_valid(rt.attributes) = 1 AND json_extract(rt.attributes, '$.result') = 'hit' THEN 1 ELSE 0 END) = 1 THEN 'cache_exact'
					WHEN MAX(CASE WHEN rt.operation = 'provider.chat' AND rt.status = 'success' THEN 1 ELSE 0 END) = 1 THEN 'provider_chat'
					ELSE 'provider_chat'
				END AS answer_source,
				COALESCE(MAX(CASE WHEN rt.operation='http.response' THEN rt.model END), MAX(rt.model), '') AS model
			FROM request_traces rt
			INNER JOIN request_candidates rc ON rc.request_id = rt.request_id
			GROUP BY rt.request_id
		)
		SELECT request_id, method, path, request_status, duration_ms, created_at, step_count, answer_source, task_type, model
		FROM request_agg
		WHERE (? = '' OR request_status = ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	listArgs := []interface{}{operation, operation, startTime, startTime, endTime, endTime, status, status, limit, offset}
	rows, err := h.db.Query(listQuery, listArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "query_failed", "message": err.Error()},
		})
		return
	}
	defer rows.Close()

	summaries := make([]traceSummary, 0)
	for rows.Next() {
		var row traceSummary
		if scanErr := rows.Scan(
			&row.RequestID,
			&row.Method,
			&row.Path,
			&row.Status,
			&row.DurationMs,
			&row.CreatedAt,
			&row.StepCount,
			&row.AnswerSource,
			&row.TaskType,
			&row.Model,
		); scanErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   gin.H{"code": "scan_failed", "message": scanErr.Error()},
			})
			return
		}
		summaries = append(summaries, row)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "scan_failed", "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summaries,
		"total":   total,
	})
}

// GET /api/admin/traces/:request_id.
func (h *TraceHandler) GetTraceDetail(c *gin.Context) {
	requestID := c.Param("request_id")

	query := `
		SELECT id, request_id, trace_id, span_id, parent_span_id, operation, status,
		       start_time, end_time, duration_ms, attributes, events, user_id, method, path, model, provider, error, created_at
		FROM request_traces
		WHERE request_id = ?
		ORDER BY rowid ASC
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

	traces, err := scanRequestTraces(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "scan_failed", "message": err.Error()},
		})
		return
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

func (h *TraceHandler) ClearTraces(c *gin.Context) {
	result, err := h.db.Exec("DELETE FROM request_traces")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "delete_failed", "message": err.Error()},
		})
		return
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "rows_affected_failed", "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": deleted,
		},
	})
}

func parseRFC3339Flexible(v string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t
	}
	return time.Time{}
}

func scanRequestTraces(rows *sql.Rows) ([]*tracing.RequestTrace, error) {
	traces := make([]*tracing.RequestTrace, 0)
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

		trace.StartTime = parseRFC3339Flexible(startTimeStr)
		trace.EndTime = parseRFC3339Flexible(endTimeStr)
		trace.CreatedAt = parseRFC3339Flexible(createdAtStr)
		if attrsJSON != "" {
			if err := json.Unmarshal([]byte(attrsJSON), &trace.Attributes); err != nil {
				continue
			}
		}
		if eventsJSON != "" {
			if err := json.Unmarshal([]byte(eventsJSON), &trace.Events); err != nil {
				continue
			}
		}
		traces = append(traces, trace)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return traces, nil
}
