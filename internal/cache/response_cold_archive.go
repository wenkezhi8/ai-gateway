package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultResponseColdArchiveRetryMax = 2

type ResponseColdArchiveStats struct {
	Enabled             bool   `json:"cold_archive_enabled"`
	Mode                string `json:"cold_archive_mode"`
	NearExpirySeconds   int    `json:"cold_archive_near_expiry_seconds"`
	ScanIntervalSeconds int    `json:"cold_archive_scan_interval_seconds"`
	ArchiveEnqueued     int64  `json:"archive_enqueued"`
	ArchiveSucceeded    int64  `json:"archive_succeeded"`
	ArchiveFailed       int64  `json:"archive_failed"`
	ArchiveRetrying     int64  `json:"archive_retrying"`
	ArchiveQueueDepth   int    `json:"archive_queue_depth"`
	ArchiveLastError    string `json:"archive_last_error,omitempty"`
}

type responseColdArchiveJob struct {
	key      string
	response *CachedResponse
	ttl      time.Duration
	attempt  int
}

type ResponseColdArchiveService struct {
	manager         *Manager
	tieredStore     *TieredVectorStore
	embedderFactory func(CacheSettings) EmbeddingProvider
	textNormalizer  *TextNormalizer
	queue           chan responseColdArchiveJob
	retryDelay      time.Duration
	retryMax        int

	startOnce sync.Once
	pendingMu sync.Mutex
	pending   map[string]struct{}
	statsMu   sync.RWMutex
	stats     ResponseColdArchiveStats
}

func NewResponseColdArchiveService(manager *Manager, tieredStore *TieredVectorStore, factory func(CacheSettings) EmbeddingProvider) *ResponseColdArchiveService {
	if factory == nil {
		factory = func(settings CacheSettings) EmbeddingProvider {
			return NewOllamaEmbeddingService(OllamaEmbeddingConfig{
				BaseURL:      settings.VectorOllamaBaseURL,
				Model:        settings.VectorOllamaEmbeddingModel,
				Timeout:      time.Duration(settings.VectorOllamaEmbeddingTimeoutMs) * time.Millisecond,
				EndpointMode: settings.VectorOllamaEndpointMode,
			})
		}
	}
	service := &ResponseColdArchiveService{
		manager:         manager,
		tieredStore:     tieredStore,
		embedderFactory: factory,
		textNormalizer:  NewTextNormalizer(),
		queue:           make(chan responseColdArchiveJob, 256),
		retryDelay:      200 * time.Millisecond,
		retryMax:        defaultResponseColdArchiveRetryMax,
		pending:         map[string]struct{}{},
	}
	service.refreshStatsFromSettings()
	return service
}

func (s *ResponseColdArchiveService) Start(ctx context.Context) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.startOnce.Do(func() {
		go s.workerLoop(ctx)
		go s.scanLoop(ctx)
	})
}

func (s *ResponseColdArchiveService) NotifyWrite(key string, response *CachedResponse, ttl time.Duration) {
	if s == nil || response == nil {
		return
	}
	s.enqueue(responseColdArchiveJob{key: strings.TrimSpace(key), response: cloneCachedResponse(response), ttl: ttl})
}

func (s *ResponseColdArchiveService) ScanExpiringResponses(ctx context.Context) (int, error) {
	if s == nil || s.manager == nil {
		return 0, nil
	}
	settings := s.manager.GetSettings()
	s.refreshStatsFromSettings()
	if !settings.VectorEnabled || !settings.ColdVectorEnabled || !settings.ColdArchiveEnabled {
		return 0, nil
	}
	window := time.Duration(settings.ColdArchiveNearExpirySeconds) * time.Second
	if window <= 0 {
		return 0, nil
	}

	keys := s.manager.Cache().Keys("ai-response:*")
	count := 0
	for _, key := range keys {
		remaining, ok := s.remainingTTL(ctx, key)
		if !ok || remaining <= 0 || remaining > window {
			continue
		}
		cached, err := s.manager.ResponseCache.Get(ctx, key)
		if err != nil || cached == nil {
			continue
		}
		s.enqueue(responseColdArchiveJob{key: key, response: cloneCachedResponse(cached), ttl: remaining})
		count++
	}
	return count, nil
}

