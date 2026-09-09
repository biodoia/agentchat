package tray

import (
	"log/slog"
	"os"
	"testing"
)

func TestNewSNIItem(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	item := NewSNIItem("test-agent", "dialog-information", "Test Agent", "http://127.0.0.1:18950", log)

	if item.name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %s", item.name)
	}
	if item.iconName != "dialog-information" {
		t.Errorf("expected icon 'dialog-information', got %s", item.iconName)
	}
	if item.title != "Test Agent" {
		t.Errorf("expected title 'Test Agent', got %s", item.title)
	}
	if item.status != "Active" {
		t.Errorf("expected status 'Active', got %s", item.status)
	}
	if item.relayURL != "http://127.0.0.1:18950" {
		t.Errorf("expected relayURL 'http://127.0.0.1:18950', got %s", item.relayURL)
	}
}

func TestNewSNIItemDefaultRelay(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	item := NewSNIItem("test", "icon", "title", "", log)

	if item.relayURL != "http://127.0.0.1:18950" {
		t.Errorf("expected default relayURL, got %s", item.relayURL)
	}
}

func TestMenuItems(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	item := NewSNIItem("test", "icon", "title", "http://localhost:18950", log)

	if len(item.menuItems) < 5 {
		t.Errorf("expected at least 5 menu items, got %d", len(item.menuItems))
	}

	// Check specific items exist
	labels := make(map[string]bool)
	for _, mi := range item.menuItems {
		labels[mi.Label] = true
	}
	for _, expected := range []string{"Show Chat", "New Group", "List Groups", "Status", "Quit"} {
		if !labels[expected] {
			t.Errorf("missing menu item: %s", expected)
		}
	}
}

func TestMenuItemsHaveActions(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	item := NewSNIItem("test", "icon", "title", "", log)

	for _, mi := range item.menuItems {
		if !mi.Sep && mi.Action == nil {
			t.Errorf("menu item '%s' has no action", mi.Label)
		}
	}
}

func TestSetOnClick(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	item := NewSNIItem("test", "icon", "title", "", log)

	called := false
	item.SetOnClick(func() { called = true })

	if item.onClick == nil {
		t.Error("expected onClick to be set")
	}
	item.onClick()
	if !called {
		t.Error("expected onClick to be called")
	}
}

func TestSetOnMenu(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	item := NewSNIItem("test", "icon", "title", "", log)

	called := false
	item.SetOnMenu(func() { called = true })

	if item.onMenu == nil {
		t.Error("expected onMenu to be set")
	}
	item.onMenu()
	if !called {
		t.Error("expected onMenu to be called")
	}
}

func TestSetMenuItems(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	item := NewSNIItem("test", "icon", "title", "", log)

	custom := []MenuItem{
		{Label: "Custom1", Action: func() {}},
		{Label: "Custom2", Action: func() {}},
	}
	item.SetMenuItems(custom)

	if len(item.menuItems) != 2 {
		t.Errorf("expected 2 custom items, got %d", len(item.menuItems))
	}
	if item.menuItems[0].Label != "Custom1" {
		t.Errorf("expected 'Custom1', got %s", item.menuItems[0].Label)
	}
}

func TestMenuItemStruct(t *testing.T) {
	called := false
	item := MenuItem{
		Label:  "Test",
		Action: func() { called = true },
		Sep:    true,
	}

	if item.Label != "Test" {
		t.Errorf("expected 'Test', got %s", item.Label)
	}
	if !item.Sep {
		t.Error("expected Sep=true")
	}
	item.Action()
	if !called {
		t.Error("expected Action to be called")
	}
}
