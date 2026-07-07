package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markthebault/interplan/internal/session"
)

func TestSection21MinimalProtocol(t *testing.T) {
	store := session.NewStore(filepath.Join(t.TempDir(), "state.json"))
	watcher := NewFileWatcher(false)
	defer watcher.Stop()
	file := "/tmp/doc.html"
	key := session.Key(file)
	if _, err := store.Open(file, session.URLFor(key, 37917), false); err != nil {
		t.Fatalf("Open: %v", err)
	}

	body := bytes.NewBufferString(`{
		"prompts": [{ "tag": "message", "prompt": "Change the title." }],
		"domSnapshot": "h1 Draft",
		"endSession": true
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/"+key+"/prompts", body)
	rec := httptest.NewRecorder()
	Handler(store, watcher).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/%s/prompts = %d, body %s", key, rec.Code, rec.Body.String())
	}

	poll, err := store.Poll(file)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if poll.Session.File != file || poll.Session.Status != "feedback" {
		t.Fatalf("session = %+v", poll.Session)
	}
	if !poll.Session.SessionEnded || poll.Session.EndedBy != "user" {
		t.Fatalf("ended state = %+v", poll.Session)
	}
	if poll.DOMSnapshot != "h1 Draft" {
		t.Fatalf("dom snapshot = %q", poll.DOMSnapshot)
	}
	if len(poll.Prompts) != 1 || poll.Prompts[0].Tag != "message" || poll.Prompts[0].Prompt != "Change the title." {
		t.Fatalf("prompts = %+v", poll.Prompts)
	}
	if poll.NextStep != "Apply final feedback, stop polling, do not reopen." {
		t.Fatalf("next step = %q", poll.NextStep)
	}
}

func TestSessionAndArtifactRoutesRender(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "doc.html")
	if err := os.WriteFile(file, []byte("<!doctype html><html><body><h1>Draft</h1></body></html>"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	store := session.NewStore(filepath.Join(t.TempDir(), "state.json"))
	watcher := NewFileWatcher(false)
	defer watcher.Stop()
	canonical, err := session.CanonicalPath(file)
	if err != nil {
		t.Fatalf("CanonicalPath: %v", err)
	}
	key := session.Key(canonical)
	if _, err := store.Open(canonical, session.URLFor(key, 37917), false); err != nil {
		t.Fatalf("Open: %v", err)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/session/"+key, nil)
	sessionRec := httptest.NewRecorder()
	Handler(store, watcher).ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK || !strings.Contains(sessionRec.Body.String(), "<iframe") {
		t.Fatalf("session route = %d, body %s", sessionRec.Code, sessionRec.Body.String())
	}
	if !strings.Contains(sessionRec.Body.String(), "id=\"annotate\"") || !strings.Contains(sessionRec.Body.String(), "queueAnnotation") {
		t.Fatalf("session route did not include annotation UI: %s", sessionRec.Body.String())
	}
	if !strings.Contains(sessionRec.Body.String(), "new EventSource") {
		t.Fatalf("session route did not include SSE connection: %s", sessionRec.Body.String())
	}

	artifactReq := httptest.NewRequest(http.MethodGet, "/artifact/"+key+"/index.html", nil)
	artifactRec := httptest.NewRecorder()
	Handler(store, watcher).ServeHTTP(artifactRec, artifactReq)
	if artifactRec.Code != http.StatusOK {
		t.Fatalf("artifact route = %d, body %s", artifactRec.Code, artifactRec.Body.String())
	}
	if !strings.Contains(artifactRec.Body.String(), "window.interplan") {
		t.Fatalf("artifact did not include SDK injection: %s", artifactRec.Body.String())
	}
	if !strings.Contains(sessionRec.Body.String(), "attachFrameAnnotation") || !strings.Contains(sessionRec.Body.String(), "selectorFor") || !strings.Contains(sessionRec.Body.String(), "pickAnnotationTarget") {
		t.Fatalf("session route did not include annotation capture code: %s", sessionRec.Body.String())
	}
}
