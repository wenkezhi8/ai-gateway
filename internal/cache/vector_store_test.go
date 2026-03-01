//nolint:errcheck,revive,unparam // Mock command handlers intentionally lightweight for behavior tests.
package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedisExecutor struct {
	handlers map[string]func(args ...any) (any, error)
	calls    [][]any
}

func (f *fakeRedisExecutor) Do(ctx context.Context, args ...any) *redis.Cmd {
	_ = ctx
	f.calls = append(f.calls, args)
	cmd := redis.NewCmd(context.Background(), args...)
	if len(args) == 0 {
		cmd.SetErr(errors.New("empty command"))
		return cmd
	}
	name, _ := args[0].(string)
	name = strings.ToUpper(name)
	if h, ok := f.handlers[name]; ok {
		val, err := h(args...)
		if err != nil {
			cmd.SetErr(err)
			return cmd
		}
		cmd.SetVal(val)
		return cmd
	}
	cmd.SetErr(fmt.Errorf("unexpected command: %s", name))
	return cmd
}

func TestRedisStackVectorStore_EnsureIndex_Idempotent(t *testing.T) {
	exec := &fakeRedisExecutor{
		handlers: map[string]func(args ...any) (any, error){
			"FT.CREATE": func(args ...any) (any, error) {
				return nil, errors.New("Index already exists")
			},
		},
	}

	store := NewRedisStackVectorStoreWithExecutor(exec, DefaultRedisStackVectorConfig())
	if err := store.EnsureIndex(context.Background()); err != nil {
		t.Fatalf("expected idempotent ensure index, got %v", err)
	}
}

func TestRedisStackVectorStore_UpsertAndGetExact(t *testing.T) {
	mem := map[string]string{}
	exec := &fakeRedisExecutor{
		handlers: map[string]func(args ...any) (any, error){
			"JSON.SET": func(args ...any) (any, error) {
				key := args[1].(string)
				val := args[3].(string)
				mem[key] = val
				return "OK", nil
			},
			"EXPIRE": func(args ...any) (any, error) {
				return int64(1), nil
			},
			"JSON.GET": func(args ...any) (any, error) {
				key := args[1].(string)
				val := mem[key]
				if val == "" {
					return nil, redis.Nil
				}
				return val, nil
			},
		},
	}

	store := NewRedisStackVectorStoreWithExecutor(exec, DefaultRedisStackVectorConfig())
	doc := &VectorCacheDocument{
		CacheKey: "intent:calc:expr=1+1",
		Intent:   "calc",
		TaskType: "math",
		Slots: map[string]string{
			"expr": "1+1",
		},
		NormalizedQuery: "1+1",
		Vector:          []float64{0.1, 0.2},
		Response: map[string]any{
			"role":    "assistant",
			"content": "1+1=2",
		},
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		QualityScore: 99,
		TTLSec:       int64((24 * time.Hour).Seconds()),
	}

	if err := store.Upsert(context.Background(), doc); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	got, err := store.GetExact(context.Background(), doc.CacheKey)
	if err != nil {
		t.Fatalf("get exact failed: %v", err)
	}
	if got == nil || got.CacheKey != doc.CacheKey {
		t.Fatalf("unexpected document: %+v", got)
	}
}

func TestRedisStackVectorStore_VectorSearch_FilterBySimilarity(t *testing.T) {
	exec := &fakeRedisExecutor{
		handlers: map[string]func(args ...any) (any, error){
			"FT.SEARCH": func(args ...any) (any, error) {
				return []any{
					int64(2),
					"ai:v2:cache:intent:calc:expr=1+1",
					[]any{
						"$.cache_key", "intent:calc:expr=1+1",
						"$.intent", "calc",
						"$.response", `{"role":"assistant","content":"2"}`,
						"vector_score", "0.03",
					},
					"ai:v2:cache:intent:calc:expr=2+2",
					[]any{
						"$.cache_key", "intent:calc:expr=2+2",
						"$.intent", "calc",
						"$.response", `{"role":"assistant","content":"4"}`,
						"vector_score", "0.15",
					},
				}, nil
			},
		},
	}

	cfg := DefaultRedisStackVectorConfig()
	cfg.Dimension = 2
	store := NewRedisStackVectorStoreWithExecutor(exec, cfg)

	hits, err := store.VectorSearch(context.Background(), "calc", []float64{0.1, 0.2}, 2, 0.95)
	if err != nil {
		t.Fatalf("vector search failed: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].CacheKey != "intent:calc:expr=1+1" {
		t.Fatalf("unexpected hit: %+v", hits[0])
	}
}

func TestRedisStackVectorStore_TouchTTL(t *testing.T) {
	called := false
	exec := &fakeRedisExecutor{
		handlers: map[string]func(args ...any) (any, error){
			"EXPIRE": func(args ...any) (any, error) {
				called = true
				if _, err := strconv.ParseInt(fmt.Sprintf("%v", args[2]), 10, 64); err != nil {
					t.Fatalf("expected numeric ttl, got %v", args[2])
				}
				return int64(1), nil
			},
		},
	}

	store := NewRedisStackVectorStoreWithExecutor(exec, DefaultRedisStackVectorConfig())
	if err := store.TouchTTL(context.Background(), "intent:calc:expr=1+1", int64((10 * time.Minute).Seconds())); err != nil {
		t.Fatalf("touch ttl failed: %v", err)
	}
	if !called {
		t.Fatal("expected expire to be called")
	}
}
