package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bamboo-services/bamboo-agent/tool"
)

// Bridge 连接 MCP 服务器工具到 agent 工具系统。
//
// 通过 DiscoverAndConvert 方法自动发现 MCP 服务器提供的工具，
// 并将其转换为 tool.Tool 接口实现，供 Agent 使用。
//
// 使用示例：
//
//	mcpClient := mcp.NewClient(mcp.DefaultConfig("http://localhost:8080"))
//	mcpClient.Connect(ctx)
//
//	bridge := mcp.NewBridge(mcpClient)
//	mcpTools, _ := bridge.DiscoverAndConvert(ctx)
//
//	ag := orchestrator.NewAgentBuilder().
//		WithClient(bmClient).
//		WithTools(mcpTools...).
//		Build()
type Bridge struct {
	client *Client
	tools  []MCPToolInfo
}

// NewBridge 创建一个新的 Bridge 实例。
//
// 用于连接指定的 MCP 客户端，后续可以调用 DiscoverAndConvert 发现并转换工具。
//
// 参数说明：
//   - client - MCP 客户端实例
//
// 返回：
//   - *Bridge - 新创建的 Bridge 实例
func NewBridge(client *Client) *Bridge {
	return &Bridge{client: client}
}

// DiscoverAndConvert 从 MCP 服务器发现工具并转换为 tool.Tool 接口。
//
// 调用 MCP 客户端的 DiscoverTools 方法获取工具列表，
// 然后将每个 MCP 工具包装为 mcpToolAdapter 实现的 tool.Tool 接口。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//
// 返回：
//   - []tool.Tool - 转换后的工具列表
//   - error - 发现工具失败时返回错误
func (b *Bridge) DiscoverAndConvert(ctx context.Context) ([]tool.Tool, error) {
	tools, err := b.client.DiscoverTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover MCP tools: %w", err)
	}
	b.tools = tools

	result := make([]tool.Tool, len(tools))
	for i, t := range tools {
		result[i] = &mcpToolAdapter{
			info:   t,
			client: b.client,
		}
	}
	return result, nil
}

// AsTools 返回所有已发现的 MCP 工具作为 tool.Tool 接口。
//
// 注意：必须先调用 DiscoverAndConvert 方法发现工具，否则返回空列表。
// 每次调用都会重新创建工具适配器实例。
//
// 返回：
//   - []tool.Tool - 工具列表，如果未调用 DiscoverAndConvert 则为空
func (b *Bridge) AsTools() []tool.Tool {
	result := make([]tool.Tool, len(b.tools))
	for i, t := range b.tools {
		result[i] = &mcpToolAdapter{
			info:   t,
			client: b.client,
		}
	}
	return result
}

// mcpToolAdapter 将 MCP 工具适配为 tool.Tool 接口。
//
// 实现了 tool.Tool 接口的 Info 和 Execute 方法，将 MCP 工具的元信息和调用
// 转换为 agent 工具系统的标准格式。
type mcpToolAdapter struct {
	info   MCPToolInfo // MCP 工具信息
	client *Client     // MCP 客户端，用于调用工具
}

// Info 返回工具的元信息。
//
// 将 MCP 工具的信息转换为 tool.ToolInfo 格式，包括名称、描述和参数定义。
// 参数定义通过 parseProperties 从 MCP 输入 schema 中提取。
//
// 返回：
//   - tool.ToolInfo - 工具元信息
func (a *mcpToolAdapter) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        a.info.Name,
		Description: a.info.Description,
		Parameters: tool.InputSchema{
			Type:       "object",
			Properties: parseProperties(a.info.InputSchema),
		},
	}
}

// Execute 调用 MCP 工具并返回执行结果。
//
// 将输入参数解析为 map[string]interface{} 格式，然后调用 MCP 客户端的 CallTool 方法。
// 执行结果中的文本类型内容会被合并返回。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - input - 工具输入参数（JSON 格式）
//
// 返回：
//   - *tool.ToolResult - 工具执行结果
//   - error - 参数解析失败时返回错误
func (a *mcpToolAdapter) Execute(ctx context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		return &tool.ToolResult{
			Content: fmt.Sprintf("invalid input: %v", err),
			IsError: true,
		}, nil
	}

	result, err := a.client.CallTool(ctx, a.info.Name, args)
	if err != nil {
		return &tool.ToolResult{
			Content: fmt.Sprintf("MCP tool call failed: %v", err),
			IsError: true,
		}, nil
	}

	// Combine content texts
	var content string
	for _, c := range result.Content {
		if c.Type == "text" && c.Text != "" {
			content += c.Text
		}
	}

	return &tool.ToolResult{
		Content: content,
		IsError: result.IsError,
	}, nil
}

// parseProperties 从 MCP 输入 schema 中提取属性定义。
//
// 解析 JSON 格式的 schema，提取 type、description 等字段，
// 转换为 tool.PropertyDef 格式。
//
// 参数说明：
//   - schema - MCP 工具的输入参数 schema（JSON 格式）
//
// 返回：
//   - map[string]tool.PropertyDef - 属性定义映射，key 为属性名
func parseProperties(schema json.RawMessage) map[string]tool.PropertyDef {
	if schema == nil {
		return nil
	}

	var raw struct {
		Type       string                            `json:"type"`
		Properties map[string]map[string]interface{} `json:"properties"`
		Required   []string                          `json:"required"`
	}
	if err := json.Unmarshal(schema, &raw); err != nil {
		return nil
	}

	// Return nil if no properties defined
	if raw.Properties == nil {
		return nil
	}

	props := make(map[string]tool.PropertyDef)
	for name, def := range raw.Properties {
		pd := tool.PropertyDef{}
		if t, ok := def["type"].(string); ok {
			pd.Type = t
		}
		if d, ok := def["description"].(string); ok {
			pd.Description = d
		}
		props[name] = pd
	}

	return props
}