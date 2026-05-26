package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-agent/agent"
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock Agent
// ---------------------------------------------------------------------------

type orchMockAgent struct {
	runFunc func(ctx context.Context, input string) (*agent.AgentResult, error)
}

func (m *orchMockAgent) Run(ctx context.Context, input string) (*agent.AgentResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, input)
	}
	return &agent.AgentResult{Content: "mock: " + input}, nil
}

func (m *orchMockAgent) Stream(ctx context.Context, input string) (<-chan agent.AgentEvent, error) {
	return nil, nil
}

func (m *orchMockAgent) RunWithMessages(ctx context.Context, messages []bamboo.BambooMessage) (*agent.AgentResult, error) {
	return nil, nil
}

func (m *orchMockAgent) AddTool(t tool.Tool) error { return nil }

func (m *orchMockAgent) SetSystemPrompt(prompt string) {}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resultAgent returns a mock agent that produces a fixed content string.
func resultAgent(content string) *orchMockAgent {
	return &orchMockAgent{
		runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
			return &agent.AgentResult{Content: content}, nil
		},
	}
}

// delayAgent returns a mock agent that sleeps for the given duration, then returns.
func delayAgent(d time.Duration, content string) *orchMockAgent {
	return &orchMockAgent{
		runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
			select {
			case <-time.After(d):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return &agent.AgentResult{Content: content}, nil
		},
	}
}

// failAgent returns a mock agent that always returns an error.
func failAgent(err error) *orchMockAgent {
	return &orchMockAgent{
		runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
			return nil, err
		},
	}
}

// slowAgent records its start time and sleeps; used to verify parallelism.
func slowAgent(d time.Duration, content string, start *atomic.Int64) *orchMockAgent {
	return &orchMockAgent{
		runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
			start.Store(time.Now().UnixMilli())
			time.Sleep(d)
			return &agent.AgentResult{Content: content}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestOrchestrator_RegisterAgent 测试注册 Agent 功能。
func TestOrchestrator_RegisterAgent(t *testing.T) {
	o := NewOrchestrator()

	err := o.RegisterAgent("a", resultAgent("a"))
	assert.NoError(t, err)

	err = o.RegisterAgent("a", resultAgent("a2"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestOrchestrator_ExecuteSequential 测试顺序执行任务。
func TestOrchestrator_ExecuteSequential(t *testing.T) {
	o := NewOrchestrator()

	var order []string
	var mu sync.Mutex

	for _, name := range []string{"a", "b", "c"} {
		name := name
		o.RegisterAgent(name, &orchMockAgent{
			runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
				mu.Lock()
				order = append(order, name)
				mu.Unlock()
				return &agent.AgentResult{Content: name}, nil
			},
		})
	}

	tasks := []Task{
		{ID: "a", Input: "do a"},
		{ID: "b", Input: "do b"},
		{ID: "c", Input: "do c"},
	}

	results, err := o.ExecuteSequential(context.Background(), tasks)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"a", "b", "c"}, order)
	for _, r := range results {
		assert.Equal(t, TaskCompleted, r.Status)
	}
}

// TestOrchestrator_ExecuteParallel 测试并行执行任务。
func TestOrchestrator_ExecuteParallel(t *testing.T) {
	o := NewOrchestrator()

	var startTimes []int64
	var mu sync.Mutex

	for _, name := range []string{"a", "b", "c"} {
		name := name
		o.RegisterAgent(name, &orchMockAgent{
			runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
				mu.Lock()
				startTimes = append(startTimes, time.Now().UnixMilli())
				mu.Unlock()
				time.Sleep(50 * time.Millisecond)
				return &agent.AgentResult{Content: name}, nil
			},
		})
	}

	tasks := []Task{
		{ID: "a", Input: "do a"},
		{ID: "b", Input: "do b"},
		{ID: "c", Input: "do c"},
	}

	results, err := o.ExecuteParallel(context.Background(), tasks)
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, TaskCompleted, r.Status)
	}

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, startTimes, 3)
	spread := startTimes[2] - startTimes[0]
	assert.Less(t, spread, int64(200), "tasks should start within 200ms of each other (parallel)")
}

