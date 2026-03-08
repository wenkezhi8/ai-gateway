package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-gateway/internal/cache"
	vectordb "ai-gateway/internal/vector-db"

	"github.com/gin-gonic/gin"
)

func TestCacheHandler_GetRequestStats_ShouldUseDefault24hWindowAndSupportSourceFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTraceTestDB(t)
	handler := NewCacheHandler(cache.NewManagerWithCache(cache.NewMemoryCache()))
	handler.SetTraceDB(db)

	router := gin.New()
	router.GET("/api/admin/cache/request-stats", handler.GetCacheRequestStats)

	now := time.Now().UTC()
	inside := now.Add(-2 * time.Hour)
	outside := now.Add(-30 * time.Hour)

	insertTraceSpan(t, db, "req-exact-raw", "cache.read-exact", "success", "GET", 12, inside, map[string]any{"result": "hit", "cache_layer": "exact_raw"})
	insertTraceSpan(t, db, "req-exact-raw", "http.response", "success", "GET", 120, inside.Add(time.Second), map[string]any{
		"cache_layer":          "exact_raw",
		"user_message_preview": "question-raw",
		"ai_response_preview":  "answer-raw",
	})

	insertTraceSpan(t, db, "req-exact-prompt", "cache.read-exact", "success", "GET", 13, inside.Add(2*time.Second), map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-exact-prompt", "http.response", "success", "GET", 121, inside.Add(3*time.Second), map[string]any{
		"cache_layer":          "exact",
		"user_message_preview": "question-prompt",
		"ai_response_preview":  "answer-prompt",
	})

	insertTraceSpan(t, db, "req-v2", "cache.read-v2", "success", "GET", 14, inside.Add(4*time.Second), map[string]any{"result": "hit", "layer": "vector-exact"})
	insertTraceSpan(t, db, "req-v2", "http.response", "success", "GET", 122, inside.Add(5*time.Second), map[string]any{
		"cache_layer":          "vector-exact",
		"user_message_preview": "question-v2",
		"ai_response_preview":  "answer-v2",
	})

	insertTraceSpan(t, db, "req-provider", "provider.chat", "success", "POST", 220, inside.Add(6*time.Second), nil)
	insertTraceSpan(t, db, "req-provider", "http.response", "success", "POST", 260, inside.Add(7*time.Second), map[string]any{
		"user_message_preview": "question-provider",
		"ai_response_preview":  "answer-provider",
	})

	insertTraceSpan(t, db, "req-old", "cache.read-exact", "success", "GET", 10, outside, map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-old", "http.response", "success", "GET", 90, outside.Add(time.Second), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-stats", http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TotalRequests    int            `json:"total_requests"`
			CacheHitRequests int            `json:"cache_hit_requests"`
			CacheHitRate     float64        `json:"cache_hit_rate"`
			SourceBreakdown  map[string]int `json:"source_breakdown"`
			WindowStart      string         `json:"window_start"`
			WindowEnd        string         `json:"window_end"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("success=false")
	}
	if resp.Data.TotalRequests != 4 {
		t.Fatalf("total_requests=%d want=4", resp.Data.TotalRequests)
	}
	if resp.Data.CacheHitRequests != 3 {
		t.Fatalf("cache_hit_requests=%d want=3", resp.Data.CacheHitRequests)
	}
	if resp.Data.CacheHitRate != 0.75 {
		t.Fatalf("cache_hit_rate=%v want=0.75", resp.Data.CacheHitRate)
	}
	if resp.Data.SourceBreakdown["exact_raw"] != 1 {
		t.Fatalf("source_breakdown[exact_raw]=%d want=1", resp.Data.SourceBreakdown["exact_raw"])
	}
	if resp.Data.SourceBreakdown["exact_prompt"] != 1 {
		t.Fatalf("source_breakdown[exact_prompt]=%d want=1", resp.Data.SourceBreakdown["exact_prompt"])
	}
	if resp.Data.SourceBreakdown["v2"] != 1 {
		t.Fatalf("source_breakdown[v2]=%d want=1", resp.Data.SourceBreakdown["v2"])
	}
	if resp.Data.SourceBreakdown["provider_chat"] != 1 {
		t.Fatalf("source_breakdown[provider_chat]=%d want=1", resp.Data.SourceBreakdown["provider_chat"])
	}

	windowStart, err := time.Parse(time.RFC3339, resp.Data.WindowStart)
	if err != nil {
		t.Fatalf("parse window_start failed: %v", err)
	}
	windowEnd, err := time.Parse(time.RFC3339, resp.Data.WindowEnd)
	if err != nil {
		t.Fatalf("parse window_end failed: %v", err)
	}
	windowDuration := windowEnd.Sub(windowStart)
	if windowDuration < 23*time.Hour || windowDuration > 25*time.Hour {
		t.Fatalf("window duration=%s want around 24h", windowDuration)
	}

	assertStatsFilter := func(source string, wantTotal int, wantHitCount int, wantKey string) {
		t.Helper()
		filterReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-stats?source="+source, http.NoBody)
		filterRec := httptest.NewRecorder()
		router.ServeHTTP(filterRec, filterReq)

		if filterRec.Code != http.StatusOK {
			t.Fatalf("source=%s status=%d body=%s", source, filterRec.Code, filterRec.Body.String())
		}

		var filteredResp struct {
			Success bool `json:"success"`
			Data    struct {
				TotalRequests    int            `json:"total_requests"`
				CacheHitRequests int            `json:"cache_hit_requests"`
				SourceBreakdown  map[string]int `json:"source_breakdown"`
			} `json:"data"`
		}
		if err := json.Unmarshal(filterRec.Body.Bytes(), &filteredResp); err != nil {
			t.Fatalf("source=%s decode filtered response failed: %v", source, err)
		}
		if filteredResp.Data.TotalRequests != wantTotal {
			t.Fatalf("source=%s total_requests=%d want=%d", source, filteredResp.Data.TotalRequests, wantTotal)
		}
		if filteredResp.Data.CacheHitRequests != wantHitCount {
			t.Fatalf("source=%s cache_hit_requests=%d want=%d", source, filteredResp.Data.CacheHitRequests, wantHitCount)
		}
		if len(filteredResp.Data.SourceBreakdown) != 1 || filteredResp.Data.SourceBreakdown[wantKey] != wantTotal {
			t.Fatalf("source=%s source_breakdown=%v want only %s=%d", source, filteredResp.Data.SourceBreakdown, wantKey, wantTotal)
		}
	}

	assertStatsFilter("exact_raw", 1, 1, "exact_raw")
	assertStatsFilter("cache_exact", 1, 1, "exact_prompt")
	assertStatsFilter("cache_v2", 1, 1, "v2")

	windowReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-stats?window=1h", http.NoBody)
	windowRec := httptest.NewRecorder()
	router.ServeHTTP(windowRec, windowReq)
	if windowRec.Code != http.StatusOK {
		t.Fatalf("window status=%d body=%s", windowRec.Code, windowRec.Body.String())
	}
	var windowResp struct {
		Success bool `json:"success"`
		Data    struct {
			TotalRequests int `json:"total_requests"`
		} `json:"data"`
	}
	if err := json.Unmarshal(windowRec.Body.Bytes(), &windowResp); err != nil {
		t.Fatalf("decode window response failed: %v", err)
	}
	if windowResp.Data.TotalRequests != 0 {
		t.Fatalf("window total_requests=%d want=0", windowResp.Data.TotalRequests)
	}
}

func TestCacheHandler_GetRequestHits_ShouldSupportSourceFilterAndPaginationNormalization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTraceTestDB(t)
	handler := NewCacheHandler(cache.NewManagerWithCache(cache.NewMemoryCache()))
	handler.SetTraceDB(db)

	router := gin.New()
	router.GET("/api/admin/cache/request-hits", handler.GetCacheRequestHits)

	base := time.Now().UTC().Add(-3 * time.Hour)

	insertTraceSpan(t, db, "req-v2", "cache.read-v2", "success", "GET", 11, base, map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-v2", "http.response", "success", "GET", 111, base.Add(time.Second), map[string]any{
		"cache_layer":          "v2",
		"user_message_preview": "u-v2",
		"ai_response_preview":  "a-v2",
	})

	insertTraceSpan(t, db, "req-sem", "cache.read-semantic", "success", "GET", 12, base.Add(2*time.Second), map[string]any{"result": "hit"})
	insertTraceSpan(t, db, "req-sem", "http.response", "success", "GET", 112, base.Add(3*time.Second), map[string]any{
		"cache_layer":          "semantic",
		"user_message_preview": "u-sem",
		"ai_response_preview":  "a-sem",
	})

	insertTraceSpan(t, db, "req-exact-raw", "cache.read-exact", "success", "GET", 13, base.Add(4*time.Second), map[string]any{"result": "hit", "cache_layer": "exact_raw"})
	insertTraceSpan(t, db, "req-exact-raw", "http.response", "success", "GET", 113, base.Add(5*time.Second), map[string]any{
		"cache_layer":          "exact_raw",
		"user_message_preview": "u-exact-raw",
		"ai_response_preview":  "a-exact-raw",
	})

	insertTraceSpan(t, db, "req-exact-prompt", "cache.read-exact", "success", "GET", 14, base.Add(6*time.Second), map[string]any{"result": "hit", "cache_layer": "exact_prompt"})
	insertTraceSpan(t, db, "req-exact-prompt", "http.response", "success", "GET", 114, base.Add(7*time.Second), map[string]any{
		"cache_layer":          "exact_prompt",
		"user_message_preview": "u-exact-prompt",
		"ai_response_preview":  "a-exact-prompt",
	})

	insertTraceSpan(t, db, "req-provider", "provider.chat", "success", "POST", 210, base.Add(8*time.Second), nil)
	insertTraceSpan(t, db, "req-provider", "http.response", "success", "POST", 310, base.Add(9*time.Second), map[string]any{
		"user_message_preview": "u-provider",
		"ai_response_preview":  "a-provider",
	})

	insertTraceSpan(t, db, "req-old", "cache.read-v2", "success", "GET", 9, time.Now().UTC().Add(-48*time.Hour), map[string]any{"result": "hit"})

	type hitsResponse struct {
		Success bool `json:"success"`
		Data    struct {
			Entries []struct {
				RequestID          string `json:"request_id"`
				AnswerSource       string `json:"answer_source"`
				UserMessagePreview string `json:"user_message_preview"`
				AIResponsePreview  string `json:"ai_response_preview"`
			} `json:"entries"`
			Total       int    `json:"total"`
			Page        int    `json:"page"`
			PageSize    int    `json:"page_size"`
			WindowStart string `json:"window_start"`
			WindowEnd   string `json:"window_end"`
		} `json:"data"`
	}

	assertFilter := func(source string, wantTotal int, wantAnswerSource string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-hits?source="+source, http.NoBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("source=%s status=%d body=%s", source, rec.Code, rec.Body.String())
		}

		var resp hitsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("source=%s decode failed: %v", source, err)
		}
		if resp.Data.Total != wantTotal {
			t.Fatalf("source=%s total=%d want=%d", source, resp.Data.Total, wantTotal)
		}
		for _, entry := range resp.Data.Entries {
			if wantAnswerSource != "" && entry.AnswerSource != wantAnswerSource {
				t.Fatalf("source=%s got answer_source=%s want=%s", source, entry.AnswerSource, wantAnswerSource)
			}
		}
	}

	assertFilter("all", 5, "")
	assertFilter("v2", 1, "v2")
	assertFilter("semantic", 1, "semantic")
	assertFilter("exact_raw", 1, "exact_raw")
	assertFilter("exact_prompt", 1, "exact_prompt")
	assertFilter("cache_v2", 1, "v2")
	assertFilter("cache_semantic", 1, "semantic")
	assertFilter("cache_exact", 1, "exact_prompt")
	assertFilter("provider_chat", 1, "provider_chat")

	normalizedReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-hits?page=0&page_size=0", http.NoBody)
	normalizedRec := httptest.NewRecorder()
	router.ServeHTTP(normalizedRec, normalizedReq)

	if normalizedRec.Code != http.StatusOK {
		t.Fatalf("normalized status=%d body=%s", normalizedRec.Code, normalizedRec.Body.String())
	}

	var normalizedResp hitsResponse
	if err := json.Unmarshal(normalizedRec.Body.Bytes(), &normalizedResp); err != nil {
		t.Fatalf("decode normalized response failed: %v", err)
	}
	if normalizedResp.Data.Page != 1 {
		t.Fatalf("page=%d want=1", normalizedResp.Data.Page)
	}
	if normalizedResp.Data.PageSize != 20 {
		t.Fatalf("page_size=%d want=20", normalizedResp.Data.PageSize)
	}

	maxReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-hits?page=0&page_size=1000", http.NoBody)
	maxRec := httptest.NewRecorder()
	router.ServeHTTP(maxRec, maxReq)

	if maxRec.Code != http.StatusOK {
		t.Fatalf("max page_size status=%d body=%s", maxRec.Code, maxRec.Body.String())
	}

	var maxResp hitsResponse
	if err := json.Unmarshal(maxRec.Body.Bytes(), &maxResp); err != nil {
		t.Fatalf("decode max page_size response failed: %v", err)
	}
	if maxResp.Data.Page != 1 {
		t.Fatalf("max page_size page=%d want=1", maxResp.Data.Page)
	}
	if maxResp.Data.PageSize != 200 {
		t.Fatalf("max page_size page_size=%d want=200", maxResp.Data.PageSize)
	}

	entryMap := map[string]struct {
		userPreview string
		aiPreview   string
	}{}
	for _, entry := range normalizedResp.Data.Entries {
		entryMap[entry.RequestID] = struct {
			userPreview string
			aiPreview   string
		}{
			userPreview: entry.UserMessagePreview,
			aiPreview:   entry.AIResponsePreview,
		}
	}

	if got, ok := entryMap["req-v2"]; !ok {
		t.Fatalf("req-v2 missing from entries")
	} else {
		if got.userPreview != "u-v2" {
			t.Fatalf("req-v2 user_message_preview=%q want=u-v2", got.userPreview)
		}
		if got.aiPreview != "a-v2" {
			t.Fatalf("req-v2 ai_response_preview=%q want=a-v2", got.aiPreview)
		}
	}

	if _, err := time.Parse(time.RFC3339, normalizedResp.Data.WindowStart); err != nil {
		t.Fatalf("parse window_start failed: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, normalizedResp.Data.WindowEnd); err != nil {
		t.Fatalf("parse window_end failed: %v", err)
	}
}

func TestRegisterRoutes_ShouldExposeCacheRequestLevelEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newTraceTestDB(t)
	cacheHandler := NewCacheHandler(cache.NewManagerWithCache(cache.NewMemoryCache()))
	cacheHandler.SetTraceDB(db)

	engine := gin.New()
	adminGroup := engine.Group("/api/admin")

	RegisterRoutes(adminGroup, &Handlers{
		Account:     &AccountHandler{},
		Provider:    &ProviderHandler{},
		Routing:     &RoutingHandler{},
		Cache:       cacheHandler,
		Knowledge:   &KnowledgeHandler{},
		Dashboard:   &DashboardHandler{},
		SmartRouter: &RouterHandler{},
		APIKey:      &APIKeyHandler{},
		Upload:      &UploadHandler{},
		Alert:       &AlertHandler{},
		Feedback:    &FeedbackHandler{},
		Ops:         &OpsHandler{},
		Usage:       &UsageHandler{},
		Settings:    &SettingsHandler{},
		Trace:       &TraceHandler{},
		Edition:     &EditionHandler{},
		VectorDB:    &vectordb.CollectionHandler{},
	})

	statsReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-stats", http.NoBody)
	statsRec := httptest.NewRecorder()
	engine.ServeHTTP(statsRec, statsReq)
	if statsRec.Code == http.StatusNotFound {
		t.Fatalf("expected /api/admin/cache/request-stats to be registered, got 404")
	}

	hitsReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/request-hits", http.NoBody)
	hitsRec := httptest.NewRecorder()
	engine.ServeHTTP(hitsRec, hitsReq)
	if hitsRec.Code == http.StatusNotFound {
		t.Fatalf("expected /api/admin/cache/request-hits to be registered, got 404")
	}
}
