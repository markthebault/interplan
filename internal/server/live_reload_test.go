package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWatcherDetectsChanges(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.html")
	if err := os.WriteFile(file, []byte("<!doctype html><h1>v1</h1>"), 0o600); err != nil {
		t.Fatal(err)
	}

	watcher := NewFileWatcher(false)
	defer watcher.Stop()

	// Start watching
	changeChan := watcher.Watch(file)
	defer watcher.Unwatch(file, changeChan)

	// Wait a bit to ensure initial mod time is recorded
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(file, []byte("<!doctype html><h1>v2</h1>"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Should receive notification within reasonable time
	select {
	case <-changeChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive change notification within 2 seconds")
	}
}
