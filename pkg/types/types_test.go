package types

import (
	"testing"
)

func TestDefaultAgents(t *testing.T) {
	if len(DefaultAgents) < 5 {
		t.Errorf("expected at least 5 default agents, got %d", len(DefaultAgents))
	}

	// Check MiMoCode exists
	mimo, ok := DefaultAgents["MiMoCode"]
	if !ok {
		t.Fatal("expected MiMoCode in defaults")
	}
	if mimo.Color != "#00FF88" {
		t.Errorf("expected MiMoCode color #00FF88, got %s", mimo.Color)
	}
	if mimo.Avatar != "🟢" {
		t.Errorf("expected MiMoCode avatar 🟢, got %s", mimo.Avatar)
	}
}

func TestPriorityConstants(t *testing.T) {
	if PriorityNormal != "normal" {
		t.Errorf("expected 'normal', got %s", PriorityNormal)
	}
	if PriorityStealFocus != "steal-focus" {
		t.Errorf("expected 'steal-focus', got %s", PriorityStealFocus)
	}
}

func TestMessageFields(t *testing.T) {
	msg := Message{
		ID:       "test-id",
		Group:    "general",
		Sender:   "Alice",
		Color:    "#FF0000",
		Avatar:   "🔴",
		Body:     "hello",
		Priority: PriorityNormal,
	}
	if msg.ID != "test-id" {
		t.Errorf("expected test-id, got %s", msg.ID)
	}
	if msg.Priority != PriorityNormal {
		t.Errorf("expected normal priority, got %s", msg.Priority)
	}
}

func TestGroupFields(t *testing.T) {
	g := Group{
		Name:    "dev",
		Members: []string{"Alice", "Bob"},
	}
	if g.Name != "dev" {
		t.Errorf("expected dev, got %s", g.Name)
	}
	if len(g.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(g.Members))
	}
}
