//nolint:godot,exhaustive,gocritic,goconst
package limiter

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	logpkg "ai-gateway/pkg/logger"
)

// AccountManager manages multiple AI provider accounts with automatic switching
type AccountManager struct {
	accounts       map[string]*AccountConfig
	statuses       map[string]*AccountStatus
	activeAccounts map[string]string // provider -> active account ID
	switchHistory  []SwitchEvent
	switchSaver    func([]SwitchEvent)
	limits         map[string]map[LimitType]AccountLimiter // accountID -> limitType -> limiter
	mu             sync.RWMutex
	store          *RedisStore
	logger         *logrus.Logger
	alertChan      chan Alert
	switchTimeout  time.Duration // Max time for switch operation
	scheduler      *AccountScheduler
	schedulerOnce  sync.Once
}

// AccountLimiter interface for account-specific limiters
type AccountLimiter interface {
	Allow(ctx context.Context, key string, cost int64) (bool, error)
	GetUsage(ctx context.Context, key string) (*Usage, error)
}

// NewAccountManager creates a new account manager
func NewAccountManager(store *RedisStore, logger *logrus.Logger) *AccountManager {
	if logger == nil {
		logger = logpkg.WithField("component", "limiter").Logger
	}
	return &AccountManager{
		accounts:       make(map[string]*AccountConfig),
		statuses:       make(map[string]*AccountStatus),
		activeAccounts: make(map[string]string),
		switchHistory:  make([]SwitchEvent, 0),
		limits:         make(map[string]map[LimitType]AccountLimiter),
		store:          store,
		logger:         logger,
		alertChan:      make(chan Alert, 100),
		switchTimeout:  3 * time.Second,
	}
}

// AddAccount adds a new account
func (m *AccountManager) AddAccount(config *AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("account ID is required")
	}

	m.accounts[config.ID] = config
	m.statuses[config.ID] = &AccountStatus{
		Account:      config,
		IsActive:     false,
		CurrentUsage: make(map[LimitType]*Usage),
	}

	// Initialize limiters for this account
	m.limits[config.ID] = make(map[LimitType]AccountLimiter)
	for _, limitConfig := range config.Limits {
		switch limitConfig.Type {
		case LimitTypeToken:
			m.limits[config.ID][limitConfig.Type] = NewTokenLimiter(
				m.store, limitConfig.Period, limitConfig.Limit, limitConfig.Warning,
			)
		case LimitTypeRPM:
			m.limits[config.ID][limitConfig.Type] = NewRPMLimiter(
				m.store, int(limitConfig.Limit), int(limitConfig.Limit)*2, limitConfig.Warning,
			)
		}
	}

	// Set as active if this is the first account for this provider type
	providerType := config.ProviderType
	if providerType == "" {
		providerType = config.Provider
	}
	if _, ok := m.activeAccounts[providerType]; !ok && config.Enabled {
		m.activeAccounts[providerType] = config.ID
		m.statuses[config.ID].IsActive = true
	}

	m.logger.WithFields(logrus.Fields{
		"account_id": config.ID,
		"provider":   config.Provider,
		"priority":   config.Priority,
	}).Info("Account added")

	return nil
}

