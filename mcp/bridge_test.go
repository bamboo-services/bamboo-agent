package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestBridgeServer creates a mock MCP server for bridge testing.
func newTestBridgeServer() *httptest.Server {
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
					"tools": [{
						"name": "test_tool",
						"description": "A test tool",
						"inputSchema": {
							"type": "object",
							"properties": {
								"query": {
									"type": "string",
									"description": "Search query"
								}
							},
							"required": ["query"]
						}
					}]
				}`),
			})
		case "tools/call":
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`{
					"content": [{"type": "text", "text": "result from mcp"}],
					"isError": false
				}`),
			})
		}
	})
	return httptest.NewServer(mux)
}

// newTestBridgeErrorServer creates a server that returns error on tool calls.
func newTestBridgeErrorServer() *httptest.Server {
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
					"tools": [{
						"name": "error_tool",
						"description": "A tool that errors",
						"inputSchema": {"type": "object"}
					}]
				}`),
			})
		case "tools/call":
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32000,
					Message: "Tool execution failed",
				},
			})
		}
	})
	return httptest.NewServer(mux)
}

// TestNewBridge 测试创建新桥接器。
func TestNewBridge(t *testing.T) {
	cfg := DefaultConfig("http://localhost:8080")
	client := NewClient(cfg)
	bridge := NewBridge(client)

	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}
	if bridge.client != client {
		t.Error("expected bridge to hold the provided client")
	}
}

// TestBridgeAsTools 测试桥接器转换为工具。
func TestBridgeAsTools(t *testing.T) {
	cfg := DefaultConfig("http://localhost:8080")
	client := NewClient(cfg)
	bridge := NewBridge(client)

	// Manually set tools for testing (without calling DiscoverAndConvert)
	bridge.tools = []MCPToolInfo{
		{
			Name:        "tool1",
			Description: "First tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		{
			Name:        "tool2",
			Description: "Second tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
	}

	tools := bridge.AsTools()

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	// Verify tools are mcpToolAdapter instances
	if tools[0].Info().Name != "tool1" {
		t.Errorf("expected first tool name 'tool1', got %q", tools[0].Info().Name)
	}
	if tools[1].Info().Name != "tool2" {
		t.Errorf("expected second tool name 'tool2', got %q", tools[1].Info().Name)
	}
}

// TestBridgeDiscoverAndConvert 测试桥接器发现和转换工具。
func TestBridgeDiscoverAndConvert(t *testing.T) {
	server := newTestBridgeServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)
	bridge := NewBridge(client)

	ctx := context.Background()
	tools, err := bridge.DiscoverAndConvert(ctx)
	if err != nil {
		t.Fatalf("failed to discover and convert: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	info := tools[0].Info()
	if info.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", info.Name)
	}
	if info.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", info.Description)
	}

	// Verify properties were parsed
	props := info.Parameters.Properties
	if props == nil {
		t.Fatal("expected properties to be non-nil")
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 property, got %d", len(props))
	}

	queryProp, ok := props["query"]
	if !ok {
		t.Fatal("expected 'query' property to exist")
	}
	if queryProp.Type != "string" {
		t.Errorf("expected query type 'string', got %q", queryProp.Type)
	}
	if queryProp.Description != "Search query" {
		t.Errorf("expected query description 'Search query', got %q", queryProp.Description)
	}
}

