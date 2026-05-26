package tool

import (
	"context"
	"encoding/json"
)

// Tool 定义了所有工具必须实现的接口。
//
// 定义了工具的完整能力，包括：
//   - Info - 获取工具的元信息
//   - Execute - 执行工具逻辑
type Tool interface {
	// Info 返回工具的元信息，包括名称、描述和参数定义。
	//
	// 返回：
	//   - ToolInfo - 工具的元信息
	Info() ToolInfo

	// Execute 执行工具并返回执行结果或错误。
	//
	// 参数说明：
	//   - ctx - 上下文，用于取消和超时控制
	//   - input - 工具输入参数（JSON 格式）
	//
	// 返回：
	//   - *ToolResult - 工具执行结果
	//   - error - 执行错误，如参数解析失败或执行异常
	Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}

// ToolInfo 保存工具的元信息。
//
// 包含工具的名称、描述和输入参数定义。
type ToolInfo struct {
	// Name 是工具的唯一标识符。
	Name string `json:"name"`

	// Description 是工具功能的中文描述。
	Description string `json:"description"`

	// Parameters 定义工具的输入参数 schema。
	Parameters InputSchema `json:"parameters"`
}

// ToolResult 保存工具执行的结果。
//
// 包含执行结果内容和错误状态标识。
type ToolResult struct {
	// Content 是工具执行结果的文本内容。
	Content string `json:"content"`

	// IsError 标识本次执行是否为错误结果。
	IsError bool `json:"isError"`
}