package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/constants"
	"ai-gateway/internal/routing"
)

const (
	goosDarwin = "darwin"
	goosLinux  = "linux"

	healthStatusUnknown   = "unknown"
	healthStatusHealthy   = "healthy"
	healthStatusUnhealthy = "unhealthy"
	healthStatusDisabled  = "disabled"

	preloadStatusSuccess = "success"
	preloadStatusFailed  = "failed"

	embeddingEndpointModeAuto       = "auto"
	embeddingEndpointModeEmbed      = "embed"
	embeddingEndpointModeEmbeddings = "embeddings"
)

type StartupMode string

const (
	StartupModeAuto   StartupMode = "auto"
	StartupModeApp    StartupMode = "app"
	StartupModeCLI    StartupMode = "cli"
	StartupModeManual StartupMode = "manual"
)

type MonitorConfig struct {
	Enabled                bool `json:"enabled"`
	CheckIntervalSeconds   int  `json:"check_interval_seconds"`
	AutoRestart            bool `json:"auto_restart"`
	MaxRestartAttempts     int  `json:"max_restart_attempts"`
	RestartCooldownSeconds int  `json:"restart_cooldown_seconds"`
}

type PreloadTargetKind string

const (
	PreloadTargetIntent    PreloadTargetKind = "intent"
	PreloadTargetEmbedding PreloadTargetKind = "embedding"
)

type PreloadConfig struct {
	AutoOnStartup  bool                `json:"auto_on_startup"`
	Targets        []PreloadTargetKind `json:"targets"`
	TimeoutSeconds int                 `json:"timeout_seconds"`
}

type PreloadTarget struct {
	Kind                  PreloadTargetKind `json:"kind"`
	Model                 string            `json:"model"`
	BaseURL               string            `json:"base_url"`
	EmbeddingEndpointMode string            `json:"embedding_endpoint_mode,omitempty"`
}

type PreloadResult struct {
	Kind       PreloadTargetKind `json:"kind"`
	Model      string            `json:"model"`
	Status     string            `json:"status"`
	DurationMs int64             `json:"duration_ms"`
	Error      string            `json:"error,omitempty"`
}

type OllamaServiceConfig struct {
	StartupMode           StartupMode   `json:"startup_mode"`
	AutoDetectPriority    []StartupMode `json:"auto_detect_priority"`
	Preload               PreloadConfig `json:"preload"`
	Monitoring            MonitorConfig `json:"monitoring"`
	StartupTimeoutSeconds int           `json:"startup_timeout_seconds"`
	HealthCheckTimeoutMs  int           `json:"health_check_timeout_ms"`
}

type MonitorStatus struct {
	Enabled         bool   `json:"enabled"`
	HealthStatus    string `json:"health_status"`
	LastCheckTime   string `json:"last_check_time"`
	RestartAttempts int    `json:"restart_attempts"`
	LastRestartTime string `json:"last_restart_time"`
	LastError       string `json:"last_error"`
}

type StartResult struct {
	Output         string      `json:"output"`
	StartupMode    StartupMode `json:"startup_mode"`
	Command        string      `json:"command"`
	AlreadyRunning bool        `json:"already_running"`
}

type StartError struct {
	Code        string
	Message     string
	Hint        string
	Output      string
	StartupMode StartupMode
	Command     string
}

func (e *StartError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return "failed to start ollama"
}

type OllamaService struct {
	mu sync.RWMutex

	config      OllamaServiceConfig
	monitor     MonitorStatus
	lastError   string
	activeMode  StartupMode
	lastCommand string
	lastOutput  string

	goos            string
	commandExistsFn func(string) bool
	appInstalledFn  func() bool
	runShellFn      func(time.Duration, string) (string, error)
	checkRunningFn  func(context.Context, *routing.ClassifierConfig) (bool, []string, string)
	sleepFn         func(time.Duration)
	nowFn           func() time.Time
}

