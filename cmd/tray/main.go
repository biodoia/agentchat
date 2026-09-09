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

	relayURL := os.Getenv("AGENTCHAT_RELAY")
	if relayURL == "" {
		relayURL = "http://127.0.0.1:18950"
	}

	item := tray.NewSNIItem("agentchat", "internet-chat", "AgentChat - AI Group Chat", relayURL, log)
	item.SetOnClick(func() {
		exec.Command("foot", "-T", "AgentChat", "agentchat-tui").Start()
	})

	if err := item.Run(); err != nil {
		log.Error("tray failed", "err", err)
		os.Exit(1)
	}
}
