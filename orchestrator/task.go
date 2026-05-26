package orchestrator

import (
	"github.com/bamboo-services/bamboo-agent/agent"
)

// TaskStatus represents the current status of a task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// Task represents a unit of work to be executed by an Agent.
type Task struct {
	// ID is the unique identifier for this task.
	ID string
	// Description is a human-readable description of the task.
	Description string
	// Agent is the agent that will execute this task.
	Agent agent.Agent
	// Input is the input text for the agent.
	Input string
	// DependsOn lists task IDs that must complete before this task starts.
	DependsOn []string
	// Status tracks the current execution status.
	Status TaskStatus
	// Result holds the agent's result after execution.
	Result *agent.AgentResult
	// Error holds any error from execution.
	Error error
}