func (s *ResponseColdArchiveService) Stats() ResponseColdArchiveStats {
	if s == nil {
		return ResponseColdArchiveStats{}
	}
	s.refreshStatsFromSettings()
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	stats := s.stats
	stats.ArchiveQueueDepth = len(s.queue)
	return stats
}

func (s *ResponseColdArchiveService) workerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.queue:
			s.handleJob(ctx, job)
		}
	}
}

func (s *ResponseColdArchiveService) scanLoop(ctx context.Context) {
	for {
		settings := s.currentSettings()
		interval := time.Duration(settings.ColdArchiveScanIntervalSeconds) * time.Second
		if interval <= 0 {
			interval = 30 * time.Second
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			_, _ = s.ScanExpiringResponses(ctx)
		}
	}
}

func (s *ResponseColdArchiveService) handleJob(ctx context.Context, job responseColdArchiveJob) {
	defer s.releasePending(job.key)
	archiveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.archiveResponse(archiveCtx, job); err != nil {
		s.recordFailure(err)
		if job.attempt < s.retryMax && !isContextError(err) {
			s.recordRetrying()
			nextJob := job
			nextJob.attempt++
			s.requeueLater(nextJob)
			return
		}
		return
	}
	s.recordSuccess()
}

func (s *ResponseColdArchiveService) archiveResponse(ctx context.Context, job responseColdArchiveJob) error {
	settings := s.currentSettings()
	if !settings.VectorEnabled || !settings.ColdVectorEnabled || !settings.ColdArchiveEnabled || s.tieredStore == nil {
		return nil
	}
	response := job.response
	if response == nil || len(response.Body) == 0 {
		return nil
	}
	taskType := normalizeResponseArchiveTaskType(response.TaskType)
	if !shouldArchiveResponseTaskType(settings.ColdArchiveMode, taskType) {
		return nil
	}
	prompt := strings.TrimSpace(response.Prompt)
	if prompt == "" {
		return nil
	}
	normalizedQuery := s.textNormalizer.Normalize(prompt)
	if normalizedQuery == "" {
		normalizedQuery = prompt
	}
	embedder := s.embedderFactory(settings)
	embedding, err := embedder.GetEmbedding(ctx, normalizedQuery)
	if err != nil {
		return err
	}
	if expected := settings.VectorOllamaEmbeddingDimension; expected > 0 && len(embedding) != expected {
		return fmt.Errorf("archive embedding dimension mismatch: got=%d want=%d", len(embedding), expected)
	}
	createTS := response.CreatedAt.Unix()
	if createTS <= 0 {
		createTS = time.Now().Unix()
	}
	ttlSec := int64(job.ttl.Seconds())
	if ttlSec <= 0 {
		ttlSec = int64((24 * time.Hour).Seconds())
	}
	doc := &VectorCacheDocument{
		CacheKey:        buildResponseArchiveCacheKey(taskType, normalizedQuery, response.Provider, response.Model),
		Intent:          taskType,
		TaskType:        taskType,
		NormalizedQuery: normalizedQuery,
		Vector:          embedding,
		Response:        json.RawMessage(response.Body),
		Provider:        response.Provider,
		Model:           response.Model,
		QualityScore:    85,
		CreateTS:        createTS,
		LastHitTS:       createTS,
		ExpireTS:        createTS + ttlSec,
		TTLSec:          ttlSec,
		Tier:            VectorTierCold,
	}
	return s.tieredStore.ArchiveCold(ctx, doc)
}

func (s *ResponseColdArchiveService) enqueue(job responseColdArchiveJob) {
	if s == nil || job.response == nil {
		return
	}
	settings := s.currentSettings()
	if !settings.VectorEnabled || !settings.ColdVectorEnabled || !settings.ColdArchiveEnabled {
		return
	}
	key := strings.TrimSpace(job.key)
	if key == "" {
		key = buildResponseArchiveCacheKey(normalizeResponseArchiveTaskType(job.response.TaskType), strings.TrimSpace(job.response.Prompt), job.response.Provider, job.response.Model)
	}
	job.key = key
	if !s.markPending(key) {
		return
	}
	s.recordEnqueued()
	select {
	case s.queue <- job:
	default:
		s.releasePending(key)
		s.recordFailure(fmt.Errorf("response cold archive queue is full"))
	}
}

