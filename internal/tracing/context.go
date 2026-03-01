package tracing

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const requestIDKey ctxKey = "request-id"

func GenerateRequestID() string {
	return uuid.New().String()
}

func GetRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func SetRequestIDToContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}
