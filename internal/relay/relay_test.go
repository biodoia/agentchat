package relay

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/biodoia/agentchat/pkg/types"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return New("127.0.0.1:0", "", log) // no data dir, in-memory only
}

func TestHealth(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestSendMessage(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{
		"group":  "general",
		"sender": "TestBot",
		"body":   "hello world",
	})
	req := httptest.NewRequest("POST", "/api/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var msg types.Message
	json.Unmarshal(w.Body.Bytes(), &msg)
	if msg.Body != "hello world" {
		t.Errorf("expected 'hello world', got %s", msg.Body)
	}
	if msg.Sender != "TestBot" {
		t.Errorf("expected sender TestBot, got %s", msg.Sender)
	}
}

func TestSendMessageDefaultGroup(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{
		"sender": "Alice",
		"body":   "no group specified",
	})
	req := httptest.NewRequest("POST", "/api/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var msg types.Message
	json.Unmarshal(w.Body.Bytes(), &msg)
	if msg.Group != "general" {
		t.Errorf("expected default group 'general', got %s", msg.Group)
	}
}

func TestGetMessages(t *testing.T) {
	s := newTestServer(t)
	// Send a message first
	s.SendMessage("general", "Bob", "test msg", types.PriorityNormal)

	req := httptest.NewRequest("GET", "/api/messages?group=general", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var msgs []*types.Message
	json.Unmarshal(w.Body.Bytes(), &msgs)
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

func TestJoinGroup(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{
		"agent": "Charlie",
		"group": "dev",
	})
	req := httptest.NewRequest("POST", "/api/group/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// Verify group exists and has member
	g := s.hub.GetGroup("dev")
	if g == nil {
		t.Fatal("expected 'dev' group to be created")
	}
	found := false
	for _, m := range g.Members {
		if m == "Charlie" {
			found = true
		}
	}
	if !found {
		t.Error("expected Charlie in dev group")
	}
}

func TestLeaveGroup(t *testing.T) {
	s := newTestServer(t)
	s.hub.JoinGroup("Alice", "general")

	body, _ := json.Marshal(map[string]string{
		"agent": "Alice",
		"group": "general",
	})
	req := httptest.NewRequest("POST", "/api/group/leave", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	g := s.hub.GetGroup("general")
	for _, m := range g.Members {
		if m == "Alice" {
			t.Error("Alice should have left the group")
		}
	}
}

func TestListGroups(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/groups", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var groups []*types.Group
	json.Unmarshal(w.Body.Bytes(), &groups)
	if len(groups) < 1 {
		t.Error("expected at least 1 group (general)")
	}
}

func TestRegisterAgent(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(types.AgentProfile{
		Name:   "NewBot",
		Color:  "#FF00FF",
		Avatar: "🤖",
	})
	req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAgentCard(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/.well-known/agent.json", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var card map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &card)
	if card["name"] != "AgentChat Relay" {
		t.Errorf("expected AgentChat Relay, got %v", card["name"])
	}
}

func TestSendMessageInvalid(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/message", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestSendMessageStealFocus(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{
		"group":    "general",
		"sender":   "Urgent",
		"body":     "ATTENTION!",
		"priority": "steal-focus",
	})
	req := httptest.NewRequest("POST", "/api/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var msg types.Message
	json.Unmarshal(w.Body.Bytes(), &msg)
	if msg.Priority != types.PriorityStealFocus {
		t.Errorf("expected steal-focus, got %s", msg.Priority)
	}
}

// Test that messages are broadcast through WebSocket subscribers
func TestWSBroadcast(t *testing.T) {
	s := newTestServer(t)
	ch := s.hub.Subscribe("general")

	go func() {
		time.Sleep(10 * time.Millisecond)
		s.SendMessage("general", "Alice", "broadcast test", types.PriorityNormal)
	}()

	select {
	case msg := <-ch:
		if msg.Body != "broadcast test" {
			t.Errorf("expected 'broadcast test', got %s", msg.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}
