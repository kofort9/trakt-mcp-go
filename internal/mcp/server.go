package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "trakt-mcp-go"
	ServerVersion   = "0.1.0"
)

// ToolHandler is a function that handles a tool call.
type ToolHandler func(ctx context.Context, args json.RawMessage) (ToolCallResult, error)

// Server is an MCP server that communicates over stdio.
type Server struct {
	tools    map[string]Tool
	handlers map[string]ToolHandler
	logger   *slog.Logger

	mu          sync.RWMutex
	initialized bool
}

// NewServer creates a new MCP server.
func NewServer(logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	return &Server{
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
		logger:   logger,
	}
}

// RegisterTool registers a tool with the server.
func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = tool
	s.handlers[tool.Name] = handler
	s.logger.Debug("registered tool", "name", tool.Name)
}

// Run starts the server, reading from stdin and writing to stdout.
func (s *Server) Run(ctx context.Context) error {
	return s.RunWithIO(ctx, os.Stdin, os.Stdout)
}

// RunWithIO starts the server with custom I/O streams (useful for testing).
func (s *Server) RunWithIO(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Increase buffer size for large messages
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	s.logger.Info("server starting", "version", ServerVersion)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.handleMessage(ctx, line)
		if resp != nil {
			if err := s.writeResponse(out, resp); err != nil {
				s.logger.Error("failed to write response", "error", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func (s *Server) handleMessage(ctx context.Context, data []byte) *Response {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		s.logger.Error("failed to parse request", "error", err)
		return &Response{
			JSONRPC: "2.0",
			Error:   &Error{Code: ParseError, Message: "Parse error"},
		}
	}

	if req.JSONRPC != "2.0" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: InvalidRequest, Message: "Invalid JSON-RPC version"},
		}
	}

	s.logger.Debug("handling request", "method", req.Method)

	result, err := s.dispatch(ctx, req.Method, req.Params)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   err,
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage) (any, *Error) {
	switch method {
	case "initialize":
		return s.handleInitialize(params)
	case "initialized":
		// Notification, no response needed
		return nil, nil
	case "tools/list":
		return s.handleToolsList()
	case "tools/call":
		return s.handleToolsCall(ctx, params)
	default:
		return nil, &Error{Code: MethodNotFound, Message: fmt.Sprintf("Method not found: %s", method)}
	}
}

func (s *Server) handleInitialize(params json.RawMessage) (*InitializeResult, *Error) {
	var p InitializeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &Error{Code: InvalidParams, Message: "Invalid initialize params"}
	}

	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	s.logger.Info("initialized",
		"client", p.ClientInfo.Name,
		"clientVersion", p.ClientInfo.Version,
		"protocolVersion", p.ProtocolVersion,
	)

	return &InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: Capabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: Implementation{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}, nil
}

func (s *Server) handleToolsList() (*ToolsListResult, *Error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, t)
	}

	return &ToolsListResult{Tools: tools}, nil
}

func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (*ToolCallResult, *Error) {
	var p ToolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &Error{Code: InvalidParams, Message: "Invalid tools/call params"}
	}

	s.mu.RLock()
	handler, ok := s.handlers[p.Name]
	s.mu.RUnlock()

	if !ok {
		return nil, &Error{Code: InvalidParams, Message: fmt.Sprintf("Unknown tool: %s", p.Name)}
	}

	s.logger.Debug("calling tool", "name", p.Name)

	result, err := handler(ctx, p.Arguments)
	if err != nil {
		s.logger.Error("tool error", "name", p.Name, "error", err)
		return &ToolCallResult{
			Content: []Content{TextContent(err.Error())},
			IsError: true,
		}, nil
	}

	return &result, nil
}

func (s *Server) writeResponse(out io.Writer, resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", data)
	return err
}
