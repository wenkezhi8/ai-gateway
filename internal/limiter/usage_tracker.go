//nolint:godot,gocritic,revive
package limiter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	logpkg "ai-gateway/pkg/logger"
)

// UsageTracker tracks usage across all accounts with real-time statistics and alerts
type UsageTracker struct {
	store       *RedisStore
	manager     *AccountManager
	logger      *logrus.Logger
	alerts      chan Alert
	subscribers []chan Alert
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
}

// UsageReport represents a comprehensive usage report
type UsageReport struct {
	Timestamp      time.Time                      `json:"timestamp"`
	AccountReports map[string]*AccountUsageReport `json:"accounts"`
	Summary        *UsageSummary                  `json:"summary"`
}

// AccountUsageReport represents usage for a single account
type AccountUsageReport struct {
	AccountID   string               `json:"account_id"`
	Provider    string               `json:"provider"`
	IsActive    bool                 `json:"is_active"`
	UsageByType map[LimitType]*Usage `json:"usage_by_type"`
	Status      string               `json:"status"` // healthy, warning, exceeded
}

// UsageSummary represents aggregate usage statistics
type UsageSummary struct {
	TotalAccounts    int     `json:"total_accounts"`
	ActiveAccounts   int     `json:"active_accounts"`
	WarningAccounts  int     `json:"warning_accounts"`
	ExceededAccounts int     `json:"exceeded_accounts"`
	TotalTokensUsed  int64   `json:"total_tokens_used"`
	TotalRequests    int64   `json:"total_requests"`
	AvgUsagePercent  float64 `json:"avg_usage_percent"`
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker(store *RedisStore, manager *AccountManager, logger *logrus.Logger) *UsageTracker {
	if logger == nil {
		logger = logpkg.WithField("component", "limiter").Logger
	}
	return &UsageTracker{
		store:       store,
		manager:     manager,
		logger:      logger,
		alerts:      make(chan Alert, 200),
		subscribers: make([]chan Alert, 0),
		stopChan:    make(chan struct{}),
	}
}

// Start starts the usage tracker background monitoring
func (t *UsageTracker) Start(ctx context.Context, interval time.Duration) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("usage tracker already running")
	}
	t.running = true
	t.mu.Unlock()

	// Start alert forwarder
	go t.forwardAlerts()

	// Start monitoring loop
	go t.monitorLoop(ctx, interval)

	// Subscribe to account manager alerts
	go func() {
		for alert := range t.manager.Alerts() {
			t.alerts <- alert
		}
	}()

	t.logger.Info("Usage tracker started")
	return nil
}

// Stop stops the usage tracker
func (t *UsageTracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return
	}

	close(t.stopChan)
	t.running = false
	t.logger.Info("Usage tracker stopped")
}

// monitorLoop periodically checks all accounts
func (t *UsageTracker) monitorLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.checkAllAccounts(ctx)
		}
	}
}

