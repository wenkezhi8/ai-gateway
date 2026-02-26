package handler

import (
	"ai-gateway/internal/routing"
	"testing"
	"time"
)

func TestApplyControlTTLBand(t *testing.T) {
	base := 30 * time.Minute
	signals := &routing.ControlSignals{TTLBand: "long"}

	if got := applyControlTTLBand(base, routing.ControlConfig{}, signals); got != base {
		t.Fatalf("expected base ttl when control disabled, got %v", got)
	}

	cfg := routing.ControlConfig{Enable: true, CacheWriteGateEnable: true}
	if got := applyControlTTLBand(base, cfg, signals); got != 7*24*time.Hour {
		t.Fatalf("expected long ttl mapping, got %v", got)
	}

	signals.TTLBand = "medium"
	if got := applyControlTTLBand(base, cfg, signals); got != 24*time.Hour {
		t.Fatalf("expected medium ttl mapping, got %v", got)
	}

	signals.TTLBand = "short"
	if got := applyControlTTLBand(base, cfg, signals); got != time.Hour {
		t.Fatalf("expected short ttl mapping, got %v", got)
	}
}

func TestShouldAllowCacheWrite(t *testing.T) {
	allow := true
	deny := false

	cfg := routing.ControlConfig{Enable: true, CacheWriteGateEnable: true}
	if !shouldAllowCacheWrite(cfg, &routing.ControlSignals{Cacheable: &allow}) {
		t.Fatal("expected write allowed")
	}
	if shouldAllowCacheWrite(cfg, &routing.ControlSignals{Cacheable: &deny}) {
		t.Fatal("expected write denied")
	}

	if !shouldAllowCacheWrite(routing.ControlConfig{}, &routing.ControlSignals{Cacheable: &deny}) {
		t.Fatal("expected write allowed when control disabled")
	}
}

func TestApplyControlToolGate(t *testing.T) {
	req := &ChatCompletionRequest{
		Tools: []Tool{{Type: "function", Function: Function{Name: "lookup"}}},
		ToolChoice: map[string]interface{}{
			"type": "function",
		},
	}

	cfg := routing.ControlConfig{Enable: true, ToolGateEnable: true}
	assessment := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{ToolNeeded: boolPtr(false)}}

	applyControlToolGate(req, cfg, assessment)

	if len(req.Tools) != 0 {
		t.Fatalf("expected tools cleared, got %d", len(req.Tools))
	}
	if req.ToolChoice != nil {
		t.Fatal("expected tool choice cleared")
	}

	req2 := &ChatCompletionRequest{Tools: []Tool{{Type: "function", Function: Function{Name: "lookup"}}}}
	assessment2 := &routing.AssessmentResult{ControlSignals: &routing.ControlSignals{ToolNeeded: boolPtr(true)}}
	applyControlToolGate(req2, cfg, assessment2)
	if len(req2.Tools) == 0 {
		t.Fatal("expected tools preserved when tool_needed=true")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