// TestOrchestrator_Execute_DependencyGraph 测试依赖图执行。
func TestOrchestrator_Execute_DependencyGraph(t *testing.T) {
	o := NewOrchestrator()

	var execOrder []string
	var mu sync.Mutex

	register := func(name string) {
		o.RegisterAgent(name, &orchMockAgent{
			runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
				mu.Lock()
				execOrder = append(execOrder, name)
				mu.Unlock()
				time.Sleep(30 * time.Millisecond)
				return &agent.AgentResult{Content: name}, nil
			},
		})
	}
	register("a")
	register("b")
	register("c")

	tasks := []Task{
		{ID: "a", Input: "do a"},
		{ID: "b", Input: "do b"},
		{ID: "c", Input: "do c", DependsOn: []string{"a", "b"}},
	}

	results, err := o.Execute(context.Background(), tasks)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	for _, r := range results {
		assert.Equal(t, TaskCompleted, r.Status, fmt.Sprintf("task %s should be completed", r.ID))
	}

	idx := func(name string) int {
		for i, n := range execOrder {
			if n == name {
				return i
			}
		}
		return -1
	}
	assert.Less(t, idx("a"), idx("c"), "a should execute before c")
	assert.Less(t, idx("b"), idx("c"), "b should execute before c")
}

// TestOrchestrator_Execute_TaskFailureIsolation 测试任务失败隔离。
func TestOrchestrator_Execute_TaskFailureIsolation(t *testing.T) {
	o := NewOrchestrator()

	o.RegisterAgent("a", failAgent(errors.New("boom")))
	o.RegisterAgent("b", resultAgent("b-result"))

	tasks := []Task{
		{ID: "a", Input: "fail"},
		{ID: "b", Input: "succeed"},
	}

	results, err := o.Execute(context.Background(), tasks)
	require.NoError(t, err)

	for _, r := range results {
		if r.ID == "a" {
			assert.Equal(t, TaskFailed, r.Status)
			assert.Error(t, r.Error)
		}
		if r.ID == "b" {
			assert.Equal(t, TaskCompleted, r.Status)
			assert.Equal(t, "b-result", r.Result.Content)
		}
	}
}

// TestOrchestrator_Execute_FailedDependencyCascades 测试失败依赖级联。
func TestOrchestrator_Execute_FailedDependencyCascades(t *testing.T) {
	o := NewOrchestrator()

	o.RegisterAgent("a", failAgent(errors.New("boom")))
	o.RegisterAgent("b", resultAgent("b-result"))
	o.RegisterAgent("c", resultAgent("c-result"))

	tasks := []Task{
		{ID: "a", Input: "fail"},
		{ID: "b", Input: "succeed"},
		{ID: "c", Input: "depends-on-failed", DependsOn: []string{"a"}},
	}

	results, err := o.Execute(context.Background(), tasks)
	require.NoError(t, err)

	for _, r := range results {
		switch r.ID {
		case "a":
			assert.Equal(t, TaskFailed, r.Status)
		case "b":
			assert.Equal(t, TaskCompleted, r.Status)
		case "c":
			assert.Equal(t, TaskFailed, r.Status)
			assert.Contains(t, r.Error.Error(), "dependency")
		}
	}
}

