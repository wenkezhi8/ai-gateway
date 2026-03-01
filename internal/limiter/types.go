//nolint:godot
package limiter

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrQuotaExceeded         = errors.New("quota exceeded")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrNoAvailableAccount    = errors.New("no available account")
	ErrInvalidPeriod         = errors.New("invalid period")
	ErrAccountUnavailable    = errors.New("account unavailable")
	ErrConcurrencyExceeded   = errors.New("concurrency limit exceeded")
	ErrSchedulerDisabled     = errors.New("scheduler disabled")
	ErrStickySessionNotFound = errors.New("sticky session not found")
)

// Period defines the time period for rate limiting
type Period string

const (
	PeriodMinute Period = "minute"
	PeriodHour   Period = "hour"
	PeriodHour5  Period = "5hour"
	PeriodDay    Period = "day"
	PeriodWeek   Period = "week"
	PeriodMonth  Period = "month"
)

// PeriodDuration returns the duration for a period
func PeriodDuration(p Period) (time.Duration, error) {
	switch p {
	case PeriodMinute:
		return time.Minute, nil
	case PeriodHour:
		return time.Hour, nil
	case PeriodHour5:
		return 5 * time.Hour, nil
	case PeriodDay:
		return 24 * time.Hour, nil
	case PeriodWeek:
		return 7 * 24 * time.Hour, nil
	case PeriodMonth:
		return 30 * 24 * time.Hour, nil
	default:
		return 0, ErrInvalidPeriod
	}
}

// LimitType defines the type of limit
type LimitType string

const (
	LimitTypeToken      LimitType = "token"      // Token usage limit
	LimitTypeRPM        LimitType = "rpm"        // Requests per minute
	LimitTypeConcurrent LimitType = "concurrent" // Concurrent requests
	LimitTypeRequest    LimitType = "request"    // Request count limit (for Coding Plan)
	LimitTypeHour5      LimitType = "hour5"      // 5-hour limit (for Coding Plan)
	LimitTypeWeek       LimitType = "week"       // Weekly limit (for Coding Plan)
	LimitTypeMonth      LimitType = "month"      // Monthly limit (for Coding Plan)
)

// Usage represents current usage information
type Usage struct {
	Key          string    `json:"key"`
	Used         int64     `json:"used"`
	Limit        int64     `json:"limit"`
	Remaining    int64     `json:"remaining"`
	ResetAt      time.Time `json:"reset_at"`
	Period       Period    `json:"period"`
	PercentUsed  float64   `json:"percent_used"`
	WarningLevel string    `json:"warning_level,omitempty"`
}

// AccountConfig represents an account configuration with limits
type AccountConfig struct {
	ID                string                     `json:"id"`
	Name              string                     `json:"name"`
	Provider          string                     `json:"provider"`
	ProviderType      string                     `json:"provider_type,omitempty"` // Backend provider type (e.g., "openai" for deepseek)
	APIKey            string                     `json:"api_key"`
	BaseURL           string                     `json:"base_url"`
	Enabled           bool                       `json:"enabled"`
	Priority          int                        `json:"priority"`
	Concurrency       int                        `json:"concurrency,omitempty"`   // Maximum concurrent requests (0 = unlimited)
	HealthStatus      string                     `json:"health_status,omitempty"` // healthy, degraded, unhealthy
	Limits            map[LimitType]*LimitConfig `json:"limits"`
	CodingPlanEnabled bool                       `json:"coding_plan_enabled,omitempty"` // AI 编程订阅开关
}

// LimitConfig represents a single limit configuration
type LimitConfig struct {
	Type    LimitType `json:"type"`
	Period  Period    `json:"period"`
	Limit   int64     `json:"limit"`
	Warning float64   `json:"warning"` // Warning threshold (0.9 = 90%)
}

// AccountStatus represents the current status of an account
type AccountStatus struct {
	Account      *AccountConfig       `json:"account"`
	IsActive     bool                 `json:"is_active"`
	CurrentUsage map[LimitType]*Usage `json:"current_usage"`
	LastSwitched time.Time            `json:"last_switched"`
	SwitchReason string               `json:"switch_reason,omitempty"`
}