func DefaultOllamaServiceConfig() OllamaServiceConfig {
	return OllamaServiceConfig{
		StartupMode:        StartupModeAuto,
		AutoDetectPriority: []StartupMode{StartupModeApp, StartupModeCLI},
		Preload: PreloadConfig{
			AutoOnStartup:  false,
			Targets:        []PreloadTargetKind{PreloadTargetIntent, PreloadTargetEmbedding},
			TimeoutSeconds: 180,
		},
		Monitoring: MonitorConfig{
			Enabled:                true,
			CheckIntervalSeconds:   30,
			AutoRestart:            true,
			MaxRestartAttempts:     3,
			RestartCooldownSeconds: 10,
		},
		StartupTimeoutSeconds: 12,
		HealthCheckTimeoutMs:  1500,
	}
}

func NewOllamaService(cfg *OllamaServiceConfig) *OllamaService {
	normalized := normalizeConfig(cfg)
	return &OllamaService{
		config:          normalized,
		goos:            runtime.GOOS,
		commandExistsFn: defaultCommandExists,
		appInstalledFn:  defaultOllamaAppInstalled,
		runShellFn:      defaultRunShellCommand,
		checkRunningFn:  defaultCheckRunning,
		sleepFn:         time.Sleep,
		nowFn:           time.Now,
		monitor: MonitorStatus{
			Enabled:      normalized.Monitoring.Enabled,
			HealthStatus: healthStatusUnknown,
		},
	}
}

func (s *OllamaService) GetConfig() OllamaServiceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *OllamaService) UpdateConfig(next *OllamaServiceConfig) OllamaServiceConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = normalizeConfig(next)
	s.monitor.Enabled = s.config.Monitoring.Enabled
	if !s.config.Monitoring.Enabled {
		s.monitor.HealthStatus = healthStatusDisabled
	}
	return s.config
}

func (s *OllamaService) GetMonitorStatus() MonitorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.monitor
}

func (s *OllamaService) GetLastError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

func (s *OllamaService) GetActiveMode() StartupMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.activeMode == "" {
		return s.config.StartupMode
	}
	return s.activeMode
}

func (s *OllamaService) Start(ctx context.Context, cfg *routing.ClassifierConfig) (*StartResult, error) {
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	running, _, _ := s.checkRunningFn(runCtx, cfg)
	cancel()
	if running {
		s.mu.Lock()
		s.lastError = ""
		s.monitor.HealthStatus = healthStatusHealthy
		s.mu.Unlock()
		return &StartResult{AlreadyRunning: true, StartupMode: s.GetActiveMode()}, nil
	}

	mode, command, err := s.resolveStartCommand()
	if err != nil {
		s.recordFailure(err)
		return nil, err
	}

	output, runErr := s.runShellFn(constants.AdminOllamaStartCommandTimeout, command)
	if runErr != nil {
		startErr := &StartError{
			Code:        "start_failed",
			Message:     runErr.Error(),
			Hint:        hintForMode(mode),
			Output:      output,
			StartupMode: mode,
			Command:     command,
		}
		s.recordFailure(startErr)
		return nil, startErr
	}

	deadline := s.nowFn().Add(time.Duration(s.GetConfig().StartupTimeoutSeconds) * time.Second)
	for s.nowFn().Before(deadline) {
		checkCtx, stop := context.WithTimeout(ctx, time.Duration(s.GetConfig().HealthCheckTimeoutMs)*time.Millisecond)
		alive, _, _ := s.checkRunningFn(checkCtx, cfg)
		stop()
		if alive {
			s.mu.Lock()
			s.activeMode = mode
			s.lastCommand = command
			s.lastOutput = output
			s.lastError = ""
			s.monitor.HealthStatus = healthStatusHealthy
			s.mu.Unlock()
			return &StartResult{Output: output, StartupMode: mode, Command: command}, nil
		}
		s.sleepFn(constants.AdminOllamaStartProbeInterval)
	}

	timeoutErr := &StartError{
		Code:        "start_timeout",
		Message:     "ollama did not become ready in time",
		Hint:        hintForMode(mode),
		Output:      output,
		StartupMode: mode,
		Command:     command,
	}
	s.recordFailure(timeoutErr)
	return nil, timeoutErr
}

