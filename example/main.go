// Package main 演示 bamboo-agent 框架的 Agent + SubAgent 协作模式。
//
// 本示例展示 "Agent as Tool" 模式：主 Agent 通过工具调用激活 SubAgent，
// SubAgent 独立执行子任务（使用自己的工具集），并将结果返回给主 Agent。
//
// 运行方式：
//
//	go run ./example/
//
// 无需任何外部依赖或 API Key，使用内置 Mock 客户端模拟完整的 AI 交互。
// 如需连接真实 AI 服务，参见底部 createRealClient() 的注释。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bamboo-services/bamboo-agent/agent"
	"github.com/bamboo-services/bamboo-agent/orchestrator"
	"github.com/bamboo-services/bamboo-agent/tool"
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// ---------------------------------------------------------------------------
// 自定义工具：搜索引擎模拟
// ---------------------------------------------------------------------------

// searchTool 模拟一个搜索引擎工具。
//
// SubAgent 使用此工具进行信息检索。
// 在真实场景中，可替换为实际的搜索 API、数据库查询等。
type searchTool struct{}

func (t *searchTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "search",
		Description: "搜索引擎，用于检索信息。输入搜索关键词，返回相关结果。",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"query": {
					Type:        "string",
					Description: "搜索关键词",
				},
			},
			Required: []string{"query"},
		},
	}
}

func (t *searchTool) Execute(_ context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("search tool: parse input: %w", err)
	}

	// 模拟搜索结果 —— 真实场景中这里调用实际搜索 API
	mockResults := map[string]string{
		"Go":         "Go 是 Google 开发的开源编程语言，以简洁、高效和并发支持著称。当前最新版本支持泛型。",
		"AI Agent":   "AI Agent 是能够自主感知环境、做出决策并执行动作的智能体。核心能力包括工具调用、自我迭代和多 Agent 协作。",
		"bamboo":     "Bamboo Agent 是基于 BambooMessages SDK 的 Go 语言 AI Agent 框架，支持 ReAct 循环、并发工具执行和多 Agent 编排。",
	}

	if result, ok := mockResults[params.Query]; ok {
		return &tool.ToolResult{Content: result}, nil
	}
	return &tool.ToolResult{
		Content: fmt.Sprintf("搜索 %q 的结果：找到了一些相关信息，但需要进一步分析。", params.Query),
	}, nil
}

// ---------------------------------------------------------------------------
// 自定义工具：计算器
// ---------------------------------------------------------------------------

// calcTool 简单计算器工具，供主 Agent 直接使用。
type calcTool struct{}

func (t *calcTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "calculator",
		Description: "简单计算器，计算数学表达式",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"expression": {
					Type:        "string",
					Description: "数学表达式（如 2+3*4）",
				},
			},
			Required: []string{"expression"},
		},
	}
}

func (t *calcTool) Execute(_ context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var params struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("calc tool: parse input: %w", err)
	}

	// 模拟计算结果
	return &tool.ToolResult{
		Content: fmt.Sprintf("%s = 42（模拟结果）", params.Expression),
	}, nil
}

// ---------------------------------------------------------------------------
// Mock 客户端工厂
// ---------------------------------------------------------------------------

// newSubAgentMockClient 创建 SubAgent 的 Mock 客户端。
//
// 模拟行为：
//   - 第 1 轮：返回 tool_use（调用 search 工具）
//   - 第 2 轮：返回文本（综合搜索结果给出回答）
func newSubAgentMockClient() *mockBambooClient {
	callCount := 0
	return &mockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			callCount++
			ch := make(chan bamboo.StreamEvent, 20)
			go func() {
				defer close(ch)
				switch callCount {
				case 1:
					// 第一轮：调用 search 工具
					pushToolUseResponse(ch, "sub_call_1", "search", `{"query":"AI Agent"}`)
				default:
					// 第二轮：综合结果返回文本
					pushTextResponse(ch, "根据搜索结果，AI Agent 是能够自主感知环境、做出决策并执行动作的智能体。核心能力包括工具调用、自我迭代和多 Agent 协作。Bamboo Agent 是一个 Go 语言实现的框架，支持 ReAct 循环和并发工具执行。")
				}
			}()
			return ch, nil
		},
	}
}

// newMainAgentMockClient 创建主 Agent 的 Mock 客户端。
//
// 模拟行为：
//   - 第 1 轮：返回 tool_use（委派任务给 researcher SubAgent）
//   - 第 2 轮：返回文本（综合 SubAgent 的研究结果给出最终回答）
func newMainAgentMockClient() *mockBambooClient {
	callCount := 0
	return &mockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			callCount++
			ch := make(chan bamboo.StreamEvent, 20)
			go func() {
				defer close(ch)
				switch callCount {
				case 1:
					// 第一轮：委派给 SubAgent
					pushToolUseResponse(ch, "main_call_1", "delegate_researcher", `{"task":"调研 AI Agent 的核心概念和 Bamboo Agent 框架特点"}`)
				default:
					// 第二轮：综合 SubAgent 结果给出最终回答
					pushTextResponse(ch, "经过我的调研员 Agent 的深入分析，以下是关于 AI Agent 的综合报告：\n\n**AI Agent 核心**：AI Agent 是能够自主感知、决策和执行的智能体。\n\n**Bamboo Agent 特点**：基于 Go 语言，支持 ReAct 循环、并发工具执行和多 Agent 编排。\n\n**SubAgent 模式**：通过 Agent-as-Tool 模式，主 Agent 可以将复杂子任务委派给专门的 Agent 处理。\n\n报告完毕！")
				}
			}()
			return ch, nil
		},
	}
}

