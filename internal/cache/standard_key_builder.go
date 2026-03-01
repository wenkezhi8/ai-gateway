package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// BuildStandardKey generates a stable key in form:
// intent:<intent>:k1=v1,k2=v2.
func BuildStandardKey(intent string, slots map[string]string) string {
	normalizedIntent := strings.TrimSpace(intent)
	if normalizedIntent == "" {
		normalizedIntent = "unknown"
	}

	if len(slots) == 0 {
		return fmt.Sprintf("intent:%s", normalizedIntent)
	}

	keys := make([]string, 0, len(slots))
	for k, v := range slots {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return fmt.Sprintf("intent:%s", normalizedIntent)
	}

	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.TrimSpace(slots[key])))
	}

	return fmt.Sprintf("intent:%s:%s", normalizedIntent, strings.Join(parts, ","))
}

// BuildTaskTypeStandardKey generates a stable key from task type + normalized query.
func BuildTaskTypeStandardKey(taskType, normalizedQuery string) string {
	normalizedTaskType := strings.ToLower(strings.TrimSpace(taskType))
	if normalizedTaskType == "" {
		normalizedTaskType = "unknown"
	}
	normalizedQuery = strings.TrimSpace(normalizedQuery)
	if normalizedQuery == "" {
		return BuildStandardKey(normalizedTaskType, map[string]string{
			"task_type": normalizedTaskType,
		})
	}

	hash := sha256.Sum256([]byte(normalizedQuery))
	queryHash := hex.EncodeToString(hash[:])

	return BuildStandardKey(normalizedTaskType, map[string]string{
		"task_type":  normalizedTaskType,
		"query_hash": queryHash,
	})
}
