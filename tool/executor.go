package tool

import (
	"context"
	"encoding/json"
	"sync"
)

// ToolCallInput 表示来自 AI 的单个工具调用。
//
// 保存工具调用的标识符、名称和参数，用于并发执行。
type ToolCallInput struct {
	// ID 是调用标识符，由 AI 分配，用于匹配输入和输出。
	ID string

	// Name 是工具名称，用于在注册表中查找对应的工具。
	Name string

	// Input 是调用参数，以 JSON 格式传递给工具。
	Input json.RawMessage
}

// ToolCallOutput 表示单个工具调用的结果。
//
// 保存工具调用的结果、错误信息,用于匹配 ToolCallInput。
type ToolCallOutput struct {
	// ID 是调用标识符,与 ToolCallInput.ID 匹配。
	ID string

	// Result 是执行结果,包含工具返回的内容。
	Result *ToolResult

	// Error 是执行错误,如工具不存在或执行失败。
	Error error
}

// ToolExecutor 并发执行工具调用。
//
// 使用信号量控制并发数量,支持 Context 取消。
type ToolExecutor struct {
	// registry 是工具注册表,用于查找和执行工具。
	registry *Registry

	// maxConcurrent 是最大并发数,用于控制 goroutine 数量。
	maxConcurrent int
}

// NewToolExecutor 创建新的工具执行器。
//
// 参数说明：
//   - registry - 工具注册表,用于查找和执行工具
//   - maxConcurrent - 最大并发数,若 <= 0 则默认为 10
//
// 返回：
//   - *ToolExecutor - 新创建的工具执行器
func NewToolExecutor(registry *Registry, maxConcurrent int) *ToolExecutor {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	return &ToolExecutor{
		registry:      registry,
		maxConcurrent: maxConcurrent,
	}
}

// ExecuteAll 并发执行多个工具调用。
//
// 单个工具失败不影响其他工具的执行,Context 取消会停止所有执行。
// 使用信号量控制最大并发数,保证 goroutine 数量可控。
//
// 参数说明：
//   - ctx - 上下文,用于取消和超时控制
//   - toolCalls - 工具调用列表
//
// 返回：
//   - []ToolCallOutput - 执行结果列表,与输入顺序一致
//   - error - 暂不返回错误
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
