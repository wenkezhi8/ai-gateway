package vectordb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) CreateAlertRule(ctx context.Context, req *CreateAlertRuleRequest) (*AlertRule, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Metric) == "" {
		return nil, fmt.Errorf("metric is required")
	}
	if strings.TrimSpace(req.Operator) == "" {
		return nil, fmt.Errorf("operator is required")
	}
	if req.Threshold <= 0 {
		return nil, fmt.Errorf("threshold must be positive")
	}
	if strings.TrimSpace(req.Duration) == "" {
		return nil, fmt.Errorf("duration is required")
	}

	now := time.Now().UTC()
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := &AlertRule{
		Name:      strings.TrimSpace(req.Name),
		Metric:    strings.TrimSpace(req.Metric),
		Operator:  strings.TrimSpace(req.Operator),
		Threshold: req.Threshold,
		Duration:  strings.TrimSpace(req.Duration),
		Channels:  copyTags(req.Channels),
		Enabled:   enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateAlertRule(ctx, rule); err != nil {
		return nil, err
	}
	result := *rule
	return &result, nil
}

func (s *Service) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	rules, err := s.repo.ListAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range rules {
		rules[idx].Channels = copyTags(rules[idx].Channels)
	}
	return rules, nil
}

func (s *Service) UpdateAlertRule(ctx context.Context, id int64, req *UpdateAlertRuleRequest) error {
	if s.repo == nil {
		return fmt.Errorf("repository is required")
	}
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	if req == nil {
		return fmt.Errorf("request is required")
	}
	return s.repo.UpdateAlertRule(ctx, id, req)
}

func (s *Service) DeleteAlertRule(ctx context.Context, id int64) error {
	if s.repo == nil {
		return fmt.Errorf("repository is required")
	}
	if id <= 0 {
		return fmt.Errorf("id must be positive")
	}
	return s.repo.DeleteAlertRule(ctx, id)
}

func (s *Service) GetVectorMetricsSummary(ctx context.Context) (*MetricsSummary, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	collections, err := s.repo.List(ctx, &ListCollectionsQuery{})
	if err != nil {
		return nil, err
	}
	jobs, err := s.repo.SummarizeImportJobs(ctx, &ListImportJobsQuery{})
	if err != nil {
		return nil, err
	}
	rules, err := s.repo.ListAlertRules(ctx)
	if err != nil {
		return nil, err
	}

	enabled := 0
	for idx := range rules {
		if rules[idx].Enabled {
			enabled++
		}
	}

	return &MetricsSummary{
		CollectionsTotal: len(collections),
		ImportJobs:       jobs,
		AlertRulesTotal:  len(rules),
		EnabledRules:     enabled,
	}, nil
}

func (s *Service) NotifyAlertChannels(ctx context.Context, req *NotifyAlertChannelsRequest) (*NotifyAlertChannelsResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	ruleName := strings.TrimSpace(req.RuleName)
	if ruleName == "" {
		return nil, fmt.Errorf("rule_name is required")
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}
	channels := normalizeAlertChannels(req.Channels)
	if len(channels) == 0 {
		return nil, fmt.Errorf("channels is required")
	}
	operator := strings.TrimSpace(req.Operator)
	if operator == "" {
		operator = "system"
	}

	sent := 0
	for _, channel := range channels {
		details := fmt.Sprintf("rule=%s channel=%s message=%s", ruleName, channel, message)
		audit := &AuditLog{
			UserID:       operator,
			Action:       "alert_notify",
			ResourceType: "alert_rule",
			ResourceID:   ruleName,
			Details:      details,
		}
		if err := s.repo.CreateAuditLog(ctx, audit); err != nil {
			return nil, err
		}
		sent++
	}

	return &NotifyAlertChannelsResponse{
		RuleName: ruleName,
		Channels: channels,
		Total:    len(channels),
		Sent:     sent,
		Failed:   len(channels) - sent,
	}, nil
}

func normalizeAlertChannels(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		channel := strings.ToLower(strings.TrimSpace(item))
		if channel == "" {
			continue
		}
		if channel != "webhook" && channel != "email" && channel != "console" {
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		normalized = append(normalized, channel)
	}
	return normalized
}
