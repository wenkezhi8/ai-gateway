package audit

import (
	cryptorand "crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ActionType string

const (
	ActionLogin       ActionType = "login"
	ActionLogout      ActionType = "logout"
	ActionCreate      ActionType = "create"
	ActionUpdate      ActionType = "update"
	ActionDelete      ActionType = "delete"
	ActionSwitch      ActionType = "switch"
	ActionConfig      ActionType = "config"
	ActionCacheClear  ActionType = "cache_clear"
	ActionTestConnect ActionType = "test_connect"
	ActionForceSwitch ActionType = "force_switch"
)

type ResourceType string

const (
	ResourceAccount  ResourceType = "account"
	ResourceProvider ResourceType = "provider"
	ResourceRouting  ResourceType = "routing"
	ResourceCache    ResourceType = "cache"
	ResourceConfig   ResourceType = "config"
	ResourceAuth     ResourceType = "auth"
	ResourceSystem   ResourceType = "system"
)

type LogEntry struct {
	ID         string       `json:"id"`
	Timestamp  time.Time    `json:"timestamp"`
	UserID     string       `json:"user_id"`
	Username   string       `json:"username"`
	IP         string       `json:"ip"`
	UserAgent  string       `json:"user_agent"`
	Action     ActionType   `json:"action"`
	Resource   ResourceType `json:"resource"`
	ResourceID string       `json:"resource_id"`
	Detail     string       `json:"detail"`
	OldData    interface{}  `json:"old_data,omitempty"`
	NewData    interface{}  `json:"new_data,omitempty"`
	Status     string       `json:"status"`
	Error      string       `json:"error,omitempty"`
}

type Logger struct {
	mu       sync.RWMutex
	logs     []LogEntry
	filePath string
	maxLogs  int
	stopCh   chan struct{}
}

var (
	globalLogger *Logger
	once         sync.Once
)

func InitLogger(filePath string, maxLogs int) *Logger {
	once.Do(func() {
		globalLogger = &Logger{
			logs:     make([]LogEntry, 0),
			filePath: filePath,
			maxLogs:  maxLogs,
			stopCh:   make(chan struct{}),
		}
		globalLogger.loadFromFile()
	})
	return globalLogger
}

func GetLogger() *Logger {
	return globalLogger
}

//nolint:gocritic // Kept by-value for API compatibility.
func (l *Logger) Log(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	entry.ID = generateID()

	l.logs = append(l.logs, entry)

	if l.maxLogs > 0 && len(l.logs) > l.maxLogs {
		l.logs = l.logs[len(l.logs)-l.maxLogs:]
	}

	go l.saveToFile()
}

func (l *Logger) LogAction(userID, username, ip, userAgent string, action ActionType, resource ResourceType, resourceID, detail string, oldData, newData interface{}, status, errMsg string) {
	l.Log(LogEntry{
		UserID:     userID,
		Username:   username,
		IP:         ip,
		UserAgent:  userAgent,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     detail,
		OldData:    oldData,
		NewData:    newData,
		Status:     status,
		Error:      errMsg,
	})
}

func (l *Logger) GetLogs(limit, offset int, filters map[string]interface{}) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]LogEntry, 0)
	for i := range l.logs {
		logEntry := &l.logs[i]
		if l.matchFilters(logEntry, filters) {
			result = append(result, *logEntry)
		}
	}

	if offset >= len(result) {
		return []LogEntry{}
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end]
}

func (l *Logger) GetLogsByUser(userID string, limit int) []LogEntry {
	return l.GetLogs(limit, 0, map[string]interface{}{"user_id": userID})
}

func (l *Logger) GetLogsByResource(resource ResourceType, limit int) []LogEntry {
	return l.GetLogs(limit, 0, map[string]interface{}{"resource": resource})
}

func (l *Logger) GetLogsByAction(action ActionType, limit int) []LogEntry {
	return l.GetLogs(limit, 0, map[string]interface{}{"action": action})
}

func (l *Logger) Count(filters map[string]interface{}) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	count := 0
	for i := range l.logs {
		if l.matchFilters(&l.logs[i], filters) {
			count++
		}
	}
	return count
}

func (l *Logger) matchFilters(log *LogEntry, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		if !matchFilterValue(log, key, value) {
			return false
		}
	}
	return true
}

func matchFilterValue(log *LogEntry, key string, value interface{}) bool {
	switch key {
	case "user_id":
		expected, ok := value.(string)
		return ok && log.UserID == expected
	case "action":
		expected, ok := value.(ActionType)
		return ok && log.Action == expected
	case "resource":
		expected, ok := value.(ResourceType)
		return ok && log.Resource == expected
	case "resource_id":
		expected, ok := value.(string)
		return ok && log.ResourceID == expected
	case "status":
		expected, ok := value.(string)
		return ok && log.Status == expected
	case "start_time":
		expected, ok := value.(time.Time)
		return ok && !log.Timestamp.Before(expected)
	case "end_time":
		expected, ok := value.(time.Time)
		return ok && !log.Timestamp.After(expected)
	default:
		return true
	}
}

func (l *Logger) loadFromFile() {
	if l.filePath == "" {
		return
	}

	dir := filepath.Dir(l.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	data, err := os.ReadFile(l.filePath)
	if err != nil {
		return
	}

	var logs []LogEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		return
	}

	l.logs = logs
}

func (l *Logger) saveToFile() {
	if l.filePath == "" {
		return
	}

	l.mu.RLock()
	data, err := json.MarshalIndent(l.logs, "", "  ")
	l.mu.RUnlock()

	if err != nil {
		return
	}

	dir := filepath.Dir(l.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	if err := os.WriteFile(l.filePath, data, 0640); err != nil {
		return
	}
}

func (l *Logger) Close() {
	close(l.stopCh)
	l.saveToFile()
}

func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = make([]LogEntry, 0)
	go l.saveToFile()
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if n <= 0 {
		return ""
	}

	raw := make([]byte, n)
	if _, err := cryptorand.Read(raw); err != nil {
		// Fallback keeps behavior available even if secure RNG is unavailable.
		seed := time.Now().UnixNano()
		shifts := []uint{0, 8, 16, 24, 32, 40, 48, 56}
		for i := range raw {
			shift := shifts[i%8]
			raw[i] = byte(seed>>shift) + byte(i*31)
		}
	}

	b := make([]byte, n)
	for i := range b {
		b[i] = letters[int(raw[i])%len(letters)]
	}
	return string(b)
}
