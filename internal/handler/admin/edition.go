package admin

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/config"

	"github.com/gin-gonic/gin"
)

type EditionHandler struct {
	configPath string
}

var dependencyStatusProvider = checkAllDependencies

type EditionSetupRuntime string

const (
	EditionSetupRuntimeDocker EditionSetupRuntime = "docker"
	EditionSetupRuntimeNative EditionSetupRuntime = "native"
)

type EditionSetupStatus string

const (
	EditionSetupStatusPending EditionSetupStatus = "pending"
	EditionSetupStatusRunning EditionSetupStatus = "running"
	EditionSetupStatusSuccess EditionSetupStatus = "success"
	EditionSetupStatusFailed  EditionSetupStatus = "failed"
)

const (
	editionSetupLogMaxLines      = 500
	editionSetupLogMaxLineLength = 2048
)

type EditionSetupRequest struct {
	Edition            config.EditionType  `json:"edition" binding:"required"`
	Runtime            EditionSetupRuntime `json:"runtime" binding:"required"`
	ApplyConfig        bool                `json:"apply_config"`
	PullEmbeddingModel bool                `json:"pull_embedding_model"`
}

type EditionSetupTask struct {
	TaskID     string                      `json:"task_id"`
	Edition    config.EditionType          `json:"edition"`
	Runtime    EditionSetupRuntime         `json:"runtime"`
	Status     EditionSetupStatus          `json:"status"`
	AcceptedAt time.Time                   `json:"accepted_at"`
	StartedAt  *time.Time                  `json:"started_at,omitempty"`
	FinishedAt *time.Time                  `json:"finished_at,omitempty"`
	Summary    string                      `json:"summary"`
	Logs       string                      `json:"logs"`
	Health     map[string]DependencyStatus `json:"health"`
	Message    string                      `json:"message,omitempty"`
}

var editionSetupTaskStore = struct {
	sync.RWMutex
	tasks map[string]*EditionSetupTask
}{
	tasks: map[string]*EditionSetupTask{},
}

type EditionSetupLogAppender func(line string)

type editionSetupExecutorFunc func(configPath string, req EditionSetupRequest, appendLog EditionSetupLogAppender) (string, error)

var editionSetupExecutor editionSetupExecutorFunc = runEditionSetupScript

func NewEditionHandler() *EditionHandler {
	return &EditionHandler{configPath: config.ResolveConfigPath()}
}

func isValidEditionSetupRuntime(runtime EditionSetupRuntime) bool {
	return runtime == EditionSetupRuntimeDocker || runtime == EditionSetupRuntimeNative
}

func generateEditionSetupTaskID() string {
	return fmt.Sprintf("edition-setup-%d", time.Now().UnixNano())
}

