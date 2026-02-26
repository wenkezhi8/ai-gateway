package routing

import (
	"context"
	"errors"
	"time"
)

type ClassificationSource string

const (
	ClassificationSourceHeuristic ClassificationSource = "heuristic"
	ClassificationSourceOllama    ClassificationSource = "ollama"
	ClassificationSourceFallback  ClassificationSource = "fallback"
)

type ClassifierConfig struct {
	Enabled             bool          `json:"enabled"`
	ShadowMode          bool          `json:"shadow_mode"`
	Provider            string        `json:"provider"`
	BaseURL             string        `json:"base_url"`
	ActiveModel         string        `json:"active_model"`
	CandidateModels     []string      `json:"candidate_models"`
	TimeoutMs           int           `json:"timeout_ms"`
	ConfidenceThreshold float64       `json:"confidence_threshold"`
	FailOpen            bool          `json:"fail_open"`
	MaxInputChars       int           `json:"max_input_chars"`
	Control             ControlConfig `json:"control"`
}

type ControlConfig struct {
	Enable                    bool `json:"enable"`
	ShadowOnly                bool `json:"shadow_only"`
	NormalizedQueryReadEnable bool `json:"normalized_query_read_enable"`
	CacheWriteGateEnable      bool `json:"cache_write_gate_enable"`
	RiskTagEnable             bool `json:"risk_tag_enable"`
	RiskBlockEnable           bool `json:"risk_block_enable"`
	ToolGateEnable            bool `json:"tool_gate_enable"`
	ModelFitEnable            bool `json:"model_fit_enable"`
}

func DefaultClassifierConfig() ClassifierConfig {
	return ClassifierConfig{
		Enabled:             true,
		Provider:            "ollama",
		BaseURL:             "http://127.0.0.1:11434",
		ActiveModel:         "qwen2.5:0.5b-instruct",
		CandidateModels:     []string{"qwen2.5:0.5b-instruct"},
		TimeoutMs:           120,
		ConfidenceThreshold: 0.65,
		FailOpen:            true,
		MaxInputChars:       4000,
		Control: ControlConfig{
			Enable:                    false,
			ShadowOnly:                true,
			NormalizedQueryReadEnable: false,
			CacheWriteGateEnable:      false,
			RiskTagEnable:             false,
			RiskBlockEnable:           false,
			ToolGateEnable:            false,
			ModelFitEnable:            false,
		},
	}
}

type ClassifierHealth struct {
	Healthy    bool   `json:"healthy"`
	Model      string `json:"model"`
	Provider   string `json:"provider"`
	LatencyMs  int64  `json:"latency_ms"`
	Message    string `json:"message"`
	CheckedAt  int64  `json:"checked_at"`
	StatusCode int    `json:"status_code,omitempty"`
}

type ClassifierStats struct {
	TotalRequests          int64   `json:"total_requests"`
	HeuristicOnly          int64   `json:"heuristic_only"`
	ShadowRequests         int64   `json:"shadow_requests"`
	LLMAttempts            int64   `json:"llm_attempts"`
	LLMSuccess             int64   `json:"llm_success"`
	Fallbacks              int64   `json:"fallbacks"`
	LowConfidenceFallbacks int64   `json:"low_confidence_fallbacks"`
	ErrorFallbacks         int64   `json:"error_fallbacks"`
	AvgLLMLatencyMs        float64 `json:"avg_llm_latency_ms"`
	AvgControlLatencyMs    float64 `json:"avg_control_latency_ms"`
	ParseErrors            int64   `json:"parse_errors"`
	ControlFieldsMissing   int64   `json:"control_fields_missing"`
	LastSource             string  `json:"last_source"`
	LastFallbackReason     string  `json:"last_fallback_reason"`
}

var ErrClassifierParseOutput = errors.New("classifier_parse_output")

type TaskClassifier interface {
	Classify(ctx context.Context, prompt, contextText string) (*AssessmentResult, error)
	Health(ctx context.Context) *ClassifierHealth
	UpdateConfig(cfg ClassifierConfig)
	GetConfig() ClassifierConfig
}

func clampClassifierConfig(cfg ClassifierConfig) ClassifierConfig {
	def := DefaultClassifierConfig()
	if cfg.Provider == "" {
		cfg.Provider = def.Provider
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = def.BaseURL
	}
	if cfg.ActiveModel == "" {
		cfg.ActiveModel = def.ActiveModel
	}
	if len(cfg.CandidateModels) == 0 {
		cfg.CandidateModels = append([]string{}, def.CandidateModels...)
	}
	if cfg.TimeoutMs <= 0 {
		cfg.TimeoutMs = def.TimeoutMs
	}
	if cfg.ConfidenceThreshold <= 0 || cfg.ConfidenceThreshold > 1 {
		cfg.ConfidenceThreshold = def.ConfidenceThreshold
	}
	if cfg.MaxInputChars <= 0 {
		cfg.MaxInputChars = def.MaxInputChars
	}
	cfg.Control = clampControlConfig(cfg.Control, def.Control)
	if !cfg.FailOpen {
		cfg.FailOpen = true
	}
	return cfg
}

func clampControlConfig(cfg ControlConfig, def ControlConfig) ControlConfig {
	// Keep user-provided flags as-is. This method exists to centralize
	// defaulting behavior when new control fields are added later.
	_ = def
	return cfg
}

func ClampClassifierConfig(cfg ClassifierConfig) ClassifierConfig {
	return clampClassifierConfig(cfg)
}

func classifierTimeout(cfg ClassifierConfig) time.Duration {
	if cfg.TimeoutMs <= 0 {
		cfg = clampClassifierConfig(cfg)
	}
	return time.Duration(cfg.TimeoutMs) * time.Millisecond
}
