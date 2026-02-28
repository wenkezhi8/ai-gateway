package limiter

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// AccountScheduler implements the three-layer account scheduling strategy
type AccountScheduler struct {
	config             SchedulerConfig
	stickyManager      *StickySessionManager
	concurrencyManager *ConcurrencyManager
	runtimeStats       *RuntimeStatsManager
	getAccountsFunc    func(providerType string) []*AccountConfig

	metrics schedulerMetrics
	mu      sync.RWMutex
}

type schedulerMetrics struct {
	selectTotal        atomic.Int64
	stickyResponseHit  atomic.Int64
	stickySessionHit   atomic.Int64
	loadBalanceSelect  atomic.Int64
	accountSwitchTotal atomic.Int64
	latencyMsTotal     atomic.Int64
}

// NewAccountScheduler creates a new account scheduler
func NewAccountScheduler(
	config SchedulerConfig,
	store *RedisStore,
	getAccountsFunc func(providerType string) []*AccountConfig,
) *AccountScheduler {
	s := &AccountScheduler{
		config:             config,
		stickyManager:      NewStickySessionManager(store, config),
		concurrencyManager: NewConcurrencyManager(store),
		runtimeStats:       NewRuntimeStatsManager(config.EWMAAlpha),
		getAccountsFunc:    getAccountsFunc,
	}

	// Start cleanup goroutine
	go s.cleanupLoop()

	return s
}

// cleanupLoop periodically cleans up expired entries
func (s *AccountScheduler) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		if s.stickyManager != nil {
			s.stickyManager.CleanupExpired()
		}
	}
}

// Select selects an account using the three-layer strategy
func (s *AccountScheduler) Select(ctx context.Context, req ScheduleRequest) (*AccountConfig, ScheduleDecision, func(), error) {
	start := time.Now()
	decision := ScheduleDecision{}

	defer func() {
		decision.Score = 0
		s.metrics.latencyMsTotal.Add(time.Since(start).Milliseconds())
		s.metrics.selectTotal.Add(1)
	}()

	if !s.config.Enabled {
		return nil, decision, nil, ErrSchedulerDisabled
	}

	providerType := req.ProviderType
	if providerType == "" {
		return nil, decision, nil, errors.New("provider type is required")
	}

	// Layer 1: Previous Response ID sticky
	if req.PreviousResponseID != "" {
		accountID, err := s.stickyManager.GetResponseAccount(ctx, req.PreviousResponseID)
		if err == nil && accountID != "" {
			account := s.findAccountByID(providerType, accountID)
			if account != nil && account.Enabled && !s.isAccountExcluded(account.ID, req.ExcludedIDs) {
				// Try to acquire slot
				result, err := s.concurrencyManager.TryAcquire(ctx, account.ID, account.Concurrency)
				if err == nil && result.Acquired {
					decision.Layer = ScheduleLayerPreviousResponse
					decision.SelectedAccountID = account.ID
					decision.StickyHit = true
					s.metrics.stickyResponseHit.Add(1)

					// Bind session if provided
					if req.SessionHash != "" {
						_ = s.stickyManager.BindSession(ctx, providerType, req.SessionHash, account.ID)
					}

					return account, decision, result.ReleaseFunc, nil
				}
			}
		}
	}

	// Layer 2: Session Hash sticky
	if req.SessionHash != "" {
		accountID, err := s.stickyManager.GetSessionAccount(ctx, providerType, req.SessionHash)
		if err == nil && accountID != "" {
			account := s.findAccountByID(providerType, accountID)
			stats := s.runtimeStats.GetStats(accountID)

			// Check if we should clear sticky
			if account != nil && account.Enabled && !s.isAccountExcluded(account.ID, req.ExcludedIDs) {
				if !ShouldClearSticky(account, stats, req.Model) {
					// Try to acquire slot
					result, err := s.concurrencyManager.TryAcquire(ctx, account.ID, account.Concurrency)
					if err == nil && result.Acquired {
						decision.Layer = ScheduleLayerSessionSticky
						decision.SelectedAccountID = account.ID
						decision.StickyHit = true
						s.metrics.stickySessionHit.Add(1)

						// Refresh TTL
						_ = s.stickyManager.RefreshSessionTTL(ctx, providerType, req.SessionHash)

						return account, decision, result.ReleaseFunc, nil
					}
				} else {
					// Clear sticky session
					_ = s.stickyManager.DeleteSession(ctx, providerType, req.SessionHash)
				}
			}
		}
	}

	// Layer 3: Load Balance
	account, releaseFunc, lbDecision, err := s.selectByLoadBalance(ctx, providerType, req)
	if err != nil {
		return nil, decision, nil, err
	}

	decision.Layer = ScheduleLayerLoadBalance
	decision.SelectedAccountID = account.ID
	decision.StickyHit = false
	decision.CandidateCount = lbDecision.CandidateCount
	decision.Score = lbDecision.Score
	s.metrics.loadBalanceSelect.Add(1)

	// Bind session if provided
	if req.SessionHash != "" {
		_ = s.stickyManager.BindSession(ctx, providerType, req.SessionHash, account.ID)
	}

	return account, decision, releaseFunc, nil
}

// SelectResult represents the result of account selection
type SelectResult struct {
	Account     *AccountConfig
	Decision    ScheduleDecision
	ReleaseFunc func()
}

