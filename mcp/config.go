package mcp

import (
	"encoding/json"
	"time"
)

// Config 保存 MCP 客户端配置。
//
// 包含以下核心字段：
//   - ServerURL - MCP 服务器基础 URL
//   - Timeout - 请求超时时间
//   - Headers - 自定义 HTTP 请求头
type Config struct {
	// ServerURL 是 MCP 服务器的基础 URL。
	ServerURL string

	// Timeout 是请求超时时间。
	Timeout time.Duration

	// Headers 是随请求发送的自定义 HTTP 请求头。
	Headers map[string]string
}

// DefaultConfig 返回一个默认的 MCP 配置。
//
// 使用默认的 30 秒超时时间和空的请求头。
//
// 参数说明：
//   - serverURL - MCP 服务器的基础 URL
//
// 返回：
//   - Config - 默认配置实例
func DefaultConfig(serverURL string) Config {
	return Config{
		ServerURL: serverURL,
		Timeout:   30 * time.Second,
		Headers:   make(map[string]string),
	}
}

// MCPContent 表示 MCP 响应中的内容块。
//
// 包含以下字段：
//   - Type - 内容类型（text、image、resource 等）
//   - Text - 文本内容（可选）
//   - Data - 原始数据（可选）
type MCPContent struct {
	Type string          `json:"type"`

	Text string          `json:"text,omitempty"`

	Data json.RawMessage `json:"data,omitempty"`
}

// MCPToolInfo 表示从 MCP 服务器发现的工具信息。
//
// 包含工具的名称、描述和输入参数定义。
type MCPToolInfo struct {
	Name        string          `json:"name"`

	Description string          `json:"description"`

	InputSchema json.RawMessage `json:"inputSchema"`
}

// MCPToolResult 表示调用 MCP 工具的结果。
//
// 包含工具执行返回的内容和是否为错误状态。
type MCPToolResult struct {
	Content []MCPContent `json:"content"`

	IsError bool         `json:"isError"`
}

// JSONRPCRequest 表示 JSON-RPC 2.0 请求。
//
// 包含协议版本、请求 ID、方法名和参数。
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`

	ID      int         `json:"id"`

	Method  string      `json:"method"`

	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse 表示 JSON-RPC 2.0 响应。
//
// 包含协议版本、请求 ID、结果或错误。
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`

	ID      int             `json:"id"`

	Result  json.RawMessage `json:"result,omitempty"`

	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError 表示 JSON-RPC 错误。
//
// 包含错误码、错误消息和可选的附加数据。
type JSONRPCError struct {
	Code    int         `json:"code"`

	Message string      `json:"message"`

	Data    interface{} `json:"data,omitempty"`
}

// ToolsListParams 是 tools/list 方法的参数。
//
// 目前为空，无需额外参数。
type ToolsListParams struct{}

// ToolsListResult 是 tools/list 方法的返回结果。
//
// 包含可用的工具列表。
type ToolsListResult struct {
	Tools []MCPToolInfo `json:"tools"`
}

// ToolsCallParams 是 tools/call 方法的参数。
//
// 包含要调用的工具名称和参数。
type ToolsCallParams struct {
	Name      string                 `json:"name"`

	Arguments map[string]interface{} `json:"arguments,omitempty"`
}