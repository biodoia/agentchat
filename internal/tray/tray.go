// Package tray implements a StatusNotifierItem D-Bus system tray icon
// for Sway/Wayland. Compatible with waybar's tray module.
package tray

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	sniWatcherName = "org.kde.StatusNotifierWatcher"
	sniWatcherPath = "/StatusNotifierWatcher"
	sniItemIface   = "org.kde.StatusNotifierItem"
	sniItemPath    = "/StatusNotifierItem"
)

// SNIItem is a StatusNotifierItem D-Bus tray icon.
type SNIItem struct {
	conn     *dbus.Conn
	name     string
	iconName string
	title    string
	status   string // "Active", "NeedsAttention", "Passive"
	tooltip  string
	log      *slog.Logger
	onClick  func()
	onMenu   func()
	menuItems []MenuItem
}

// MenuItem is one entry in the context menu.
type MenuItem struct {
	Label   string
	Action  func()
	Sep     bool // separator after this item
}

// NewSNIItem creates a new StatusNotifierItem.
func NewSNIItem(name, icon, title string, log *slog.Logger) *SNIItem {
	return &SNIItem{
		name:     name,
		iconName: icon,
		title:    title,
		status:   "Active",
		log:      log,
		menuItems: []MenuItem{
			{Label: "Show Chat", Action: func() { exec.Command("agentchat-tui").Start() }},
			{Sep: true},
			{Label: "New Group", Action: func() { showInputDialog("New Group") }},
			{Label: "List Agents", Action: func() { showInfo("agentchat-mcp groups") }},
			{Sep: true},
			{Label: "Status", Action: func() { showInfo("Relay running on :18950") }},
			{Sep: true},
			{Label: "Restart Relay", Action: func() { exec.Command("systemctl", "--user", "restart", "agentchat-relay").Run() }},
			{Label: "Quit", Action: func() { /* shutdown */ }},
		},
	}
}

// SetOnClick sets the left-click handler.
func (s *SNIItem) SetOnClick(fn func()) { s.onClick = fn }

// SetOnMenu sets the right-click handler.
func (s *SNIItem) SetOnMenu(fn func()) { s.onMenu = fn }

// SetMenuItems replaces the context menu items.
func (s *SNIItem) SetMenuItems(items []MenuItem) { s.menuItems = items }

// Run registers the item on D-Bus and blocks.
func (s *SNIItem) Run() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("tray: connect session bus: %w", err)
	}
	s.conn = conn
	defer conn.Close()

	// Request a unique name
	reply, err := conn.RequestName(s.sniItemName(), dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("tray: request name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("tray: name already taken")
	}

	// Export properties
	s.exportProperties()

	// Export methods
	s.exportMethods()

	// Export introspection
	s.exportIntrospect()

	// Register with StatusNotifierWatcher
	if err := s.registerWithWatcher(); err != nil {
		s.log.Warn("failed to register with SNI watcher (waybar may not be running)", "err", err)
	}

	s.log.Info("SNI tray icon registered", "name", s.name, "icon", s.iconName)

	// Block forever
	select {}
}

func (s *SNIItem) sniItemName() string {
	return fmt.Sprintf("org.kde.StatusNotifierItem-%d", 1)
}

func (s *SNIItem) exportProperties() {
	propsSpec := map[string]map[string]*prop.Prop{
		sniItemIface: {
			"Category": {Value: "Application", Writable: false},
			"Id":       {Value: s.name, Writable: false},
			"Title":    {Value: s.title, Writable: false},
			"Status":   {Value: s.status, Writable: false},
			"IconName": {Value: s.iconName, Writable: false},
			"ToolTip":  {Value: s.tooltip, Writable: false},
		},
	}
	_, err := prop.Export(s.conn, sniItemPath, propsSpec)
	if err != nil {
		s.log.Error("export properties", "err", err)
	}
}

func (s *SNIItem) exportMethods() {
	// The SNI interface methods
	s.conn.Export(s, sniItemPath, sniItemIface)
}

func (s *SNIItem) exportIntrospect() {
	node := &introspect.Node{
		Interfaces: []introspect.Interface{
			{
				Name: sniItemIface,
				Methods: []introspect.Method{
					{Name: "Activate", Args: []introspect.Arg{
						{Name: "x", Type: "i", Direction: "in"},
						{Name: "y", Type: "i", Direction: "in"},
					}},
					{Name: "ContextMenu", Args: []introspect.Arg{
						{Name: "x", Type: "i", Direction: "in"},
						{Name: "y", Type: "i", Direction: "in"},
					}},
					{Name: "Scroll", Args: []introspect.Arg{
						{Name: "delta", Type: "i", Direction: "in"},
						{Name: "orientation", Type: "s", Direction: "in"},
					}},
				},
				Properties: []introspect.Property{
					{Name: "Category", Type: "s", Access: "read"},
					{Name: "Id", Type: "s", Access: "read"},
					{Name: "Title", Type: "s", Access: "read"},
					{Name: "Status", Type: "s", Access: "read"},
					{Name: "IconName", Type: "s", Access: "read"},
					{Name: "ToolTip", Type: "s", Access: "read"},
				},
			},
		},
	}
	s.conn.Export(introspect.NewIntrospectable(node), sniItemPath, "org.freedesktop.DBus.Introspectable")
}

func (s *SNIItem) registerWithWatcher() error {
	// Check if watcher exists
	watcher := s.conn.Object(sniWatcherName, sniWatcherPath)
	call := watcher.Call(sniWatcherName+".RegisterStatusNotifierItem", 0, s.sniItemName())
	return call.Err
}

// Activate is called on left-click.
func (s *SNIItem) Activate(x, y int32) *dbus.Error {
	s.log.Debug("tray activate (left-click)")
	if s.onClick != nil {
		go s.onClick()
	}
	return nil
}

// ContextMenu is called on right-click.
func (s *SNIItem) ContextMenu(x, y int32) *dbus.Error {
	s.log.Debug("tray context menu (right-click)")
	if s.onMenu != nil {
		go s.onMenu()
	} else {
		go s.showWofiMenu()
	}
	return nil
}

// Scroll is called on scroll events.
func (s *SNIItem) Scroll(delta int32, orientation string) *dbus.Error {
	s.log.Debug("tray scroll", "delta", delta, "orientation", orientation)
	return nil
}

// showWofiMenu displays a wofi context menu with the configured items.
func (s *SNIItem) showWofiMenu() {
	if len(s.menuItems) == 0 {
		return
	}

	var labels []string
	for _, item := range s.menuItems {
		labels = append(labels, item.Label)
	}

	cmd := exec.Command("wofi", "--dmenu", "--prompt", "AgentChat",
		"--width", "280", "--height", fmt.Sprintf("%d", len(labels)*35+50),
		"--cache-file", "/dev/null")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.log.Error("wofi stdin", "err", err)
		return
	}

	go func() {
		defer stdin.Close()
		for _, label := range labels {
			fmt.Fprintln(stdin, label)
		}
	}()

	out, err := cmd.Output()
	if err != nil {
		return // cancelled
	}

	choice := strings.TrimSpace(string(out))
	for _, item := range s.menuItems {
		if item.Label == choice && item.Action != nil {
			go item.Action()
			return
		}
	}
}

func showInputDialog(prompt string) {
	exec.Command("wofi", "--dmenu", "--prompt", prompt).Run()
}

func showInfo(msg string) {
	exec.Command("notify-send", "--app-name", "AgentChat", msg).Run()
}