func (s *OllamaService) CheckAndAutoRestart(ctx context.Context, cfg *routing.ClassifierConfig) error {
	running, _, detail := s.checkRunningFn(ctx, cfg)
	now := s.nowFn().Format(time.RFC3339)

	s.mu.Lock()
	s.monitor.LastCheckTime = now
	s.monitor.Enabled = s.config.Monitoring.Enabled
	if running {
		s.monitor.HealthStatus = healthStatusHealthy
		s.monitor.LastError = ""
		s.lastError = ""
		s.mu.Unlock()
		return nil
	}

	s.monitor.HealthStatus = healthStatusUnhealthy
	if strings.TrimSpace(detail) != "" {
		s.monitor.LastError = detail
		s.lastError = detail
	}

	cfgSnapshot := s.config
	restartAttempts := s.monitor.RestartAttempts
	lastRestart := s.monitor.LastRestartTime
	s.mu.Unlock()

	if !cfgSnapshot.Monitoring.Enabled || !cfgSnapshot.Monitoring.AutoRestart {
		return nil
	}
	if cfgSnapshot.StartupMode == StartupModeManual {
		return nil
	}
	if restartAttempts >= cfgSnapshot.Monitoring.MaxRestartAttempts {
		return nil
	}

	if lastRestart != "" && cfgSnapshot.Monitoring.RestartCooldownSeconds > 0 {
		if parsed, err := time.Parse(time.RFC3339, lastRestart); err == nil {
			if time.Since(parsed) < time.Duration(cfgSnapshot.Monitoring.RestartCooldownSeconds)*time.Second {
				return nil
			}
		}
	}

	_, err := s.Start(ctx, cfg)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.monitor.RestartAttempts++
	s.monitor.LastRestartTime = s.nowFn().Format(time.RFC3339)
	s.monitor.HealthStatus = healthStatusHealthy
	s.monitor.LastError = ""
	s.mu.Unlock()
	return nil
}

func (s *OllamaService) PreloadModels(ctx context.Context, targets []PreloadTarget, timeoutSeconds int) []PreloadResult {
	if len(targets) == 0 {
		return []PreloadResult{}
	}

	effectiveTimeoutSeconds := timeoutSeconds
	if effectiveTimeoutSeconds <= 0 {
		effectiveTimeoutSeconds = s.GetConfig().Preload.TimeoutSeconds
	}
	if effectiveTimeoutSeconds <= 0 {
		effectiveTimeoutSeconds = 180
	}

	results := make([]PreloadResult, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		model := strings.TrimSpace(target.Model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}

		start := time.Now()
		err := s.preloadOne(ctx, target, effectiveTimeoutSeconds)
		result := PreloadResult{
			Kind:       target.Kind,
			Model:      model,
			DurationMs: time.Since(start).Milliseconds(),
			Status:     preloadStatusSuccess,
		}
		if err != nil {
			result.Status = preloadStatusFailed
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results
}

func (s *OllamaService) preloadOne(ctx context.Context, target PreloadTarget, timeoutSeconds int) error {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 180
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	baseURL := strings.TrimSpace(target.BaseURL)
	if baseURL == "" {
		baseURL = constants.ClassifierDefaultBaseURL
	}

	switch target.Kind {
	case PreloadTargetIntent:
		return s.preloadIntentModel(requestCtx, baseURL, target.Model)
	case PreloadTargetEmbedding:
		return s.preloadEmbeddingModel(requestCtx, baseURL, target.Model, target.EmbeddingEndpointMode)
	default:
		return fmt.Errorf("unsupported preload target kind: %s", target.Kind)
	}
}

func (s *OllamaService) preloadIntentModel(ctx context.Context, baseURL, model string) error {
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "warmup"},
		},
		"stream":     false,
		"keep_alive": "-1m",
	}
	responseBody, statusCode, err := s.postOllamaJSON(ctx, baseURL, "/api/chat", payload)
	if err != nil {
		return err
	}
	if statusCode >= http.StatusBadRequest {
		return fmt.Errorf("ollama /api/chat status %d: %s", statusCode, extractOllamaError(responseBody))
	}
	if message := extractOllamaErrorFromSuccess(responseBody); message != "" {
		return fmt.Errorf("ollama /api/chat preload failed: %s", message)
	}
	return nil
}

func (s *OllamaService) preloadEmbeddingModel(ctx context.Context, baseURL, model, mode string) error {
	normalizedMode := normalizeEmbeddingEndpointMode(mode)
	if normalizedMode == embeddingEndpointModeEmbed {
		return s.preloadEmbeddingByPath(ctx, baseURL, model, "/api/embed")
	}
	if normalizedMode == embeddingEndpointModeEmbeddings {
		return s.preloadEmbeddingByPath(ctx, baseURL, model, "/api/embeddings")
	}

	if err := s.preloadEmbeddingByPath(ctx, baseURL, model, "/api/embed"); err == nil {
		return nil
	}
	return s.preloadEmbeddingByPath(ctx, baseURL, model, "/api/embeddings")
}

