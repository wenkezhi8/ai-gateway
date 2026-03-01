package cache

import "testing"

func TestTextNormalizer_RemoveFillerAndWhitespace(t *testing.T) {
	n := NewTextNormalizer()
	got := n.Normalize("  请问  帮我  算一下  1 加 1 呀  ")
	if got != "算一下 1 加 1" {
		t.Fatalf("expected normalized text, got %q", got)
	}
}

func TestTextNormalizer_FullWidthAndNumber(t *testing.T) {
	n := NewTextNormalizer()
	got := n.Normalize("　请帮我算一＋二？？")
	if got != "算1+2?" {
		t.Fatalf("expected full-width and number normalization, got %q", got)
	}
}

func TestTextNormalizer_DeduplicatePunctuation(t *testing.T) {
	n := NewTextNormalizer()
	got := n.Normalize("hello！！！？？")
	if got != "hello!?" {
		t.Fatalf("expected deduplicated punctuation, got %q", got)
	}
}
