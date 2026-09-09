// Package a2aagent provides the A2A (Agent-to-Agent) server implementation
// for AgentChat. It wraps the chat hub and exposes it as an A2A-compliant agent.
package a2aagent

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"net/http"

	a2a "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/biodoia/agentchat/internal/chat"
	"github.com/biodoia/agentchat/pkg/types"
)

// ChatExecutor implements a2asrv.AgentExecutor for the group chat agent.
type ChatExecutor struct {
	hub *chat.Hub
	log *slog.Logger
}

// NewChatExecutor creates a new A2A chat executor.
func NewChatExecutor(hub *chat.Hub, log *slog.Logger) *ChatExecutor {
	return &ChatExecutor{hub: hub, log: log}
}

// Execute handles an incoming A2A message and broadcasts it to the chat hub.
func (e *ChatExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		msg := execCtx.Message
		if msg == nil {
			yield(nil, fmt.Errorf("no message in request"))
			return
		}

		// Extract text from parts using the Text() helper
		var body string
		for _, part := range msg.Parts {
			if t := part.Text(); t != "" {
				body = t
				break
			}
		}
		if body == "" {
			yield(nil, fmt.Errorf("no text part in message"))
			return
		}

		// Extract metadata
		sender := extractString(msg.Metadata, "sender", "unknown")
		group := extractString(msg.Metadata, "group", "general")
		priority := types.PriorityNormal
		if extractString(msg.Metadata, "priority", "") == "steal-focus" {
			priority = types.PriorityStealFocus
		}

		// Send to chat hub
		chatMsg := e.hub.SendMessage(group, sender, body, priority)
		e.log.Info("A2A message", "sender", sender, "group", group, "body", truncate(body, 60))

		// Build response
		respData, _ := json.Marshal(map[string]string{
			"status":    "sent",
			"messageId": chatMsg.ID,
			"group":     chatMsg.Group,
		})

		responseMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(string(respData)))
		yield(responseMsg, nil)
	}
}

// Cancel handles task cancellation.
func (e *ChatExecutor) Cancel(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

// SetupA2AServer creates A2A HTTP handlers and registers them on the mux.
func SetupA2AServer(mux *http.ServeMux, hub *chat.Hub, addr string, log *slog.Logger) {
	executor := NewChatExecutor(hub, log)
	handler := a2asrv.NewHandler(executor)

	jsonrpcHandler := a2asrv.NewJSONRPCHandler(handler)
	mux.Handle("/a2a/", jsonrpcHandler)

	// Agent Card endpoint
	card := buildAgentCard()
	cardData, _ := json.Marshal(card)
	mux.HandleFunc("/.well-known/agent.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cardData)
	})

	log.Info("A2A server registered", "addr", addr)
}

func buildAgentCard() *a2a.AgentCard {
	return &a2a.AgentCard{
		Name:        "AgentChat Relay",
		Description: "Multi-agent group chat server. Agents discover each other and communicate in real-time via chat groups.",
		Version:     "0.1.0",
		Capabilities: a2a.AgentCapabilities{
			Streaming:         true,
			PushNotifications: true,
		},
		Skills: []a2a.AgentSkill{
			{
				ID:          "group-chat",
				Name:        "Group Chat",
				Description: "Join groups, send messages, receive real-time updates. Multi-agent coordination via chat.",
				Examples: []string{
					"Join the 'general' group and say hello",
					"Send a steal-focus message to coordinate urgent work",
				},
			},
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}
}

func extractString(m map[string]any, key, fallback string) string {
	if m == nil {
		return fallback
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return fallback
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
