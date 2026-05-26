package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testMCPServer creates a mock MCP server for testing.
func testMCPServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "initialize":
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
			})
		case "tools/list":
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`{
					"tools": [
						{
							"name": "test_tool",
							"description": "A test tool",
							"inputSchema": {"type": "object"}
						}
					]
				}`),
			})
		case "tools/call":
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`{
					"content": [{"type": "text", "text": "tool result"}],
					"isError": false
				}`),
			})
		default:
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			})
		}
	})
	return httptest.NewServer(mux)
}

// testMCPRPCErrServer creates a server that always returns RPC errors.
func testMCPRPCErrServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Internal server error",
				Data:    "something went wrong",
			},
		})
	})
	return httptest.NewServer(mux)
}

// testMCPBadStatusServer creates a server that returns non-200 status.
func testMCPBadStatusServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})
	return httptest.NewServer(mux)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("http://localhost:8080")

	if cfg.ServerURL != "http://localhost:8080" {
		t.Errorf("expected ServerURL 'http://localhost:8080', got %q", cfg.ServerURL)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.Headers == nil {
		t.Error("expected Headers to be non-nil")
	}
}

func TestConfigWithCustomValues(t *testing.T) {
	cfg := Config{
		ServerURL: "http://example.com/mcp",
		Timeout:   10 * time.Second,
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	}

	if cfg.ServerURL != "http://example.com/mcp" {
		t.Errorf("expected ServerURL 'http://example.com/mcp', got %q", cfg.ServerURL)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", cfg.Timeout)
	}
	if cfg.Headers["Authorization"] != "Bearer token123" {
		t.Error("expected Authorization header")
	}
}

func TestMCPContentSerialization(t *testing.T) {
	content := MCPContent{
		Type: "text",
		Text: "hello world",
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal MCPContent: %v", err)
	}

	var decoded MCPContent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal MCPContent: %v", err)
	}

	if decoded.Type != "text" {
		t.Errorf("expected Type 'text', got %q", decoded.Type)
	}
	if decoded.Text != "hello world" {
		t.Errorf("expected Text 'hello world', got %q", decoded.Text)
	}
}

func TestMCPContentWithData(t *testing.T) {
	content := MCPContent{
		Type: "image",
		Data: json.RawMessage(`{"url":"http://example.com/img.png"}`),
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MCPContent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != "image" {
		t.Errorf("expected Type 'image', got %q", decoded.Type)
	}
}

func TestMCPToolInfoSerialization(t *testing.T) {
	info := MCPToolInfo{
		Name:        "search",
		Description: "Search for items",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal MCPToolInfo: %v", err)
	}

	var decoded MCPToolInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal MCPToolInfo: %v", err)
	}

	if decoded.Name != "search" {
		t.Errorf("expected Name 'search', got %q", decoded.Name)
	}
	if decoded.Description != "Search for items" {
		t.Errorf("expected Description 'Search for items', got %q", decoded.Description)
	}
}

func TestMCPToolResultSerialization(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: "result text"},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal MCPToolResult: %v", err)
	}

	var decoded MCPToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal MCPToolResult: %v", err)
	}

	if len(decoded.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(decoded.Content))
	}
	if decoded.Content[0].Text != "result text" {
		t.Errorf("expected Text 'result text', got %q", decoded.Content[0].Text)
	}
	if decoded.IsError {
		t.Error("expected IsError to be false")
	}
}

func TestMCPToolResultError(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: "something went wrong"},
		},
		IsError: true,
	}

	if !result.IsError {
		t.Error("expected IsError to be true")
	}
}

func TestJSONRPCRequestSerialization(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  nil,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal JSONRPCRequest: %v", err)
	}

	var decoded JSONRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSONRPCRequest: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC '2.0', got %q", decoded.JSONRPC)
	}
	if decoded.ID != 1 {
		t.Errorf("expected ID 1, got %d", decoded.ID)
	}
	if decoded.Method != "tools/list" {
		t.Errorf("expected Method 'tools/list', got %q", decoded.Method)
	}
}

func TestJSONRPCResponseSerialization(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"tools":[]}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal JSONRPCResponse: %v", err)
	}

	var decoded JSONRPCResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSONRPCResponse: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC '2.0', got %q", decoded.JSONRPC)
	}
	if decoded.Error != nil {
		t.Error("expected no error in response")
	}
}

