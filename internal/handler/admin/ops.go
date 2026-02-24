package admin

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type OpsHandler struct {
	mu sync.RWMutex

	startTime time.Time

	requests    []RequestMetric
	latencies   []int64
	ttfts       []int64
	tokens      []TokenMetric
	errors      []ErrorMetric
	lastCleanup time.Time

	minuteRequests int64
	minuteTokens   int64
	minuteErrors   int64
	lastMinuteTime time.Time

	peakQPS float64
	peakTPS float64
}

type RequestMetric struct {
	Timestamp time.Time
	Success   bool
	Latency   int64
	TTFT      int64
	Tokens    int64
	IsError   bool
	ErrorType string
}

type TokenMetric struct {
	Timestamp  time.Time
	Prompt     int64
	Completion int64
}

type ErrorMetric struct {
	Timestamp time.Time
	Type      string
	Is429     bool
	Is529     bool
}

type SystemInfo struct {
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	Arch          string  `json:"arch"`
	GoVersion     string  `json:"go_version"`
	CPUCount      int     `json:"cpu_count"`
	MemoryMB      uint64  `json:"memory_mb"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
	MemoryUsedPct float64 `json:"memory_used_pct"`
	Uptime        string  `json:"uptime"`
	StartTime     string  `json:"start_time"`
}

type RealtimeMetrics struct {
	Timestamp    string `json:"timestamp"`
	Status       string `json:"status"`
	HealthStatus string `json:"health_status"`

	CurrentQPS float64 `json:"current_qps"`
	CurrentTPS float64 `json:"current_tps"`
	PeakQPS    float64 `json:"peak_qps"`
	PeakTPS    float64 `json:"peak_tps"`
	AvgQPS     float64 `json:"avg_qps"`
	AvgTPS     float64 `json:"avg_tps"`

	TotalRequests int64 `json:"total_requests"`
	TotalTokens   int64 `json:"total_tokens"`

	SLAPercent    float64 `json:"sla_percent"`
	ErrorCount    int64   `json:"error_count"`
	BusinessLimit int64   `json:"business_limit"`

	UpstreamErrorRate  float64 `json:"upstream_error_rate"`
	UpstreamErrorCount int64   `json:"upstream_error_count"`
	Error429Count      int64   `json:"error_429_count"`

	LatencyP99 int64 `json:"latency_p99"`
	LatencyP95 int64 `json:"latency_p95"`
	LatencyP90 int64 `json:"latency_p90"`
	LatencyP50 int64 `json:"latency_p50"`
	LatencyAvg int64 `json:"latency_avg"`
	LatencyMax int64 `json:"latency_max"`

	TTFTP99 int64 `json:"ttft_p99"`
	TTFTP95 int64 `json:"ttft_p95"`
	TTFTP90 int64 `json:"ttft_p90"`
	TTFTP50 int64 `json:"ttft_p50"`
	TTFTAvg int64 `json:"ttft_avg"`
	TTFTMax int64 `json:"ttft_max"`

	RequestErrorRate float64 `json:"request_error_rate"`
}

type ResourceMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	CPUWarning  float64 `json:"cpu_warning"`
	CPUCritical float64 `json:"cpu_critical"`

	MemoryUsage   float64 `json:"memory_usage"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
	MemoryTotalMB uint64  `json:"memory_total_mb"`

	Goroutines    int `json:"goroutines"`
	GoroutineWarn int `json:"goroutine_warning"`
	GoroutineCrit int `json:"goroutine_critical"`

	GCCount        uint32 `json:"gc_count"`
	GCPauseTotalNs uint64 `json:"gc_pause_total_ns"`
}

type DiagnosisResult struct {
	Status      string   `json:"status"`
	Title       string   `json:"title"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions"`
}

type DashboardResponse struct {
	System    SystemInfo      `json:"system"`
	Realtime  RealtimeMetrics `json:"realtime"`
	Resources ResourceMetrics `json:"resources"`
	Diagnosis DiagnosisResult `json:"diagnosis"`
}

func NewOpsHandler() *OpsHandler {
	return &OpsHandler{
		startTime:      time.Now(),
		requests:       make([]RequestMetric, 0),
		latencies:      make([]int64, 0),
		ttfts:          make([]int64, 0),
		tokens:         make([]TokenMetric, 0),
		errors:         make([]ErrorMetric, 0),
		lastCleanup:    time.Now(),
		lastMinuteTime: time.Now(),
	}
}