func cloneDependencyStatusMap(src map[string]DependencyStatus) map[string]DependencyStatus {
	if len(src) == 0 {
		return map[string]DependencyStatus{}
	}
	dst := make(map[string]DependencyStatus, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneEditionSetupTask(task *EditionSetupTask) *EditionSetupTask {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.Health = cloneDependencyStatusMap(task.Health)
	return &cloned
}

func upsertEditionSetupTask(task *EditionSetupTask) {
	editionSetupTaskStore.Lock()
	defer editionSetupTaskStore.Unlock()
	editionSetupTaskStore.tasks[task.TaskID] = cloneEditionSetupTask(task)
}

func getEditionSetupTask(taskID string) (*EditionSetupTask, bool) {
	editionSetupTaskStore.RLock()
	defer editionSetupTaskStore.RUnlock()
	task, ok := editionSetupTaskStore.tasks[taskID]
	if !ok {
		return nil, false
	}
	return cloneEditionSetupTask(task), true
}

func trimEditionSetupLogs(raw string, maxLines int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func truncateEditionSetupLogContent(raw string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	content := strings.TrimSpace(raw)
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

func formatEditionSetupLogLine(source string, line string) string {
	src := strings.TrimSpace(source)
	if src == "" {
		src = "stdout"
	}
	content := truncateEditionSetupLogContent(line, editionSetupLogMaxLineLength)
	if content == "" {
		return ""
	}
	return fmt.Sprintf("[%s][%s] %s", time.Now().UTC().Format(time.RFC3339), src, content)
}

func resolveEditionSetupScriptPath(configPath string) string {
	if absConfigPath, err := filepath.Abs(configPath); err == nil {
		candidate := filepath.Join(filepath.Dir(absConfigPath), "..", "scripts", "setup-edition-env.sh")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate
		}
	}
	return filepath.Clean("./scripts/setup-edition-env.sh")
}

func streamEditionSetupLogs(reader io.Reader, source string, appendLog EditionSetupLogAppender) {
	if reader == nil || appendLog == nil {
		return
	}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		formatted := formatEditionSetupLogLine(source, scanner.Text())
		if formatted == "" {
			continue
		}
		appendLog(formatted)
	}
}

func runEditionSetupScript(configPath string, req EditionSetupRequest, appendLog EditionSetupLogAppender) (string, error) {
	scriptPath := resolveEditionSetupScriptPath(configPath)
	args := []string{
		"--edition", string(req.Edition),
		"--runtime", string(req.Runtime),
		"--apply-config", strconv.FormatBool(req.ApplyConfig),
		"--pull-embedding-model", strconv.FormatBool(req.PullEmbeddingModel),
		"--config-path", configPath,
	}

	cmd := exec.Command(scriptPath, args...)
	summary := fmt.Sprintf("edition=%s runtime=%s", req.Edition, req.Runtime)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return summary, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return summary, err
	}

	if err := cmd.Start(); err != nil {
		return summary, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamEditionSetupLogs(stdoutPipe, "stdout", appendLog)
	}()
	go func() {
		defer wg.Done()
		streamEditionSetupLogs(stderrPipe, "stderr", appendLog)
	}()

	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		return summary, err
	}
	return summary, nil
}

func resetEditionSetupTasksForTest() {
	editionSetupTaskStore.Lock()
	defer editionSetupTaskStore.Unlock()
	editionSetupTaskStore.tasks = map[string]*EditionSetupTask{}
}

func (h *EditionHandler) executeSetupTask(taskID string, req EditionSetupRequest) {
	task, ok := getEditionSetupTask(taskID)
	if !ok {
		return
	}

	startedAt := time.Now().UTC()
	task.Status = EditionSetupStatusRunning
	task.StartedAt = &startedAt
	upsertEditionSetupTask(task)

	appendLog := func(line string) {
		logLine := strings.TrimSpace(line)
		if logLine == "" {
			return
		}
		currentTask, exists := getEditionSetupTask(taskID)
		if !exists {
			return
		}
		if currentTask.Logs == "" {
			currentTask.Logs = trimEditionSetupLogs(logLine, editionSetupLogMaxLines)
		} else {
			currentTask.Logs = trimEditionSetupLogs(currentTask.Logs+"\n"+logLine, editionSetupLogMaxLines)
		}
		upsertEditionSetupTask(currentTask)
	}

	summary, runErr := editionSetupExecutor(h.configPath, req, appendLog)
	cfg, cfgErr := config.LoadFromPath(h.configPath)
	if cfgErr != nil {
		cfg = config.DefaultConfig()
	}
	health := dependencyStatusProvider(cfg)

	if currentTask, exists := getEditionSetupTask(taskID); exists {
		task = currentTask
	}

	finishedAt := time.Now().UTC()
	task.FinishedAt = &finishedAt
	task.Summary = summary
	task.Health = cloneDependencyStatusMap(health)
	if runErr != nil {
		task.Status = EditionSetupStatusFailed
		task.Message = runErr.Error()
	} else {
		task.Status = EditionSetupStatusSuccess
		task.Message = "setup finished"
	}
	upsertEditionSetupTask(task)
}

