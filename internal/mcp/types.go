// Package mcp implements the Model Context Protocol (MCP) server.
// MCP uses JSON-RPC 2.0 over stdio for communication with AI assistants.
package mcp

import "encoding/json"

// JSON-RPC 2.0 types

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // Can be string, number, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP Protocol types

// InitializeParams contains parameters for the initialize request.
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ClientInfo      Implementation `json:"clientInfo"`
}

// InitializeResult contains the response to an initialize request.
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ServerInfo      Implementation `json:"serverInfo"`
}

// Capabilities describes what the server can do.
type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability describes tool-related capabilities.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// Implementation identifies a client or server.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool represents an MCP tool that can be called.
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	InputSchema JSONSchema `json:"inputSchema"`
}

// JSONSchema is a simplified JSON Schema for tool parameters.
type JSONSchema struct {
	Type                 string                `json:"type"`
	Properties           map[string]JSONSchema `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Description          string                `json:"description,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
	AdditionalProperties bool                  `json:"additionalProperties,omitempty"`
}

// ToolsListResult contains the response to a tools/list request.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams contains parameters for a tools/call request.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult contains the response to a tools/call request.
type ToolCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents a piece of content in a tool response.
type Content struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
}

// TextContent creates a text content item.
func TextContent(text string) Content {
	return Content{Type: "text", Text: text}
}

// ErrorContent creates an error content item.
func ErrorContent(err error) ToolCallResult {
	return ToolCallResult{
		Content: []Content{TextContent(err.Error())},
		IsError: true,
	}
}