func TestJSONRPCResponseWithError(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      2,
		Error: &JSONRPCError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded JSONRPCResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("expected error in response")
	}
	if decoded.Error.Code != -32600 {
		t.Errorf("expected error code -32600, got %d", decoded.Error.Code)
	}
	if decoded.Error.Message != "Invalid Request" {
		t.Errorf("expected error message 'Invalid Request', got %q", decoded.Error.Message)
	}
}

func TestNewClient(t *testing.T) {
	cfg := DefaultConfig("http://localhost:9090")
	client := NewClient(cfg)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.IsConnected() {
		t.Error("expected client to not be connected initially")
	}
}

func TestClientConnect(t *testing.T) {
	server := testMCPServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if !client.IsConnected() {
		t.Error("expected client to be connected after Connect()")
	}
}

func TestClientConnectWithCustomHeaders(t *testing.T) {
	receivedHeaders := make(map[string]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders["X-Custom"] = r.Header.Get("X-Custom")
		receivedHeaders["Authorization"] = r.Header.Get("Authorization")

		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
		})
	}))
	defer server.Close()

	cfg := Config{
		ServerURL: server.URL,
		Timeout:   5 * time.Second,
		Headers: map[string]string{
			"X-Custom":     "test-value",
			"Authorization": "Bearer my-token",
		},
	}
	client := NewClient(cfg)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	if receivedHeaders["X-Custom"] != "test-value" {
		t.Errorf("expected X-Custom header 'test-value', got %q", receivedHeaders["X-Custom"])
	}
	if receivedHeaders["Authorization"] != "Bearer my-token" {
		t.Errorf("expected Authorization header, got %q", receivedHeaders["Authorization"])
	}
}

func TestClientDiscoverTools(t *testing.T) {
	server := testMCPServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	tools, err := client.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("failed to discover tools: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", tools[0].Name)
	}
	if tools[0].Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", tools[0].Description)
	}
}

func TestClientCallTool(t *testing.T) {
	server := testMCPServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{
		"arg1": "value1",
	})
	if err != nil {
		t.Fatalf("failed to call tool: %v", err)
	}

	if result.IsError {
		t.Error("expected IsError to be false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Text != "tool result" {
		t.Errorf("expected text 'tool result', got %q", result.Content[0].Text)
	}
}

func TestClientCallToolWithNoArgs(t *testing.T) {
	server := testMCPServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	result, err := client.CallTool(ctx, "test_tool", nil)
	if err != nil {
		t.Fatalf("failed to call tool: %v", err)
	}

	if result.IsError {
		t.Error("expected IsError to be false")
	}
}

func TestClientClose(t *testing.T) {
	server := testMCPServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}
	if client.IsConnected() {
		t.Error("expected client to be disconnected after Close()")
	}
}

func TestClientConnectionFailure(t *testing.T) {
	cfg := DefaultConfig("http://127.0.0.1:0")
	cfg.Timeout = 1 * time.Second
	client := NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected connection to fail")
	}
	if client.IsConnected() {
		t.Error("expected client to not be connected after failure")
	}
}

func TestClientBadStatusError(t *testing.T) {
	server := testMCPBadStatusServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	_, err := client.DiscoverTools(ctx)
	if err == nil {
		t.Fatal("expected error from bad status server")
	}
}

func TestClientRPCError(t *testing.T) {
	server := testMCPRPCErrServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected RPC error")
	}
}

func TestClientRPCToolCallError(t *testing.T) {
	server := testMCPRPCErrServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()
	_, err := client.CallTool(ctx, "failing_tool", nil)
	if err == nil {
		t.Fatal("expected RPC error from tool call")
	}
}

func TestClientIDIncrement(t *testing.T) {
	server := testMCPServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)

	ctx := context.Background()

	// Multiple calls should use incrementing IDs
	_, err1 := client.DiscoverTools(ctx)
	_, err2 := client.DiscoverTools(ctx)
	_, err3 := client.DiscoverTools(ctx)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("expected all calls to succeed")
	}
}

func TestToolsCallParamsSerialization(t *testing.T) {
	params := ToolsCallParams{
		Name: "search",
		Arguments: map[string]interface{}{
			"query": "test",
			"limit": 10,
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal ToolsCallParams: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded["name"] != "search" {
		t.Errorf("expected name 'search', got %v", decoded["name"])
	}
}

func TestToolsListResultEmpty(t *testing.T) {
	raw := json.RawMessage(`{"tools":[]}`)
	var result ToolsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("failed to unmarshal empty tools: %v", err)
	}
	if len(result.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result.Tools))
	}
}
