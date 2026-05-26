package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

// Client is an MCP protocol client.
type Client struct {
	config     Config
	httpClient *http.Client
	nextID     atomic.Int64
	connected  bool
}

// NewClient creates a new MCP client.
func NewClient(config Config) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Connect verifies the MCP server is reachable.
func (c *Client) Connect(ctx context.Context) error {
	// Send an initialize request to verify connectivity
	_, err := c.call(ctx, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "bamboo-agent",
			"version": "0.1.0",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	c.connected = true
	return nil
}

// DiscoverTools calls tools/list to get available tools.
func (c *Client) DiscoverTools(ctx context.Context) ([]MCPToolInfo, error) {
	resp, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools list: %w", err)
	}

	return result.Tools, nil
}

// CallTool calls a specific tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*MCPToolResult, error) {
	resp, err := c.call(ctx, "tools/call", ToolsCallParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", name, err)
	}

	var result MCPToolResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	return &result, nil
}

// Close disconnects from the MCP server.
func (c *Client) Close() error {
	c.connected = false
	c.httpClient.CloseIdleConnections()
	return nil
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	return c.connected
}

// call sends a JSON-RPC request.
func (c *Client) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := int(c.nextID.Add(1))

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.ServerURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range c.config.Headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error [%d]: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
