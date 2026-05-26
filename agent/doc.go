// Package agent 提供 AI Agent 的核心能力。
//
// 包含以下核心组件：
//   - Agent — 定义 Agent 生命周期（Run / Stream / RunWithMessages）
//   - Session — 内存会话管理，维护对话历史
//   - ReActLoop — Reason-Act 迭代策略
//   - ContextCompressor — 长对话自动压缩
//   - AgentConfig — 配置参数
//   - Option — Functional Options 配置模式
//
// 基本用法：
//
//	ag := agent.NewAgent(client, agent.DefaultConfig())
//	result, err := ag.Run(ctx, "你好")
package agent