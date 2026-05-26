// Package tool 提供了 Bamboo Agent 框架的工具系统核心组件
//
// 工具系统是 AI Agent 执行外部操作的核心能力,支持工具注册、并发执行和适配器转换。
//
// 主要组件:
//   - Tool 接口 - 定义所有工具必须实现的基本方法(Info、Execute)
//   - InputSchema - 工具参数的输入模式定义(Type、Properties、Required)
//   - Registry - 线程安全的工具注册表,支持工具注册、查找和批量注册
//   - ToolExecutor - 并发执行器,支持多工具并发调用和并发控制
//   - BambooAdapter - 适配器,将内部工具类型转换为 Bamboo SDK 类型
//
// 使用示例:
//   registry := tool.NewRegistry()
//   registry.Register(&MyTool{})
//   executor := tool.NewToolExecutor(registry, 10)
//   results := executor.ExecuteAll(ctx, toolCalls)
package tool