package audit

import (
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

func (l *Logger) LogAction(userID, username, ip, userAgent string, action ActionType, resource ResourceType, resourceID, detail string, oldData, newData interface{}, status string, err string) {
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
		Error:      err,
	})
}

func (l *Logger) GetLogs(limit, offset int, filters map[string]interface{}) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]LogEntry, 0)
	for _, log := range l.logs {
		if l.matchFilters(log, filters) {
			result = append(result, log)
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
	for _, log := range l.logs {
		if l.matchFilters(log, filters) {
			count++
		}
	}
	return count
}

func (l *Logger) matchFilters(log LogEntry, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "user_id":
			if log.UserID != value.(string) {
				return false
			}
		case "action":
			if log.Action != value.(ActionType) {
				return false
			}
		case "resource":
			if log.Resource != value.(ResourceType) {
				return false
			}
		case "resource_id":
			if log.ResourceID != value.(string) {
				return false
			}
		case "status":
			if log.Status != value.(string) {
				return false
			}
		case "start_time":
			if log.Timestamp.Before(value.(time.Time)) {
				return false
			}
		case "end_time":
			if log.Timestamp.After(value.(time.Time)) {
				return false
			}
		}
	}
	return true
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
	os.MkdirAll(dir, 0755)

	os.WriteFile(l.filePath, data, 0644)
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
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
