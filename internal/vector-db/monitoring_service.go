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
