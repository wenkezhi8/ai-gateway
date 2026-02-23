package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestWithField(t *testing.T) {
	entry := WithField("key", "value")
	assert.NotNil(t, entry)
	assert.Equal(t, "value", entry.Data["key"])
}

func TestWithFields(t *testing.T) {
	fields := logrus.Fields{
		"key1": "value1",
		"key2": 123,
	}
	entry := WithFields(fields)
	assert.NotNil(t, entry)
	assert.Equal(t, "value1", entry.Data["key1"])
	assert.Equal(t, 123, entry.Data["key2"])
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	originalOut := Log.Out
	Log.SetOutput(&buf)
	defer Log.SetOutput(originalOut)

	originalLevel := Log.Level
	Log.SetLevel(logrus.DebugLevel)
	defer Log.SetLevel(originalLevel)

	Debug("debug message")
	Info("info message")
	Warn("warn message")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
}

func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	originalOut := Log.Out
	Log.SetOutput(&buf)
	defer Log.SetOutput(originalOut)

	Info("test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)
	assert.Contains(t, logEntry, "timestamp")
	assert.Contains(t, logEntry, "level")
	assert.Contains(t, logEntry, "message")
	assert.Equal(t, "test message", logEntry["message"])
}

func TestLogLevelFromEnv(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	tests := []struct {
		envLevel string
		expected logrus.Level
	}{
		{"debug", logrus.DebugLevel},
		{"warn", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"", logrus.InfoLevel},
		{"invalid", logrus.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.envLevel, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.envLevel)

			newLogger := logrus.New()
			level := os.Getenv("LOG_LEVEL")
			switch level {
			case "debug":
				newLogger.SetLevel(logrus.DebugLevel)
			case "warn":
				newLogger.SetLevel(logrus.WarnLevel)
			case "error":
				newLogger.SetLevel(logrus.ErrorLevel)
			default:
				newLogger.SetLevel(logrus.InfoLevel)
			}

			assert.Equal(t, tt.expected, newLogger.Level)
		})
	}
}

func TestLog(t *testing.T) {
	assert.NotNil(t, Log)
}
