package session

import (
	"time"

	"github.com/markthebault/interplan/internal/protocol"
)

type State struct {
	Sessions map[string]*Session `json:"sessions"`
}

type Session struct {
	Key                        string                   `json:"key"`
	File                       string                   `json:"file"`
	URL                        string                   `json:"url"`
	Status                     string                   `json:"status"`
	EndedBy                    string                   `json:"ended_by"`
	PendingPrompts             int                      `json:"pending_prompts"`
	Prompts                    []protocol.Prompt        `json:"prompts"`
	LayoutWarnings             []protocol.LayoutWarning `json:"layout_warnings"`
	DeliveredLayoutWarningKeys []string                 `json:"delivered_layout_warning_keys"`
	DOMSnapshot                string                   `json:"dom_snapshot"`
	Chat                       []ChatMessage            `json:"chat"`
	UpdatedAt                  time.Time                `json:"updated_at"`
}

type ChatMessage struct {
	Role    string    `json:"role"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

type PromptPost struct {
	Prompts     []protocol.Prompt `json:"prompts"`
	DOMSnapshot string            `json:"domSnapshot"`
	EndSession  bool              `json:"endSession"`
}

type LayoutWarningPost struct {
	Warnings []protocol.LayoutWarning `json:"warnings"`
}
