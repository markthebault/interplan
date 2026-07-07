package protocol

type SessionResponse struct {
	Session  SessionInfo `json:"session" toon:"session"`
	NextStep string      `json:"next_step,omitempty" toon:"next_step,omitempty"`
}

type SessionRequest struct {
	File   string `json:"file"`
	Reopen bool   `json:"reopen"`
}

type PollResponse struct {
	Session     PollSessionInfo `json:"session" toon:"session"`
	DOMSnapshot string          `json:"dom_snapshot,omitempty" toon:"dom_snapshot,omitempty"`
	Prompts     []Prompt        `json:"prompts,omitempty" toon:"prompts,omitempty"`
	Layout      []LayoutWarning `json:"layout_warnings,omitempty" toon:"layout_warnings,omitempty"`
	NextStep    string          `json:"next_step,omitempty" toon:"next_step,omitempty"`
}

type SessionListResponse struct {
	Sessions []SessionInfo `json:"sessions" toon:"sessions"`
	NextStep string        `json:"next_step" toon:"next_step"`
}

type SessionInfo struct {
	File   string `json:"file" toon:"file"`
	URL    string `json:"url,omitempty" toon:"url,omitempty"`
	Status string `json:"status" toon:"status"`
}

type PollSessionInfo struct {
	File         string `json:"file" toon:"file"`
	Status       string `json:"status" toon:"status"`
	SessionEnded bool   `json:"session_ended" toon:"session_ended"`
	EndedBy      string `json:"ended_by,omitempty" toon:"ended_by,omitempty"`
}

type Prompt struct {
	UID      string         `json:"uid,omitempty" toon:"uid,omitempty"`
	Tag      string         `json:"tag" toon:"tag"`
	Prompt   string         `json:"prompt" toon:"prompt"`
	Text     string         `json:"text,omitempty" toon:"text,omitempty"`
	Selector string         `json:"selector,omitempty" toon:"selector,omitempty"`
	QueueKey string         `json:"queue_key,omitempty" toon:"queue_key,omitempty"`
	Target   map[string]any `json:"target,omitempty" toon:"target,omitempty"`
	Value    any            `json:"value,omitempty" toon:"value,omitempty"`
}

type LayoutWarning struct {
	Key        string         `json:"key" toon:"key"`
	Kind       string         `json:"kind,omitempty" toon:"kind,omitempty"`
	Severity   string         `json:"severity,omitempty" toon:"severity,omitempty"`
	Message    string         `json:"message" toon:"message"`
	Selector   string         `json:"selector,omitempty" toon:"selector,omitempty"`
	OverflowPx int            `json:"overflow_px,omitempty" toon:"overflow_px,omitempty"`
	Viewport   map[string]any `json:"viewport,omitempty" toon:"viewport,omitempty"`
	Box        map[string]any `json:"box,omitempty" toon:"box,omitempty"`
	Persistent bool           `json:"persistent,omitempty" toon:"persistent,omitempty"`
}
