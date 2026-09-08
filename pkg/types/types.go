// Package types defines the shared data types for AgentChat.
package types

import "time"

// Priority controls how a message interrupts recipients.
type Priority string

const (
	PriorityNormal     Priority = "normal"
	PriorityStealFocus Priority = "steal-focus"
)

// Message is one chat message in a group.
type Message struct {
	ID        string    `json:"id"`
	Group     string    `json:"group"`
	Sender    string    `json:"sender"`
	Color     string    `json:"color"`
	Avatar    string    `json:"avatar"`
	Body      string    `json:"body"`
	TS        time.Time `json:"ts"`
	Priority  Priority  `json:"priority"`
}

// AgentProfile is a registered agent in the chat.
type AgentProfile struct {
	Name   string `json:"name"`
	Color  string `json:"color"`
	Avatar string `json:"avatar"`
	Program string `json:"program"`
}

// Group is a named chat room.
type Group struct {
	Name      string    `json:"name"`
	Members   []string  `json:"members"`
	CreatedAt time.Time `json:"created_at"`
}

// Known agent color/avatar assignments.
var DefaultAgents = map[string]AgentProfile{
	"MiMoCode":  {Name: "MiMoCode", Color: "#00FF88", Avatar: "🟢", Program: "mimocode"},
	"Claude":    {Name: "Claude", Color: "#6B8AFF", Avatar: "🔵", Program: "claude-code"},
	"Codex":     {Name: "Codex", Color: "#FF6B6B", Avatar: "🔴", Program: "codex"},
	"Kiro":      {Name: "Kiro", Color: "#FFB347", Avatar: "🟠", Program: "kiro"},
	"Grok":      {Name: "Grok", Color: "#DDA0DD", Avatar: "🟣", Program: "grok"},
	"Gemini":    {Name: "Gemini", Color: "#87CEEB", Avatar: "🔷", Program: "gemini"},
	"Qwen":      {Name: "Qwen", Color: "#FFD700", Avatar: "🟡", Program: "qwen"},
	"Hermes":    {Name: "Hermes", Color: "#98FB98", Avatar: "🟩", Program: "hermes"},
}

// WSMessage is a WebSocket frame for real-time delivery.
type WSMessage struct {
	Type    string   `json:"type"` // "message", "join", "leave", "typing"
	Message *Message `json:"message,omitempty"`
	Agent   string   `json:"agent,omitempty"`
	Group   string   `json:"group,omitempty"`
}
