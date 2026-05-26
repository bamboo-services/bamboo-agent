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

// Client 是 MCP 协议客户端。
//
// 负责与 MCP 服务器通信，包括连接、工具发现、工具调用等功能。
type Client struct {
	config     Config
	httpClient *http.Client
	nextID     atomic.Int64
	connected  bool
}

// NewClient 创建一个新的 MCP 客户端。
//
// 使用提供的配置初始化客户端，包括服务器 URL 和超时设置。
//
// 参数说明：
//   - config - MCP 客户端配置
//
// 返回：
//   - *Client - 新创建的客户端实例
func NewClient(config Config) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Connect 验证 MCP 服务器是否可达。
//
// 发送一个 initialize 请求来验证连接，成功后标记为已连接状态。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//
// 返回：
//   - error - 连接失败时返回错误
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

// DiscoverTools 调用 tools/list 获取可用工具列表。
//
// 从 MCP 服务器查询所有可用的工具及其信息。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//
// 返回：
//   - []MCPToolInfo - 工具信息列表
//   - error - 查询失败时返回错误
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

// CallTool 调用 MCP 服务器上的特定工具。
//
// 执行指定的工具并返回执行结果。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - name - 工具名称
//   - arguments - 工具参数（可选）
//
// 返回：
//   - *MCPToolResult - 工具执行结果
//   - error - 调用失败时返回错误
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

// Close 断开与 MCP 服务器的连接。
//
// 清理 HTTP 客户端资源并标记为未连接状态。
//
// 返回：
//   - error - 目前总是返回 nil
func (c *Client) Close() error {
	c.connected = false
	c.httpClient.CloseIdleConnections()
	return nil
}

// IsConnected 返回客户端是否已连接。
//
// 返回：
//   - bool - true 表示已连接，false 表示未连接
func (c *Client) IsConnected() bool {
	return c.connected
}

// call 发送 JSON-RPC 请求。
//
// 内部方法，处理请求序列化、HTTP 发送、响应解析等底层逻辑。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - method - JSON-RPC 方法名
//   - params - 方法参数（可选）
//
// 返回：
//   - json.RawMessage - 响应结果
//   - error - 请求失败或 RPC 错误时返回错误
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