// Package routing provides feedback collection for continuous optimization
// 改动点: 新增效果评估闭环模块，自动收集反馈优化路由规则
package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// FeedbackType represents the type of feedback
type FeedbackType string

const (
	FeedbackPositive FeedbackType = "positive"
	FeedbackNegative FeedbackType = "negative"
	FeedbackNeutral  FeedbackType = "neutral"
)

// Feedback represents user feedback on a response
type Feedback struct {
	ID           string          `json:"id"`
	RequestID    string          `json:"request_id"`
	Model        string          `json:"model"`
	Provider     string          `json:"provider"`
	TaskType     TaskType        `json:"task_type"`
	Difficulty   DifficultyLevel `json:"difficulty"`
	FeedbackType FeedbackType    `json:"feedback_type"`
	Rating       int             `json:"rating"` // 1-5
	LatencyMs    int64           `json:"latency_ms"`
	TokensUsed   int             `json:"tokens_used"`
	Timestamp    time.Time       `json:"timestamp"`
	Comment      string          `json:"comment,omitempty"`
	UserID       string          `json:"user_id,omitempty"`
}

// ModelPerformance tracks performance metrics for a model
type ModelPerformance struct {
	Model            string                   `json:"model"`
	Provider         string                   `json:"provider"`
	TotalRequests    int64                    `json:"total_requests"`
	SuccessCount     int64                    `json:"success_count"`
	FailureCount     int64                    `json:"failure_count"`
	AvgLatencyMs     int64                    `json:"avg_latency_ms"`
	AvgRating        float64                  `json:"avg_rating"`
	PositiveFeedback int64                    `json:"positive_feedback"`
	NegativeFeedback int64                    `json:"negative_feedback"`
	TaskTypeStats    map[string]*TaskTypeStat `json:"task_type_stats"`
	LastUpdated      time.Time                `json:"last_updated"`
}

// TaskTypeStat tracks stats for a specific task type
type TaskTypeStat struct {
	TotalRequests int64   `json:"total_requests"`
	SuccessCount  int64   `json:"success_count"`
	AvgLatencyMs  int64   `json:"avg_latency_ms"`
	AvgRating     float64 `json:"avg_rating"`
	SuccessRate   float64 `json:"success_rate"`
}

// FeedbackCollector collects and analyzes feedback
type FeedbackCollector struct {
	mu          sync.RWMutex
	feedback    []Feedback
	performance map[string]*ModelPerformance // key: model
	assessor    *DifficultyAssessor
	smartRouter *SmartRouter
	maxFeedback int
	persistFile string
}

// OptimizationResult describes one optimization run outcome
type OptimizationResult struct {
	ModelsScanned      int `json:"models_scanned"`
	ModelsEligible     int `json:"models_eligible"`
	ModelsUpdated      int `json:"models_updated"`
	MinSamplesPerModel int `json:"min_samples_per_model"`
}

var feedbackLogger = logrus.WithField("component", "feedback_collector")

// NewFeedbackCollector creates a new feedback collector
func NewFeedbackCollector(assessor *DifficultyAssessor, smartRouter *SmartRouter) *FeedbackCollector {
	fc := &FeedbackCollector{
		feedback:    make([]Feedback, 0),
		performance: make(map[string]*ModelPerformance),
		assessor:    assessor,
		smartRouter: smartRouter,
		maxFeedback: 10000,
		persistFile: "data/feedback.json",
	}

	fc.loadFromFile()
	go fc.periodicOptimize()

	return fc
}

// RecordFeedback records user feedback
// 改动点: 记录用户反馈并更新模型评分
func (fc *FeedbackCollector) RecordFeedback(feedback Feedback) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	feedback.Timestamp = time.Now()
	fc.feedback = append(fc.feedback, feedback)

	// Trim if exceeds max
	if len(fc.feedback) > fc.maxFeedback {
		fc.feedback = fc.feedback[len(fc.feedback)-fc.maxFeedback:]
	}

	// Update performance metrics
	fc.updatePerformance(feedback)

	// Update difficulty assessor success rate
	if feedback.FeedbackType == FeedbackPositive {
		fc.assessor.UpdateSuccessRate(feedback.Model, feedback.TaskType, true)
	} else if feedback.FeedbackType == FeedbackNegative {
		fc.assessor.UpdateSuccessRate(feedback.Model, feedback.TaskType, false)
	}

	feedbackLogger.WithFields(logrus.Fields{
		"model":         feedback.Model,
		"feedback_type": feedback.FeedbackType,
		"rating":        feedback.Rating,
		"task_type":     feedback.TaskType,
	}).Info("Recorded feedback")
}