// checkAllAccounts checks usage for all accounts
func (t *UsageTracker) checkAllAccounts(ctx context.Context) {
	accounts := t.manager.GetAllAccounts()

	for _, account := range accounts {
		if !account.Enabled {
			continue
		}

		status, err := t.manager.GetAccountStatus(account.ID)
		if err != nil {
			t.logger.WithError(err).WithField("account_id", account.ID).Error("Failed to get account status")
			continue
		}

		// Check each limit type
		for limitType := range t.manager.limits[account.ID] {
			usage, err := t.getAccountUsage(ctx, account.ID, limitType)
			if err != nil {
				t.logger.WithError(err).WithFields(logrus.Fields{
					"account_id": account.ID,
					"limit_type": limitType,
				}).Error("Failed to get usage")
				continue
			}

			status.CurrentUsage[limitType] = usage

			// Check thresholds
			if usage.PercentUsed >= 100 {
				t.sendAlert(Alert{
					Type:        AlertExceeded,
					AccountID:   account.ID,
					LimitType:   limitType,
					CurrentUsed: usage.Used,
					Limit:       usage.Limit,
					PercentUsed: usage.PercentUsed,
					Timestamp:   time.Now(),
					Message:     fmt.Sprintf("Account %s %s limit exceeded (%.1f%%)", account.ID, limitType, usage.PercentUsed),
				})
			} else if usage.PercentUsed >= 90 {
				t.sendAlert(Alert{
					Type:        AlertCritical,
					AccountID:   account.ID,
					LimitType:   limitType,
					CurrentUsed: usage.Used,
					Limit:       usage.Limit,
					PercentUsed: usage.PercentUsed,
					Timestamp:   time.Now(),
					Message:     fmt.Sprintf("Account %s %s at critical level (%.1f%%)", account.ID, limitType, usage.PercentUsed),
				})
			} else if usage.PercentUsed >= 80 {
				t.sendAlert(Alert{
					Type:        AlertWarning,
					AccountID:   account.ID,
					LimitType:   limitType,
					CurrentUsed: usage.Used,
					Limit:       usage.Limit,
					PercentUsed: usage.PercentUsed,
					Timestamp:   time.Now(),
					Message:     fmt.Sprintf("Account %s %s approaching limit (%.1f%%)", account.ID, limitType, usage.PercentUsed),
				})
			}
		}
	}
}

// getAccountUsage gets usage for a specific account and limit type
func (t *UsageTracker) getAccountUsage(ctx context.Context, accountID string, limitType LimitType) (*Usage, error) {
	t.manager.mu.RLock()
	limits, ok := t.manager.limits[accountID]
	t.manager.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	limiter, ok := limits[limitType]
	if !ok {
		return nil, fmt.Errorf("limit type not configured: %s", limitType)
	}

	return limiter.GetUsage(ctx, accountID)
}

// GetReport generates a comprehensive usage report
func (t *UsageTracker) GetReport(ctx context.Context) (*UsageReport, error) {
	accounts := t.manager.GetAllAccounts()
	report := &UsageReport{
		Timestamp:      time.Now(),
		AccountReports: make(map[string]*AccountUsageReport),
		Summary:        &UsageSummary{},
	}

	var totalPercent float64
	var accountCount int

	for _, account := range accounts {
		accountReport := &AccountUsageReport{
			AccountID:   account.ID,
			Provider:    account.Provider,
			UsageByType: make(map[LimitType]*Usage),
		}

		status, err := t.manager.GetAccountStatus(account.ID)
		if err == nil {
			accountReport.IsActive = status.IsActive
			accountReport.UsageByType = status.CurrentUsage

			// Determine status
			for _, usage := range status.CurrentUsage {
				if usage.PercentUsed >= 100 {
					accountReport.Status = "exceeded"
					report.Summary.ExceededAccounts++
				} else if usage.PercentUsed >= 90 {
					if accountReport.Status != "exceeded" {
						accountReport.Status = "warning"
					}
					report.Summary.WarningAccounts++
				} else if accountReport.Status == "" {
					accountReport.Status = "healthy"
				}

				totalPercent += usage.PercentUsed
				accountCount++

				// Accumulate totals
				if usage, ok := status.CurrentUsage[LimitTypeToken]; ok {
					report.Summary.TotalTokensUsed += usage.Used
				}
			}
		}

		if accountReport.Status == "" {
			accountReport.Status = "healthy"
		}

		if accountReport.IsActive {
			report.Summary.ActiveAccounts++
		}

		report.AccountReports[account.ID] = accountReport
		report.Summary.TotalAccounts++
	}

	if accountCount > 0 {
		report.Summary.AvgUsagePercent = totalPercent / float64(accountCount)
	}

	return report, nil
}

