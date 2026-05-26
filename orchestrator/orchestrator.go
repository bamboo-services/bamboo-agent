package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-agent/agent"
)

// Orchestrator manages multi-agent task execution with dependency resolution.
type Orchestrator struct {
	mu     sync.RWMutex
	agents map[string]agent.Agent
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		agents: make(map[string]agent.Agent),
	}
}

// RegisterAgent registers an agent with the given name.
func (o *Orchestrator) RegisterAgent(name string, ag agent.Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.agents[name]; exists {
		return fmt.Errorf("agent %q already registered", name)
	}
	o.agents[name] = ag
	return nil
}

// resolveAgent finds the agent for a task: prefers task.Agent, falls back to registered agents.
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

// Execute runs tasks respecting dependency ordering.
// Tasks without dependencies run in parallel. Tasks with dependencies wait.
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

// ExecuteSequential runs all tasks one after another.
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

// ExecuteParallel runs all tasks concurrently.
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