// RecordRequestResult records a request result
func (fc *FeedbackCollector) RecordRequestResult(model, provider string, taskType TaskType, difficulty DifficultyLevel, success bool, latencyMs int64, tokensUsed int) {
	feedbackType := FeedbackNeutral
	if success {
		feedbackType = FeedbackPositive
	} else {
		feedbackType = FeedbackNegative
	}

	fc.RecordFeedback(Feedback{
		Model:        model,
		Provider:     provider,
		TaskType:     taskType,
		Difficulty:   difficulty,
		FeedbackType: feedbackType,
		LatencyMs:    latencyMs,
		TokensUsed:   tokensUsed,
	})
}

// updatePerformance updates performance metrics
func (fc *FeedbackCollector) updatePerformance(feedback Feedback) {
	perf, ok := fc.performance[feedback.Model]
	if !ok {
		perf = &ModelPerformance{
			Model:         feedback.Model,
			Provider:      feedback.Provider,
			TaskTypeStats: make(map[string]*TaskTypeStat),
		}
		fc.performance[feedback.Model] = perf
	}

	perf.TotalRequests++
	perf.LastUpdated = time.Now()

	if feedback.FeedbackType == FeedbackPositive {
		perf.SuccessCount++
		perf.PositiveFeedback++
	} else if feedback.FeedbackType == FeedbackNegative {
		perf.FailureCount++
		perf.NegativeFeedback++
	}

	// Update average latency
	if perf.AvgLatencyMs == 0 {
		perf.AvgLatencyMs = feedback.LatencyMs
	} else {
		perf.AvgLatencyMs = (perf.AvgLatencyMs + feedback.LatencyMs) / 2
	}

	// Update average rating
	if feedback.Rating > 0 {
		if perf.AvgRating == 0 {
			perf.AvgRating = float64(feedback.Rating)
		} else {
			perf.AvgRating = (perf.AvgRating + float64(feedback.Rating)) / 2
		}
	}

	// Update task type stats
	taskKey := string(feedback.TaskType)
	taskStat, ok := perf.TaskTypeStats[taskKey]
	if !ok {
		taskStat = &TaskTypeStat{}
		perf.TaskTypeStats[taskKey] = taskStat
	}
	taskStat.TotalRequests++
	if feedback.FeedbackType == FeedbackPositive {
		taskStat.SuccessCount++
	}
	if taskStat.AvgLatencyMs == 0 {
		taskStat.AvgLatencyMs = feedback.LatencyMs
	} else {
		taskStat.AvgLatencyMs = (taskStat.AvgLatencyMs + feedback.LatencyMs) / 2
	}
	if taskStat.TotalRequests > 0 {
		taskStat.SuccessRate = float64(taskStat.SuccessCount) / float64(taskStat.TotalRequests)
	}
}

// GetPerformance returns performance metrics for a model
func (fc *FeedbackCollector) GetPerformance(model string) *ModelPerformance {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.performance[model]
}

// GetAllPerformance returns all performance metrics
func (fc *FeedbackCollector) GetAllPerformance() map[string]*ModelPerformance {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	result := make(map[string]*ModelPerformance)
	for k, v := range fc.performance {
		result[k] = v
	}
	return result
}

// GetTopModels returns top performing models
func (fc *FeedbackCollector) GetTopModels(taskType TaskType, limit int) []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	type scoredModel struct {
		model string
		score float64
	}

	var scored []scoredModel
	for model, perf := range fc.performance {
		score := fc.calculatePerformanceScore(perf, taskType)
		scored = append(scored, scoredModel{model: model, score: score})
	}

	// Sort by score
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top N
	result := make([]string, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].model)
	}
	return result
}