// Subscribe subscribes to usage alerts
func (t *UsageTracker) Subscribe() chan Alert {
	t.mu.Lock()
	defer t.mu.Unlock()

	ch := make(chan Alert, 50)
	t.subscribers = append(t.subscribers, ch)
	return ch
}

// Unsubscribe unsubscribes from usage alerts
func (t *UsageTracker) Unsubscribe(ch chan Alert) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, subscriber := range t.subscribers {
		if subscriber == ch {
			t.subscribers = append(t.subscribers[:i], t.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// forwardAlerts forwards alerts to all subscribers
func (t *UsageTracker) forwardAlerts() {
	for alert := range t.alerts {
		t.mu.RLock()
		subscribers := make([]chan Alert, len(t.subscribers))
		copy(subscribers, t.subscribers)
		t.mu.RUnlock()

		for _, ch := range subscribers {
			select {
			case ch <- alert:
			default:
				t.logger.Warn("Alert subscriber channel full, dropping alert")
			}
		}

		// Log the alert
		t.logger.WithFields(logrus.Fields{
			"type":       alert.Type,
			"account_id": alert.AccountID,
			"limit_type": alert.LimitType,
			"percent":    alert.PercentUsed,
			"message":    alert.Message,
		}).Info("Usage alert")
	}
}

// sendAlert sends an alert to the tracker
func (t *UsageTracker) sendAlert(alert Alert) {
	select {
	case t.alerts <- alert:
	default:
		t.logger.Warn("Alert channel full, dropping alert")
	}
}

// RecordUsage records usage for an account
func (t *UsageTracker) RecordUsage(ctx context.Context, accountID string, tokens int64) error {
	// Store in Redis for persistence
	key := fmt.Sprintf("usage_history:%s", accountID)
	now := time.Now().UnixNano()

	data := map[string]interface{}{
		"tokens":    tokens,
		"timestamp": now,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Add to sorted set with timestamp as score
	if err := t.store.ZAdd(ctx, key, float64(now), string(jsonData)); err != nil {
		return err
	}

	// Set expiration (keep 30 days of history)
	return t.store.Expire(ctx, key, 30*24*time.Hour)
}

// GetUsageHistory gets usage history for an account
func (t *UsageTracker) GetUsageHistory(ctx context.Context, accountID string, since time.Time) ([]map[string]interface{}, error) {
	key := fmt.Sprintf("usage_history:%s", accountID)

	entries, err := t.store.ZRangeByScore(ctx, key,
		fmt.Sprintf("%d", since.UnixNano()),
		"+inf",
		nil,
	)
	if err != nil {
		return nil, err
	}

	history := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(entry), &data); err != nil {
			continue
		}
		history = append(history, data)
	}

	return history, nil
}

// PredictExhaustion predicts when an account will exhaust its limit
func (t *UsageTracker) PredictExhaustion(ctx context.Context, accountID string, limitType LimitType) (*time.Time, error) {
	usage, err := t.getAccountUsage(ctx, accountID, limitType)
	if err != nil {
		return nil, err
	}

	if usage.Used >= usage.Limit {
		return nil, nil // Already exhausted
	}

	// Get recent usage history to calculate rate
	history, err := t.GetUsageHistory(ctx, accountID, time.Now().Add(-24*time.Hour))
	if err != nil || len(history) < 2 {
		return nil, fmt.Errorf("insufficient history for prediction")
	}

	// Calculate average rate
	var totalTokens int64
	for _, h := range history {
		if tokens, ok := h["tokens"].(float64); ok {
			totalTokens += int64(tokens)
		}
	}

	hours := 24.0
	ratePerHour := float64(totalTokens) / hours
	remaining := float64(usage.Limit - usage.Used)

	if ratePerHour <= 0 {
		return nil, nil
	}

	hoursUntilExhaustion := remaining / ratePerHour
	exhaustionTime := time.Now().Add(time.Duration(hoursUntilExhaustion) * time.Hour)

	return &exhaustionTime, nil
}
