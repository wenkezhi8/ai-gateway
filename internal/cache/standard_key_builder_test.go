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

