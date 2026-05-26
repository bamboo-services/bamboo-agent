package tool

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// setupTestRegistry 创建包含指定工具的测试 Registry
func setupTestRegistry(tools ...Tool) *Registry {
	r := NewRegistry()
	for _, t := range tools {
		_ = r.Register(t)
	}
	return r
}

func TestExecutor_ConcurrentExecution(t *testing.T) {
	var callCount int32
	tool1 := newCustomExecuteTool("tool-a", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		atomic.AddInt32(&callCount, 1)
		return &ToolResult{Content: "result-a", IsError: false}, nil
	})
	tool2 := newCustomExecuteTool("tool-b", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		atomic.AddInt32(&callCount, 1)
		return &ToolResult{Content: "result-b", IsError: false}, nil
	})
	tool3 := newCustomExecuteTool("tool-c", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		atomic.AddInt32(&callCount, 1)
		return &ToolResult{Content: "result-c", IsError: false}, nil
	})

	registry := setupTestRegistry(tool1, tool2, tool3)
	executor := NewToolExecutor(registry, 10)

	calls := []ToolCallInput{
		{ID: "call-1", Name: "tool-a", Input: json.RawMessage(`{}`)},
		{ID: "call-2", Name: "tool-b", Input: json.RawMessage(`{}`)},
		{ID: "call-3", Name: "tool-c", Input: json.RawMessage(`{}`)},
	}

	results, err := executor.ExecuteAll(context.Background(), calls)
	if err != nil {
		t.Fatalf("ExecuteAll() returned error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	if atomic.LoadInt32(&callCount) != 3 {
		t.Errorf("Expected 3 tool calls, got %d", callCount)
	}

	// 验证结果按索引顺序排列（ID 匹配）
	expectedIDs := []string{"call-1", "call-2", "call-3"}
	expectedContents := []string{"result-a", "result-b", "result-c"}
	for i, r := range results {
		if r.ID != expectedIDs[i] {
			t.Errorf("results[%d].ID = %q, want %q", i, r.ID, expectedIDs[i])
		}
		if r.Error != nil {
			t.Errorf("results[%d].Error = %v, want nil", i, r.Error)
		}
		if r.Result.Content != expectedContents[i] {
			t.Errorf("results[%d].Result.Content = %q, want %q", i, r.Result.Content, expectedContents[i])
		}
	}
}

