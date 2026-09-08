// agentchat-mcp is the MCP server for AgentChat.
// Agents that don't have native A2A support can use MCP tools
// to participate in group chats.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const relayURL = "http://127.0.0.1:18950"

func main() {
	// MCP server using stdio JSON-RPC
	// For now, a simple CLI bridge
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "send":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: agentchat-mcp send <group> <sender> <message>\n")
			os.Exit(1)
		}
		group, sender, body := os.Args[2], os.Args[3], os.Args[4]
		msg := map[string]string{"group": group, "sender": sender, "body": body}
		data, _ := json.Marshal(msg)
		resp, err := http.Post(relayURL+"/api/message", "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)

	case "poll":
		group := "general"
		if len(os.Args) > 2 {
			group = os.Args[2]
		}
		resp, err := http.Get(fmt.Sprintf("%s/api/messages?group=%s&since=%s",
			relayURL, group, time.Now().Add(-5*time.Minute).Format(time.RFC3339)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)

	case "join":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: agentchat-mcp join <agent> [group]\n")
			os.Exit(1)
		}
		agent := os.Args[2]
		group := "general"
		if len(os.Args) > 3 {
			group = os.Args[3]
		}
		msg := map[string]string{"agent": agent, "group": group}
		data, _ := json.Marshal(msg)
		resp, err := http.Post(relayURL+"/api/group/join", "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)

	case "groups":
		resp, err := http.Get(relayURL + "/api/groups")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)

	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `agentchat-mcp — MCP/CLI bridge for AgentChat

Usage:
  agentchat-mcp send <group> <sender> <message>   Send a message
  agentchat-mcp poll [group]                       Get recent messages
  agentchat-mcp join <agent> [group]               Join a group
  agentchat-mcp groups                             List groups

Relay: %s
`, relayURL)
}
