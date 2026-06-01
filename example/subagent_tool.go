package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bamboo-services/bamboo-agent/agent"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// SubAgentTool 将 Agent 包装为 Tool，使主 Agent 可以通过工具调用激活 SubAgent。
//
// 这是 "Agent as Tool" 模式的核心实现：
//   - 实现 tool.Tool 接口，可注册到任何 Agent 的工具表中
//   - Execute() 内部调用 SubAgent 的 Run() 方法
//   - SubAgent 的执行结果（AgentResult.Content）作为工具返回值
//
// 使用场景：主 Agent 需要将复杂子任务委派给专门的 Agent 处理，
// 例如：调研员 Agent、代码审查 Agent、翻译 Agent 等。
type SubAgentTool struct {
	// name 是工具名称，主 Agent 通过此名称调用 SubAgent。
	name string

	// description 是工具描述，告诉主 Agent 何时应该调用此工具。
	description string

	// subAgent 是被包装的 Agent 实例。
	subAgent agent.Agent
}

// NewSubAgentTool 创建一个新的 SubAgentTool。
//
// 参数说明：
//   - name: 工具名称（如 "delegate_researcher"）
//   - description: 工具描述（告诉主 Agent 此工具的用途）
//   - subAgent: 要包装的 Agent 实例
func NewSubAgentTool(name, description string, subAgent agent.Agent) *SubAgentTool {
	return &SubAgentTool{
		name:        name,
		description: description,
		subAgent:    subAgent,
	}
}

// subAgentInput 定义 SubAgentTool 的输入参数。
type subAgentInput struct {
	// Task 是传递给 SubAgent 的任务描述。
	Task string `json:"task"`
}

// Info 返回工具元信息。
//
// 定义了 "task" 参数，主 Agent 通过此参数传递具体子任务。
func (t *SubAgentTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        t.name,
		Description: t.description,
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"task": {
					Type:        "string",
					Description: "需要委派给子 Agent 执行的任务描述",
				},
			},
			Required: []string{"task"},
		},
	}
}

// Execute 执行工具逻辑 —— 激活 SubAgent 处理子任务。
//
// 流程：
//  1. 解析输入参数中的 "task" 字段
//  2. 调用 SubAgent.Run() 执行子任务
//  3. 将 SubAgent 的输出作为工具结果返回
//
// 如果 SubAgent 执行失败，返回 isError=true 的 ToolResult，
// 主 Agent 的 ReAct 循环会将此作为错误信息继续迭代。
func (t *SubAgentTool) Execute(ctx context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var params subAgentInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("subagent tool: parse input failed: %w", err)
	}

	if params.Task == "" {
		return &tool.ToolResult{
			Content: "错误：task 参数不能为空",
			IsError: true,
		}, nil
	}

	fmt.Printf("  🔄 [SubAgentTool] 激活 SubAgent: %q\n", params.Task)

	result, err := t.subAgent.Run(ctx, params.Task)
	if err != nil {
		return &tool.ToolResult{
			Content: fmt.Sprintf("SubAgent 执行失败: %v", err),
			IsError: true,
		}, nil
	}

	fmt.Printf("  ✅ [SubAgentTool] SubAgent 完成，迭代 %d 次\n", result.Iterations)

	return &tool.ToolResult{
		Content: result.Content,
		IsError: false,
	}, nil
}