func (s *OllamaService) preloadEmbeddingByPath(ctx context.Context, baseURL, model, path string) error {
	payload := map[string]any{
		"model":      model,
		"keep_alive": "-1m",
	}
	if path == "/api/embed" {
		payload["input"] = "warmup"
	} else {
		payload["prompt"] = "warmup"
	}

	responseBody, statusCode, err := s.postOllamaJSON(ctx, baseURL, path, payload)
	if err != nil {
		return err
	}
	if statusCode >= http.StatusBadRequest {
		return fmt.Errorf("ollama %s status %d: %s", path, statusCode, extractOllamaError(responseBody))
	}
	if message := extractOllamaErrorFromSuccess(responseBody); message != "" {
		return fmt.Errorf("ollama %s preload failed: %s", path, message)
	}
	return nil
}

func (s *OllamaService) postOllamaJSON(ctx context.Context, baseURL, path string, payload map[string]any) (responseBody []byte, statusCode int, err error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal ollama preload payload: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build ollama preload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call ollama preload endpoint: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read ollama preload response: %w", err)
	}
	return responseBody, resp.StatusCode, nil
}

func (s *OllamaService) resolveStartCommand() (StartupMode, string, error) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if cfg.StartupMode == StartupModeManual {
		return StartupModeManual, "", &StartError{Code: "manual_mode", Message: "startup mode is manual", Hint: hintForMode(StartupModeManual)}
	}

	if s.goos != goosDarwin && s.goos != goosLinux {
		return "", "", &StartError{Code: "unsupported_os", Message: "current OS is not supported for auto start"}
	}

	if cfg.StartupMode == StartupModeAuto {
		for _, mode := range cfg.AutoDetectPriority {
			command, ok := s.commandForMode(mode)
			if ok {
				return mode, command, nil
			}
		}
		return "", "", &StartError{Code: "startup_mode_unavailable", Message: "no available startup mode found", Hint: hintForMode(StartupModeCLI)}
	}

	command, ok := s.commandForMode(cfg.StartupMode)
	if !ok {
		if cfg.StartupMode == StartupModeApp {
			return "", "", &StartError{Code: "app_not_installed", Message: "Ollama.app not found", Hint: hintForMode(StartupModeApp)}
		}
		if cfg.StartupMode == StartupModeCLI {
			return "", "", &StartError{Code: "ollama_not_installed", Message: "ollama not installed", Hint: hintForMode(StartupModeCLI)}
		}
		return "", "", &StartError{Code: "invalid_startup_mode", Message: fmt.Sprintf("unsupported startup mode: %s", cfg.StartupMode)}
	}

	return cfg.StartupMode, command, nil
}

func (s *OllamaService) commandForMode(mode StartupMode) (string, bool) {
	switch mode {
	case StartupModeAuto, StartupModeManual:
		return "", false
	case StartupModeApp:
		if s.goos != goosDarwin || !s.appInstalledFn() {
			return "", false
		}
		return "open -a Ollama", true
	case StartupModeCLI:
		if !s.commandExistsFn("ollama") {
			return "", false
		}
		return "nohup ollama serve >/tmp/ollama.log 2>&1 &", true
	default:
		return "", false
	}
}

func (s *OllamaService) recordFailure(err error) {
	message := err.Error()
	if se, ok := err.(*StartError); ok {
		message = se.Message
	}
	s.mu.Lock()
	s.lastError = message
	s.monitor.HealthStatus = healthStatusUnhealthy
	s.monitor.LastError = message
	s.mu.Unlock()
}

