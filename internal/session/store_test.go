package session

import (
	"path/filepath"
	"testing"

	"github.com/markthebault/interplan/internal/protocol"
)

func TestStoreOpenPollAndUserEndedGate(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state.json"))
	file := "/tmp/doc.html"
	if _, err := store.Open(file, URLFor(Key(file), 37917), false); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := store.AddPrompts(Key(file), PromptPost{
		Prompts:     []protocol.Prompt{{Tag: "message", Prompt: "Change the title."}},
		DOMSnapshot: "h1 Draft",
		EndSession:  true,
	}); err != nil {
		t.Fatalf("AddPrompts: %v", err)
	}
	poll, err := store.Poll(file)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if poll.Session.Status != "feedback" || !poll.Session.SessionEnded || poll.Session.EndedBy != "user" {
		t.Fatalf("Poll session = %+v", poll.Session)
	}
	if len(poll.Prompts) != 1 || poll.Prompts[0].Prompt != "Change the title." {
		t.Fatalf("Prompts = %+v", poll.Prompts)
	}
	nextPoll, err := store.Poll(file)
	if err != nil {
		t.Fatalf("second Poll: %v", err)
	}
	if nextPoll.Session.Status != "ended" || !nextPoll.Session.SessionEnded || nextPoll.Session.EndedBy != "user" {
		t.Fatalf("second poll session = %+v", nextPoll.Session)
	}
	if _, err := store.Open(file, URLFor(Key(file), 37917), false); err == nil {
		t.Fatal("Open without --reopen succeeded after user-ended session")
	}
	if _, err := store.Open(file, URLFor(Key(file), 37917), true); err != nil {
		t.Fatalf("Open with --reopen: %v", err)
	}
}

func TestAgentEndedSessionCanReopenNormally(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state.json"))
	file := "/tmp/doc.html"
	if _, err := store.Open(file, URLFor(Key(file), 37917), false); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := store.End(file, "agent"); err != nil {
		t.Fatalf("End: %v", err)
	}
	reopened, err := store.Open(file, URLFor(Key(file), 37917), false)
	if err != nil {
		t.Fatalf("agent-ended session should reopen normally: %v", err)
	}
	if reopened.EndedBy != "" || reopened.Status != "open" {
		t.Fatalf("reopened = %+v", reopened)
	}
}

func TestStructuredPromptAndLayoutWarningFieldsSurvivePoll(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state.json"))
	file := "/tmp/doc.html"
	key := Key(file)
	if _, err := store.Open(file, URLFor(key, 37917), false); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := store.AddPrompts(key, PromptPost{
		Prompts: []protocol.Prompt{{
			UID:      "p_mermaid_1",
			Tag:      "mermaid-node",
			Prompt:   "Mention retry behavior.",
			QueueKey: "checkout-flow-PaymentFailed",
			Target: map[string]any{
				"kind":       "mermaid-node",
				"diagram_id": "checkout-flow",
				"node_id":    "PaymentFailed",
			},
			Value: map[string]any{"selected": true},
		}},
	}); err != nil {
		t.Fatalf("AddPrompts: %v", err)
	}
	if _, err := store.AddLayoutWarnings(key, LayoutWarningPost{
		Warnings: []protocol.LayoutWarning{{
			Key:        "overflow-main",
			Kind:       "horizontal-overflow",
			Severity:   "error",
			Message:    "Document overflows horizontally.",
			Selector:   "main",
			OverflowPx: 24,
			Viewport:   map[string]any{"width": 390},
			Persistent: true,
		}},
	}); err != nil {
		t.Fatalf("AddLayoutWarnings: %v", err)
	}
	poll, err := store.Poll(file)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if got := poll.Prompts[0].Target["node_id"]; got != "PaymentFailed" {
		t.Fatalf("node_id = %v", got)
	}
	if got := poll.Layout[0].OverflowPx; got != 24 {
		t.Fatalf("overflow = %d", got)
	}
}
