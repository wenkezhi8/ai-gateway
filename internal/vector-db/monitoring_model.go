package vectordb

import "time"

type AlertRule struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Metric    string    `json:"metric"`
	Operator  string    `json:"operator"`
	Threshold float64   `json:"threshold"`
	Duration  string    `json:"duration"`
	Channels  []string  `json:"channels"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateAlertRuleRequest struct {
	Name      string   `json:"name" binding:"required"`
	Metric    string   `json:"metric" binding:"required"`
	Operator  string   `json:"operator" binding:"required"`
	Threshold float64  `json:"threshold" binding:"required"`
	Duration  string   `json:"duration" binding:"required"`
	Channels  []string `json:"channels"`
	Enabled   *bool    `json:"enabled"`
}

type UpdateAlertRuleRequest struct {
	Name      *string   `json:"name"`
	Metric    *string   `json:"metric"`
	Operator  *string   `json:"operator"`
	Threshold *float64  `json:"threshold"`
	Duration  *string   `json:"duration"`
	Channels  *[]string `json:"channels"`
	Enabled   *bool     `json:"enabled"`
}

type MetricsSummary struct {
	CollectionsTotal int               `json:"collections_total"`
	ImportJobs       *ImportJobSummary `json:"import_jobs"`
	AlertRulesTotal  int               `json:"alert_rules_total"`
	EnabledRules     int               `json:"enabled_rules"`
}

type NotifyAlertChannelsRequest struct {
	RuleName string   `json:"rule_name" binding:"required"`
	Message  string   `json:"message" binding:"required"`
	Channels []string `json:"channels" binding:"required"`
	Operator string   `json:"operator"`
}

type NotifyAlertChannelsResponse struct {
	RuleName string   `json:"rule_name"`
	Channels []string `json:"channels"`
	Total    int      `json:"total"`
	Sent     int      `json:"sent"`
	Failed   int      `json:"failed"`
}
