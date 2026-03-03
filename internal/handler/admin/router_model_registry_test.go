package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

func TestRouterHandler_ModelRegistryEndpoints_ShouldListUpsertDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	smartRouter := routing.NewSmartRouter()
	smartRouter.UpdateModelScore("gpt-4o", &routing.ModelScore{
		Model:        "gpt-4o",
		Provider:     "openai",
		DisplayName:  "GPT-4o",
		QualityScore: 91,
		SpeedScore:   88,
		CostScore:    70,
		Enabled:      true,
	})

	handler := &RouterHandler{router: smartRouter}
	r := gin.New()
	r.GET("/api/admin/router/model-registry", handler.GetModelRegistry)
	r.PUT("/api/admin/router/model-registry/:model", handler.UpsertModelRegistry)
	r.DELETE("/api/admin/router/model-registry/:model", handler.DeleteModelRegistry)

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/router/model-registry", http.NoBody)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Success bool `json:"success"`
		Data    []struct {
			Model       string `json:"model"`
			Provider    string `json:"provider"`
			DisplayName string `json:"display_name"`
			Enabled     bool   `json:"enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if !listResp.Success {
		t.Fatalf("expected success=true")
	}
	if len(listResp.Data) == 0 {
		t.Fatalf("expected at least one model")
	}

	upsertBody := bytes.NewBufferString(`{"provider":"qwen","display_name":"Qwen Max","enabled":true}`)
	upsertReq := httptest.NewRequest(http.MethodPut, "/api/admin/router/model-registry/qwen-max", upsertBody)
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertRec := httptest.NewRecorder()
	r.ServeHTTP(upsertRec, upsertReq)
	if upsertRec.Code != http.StatusOK {
		t.Fatalf("upsert status=%d body=%s", upsertRec.Code, upsertRec.Body.String())
	}

	if score := smartRouter.GetModelScore("qwen-max"); score == nil {
		t.Fatalf("expected qwen-max to exist after upsert")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/router/model-registry/qwen-max", http.NoBody)
	deleteRec := httptest.NewRecorder()
	r.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	if score := smartRouter.GetModelScore("qwen-max"); score != nil {
		t.Fatalf("expected qwen-max to be deleted")
	}
}

func TestRouterHandler_UpsertModelRegistry_ShouldValidateProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &RouterHandler{router: routing.NewSmartRouter()}
	r := gin.New()
	r.PUT("/api/admin/router/model-registry/:model", handler.UpsertModelRegistry)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/router/model-registry/gpt-4o", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouterHandler_ModelRegistry_ShouldKeepExistingInternalScoresWhenUpsert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	smartRouter := routing.NewSmartRouter()
	smartRouter.UpdateModelScore("gpt-4o", &routing.ModelScore{
		Model:        "gpt-4o",
		Provider:     "openai",
		DisplayName:  "GPT-4o",
		QualityScore: 11,
		SpeedScore:   22,
		CostScore:    33,
		Enabled:      true,
	})

	handler := &RouterHandler{router: smartRouter}
	r := gin.New()
	r.PUT("/api/admin/router/model-registry/:model", handler.UpsertModelRegistry)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/router/model-registry/gpt-4o", bytes.NewBufferString(`{"provider":"openai","display_name":"GPT-4o New","enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	score := smartRouter.GetModelScore("gpt-4o")
	if score == nil {
		t.Fatalf("expected model score to exist")
	}
	if score.QualityScore != 11 || score.SpeedScore != 22 || score.CostScore != 33 {
		t.Fatalf("expected internal scores unchanged, got q=%d s=%d c=%d", score.QualityScore, score.SpeedScore, score.CostScore)
	}
	if score.DisplayName != "GPT-4o New" {
		t.Fatalf("expected display name updated, got %s", score.DisplayName)
	}
	if score.Enabled {
		t.Fatalf("expected enabled=false after upsert")
	}
}
