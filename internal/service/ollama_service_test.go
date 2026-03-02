package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ai-gateway/internal/routing"
)

func TestOllamaService_ResolveStartCommand_AutoFallbackToCLI(t *testing.T) {
	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	svc.goos = goosDarwin
	svc.config.StartupMode = StartupModeAuto
	svc.appInstalledFn = func() bool { return false }
	svc.commandExistsFn = func(name string) bool { return name == "ollama" }

	mode, command, err := svc.resolveStartCommand()
	if err != nil {
		t.Fatalf("resolveStartCommand returned error: %v", err)
	}
	if mode != StartupModeCLI {
		t.Fatalf("expected mode %q, got %q", StartupModeCLI, mode)
	}
	if !strings.Contains(command, "ollama serve") {
		t.Fatalf("expected CLI command to contain ollama serve, got %q", command)
	}
}

func TestOllamaService_Start_ManualModeReturnsError(t *testing.T) {
	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	svc.config.StartupMode = StartupModeManual
	svc.checkRunningFn = func(context.Context, *routing.ClassifierConfig) (bool, []string, string) {
		return false, nil, "not running"
	}

	_, err := svc.Start(context.Background(), &routing.ClassifierConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var startErr *StartError
	if !errors.As(err, &startErr) {
		t.Fatalf("expected StartError, got %T", err)
	}
	if startErr.Code != "manual_mode" {
		t.Fatalf("expected error code manual_mode, got %q", startErr.Code)
	}
}

func TestOllamaService_CheckAndAutoRestart_WhenUnhealthy(t *testing.T) {
	defaultCfg := DefaultOllamaServiceConfig()
	svc := NewOllamaService(&defaultCfg)
	svc.config.StartupMode = StartupModeCLI
	svc.config.Monitoring.Enabled = true
	svc.config.Monitoring.AutoRestart = true
	svc.config.Monitoring.MaxRestartAttempts = 3
	svc.config.Monitoring.RestartCooldownSeconds = 0
	svc.config.StartupTimeoutSeconds = 1

	running := false
	svc.checkRunningFn = func(context.Context, *routing.ClassifierConfig) (bool, []string, string) {
		if running {
			return true, nil, "ok"
		}
		return false, nil, "not running"
	}
	svc.commandExistsFn = func(name string) bool { return name == "ollama" }
	svc.runShellFn = func(time.Duration, string) (string, error) {
		running = true
		return "started", nil
	}
	svc.sleepFn = func(time.Duration) {}

	err := svc.CheckAndAutoRestart(context.Background(), &routing.ClassifierConfig{})
	if err != nil {
		t.Fatalf("CheckAndAutoRestart returned error: %v", err)
	}
	if !running {
		t.Fatal("expected service to become running after auto restart")
	}
	status := svc.GetMonitorStatus()
	if status.RestartAttempts != 1 {
		t.Fatalf("expected restart attempts to be 1, got %d", status.RestartAttempts)
	}
}
