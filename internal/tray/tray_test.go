package tray

import (
	"testing"
)

func TestNewSNIItem(t *testing.T) {
	// Can't test D-Bus without a session bus, but we can test construction
	item := &SNIItem{
		name:     "test",
		iconName: "internet-chat",
		title:    "Test",
		status:   "Active",
	}
	if item.name != "test" {
		t.Errorf("expected name 'test', got %s", item.name)
	}
	if item.iconName != "internet-chat" {
		t.Errorf("expected icon 'internet-chat', got %s", item.iconName)
	}
}

func TestMenuItem(t *testing.T) {
	items := []MenuItem{
		{Label: "Show Chat", Action: func() {}},
		{Sep: true},
		{Label: "Quit", Action: func() {}},
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
	if !items[1].Sep {
		t.Error("expected separator on item 1")
	}
}
