package vectordb

import (
	"context"
	"testing"
)

func TestMonitoringService_ValidationBranches_ShouldFailFast(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithDeps(nil, &mockBackend{})
	if _, err := svc.CreateAlertRule(context.Background(), &CreateAlertRuleRequest{}); err == nil {
		t.Fatal("CreateAlertRule() should fail when repo is nil")
	}
	if _, err := svc.ListAlertRules(context.Background()); err == nil {
		t.Fatal("ListAlertRules() should fail when repo is nil")
	}
	if err := svc.UpdateAlertRule(context.Background(), 1, &UpdateAlertRuleRequest{}); err == nil {
		t.Fatal("UpdateAlertRule() should fail when repo is nil")
	}
	if err := svc.DeleteAlertRule(context.Background(), 1); err == nil {
		t.Fatal("DeleteAlertRule() should fail when repo is nil")
	}
	if _, err := svc.GetVectorMetricsSummary(context.Background()); err == nil {
		t.Fatal("GetVectorMetricsSummary() should fail when repo is nil")
	}
	if _, err := svc.NotifyAlertChannels(context.Background(), &NotifyAlertChannelsRequest{}); err == nil {
		t.Fatal("NotifyAlertChannels() should fail when repo is nil")
	}

	svc = NewServiceWithDeps(&mockRepo{}, &mockBackend{})
	if _, err := svc.CreateAlertRule(context.Background(), nil); err == nil {
		t.Fatal("CreateAlertRule(nil) should fail")
	}
	if _, err := svc.CreateAlertRule(context.Background(), &CreateAlertRuleRequest{Name: " ", Metric: "m", Operator: "gt", Threshold: 1, Duration: "1m"}); err == nil {
		t.Fatal("CreateAlertRule(empty name) should fail")
	}
	if _, err := svc.CreateAlertRule(context.Background(), &CreateAlertRuleRequest{Name: "n", Metric: " ", Operator: "gt", Threshold: 1, Duration: "1m"}); err == nil {
		t.Fatal("CreateAlertRule(empty metric) should fail")
	}
	if _, err := svc.CreateAlertRule(context.Background(), &CreateAlertRuleRequest{Name: "n", Metric: "m", Operator: " ", Threshold: 1, Duration: "1m"}); err == nil {
		t.Fatal("CreateAlertRule(empty operator) should fail")
	}
	if _, err := svc.CreateAlertRule(context.Background(), &CreateAlertRuleRequest{Name: "n", Metric: "m", Operator: "gt", Threshold: 0, Duration: "1m"}); err == nil {
		t.Fatal("CreateAlertRule(non-positive threshold) should fail")
	}
	if _, err := svc.CreateAlertRule(context.Background(), &CreateAlertRuleRequest{Name: "n", Metric: "m", Operator: "gt", Threshold: 1, Duration: " "}); err == nil {
		t.Fatal("CreateAlertRule(empty duration) should fail")
	}

	if err := svc.UpdateAlertRule(context.Background(), 0, &UpdateAlertRuleRequest{}); err == nil {
		t.Fatal("UpdateAlertRule(non-positive id) should fail")
	}
	if err := svc.UpdateAlertRule(context.Background(), 1, nil); err == nil {
		t.Fatal("UpdateAlertRule(nil request) should fail")
	}
	if err := svc.DeleteAlertRule(context.Background(), 0); err == nil {
		t.Fatal("DeleteAlertRule(non-positive id) should fail")
	}

	if _, err := svc.NotifyAlertChannels(context.Background(), nil); err == nil {
		t.Fatal("NotifyAlertChannels(nil) should fail")
	}
	if _, err := svc.NotifyAlertChannels(context.Background(), &NotifyAlertChannelsRequest{RuleName: " ", Message: "m", Channels: []string{"console"}}); err == nil {
		t.Fatal("NotifyAlertChannels(empty rule_name) should fail")
	}
	if _, err := svc.NotifyAlertChannels(context.Background(), &NotifyAlertChannelsRequest{RuleName: "r", Message: " ", Channels: []string{"console"}}); err == nil {
		t.Fatal("NotifyAlertChannels(empty message) should fail")
	}
	if _, err := svc.NotifyAlertChannels(context.Background(), &NotifyAlertChannelsRequest{RuleName: "r", Message: "m", Channels: []string{"unknown"}}); err == nil {
		t.Fatal("NotifyAlertChannels(invalid channels) should fail")
	}
}

func TestBackupService_ValidationBranches_ShouldFailFast(t *testing.T) {
	t.Parallel()

	svc := NewServiceWithDeps(nil, &mockBackend{})
	if _, err := svc.CreateBackup(context.Background(), &CreateBackupRequest{}); err == nil {
		t.Fatal("CreateBackup() should fail when repo is nil")
	}
	if _, err := svc.ListBackups(context.Background(), &ListBackupsQuery{}); err == nil {
		t.Fatal("ListBackups() should fail when repo is nil")
	}
	if _, err := svc.TriggerRestore(context.Background(), 1, "system"); err == nil {
		t.Fatal("TriggerRestore() should fail when repo is nil")
	}
	if _, err := svc.RetryBackupTask(context.Background(), 1); err == nil {
		t.Fatal("RetryBackupTask() should fail when repo is nil")
	}
	if _, err := svc.RunBackupPolicy(context.Background(), &RunBackupPolicyRequest{CollectionName: "docs"}); err == nil {
		t.Fatal("RunBackupPolicy() should fail when repo is nil")
	}

	svc = NewServiceWithDeps(&mockRepo{}, &mockBackend{})
	if _, err := svc.CreateBackup(context.Background(), nil); err == nil {
		t.Fatal("CreateBackup(nil) should fail")
	}
	if _, err := svc.CreateBackup(context.Background(), &CreateBackupRequest{CollectionName: " "}); err == nil {
		t.Fatal("CreateBackup(empty collection) should fail")
	}
	if _, err := svc.TriggerRestore(context.Background(), 0, "system"); err == nil {
		t.Fatal("TriggerRestore(non-positive id) should fail")
	}
	if _, err := svc.RetryBackupTask(context.Background(), 0); err == nil {
		t.Fatal("RetryBackupTask(non-positive id) should fail")
	}
	if _, err := svc.RunBackupPolicy(context.Background(), nil); err == nil {
		t.Fatal("RunBackupPolicy(nil) should fail")
	}
	if _, err := svc.RunBackupPolicy(context.Background(), &RunBackupPolicyRequest{CollectionName: " "}); err == nil {
		t.Fatal("RunBackupPolicy(empty collection) should fail")
	}
}
