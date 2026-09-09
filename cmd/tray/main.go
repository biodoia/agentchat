// agentchat-tray is the system tray icon for AgentChat on Sway/Wayland.
// Uses StatusNotifierItem D-Bus protocol (waybar compatible).
package main

import (
	"log/slog"
	"os"
	"os/exec"

	"github.com/biodoia/agentchat/internal/tray"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	item := tray.NewSNIItem("agentchat", "internet-chat", "AgentChat - AI Group Chat", log)
	item.SetOnClick(func() {
		exec.Command("foot", "-T", "AgentChat", "agentchat-tui").Start()
	})

	if err := item.Run(); err != nil {
		log.Error("tray failed", "err", err)
		os.Exit(1)
	}
}
