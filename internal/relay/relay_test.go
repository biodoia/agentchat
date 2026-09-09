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

func TestStatusEndpoint(t *testing.T) {
	s := newTestServer(t)
	s.SendMessage("general", "Alice", "test", types.PriorityNormal)

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["service"] != "agentchat-relay" {
		t.Errorf("expected agentchat-relay, got %v", resp["service"])
	}
	if resp["version"] != "0.2.0" {
		t.Errorf("expected 0.2.0, got %v", resp["version"])
	}
	if resp["totalMessages"].(float64) < 1 {
		t.Errorf("expected at least 1 message, got %v", resp["totalMessages"])
	}
	groups := resp["groups"].([]interface{})
	if len(groups) < 1 {
		t.Error("expected at least 1 group")
	}
}

func TestGetMessagesEmpty(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/messages?group=nonexistent", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJoinGroupInvalidBody(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/group/join", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestLeaveGroupInvalidBody(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/group/leave", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestRegisterAgentInvalidBody(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestSendMessageEmptyBody(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{
		"group":  "general",
		"sender": "Bot",
		"body":   "",
	})
	req := httptest.NewRequest("POST", "/api/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	// Empty body should still succeed (no validation on body content)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSendMessageCustomGroup(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{
		"group":  "dev-team",
		"sender": "Alice",
		"body":   "hello dev team",
	})
	req := httptest.NewRequest("POST", "/api/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var msg types.Message
	json.Unmarshal(w.Body.Bytes(), &msg)
	if msg.Group != "dev-team" {
		t.Errorf("expected group 'dev-team', got %s", msg.Group)
	}
}

func TestGetMessagesWithSince(t *testing.T) {
	s := newTestServer(t)
	s.SendMessage("general", "Alice", "old msg", types.PriorityNormal)

	req := httptest.NewRequest("GET", "/api/messages?group=general&since=2099-01-01T00:00:00Z", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var msgs []*types.Message
	json.Unmarshal(w.Body.Bytes(), &msgs)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages (future since), got %d", len(msgs))
	}
}

func TestJoinGroupCreatesGroup(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{
		"agent": "Alice",
		"group": "brand-new-group",
	})
	req := httptest.NewRequest("POST", "/api/group/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify group exists
	g := s.hub.GetGroup("brand-new-group")
	if g == nil {
		t.Fatal("expected group to be created")
	}
}

func TestStatusWithMultipleGroups(t *testing.T) {
	s := newTestServer(t)
	s.hub.JoinGroup("Alice", "general")
	s.hub.JoinGroup("Bob", "dev")
	s.SendMessage("general", "Alice", "msg1", types.PriorityNormal)
	s.SendMessage("general", "Bob", "msg2", types.PriorityNormal)
	s.SendMessage("dev", "Charlie", "msg3", types.PriorityNormal)

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	total := resp["totalMessages"].(float64)
	if total != 3 {
		t.Errorf("expected 3 total messages, got %v", total)
	}

	groups := resp["groups"].([]interface{})
	if len(groups) < 2 {
		t.Errorf("expected at least 2 groups, got %d", len(groups))
	}
}

func TestListGroupsContent(t *testing.T) {
	s := newTestServer(t)
	s.hub.JoinGroup("Alice", "alpha")
	s.hub.JoinGroup("Bob", "beta")

	req := httptest.NewRequest("GET", "/api/groups", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	var groups []*types.Group
	json.Unmarshal(w.Body.Bytes(), &groups)

	names := make(map[string]bool)
	for _, g := range groups {
		names[g.Name] = true
	}
	if !names["alpha"] || !names["beta"] || !names["general"] {
		t.Errorf("expected alpha, beta, general; got %v", names)
	}
}

func TestGetHub(t *testing.T) {
	s := newTestServer(t)
	hub := s.GetHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
}

func TestBroadcast(t *testing.T) {
	s := newTestServer(t)
	s.SendMessage("general", "Alice", "before", types.PriorityNormal)
	s.Broadcast("system broadcast")

	msgs := s.hub.GetMessages("general", time.Time{}, 10)
	last := msgs[len(msgs)-1]
	if last.Body != "system broadcast" {
		t.Errorf("expected 'system broadcast', got %s", last.Body)
	}
	if last.Sender != "system" {
		t.Errorf("expected sender 'system', got %s", last.Sender)
	}
}

func TestListGroupNames(t *testing.T) {
	s := newTestServer(t)
	s.hub.JoinGroup("Alice", "alpha")
	s.hub.JoinGroup("Bob", "beta")

	names := s.ListGroupNames()
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["general"] || !nameSet["alpha"] || !nameSet["beta"] {
		t.Errorf("expected general, alpha, beta; got %v", names)
	}
}

func TestSubscribeGroup(t *testing.T) {
	s := newTestServer(t)
	ch := s.SubscribeGroup("general")

	go func() {
		time.Sleep(10 * time.Millisecond)
		s.SendMessage("general", "Alice", "subscribed!", types.PriorityNormal)
	}()

	select {
	case msg := <-ch:
		if msg.Body != "subscribed!" {
			t.Errorf("expected 'subscribed!', got %s", msg.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestSendMessageViaHelper(t *testing.T) {
	s := newTestServer(t)
	msg := s.SendMessage("general", "Helper", "via helper", types.PriorityStealFocus)
	if msg.Body != "via helper" {
		t.Errorf("expected 'via helper', got %s", msg.Body)
	}
	if msg.Priority != types.PriorityStealFocus {
		t.Errorf("expected steal-focus, got %s", msg.Priority)
	}
}

func TestTruncate(t *testing.T) {
	short := truncate("hi", 10)
	if short != "hi" {
		t.Errorf("expected 'hi', got %s", short)
	}
	long := truncate("this is a very long string", 10)
	if len(long) > 13 { // 10 + "..."
		t.Errorf("expected truncation, got %s", long)
	}
}
