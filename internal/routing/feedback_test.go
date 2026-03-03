package routing

import (
	"testing"
	"time"
)

func getStatInt(t *testing.T, stats map[string]interface{}, key string) int {
	t.Helper()
	v, ok := stats[key].(int)
	if !ok {
		t.Fatalf("stat %q type = %T, want int", key, stats[key])
	}
	return v
}

func getStatInt64(t *testing.T, stats map[string]interface{}, key string) int64 {
	t.Helper()
	v, ok := stats[key].(int64)
	if !ok {
		t.Fatalf("stat %q type = %T, want int64", key, stats[key])
	}
	return v
}

func TestFeedbackCollector_RecordFeedback(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	feedback := Feedback{
		Model:        "gpt-4o",
		Provider:     "openai",
		TaskType:     TaskTypeCode,
		Difficulty:   DifficultyMedium,
		FeedbackType: FeedbackPositive,
		Rating:       5,
		LatencyMs:    500,
		TokensUsed:   1000,
	}

	collector.RecordFeedback(feedback)

	perf := collector.GetPerformance("gpt-4o")
	if perf == nil {
		t.Fatal("expected performance data for model")
	}
	if perf.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", perf.TotalRequests)
	}
	if perf.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", perf.SuccessCount)
	}
	if perf.PositiveFeedback != 1 {
		t.Errorf("expected 1 positive feedback, got %d", perf.PositiveFeedback)
	}
}

func TestFeedbackCollector_RecordRequestResult(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// Record successful request
	collector.RecordRequestResult("deepseek-chat", "deepseek", TaskTypeChat, DifficultyLow, true, 200, 500)

	perf := collector.GetPerformance("deepseek-chat")
	if perf == nil {
		t.Fatal("expected performance data")
	}
	if perf.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", perf.SuccessCount)
	}

	// Record failed request
	collector.RecordRequestResult("deepseek-chat", "deepseek", TaskTypeChat, DifficultyLow, false, 0, 0)

	perf = collector.GetPerformance("deepseek-chat")
	if perf.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", perf.FailureCount)
	}
}

func TestFeedbackCollector_GetAllPerformance(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// Record feedback for multiple models
	collector.RecordFeedback(Feedback{Model: "model-a", Provider: "provider-a", FeedbackType: FeedbackPositive})
	collector.RecordFeedback(Feedback{Model: "model-b", Provider: "provider-b", FeedbackType: FeedbackNegative})

	allPerf := collector.GetAllPerformance()
	if len(allPerf) != 2 {
		t.Errorf("expected 2 models, got %d", len(allPerf))
	}
}

func TestFeedbackCollector_GetTopModels(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// Record multiple feedbacks
	for i := 0; i < 5; i++ {
		collector.RecordFeedback(Feedback{
			Model:        "gpt-4o",
			FeedbackType: FeedbackPositive,
			Rating:       5,
			LatencyMs:    200,
		})
	}

	for i := 0; i < 5; i++ {
		collector.RecordFeedback(Feedback{
			Model:        "gpt-3.5-turbo",
			FeedbackType: FeedbackPositive,
			Rating:       3,
			LatencyMs:    500,
		})
	}

	for i := 0; i < 3; i++ {
		collector.RecordFeedback(Feedback{
			Model:        "bad-model",
			FeedbackType: FeedbackNegative,
			Rating:       1,
			LatencyMs:    1000,
		})
	}

	top := collector.GetTopModels(TaskTypeChat, 3)
	if len(top) == 0 {
		t.Error("expected at least one top model")
	}

	// gpt-4o should be top (higher rating, lower latency)
	if len(top) > 0 && top[0] != "gpt-4o" {
		t.Logf("Warning: expected gpt-4o to be top, got %s", top[0])
	}
}

