// Package notify sends desktop notifications on Manjaro Sway (Wayland).
// Uses notify-send (libnotify) which integrates with mako/dunst.
package notify

import (
	"fmt"
	"os/exec"
)

// Desktop sends a notification via notify-send.
// On Sway/mako this creates a popup notification.
func Desktop(summary, body, icon, urgency string) error {
	args := []string{}
	if urgency != "" {
		args = append(args, "--urgency", urgency)
	}
	if icon != "" {
		args = append(args, "--icon", icon)
	}
	args = append(args, summary, body)
	return exec.Command("notify-send", args...).Run()
}

// ChatMessage sends a chat notification with the agent's avatar.
func ChatMessage(sender, body, color string) error {
	summary := fmt.Sprintf("💬 %s", sender)
	return Desktop(summary, body, "dialog-information", "normal")
}

// StealFocus sends a high-urgency notification that grabs focus.
// On mako, this overrides the timeout and stays until dismissed.
func StealFocus(sender, body string) error {
	summary := fmt.Sprintf("🚨 %s needs attention!", sender)
	return Desktop(summary, body, "dialog-warning", "critical")
}
