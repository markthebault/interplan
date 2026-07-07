package server

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileWatcher struct {
	mu        sync.RWMutex
	watchers  map[string][]chan struct{}
	modTimes  map[string]time.Time
	stopChan  chan struct{}
	debugMode bool
}

func NewFileWatcher(debugMode bool) *FileWatcher {
	w := &FileWatcher{
		watchers:  make(map[string][]chan struct{}),
		modTimes:  make(map[string]time.Time),
		stopChan:  make(chan struct{}),
		debugMode: debugMode,
	}
	go w.watchLoop()
	return w
}

func (w *FileWatcher) Watch(path string) <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Initialize mod time if not tracking yet
	if _, exists := w.modTimes[path]; !exists {
		if info, err := os.Stat(path); err == nil {
			w.modTimes[path] = info.ModTime()
		}
	}

	ch := make(chan struct{}, 1)
	w.watchers[path] = append(w.watchers[path], ch)
	
	if w.debugMode {
		log.Printf("[watcher] added watch for %s (total watchers: %d)", path, len(w.watchers[path]))
	}
	
	return ch
}

func (w *FileWatcher) Unwatch(path string, ch <-chan struct{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	channels := w.watchers[path]
	for i, c := range channels {
		if c == ch {
			w.watchers[path] = append(channels[:i], channels[i+1:]...)
			close(c)
			break
		}
	}

	if len(w.watchers[path]) == 0 {
		delete(w.watchers, path)
		delete(w.modTimes, path)
		if w.debugMode {
			log.Printf("[watcher] removed all watches for %s", path)
		}
	}
}

func (w *FileWatcher) watchLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.checkFiles()
		}
	}
}

func (w *FileWatcher) checkFiles() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for path, lastMod := range w.modTimes {
		info, err := os.Stat(path)
		if err != nil {
			if w.debugMode {
				log.Printf("[watcher] stat error for %s: %v", path, err)
			}
			continue
		}

		currentMod := info.ModTime()
		if currentMod.After(lastMod) {
			w.modTimes[path] = currentMod
			if w.debugMode {
				log.Printf("[watcher] detected change in %s", filepath.Base(path))
			}
			
			// Notify all watchers
			for _, ch := range w.watchers[path] {
				select {
				case ch <- struct{}{}:
				default:
					// Channel already has pending notification
				}
			}
		}
	}
}

func (w *FileWatcher) Stop() {
	close(w.stopChan)
}
