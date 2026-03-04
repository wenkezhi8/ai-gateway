package routing

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

type HybridTaskClassifier struct {
	mu         sync.RWMutex
	assessor   *DifficultyAssessor
	classifier TaskClassifier
	cfg        ClassifierConfig
	statsMu    sync.RWMutex
	stats      ClassifierStats
}

//nolint:gocritic // keep value-type config API for compatibility.
func NewHybridTaskClassifier(assessor *DifficultyAssessor, cfg ClassifierConfig) *HybridTaskClassifier {
	cfg = clampClassifierConfig(cfg)
	return &HybridTaskClassifier{
		assessor:   assessor,
		classifier: NewOllamaTaskClassifier(cfg),
		cfg:        cfg,
	}
}

//nolint:gocritic // keep value-type config API for compatibility.
func (h *HybridTaskClassifier) UpdateConfig(cfg ClassifierConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg = clampClassifierConfig(cfg)
	h.classifier.UpdateConfig(h.cfg)
}

func (h *HybridTaskClassifier) GetConfig() ClassifierConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.cfg
}

func (h *HybridTaskClassifier) SwitchModel(model string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg.ActiveModel = strings.TrimSpace(model)
	h.cfg = clampClassifierConfig(h.cfg)
	h.classifier.UpdateConfig(h.cfg)
}

func (h *HybridTaskClassifier) Health(ctx context.Context) *ClassifierHealth {
	h.mu.RLock()
	classifier := h.classifier
	h.mu.RUnlock()
	return classifier.Health(ctx)
}

func (h *HybridTaskClassifier) Classify(ctx context.Context, prompt, contextText string) *AssessmentResult {
	h.recordTotal()
	h.mu.RLock()
	classifier := h.classifier
	cfg := h.cfg
	h.mu.RUnlock()

	sanitizedPrompt := SanitizeIntentInput(prompt)
	sanitizedContext := SanitizeIntentInput(contextText)
	if IsShortGreetingIntent(sanitizedPrompt) {
		h.recordHeuristicOnly("greeting_short_circuit")
		result := h.assessor.AssessWithResult(sanitizedPrompt, sanitizedContext)
		if result == nil {
			result = &AssessmentResult{Dimensions: map[string]float64{}}
		}
		result.TaskType = TaskTypeChat
		result.Difficulty = DifficultyLow
		if result.Confidence < 0.90 {
			result.Confidence = 0.90
		}
		result.Source = ClassificationSourceHeuristic
		result.FallbackReason = "greeting_short_circuit"
		result.SuggestedTTL = h.assessor.getSuggestedTTL(TaskTypeChat, DifficultyLow)
		if strings.TrimSpace(result.SemanticSignature) == "" {
			result.SemanticSignature = buildFallbackSignature(TaskTypeChat, sanitizedPrompt)
		}
		return result
	}

	if !cfg.Enabled {
		h.recordHeuristicOnly("classifier_disabled")
		return h.fallback(sanitizedPrompt, sanitizedContext, ClassificationSourceHeuristic, "classifier_disabled")
	}
	if cfg.ShadowMode {
		h.recordShadowMode()
		go h.shadowClassify(sanitizedPrompt, sanitizedContext)
		return h.fallback(sanitizedPrompt, sanitizedContext, ClassificationSourceHeuristic, "shadow_mode")
	}

	start := time.Now()
	h.recordLLMAttempt()
	result, err := classifier.Classify(ctx, sanitizedPrompt, sanitizedContext)
	latencyMs := time.Since(start).Milliseconds()
	h.recordLLMLatency(latencyMs)
	if err != nil {
		if errors.Is(err, ErrClassifierParseOutput) {
			h.recordParseError()
		}
		h.recordFallback("classifier_error")
		if cfg.FailOpen {
			return h.fallback(sanitizedPrompt, sanitizedContext, ClassificationSourceFallback, "classifier_error")
		}
		return h.fallback(sanitizedPrompt, sanitizedContext, ClassificationSourceHeuristic, "classifier_error")
	}
	if result == nil {
		h.recordFallback("classifier_nil_result")
		return h.fallback(sanitizedPrompt, sanitizedContext, ClassificationSourceFallback, "classifier_nil_result")
	}
	if result.Confidence < cfg.ConfidenceThreshold {
		h.recordLowConfidenceFallback()
		return h.fallback(sanitizedPrompt, sanitizedContext, ClassificationSourceFallback, "low_confidence")
	}
	if result.TaskType == TaskTypeUnknown {
		h.recordFallback("unknown_task_type")
		return h.fallback(sanitizedPrompt, sanitizedContext, ClassificationSourceFallback, "unknown_task_type")
	}
	h.recordControlCoverage(result)
	h.recordControlLatency(latencyMs)
	h.recordLLMSuccess(string(ClassificationSourceOllama))
	result.SuggestedTTL = h.assessor.getSuggestedTTL(result.TaskType, result.Difficulty)
	if result.SemanticSignature == "" {
		result.SemanticSignature = buildFallbackSignature(result.TaskType, sanitizedPrompt)
	}
	if result.Source == "" {
		result.Source = ClassificationSourceOllama
	}
	return result
}