func (s *ResponseColdArchiveService) requeueLater(job responseColdArchiveJob) {
	time.AfterFunc(s.retryDelay, func() {
		s.pendingMu.Lock()
		delete(s.pending, job.key)
		s.pendingMu.Unlock()
		s.enqueue(job)
	})
}

func (s *ResponseColdArchiveService) remainingTTL(ctx context.Context, key string) (time.Duration, bool) {
	if rc, ok := s.manager.Cache().(*RedisCache); ok {
		ttl, err := rc.TTL(ctx, key)
		if err != nil || ttl <= 0 {
			return 0, false
		}
		return ttl, true
	}
	if mc, ok := s.manager.Cache().(*MemoryCache); ok {
		meta := mc.GetMeta(key)
		if meta == nil || meta.TTL <= 0 {
			return 0, false
		}
		expiresAt := meta.CreatedAt.Add(time.Duration(meta.TTL) * time.Second)
		remaining := time.Until(expiresAt)
		if remaining <= 0 {
			return 0, false
		}
		return remaining, true
	}
	return 0, false
}

func (s *ResponseColdArchiveService) currentSettings() CacheSettings {
	if s == nil || s.manager == nil {
		return DefaultCacheSettings()
	}
	return s.manager.GetSettings()
}

func (s *ResponseColdArchiveService) refreshStatsFromSettings() {
	settings := s.currentSettings()
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	s.stats.Enabled = settings.ColdArchiveEnabled
	if strings.TrimSpace(settings.ColdArchiveMode) == "" {
		s.stats.Mode = ColdArchiveModeReusable
	} else {
		s.stats.Mode = settings.ColdArchiveMode
	}
	s.stats.NearExpirySeconds = settings.ColdArchiveNearExpirySeconds
	s.stats.ScanIntervalSeconds = settings.ColdArchiveScanIntervalSeconds
}

func (s *ResponseColdArchiveService) markPending(key string) bool {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if _, ok := s.pending[key]; ok {
		return false
	}
	s.pending[key] = struct{}{}
	return true
}

func (s *ResponseColdArchiveService) releasePending(key string) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	delete(s.pending, key)
}

func (s *ResponseColdArchiveService) recordEnqueued() {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	s.stats.ArchiveEnqueued++
}

func (s *ResponseColdArchiveService) recordSuccess() {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	s.stats.ArchiveSucceeded++
}

func (s *ResponseColdArchiveService) recordRetrying() {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	s.stats.ArchiveRetrying++
}

func (s *ResponseColdArchiveService) recordFailure(err error) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	s.stats.ArchiveFailed++
	if err != nil {
		s.stats.ArchiveLastError = err.Error()
	}
}

func normalizeResponseArchiveTaskType(taskType string) string {
	normalized := strings.ToLower(strings.TrimSpace(taskType))
	if normalized == "" {
		return "unknown"
	}
	return normalized
}

func shouldArchiveResponseTaskType(mode, taskType string) bool {
	if strings.ToLower(strings.TrimSpace(mode)) == ColdArchiveModeAll {
		return true
	}
	switch normalizeResponseArchiveTaskType(taskType) {
	case "chat", "creative", "unknown":
		return false
	default:
		return true
	}
}

func buildResponseArchiveCacheKey(taskType, normalizedQuery, provider, model string) string {
	raw := strings.Join([]string{
		normalizeResponseArchiveTaskType(taskType),
		strings.TrimSpace(normalizedQuery),
		strings.TrimSpace(provider),
		strings.TrimSpace(model),
	}, "\n")
	hash := sha256.Sum256([]byte(raw))
	return "response-archive:" + normalizeResponseArchiveTaskType(taskType) + ":" + hex.EncodeToString(hash[:])
}

func cloneCachedResponse(response *CachedResponse) *CachedResponse {
	if response == nil {
		return nil
	}
	copyResp := *response
	copyResp.Body = append(json.RawMessage(nil), response.Body...)
	if response.Headers != nil {
		copyResp.Headers = make(map[string]string, len(response.Headers))
		for key, value := range response.Headers {
			copyResp.Headers[key] = value
		}
	}
	if response.HitModels != nil {
		copyResp.HitModels = make(map[string]int64, len(response.HitModels))
		for key, value := range response.HitModels {
			copyResp.HitModels[key] = value
		}
	}
	return &copyResp
}

func isContextError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "context canceled") || strings.Contains(strings.ToLower(err.Error()), "deadline exceeded")
}
