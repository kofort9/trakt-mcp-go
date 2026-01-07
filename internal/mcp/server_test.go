package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestServer_Initialize(t *testing.T) {
	server := NewServer(nil)

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = server.RunWithIO(ctx, strings.NewReader(input+"\n"), &buf)
		close(done)
	}()

	// Wait for server to finish processing (input is finite)
	<-done

	var resp Response
	if err := json.NewDecoder(&buf).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Result is a map from JSON unmarshaling
	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", resp.Result)
	}
	if resultMap["protocolVersion"] != ProtocolVersion {
		t.Errorf("expected protocol version %s, got %v", ProtocolVersion, resultMap["protocolVersion"])
	}
}

func TestServer_ToolsList(t *testing.T) {
	server := NewServer(nil)

	// Register a test tool
	server.RegisterTool(Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: JSONSchema{Type: "object"},
	}, func(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
		return ToolCallResult{Content: []Content{TextContent("ok")}}, nil
	})

	// Initialize first
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	listReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	input := initReq + "\n" + listReq + "\n"

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = server.RunWithIO(ctx, strings.NewReader(input), &buf)
		close(done)
	}()

	// Wait for both responses
	<-done

	// Parse responses
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected 2 responses, got %d: %s", len(lines), buf.String())
	}

	var listResp Response
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("failed to decode tools/list response: %v", err)
	}

	if listResp.Error != nil {
		t.Fatalf("unexpected error: %v", listResp.Error)
	}

	resultMap, ok := listResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", listResp.Result)
	}

	tools, ok := resultMap["tools"].([]any)
	if !ok {
		t.Fatalf("tools not found in result")
	}

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestServer_ToolsCall(t *testing.T) {
	server := NewServer(nil)

	// Register a test tool
	server.RegisterTool(Tool{
		Name:        "echo",
		Description: "Echo the input",
		InputSchema: JSONSchema{Type: "object"},
	}, func(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
		return ToolCallResult{Content: []Content{TextContent("echoed: " + string(args))}}, nil
	})

	// Initialize, then call tool
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	callReq := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"msg":"hello"}}}`
	input := initReq + "\n" + callReq + "\n"

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = server.RunWithIO(ctx, strings.NewReader(input), &buf)
		close(done)
	}()

	<-done

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}

	var callResp Response
	if err := json.Unmarshal([]byte(lines[1]), &callResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if callResp.Error != nil {
		t.Fatalf("unexpected error: %v", callResp.Error)
	}
}

func TestServer_MethodNotFound(t *testing.T) {
	server := NewServer(nil)

	input := `{"jsonrpc":"2.0","id":1,"method":"unknown/method","params":{}}`

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = server.RunWithIO(ctx, strings.NewReader(input+"\n"), &buf)
		close(done)
	}()

	<-done

	var resp Response
	if err := json.NewDecoder(&buf).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}

	if resp.Error.Code != MethodNotFound {
		t.Errorf("expected error code %d, got %d", MethodNotFound, resp.Error.Code)
	}
}

func TestServer_InvalidJSON(t *testing.T) {
	server := NewServer(nil)

	input := `{invalid json`

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = server.RunWithIO(ctx, strings.NewReader(input+"\n"), &buf)
		close(done)
	}()

	<-done

	var resp Response
	if err := json.NewDecoder(&buf).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if resp.Error.Code != ParseError {
		t.Errorf("expected error code %d, got %d", ParseError, resp.Error.Code)
	}
}

func TestServer_UninitializedToolCall(t *testing.T) {
	server := NewServer(nil)

	server.RegisterTool(Tool{
		Name:        "test",
		Description: "Test tool",
		InputSchema: JSONSchema{Type: "object"},
	}, func(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
		return ToolCallResult{Content: []Content{TextContent("ok")}}, nil
	})

	// Call tool without initializing
	callReq := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test","arguments":{}}}`

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = server.RunWithIO(ctx, strings.NewReader(callReq+"\n"), &buf)
		close(done)
	}()

	<-done

	var resp Response
	if err := json.NewDecoder(&buf).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error for uninitialized tool call")
	}

	if resp.Error.Code != InternalError {
		t.Errorf("expected error code %d, got %d", InternalError, resp.Error.Code)
	}
}