func (h *HybridTaskClassifier) GetStats() ClassifierStats {
	h.statsMu.RLock()
	defer h.statsMu.RUnlock()
	return h.stats
}

func (h *HybridTaskClassifier) shadowClassify(prompt, contextText string) {
	h.mu.RLock()
	classifier := h.classifier
	cfg := h.cfg
	h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), classifierTimeout(cfg))
	defer cancel()

	start := time.Now()
	h.recordLLMAttempt()
	result, err := classifier.Classify(ctx, prompt, contextText)
	h.recordLLMLatency(time.Since(start).Milliseconds())
	if err != nil || result == nil {
		h.recordErrorFallback()
		return
	}
	h.recordLLMSuccess(string(ClassificationSourceOllama))
}

func (h *HybridTaskClassifier) recordTotal() {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.TotalRequests++
}

func (h *HybridTaskClassifier) recordHeuristicOnly(reason string) {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.HeuristicOnly++
	h.stats.LastSource = string(ClassificationSourceHeuristic)
	h.stats.LastFallbackReason = reason
}

func (h *HybridTaskClassifier) recordShadowMode() {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.ShadowRequests++
	h.stats.HeuristicOnly++
	h.stats.LastSource = string(ClassificationSourceHeuristic)
	h.stats.LastFallbackReason = "shadow_mode"
}

func (h *HybridTaskClassifier) recordLLMAttempt() {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.LLMAttempts++
}

func (h *HybridTaskClassifier) recordLLMSuccess(source string) {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.LLMSuccess++
	h.stats.LastSource = source
	h.stats.LastFallbackReason = ""
}

func (h *HybridTaskClassifier) recordFallback(reason string) {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.Fallbacks++
	h.stats.ErrorFallbacks++
	h.stats.LastSource = string(ClassificationSourceFallback)
	h.stats.LastFallbackReason = reason
}

func (h *HybridTaskClassifier) recordLowConfidenceFallback() {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.Fallbacks++
	h.stats.LowConfidenceFallbacks++
	h.stats.LastSource = string(ClassificationSourceFallback)
	h.stats.LastFallbackReason = "low_confidence"
}

func (h *HybridTaskClassifier) recordErrorFallback() {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.Fallbacks++
	h.stats.ErrorFallbacks++
	h.stats.LastSource = string(ClassificationSourceFallback)
	h.stats.LastFallbackReason = "classifier_error"
}

func (h *HybridTaskClassifier) recordLLMLatency(latencyMs int64) {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	if h.stats.AvgLLMLatencyMs == 0 {
		h.stats.AvgLLMLatencyMs = float64(latencyMs)
		return
	}
	h.stats.AvgLLMLatencyMs = (h.stats.AvgLLMLatencyMs + float64(latencyMs)) / 2
}

func (h *HybridTaskClassifier) recordControlLatency(latencyMs int64) {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	if h.stats.AvgControlLatencyMs == 0 {
		h.stats.AvgControlLatencyMs = float64(latencyMs)
		return
	}
	h.stats.AvgControlLatencyMs = (h.stats.AvgControlLatencyMs + float64(latencyMs)) / 2
}

func (h *HybridTaskClassifier) recordParseError() {
	h.statsMu.Lock()
	defer h.statsMu.Unlock()
	h.stats.ParseErrors++
}

func (h *HybridTaskClassifier) recordControlCoverage(result *AssessmentResult) {
	if result == nil || result.ControlSignals == nil {
		h.statsMu.Lock()
		h.stats.ControlFieldsMissing++
		h.statsMu.Unlock()
		return
	}
	if strings.TrimSpace(result.ControlSignals.NormalizedQuery) == "" {
		h.statsMu.Lock()
		h.stats.ControlFieldsMissing++
		h.statsMu.Unlock()
	}
}

func (h *HybridTaskClassifier) fallback(prompt, contextText string, source ClassificationSource, reason string) *AssessmentResult {
	base := h.assessor.AssessWithResult(prompt, contextText)
	if base == nil {
		base = &AssessmentResult{
			TaskType:     TaskTypeUnknown,
			Difficulty:   DifficultyMedium,
			Confidence:   0.5,
			Dimensions:   map[string]float64{},
			SuggestedTTL: 24 * time.Hour,
		}
	}
	base.Source = source
	base.FallbackReason = reason
	if strings.TrimSpace(base.SemanticSignature) == "" {
		base.SemanticSignature = buildFallbackSignature(base.TaskType, prompt)
	}
	return base
}
