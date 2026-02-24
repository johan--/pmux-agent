// Package protocol defines wire protocol message types matching @pocketmux/shared.
//
// All struct field names use msgpack tags matching the camelCase names from the
// TypeScript protocol types. This ensures cross-language compatibility.
package protocol

// Message is implemented by all protocol messages.
type Message interface {
	MessageType() string
}

// --- Mobile → Agent (Requests) ---

// ListSessionsRequest requests the full session tree from the agent.
type ListSessionsRequest struct {
	Type string `msgpack:"type"`
}

func (m *ListSessionsRequest) MessageType() string { return "list_sessions" }

// AttachRequest attaches to a specific pane with the given terminal dimensions.
type AttachRequest struct {
	Type   string `msgpack:"type"`
	PaneID string `msgpack:"paneId"`
	Cols   int    `msgpack:"cols"`
	Rows   int    `msgpack:"rows"`
}

func (m *AttachRequest) MessageType() string { return "attach" }

// DetachRequest detaches from the currently attached pane.
type DetachRequest struct {
	Type string `msgpack:"type"`
}

func (m *DetachRequest) MessageType() string { return "detach" }

// InputRequest sends terminal input data (raw bytes) to the attached pane.
type InputRequest struct {
	Type string `msgpack:"type"`
	Data []byte `msgpack:"data"`
}

func (m *InputRequest) MessageType() string { return "input" }

// ResizeRequest resizes the attached pane to the given dimensions.
type ResizeRequest struct {
	Type string `msgpack:"type"`
	Cols int    `msgpack:"cols"`
	Rows int    `msgpack:"rows"`
}

func (m *ResizeRequest) MessageType() string { return "resize" }

// CreateSessionRequest creates a new tmux session.
type CreateSessionRequest struct {
	Type    string  `msgpack:"type"`
	Name    *string `msgpack:"name,omitempty"`
	Command *string `msgpack:"command,omitempty"`
}

func (m *CreateSessionRequest) MessageType() string { return "create_session" }

// KillSessionRequest kills a tmux session by ID.
type KillSessionRequest struct {
	Type    string `msgpack:"type"`
	Session string `msgpack:"session"`
}

func (m *KillSessionRequest) MessageType() string { return "kill_session" }

// PingRequest is a latency measurement request.
type PingRequest struct {
	Type string `msgpack:"type"`
}

func (m *PingRequest) MessageType() string { return "ping" }

// --- Agent → Mobile (Events) ---

// SessionsEvent returns the full session tree.
type SessionsEvent struct {
	Type     string        `msgpack:"type"`
	Sessions []TmuxSession `msgpack:"sessions"`
}

func (m *SessionsEvent) MessageType() string { return "sessions" }

// OutputEvent sends terminal output data (raw bytes) from the attached pane.
type OutputEvent struct {
	Type string `msgpack:"type"`
	Data []byte `msgpack:"data"`
}

func (m *OutputEvent) MessageType() string { return "output" }

// AttachedEvent confirms successful pane attachment.
type AttachedEvent struct {
	Type   string `msgpack:"type"`
	PaneID string `msgpack:"paneId"`
}

func (m *AttachedEvent) MessageType() string { return "attached" }

// DetachedEvent confirms pane detachment.
type DetachedEvent struct {
	Type string `msgpack:"type"`
}

func (m *DetachedEvent) MessageType() string { return "detached" }

// SessionCreatedEvent confirms a new session was created.
type SessionCreatedEvent struct {
	Type    string `msgpack:"type"`
	Session string `msgpack:"session"`
	Name    string `msgpack:"name"`
}

func (m *SessionCreatedEvent) MessageType() string { return "session_created" }

// SessionEndedEvent reports a session was killed or exited.
type SessionEndedEvent struct {
	Type    string `msgpack:"type"`
	Session string `msgpack:"session"`
}

func (m *SessionEndedEvent) MessageType() string { return "session_ended" }

// ErrorEvent reports an error to the mobile client.
type ErrorEvent struct {
	Type    string `msgpack:"type"`
	Code    string `msgpack:"code"`
	Message string `msgpack:"message"`
}

func (m *ErrorEvent) MessageType() string { return "error" }

// PongEvent responds to a ping with latency measurement.
type PongEvent struct {
	Type    string `msgpack:"type"`
	Latency int    `msgpack:"latency"`
}

func (m *PongEvent) MessageType() string { return "pong" }

// --- tmux Data Types ---

// TmuxSession represents a tmux session with its window tree.
type TmuxSession struct {
	ID           string       `msgpack:"id"`
	Name         string       `msgpack:"name"`
	Created      int64        `msgpack:"created"`
	Windows      []TmuxWindow `msgpack:"windows"`
	LastActivity int64        `msgpack:"lastActivity"`
	Attached     bool         `msgpack:"attached"`
}

// TmuxWindow represents a window within a tmux session.
type TmuxWindow struct {
	ID     string     `msgpack:"id"`
	Name   string     `msgpack:"name"`
	Index  int        `msgpack:"index"`
	Active bool       `msgpack:"active"`
	Panes  []TmuxPane `msgpack:"panes"`
}

// TmuxPane represents a pane within a tmux window.
type TmuxPane struct {
	ID             string   `msgpack:"id"`
	Index          int      `msgpack:"index"`
	Active         bool     `msgpack:"active"`
	Size           PaneSize `msgpack:"size"`
	Title          string   `msgpack:"title"`
	CurrentCommand string   `msgpack:"currentCommand"`
}

// PaneSize holds terminal dimensions.
type PaneSize struct {
	Cols int `msgpack:"cols"`
	Rows int `msgpack:"rows"`
}