// GetAccount returns an account by ID
func (m *AccountManager) GetAccount(accountID string) (*AccountConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	account, ok := m.accounts[accountID]
	if !ok {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	// Return a copy to avoid race conditions
	config := *account
	return &config, nil
}

// RemoveAccount removes an account
func (m *AccountManager) RemoveAccount(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	account, ok := m.accounts[accountID]
	if !ok {
		return fmt.Errorf("account not found: %s", accountID)
	}

	// Get provider type for this account
	providerType := account.ProviderType
	if providerType == "" {
		providerType = account.Provider
	}

	// If this is the active account, switch to another
	if m.activeAccounts[providerType] == accountID {
		m.switchToNextAccount(providerType, "account removed")
	}

	delete(m.accounts, accountID)
	delete(m.statuses, accountID)
	delete(m.limits, accountID)

	m.logger.WithField("account_id", accountID).Info("Account removed")
	return nil
}

// GetActiveAccount returns the active account for a provider type
func (m *AccountManager) GetActiveAccount(providerType string) (*AccountConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accountID, ok := m.activeAccounts[providerType]
	if !ok {
		return nil, ErrNoAvailableAccount
	}

	account, ok := m.accounts[accountID]
	if !ok {
		return nil, ErrNoAvailableAccount
	}

	return account, nil
}

// GetAccountByProviderAndBaseURL returns an account matching both provider and base URL
func (m *AccountManager) GetAccountByProviderAndBaseURL(provider string, baseURL string) *AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, account := range m.accounts {
		providerType := account.ProviderType
		if providerType == "" {
			providerType = account.Provider
		}
		if providerType == provider && account.BaseURL == baseURL && account.Enabled {
			return account
		}
	}
	return nil
}

// GetAccountByBaseURLAndType returns an account matching base URL and provider type
// Also matches by provider name for OpenAI-compatible providers
func (m *AccountManager) GetAccountByBaseURLAndType(baseURL string, providerType string) *AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, account := range m.accounts {
		accProviderType := account.ProviderType
		if accProviderType == "" {
			accProviderType = account.Provider
		}
		// Match by provider type OR by original provider name (for OpenAI-compatible providers)
		if account.BaseURL == baseURL && account.Enabled {
			if accProviderType == providerType || account.Provider == providerType {
				return account
			}
		}
	}
	return nil
}

// GetAccountByProvider returns an enabled account by provider name (ignores base URL)
func (m *AccountManager) GetAccountByProvider(providerName string) *AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, account := range m.accounts {
		if account.Provider == providerName && account.Enabled {
			return account
		}
	}
	return nil
}

// CheckAndSwitch checks limits and switches accounts if needed
func (m *AccountManager) CheckAndSwitch(ctx context.Context, provider string, estimatedCost int64) (*AccountConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	accountID, ok := m.activeAccounts[provider]
	if !ok {
		// Try to find any available account for this provider
		if err := m.findAndSetActive(provider); err != nil {
			return nil, ErrNoAvailableAccount
		}
		accountID = m.activeAccounts[provider]
	}

	account := m.accounts[accountID]

	// Check if account is enabled
	if !account.Enabled {
		m.switchToNextAccount(provider, "account disabled")
		return m.CheckAndSwitch(ctx, provider, estimatedCost)
	}

	// Check limits
	shouldSwitch := false
	switchReason := ""

	for limitType, limiter := range m.limits[accountID] {
		usage, err := limiter.GetUsage(ctx, accountID)
		if err != nil {
			m.logger.WithError(err).Error("Failed to get usage")
			continue
		}

		m.statuses[accountID].CurrentUsage[limitType] = usage

		// Check if limit exceeded
		allowed, err := limiter.Allow(ctx, accountID, estimatedCost)
		if err != nil {
			m.logger.WithError(err).Error("Failed to check limit")
			continue
		}

		if !allowed {
			shouldSwitch = true
			switchReason = fmt.Sprintf("%s limit exceeded", limitType)

			// Send alert
			m.sendAlert(Alert{
				Type:        AlertExceeded,
				AccountID:   accountID,
				LimitType:   limitType,
				CurrentUsed: usage.Used,
				Limit:       usage.Limit,
				PercentUsed: usage.PercentUsed,
				Timestamp:   time.Now(),
				Message:     fmt.Sprintf("Account %s %s limit exceeded", accountID, limitType),
			})
			break
		}

		// Check warning threshold
		if usage.WarningLevel == "warning" {
			m.sendAlert(Alert{
				Type:        AlertWarning,
				AccountID:   accountID,
				LimitType:   limitType,
				CurrentUsed: usage.Used,
				Limit:       usage.Limit,
				PercentUsed: usage.PercentUsed,
				Timestamp:   time.Now(),
				Message:     fmt.Sprintf("Account %s %s usage at %.1f%%", accountID, limitType, usage.PercentUsed),
			})
		}
	}

	if shouldSwitch {
		m.switchToNextAccount(provider, switchReason)
		return m.CheckAndSwitch(ctx, provider, estimatedCost)
	}

	return account, nil
}

