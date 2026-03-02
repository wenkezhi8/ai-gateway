package vectordb

import (
	"context"
	"testing"
)

func TestMonitoringService_AlertRulesCRUD(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	created, err := svc.CreateAlertRule(context.Background(), &CreateAlertRuleRequest{
		Name:      "high-latency",
		Metric:    "search_p95_ms",
		Operator:  "gt",
		Threshold: 500,
		Duration:  "5m",
		Channels:  []string{"webhook"},
	})
	if err != nil {
		t.Fatalf("CreateAlertRule() error = %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("CreateAlertRule() id=%d, want >0", created.ID)
	}

	rules, err := svc.ListAlertRules(context.Background())
	if err != nil {
		t.Fatalf("ListAlertRules() error = %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("ListAlertRules() len=%d, want 1", len(rules))
	}

	name := "updated-latency"
	enabled := false
	updateErr := svc.UpdateAlertRule(context.Background(), created.ID, &UpdateAlertRuleRequest{Name: &name, Enabled: &enabled})
	if updateErr != nil {
		t.Fatalf("UpdateAlertRule() error = %v", updateErr)
	}

	rules, err = svc.ListAlertRules(context.Background())
	if err != nil {
		t.Fatalf("ListAlertRules() error = %v", err)
	}
	if rules[0].Name != name || rules[0].Enabled != enabled {
		t.Fatalf("ListAlertRules() rule=%+v", rules[0])
	}

	deleteErr := svc.DeleteAlertRule(context.Background(), created.ID)
	if deleteErr != nil {
		t.Fatalf("DeleteAlertRule() error = %v", deleteErr)
	}

	rules, err = svc.ListAlertRules(context.Background())
	if err != nil {
		t.Fatalf("ListAlertRules() error = %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("ListAlertRules() len=%d, want 0", len(rules))
	}
}

func TestMonitoringService_GetMetricsSummary_ShouldReturnAggregates(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {ID: "job_1", Status: ImportJobStatusPending},
			"job_2": {ID: "job_2", Status: ImportJobStatusFailed},
		},
		alertRules: map[int64]*AlertRule{
			1: {ID: 1, Name: "r1", Enabled: true},
			2: {ID: 2, Name: "r2", Enabled: false},
		},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	summary, err := svc.GetVectorMetricsSummary(context.Background())
	if err != nil {
		t.Fatalf("GetVectorMetricsSummary() error = %v", err)
	}
	if summary.CollectionsTotal != 0 {
		t.Fatalf("GetVectorMetricsSummary() collections=%d, want 0", summary.CollectionsTotal)
	}
	if summary.ImportJobs.Total != 2 || summary.ImportJobs.Failed != 1 {
		t.Fatalf("GetVectorMetricsSummary() import_jobs=%+v", summary.ImportJobs)
	}
	if summary.AlertRulesTotal != 2 || summary.EnabledRules != 1 {
		t.Fatalf("GetVectorMetricsSummary() summary=%+v", summary)
	}
}

func TestMonitoringService_NotifyAlertChannels_ShouldSupportMultiChannels(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	result, err := svc.NotifyAlertChannels(context.Background(), &NotifyAlertChannelsRequest{
		RuleName: "high-latency",
		Message:  "search_p95_ms exceeded",
		Channels: []string{"webhook", "email", "console"},
		Operator: "tester",
	})
	if err != nil {
		t.Fatalf("NotifyAlertChannels() error = %v", err)
	}
	if result.Total != 3 || result.Sent != 3 {
		t.Fatalf("NotifyAlertChannels() result=%+v", result)
	}
	if len(repo.auditLogs) != 3 {
		t.Fatalf("NotifyAlertChannels() audit logs=%d, want 3", len(repo.auditLogs))
	}
}