// TaskTypeDistribution represents the distribution of task types
type TaskTypeDistribution struct {
	TaskType string `json:"task_type"`
	Count    int64  `json:"count"`
	Percent  int    `json:"percent"`
}

// GetTaskTypeDistribution returns the distribution of task types from feedback
func (fc *FeedbackCollector) GetTaskTypeDistribution() []TaskTypeDistribution {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	counts := make(map[string]int64)
	var total int64

	// Prefer aggregated performance stats to avoid double-counting.
	// Feedback entries and performance are updated together by RecordFeedback.
	for _, perf := range fc.performance {
		for taskType, stat := range perf.TaskTypeStats {
			counts[taskType] += stat.TotalRequests
			total += stat.TotalRequests
		}
	}

	// Fallback for legacy data: if no performance stats, derive from feedback only.
	if total == 0 {
		for _, f := range fc.feedback {
			taskType := string(f.TaskType)
			if taskType == "" {
				taskType = "other"
			}
			counts[taskType]++
			total++
		}
	}

	result := make([]TaskTypeDistribution, 0, len(counts))
	for taskType, count := range counts {
		percent := 0
		if total > 0 {
			percent = int(float64(count) / float64(total) * 100)
		}
		result = append(result, TaskTypeDistribution{
			TaskType: taskType,
			Count:    count,
			Percent:  percent,
		})
	}

	// Sort by count descending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Count > result[i].Count {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// calculatePerformanceScore calculates a performance score for a model
func (fc *FeedbackCollector) calculatePerformanceScore(perf *ModelPerformance, taskType TaskType) float64 {
	score := 0.0

	// Base score from success rate
	if perf.TotalRequests > 0 {
		successRate := float64(perf.SuccessCount) / float64(perf.TotalRequests)
		score += successRate * 40
	}

	// Score from ratings
	score += perf.AvgRating * 10

	// Score from feedback ratio
	if perf.PositiveFeedback+perf.NegativeFeedback > 0 {
		positiveRatio := float64(perf.PositiveFeedback) / float64(perf.PositiveFeedback+perf.NegativeFeedback)
		score += positiveRatio * 20
	}

	// Task-specific bonus
	if taskStat, ok := perf.TaskTypeStats[string(taskType)]; ok {
		score += taskStat.SuccessRate * 20
	}

	// Latency penalty (lower is better)
	if perf.AvgLatencyMs > 0 {
		latencyScore := 100.0 / float64(perf.AvgLatencyMs/1000+1)
		score += latencyScore
	}

	return score
}

// OptimizeScores optimizes model scores based on feedback
// 改动点: 基于反馈自动优化模型评分
func (fc *FeedbackCollector) OptimizeScores() {
	fc.OptimizeScoresWithResult()
}

// OptimizeScoresWithResult optimizes scores and returns detailed summary
func (fc *FeedbackCollector) OptimizeScoresWithResult() OptimizationResult {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	scores := fc.smartRouter.GetAllModelScores()
	updated := 0
	eligible := 0
	minSamples := 10
	result := OptimizationResult{
		ModelsScanned:      len(fc.performance),
		MinSamplesPerModel: minSamples,
	}

	for model, perf := range fc.performance {
		if perf.TotalRequests < int64(minSamples) {
			continue // Not enough data
		}
		eligible++

		score, ok := scores[model]
		if !ok {
			continue
		}

		// Compute deterministic target scores.
		// This keeps optimization idempotent: repeated runs do not drift scores.
		successRate := float64(perf.SuccessCount) / float64(perf.TotalRequests)
		ratingRatio := perf.AvgRating / 5.0
		if ratingRatio < 0 {
			ratingRatio = 0
		}
		if ratingRatio > 1 {
			ratingRatio = 1
		}

		targetQuality := int(successRate*70 + ratingRatio*30)
		if targetQuality > 100 {
			targetQuality = 100
		}
		if targetQuality < 0 {
			targetQuality = 0
		}

		targetSpeed := 100 - int(perf.AvgLatencyMs/100)
		if targetSpeed < 0 {
			targetSpeed = 0
		}
		if targetSpeed > 100 {
			targetSpeed = 100
		}

		newQuality := targetQuality
		newSpeed := targetSpeed

		if newQuality != score.QualityScore || newSpeed != score.SpeedScore {
			oldQuality := score.QualityScore
			oldSpeed := score.SpeedScore
			score.QualityScore = newQuality
			score.SpeedScore = newSpeed
			fc.smartRouter.UpdateModelScore(model, score)
			updated++

			feedbackLogger.WithFields(logrus.Fields{
				"model":          model,
				"old_quality":    oldQuality,
				"old_speed":      oldSpeed,
				"new_quality":    newQuality,
				"new_speed":      newSpeed,
				"success_rate":   successRate,
				"avg_rating":     perf.AvgRating,
				"avg_latency_ms": perf.AvgLatencyMs,
			}).Info("Optimized model score based on feedback")
		}
	}

	if updated > 0 {
		fc.smartRouter.SaveToFile()
		feedbackLogger.WithField("models_updated", updated).Info("Feedback-based optimization completed")
	}

	result.ModelsEligible = eligible
	result.ModelsUpdated = updated
	return result
}

// periodicOptimize runs periodic optimization
func (fc *FeedbackCollector) periodicOptimize() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		fc.OptimizeScores()
		fc.SaveToFile()
	}
}

