package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type SpanRecorder struct {
	db *sql.DB
}

func NewSpanRecorder(db *sql.DB) *SpanRecorder {
	return &SpanRecorder{db: db}
}

func (r *SpanRecorder) RecordSpan(ctx context.Context, spanName string, fn func(context.Context) error) error {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("ai-gateway")

	ctx, span := tracer.Start(ctx, spanName, trace.WithTimestamp(time.Now()))
	defer span.End(trace.WithTimestamp(time.Now()))

	startTime := time.Now()
	err := fn(ctx)
	endTime := time.Now()

	// Record to database
	requestID := GetRequestIDFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	status := "success"
	errorMsg := ""
	if err != nil {
		status = "error"
		errorMsg = err.Error()
		span.SetStatus(codes.Error, errorMsg)
		span.RecordError(err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	attrs := extractAttributes(span)
	events := extractEvents(span)

	traceRecord := &RequestTrace{
		ID:         uuid.New().String(),
		RequestID:  requestID,
		TraceID:    traceID,
		SpanID:     spanID,
		Operation:  spanName,
		Status:     status,
		StartTime:  startTime,
		EndTime:    endTime,
		DurationMs: endTime.Sub(startTime).Milliseconds(),
		Attributes: attrs,
		Events:     events,
		Error:      errorMsg,
		CreatedAt:  time.Now(),
	}

	if err := r.saveToDB(traceRecord); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to save trace record: %v\n", err)
	}

	return err
}

func (r *SpanRecorder) saveToDB(trace *RequestTrace) error {
	attrsJSON, _ := json.Marshal(trace.Attributes)
	eventsJSON, _ := json.Marshal(trace.Events)

	_, err := r.db.Exec(`
		INSERT INTO request_traces (
			id, request_id, trace_id, span_id, parent_span_id,
			operation, status, start_time, end_time, duration_ms,
			attributes, events, user_id, method, path, model, provider, error, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		trace.ID, trace.RequestID, trace.TraceID, trace.SpanID, trace.ParentSpanID,
		trace.Operation, trace.Status, trace.StartTime.Format(time.RFC3339), trace.EndTime.Format(time.RFC3339), trace.DurationMs,
		string(attrsJSON), string(eventsJSON), trace.UserID, trace.Method, trace.Path, trace.Model, trace.Provider, trace.Error, trace.CreatedAt.Format(time.RFC3339),
	)

	return err
}

func extractAttributes(span trace.Span) JSONB {
	// In a real implementation, you'd extract attributes from the span
	// For now, return empty map
	return JSONB{}
}

func extractEvents(span trace.Span) JSONB {
	// In a real implementation, you'd extract events from the span
	// For now, return empty map
	return JSONB{}
}

func (r *SpanRecorder) GetTracesByRequestID(requestID string) ([]*RequestTrace, error) {
	rows, err := r.db.Query(`
		SELECT id, request_id, trace_id, span_id, parent_span_id, operation, status,
		       start_time, end_time, duration_ms, attributes, events, user_id, method, path, model, provider, error, created_at
		FROM request_traces
		WHERE request_id = ?
		ORDER BY start_time ASC
	`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []*RequestTrace
	for rows.Next() {
		trace := &RequestTrace{}
		var attrsJSON, eventsJSON string
		var startTime, endTime, createdAt string

		err := rows.Scan(
			&trace.ID, &trace.RequestID, &trace.TraceID, &trace.SpanID, &trace.ParentSpanID,
			&trace.Operation, &trace.Status, &startTime, &endTime, &trace.DurationMs,
			&attrsJSON, &eventsJSON, &trace.UserID, &trace.Method, &trace.Path, &trace.Model, &trace.Provider, &trace.Error, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		trace.StartTime, _ = time.Parse(time.RFC3339, startTime)
		trace.EndTime, _ = time.Parse(time.RFC3339, endTime)
		trace.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		json.Unmarshal([]byte(attrsJSON), &trace.Attributes)
		json.Unmarshal([]byte(eventsJSON), &trace.Events)

		traces = append(traces, trace)
	}

	return traces, nil
}

func AddAttribute(ctx context.Context, key string, value interface{}) {
	span := trace.SpanFromContext(ctx)
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	}
}
