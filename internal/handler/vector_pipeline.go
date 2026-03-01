package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/intent"
	"ai-gateway/internal/routing"
)

type VectorPipeline struct {
	vectorStore    cache.VectorCacheStore
	textNormalizer *cache.TextNormalizer
	getSettings    func() cache.CacheSettings
}

func NewVectorPipeline(
	vectorStore cache.VectorCacheStore,
	textNormalizer *cache.TextNormalizer,
	getSettings func() cache.CacheSettings,
) *VectorPipeline {
	if textNormalizer == nil {
		textNormalizer = cache.NewTextNormalizer()
	}
	if getSettings == nil {
		getSettings = cache.DefaultCacheSettings
	}
	return &VectorPipeline{
		vectorStore:    vectorStore,
		textNormalizer: textNormalizer,
		getSettings:    getSettings,
	}
}

//nolint:gocyclo,gocritic
func (p *VectorPipeline) Read(
	ctx context.Context,
	prompt string,
	normalizedQuery string,
	taskType string,
	settings *cache.CacheSettings,
	thresholdResolver func(string, cache.CacheSettings) float64,
) (*intent.EmbeddingResult, []byte, bool, string, string) {
	resolvedSettings := p.resolveSettings(settings)
	if p.vectorStore == nil || !resolvedSettings.VectorEnabled || !resolvedSettings.VectorPipelineEnabled {
		return nil, nil, false, "", ""
	}

	intentResult, err := p.buildIntentEmbeddingResult(ctx, prompt, normalizedQuery, taskType, resolvedSettings)
	if err != nil || intentResult == nil || strings.TrimSpace(intentResult.StandardKey) == "" {
		return nil, nil, false, "", ""
	}

	exactDoc, err := p.vectorStore.GetExact(ctx, intentResult.StandardKey)
	if err == nil && exactDoc != nil && exactDoc.Response != nil {
		if payload, mErr := json.Marshal(exactDoc.Response); mErr == nil && len(payload) > 0 {
			return intentResult, payload, true, "vector-exact", exactDoc.CacheKey
		}
	}

	threshold := 0.92
	if thresholdResolver != nil {
		threshold = thresholdResolver(intentResult.Intent, resolvedSettings)
	}
	hits, err := p.vectorStore.VectorSearch(ctx, intentResult.Intent, intentResult.Embedding, 1, threshold)
	if err != nil || len(hits) == 0 {
		return intentResult, nil, false, "", ""
	}

	hit := hits[0]
	payload := hit.Response
	if len(payload) == 0 && strings.TrimSpace(hit.CacheKey) != "" {
		if doc, gErr := p.vectorStore.GetExact(ctx, hit.CacheKey); gErr == nil && doc != nil && doc.Response != nil {
			if b, mErr := json.Marshal(doc.Response); mErr == nil {
				payload = b
			}
		}
	}
	if len(payload) == 0 {
		return intentResult, nil, false, "", ""
	}
	return intentResult, payload, true, "vector-semantic", hit.CacheKey
}

//nolint:gocritic
func (p *VectorPipeline) Write(
	ctx context.Context,
	intentResult *intent.EmbeddingResult,
	providerName string,
	model string,
	taskType routing.TaskType,
	response ChatCompletionResponse,
	ttlSec int64,
	settings *cache.CacheSettings,
) {
	resolvedSettings := p.resolveSettings(settings)
	if intentResult == nil || p.vectorStore == nil || !resolvedSettings.VectorEnabled || !resolvedSettings.VectorPipelineEnabled || !resolvedSettings.VectorWritebackEnabled {
		return
	}

	intentName := strings.ToLower(strings.TrimSpace(intentResult.Intent))
	if intentName == "" || intentName == "unknown" {
		return
	}
	if strings.TrimSpace(intentResult.StandardKey) == "" || len(intentResult.Embedding) == 0 || !hasMeaningfulAssistantResponse(&response) {
		return
	}
	if ttlSec <= 0 {
		ttlSec = int64((24 * time.Hour).Seconds())
	}

	doc := &cache.VectorCacheDocument{
		CacheKey:        intentResult.StandardKey,
		Intent:          intentResult.Intent,
		TaskType:        string(taskType),
		Slots:           intentResult.Slots,
		NormalizedQuery: intentResult.NormalizedText,
		Vector:          intentResult.Embedding,
		Response:        response,
		Provider:        providerName,
		Model:           model,
		QualityScore:    90,
		TTLSec:          ttlSec,
	}

	go func() {
		writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := p.vectorStore.Upsert(writeCtx, doc); err != nil {
			return
		}
		if err := p.vectorStore.TouchTTL(writeCtx, doc.CacheKey, ttlSec); err != nil {
			return
		}
	}()
}

func (p *VectorPipeline) resolveSettings(settings *cache.CacheSettings) cache.CacheSettings {
	if settings != nil {
		return *settings
	}
	if p.getSettings != nil {
		return p.getSettings()
	}
	return cache.DefaultCacheSettings()
}

//nolint:gocritic
func (p *VectorPipeline) buildIntentEmbeddingResult(
	ctx context.Context,
	prompt string,
	normalizedQuery string,
	taskType string,
	settings cache.CacheSettings,
) (*intent.EmbeddingResult, error) {
	normalizedText := strings.TrimSpace(normalizedQuery)
	if normalizedText == "" {
		if p.textNormalizer != nil {
			normalizedText = p.textNormalizer.Normalize(prompt)
		}
		if normalizedText == "" {
			normalizedText = strings.TrimSpace(prompt)
		}
	}
	if normalizedText == "" {
		return nil, nil
	}

	normalizedTaskType := strings.ToLower(strings.TrimSpace(taskType))
	if normalizedTaskType == "" {
		normalizedTaskType = "unknown"
	}

	embedder := cache.NewOllamaEmbeddingService(cache.OllamaEmbeddingConfig{
		BaseURL:      settings.VectorOllamaBaseURL,
		Model:        settings.VectorOllamaEmbeddingModel,
		Timeout:      time.Duration(settings.VectorOllamaEmbeddingTimeoutMs) * time.Millisecond,
		EndpointMode: settings.VectorOllamaEndpointMode,
	})

	embedding, err := embedder.GetEmbedding(ctx, normalizedText)
	if err != nil {
		return nil, err
	}
	if expected := settings.VectorOllamaEmbeddingDimension; expected > 0 && len(embedding) != expected {
		return nil, nil
	}

	hash := sha256.Sum256([]byte(normalizedText))
	queryHash := hex.EncodeToString(hash[:])

	return &intent.EmbeddingResult{
		Intent:         normalizedTaskType,
		Slots:          map[string]string{"task_type": normalizedTaskType, "query_hash": queryHash},
		StandardKey:    cache.BuildTaskTypeStandardKey(normalizedTaskType, normalizedText),
		Embedding:      embedding,
		EmbeddingDim:   len(embedding),
		Confidence:     1,
		EngineVersion:  "ollama-vector-pipeline/v1",
		NormalizedText: normalizedText,
	}, nil
}