// SaveToFile saves feedback and performance data to file
func (fc *FeedbackCollector) SaveToFile() error {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	dir := filepath.Dir(fc.persistFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data := struct {
		Feedback    []Feedback                   `json:"feedback"`
		Performance map[string]*ModelPerformance `json:"performance"`
	}{
		Feedback:    fc.feedback,
		Performance: fc.performance,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fc.persistFile, jsonData, 0644)
}

// loadFromFile loads feedback and performance data from file
func (fc *FeedbackCollector) loadFromFile() {
	data, err := os.ReadFile(fc.persistFile)
	if err != nil {
		return
	}

	var saved struct {
		Feedback    []Feedback                   `json:"feedback"`
		Performance map[string]*ModelPerformance `json:"performance"`
	}

	if err := json.Unmarshal(data, &saved); err != nil {
		feedbackLogger.WithError(err).Warn("Failed to parse feedback file")
		return
	}

	fc.feedback = saved.Feedback
	fc.performance = saved.Performance

	feedbackLogger.WithFields(logrus.Fields{
		"feedback_count": len(fc.feedback),
		"models_tracked": len(fc.performance),
	}).Info("Loaded feedback data from file")
}

// GetFeedbackStats returns overall feedback statistics
func (fc *FeedbackCollector) GetFeedbackStats() map[string]interface{} {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	var totalPositive, totalNegative, totalNeutral int64
	var totalRating float64
	var ratingCount int64

	for _, f := range fc.feedback {
		switch f.FeedbackType {
		case FeedbackPositive:
			totalPositive++
		case FeedbackNegative:
			totalNegative++
		case FeedbackNeutral:
			totalNeutral++
		}
		if f.Rating > 0 {
			totalRating += float64(f.Rating)
			ratingCount++
		}
	}

	avgRating := 0.0
	if ratingCount > 0 {
		avgRating = totalRating / float64(ratingCount)
	}

	return map[string]interface{}{
		"total_feedback": len(fc.feedback),
		"positive_count": totalPositive,
		"negative_count": totalNegative,
		"neutral_count":  totalNeutral,
		"avg_rating":     avgRating,
		"models_tracked": len(fc.performance),
	}
}

// GetRecentFeedback returns recent feedback entries
func (fc *FeedbackCollector) GetRecentFeedback(limit int) []Feedback {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if len(fc.feedback) <= limit {
		result := make([]Feedback, len(fc.feedback))
		copy(result, fc.feedback)
		return result
	}

	result := make([]Feedback, limit)
	start := len(fc.feedback) - limit
	copy(result, fc.feedback[start:])
	return result
}

// ClearOldFeedback clears feedback older than specified duration
func (fc *FeedbackCollector) ClearOldFeedback(olderThan time.Duration) int {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	newFeedback := make([]Feedback, 0)
	cleared := 0

	for _, f := range fc.feedback {
		if f.Timestamp.After(cutoff) {
			newFeedback = append(newFeedback, f)
		} else {
			cleared++
		}
	}

	fc.feedback = newFeedback
	return cleared
}
