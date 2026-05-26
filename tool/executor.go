package tool

import (
	"context"
	"encoding/json"
	"sync"
)

// ToolCallInput 表示来自 AI 的单个工具调用
type ToolCallInput struct {
	ID    string          // 调用 ID（由 AI 分配）
	Name  string          // 工具名称
	Input json.RawMessage // 调用参数
}

// ToolCallOutput 表示单个工具调用的结果
type ToolCallOutput struct {
	ID     string      // 调用 ID（与输入匹配）
	Result *ToolResult // 执行结果
	Error  error       // 执行错误（如有）
}

// ToolExecutor 并发执行工具调用
type ToolExecutor struct {
	registry      *Registry
	maxConcurrent int
}

// NewToolExecutor 创建新的工具执行器
// maxConcurrent 控制最大并发数，若 <= 0 则默认为 10
func NewToolExecutor(registry *Registry, maxConcurrent int) *ToolExecutor {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	return &ToolExecutor{
		registry:      registry,
		maxConcurrent: maxConcurrent,
	}
}

// ExecuteAll 并发执行多个工具调用
// 单个工具失败不影响其他工具的执行
// Context 取消会停止所有执行
func (e *ToolExecutor) ExecuteAll(ctx context.Context, toolCalls []ToolCallInput) ([]ToolCallOutput, error) {
	results := make([]ToolCallOutput, len(toolCalls))

	if len(toolCalls) == 0 {
		return results, nil
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, e.maxConcurrent) // 带缓冲的 channel 作为信号量

	for i, call := range toolCalls {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量（达到 maxConcurrent 时阻塞）

		go func(idx int, tc ToolCallInput) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 执行前检查 context 是否已取消
			select {
			case <-ctx.Done():
				results[idx] = ToolCallOutput{
					ID:    tc.ID,
					Error: ctx.Err(),
				}
				return
			default:
			}

			// 执行工具
			result, err := e.registry.Execute(ctx, tc.Name, tc.Input)
			results[idx] = ToolCallOutput{
				ID:     tc.ID,
				Result: result,
				Error:  err,
			}
		}(i, call)
	}

	wg.Wait()
	return results, nil
}