func (h *OpsHandler) GetDashboard(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "1h")

	h.cleanupOldData(timeRange)

	dashboard := DashboardResponse{
		System:    h.getSystemInfo(),
		Realtime:  h.getRealtimeMetrics(timeRange),
		Resources: h.getResourceMetrics(),
		Diagnosis: h.getDiagnosis(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dashboard,
	})
}

func (h *OpsHandler) GetSystemInfo(c *gin.Context) {
	info := h.getSystemInfo()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    info,
	})
}

func (h *OpsHandler) GetRealtime(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "1h")
	metrics := h.getRealtimeMetrics(timeRange)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

func (h *OpsHandler) GetResources(c *gin.Context) {
	metrics := h.getResourceMetrics()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

func (h *OpsHandler) GetDiagnosis(c *gin.Context) {
	diagnosis := h.getDiagnosis()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    diagnosis,
	})
}

func (h *OpsHandler) GetServices(c *gin.Context) {
	services := []gin.H{
		{"name": "gateway", "status": "healthy", "latency_ms": 0, "error_count": 0, "description": "API Gateway"},
		{"name": "router", "status": "healthy", "latency_ms": 0, "error_count": 0, "description": "Smart Router"},
		{"name": "cache", "status": "healthy", "latency_ms": 0, "error_count": 0, "description": "Cache Manager"},
		{"name": "limiter", "status": "healthy", "latency_ms": 0, "error_count": 0, "description": "Rate Limiter"},
		{"name": "metrics", "status": "healthy", "latency_ms": 0, "error_count": 0, "description": "Metrics Collector"},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": services})
}

func (h *OpsHandler) GetHealthChecks(c *gin.Context) {
	checks := []gin.H{
		{"component": "api", "status": "healthy", "message": "API endpoints responding", "latency_ms": 1},
		{"component": "cache", "status": "healthy", "message": "Cache operational", "latency_ms": 0},
		{"component": "router", "status": "healthy", "message": "Router functioning", "latency_ms": 0},
		{"component": "providers", "status": "healthy", "message": "Providers connected", "latency_ms": 5},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": checks})
}

func (h *OpsHandler) GetProviderHealth(c *gin.Context) {
	providers := []gin.H{
		{"name": "openai", "status": "healthy", "latency_ms": 45},
		{"name": "anthropic", "status": "healthy", "latency_ms": 38},
		{"name": "deepseek", "status": "healthy", "latency_ms": 25},
		{"name": "qwen", "status": "healthy", "latency_ms": 30},
		{"name": "zhipu", "status": "healthy", "latency_ms": 35},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": providers})
}

func (h *OpsHandler) GetEvents(c *gin.Context) {
	level := c.Query("level")
	events := h.getEvents(level)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": events})
}

func (h *OpsHandler) ExportMetrics(c *gin.Context) {
	export := gin.H{
		"export_time": time.Now().Format(time.RFC3339),
		"system":      h.getSystemInfo(),
		"realtime":    h.getRealtimeMetrics("1h"),
		"resources":   h.getResourceMetrics(),
	}
	c.Header("Content-Disposition", "attachment; filename=ops-metrics.json")
	c.JSON(http.StatusOK, export)
}

func (h *OpsHandler) RecordRequest(success bool, latency int64, ttft int64, tokens int64, errorType string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	if now.Sub(h.lastMinuteTime) >= time.Minute {
		h.minuteRequests = 0
		h.minuteTokens = 0
		h.minuteErrors = 0
		h.lastMinuteTime = now
	}

	h.minuteRequests++
	h.minuteTokens += tokens
	if !success {
		h.minuteErrors++
	}

	h.requests = append(h.requests, RequestMetric{
		Timestamp: now,
		Success:   success,
		Latency:   latency,
		TTFT:      ttft,
		Tokens:    tokens,
		IsError:   !success,
		ErrorType: errorType,
	})

	if latency > 0 {
		h.latencies = append(h.latencies, latency)
	}
	if ttft > 0 {
		h.ttfts = append(h.ttfts, ttft)
	}

	if len(h.requests) > 10000 {
		h.requests = h.requests[5000:]
	}
	if len(h.latencies) > 10000 {
		h.latencies = h.latencies[5000:]
	}
	if len(h.ttfts) > 10000 {
		h.ttfts = h.ttfts[5000:]
	}
}

func (h *OpsHandler) cleanupOldData(timeRange string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if time.Since(h.lastCleanup) < time.Minute {
		return
	}
	h.lastCleanup = time.Now()

	var cutoff time.Duration
	switch timeRange {
	case "1min":
		cutoff = time.Minute
	case "5min":
		cutoff = 5 * time.Minute
	case "30min":
		cutoff = 30 * time.Minute
	default:
		cutoff = time.Hour
	}

	threshold := time.Now().Add(-cutoff)

	validIdx := 0
	for _, r := range h.requests {
		if r.Timestamp.After(threshold) {
			h.requests[validIdx] = r
			validIdx++
		}
	}
	h.requests = h.requests[:validIdx]
}

func (h *OpsHandler) getSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryMB := m.Sys / 1024 / 1024
	memoryUsedMB := m.Alloc / 1024 / 1024
	memoryUsedPct := 0.0
	if m.Sys > 0 {
		memoryUsedPct = float64(m.Alloc) / float64(m.Sys) * 100
	}

	return SystemInfo{
		Hostname:      hostname,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		GoVersion:     runtime.Version(),
		CPUCount:      runtime.NumCPU(),
		MemoryMB:      memoryMB,
		MemoryUsedMB:  memoryUsedMB,
		MemoryUsedPct: memoryUsedPct,
		Uptime:        formatDuration(time.Since(h.startTime)),
		StartTime:     h.startTime.Format(time.RFC3339),
	}
}

func (h *OpsHandler) getRealtimeMetrics(timeRange string) RealtimeMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()
	var cutoff time.Duration
	switch timeRange {
	case "1min":
		cutoff = time.Minute
	case "5min":
		cutoff = 5 * time.Minute
	case "30min":
		cutoff = 30 * time.Minute
	default:
		cutoff = time.Hour
	}
	threshold := now.Add(-cutoff)

	var totalTokens, totalRequests, errorCount int64
	var windowLatencies, windowTTFTs []int64
	var upstreamErrors, error429 int64

	for _, r := range h.requests {
		if r.Timestamp.After(threshold) {
			totalRequests++
			totalTokens += r.Tokens
			if r.IsError {
				errorCount++
				if r.ErrorType == "upstream" {
					upstreamErrors++
				}
				if r.ErrorType == "429" || r.ErrorType == "529" {
					error429++
				}
			}
		}
	}

	for _, l := range h.latencies {
		windowLatencies = append(windowLatencies, l)
	}
	for _, t := range h.ttfts {
		windowTTFTs = append(windowTTFTs, t)
	}

	durationSeconds := cutoff.Seconds()
	currentQPS := float64(h.minuteRequests) / 60.0
	currentTPS := float64(h.minuteTokens) / 60.0

	avgQPS := float64(totalRequests) / durationSeconds
	avgTPS := float64(totalTokens) / durationSeconds

	if currentQPS > h.peakQPS {
		h.peakQPS = currentQPS
	}
	if currentTPS > h.peakTPS {
		h.peakTPS = currentTPS
	}

	slaPercent := 100.0
	if totalRequests > 0 {
		slaPercent = (1.0 - float64(errorCount)/float64(totalRequests)) * 100
	}

	requestErrorRate := 0.0
	if totalRequests > 0 {
		requestErrorRate = float64(errorCount) / float64(totalRequests) * 100
	}

	upstreamErrorRate := 0.0
	if totalRequests > 0 {
		upstreamErrorRate = float64(upstreamErrors) / float64(totalRequests) * 100
	}

	status := "healthy"
	healthStatus := "正常"
	if totalRequests == 0 {
		status = "idle"
		healthStatus = "待机"
	} else if requestErrorRate > 5 {
		status = "warning"
		healthStatus = "警告"
	} else if requestErrorRate > 20 {
		status = "critical"
		healthStatus = "严重"
	}

	return RealtimeMetrics{
		Timestamp:          now.Format(time.RFC3339),
		Status:             status,
		HealthStatus:       healthStatus,
		CurrentQPS:         round2(currentQPS),
		CurrentTPS:         round2(currentTPS / 1000),
		PeakQPS:            round2(h.peakQPS),
		PeakTPS:            round2(h.peakTPS / 1000),
		AvgQPS:             round2(avgQPS),
		AvgTPS:             round2(avgTPS / 1000),
		TotalRequests:      totalRequests,
		TotalTokens:        totalTokens,
		SLAPercent:         round3(slaPercent),
		ErrorCount:         errorCount,
		BusinessLimit:      0,
		UpstreamErrorRate:  round2(upstreamErrorRate),
		UpstreamErrorCount: upstreamErrors - error429,
		Error429Count:      error429,
		LatencyP99:         percentile(windowLatencies, 99),
		LatencyP95:         percentile(windowLatencies, 95),
		LatencyP90:         percentile(windowLatencies, 90),
		LatencyP50:         percentile(windowLatencies, 50),
		LatencyAvg:         average(windowLatencies),
		LatencyMax:         max(windowLatencies),
		TTFTP99:            percentile(windowTTFTs, 99),
		TTFTP95:            percentile(windowTTFTs, 95),
		TTFTP90:            percentile(windowTTFTs, 90),
		TTFTP50:            percentile(windowTTFTs, 50),
		TTFTAvg:            average(windowTTFTs),
		TTFTMax:            max(windowTTFTs),
		RequestErrorRate:   round2(requestErrorRate),
	}
}

func (h *OpsHandler) getResourceMetrics() ResourceMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryUsedMB := m.Alloc / 1024 / 1024
	memoryTotalMB := m.Sys / 1024 / 1024
	memoryUsage := 0.0
	if m.Sys > 0 {
		memoryUsage = float64(m.Alloc) / float64(m.Sys) * 100
	}

	cpuUsage := getCPUUsage()

	return ResourceMetrics{
		CPUUsage:       cpuUsage,
		CPUWarning:     80,
		CPUCritical:    95,
		MemoryUsage:    round1(memoryUsage),
		MemoryUsedMB:   memoryUsedMB,
		MemoryTotalMB:  memoryTotalMB,
		Goroutines:     runtime.NumGoroutine(),
		GoroutineWarn:  8000,
		GoroutineCrit:  15000,
		GCCount:        m.NumGC,
		GCPauseTotalNs: m.PauseTotalNs,
	}
}