// ---------------------------------------------------------------------------
// 真实客户端工厂（预留）
// ---------------------------------------------------------------------------

// createRealClient 创建连接真实 AI 服务的 BambooClient。
//
// 使用方式：取消注释并在环境变量中配置 API Key 后即可切换到真实模式。
// 需要导入 bamboo-messages 的 provider 包：
//
//	import "github.com/bamboo-services/bamboo-messages/internal/provider/anthropic"
//
//	func createRealClient() bamboo.BambooClient {
//	    apiKey := os.Getenv("ANTHROPIC_API_KEY")
//	    if apiKey == "" {
//	        panic("请设置 ANTHROPIC_API_KEY 环境变量")
//	    }
//	    p := anthropic.NewProvider(apiKey, "claude-sonnet-4-20250514")
//	    return bamboo.NewClient(p)
//	}
//
// 也可使用 OpenAI provider：
//
//	import "github.com/bamboo-services/bamboo-messages/internal/provider/openai/completions"
//
//	func createRealClient() bamboo.BambooClient {
//	    apiKey := os.Getenv("OPENAI_API_KEY")
//	    p := completions.NewProvider(apiKey, "gpt-4o")
//	    return bamboo.NewClient(p)
//	}

// ---------------------------------------------------------------------------
// 主函数
// ---------------------------------------------------------------------------

func main() {
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║      Bamboo Agent — Agent + SubAgent 协作示例           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ======================================================================
	// 步骤 1: 创建 SubAgent（调研员 Agent）
	// ======================================================================
	fmt.Println("📋 步骤 1: 创建 SubAgent（调研员 Agent）")
	fmt.Println("─────────────────────────────────────────")

	// SubAgent 拥有自己的工具集和 Mock 客户端
	subAgent := orchestrator.NewAgentBuilder().
		WithClient(newSubAgentMockClient()).
		WithSystemPrompt("你是一个专业的调研员。使用 search 工具搜索信息，然后综合分析给出详细回答。").
		WithConfig(agent.AgentConfig{
			MaxTokens:     4096,
			MaxIterations: 5,
		}).
		WithTools(&searchTool{}).
		Build()

	fmt.Println("  ✓ SubAgent 创建完成，已注册工具: search")
	fmt.Println()

	// ======================================================================
	// 步骤 2: 将 SubAgent 包装为 Tool
	// ======================================================================
	fmt.Println("📋 步骤 2: 将 SubAgent 包装为 Tool")
	fmt.Println("─────────────────────────────────────────")

	researcherTool := NewSubAgentTool(
		"delegate_researcher",
		"将调研任务委派给专业的调研员 Agent。调研员会使用搜索引擎检索信息并给出详细分析报告。",
		subAgent,
	)

	fmt.Println("  ✓ SubAgentTool 创建完成")
	fmt.Println("    - 工具名称: delegate_researcher")
	fmt.Println("    - 输入参数: task (string) — 调研任务描述")
	fmt.Println()

	// ======================================================================
	// 步骤 3: 创建主 Agent（协调者 Agent）
	// ======================================================================
	fmt.Println("📋 步骤 3: 创建主 Agent（协调者 Agent）")
	fmt.Println("─────────────────────────────────────────")

	// 主 Agent 同时拥有直接使用的工具和 SubAgent 工具
	mainAgent := orchestrator.NewAgentBuilder().
		WithClient(newMainAgentMockClient()).
		WithSystemPrompt("你是一个协调者 Agent。你可以直接使用工具完成任务，也可以将复杂调研任务委派给调研员 Agent。").
		WithConfig(agent.AgentConfig{
			MaxTokens:     4096,
			MaxIterations: 10,
		}).
		WithTools(
			&calcTool{},        // 主 Agent 直接使用的工具
			researcherTool,     // SubAgent 包装的工具
		).
		Build()

	fmt.Println("  ✓ 主 Agent 创建完成，已注册工具: calculator, delegate_researcher")
	fmt.Println()

	// ======================================================================
	// 步骤 4: 运行主 Agent
	// ======================================================================
	fmt.Println("📋 步骤 4: 运行主 Agent")
	fmt.Println("─────────────────────────────────────────")
	userInput := "请帮我调研一下 AI Agent 的核心概念，特别是 Bamboo Agent 框架的特点"
	fmt.Printf("  👤 用户输入: %q\n\n", userInput)
	fmt.Println("  ─── Agent 执行开始 ───")

	start := time.Now()
	result, err := mainAgent.Run(ctx, userInput)
	elapsed := time.Since(start)

	fmt.Println("  ─── Agent 执行结束 ───")
	fmt.Println()

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Agent 执行失败: %v\n", err)
		os.Exit(1)
	}

	// ======================================================================
	// 步骤 5: 展示结果
	// ======================================================================
	fmt.Println("📋 步骤 5: 执行结果")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("  ⏱️  耗时: %v\n", elapsed)
	fmt.Printf("  🔄 迭代次数: %d\n", result.Iterations)
	fmt.Printf("  🔧 工具调用: %d 次\n", len(result.ToolCalls))
	fmt.Printf("  📊 Token 使用: 输入=%d, 输出=%d\n", result.Usage.InputTokens, result.Usage.OutputTokens)
	fmt.Println()

	if len(result.ToolCalls) > 0 {
		fmt.Println("  📝 工具调用详情:")
		for i, tc := range result.ToolCalls {
			status := "✅"
			if tc.IsError {
				status = "❌"
			}
			fmt.Printf("    %s [%d] %s\n", status, i+1, tc.Name)
			fmt.Printf("       输入: %s\n", truncate(string(tc.Input), 80))
			fmt.Printf("       结果: %s\n", truncate(tc.Result, 80))
		}
		fmt.Println()
	}

	fmt.Println("  🤖 Agent 最终回复:")
	fmt.Println("  ┌──────────────────────────────────────")
	for _, line := range splitLines(result.Content) {
		fmt.Printf("  │ %s\n", line)
	}
	fmt.Println("  └──────────────────────────────────────")
	fmt.Println()

	// ======================================================================
	// 附加：展示 Orchestrator 编排模式
	// ======================================================================
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║      附加演示: Orchestrator 多 Agent 编排               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	demoOrchestrator(ctx)

	fmt.Println()
	fmt.Println("✨ 示例运行完毕！")
}

