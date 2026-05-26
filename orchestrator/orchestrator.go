package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-agent/agent"
)

// Orchestrator 管理多 Agent 任务执行，支持依赖解析。
//
// 提供以下核心功能：
//   - Agent 注册与管理
//   - 依赖关系解析
//   - 并发与串行执行模式
//
// 内部使用读写锁保护 Agent 注册表，确保并发安全。
type Orchestrator struct {
	mu     sync.RWMutex
	agents map[string]agent.Agent
}

// NewOrchestrator 创建一个新的 Orchestrator 实例。
//
// 返回：
//   - *Orchestrator - 初始化后的 Orchestrator
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		agents: make(map[string]agent.Agent),
	}
}

// RegisterAgent 向 Orchestrator 注册一个 Agent。
//
// 如果名称已存在则返回错误。
//
// 参数说明：
//   - name - Agent 名称，作为唯一标识符
//   - ag - Agent 实例
//
// 返回：
//   - error - 注册失败时返回错误（如名称重复）
func (o *Orchestrator) RegisterAgent(name string, ag agent.Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.agents[name]; exists {
		return fmt.Errorf("agent %q already registered", name)
	}
	o.agents[name] = ag
	return nil
}

// resolveAgent 查找任务对应的 Agent。
//
// 优先使用任务自带的 Agent，如果不存在则从注册表中查找。
//
// 参数说明：
//   - t - 任务实例
//
// 返回：
//   - agent.Agent - 找到的 Agent
//   - error - 未找到 Agent 时返回错误
func (o *Orchestrator) resolveAgent(t *Task) (agent.Agent, error) {
	if t.Agent != nil {
		return t.Agent, nil
	}
	o.mu.RLock()
	ag, ok := o.agents[t.ID]
	o.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no agent registered for task %q", t.ID)
	}
	return ag, nil
}

// Execute 执行任务列表，自动处理依赖关系。
//
// 无依赖的任务会并行执行，有依赖的任务会等待依赖完成。
// 依赖失败的任务会自动标记为失败，不再执行。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - tasks - 任务列表
//
// 返回：
//   - []Task - 执行后的任务列表（包含执行状态和结果）
//   - error - 上下文取消时返回错误
func (o *Orchestrator) Execute(ctx context.Context, tasks []Task) ([]Task, error) {
	// Build task map for dependency lookups
	taskMap := make(map[string]*Task)
	for i := range tasks {
		tasks[i].Status = TaskPending
		taskMap[tasks[i].ID] = &tasks[i]
	}

	// Track completion
	var mu sync.Mutex
	done := make(map[string]bool)
	failed := make(map[string]bool)

	// Process tasks iteratively: find ready tasks, execute them, repeat
	for {
		// First pass: cascade failures from failed dependencies
		mu.Lock()
		changed := true
		for changed {
			changed = false
			for i := range tasks {
				t := &tasks[i]
				if t.Status != TaskPending {
					continue
				}
				for _, dep := range t.DependsOn {
					if failed[dep] {
						t.Status = TaskFailed
						t.Error = fmt.Errorf("dependency %q failed", dep)
						done[t.ID] = true
						failed[t.ID] = true
						changed = true
						break
					}
				}
			}
		}

		// Find all tasks that are ready (all deps satisfied and not failed)
		var ready []*Task
		for i := range tasks {
			t := &tasks[i]
			if t.Status != TaskPending {
				continue
			}
			allDepsDone := true
			for _, dep := range t.DependsOn {
				if !done[dep] {
					allDepsDone = false
					break
				}
			}
			if allDepsDone {
				ready = append(ready, t)
			}
		}
		mu.Unlock()

		if len(ready) == 0 {
			// Check if all tasks are done (no more pending)
			mu.Lock()
			allDone := true
			for i := range tasks {
				if tasks[i].Status == TaskPending {
					allDone = false
					break
				}
			}
			mu.Unlock()
			if allDone {
				break
			}

			// Wait before re-checking
			select {
			case <-ctx.Done():
				return tasks, ctx.Err()
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}

		// Execute ready tasks in parallel
		var wg sync.WaitGroup
		for _, t := range ready {
			t.Status = TaskRunning
			wg.Add(1)
			go func(task *Task) {
				defer wg.Done()

				ag, err := o.resolveAgent(task)
				mu.Lock()
				if err != nil {
					task.Status = TaskFailed
					task.Error = err
					failed[task.ID] = true
				} else {
					result, runErr := ag.Run(ctx, task.Input)
					if runErr != nil {
						task.Status = TaskFailed
						task.Error = runErr
						failed[task.ID] = true
					} else {
						task.Status = TaskCompleted
						task.Result = result
					}
				}
				done[task.ID] = true
				mu.Unlock()
			}(t)
		}
		wg.Wait()
	}

	return tasks, nil
}

// ExecuteSequential 顺序执行所有任务。
//
// 任务会按照列表顺序一个接一个执行，忽略依赖关系。
// 即使某个任务失败，后续任务仍会继续执行。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - tasks - 任务列表
//
// 返回：
//   - []Task - 执行后的任务列表（包含执行状态和结果）
//   - error - 上下文取消时返回错误
func (o *Orchestrator) ExecuteSequential(ctx context.Context, tasks []Task) ([]Task, error) {
	for i := range tasks {
		tasks[i].Status = TaskRunning

		ag, err := o.resolveAgent(&tasks[i])
		if err != nil {
			tasks[i].Status = TaskFailed
			tasks[i].Error = err
			continue
		}

		result, runErr := ag.Run(ctx, tasks[i].Input)
		if runErr != nil {
			tasks[i].Status = TaskFailed
			tasks[i].Error = runErr
		} else {
			tasks[i].Status = TaskCompleted
			tasks[i].Result = result
		}

		// Check context between tasks
		select {
		case <-ctx.Done():
			return tasks, ctx.Err()
		default:
		}
	}
	return tasks, nil
}

// ExecuteParallel 并发执行所有任务。
//
// 所有任务同时启动，忽略依赖关系。
// 即使某个任务失败，其他任务仍会继续执行。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - tasks - 任务列表
//
// 返回：
//   - []Task - 执行后的任务列表（包含执行状态和结果）
func (o *Orchestrator) ExecuteParallel(ctx context.Context, tasks []Task) ([]Task, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range tasks {
		tasks[i].Status = TaskRunning
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			ag, err := o.resolveAgent(&tasks[idx])
			mu.Lock()
			if err != nil {
				tasks[idx].Status = TaskFailed
				tasks[idx].Error = err
			} else {
				result, runErr := ag.Run(ctx, tasks[idx].Input)
				if runErr != nil {
					tasks[idx].Status = TaskFailed
					tasks[idx].Error = runErr
				} else {
					tasks[idx].Status = TaskCompleted
					tasks[idx].Result = result
				}
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	return tasks, nil
}