// switchToNextAccount switches to the next available account (must be called with lock held)
func (m *AccountManager) switchToNextAccount(providerType, reason string) {
	startTime := time.Now()
	oldAccountID := m.activeAccounts[providerType]

	// Get all accounts for this provider type sorted by priority
	var candidates []*AccountConfig
	for _, account := range m.accounts {
		accProviderType := account.ProviderType
		if accProviderType == "" {
			accProviderType = account.Provider
		}
		if accProviderType == providerType && account.Enabled && account.ID != oldAccountID {
			candidates = append(candidates, account)
		}
	}

	if len(candidates) == 0 {
		m.logger.WithFields(logrus.Fields{
			"provider": providerType,
			"reason":   reason,
		}).Warn("No available accounts to switch to")
		return
	}

	// Sort by priority (higher is better)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})

	// Switch to highest priority account
	newAccount := candidates[0]
	m.activeAccounts[providerType] = newAccount.ID
	m.statuses[newAccount.ID].IsActive = true
	m.statuses[newAccount.ID].LastSwitched = time.Now()
	m.statuses[newAccount.ID].SwitchReason = reason

	if oldAccountID != "" {
		if status, ok := m.statuses[oldAccountID]; ok {
			status.IsActive = false
		}
	}

	switchDuration := time.Since(startTime)
	switchEvent := SwitchEvent{
		FromAccount: oldAccountID,
		ToAccount:   newAccount.ID,
		Reason:      reason,
		Timestamp:   time.Now(),
		Duration:    switchDuration,
	}
	m.appendSwitchEventLocked(switchEvent)

	m.logger.WithFields(logrus.Fields{
		"from_account": oldAccountID,
		"to_account":   newAccount.ID,
		"provider":     providerType,
		"reason":       reason,
		"duration_ms":  switchDuration.Milliseconds(),
	}).Info("Account switched")
}

// findAndSetActive finds and sets an active account for a provider type (must be called with lock held)
func (m *AccountManager) findAndSetActive(providerType string) error {
	var candidates []*AccountConfig
	for _, account := range m.accounts {
		accProviderType := account.ProviderType
		if accProviderType == "" {
			accProviderType = account.Provider
		}
		if accProviderType == providerType && account.Enabled {
			candidates = append(candidates, account)
		}
	}

	if len(candidates) == 0 {
		return ErrNoAvailableAccount
	}

	// Sort by priority
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})

	m.activeAccounts[providerType] = candidates[0].ID
	m.statuses[candidates[0].ID].IsActive = true

	return nil
}

// ConsumeUsage consumes usage for an account
func (m *AccountManager) ConsumeUsage(ctx context.Context, accountID string, tokens int64) error {
	m.mu.RLock()
	limits, ok := m.limits[accountID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("account not found: %s", accountID)
	}

	for limitType, limiter := range limits {
		switch limitType {
		case LimitTypeToken:
			if tl, ok := limiter.(*TokenLimiter); ok {
				if err := tl.Consume(ctx, accountID, tokens); err != nil {
					m.logger.WithError(err).WithField("account_id", accountID).Error("Failed to consume token limit")
				}
			}
		case LimitTypeRPM:
			if rl, ok := limiter.(*RPMLimiter); ok {
				if err := rl.RecordRequest(ctx, accountID); err != nil {
					m.logger.WithError(err).WithField("account_id", accountID).Error("Failed to record RPM")
				}
			}
		}
	}

	return nil
}