// demoOrchestrator 展示 Orchestrator 的多 Agent 编排能力。
//
// 与上面的 "Agent as Tool" 模式不同，Orchestrator 是外部编排模式：
//   - Agent 本身不知道其他 Agent 的存在
//   - 通过依赖图控制执行顺序和并行度
//   - 适合流水线式的工作流
func demoOrchestrator(ctx context.Context) {
	orch := orchestrator.NewOrchestrator()

	// 注册三个独立的 Agent
	orch.RegisterAgent("researcher", orchestrator.NewAgentBuilder().
		WithClient(newSubAgentMockClient()).
		WithSystemPrompt("你是调研员").
		WithConfig(agent.AgentConfig{MaxTokens: 2048, MaxIterations: 3}).
		WithTools(&searchTool{}).
		Build(),
	)

	// 分析师 Agent 使用独立的 Mock
	analystCallCount := 0
	analystClient := &mockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			analystCallCount++
			ch := make(chan bamboo.StreamEvent, 10)
			go func() {
				defer close(ch)
				pushTextResponse(ch, fmt.Sprintf("分析完成（第 %d 轮）：基于调研数据，AI Agent 市场正在快速增长。", analystCallCount))
			}()
			return ch, nil
		},
	}
	orch.RegisterAgent("analyst", orchestrator.NewAgentBuilder().
		WithClient(analystClient).
		WithSystemPrompt("你是数据分析师").
		WithConfig(agent.AgentConfig{MaxTokens: 2048, MaxIterations: 3}).
		Build(),
	)

	// 定义任务依赖图
	//
	// researcher ──→ analyst（依赖 researcher 完成后才开始）
	//
	// 通过注册表 ID 匹配对应的 Agent 实例。
	tasks := []orchestrator.Task{
		{
			ID:          "researcher",
			Description: "调研 AI Agent 概念",
			Input:       "调研 AI Agent 的核心概念",
		},
		{
			ID:          "analyst",
			Description: "分析调研结果",
			Input:       "分析 AI Agent 市场趋势",
			DependsOn:   []string{"researcher"},
		},
	}

	fmt.Println("  📊 任务依赖图:")
	fmt.Println("     researcher ──→ analyst")
	fmt.Println()

	results, err := orch.Execute(ctx, tasks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ❌ Orchestrator 执行失败: %v\n", err)
		return
	}

	fmt.Println("  📊 执行结果:")
	for _, r := range results {
		icon := "✅"
		if r.Status == orchestrator.TaskFailed {
			icon = "❌"
		}
		content := ""
		if r.Result != nil {
			content = truncate(r.Result.Content, 60)
		}
		if r.Error != nil {
			content = r.Error.Error()
		}
		fmt.Printf("    %s [%s] %s → %s\n", icon, r.Status, r.ID, content)
	}
}

// ---------------------------------------------------------------------------
// 工具函数
// ---------------------------------------------------------------------------

// truncate 截断过长的字符串。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// splitLines 将字符串按换行符分割。
func splitLines(s string) []string {
	if s == "" {
		return []string{"(空)"}
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
