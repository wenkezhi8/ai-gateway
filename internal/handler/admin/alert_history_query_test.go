package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type alertHistoryResponse struct {
	Success bool `json:"success"`
	Data    struct {
		List     []AlertRecord `json:"list"`
		Total    int           `json:"total"`
		Page     int           `json:"page"`
		PageSize int           `json:"pageSize"`
	} `json:"data"`
}

func TestAlertHandler_GetHistory_ShouldSupportQueryCombination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestAlertHandler(t)
	handler.alerts = []AlertRecord{
		{ID: "a1", Time: "2026-01-01T10:00:00Z", Level: "warning", Source: "system", Message: "cpu high early", Status: "pending"},
		{ID: "a2", Time: "2026-01-02T10:00:00Z", Level: "warning", Source: "system", Message: "cpu high resolved", Status: "resolved"},
		{ID: "a3", Time: "2026-01-03T10:00:00Z", Level: "info", Source: "system", Message: "cpu high info", Status: "pending"},
		{ID: "a4", Time: "2026-01-04T10:00:00Z", Level: "warning", Source: "app", Message: "cpu high app", Status: "pending"},
		{ID: "a5", Time: "2026-01-05T10:00:00Z", Level: "warning", Source: "system", Message: "memory high", Status: "pending"},
		{ID: "a6", Time: "2026-01-06T10:00:00Z", Level: "warning", Source: "system", Message: "cpu high latest", Status: "pending"},
	}

	router := gin.New()
	router.GET("/api/admin/alerts/history", handler.GetHistory)

	url := "/api/admin/alerts/history?level=warning&status=pending&source=system&keyword=cpu&startAt=2026-01-01T00:00:00Z&endAt=2026-01-07T00:00:00Z&sortBy=time&order=asc&page=1&pageSize=2"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp alertHistoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if resp.Data.Total != 2 {
		t.Fatalf("expected total=2, got %d", resp.Data.Total)
	}
	if resp.Data.Page != 1 {
		t.Fatalf("expected page=1, got %d", resp.Data.Page)
	}
	if resp.Data.PageSize != 2 {
		t.Fatalf("expected pageSize=2, got %d", resp.Data.PageSize)
	}
	if len(resp.Data.List) != 2 {
		t.Fatalf("expected list size=2, got %d", len(resp.Data.List))
	}
	if resp.Data.List[0].ID != "a1" || resp.Data.List[1].ID != "a6" {
		t.Fatalf("expected asc order [a1, a6], got [%s, %s]", resp.Data.List[0].ID, resp.Data.List[1].ID)
	}
}

func TestAlertHandler_GetHistory_ShouldKeepStartDateEndDateCompatibility(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestAlertHandler(t)
	handler.alerts = []AlertRecord{
		{ID: "a1", Time: "2026-01-01T10:00:00Z", Level: "warning", Source: "system", Message: "one", Status: "pending"},
		{ID: "a2", Time: "2026-01-02T10:00:00Z", Level: "warning", Source: "system", Message: "two", Status: "pending"},
		{ID: "a3", Time: "2026-01-03T10:00:00Z", Level: "warning", Source: "system", Message: "three", Status: "pending"},
		{ID: "a4", Time: "2026-01-04T10:00:00Z", Level: "warning", Source: "system", Message: "four", Status: "pending"},
	}

	router := gin.New()
	router.GET("/api/admin/alerts/history", handler.GetHistory)

	url := "/api/admin/alerts/history?startDate=2026-01-02&endDate=2026-01-03&sortBy=time&order=asc"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp alertHistoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Data.Total != 2 {
		t.Fatalf("expected total=2, got %d", resp.Data.Total)
	}
	if resp.Data.Page != 1 {
		t.Fatalf("expected default page=1, got %d", resp.Data.Page)
	}
	if resp.Data.PageSize != 20 {
		t.Fatalf("expected default pageSize=20, got %d", resp.Data.PageSize)
	}
	if len(resp.Data.List) != 2 {
		t.Fatalf("expected list size=2, got %d", len(resp.Data.List))
	}
	if resp.Data.List[0].ID != "a2" || resp.Data.List[1].ID != "a3" {
		t.Fatalf("expected alias filtered ids [a2, a3], got [%s, %s]", resp.Data.List[0].ID, resp.Data.List[1].ID)
	}
}

func TestAlertHandler_GetHistory_ShouldFallbackToDefaultForInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestAlertHandler(t)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	handler.alerts = make([]AlertRecord, 0, 30)
	for i := 0; i < 30; i++ {
		handler.alerts = append(handler.alerts, AlertRecord{
			ID:      "a" + strconv.Itoa(i+1),
			Time:    base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
			Level:   "warning",
			Source:  "system",
			Message: "cpu",
			Status:  "pending",
		})
	}

	router := gin.New()
	router.GET("/api/admin/alerts/history", handler.GetHistory)

	url := "/api/admin/alerts/history?page=bad&pageSize=0&sortBy=unknown&order=unknown"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp alertHistoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Data.Page != 1 {
		t.Fatalf("expected default page=1, got %d", resp.Data.Page)
	}
	if resp.Data.PageSize != 20 {
		t.Fatalf("expected default pageSize=20, got %d", resp.Data.PageSize)
	}
	if resp.Data.Total != 30 {
		t.Fatalf("expected total=30, got %d", resp.Data.Total)
	}
	if len(resp.Data.List) != 20 {
		t.Fatalf("expected default page size list=20, got %d", len(resp.Data.List))
	}
	if resp.Data.List[0].ID != "a30" {
		t.Fatalf("expected default time desc starts with a30, got %s", resp.Data.List[0].ID)
	}
}

func TestAlertHandler_GetHistory_ShouldCapPageSizeAt100(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestAlertHandler(t)

	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	handler.alerts = make([]AlertRecord, 0, 120)
	for i := 0; i < 120; i++ {
		handler.alerts = append(handler.alerts, AlertRecord{
			ID:      "m" + strconv.Itoa(i+1),
			Time:    base.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			Level:   "warning",
			Source:  "system",
			Message: "cpu",
			Status:  "pending",
		})
	}

	router := gin.New()
	router.GET("/api/admin/alerts/history", handler.GetHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/alerts/history?pageSize=120", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp alertHistoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Data.PageSize != 100 {
		t.Fatalf("expected pageSize capped to 100, got %d", resp.Data.PageSize)
	}
	if len(resp.Data.List) != 100 {
		t.Fatalf("expected list size=100, got %d", len(resp.Data.List))
	}
}