// GetAccountStatus returns the status of an account
func (m *AccountManager) GetAccountStatus(accountID string) (*AccountStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, ok := m.statuses[accountID]
	if !ok {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	return status, nil
}

// GetAllAccounts returns all accounts
func (m *AccountManager) GetAllAccounts() []*AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accounts := make([]*AccountConfig, 0, len(m.accounts))
	for _, account := range m.accounts {
		accounts = append(accounts, account)
	}
	return accounts
}

// GetSwitchHistory returns the switch history
func (m *AccountManager) GetSwitchHistory(limit int) []SwitchEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.switchHistory) {
		limit = len(m.switchHistory)
	}

	// Return last N entries
	start := len(m.switchHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]SwitchEvent, limit)
	copy(result, m.switchHistory[start:])
	return result
}

// Alerts returns the alert channel
func (m *AccountManager) Alerts() <-chan Alert {
	return m.alertChan
}

// sendAlert sends an alert (non-blocking)
func (m *AccountManager) sendAlert(alert Alert) {
	select {
	case m.alertChan <- alert:
	default:
		m.logger.Warn("Alert channel full, dropping alert")
	}
}

// UpdateAccount updates an account configuration
func (m *AccountManager) UpdateAccount(config *AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.accounts[config.ID]; !ok {
		return fmt.Errorf("account not found: %s", config.ID)
	}

	m.accounts[config.ID] = config

	// Re-initialize limiters
	m.limits[config.ID] = make(map[LimitType]AccountLimiter)
	for _, limitConfig := range config.Limits {
		switch limitConfig.Type {
		case LimitTypeToken:
			m.limits[config.ID][limitConfig.Type] = NewTokenLimiter(
				m.store, limitConfig.Period, limitConfig.Limit, limitConfig.Warning,
			)
		case LimitTypeRPM:
			m.limits[config.ID][limitConfig.Type] = NewRPMLimiter(
				m.store, int(limitConfig.Limit), int(limitConfig.Limit)*2, limitConfig.Warning,
			)
		}
	}

	m.logger.WithField("account_id", config.ID).Info("Account updated")
	return nil
}

// ForceSwitch forces a switch to a specific account
func (m *AccountManager) ForceSwitch(providerType, accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	account, ok := m.accounts[accountID]
	if !ok {
		return fmt.Errorf("account not found: %s", accountID)
	}

	accProviderType := account.ProviderType
	if accProviderType == "" {
		accProviderType = account.Provider
	}
	if accProviderType != providerType {
		return fmt.Errorf("account %s does not belong to provider type %s", accountID, providerType)
	}

	if !account.Enabled {
		return fmt.Errorf("account %s is disabled", accountID)
	}

	oldAccountID := m.activeAccounts[providerType]
	m.activeAccounts[providerType] = accountID
	m.statuses[accountID].IsActive = true
	m.statuses[accountID].LastSwitched = time.Now()
	m.statuses[accountID].SwitchReason = "forced switch"

	if oldAccountID != "" && oldAccountID != accountID {
		if status, ok := m.statuses[oldAccountID]; ok {
			status.IsActive = false
		}

		// Record switch history
		switchEvent := SwitchEvent{
			FromAccount: oldAccountID,
			ToAccount:   accountID,
			Reason:      "forced switch",
			Timestamp:   time.Now(),
		}
		m.appendSwitchEventLocked(switchEvent)
	}

	m.logger.WithFields(logrus.Fields{
		"from_account": oldAccountID,
		"to_account":   accountID,
		"provider":     providerType,
	}).Info("Forced account switch")

	return nil
}

func (m *AccountManager) appendSwitchEventLocked(event SwitchEvent) {
	m.switchHistory = append(m.switchHistory, event)
	if len(m.switchHistory) > 100 {
		m.switchHistory = m.switchHistory[len(m.switchHistory)-100:]
	}

	if m.switchSaver != nil {
		snapshot := make([]SwitchEvent, len(m.switchHistory))
		copy(snapshot, m.switchHistory)
		go m.switchSaver(snapshot)
	}
}

func (m *AccountManager) SetSwitchHistory(history []SwitchEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	m.switchHistory = make([]SwitchEvent, len(history))
	copy(m.switchHistory, history)
}