// selectByLoadBalance selects an account using load balancing
func (s *AccountScheduler) selectByLoadBalance(ctx context.Context, providerType string, req ScheduleRequest) (*AccountConfig, func(), *ScheduleDecision, error) {
	// Get all available accounts
	accounts := s.getAvailableAccounts(providerType, req.ExcludedIDs)
	if len(accounts) == 0 {
		return nil, nil, nil, ErrNoAvailableAccount
	}

	// Get load info for all accounts
	loadMap := s.concurrencyManager.GetBatchLoadInfo(ctx, accounts)

	// Calculate scores and sort
	type scoredAccount struct {
		account  *AccountConfig
		score    float64
		loadInfo *AccountLoadInfo
	}

	scored := make([]scoredAccount, 0, len(accounts))
	for _, acc := range accounts {
		loadInfo := loadMap[acc.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: acc.ID}
		}

		stats := s.runtimeStats.GetStats(acc.ID)
		score := s.calculateScore(acc, stats, loadInfo)

		scored = append(scored, scoredAccount{
			account:  acc,
			score:    score,
			loadInfo: loadInfo,
		})
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	decision := &ScheduleDecision{
		CandidateCount: len(scored),
	}

	// Try to acquire slot from highest scored account
	for _, sa := range scored {
		result, err := s.concurrencyManager.TryAcquire(ctx, sa.account.ID, sa.account.Concurrency)
		if err == nil && result.Acquired {
			decision.SelectedAccountID = sa.account.ID
			decision.Score = sa.score
			return sa.account, result.ReleaseFunc, decision, nil
		}
	}

	// All accounts at capacity, return highest scored anyway (caller can wait)
	decision.SelectedAccountID = scored[0].account.ID
	decision.Score = scored[0].score
	return scored[0].account, nil, decision, nil
}

// calculateScore calculates the score for an account
func (s *AccountScheduler) calculateScore(account *AccountConfig, stats *AccountRuntimeStats, loadInfo *AccountLoadInfo) float64 {
	weights := s.config.ScoreWeights

	// Normalize priority (assume 0-100 range)
	priorityFactor := float64(account.Priority) / 100.0
	if priorityFactor > 1 {
		priorityFactor = 1
	}

	// Load factor (lower is better)
	loadFactor := 1.0 - (loadInfo.LoadRate / 100.0)
	if loadFactor < 0 {
		loadFactor = 0
	}

	// Error rate factor (lower is better)
	errorFactor := 1.0
	if stats != nil {
		errorFactor = 1.0 - stats.ErrorRateEWMA
	}

	// TTFT factor (lower is better)
	ttftFactor := 1.0
	if stats != nil && stats.TTFTMsEWMA > 0 {
		ttftFactor = 1.0 / (1.0 + stats.TTFTMsEWMA/1000.0)
	}

	score := weights.Priority*priorityFactor +
		weights.Load*loadFactor +
		weights.ErrorRate*errorFactor +
		weights.TTFT*ttftFactor

	return score
}

// getAvailableAccounts gets all available accounts for a provider type
func (s *AccountScheduler) getAvailableAccounts(providerType string, excludedIDs map[string]struct{}) []*AccountConfig {
	var result []*AccountConfig

	if s.getAccountsFunc == nil {
		return result
	}

	allAccounts := s.getAccountsFunc(providerType)
	for _, acc := range allAccounts {
		if !acc.Enabled {
			continue
		}
		if s.isAccountExcluded(acc.ID, excludedIDs) {
			continue
		}
		result = append(result, acc)
	}

	return result
}

// findAccountByID finds an account by ID
func (s *AccountScheduler) findAccountByID(providerType, accountID string) *AccountConfig {
	if s.getAccountsFunc == nil {
		return nil
	}

	accounts := s.getAccountsFunc(providerType)
	for _, acc := range accounts {
		if acc.ID == accountID {
			return acc
		}
	}

	return nil
}

// isAccountExcluded checks if an account is in the excluded list
func (s *AccountScheduler) isAccountExcluded(accountID string, excludedIDs map[string]struct{}) bool {
	if excludedIDs == nil {
		return false
	}
	_, excluded := excludedIDs[accountID]
	return excluded
}

// ReportResult reports a request result
func (s *AccountScheduler) ReportResult(accountID string, success bool, ttftMs int64) {
	s.runtimeStats.ReportResult(accountID, success, ttftMs)
}

// ReportSwitch reports an account switch
func (s *AccountScheduler) ReportSwitch(fromAccountID, toAccountID, reason string) {
	s.metrics.accountSwitchTotal.Add(1)
}

// BindResponse binds a response ID to an account
func (s *AccountScheduler) BindResponse(ctx context.Context, responseID, accountID string) error {
	return s.stickyManager.BindResponse(ctx, responseID, accountID)
}

// GetMetrics returns current scheduler metrics
func (s *AccountScheduler) GetMetrics() map[string]int64 {
	return map[string]int64{
		"select_total":         s.metrics.selectTotal.Load(),
		"sticky_response_hit":  s.metrics.stickyResponseHit.Load(),
		"sticky_session_hit":   s.metrics.stickySessionHit.Load(),
		"load_balance_select":  s.metrics.loadBalanceSelect.Load(),
		"account_switch_total": s.metrics.accountSwitchTotal.Load(),
		"latency_ms_total":     s.metrics.latencyMsTotal.Load(),
	}
}

// GetRuntimeStats returns runtime stats for an account
func (s *AccountScheduler) GetRuntimeStats(accountID string) *AccountRuntimeStats {
	return s.runtimeStats.GetStats(accountID)
}

// GetConcurrencyManager returns the concurrency manager
func (s *AccountScheduler) GetConcurrencyManager() *ConcurrencyManager {
	return s.concurrencyManager
}

// GetStickyManager returns the sticky session manager
func (s *AccountScheduler) GetStickyManager() *StickySessionManager {
	return s.stickyManager
}

// Normalize provider type
func normalizeProviderType(providerType string) string {
	return strings.ToLower(strings.TrimSpace(providerType))
}