func TestExecutor_SingleFailureDoesNotAffectOthers(t *testing.T) {
	tool1 := newCustomExecuteTool("good-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		return &ToolResult{Content: "good-result", IsError: false}, nil
	})
	tool2 := newCustomExecuteTool("bad-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		return nil, errors.New("tool execution failed")
	})
	tool3 := newCustomExecuteTool("another-good", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		return &ToolResult{Content: "another-good-result", IsError: false}, nil
	})

	registry := setupTestRegistry(tool1, tool2, tool3)
	executor := NewToolExecutor(registry, 10)

	calls := []ToolCallInput{
		{ID: "call-1", Name: "good-tool", Input: json.RawMessage(`{}`)},
		{ID: "call-2", Name: "bad-tool", Input: json.RawMessage(`{}`)},
		{ID: "call-3", Name: "another-good", Input: json.RawMessage(`{}`)},
	}

	results, err := executor.ExecuteAll(context.Background(), calls)
	if err != nil {
		t.Fatalf("ExecuteAll() returned error: %v", err)
	}

	// call-1: 成功
	if results[0].Error != nil {
		t.Errorf("results[0] should not have error, got: %v", results[0].Error)
	}
	if results[0].Result.Content != "good-result" {
		t.Errorf("results[0].Content = %q, want %q", results[0].Result.Content, "good-result")
	}

	// call-2: 失败
	if results[1].Error == nil {
		t.Error("results[1] should have error")
	}
	if results[1].Result != nil {
		t.Errorf("results[1].Result should be nil, got: %v", results[1].Result)
	}

	// call-3: 成功（不受 call-2 影响）
	if results[2].Error != nil {
		t.Errorf("results[2] should not have error, got: %v", results[2].Error)
	}
	if results[2].Result.Content != "another-good-result" {
		t.Errorf("results[2].Content = %q, want %q", results[2].Result.Content, "another-good-result")
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	// 创建一个会阻塞等待 context 取消的工具
	tool := newCustomExecuteTool("slow-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	registry := setupTestRegistry(tool)
	executor := NewToolExecutor(registry, 10)

	ctx, cancel := context.WithCancel(context.Background())

	calls := []ToolCallInput{
		{ID: "call-1", Name: "slow-tool", Input: json.RawMessage(`{}`)},
	}

	// 启动执行后立即取消 context
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	results, err := executor.ExecuteAll(ctx, calls)
	if err != nil {
		t.Fatalf("ExecuteAll() returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("Expected error due to context cancellation")
	}
}

func TestExecutor_MaxConcurrentLimitsParallelism(t *testing.T) {
	// 使用 maxConcurrent=2 来限制并发
	var activeCount int32
	var maxActive int32

	tool := newCustomExecuteTool("limited-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		current := atomic.AddInt32(&activeCount, 1)
		// 记录最大并发数
		for {
			old := atomic.LoadInt32(&maxActive)
			if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond) // 模拟工作

		atomic.AddInt32(&activeCount, -1)
		return &ToolResult{Content: "done", IsError: false}, nil
	})

	registry := setupTestRegistry(tool)
	executor := NewToolExecutor(registry, 2) // 最大并发 2

	// 发起 5 个调用
	calls := make([]ToolCallInput, 5)
	for i := range calls {
		calls[i] = ToolCallInput{
			ID:    "call-" + string(rune('0'+i)),
			Name:  "limited-tool",
			Input: json.RawMessage(`{}`),
		}
	}

	results, err := executor.ExecuteAll(context.Background(), calls)
	if err != nil {
		t.Fatalf("ExecuteAll() returned error: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("Expected 5 results, got %d", len(results))
	}

	// 验证最大并发数不超过 2
	if maxObs := atomic.LoadInt32(&maxActive); maxObs > 2 {
		t.Errorf("Max concurrent should be <= 2, got %d", maxObs)
	}

	// 所有调用都应该成功
	for i, r := range results {
		if r.Error != nil {
			t.Errorf("results[%d] should not have error: %v", i, r.Error)
		}
	}
}

func TestExecutor_EmptyInput(t *testing.T) {
	registry := NewRegistry()
	executor := NewToolExecutor(registry, 10)

	results, err := executor.ExecuteAll(context.Background(), nil)
	if err != nil {
		t.Fatalf("ExecuteAll() with nil input returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for nil input, got %d", len(results))
	}

	results, err = executor.ExecuteAll(context.Background(), []ToolCallInput{})
	if err != nil {
		t.Fatalf("ExecuteAll() with empty slice returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(results))
	}
}

func TestExecutor_UnknownTool(t *testing.T) {
	registry := NewRegistry()
	tool := newCustomExecuteTool("known-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		return &ToolResult{Content: "known-result", IsError: false}, nil
	})
	_ = registry.Register(tool)

	executor := NewToolExecutor(registry, 10)

	calls := []ToolCallInput{
		{ID: "call-1", Name: "known-tool", Input: json.RawMessage(`{}`)},
		{ID: "call-2", Name: "unknown-tool", Input: json.RawMessage(`{}`)},
	}

	results, err := executor.ExecuteAll(context.Background(), calls)
	if err != nil {
		t.Fatalf("ExecuteAll() returned error: %v", err)
	}

	// call-1: 已知工具 — 成功
	if results[0].Error != nil {
		t.Errorf("results[0] should not have error, got: %v", results[0].Error)
	}
	if results[0].Result.Content != "known-result" {
		t.Errorf("results[0].Content = %q, want %q", results[0].Result.Content, "known-result")
	}

	// call-2: 未知工具 — 错误
	if results[1].Error == nil {
		t.Error("results[1] should have error for unknown tool")
	}
}

func TestExecutor_ResultsPreserveOrder(t *testing.T) {
	// 创建返回各自名称的工具，验证结果顺序与输入一致
	tools := make([]Tool, 5)
	for i := 0; i < 5; i++ {
		idx := i
		tools[i] = newCustomExecuteTool("ordered-"+string(rune('A'+idx)), func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
			// 添加随机延迟以打乱实际完成顺序
			time.Sleep(time.Duration(idx*10) * time.Millisecond)
			return &ToolResult{Content: "ordered-" + string(rune('A'+idx)), IsError: false}, nil
		})
	}

	registry := setupTestRegistry(tools...)
	executor := NewToolExecutor(registry, 10)

	calls := make([]ToolCallInput, 5)
	for i := range calls {
		calls[i] = ToolCallInput{
			ID:    "call-" + string(rune('0'+i)),
			Name:  "ordered-" + string(rune('A'+i)),
			Input: json.RawMessage(`{}`),
		}
	}

	results, err := executor.ExecuteAll(context.Background(), calls)
	if err != nil {
		t.Fatalf("ExecuteAll() returned error: %v", err)
	}

	for i, r := range results {
		expectedID := "call-" + string(rune('0'+i))
		expectedContent := "ordered-" + string(rune('A'+i))

		if r.ID != expectedID {
			t.Errorf("results[%d].ID = %q, want %q", i, r.ID, expectedID)
		}
		if r.Result == nil || r.Result.Content != expectedContent {
			t.Errorf("results[%d].Content = %q, want %q", i, r.Result.Content, expectedContent)
		}
	}
}

func TestExecutor_DefaultMaxConcurrent(t *testing.T) {
	registry := NewRegistry()

	// maxConcurrent <= 0 应该默认为 10
	executor := NewToolExecutor(registry, 0)
	if executor.maxConcurrent != 10 {
		t.Errorf("maxConcurrent should default to 10, got %d", executor.maxConcurrent)
	}

	executor = NewToolExecutor(registry, -5)
	if executor.maxConcurrent != 10 {
		t.Errorf("maxConcurrent should default to 10 for negative values, got %d", executor.maxConcurrent)
	}
}