func (m *AccountManager) SetSwitchHistorySaver(saver func([]SwitchEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.switchSaver = saver
}

// getScheduler returns the scheduler instance (lazy initialization)
func (m *AccountManager) getScheduler() *AccountScheduler {
	m.schedulerOnce.Do(func() {
		config := DefaultSchedulerConfig()
		m.scheduler = NewAccountScheduler(config, m.store, m.getAccountsByProviderType)
	})
	return m.scheduler
}

// getAccountsByProviderType returns all accounts for a provider type
func (m *AccountManager) getAccountsByProviderType(providerType string) []*AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*AccountConfig
	for _, acc := range m.accounts {
		accProviderType := acc.ProviderType
		if accProviderType == "" {
			accProviderType = acc.Provider
		}
		if accProviderType == providerType {
			result = append(result, acc)
		}
	}
	return result
}

// SelectAccount selects an account using the three-layer scheduling strategy
func (m *AccountManager) SelectAccount(ctx context.Context, req ScheduleRequest) (*AccountConfig, ScheduleDecision, func(), error) {
	scheduler := m.getScheduler()
	if scheduler == nil {
		// Fallback to legacy behavior
		account, err := m.GetActiveAccount(req.ProviderType)
		return account, ScheduleDecision{Layer: ScheduleLayerLoadBalance}, nil, err
	}
	return scheduler.Select(ctx, req)
}

// ReportScheduleResult reports a request result for runtime statistics
func (m *AccountManager) ReportScheduleResult(accountID string, success bool, ttftMs int64) {
	scheduler := m.getScheduler()
	if scheduler != nil {
		scheduler.ReportResult(accountID, success, ttftMs)
	}
}

// ReportAccountSwitch reports an account switch
func (m *AccountManager) ReportAccountSwitch(fromAccountID, toAccountID, reason string) {
	scheduler := m.getScheduler()
	if scheduler != nil {
		scheduler.ReportSwitch(fromAccountID, toAccountID, reason)
	}
}

// BindResponseToAccount binds a response ID to an account for sticky sessions
func (m *AccountManager) BindResponseToAccount(ctx context.Context, responseID, accountID string) error {
	scheduler := m.getScheduler()
	if scheduler == nil {
		return nil
	}
	return scheduler.BindResponse(ctx, responseID, accountID)
}

// GetSchedulerMetrics returns scheduler metrics
func (m *AccountManager) GetSchedulerMetrics() map[string]int64 {
	scheduler := m.getScheduler()
	if scheduler == nil {
		return nil
	}
	return scheduler.GetMetrics()
}

// GetAccountRuntimeStats returns runtime statistics for an account
func (m *AccountManager) GetAccountRuntimeStats(accountID string) *AccountRuntimeStats {
	scheduler := m.getScheduler()
	if scheduler == nil {
		return nil
	}
	return scheduler.GetRuntimeStats(accountID)
}

// GetAccountLoadInfo returns current load information for an account
func (m *AccountManager) GetAccountLoadInfo(ctx context.Context, accountID string) (*AccountLoadInfo, error) {
	m.mu.RLock()
	account, exists := m.accounts[accountID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	scheduler := m.getScheduler()
	if scheduler == nil {
		return &AccountLoadInfo{AccountID: accountID}, nil
	}

	return scheduler.concurrencyManager.GetLoadInfo(ctx, accountID, account.Concurrency)
}

// SetSchedulerConfig sets the scheduler configuration
func (m *AccountManager) SetSchedulerConfig(config SchedulerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Recreate scheduler with new config
	m.scheduler = NewAccountScheduler(config, m.store, m.getAccountsByProviderType)
	m.schedulerOnce = sync.Once{}
}

// IsSchedulerEnabled returns whether the scheduler is enabled
func (m *AccountManager) IsSchedulerEnabled() bool {
	scheduler := m.getScheduler()
	return scheduler != nil && scheduler.config.Enabled
}
