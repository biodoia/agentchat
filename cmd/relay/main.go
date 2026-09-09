// agentchat-relay is the central relay server for AgentChat.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/biodoia/agentchat/internal/relay"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18950", "listen address")
	dataDir := flag.String("data", "./data", "data directory for PebbleDB persistence")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	fmt.Fprintf(os.Stderr, "╔═══════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stderr, "║    AgentChat Relay v0.2.0              ║\n")
	fmt.Fprintf(os.Stderr, "║    Multi-agent group chat server       ║\n")
	fmt.Fprintf(os.Stderr, "╚═══════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stderr, "\nListening on %s\n", *addr)
	fmt.Fprintf(os.Stderr, "Data dir:  %s\n", *dataDir)
	fmt.Fprintf(os.Stderr, "WebSocket: ws://%s/ws\n", *addr)
	fmt.Fprintf(os.Stderr, "API:       http://%s/api/\n", *addr)
	fmt.Fprintf(os.Stderr, "A2A:       http://%s/a2a/\n", *addr)
	fmt.Fprintf(os.Stderr, "AgentCard: http://%s/.well-known/agent.json\n\n", *addr)

	srv := relay.New(*addr, *dataDir, log)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}