func (h *EditionHandler) SetupEditionEnvironment(c *gin.Context) {
	var req EditionSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if !config.IsValidEditionType(req.Edition) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_edition",
			"message": "edition must be basic/standard/enterprise",
		})
		return
	}
	if !isValidEditionSetupRuntime(req.Runtime) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_runtime",
			"message": "runtime must be docker/native",
		})
		return
	}

	task := &EditionSetupTask{
		TaskID:     generateEditionSetupTaskID(),
		Edition:    req.Edition,
		Runtime:    req.Runtime,
		Status:     EditionSetupStatusPending,
		AcceptedAt: time.Now().UTC(),
		Health:     map[string]DependencyStatus{},
		Message:    "accepted",
	}
	upsertEditionSetupTask(task)

	go h.executeSetupTask(task.TaskID, req)

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"data": gin.H{
			"task_id":     task.TaskID,
			"accepted_at": task.AcceptedAt,
			"message":     "setup task accepted",
		},
	})
}

func (h *EditionHandler) GetSetupTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("taskId"))
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_task",
			"message": "task id is required",
		})
		return
	}

	task, ok := getEditionSetupTask(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "task_not_found",
			"message": "setup task not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
	})
}

func (h *EditionHandler) GetEdition(c *gin.Context) {
	cfg, err := config.LoadFromPath(h.configPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "load_config_failed",
			"message": err.Error(),
		})
		return
	}

	editionData := buildEditionResponseData(cfg)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    editionData,
	})
}

func (h *EditionHandler) GetEditionDefinitions(c *gin.Context) {
	defs := make([]config.EditionDefinition, 0, len(config.EditionDefinitions))
	ordered := []config.EditionType{config.EditionBasic, config.EditionStandard, config.EditionEnterprise}
	for _, key := range ordered {
		defs = append(defs, config.EditionDefinitions[key])
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    defs,
	})
}

func (h *EditionHandler) CheckDependencies(c *gin.Context) {
	cfg, err := config.LoadFromPath(h.configPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "load_config_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dependencyStatusProvider(cfg),
	})
}

func (h *EditionHandler) UpdateEdition(c *gin.Context) {
	var req struct {
		Type config.EditionType `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if !config.IsValidEditionType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_edition",
			"message": "edition must be basic/standard/enterprise",
		})
		return
	}

	currentCfg, err := config.LoadFromPath(h.configPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "load_config_failed",
			"message": err.Error(),
		})
		return
	}

	def := config.EditionDefinitions[req.Type]
	missing := collectMissingDependencies(&def, dependencyStatusProvider(currentCfg))
	if len(missing) > 0 {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"success": false,
			"error":   "missing_dependencies",
			"message": "缺少必需依赖服务",
			"data": gin.H{
				"missing": missing,
			},
		})
		return
	}

	updatedCfg, err := config.UpdateEditionInFile(h.configPath, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "update_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "版本配置已更新，重启后可确保全量生效",
		"data": gin.H{
			"restart_required": true,
			"edition":          buildEditionResponseData(updatedCfg),
		},
	})
}

func buildEditionResponseData(cfg *config.Config) gin.H {
	def := cfg.GetEditionConfig()
	return gin.H{
		"type":                def.Type,
		"features":            def.Features,
		"display_name":        def.DisplayName,
		"description":         def.Description,
		"dependencies":        def.Dependencies,
		"runtime":             cfg.Edition.Runtime,
		"dependency_versions": cfg.Edition.DependencyVersions,
	}
}

func collectMissingDependencies(def *config.EditionDefinition, status map[string]DependencyStatus) []string {
	missing := make([]string, 0)
	for _, dep := range def.Dependencies {
		d, ok := status[dep]
		if !ok || !d.Healthy {
			missing = append(missing, dep)
		}
	}
	return missing
}