func normalizeConfig(cfg *OllamaServiceConfig) OllamaServiceConfig {
	if cfg == nil {
		defaults := DefaultOllamaServiceConfig()
		return defaults
	}

	normalized := *cfg
	if normalized.StartupMode == "" {
		normalized.StartupMode = StartupModeAuto
	}
	if len(normalized.AutoDetectPriority) == 0 {
		normalized.AutoDetectPriority = []StartupMode{StartupModeApp, StartupModeCLI}
	}
	if len(normalized.Preload.Targets) == 0 {
		normalized.Preload.Targets = []PreloadTargetKind{PreloadTargetIntent, PreloadTargetEmbedding}
	}
	normalized.Preload.Targets = normalizePreloadTargets(normalized.Preload.Targets)
	if normalized.Preload.TimeoutSeconds <= 0 {
		normalized.Preload.TimeoutSeconds = 180
	}
	if normalized.Monitoring.CheckIntervalSeconds <= 0 {
		normalized.Monitoring.CheckIntervalSeconds = 30
	}
	if normalized.Monitoring.MaxRestartAttempts <= 0 {
		normalized.Monitoring.MaxRestartAttempts = 3
	}
	if normalized.Monitoring.RestartCooldownSeconds < 0 {
		normalized.Monitoring.RestartCooldownSeconds = 0
	}
	if normalized.StartupTimeoutSeconds <= 0 {
		normalized.StartupTimeoutSeconds = 12
	}
	if normalized.HealthCheckTimeoutMs <= 0 {
		normalized.HealthCheckTimeoutMs = 1500
	}
	return normalized
}

func hintForMode(mode StartupMode) string {
	switch mode {
	case StartupModeAuto:
		return "自动模式会按优先级选择 App/CLI，当前可尝试切换为 CLI"
	case StartupModeApp:
		return "请先安装 Ollama.app，或切换为 CLI 启动方式"
	case StartupModeCLI:
		return "请先安装 ollama 命令行工具，或手动执行 ollama serve"
	case StartupModeManual:
		return "当前为手动模式，请在终端执行: ollama serve"
	default:
		return "请检查 Ollama 安装和运行状态"
	}
}

func normalizePreloadTargets(targets []PreloadTargetKind) []PreloadTargetKind {
	if len(targets) == 0 {
		return []PreloadTargetKind{PreloadTargetIntent, PreloadTargetEmbedding}
	}
	normalized := make([]PreloadTargetKind, 0, len(targets))
	seen := make(map[PreloadTargetKind]struct{}, len(targets))
	for _, target := range targets {
		value := PreloadTargetKind(strings.ToLower(strings.TrimSpace(string(target))))
		if value != PreloadTargetIntent && value != PreloadTargetEmbedding {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return []PreloadTargetKind{PreloadTargetIntent, PreloadTargetEmbedding}
	}
	return normalized
}

func normalizeEmbeddingEndpointMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == embeddingEndpointModeEmbed || normalized == embeddingEndpointModeEmbeddings {
		return normalized
	}
	return embeddingEndpointModeAuto
}

func extractOllamaError(body []byte) string {
	if len(body) == 0 {
		return "empty response"
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if value, ok := payload["error"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return strings.TrimSpace(string(body))
}

func extractOllamaErrorFromSuccess(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if value, ok := payload["error"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func defaultCommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func defaultOllamaAppInstalled() bool {
	paths := []string{
		"/Applications/Ollama.app",
		filepath.Join(os.Getenv("HOME"), "Applications", "Ollama.app"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func defaultRunShellCommand(timeout time.Duration, command string) (string, error) {
	return runShellCommand(timeout, command)
}

func defaultCheckRunning(ctx context.Context, cfg *routing.ClassifierConfig) (running bool, models []string, detail string) {
	return checkOllamaRunning(ctx, cfg)
}

func runShellCommand(timeout time.Duration, command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if output == "" {
			return "", err
		}
		return output, fmt.Errorf("%w: %s", err, output)
	}
	return output, nil
}

func checkOllamaRunning(ctx context.Context, cfg *routing.ClassifierConfig) (running bool, models []string, detail string) {
	if cfg == nil {
		def := routing.DefaultClassifierConfig()
		cfg = &def
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = constants.ClassifierDefaultBaseURL
	}

	timeout := 2 * time.Second
	if cfg.TimeoutMs > 0 {
		candidate := time.Duration(cfg.TimeoutMs) * time.Millisecond
		if candidate < timeout {
			timeout = candidate
		}
	}

	models, err := routing.ListOllamaModels(ctx, baseURL, timeout)
	if err != nil {
		return false, nil, err.Error()
	}
	return true, models, "ok"
}
