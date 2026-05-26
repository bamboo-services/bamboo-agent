package orchestrator

import (
	"github.com/bamboo-services/bamboo-agent/agent"
)

// TaskStatus 表示任务的当前执行状态。
type TaskStatus string

const (
	// TaskPending 任务待执行
	TaskPending TaskStatus = "pending"

	// TaskRunning 任务执行中
	TaskRunning TaskStatus = "running"

	// TaskCompleted 任务已完成
	TaskCompleted TaskStatus = "completed"

	// TaskFailed 任务执行失败
	TaskFailed TaskStatus = "failed"
)

// Task 表示由 Agent 执行的工作单元。
//
// 包含任务标识、描述、执行 Agent、输入内容、依赖关系、执行状态和结果等信息。
type Task struct {
	// ID 是任务唯一标识符。
	ID string

	// Description 是任务的人类可读描述。
	Description string

	// Agent 是执行此任务的 Agent 实例。
	Agent agent.Agent

	// Input 是传递给 Agent 的输入文本。
	Input string

	// DependsOn 列出此任务开始前必须完成的其他任务 ID。
	DependsOn []string

	// Status 跟踪当前执行状态。
	Status TaskStatus

	// Result 保存执行后的 Agent 结果。
	Result *agent.AgentResult

	// Error 保存执行过程中的错误。
	Error error
}