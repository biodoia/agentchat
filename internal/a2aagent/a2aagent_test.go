package a2aagent

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	a2a "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/biodoia/agentchat/internal/chat"
)

func newTestExecutor(t *testing.T) *ChatExecutor {
	t.Helper()
	hub := chat.NewHub()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewChatExecutor(hub, log)
}

func TestExecuteSendsMessage(t *testing.T) {
	exec := newTestExecutor(t)

	// Build an A2A message with text part and metadata
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("hello from A2A"))
	msg.Metadata = map[string]any{
		"sender": "TestBot",
		"group":  "general",
	}

	execCtx := &a2asrv.ExecutorContext{
		Message:   msg,
		TaskID:    "task-1",
		ContextID: "ctx-1",
	}

	// Execute and collect events
	var events []a2a.Event
	var errs []error
	for event, err := range exec.Execute(context.Background(), execCtx) {
		if err != nil {
			errs = append(errs, err)
			break
		}
		events = append(events, event)
	}

	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (response message), got %d", len(events))
	}

	// Verify message was sent to hub
	msgs := exec.hub.GetMessages("general", time.Time{}, 10)
	if len(msgs) != 1 {
		t.Errorf("expected 1 message in hub, got %d", len(msgs))
	}
	if len(msgs) > 0 && msgs[0].Body != "hello from A2A" {
		t.Errorf("expected 'hello from A2A', got %s", msgs[0].Body)
	}
}

func TestExecuteNoMessage(t *testing.T) {
	exec := newTestExecutor(t)
	execCtx := &a2asrv.ExecutorContext{
		Message: nil,
		TaskID:  "task-nil",
	}

	var gotErr bool
	for _, err := range exec.Execute(context.Background(), execCtx) {
		if err != nil {
			gotErr = true
		}
	}
	if !gotErr {
		t.Error("expected error for nil message")
	}
}

func TestExecuteNoTextPart(t *testing.T) {
	exec := newTestExecutor(t)
	msg := &a2a.Message{
		ID:   "msg-no-text",
		Role: a2a.MessageRoleUser,
		Parts: a2a.ContentParts{
			{Content: a2a.Raw([]byte("not text"))},
		},
	}
	execCtx := &a2asrv.ExecutorContext{
		Message: msg,
		TaskID:  "task-no-text",
	}

	var gotErr bool
	for _, err := range exec.Execute(context.Background(), execCtx) {
		if err != nil {
			gotErr = true
		}
	}
	if !gotErr {
		t.Error("expected error for no text part")
	}
}

func TestExecuteExtractsMetadata(t *testing.T) {
	exec := newTestExecutor(t)
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("urgent!"))
	msg.Metadata = map[string]any{
		"sender":   "UrgentBot",
		"group":    "alerts",
		"priority": "steal-focus",
	}

	execCtx := &a2asrv.ExecutorContext{
		Message: msg,
		TaskID:  "task-urgent",
	}

	var events []a2a.Event
	for event, err := range exec.Execute(context.Background(), execCtx) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, event)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestExecuteDefaultGroup(t *testing.T) {
	exec := newTestExecutor(t)
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart("no group"))
	msg.Metadata = map[string]any{
		"sender": "Bot",
	}
	// No "group" in metadata → should default to "general"

	execCtx := &a2asrv.ExecutorContext{
		Message: msg,
		TaskID:  "task-default",
	}

	var count int
	for event, err := range exec.Execute(context.Background(), execCtx) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = event
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
}

func TestCancel(t *testing.T) {
	exec := newTestExecutor(t)
	execCtx := &a2asrv.ExecutorContext{
		TaskID:    "task-cancel",
		ContextID: "ctx-cancel",
	}

	var events []a2a.Event
	for event, err := range exec.Cancel(context.Background(), execCtx) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, event)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 cancel event, got %d", len(events))
	}
}

func TestBuildAgentCard(t *testing.T) {
	card := buildAgentCard()
	if card.Name != "AgentChat Relay" {
		t.Errorf("expected AgentChat Relay, got %s", card.Name)
	}
	if len(card.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(card.Skills))
	}
	if card.Skills[0].ID != "group-chat" {
		t.Errorf("expected skill id 'group-chat', got %s", card.Skills[0].ID)
	}
	if !card.Capabilities.Streaming {
		t.Error("expected streaming=true")
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		fallback string
		want     string
	}{
		{nil, "x", "default", "default"},
		{map[string]any{}, "x", "default", "default"},
		{map[string]any{"x": "val"}, "x", "default", "val"},
		{map[string]any{"x": 42}, "x", "default", "default"}, // not a string
	}
	for _, tt := range tests {
		got := extractString(tt.m, tt.key, tt.fallback)
		if got != tt.want {
			t.Errorf("extractString(%v, %q, %q) = %q, want %q", tt.m, tt.key, tt.fallback, got, tt.want)
		}
	}
}
