package mcp

import (
	"encoding/json"
	"time"
)

// Config holds MCP client configuration.
type Config struct {
	// ServerURL is the base URL of the MCP server.
	ServerURL string
	// Timeout is the request timeout.
	Timeout time.Duration
	// Headers are custom HTTP headers to send with requests.
	Headers map[string]string
}

// DefaultConfig returns a default MCP configuration.
func DefaultConfig(serverURL string) Config {
	return Config{
		ServerURL: serverURL,
		Timeout:   30 * time.Second,
		Headers:   make(map[string]string),
	}
}

// MCPContent represents a content block in MCP responses.
type MCPContent struct {
	Type string          `json:"type"`
	Text string          `json:"text,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// MCPToolInfo represents a tool discovered from an MCP server.
type MCPToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// MCPToolResult represents the result of calling an MCP tool.
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError"`
}

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolsListParams are params for tools/list.
type ToolsListParams struct{}

// ToolsListResult is the result of tools/list.
type ToolsListResult struct {
	Tools []MCPToolInfo `json:"tools"`
}

// ToolsCallParams are params for tools/call.
type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}