func TestFeedbackCollector_GetFeedbackStats(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// Record various feedback
	collector.RecordFeedback(Feedback{Model: "a", FeedbackType: FeedbackPositive, Rating: 5})
	collector.RecordFeedback(Feedback{Model: "b", FeedbackType: FeedbackPositive, Rating: 4})
	collector.RecordFeedback(Feedback{Model: "c", FeedbackType: FeedbackNegative, Rating: 2})
	collector.RecordFeedback(Feedback{Model: "d", FeedbackType: FeedbackNeutral})

	stats := collector.GetFeedbackStats()

	if got := getStatInt(t, stats, "total_feedback"); got != 4 {
		t.Errorf("expected 4 total feedback, got %d", got)
	}
	if got := getStatInt64(t, stats, "positive_count"); got != 2 {
		t.Errorf("expected 2 positive, got %d", got)
	}
	if got := getStatInt64(t, stats, "negative_count"); got != 1 {
		t.Errorf("expected 1 negative, got %d", got)
	}
	if got := getStatInt(t, stats, "models_tracked"); got != 4 {
		t.Errorf("expected 4 models, got %d", got)
	}
}

func TestFeedbackCollector_TaskTypeStats(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// Record feedback for different task types
	collector.RecordFeedback(Feedback{
		Model:        "model-a",
		TaskType:     TaskTypeCode,
		FeedbackType: FeedbackPositive,
	})
	collector.RecordFeedback(Feedback{
		Model:        "model-a",
		TaskType:     TaskTypeCode,
		FeedbackType: FeedbackNegative,
	})
	collector.RecordFeedback(Feedback{
		Model:        "model-a",
		TaskType:     TaskTypeChat,
		FeedbackType: FeedbackPositive,
	})

	perf := collector.GetPerformance("model-a")
	if perf == nil {
		t.Fatal("expected performance data")
	}

	codeStats := perf.TaskTypeStats[string(TaskTypeCode)]
	if codeStats == nil {
		t.Fatal("expected code task stats")
	}
	if codeStats.TotalRequests != 2 {
		t.Errorf("expected 2 code requests, got %d", codeStats.TotalRequests)
	}
	if codeStats.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", codeStats.SuccessCount)
	}

	chatStats := perf.TaskTypeStats[string(TaskTypeChat)]
	if chatStats == nil {
		t.Fatal("expected chat task stats")
	}
	if chatStats.TotalRequests != 1 {
		t.Errorf("expected 1 chat request, got %d", chatStats.TotalRequests)
	}
}

func TestFeedbackCollector_GetRecentFeedback(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// Record multiple feedback
	for i := 0; i < 10; i++ {
		collector.RecordFeedback(Feedback{
			Model:        "model",
			FeedbackType: FeedbackPositive,
		})
	}

	recent := collector.GetRecentFeedback(5)
	if len(recent) != 5 {
		t.Errorf("expected 5 recent feedback, got %d", len(recent))
	}
}

func TestFeedbackCollector_ClearOldFeedback(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// Record feedback with specific timestamps
	collector.mu.Lock()
	collector.feedback = append(collector.feedback,
		Feedback{
			Model:     "old",
			Timestamp: time.Now().Add(-48 * time.Hour),
		},
		Feedback{
			Model:     "new",
			Timestamp: time.Now().Add(-1 * time.Hour),
		},
	)
	collector.mu.Unlock()

	cleared := collector.ClearOldFeedback(24 * time.Hour)
	if cleared != 1 {
		t.Errorf("expected 1 cleared, got %d", cleared)
	}

	stats := collector.GetFeedbackStats()
	if got := getStatInt(t, stats, "total_feedback"); got != 1 {
		t.Errorf("expected 1 remaining feedback, got %d", got)
	}
}

