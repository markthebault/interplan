package server

import "testing"

func TestPromptKey(t *testing.T) {
	key, ok := promptKey("/api/abc123/prompts")
	if !ok || key != "abc123" {
		t.Fatalf("promptKey = %q, %v", key, ok)
	}
	if _, ok := promptKey("/api/abc123/nope"); ok {
		t.Fatal("unexpected match")
	}
}
