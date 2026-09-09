package main

import (
	"encoding/json"
	"testing"
)

func TestGetToolDefinitions(t *testing.T) {
	tools := getToolDefinitions()

	if len(tools) != 5 {
		t.Fatalf("expected 5 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %s has no description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Errorf("tool %s has no input schema", tool.Name)
		}
	}

	for _, expected := range []string{"chat_send", "chat_poll", "chat_join", "chat_groups", "chat_status"} {
		if !names[expected] {
			t.Errorf("missing tool: %s", expected)
		}
	}
}

func TestToolDefinitionsHaveRequiredFields(t *testing.T) {
	tools := getToolDefinitions()

	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("tool has empty name")
		}
		if len(tool.Description) < 10 {
			t.Errorf("tool %s description too short: %s", tool.Name, tool.Description)
		}
	}
}

func TestJSONRPCSerialization(t *testing.T) {
	msg := JSONRPC{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"test-id"`),
		Result:  map[string]string{"status": "ok"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded JSONRPC
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", decoded.JSONRPC)
	}
	if string(decoded.ID) != `"test-id"` {
		t.Errorf("expected id 'test-id', got %s", string(decoded.ID))
	}
}

func TestRPCErrorSerialization(t *testing.T) {
	err := &RPCError{Code: -32601, Message: "Method not found"}
	msg := JSONRPC{JSONRPC: "2.0", ID: json.RawMessage(`"1"`), Error: err}

	data, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		t.Fatalf("failed to marshal: %v", marshalErr)
	}

	var decoded JSONRPC
	json.Unmarshal(data, &decoded)

	if decoded.Error == nil {
		t.Fatal("expected error to be set")
	}
	if decoded.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", decoded.Error.Code)
	}
}
