package chat

import (
	"testing"
	"time"

	"github.com/biodoia/agentchat/pkg/types"
)

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	// Should have default "general" group
	groups := h.GetGroups()
	if len(groups) != 1 || groups[0].Name != "general" {
		t.Errorf("expected 1 group 'general', got %v", groups)
	}
}

func TestRegisterAgent(t *testing.T) {
	h := NewHub()
	h.RegisterAgent("TestBot", types.AgentProfile{
		Name:   "TestBot",
		Color:  "#FF0000",
		Avatar: "🔴",
	})
	// Agent should be in defaults
	groups := h.GetGroups()
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
}

func TestJoinGroup(t *testing.T) {
	h := NewHub()
	h.JoinGroup("Alice", "general")
	h.JoinGroup("Bob", "general")
	h.JoinGroup("Alice", "dev")

	g := h.GetGroup("general")
	if g == nil {
		t.Fatal("general group not found")
	}
	if len(g.Members) != 2 {
		t.Errorf("expected 2 members in general, got %d", len(g.Members))
	}

	dev := h.GetGroup("dev")
	if dev == nil {
		t.Fatal("dev group not created")
	}
	if len(dev.Members) != 1 || dev.Members[0] != "Alice" {
		t.Errorf("expected Alice in dev, got %v", dev.Members)
	}
}

func TestJoinGroupIdempotent(t *testing.T) {
	h := NewHub()
	h.JoinGroup("Alice", "test")
	h.JoinGroup("Alice", "test") // duplicate
	g := h.GetGroup("test")
	if len(g.Members) != 1 {
		t.Errorf("expected 1 member (idempotent), got %d", len(g.Members))
	}
}

func TestLeaveGroup(t *testing.T) {
	h := NewHub()
	h.JoinGroup("Alice", "general")
	h.JoinGroup("Bob", "general")
	h.LeaveGroup("Alice", "general")

	g := h.GetGroup("general")
	if len(g.Members) != 1 || g.Members[0] != "Bob" {
		t.Errorf("expected only Bob, got %v", g.Members)
	}
}

func TestSendMessage(t *testing.T) {
	h := NewHub()
	msg := h.SendMessage("general", "Alice", "hello!", types.PriorityNormal)
	if msg == nil {
		t.Fatal("SendMessage returned nil")
	}
	if msg.Sender != "Alice" {
		t.Errorf("expected sender Alice, got %s", msg.Sender)
	}
	if msg.Body != "hello!" {
		t.Errorf("expected body 'hello!', got %s", msg.Body)
	}
	if msg.Group != "general" {
		t.Errorf("expected group 'general', got %s", msg.Group)
	}
	if msg.Priority != types.PriorityNormal {
		t.Errorf("expected priority normal, got %s", msg.Priority)
	}
	if msg.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestSendMessageAutoRegisters(t *testing.T) {
	h := NewHub()
	msg := h.SendMessage("general", "UnknownAgent", "hi", types.PriorityNormal)
	if msg.Color == "" {
		t.Error("expected auto-assigned color for unknown agent")
	}
}

func TestSendMessageStealFocus(t *testing.T) {
	h := NewHub()
	msg := h.SendMessage("general", "Alice", "URGENT!", types.PriorityStealFocus)
	if msg.Priority != types.PriorityStealFocus {
		t.Errorf("expected steal-focus, got %s", msg.Priority)
	}
}

func TestGetMessages(t *testing.T) {
	h := NewHub()
	h.SendMessage("general", "Alice", "msg1", types.PriorityNormal)
	h.SendMessage("general", "Bob", "msg2", types.PriorityNormal)
	h.SendMessage("dev", "Alice", "msg3", types.PriorityNormal)

	msgs := h.GetMessages("general", time.Time{}, 0)
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages in general, got %d", len(msgs))
	}

	msgs = h.GetMessages("dev", time.Time{}, 0)
	if len(msgs) != 1 {
		t.Errorf("expected 1 message in dev, got %d", len(msgs))
	}
}

func TestGetMessagesSince(t *testing.T) {
	h := NewHub()
	h.SendMessage("general", "Alice", "old", types.PriorityNormal)
	time.Sleep(10 * time.Millisecond)
	cutoff := time.Now()
	time.Sleep(10 * time.Millisecond)
	h.SendMessage("general", "Bob", "new", types.PriorityNormal)

	msgs := h.GetMessages("general", cutoff, 0)
	if len(msgs) != 1 || msgs[0].Body != "new" {
		t.Errorf("expected only 'new' message, got %v", msgs)
	}
}

func TestGetMessagesLimit(t *testing.T) {
	h := NewHub()
	for i := 0; i < 10; i++ {
		h.SendMessage("general", "Alice", "msg", types.PriorityNormal)
	}
	msgs := h.GetMessages("general", time.Time{}, 3)
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages (limit), got %d", len(msgs))
	}
}

func TestSubscribe(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe("general")

	// Send a message in a goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		h.SendMessage("general", "Alice", "hello", types.PriorityNormal)
	}()

	select {
	case msg := <-ch:
		if msg.Body != "hello" {
			t.Errorf("expected 'hello', got %s", msg.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for subscribed message")
	}
}

func TestSubscribeMultiple(t *testing.T) {
	h := NewHub()
	ch1 := h.Subscribe("general")
	ch2 := h.Subscribe("general")

	h.SendMessage("general", "Alice", "broadcast", types.PriorityNormal)

	// Both should receive
	for _, ch := range []<-chan *types.Message{ch1, ch2} {
		select {
		case msg := <-ch:
			if msg.Body != "broadcast" {
				t.Errorf("expected 'broadcast', got %s", msg.Body)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for broadcast")
		}
	}
}

func TestGetGroups(t *testing.T) {
	h := NewHub()
	h.JoinGroup("Alice", "alpha")
	h.JoinGroup("Bob", "beta")
	h.JoinGroup("Charlie", "gamma")

	groups := h.GetGroups()
	// Should be sorted alphabetically: alpha, beta, general, gamma
	if len(groups) != 4 {
		t.Errorf("expected 4 groups, got %d", len(groups))
	}
	if groups[0].Name != "alpha" {
		t.Errorf("expected first group 'alpha', got %s", groups[0].Name)
	}
}
