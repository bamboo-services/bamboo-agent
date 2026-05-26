package tool

import (
	"context"
	"encoding/json"
)

// Tool 接口定义了所有工具必须实现的方法
type Tool interface {
	// Info 返回工具的元信息
	Info() ToolInfo
	// Execute 执行工具，返回执行结果或错误
	Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}

// ToolInfo 包含工具的元信息
type ToolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  InputSchema `json:"parameters"`
}

// ToolResult 包含工具执行的结果
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}