// TestOrchestrator_Execute_ContextCancellation 测试上下文取消。
func TestOrchestrator_Execute_ContextCancellation(t *testing.T) {
	o := NewOrchestrator()

	o.RegisterAgent("blocked", &orchMockAgent{
		runFunc: func(ctx context.Context, input string) (*agent.AgentResult, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	tasks := []Task{
		{ID: "blocked", Input: "will-cancel"},
	}

	doneCh := make(chan struct{})
	var results []Task
	go func() {
		results, _ = o.Execute(ctx, tasks)
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(3 * time.Second):
		t.Fatal("Execute did not return within 3 seconds")
	}

	assert.Equal(t, TaskFailed, results[0].Status)
}

// TestOrchestrator_Execute_NoAgentRegistered 测试无注册 Agent。
func TestOrchestrator_Execute_NoAgentRegistered(t *testing.T) {
	o := NewOrchestrator()

	tasks := []Task{
		{ID: "missing", Input: "no agent"},
	}

	results, err := o.Execute(context.Background(), tasks)
	require.NoError(t, err)

	assert.Equal(t, TaskFailed, results[0].Status)
	assert.Contains(t, results[0].Error.Error(), "no agent")
}

// TestOrchestrator_ExecuteSequential_ContextCancellation 测试顺序执行的上下文取消。
func TestOrchestrator_ExecuteSequential_ContextCancellation(t *testing.T) {
	o := NewOrchestrator()

	o.RegisterAgent("fast", resultAgent("fast"))
	o.RegisterAgent("slow", delayAgent(5*time.Second, "too-late"))

	ctx, cancel := context.WithCancel(context.Background())

	tasks := []Task{
		{ID: "fast", Input: "ok"},
		{ID: "slow", Input: "will-cancel"},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	results, err := o.ExecuteSequential(ctx, tasks)
	assert.Equal(t, TaskCompleted, results[0].Status)
	_ = err
}

// TestOrchestrator_ExecuteParallel_NoAgentRegistered 测试并行执行的无注册 Agent。
func TestOrchestrator_ExecuteParallel_NoAgentRegistered(t *testing.T) {
	o := NewOrchestrator()

	tasks := []Task{
		{ID: "missing-a", Input: "no agent"},
		{ID: "missing-b", Input: "no agent"},
	}

	results, err := o.ExecuteParallel(context.Background(), tasks)
	require.NoError(t, err)

	for _, r := range results {
		assert.Equal(t, TaskFailed, r.Status)
		assert.Contains(t, r.Error.Error(), "no agent")
	}
}

// TestOrchestrator_ExecuteParallel_TaskFailure 测试并行执行的任务失败。
func TestOrchestrator_ExecuteParallel_TaskFailure(t *testing.T) {
	o := NewOrchestrator()

	o.RegisterAgent("fail", failAgent(errors.New("parallel-fail")))
	o.RegisterAgent("ok", resultAgent("ok-result"))

	tasks := []Task{
		{ID: "fail", Input: "fail"},
		{ID: "ok", Input: "ok"},
	}

	results, err := o.ExecuteParallel(context.Background(), tasks)
	require.NoError(t, err)

	for _, r := range results {
		if r.ID == "fail" {
			assert.Equal(t, TaskFailed, r.Status)
		}
		if r.ID == "ok" {
			assert.Equal(t, TaskCompleted, r.Status)
			assert.Equal(t, "ok-result", r.Result.Content)
		}
	}
}

// TestOrchestrator_Execute_UsesTaskAgentField 测试使用 Task 的 Agent 字段。
func TestOrchestrator_Execute_UsesTaskAgentField(t *testing.T) {
	o := NewOrchestrator()

	tasks := []Task{
		{
			ID:    "inline",
			Input: "hello",
			Agent: resultAgent("inline-result"),
		},
	}

	results, err := o.Execute(context.Background(), tasks)
	require.NoError(t, err)
	assert.Equal(t, TaskCompleted, results[0].Status)
	assert.Equal(t, "inline-result", results[0].Result.Content)
}

// TestOrchestrator_ExecuteSequential_TaskAgentField 测试顺序执行使用 Task 的 Agent 字段。
func TestOrchestrator_ExecuteSequential_TaskAgentField(t *testing.T) {
	o := NewOrchestrator()

	tasks := []Task{
		{
			ID:    "inline",
			Input: "hello",
			Agent: resultAgent("seq-inline"),
		},
	}

	results, err := o.ExecuteSequential(context.Background(), tasks)
	require.NoError(t, err)
	assert.Equal(t, TaskCompleted, results[0].Status)
	assert.Equal(t, "seq-inline", results[0].Result.Content)
}

// TestOrchestrator_ExecuteParallel_TaskAgentField 测试并行执行使用 Task 的 Agent 字段。
func TestOrchestrator_ExecuteParallel_TaskAgentField(t *testing.T) {
	o := NewOrchestrator()

	tasks := []Task{
		{
			ID:    "inline",
			Input: "hello",
			Agent: resultAgent("par-inline"),
		},
	}

	results, err := o.ExecuteParallel(context.Background(), tasks)
	require.NoError(t, err)
	assert.Equal(t, TaskCompleted, results[0].Status)
	assert.Equal(t, "par-inline", results[0].Result.Content)
}