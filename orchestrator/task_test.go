package orchestrator

import (
	"context"
	"testing"

	"github.com/bamboo-services/bamboo-agent/agent"
	"github.com/bamboo-services/bamboo-agent/tool"
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/stretchr/testify/assert"
)

func TestTaskStatus_Values(t *testing.T) {
	// 测试 TaskStatus 常量的值
	assert.Equal(t, TaskStatus("pending"), TaskPending, "TaskPending should be 'pending'")
	assert.Equal(t, TaskStatus("running"), TaskRunning, "TaskRunning should be 'running'")
	assert.Equal(t, TaskStatus("completed"), TaskCompleted, "TaskCompleted should be 'completed'")
	assert.Equal(t, TaskStatus("failed"), TaskFailed, "TaskFailed should be 'failed'")
}

func TestTask_StructAssignment(t *testing.T) {
	// 创建一个模拟的 Agent
	mockAgent := &mockAgent{}

	// 创建一个 Task
	task := Task{
		ID:          "task-001",
		Description: "Test task description",
		Agent:       mockAgent,
		Input:       "Test input",
		DependsOn:   []string{"task-000"},
		Status:      TaskPending,
		Result:      nil,
		Error:       nil,
	}

	// 验证所有字段
	assert.Equal(t, "task-001", task.ID, "Task ID should match")
	assert.Equal(t, "Test task description", task.Description, "Task Description should match")
	assert.Equal(t, mockAgent, task.Agent, "Task Agent should match")
	assert.Equal(t, "Test input", task.Input, "Task Input should match")
	assert.Equal(t, []string{"task-000"}, task.DependsOn, "Task DependsOn should match")
	assert.Equal(t, TaskPending, task.Status, "Task Status should be TaskPending")
	assert.Nil(t, task.Result, "Task Result should be nil")
	assert.Nil(t, task.Error, "Task Error should be nil")
}

func TestTask_StatusUpdate(t *testing.T) {
	task := Task{
		ID:     "task-002",
		Status: TaskPending,
	}

	// 测试状态更新
	task.Status = TaskRunning
	assert.Equal(t, TaskRunning, task.Status, "Task status should update to TaskRunning")

	task.Status = TaskCompleted
	assert.Equal(t, TaskCompleted, task.Status, "Task status should update to TaskCompleted")

	task.Status = TaskFailed
	assert.Equal(t, TaskFailed, task.Status, "Task status should update to TaskFailed")
}

func TestTask_DependsOnMultiple(t *testing.T) {
	task := Task{
		ID:        "task-003",
		DependsOn: []string{"task-001", "task-002", "task-004"},
	}

	// 验证多个依赖项
	assert.Equal(t, 3, len(task.DependsOn), "Task should have 3 dependencies")
	assert.Contains(t, task.DependsOn, "task-001", "DependsOn should contain task-001")
	assert.Contains(t, task.DependsOn, "task-002", "DependsOn should contain task-002")
	assert.Contains(t, task.DependsOn, "task-004", "DependsOn should contain task-004")
}

func TestTask_EmptyDependsOn(t *testing.T) {
	task := Task{
		ID:        "task-004",
		DependsOn: []string{},
	}

	// 验证空依赖
	assert.Empty(t, task.DependsOn, "Task should have no dependencies")
	assert.Equal(t, 0, len(task.DependsOn), "DependsOn length should be 0")
}

func TestTask_WithResultAndError(t *testing.T) {
	mockResult := &agent.AgentResult{
		Content:    "Test result content",
		Iterations: 3,
	}
	task := Task{
		ID:     "task-005",
		Status: TaskCompleted,
		Result: mockResult,
		Error:  nil,
	}

	// 验证 Result
	assert.NotNil(t, task.Result, "Task Result should not be nil")
	assert.Equal(t, "Test result content", task.Result.Content, "Result content should match")
	assert.Equal(t, 3, task.Result.Iterations, "Result iterations should match")
	assert.Nil(t, task.Error, "Task Error should be nil")
}

func TestTask_WithError(t *testing.T) {
	expectedError := assert.AnError
	task := Task{
		ID:     "task-006",
		Status: TaskFailed,
		Result: nil,
		Error:  expectedError,
	}

	// 验证 Error
	assert.Nil(t, task.Result, "Task Result should be nil")
	assert.Equal(t, expectedError, task.Error, "Task Error should match")
}

// mockAgent 是一个用于测试的模拟 Agent 实现
type mockAgent struct{}

func (m *mockAgent) Run(ctx context.Context, input string) (*agent.AgentResult, error) {
	return &agent.AgentResult{
		Content:    "Mock result",
		Iterations: 1,
	}, nil
}

func (m *mockAgent) Stream(ctx context.Context, input string) (<-chan agent.AgentEvent, error) {
	return nil, nil
}

func (m *mockAgent) RunWithMessages(ctx context.Context, messages []bamboo.BambooMessage) (*agent.AgentResult, error) {
	return &agent.AgentResult{
		Content:    "Mock result with messages",
		Iterations: 1,
	}, nil
}

func (m *mockAgent) AddTool(t tool.Tool) error {
	return nil
}

func (m *mockAgent) SetSystemPrompt(prompt string) {
	// Mock implementation
}