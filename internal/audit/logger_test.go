package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *Logger {
	return &Logger{
		logs:     make([]LogEntry, 0),
		filePath: "",
		maxLogs:  100,
		stopCh:   make(chan struct{}),
	}
}

func TestInitLogger(t *testing.T) {
	globalLogger = nil
	once = sync.Once{}

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.json")

	logger := InitLogger(logPath, 100)
	require.NotNil(t, logger)

	logger2 := GetLogger()
	assert.Equal(t, logger, logger2)
}

func TestLogger_Log(t *testing.T) {
	logger := newTestLogger()

	entry := LogEntry{
		UserID:     "user-1",
		Username:   "testuser",
		IP:         "127.0.0.1",
		Action:     ActionLogin,
		Resource:   ResourceAuth,
		ResourceID: "session-1",
		Status:     "success",
	}

	logger.Log(entry)

	logs := logger.GetLogs(10, 0, nil)
	require.Len(t, logs, 1)
	assert.Equal(t, "user-1", logs[0].UserID)
	assert.NotEmpty(t, logs[0].ID)
	assert.False(t, logs[0].Timestamp.IsZero())
}

func TestLogger_LogAction(t *testing.T) {
	logger := newTestLogger()

	logger.LogAction("user-1", "testuser", "127.0.0.1", "Mozilla/5.0",
		ActionCreate, ResourceAccount, "acc-1", "Created account",
		nil, map[string]string{"name": "test"}, "success", "")

	logs := logger.GetLogs(10, 0, nil)
	require.Len(t, logs, 1)
	assert.Equal(t, ActionCreate, logs[0].Action)
	assert.Equal(t, ResourceAccount, logs[0].Resource)
}

func TestLogger_MaxLogs(t *testing.T) {
	logger := &Logger{
		logs:     make([]LogEntry, 0),
		filePath: "",
		maxLogs:  5,
		stopCh:   make(chan struct{}),
	}

	for i := 0; i < 10; i++ {
		logger.Log(LogEntry{
			UserID: "user-1",
			Action: ActionLogin,
			Status: "success",
		})
	}

	logs := logger.GetLogs(100, 0, nil)
	assert.Len(t, logs, 5)
}

func TestLogger_GetLogs_Pagination(t *testing.T) {
	logger := newTestLogger()

	for i := 0; i < 20; i++ {
		logger.Log(LogEntry{
			UserID: "user-1",
			Action: ActionLogin,
			Status: "success",
		})
	}

	page1 := logger.GetLogs(5, 0, nil)
	assert.Len(t, page1, 5)

	page2 := logger.GetLogs(5, 5, nil)
	assert.Len(t, page2, 5)

	page3 := logger.GetLogs(5, 100, nil)
	assert.Empty(t, page3)
}

func TestLogger_GetLogs_Filters(t *testing.T) {
	logger := newTestLogger()

	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogin, Status: "success"})
	logger.Log(LogEntry{UserID: "user-2", Action: ActionLogout, Status: "success"})
	logger.Log(LogEntry{UserID: "user-1", Action: ActionCreate, Status: "failed"})

	user1Logs := logger.GetLogs(10, 0, map[string]interface{}{"user_id": "user-1"})
	assert.Len(t, user1Logs, 2)

	loginLogs := logger.GetLogs(10, 0, map[string]interface{}{"action": ActionLogin})
	assert.Len(t, loginLogs, 1)

	successLogs := logger.GetLogs(10, 0, map[string]interface{}{"status": "success"})
	assert.Len(t, successLogs, 2)
}

func TestLogger_GetLogsByUser(t *testing.T) {
	logger := newTestLogger()

	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogin})
	logger.Log(LogEntry{UserID: "user-2", Action: ActionLogin})
	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogout})

	logs := logger.GetLogsByUser("user-1", 10)
	assert.Len(t, logs, 2)
}

func TestLogger_GetLogsByResource(t *testing.T) {
	logger := newTestLogger()

	logger.Log(LogEntry{Resource: ResourceAccount, Action: ActionCreate})
	logger.Log(LogEntry{Resource: ResourceProvider, Action: ActionUpdate})

	logs := logger.GetLogsByResource(ResourceAccount, 10)
	assert.Len(t, logs, 1)
}

func TestLogger_GetLogsByAction(t *testing.T) {
	logger := newTestLogger()

	logger.Log(LogEntry{Action: ActionLogin})
	logger.Log(LogEntry{Action: ActionLogout})
	logger.Log(LogEntry{Action: ActionLogin})

	logs := logger.GetLogsByAction(ActionLogin, 10)
	assert.Len(t, logs, 2)
}

func TestLogger_Count(t *testing.T) {
	logger := newTestLogger()

	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogin, Status: "success"})
	logger.Log(LogEntry{UserID: "user-2", Action: ActionLogin, Status: "failed"})
	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogout, Status: "success"})

	total := logger.Count(nil)
	assert.Equal(t, 3, total)

	user1Count := logger.Count(map[string]interface{}{"user_id": "user-1"})
	assert.Equal(t, 2, user1Count)

	successCount := logger.Count(map[string]interface{}{"status": "success"})
	assert.Equal(t, 2, successCount)
}

func TestLogger_SaveAndLoad(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	logger := &Logger{
		logs:     make([]LogEntry, 0),
		filePath: tmpFile.Name(),
		maxLogs:  100,
		stopCh:   make(chan struct{}),
	}

	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogin, Status: "success"})

	time.Sleep(50 * time.Millisecond)

	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var logs []LogEntry
	err = json.Unmarshal(data, &logs)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestLogger_Clear(t *testing.T) {
	logger := newTestLogger()

	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogin})
	logger.Clear()

	logs := logger.GetLogs(10, 0, nil)
	assert.Empty(t, logs)
}

func TestLogger_Close(t *testing.T) {
	logger := newTestLogger()
	logger.Log(LogEntry{UserID: "user-1", Action: ActionLogin})
	logger.Close()
}

func TestMatchFilters_TimeRange(t *testing.T) {
	logger := &Logger{}

	now := time.Now()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	entry := LogEntry{Timestamp: now}

	filters := map[string]interface{}{
		"start_time": past,
		"end_time":   future,
	}
	assert.True(t, logger.matchFilters(entry, filters))

	filters = map[string]interface{}{
		"start_time": future,
	}
	assert.False(t, logger.matchFilters(entry, filters))

	filters = map[string]interface{}{
		"end_time": past,
	}
	assert.False(t, logger.matchFilters(entry, filters))
}

func TestMatchFilters_ResourceID(t *testing.T) {
	logger := &Logger{}

	entry := LogEntry{ResourceID: "acc-1"}

	assert.True(t, logger.matchFilters(entry, map[string]interface{}{"resource_id": "acc-1"}))
	assert.False(t, logger.matchFilters(entry, map[string]interface{}{"resource_id": "acc-2"}))
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.Contains(t, id1, "-")
	assert.NotEqual(t, id1, id2)
}

func TestRandomString(t *testing.T) {
	s := randomString(16)
	assert.Len(t, s, 16)
}