// SwitchEvent represents an account switch event
type SwitchEvent struct {
	FromAccount string        `json:"from_account"`
	ToAccount   string        `json:"to_account"`
	Reason      string        `json:"reason"`
	Timestamp   time.Time     `json:"timestamp"`
	Duration    time.Duration `json:"duration"` // Time taken to switch
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertWarning  AlertType = "warning"
	AlertCritical AlertType = "critical"
	AlertExceeded AlertType = "exceeded"
)

// Alert represents a usage alert
type Alert struct {
	Type        AlertType `json:"type"`
	AccountID   string    `json:"account_id"`
	LimitType   LimitType `json:"limit_type"`
	CurrentUsed int64     `json:"current_used"`
	Limit       int64     `json:"limit"`
	PercentUsed float64   `json:"percent_used"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
}

// ScheduleLayer defines the layer at which account was selected
type ScheduleLayer string

const (
	ScheduleLayerPreviousResponse ScheduleLayer = "previous_response"
	ScheduleLayerSessionSticky    ScheduleLayer = "session_sticky"
	ScheduleLayerLoadBalance      ScheduleLayer = "load_balance"
)

// HealthStatus constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
)

// ScheduleRequest represents a request for account scheduling
type ScheduleRequest struct {
	ProviderType       string              `json:"provider_type"`
	Model              string              `json:"model"`
	SessionHash        string              `json:"session_hash,omitempty"`
	PreviousResponseID string              `json:"previous_response_id,omitempty"`
	ExcludedIDs        map[string]struct{} `json:"excluded_ids,omitempty"`
}

// ScheduleDecision represents the decision made by the scheduler
type ScheduleDecision struct {
	Layer             ScheduleLayer `json:"layer"`
	SelectedAccountID string        `json:"selected_account_id"`
	StickyHit         bool          `json:"sticky_hit"`
	Score             float64       `json:"score"`
	CandidateCount    int           `json:"candidate_count"`
	LoadSkew          float64       `json:"load_skew"`
}

// AccountRuntimeStats represents runtime statistics for an account
type AccountRuntimeStats struct {
	AccountID      string    `json:"account_id"`
	ErrorRateEWMA  float64   `json:"error_rate_ewma"` // Exponential weighted moving average of error rate
	TTFTMsEWMA     float64   `json:"ttft_ms_ewma"`    // Exponential weighted moving average of TTFT in ms
	SuccessCount   int64     `json:"success_count"`
	ErrorCount     int64     `json:"error_count"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// AccountLoadInfo represents current load information for an account
type AccountLoadInfo struct {
	AccountID         string  `json:"account_id"`
	CurrentConcurrent int     `json:"current_concurrent"`
	MaxConcurrency    int     `json:"max_concurrency"`
	LoadRate          float64 `json:"load_rate"` // 0-100
	WaitingCount      int     `json:"waiting_count"`
}

// SchedulerScoreWeights defines weights for scoring accounts
type SchedulerScoreWeights struct {
	Priority  float64 `json:"priority"`
	Load      float64 `json:"load"`
	ErrorRate float64 `json:"error_rate"`
	TTFT      float64 `json:"ttft"`
}

// DefaultSchedulerScoreWeights returns default scoring weights
func DefaultSchedulerScoreWeights() SchedulerScoreWeights {
	return SchedulerScoreWeights{
		Priority:  1.0,
		Load:      1.0,
		ErrorRate: 0.8,
		TTFT:      0.5,
	}
}

// SchedulerConfig represents configuration for the scheduler
type SchedulerConfig struct {
	Enabled                 bool                  `json:"enabled"`
	StickySessionTTL        time.Duration         `json:"sticky_session_ttl"`
	ResponseBindTTL         time.Duration         `json:"response_bind_ttl"`
	ScoreWeights            SchedulerScoreWeights `json:"score_weights"`
	EWMAAlpha               float64               `json:"ewma_alpha"` // Smoothing factor for EWMA
	MaxStickySessionWait    time.Duration         `json:"max_sticky_session_wait"`
	HealthCheckInterval     time.Duration         `json:"health_check_interval"`
	UnhealthyErrorThreshold float64               `json:"unhealthy_error_threshold"`
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Enabled:                 true,
		StickySessionTTL:        30 * time.Minute,
		ResponseBindTTL:         1 * time.Hour,
		ScoreWeights:            DefaultSchedulerScoreWeights(),
		EWMAAlpha:               0.2,
		MaxStickySessionWait:    10 * time.Second,
		HealthCheckInterval:     30 * time.Second,
		UnhealthyErrorThreshold: 0.5,
	}
}