func TestFeedbackCollector_CalculatePerformanceScore(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)

	// High performing model
	highPerf := &ModelPerformance{
		Model:            "high-perf",
		TotalRequests:    100,
		SuccessCount:     95,
		PositiveFeedback: 90,
		NegativeFeedback: 5,
		AvgRating:        4.5,
		AvgLatencyMs:     200,
		TaskTypeStats:    make(map[string]*TaskTypeStat),
	}
	highPerf.TaskTypeStats["code"] = &TaskTypeStat{SuccessRate: 0.95}

	// Low performing model
	lowPerf := &ModelPerformance{
		Model:            "low-perf",
		TotalRequests:    100,
		SuccessCount:     60,
		PositiveFeedback: 40,
		NegativeFeedback: 50,
		AvgRating:        2.0,
		AvgLatencyMs:     2000,
		TaskTypeStats:    make(map[string]*TaskTypeStat),
	}
	lowPerf.TaskTypeStats["code"] = &TaskTypeStat{SuccessRate: 0.6}

	highScore := collector.calculatePerformanceScore(highPerf, TaskTypeCode)
	lowScore := collector.calculatePerformanceScore(lowPerf, TaskTypeCode)

	if highScore <= lowScore {
		t.Errorf("expected high perf score > low perf score, got %f vs %f", highScore, lowScore)
	}
}

func TestFeedbackCollector_MaxFeedback(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)
	collector.maxFeedback = 5

	// Record more than max
	for i := 0; i < 10; i++ {
		collector.RecordFeedback(Feedback{
			Model:        "model",
			FeedbackType: FeedbackPositive,
		})
	}

	if len(collector.feedback) > collector.maxFeedback {
		t.Errorf("expected feedback <= %d, got %d", collector.maxFeedback, len(collector.feedback))
	}
}

func TestFeedbackCollector_GetTaskTypeDistribution_ShouldUseShortTTLCacheUntilForcedRefresh(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)
	collector.distributionCacheTTL = time.Hour

	collector.RecordFeedback(Feedback{
		Model:        "cache-model",
		TaskType:     TaskTypeCode,
		FeedbackType: FeedbackPositive,
	})

	first := collector.GetTaskTypeDistribution()
	if got := getTaskTypeCount(first, string(TaskTypeCode)); got != 1 {
		t.Fatalf("expected initial code count 1, got %d", got)
	}

	collector.mu.Lock()
	collector.performance["cache-model"].TaskTypeStats[string(TaskTypeCode)].TotalRequests = 9
	collector.mu.Unlock()

	cached := collector.GetTaskTypeDistribution()
	if got := getTaskTypeCount(cached, string(TaskTypeCode)); got != 1 {
		t.Fatalf("expected cached code count 1, got %d", got)
	}

	fresh := collector.GetTaskTypeDistributionCached(true)
	if got := getTaskTypeCount(fresh, string(TaskTypeCode)); got != 9 {
		t.Fatalf("expected refreshed code count 9, got %d", got)
	}
}

func TestFeedbackCollector_GetTaskTypeDistribution_ShouldInvalidateCacheAfterRecordFeedback(t *testing.T) {
	assessor := NewDifficultyAssessor()
	router := NewSmartRouter()
	collector := NewFeedbackCollector(assessor, router)
	collector.distributionCacheTTL = time.Hour

	collector.RecordFeedback(Feedback{
		Model:        "invalidate-model",
		TaskType:     TaskTypeCode,
		FeedbackType: FeedbackPositive,
	})

	_ = collector.GetTaskTypeDistribution()

	collector.RecordFeedback(Feedback{
		Model:        "invalidate-model",
		TaskType:     TaskTypeChat,
		FeedbackType: FeedbackPositive,
	})

	latest := collector.GetTaskTypeDistribution()
	if got := getTaskTypeCount(latest, string(TaskTypeCode)); got != 1 {
		t.Fatalf("expected code count 1 after invalidation, got %d", got)
	}
	if got := getTaskTypeCount(latest, string(TaskTypeChat)); got != 1 {
		t.Fatalf("expected chat count 1 after invalidation, got %d", got)
	}
}

func getTaskTypeCount(distribution []TaskTypeDistribution, taskType string) int64 {
	for _, item := range distribution {
		if item.TaskType == taskType {
			return item.Count
		}
	}
	return 0
}
