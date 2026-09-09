// agentchat-mcp is an MCP (Model Context Protocol) server for AgentChat.
// It exposes chat tools via JSON-RPC 2.0 over stdio for AI agents.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const relayURL = "http://127.0.0.1:18950"

// JSONRPC represents a JSON-RPC 2.0 message.
type JSONRPC struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolDefinition describes an MCP tool.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPC
		if err := json.Unmarshal(line, &req); err != nil {
			sendError(nil, -32700, "Parse error")
			continue
		}

		switch req.Method {
		case "initialize":
			sendResult(req.ID, map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "agentchat-mcp",
					"version": "0.2.0",
				},
			})

		case "tools/list":
			sendResult(req.ID, map[string]interface{}{
				"tools": getToolDefinitions(),
			})

		case "tools/call":
			var params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			json.Unmarshal(req.Params, &params)
			handleToolCall(req.ID, params.Name, params.Arguments)

		case "notifications/initialized":
			// No response needed
		default:
			sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

func getToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "chat_send",
			Description: "Send a message to a chat group. Creates the group if it doesn't exist.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group":    map[string]string{"type": "string", "description": "Group name (default: general)"},
					"sender":   map[string]string{"type": "string", "description": "Sender name"},
					"body":     map[string]string{"type": "string", "description": "Message text"},
					"priority": map[string]string{"type": "string", "description": "normal or steal-focus"},
				},
				"required": []string{"sender", "body"},
			},
		},
		{
			Name:        "chat_poll",
			Description: "Get recent messages from a chat group.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group": map[string]string{"type": "string", "description": "Group name (default: general)"},
					"limit": map[string]string{"type": "integer", "description": "Max messages (default: 50)"},
				},
			},
		},
		{
			Name:        "chat_join",
			Description: "Join a chat group. Creates the group if it doesn't exist.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent": map[string]string{"type": "string", "description": "Agent name"},
					"group": map[string]string{"type": "string", "description": "Group name"},
				},
				"required": []string{"agent", "group"},
			},
		},
		{
			Name:        "chat_groups",
			Description: "List all chat groups.",
			InputSchema:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "chat_status",
			Description: "Get AgentChat relay system status and statistics.",
			InputSchema:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	}
}

func handleToolCall(id json.RawMessage, name string, args json.RawMessage) {
	switch name {
	case "chat_send":
		var params struct {
			Group    string `json:"group"`
			Sender   string `json:"sender"`
			Body     string `json:"body"`
			Priority string `json:"priority"`
		}
		json.Unmarshal(args, &params)
		if params.Group == "" {
			params.Group = "general"
		}
		if params.Priority == "" {
			params.Priority = "normal"
		}
		payload, _ := json.Marshal(params)
		resp, err := http.Post(relayURL+"/api/message", "application/json", bytes.NewReader(payload))
		if err != nil {
			sendToolError(id, fmt.Sprintf("Failed to send: %v", err))
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		sendToolResult(id, string(body))

	case "chat_poll":
		var params struct {
			Group string `json:"group"`
			Limit int    `json:"limit"`
		}
		json.Unmarshal(args, &params)
		if params.Group == "" {
			params.Group = "general"
		}
		if params.Limit == 0 {
			params.Limit = 50
		}
		url := fmt.Sprintf("%s/api/messages?group=%s&since=%s",
			relayURL, params.Group,
			time.Now().Add(-24*time.Hour).Format(time.RFC3339))
		resp, err := http.Get(url)
		if err != nil {
			sendToolError(id, fmt.Sprintf("Failed to poll: %v", err))
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		sendToolResult(id, string(body))

	case "chat_join":
		var params struct {
			Agent string `json:"agent"`
			Group string `json:"group"`
		}
		json.Unmarshal(args, &params)
		payload, _ := json.Marshal(params)
		resp, err := http.Post(relayURL+"/api/group/join", "application/json", bytes.NewReader(payload))
		if err != nil {
			sendToolError(id, fmt.Sprintf("Failed to join: %v", err))
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		sendToolResult(id, string(body))

	case "chat_groups":
		resp, err := http.Get(relayURL + "/api/groups")
		if err != nil {
			sendToolError(id, fmt.Sprintf("Failed to list: %v", err))
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		sendToolResult(id, string(body))

	case "chat_status":
		resp, err := http.Get(relayURL + "/api/status")
		if err != nil {
			sendToolError(id, fmt.Sprintf("Failed to get status: %v", err))
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		sendToolResult(id, string(body))

	default:
		sendError(id, -32601, fmt.Sprintf("Unknown tool: %s", name))
	}
}

func sendResult(id json.RawMessage, result interface{}) {
	msg := JSONRPC{JSONRPC: "2.0", ID: id, Result: result}
	data, _ := json.Marshal(msg)
	fmt.Println(string(data))
}

func sendError(id json.RawMessage, code int, message string) {
	msg := JSONRPC{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: message}}
	data, _ := json.Marshal(msg)
	fmt.Println(string(data))
}

func sendToolResult(id json.RawMessage, text string) {
	sendResult(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
	})
}

func sendToolError(id json.RawMessage, text string) {
	sendResult(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"isError": true,
	})
}