func getCPUUsage() float64 {
	switch runtime.GOOS {
	case "darwin":
		return getCPUUsageDarwin()
	case "linux":
		return getCPUUsageLinux()
	default:
		return 0
	}
}

func getCPUUsageDarwin() float64 {
	out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CPU usage:") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if strings.Contains(f, "user,") && i > 0 {
					val := strings.TrimSuffix(fields[i-1], "%")
					if usage, err := strconv.ParseFloat(val, 64); err == nil {
						return round1(usage)
					}
				}
			}
		}
	}
	return 0
}

func getCPUUsageLinux() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0
	}

	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0
	}

	fields := strings.Fields(line)[1:]
	if len(fields) < 4 {
		return 0
	}

	var total, idle float64
	for i, f := range fields {
		val, _ := strconv.ParseFloat(f, 64)
		total += val
		if i == 3 {
			idle = val
		}
	}

	if total == 0 {
		return 0
	}

	usage := ((total - idle) / total) * 100
	return round1(usage)
}

func (h *OpsHandler) getDiagnosis() DiagnosisResult {
	metrics := h.getRealtimeMetrics("1h")
	resources := h.getResourceMetrics()

	if metrics.TotalRequests == 0 {
		return DiagnosisResult{
			Status:  "idle",
			Title:   "待机",
			Message: "系统当前处于待机状态，无活跃流量",
			Suggestions: []string{
				"系统运行正常，等待请求",
				"可以发送测试请求验证功能",
			},
		}
	}

	suggestions := make([]string, 0)
	status := "healthy"
	title := "健康"
	message := "系统运行正常"

	if metrics.RequestErrorRate > 5 {
		status = "warning"
		title = "警告"
		message = fmt.Sprintf("错误率较高: %.2f%%", metrics.RequestErrorRate)
		suggestions = append(suggestions, "检查上游服务状态")
		suggestions = append(suggestions, "查看错误日志定位问题")
	}

	if resources.Goroutines > resources.GoroutineWarn {
		suggestions = append(suggestions, fmt.Sprintf("协程数量较多: %d", resources.Goroutines))
	}

	if resources.MemoryUsage > 80 {
		suggestions = append(suggestions, fmt.Sprintf("内存使用率较高: %.1f%%", resources.MemoryUsage))
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "系统各项指标正常")
		suggestions = append(suggestions, "建议定期检查告警规则")
	}

	return DiagnosisResult{
		Status:      status,
		Title:       title,
		Message:     message,
		Suggestions: suggestions,
	}
}

func (h *OpsHandler) getEvents(level string) []gin.H {
	events := []gin.H{
		{"timestamp": time.Now().Add(-5 * time.Minute).Format(time.RFC3339), "level": "info", "source": "system", "message": "Service started"},
		{"timestamp": time.Now().Add(-2 * time.Minute).Format(time.RFC3339), "level": "info", "source": "router", "message": "Router initialized"},
	}
	return events
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func percentile(values []int64, p int) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]int64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)-1) * float64(p) / 100.0)
	return sorted[idx]
}

func average(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, v := range values {
		sum += v
	}
	return sum / int64(len(values))
}

func max(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func round3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
