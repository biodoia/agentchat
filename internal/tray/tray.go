// Package tray implements a system tray icon for Sway (Wayland)
// using the StatusNotifierItem (SNI) D-Bus protocol.
// This works with waybar's tray module.
package tray

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

// Tray manages a system tray icon with context menu via wofi.
type Tray struct {
	name    string
	icon    string
	tooltip string
	log     *slog.Logger
	menuFn  func(choice string) // callback when menu item selected
}

// New creates a new tray icon manager.
func New(name string, log *slog.Logger) *Tray {
	return &Tray{
		name:    name,
		icon:    "internet-chat", // freedesktop standard icon
		tooltip: "AgentChat — AI Group Chat",
		log:     log,
	}
}

// SetMenuCallback sets the function called when a menu item is selected.
func (t *Tray) SetMenuCallback(fn func(choice string)) {
	t.menuFn = fn
}

// ShowContextMenu displays a right-click context menu via wofi.
func (t *Tray) ShowContextMenu() {
	items := []string{
		"📋 Show Chat",
		"➕ New Group",
		"👥 List Agents",
	"📊 Status",
		"─────────",
		"🔄 Restart Relay",
		"⚙️  Settings",
		"❌ Quit",
	}

	// Write menu items to wofi
	cmd := exec.Command("wofi", "--dmenu", "--prompt", "AgentChat", "--width", "300", "--height", "350")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.log.Error("wofi stdin pipe", "err", err)
		return
	}

	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item)
		}
	}()

	out, err := cmd.Output()
	if err != nil {
		// User cancelled or error
		return
	}

	choice := string(out)
	if t.menuFn != nil {
		t.menuFn(choice)
	}
}

// InstallDesktopEntry creates a .desktop file for the tray app.
func InstallDesktopEntry() error {
	dir := filepath.Join(os.Getenv("HOME"), ".local", "share", "applications")
	os.MkdirAll(dir, 0o755)

	content := `[Desktop Entry]
Type=Application
Name=AgentChat
Comment=AI Agent Group Chat
Exec=agentchat-tui
Icon=internet-chat
Terminal=false
Categories=Network;Chat;
StartupNotify=true
`
	path := filepath.Join(dir, "agentchat.desktop")
	return os.WriteFile(path, []byte(content), 0o644)
}
