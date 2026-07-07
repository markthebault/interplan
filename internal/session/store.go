package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/markthebault/interplan/internal/protocol"
)

type Store struct {
	path string
	mu   sync.Mutex
}

type UserEndedError struct {
	Session *Session
}

func (e UserEndedError) Error() string {
	return "session was ended by user; pass --reopen to open it again"
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) Open(file, url string, reopen bool) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	key := Key(file)
	existing := state.Sessions[key]
	if existing != nil && existing.Status == "ended" && existing.EndedBy == "user" && !reopen {
		return nil, UserEndedError{Session: existing}
	}
	now := time.Now().UTC()
	if existing == nil {
		existing = &Session{Key: key, File: file, URL: url, Status: "open", UpdatedAt: now}
		state.Sessions[key] = existing
	} else {
		existing.File = file
		existing.URL = url
		existing.Status = "open"
		existing.EndedBy = ""
		existing.UpdatedAt = now
	}
	return existing, s.saveLocked(state)
}

func (s *Store) AddPrompts(key string, post PromptPost) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sess := state.Sessions[key]
	if sess == nil {
		return nil, os.ErrNotExist
	}
	sess.Prompts = append(sess.Prompts, post.Prompts...)
	sess.PendingPrompts = len(sess.Prompts)
	sess.DOMSnapshot = post.DOMSnapshot
	if post.EndSession {
		sess.Status = "ended"
		sess.EndedBy = "user"
	}
	sess.UpdatedAt = time.Now().UTC()
	return sess, s.saveLocked(state)
}

func (s *Store) AddLayoutWarnings(key string, post LayoutWarningPost) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sess := state.Sessions[key]
	if sess == nil {
		return nil, os.ErrNotExist
	}
	sess.LayoutWarnings = append(sess.LayoutWarnings, post.Warnings...)
	sess.UpdatedAt = time.Now().UTC()
	return sess, s.saveLocked(state)
}

func (s *Store) Poll(file string) (protocol.PollResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return protocol.PollResponse{}, err
	}
	key := Key(file)
	sess := state.Sessions[key]
	if sess == nil {
		return protocol.PollResponse{}, os.ErrNotExist
	}
	status := "waiting"
	next := "No feedback yet. Poll again after the user sends comments."
	if len(sess.Prompts) > 0 || len(sess.LayoutWarnings) > 0 {
		status = "feedback"
		next = "Apply feedback, then poll again."
	} else if sess.Status == "ended" {
		status = "ended"
		next = "Session ended. Stop polling."
	}
	if sess.Status == "ended" && sess.EndedBy == "user" {
		next = "Apply final feedback, stop polling, do not reopen."
		if len(sess.Prompts) == 0 && len(sess.LayoutWarnings) == 0 {
			next = "Session ended by user. Stop polling."
		}
	}
	out := protocol.PollResponse{
		Session: protocol.PollSessionInfo{
			File:         sess.File,
			Status:       status,
			SessionEnded: sess.Status == "ended",
			EndedBy:      sess.EndedBy,
		},
		DOMSnapshot: sess.DOMSnapshot,
		Prompts:     append([]protocol.Prompt(nil), sess.Prompts...),
		Layout:      append([]protocol.LayoutWarning(nil), sess.LayoutWarnings...),
		NextStep:    next,
	}
	sess.Prompts = nil
	sess.LayoutWarnings = nil
	sess.PendingPrompts = 0
	sess.UpdatedAt = time.Now().UTC()
	return out, s.saveLocked(state)
}

func (s *Store) GetByKey(key string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sess := state.Sessions[key]
	if sess == nil {
		return nil, os.ErrNotExist
	}
	cp := *sess
	cp.Prompts = append([]protocol.Prompt(nil), sess.Prompts...)
	cp.LayoutWarnings = append([]protocol.LayoutWarning(nil), sess.LayoutWarnings...)
	cp.Chat = append([]ChatMessage(nil), sess.Chat...)
	return &cp, nil
}

func (s *Store) End(file, endedBy string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	key := Key(file)
	sess := state.Sessions[key]
	if sess == nil {
		return nil, os.ErrNotExist
	}
	sess.Status = "ended"
	sess.EndedBy = endedBy
	sess.UpdatedAt = time.Now().UTC()
	return sess, s.saveLocked(state)
}

func (s *Store) AppendAgentReply(file, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return err
	}
	key := Key(file)
	sess := state.Sessions[key]
	if sess == nil {
		return os.ErrNotExist
	}
	sess.Chat = append(sess.Chat, ChatMessage{Role: "agent", Message: message, At: time.Now().UTC()})
	sess.UpdatedAt = time.Now().UTC()
	return s.saveLocked(state)
}

func (s *Store) loadLocked() (State, error) {
	state := State{Sessions: map[string]*Session{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Sessions == nil {
		state.Sessions = map[string]*Session{}
	}
	return state, nil
}

func (s *Store) saveLocked(state State) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