// TestMcpToolAdapterInfo 测试 MCP 工具适配器信息。
func TestMcpToolAdapterInfo(t *testing.T) {
	adapter := &mcpToolAdapter{
		info: MCPToolInfo{
			Name:        "search_tool",
			Description: "Searches for items",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"keyword": {
						"type": "string",
						"description": "Search keyword"
					},
					"limit": {
						"type": "integer",
						"description": "Max results"
					}
				},
				"required": ["keyword"]
			}`),
		},
		client: nil,
	}

	info := adapter.Info()

	if info.Name != "search_tool" {
		t.Errorf("expected name 'search_tool', got %q", info.Name)
	}
	if info.Description != "Searches for items" {
		t.Errorf("expected description 'Searches for items', got %q", info.Description)
	}
	if info.Parameters.Type != "object" {
		t.Errorf("expected parameters type 'object', got %q", info.Parameters.Type)
	}

	// Verify properties
	props := info.Parameters.Properties
	if len(props) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(props))
	}

	keyword, ok := props["keyword"]
	if !ok {
		t.Fatal("expected 'keyword' property")
	}
	if keyword.Type != "string" {
		t.Errorf("expected keyword type 'string', got %q", keyword.Type)
	}
	if keyword.Description != "Search keyword" {
		t.Errorf("expected keyword description 'Search keyword', got %q", keyword.Description)
	}

	limit, ok := props["limit"]
	if !ok {
		t.Fatal("expected 'limit' property")
	}
	if limit.Type != "integer" {
		t.Errorf("expected limit type 'integer', got %q", limit.Type)
	}
	if limit.Description != "Max results" {
		t.Errorf("expected limit description 'Max results', got %q", limit.Description)
	}
}

// TestMcpToolAdapterExecuteSuccess 测试 MCP 工具适配器成功执行。
func TestMcpToolAdapterExecuteSuccess(t *testing.T) {
	server := newTestBridgeServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)
	adapter := &mcpToolAdapter{
		info: MCPToolInfo{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		client: client,
	}

	ctx := context.Background()
	input := json.RawMessage(`{"query":"test"}`)
	result, err := adapter.Execute(ctx, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Errorf("expected IsError to be false, got true")
	}
	if result.Content != "result from mcp" {
		t.Errorf("expected content 'result from mcp', got %q", result.Content)
	}
}

// TestMcpToolAdapterExecuteInvalidJSON 测试 MCP 工具适配器执行无效 JSON。
func TestMcpToolAdapterExecuteInvalidJSON(t *testing.T) {
	cfg := DefaultConfig("http://localhost:8080")
	client := NewClient(cfg)
	adapter := &mcpToolAdapter{
		info: MCPToolInfo{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		client: client,
	}

	ctx := context.Background()
	input := json.RawMessage(`{invalid json}`)
	result, err := adapter.Execute(ctx, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsError {
		t.Error("expected IsError to be true for invalid JSON")
	}
	if result.Content == "" {
		t.Error("expected error content to be non-empty")
	}
}

// TestMcpToolAdapterExecuteMCPFailure 测试 MCP 工具适配器执行失败。
func TestMcpToolAdapterExecuteMCPFailure(t *testing.T) {
	server := newTestBridgeErrorServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)
	adapter := &mcpToolAdapter{
		info: MCPToolInfo{
			Name:        "error_tool",
			Description: "A tool that errors",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		client: client,
	}

	ctx := context.Background()
	input := json.RawMessage(`{"arg":"value"}`)
	result, err := adapter.Execute(ctx, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsError {
		t.Error("expected IsError to be true for MCP call failure")
	}
	if result.Content == "" {
		t.Error("expected error content to be non-empty")
	}
}

// TestMcpToolAdapterExecuteWithMultipleContentItems 测试 MCP 工具适配器执行多个内容项。
func TestMcpToolAdapterExecuteWithMultipleContentItems(t *testing.T) {
	// Custom server that returns multiple content items
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
					"tools": [{
						"name": "multi_tool",
						"description": "Returns multiple content items",
						"inputSchema": {"type": "object"}
					}]
				}`),
			})
		case "tools/call":
			json.NewEncoder(w).Encode(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`{
					"content": [
						{"type": "text", "text": "first"},
						{"type": "text", "text": "second"},
						{"type": "text", "text": "third"}
					],
					"isError": false
				}`),
			})
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)
	adapter := &mcpToolAdapter{
		info: MCPToolInfo{
			Name:        "multi_tool",
			Description: "Returns multiple content items",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		client: client,
	}

	ctx := context.Background()
	input := json.RawMessage(`{}`)
	result, err := adapter.Execute(ctx, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Error("expected IsError to be false")
	}
	if result.Content != "firstsecondthird" {
		t.Errorf("expected concatenated content 'firstsecondthird', got %q", result.Content)
	}
}

// TestParsePropertiesNilSchema 测试解析空属性模式。
func TestParsePropertiesNilSchema(t *testing.T) {
	props := parseProperties(nil)
	if props != nil {
		t.Errorf("expected nil for nil schema, got %v", props)
	}
}

// TestParsePropertiesEmptySchema 测试解析空模式属性。
func TestParsePropertiesEmptySchema(t *testing.T) {
	schema := json.RawMessage(`{}`)
	props := parseProperties(schema)

	if props != nil {
		t.Errorf("expected nil for empty schema, got %v", props)
	}
}

// TestParsePropertiesWithProperties 测试解析带属性的属性。
func TestParsePropertiesWithProperties(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "User name"
			},
			"age": {
				"type": "integer",
				"description": "User age"
			},
			"active": {
				"type": "boolean"
			}
		},
		"required": ["name"]
	}`)

	props := parseProperties(schema)

	if len(props) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(props))
	}

	name, ok := props["name"]
	if !ok {
		t.Fatal("expected 'name' property")
	}
	if name.Type != "string" {
		t.Errorf("expected name type 'string', got %q", name.Type)
	}
	if name.Description != "User name" {
		t.Errorf("expected name description 'User name', got %q", name.Description)
	}

	age, ok := props["age"]
	if !ok {
		t.Fatal("expected 'age' property")
	}
	if age.Type != "integer" {
		t.Errorf("expected age type 'integer', got %q", age.Type)
	}

	active, ok := props["active"]
	if !ok {
		t.Fatal("expected 'active' property")
	}
	if active.Type != "boolean" {
		t.Errorf("expected active type 'boolean', got %q", active.Type)
	}
	if active.Description != "" {
		t.Errorf("expected empty description for active, got %q", active.Description)
	}
}

// TestBridgeImplementsToolInterface 测试桥接器实现工具接口。
func TestBridgeImplementsToolInterface(t *testing.T) {
	server := newTestBridgeServer()
	defer server.Close()

	cfg := DefaultConfig(server.URL)
	client := NewClient(cfg)
	bridge := NewBridge(client)

	ctx := context.Background()
	tools, err := bridge.DiscoverAndConvert(ctx)
	if err != nil {
		t.Fatalf("failed to discover tools: %v", err)
	}

	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}

	// Verify each tool implements the tool.Tool interface
	for _, ttool := range tools {
		// This should compile and not panic if the interface is correctly implemented
		info := ttool.Info()
		if info.Name == "" {
			t.Error("expected tool to have a name")
		}
		_, err := ttool.Execute(context.Background(), json.RawMessage(`{}`))
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
	}
}