// Package orchestrator 提供多 Agent 编排能力。
//
// 支持依赖图分析、任务并行/串行执行和消息传递，使用 Builder 模式构建 Agent 实例。
//
// 核心组件：
//   - Orchestrator — 多 Agent 任务编排器，支持依赖解析、失败级联、并行/串行执行
//   - AgentBuilder — 使用 Builder 模式流畅构建 Agent 实例
//   - Task — 任务抽象，包含 ID、描述、Agent、输入、依赖、状态、结果和错误
//   - TaskStatus — 任务状态枚举（pending、running、completed、failed）
//   - Channel — Agent 间消息传递通道，支持订阅、发送、广播
//   - AgentMessage — Agent 间消息结构，包含发送方、接收方、内容和数据
//
// 使用示例：
//
//	orch := orchestrator.NewOrchestrator()
//	orch.RegisterAgent("researcher", researchAgent)
//	orch.RegisterAgent("writer", writerAgent)
//
//	tasks := []orchestrator.Task{
//	    {ID: "research", Description: "调研主题", Agent: researchAgent, Input: "AI Agent 框架现状"},
//	    {ID: "write", Description: "撰写报告", Agent: writerAgent, Input: "根据调研写报告", DependsOn: []string{"research"}},
//	}
//
//	results, err := orch.Execute(ctx, tasks)
package orchestrator
