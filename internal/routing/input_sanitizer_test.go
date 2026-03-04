package routing

import "testing"

func TestSanitizeIntentInput_RemovesBracketedMetadataTokens(t *testing.T) {
	raw := "[2026-03-04T12:34:56Z] [request_id=req-1] [message_id=msg-9] hello world"

	sanitized := SanitizeIntentInput(raw)

	if sanitized != "hello world" {
		t.Fatalf("expected metadata removed, got %q", sanitized)
	}
}

func TestSanitizeIntentInput_PreservesNonMetadataBracketText(t *testing.T) {
	raw := "use [a+b] and [x/y] for this answer"

	sanitized := SanitizeIntentInput(raw)

	if sanitized != raw {
		t.Fatalf("expected non-metadata bracket text preserved, got %q", sanitized)
	}
}

func TestIsShortGreetingIntent_WithMetadataPrefix(t *testing.T) {
	raw := "[session_id=s-1] [conversation_id=c-9] 你好"

	if !IsShortGreetingIntent(raw) {
		t.Fatal("expected short greeting intent to be true")
	}
}

func TestIsShortGreetingIntent_NonGreetingQuery(t *testing.T) {
	raw := "[request_id=req-1] hello, can you optimize this Go function?"

	if IsShortGreetingIntent(raw) {
		t.Fatal("expected non-greeting query to be false")
	}
}
