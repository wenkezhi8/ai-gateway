package cache

import "testing"

func TestBuildStandardKey_SortSlots(t *testing.T) {
	key := BuildStandardKey("translate", map[string]string{
		"text": "hello",
		"to":   "zh",
		"from": "en",
	})

	expected := "intent:translate:from=en,text=hello,to=zh"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}

func TestBuildStandardKey_EmptySlots(t *testing.T) {
	key := BuildStandardKey("chat", map[string]string{})
	if key != "intent:chat" {
		t.Fatalf("expected intent-only key, got %q", key)
	}
}

func TestBuildStandardKey_IgnoreEmptyValues(t *testing.T) {
	key := BuildStandardKey("calc", map[string]string{
		"expr": "1+1",
		"unit": "",
	})
	if key != "intent:calc:expr=1+1" {
		t.Fatalf("expected empty values to be ignored, got %q", key)
	}
}

func TestBuildTaskTypeStandardKey_StableForSameInput(t *testing.T) {
	taskType := "qa"
	normalizedQuery := "什么是缓存"

	first := BuildTaskTypeStandardKey(taskType, normalizedQuery)
	second := BuildTaskTypeStandardKey(taskType, normalizedQuery)

	if first == "" || second == "" {
		t.Fatalf("expected non-empty standard key, got first=%q second=%q", first, second)
	}
	if first != second {
		t.Fatalf("expected deterministic standard key, got first=%q second=%q", first, second)
	}
}

func TestBuildTaskTypeStandardKey_DifferentQueryShouldDiffer(t *testing.T) {
	left := BuildTaskTypeStandardKey("qa", "缓存是什么")
	right := BuildTaskTypeStandardKey("qa", "缓存原理")
	if left == right {
		t.Fatalf("expected different key for different query, got %q", left)
	}
}
