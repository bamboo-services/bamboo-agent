package tool

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

type customExecuteTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}

func (c *customExecuteTool) Info() ToolInfo {
	return ToolInfo{
		Name:        c.name,
		Description: c.description,
		Parameters: InputSchema{
			Type:       "object",
			Properties: make(map[string]PropertyDef),
		},
	}
}

func (c *customExecuteTool) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	if c.executeFunc != nil {
		return c.executeFunc(ctx, input)
	}
	return &ToolResult{Content: "custom executed", IsError: false}, nil
}

func newCustomExecuteTool(name string, executeFunc func(ctx context.Context, input json.RawMessage) (*ToolResult, error)) *customExecuteTool {
	return &customExecuteTool{
		name:        name,
		description: "Custom execute tool for testing",
		executeFunc: executeFunc,
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.List() == nil {
		t.Fatal("NewRegistry() created registry with nil list")
	}
}

func TestRegistry_RegisterSuccess(t *testing.T) {
	registry := NewRegistry()
	tool := &mockTool{name: "test-tool", description: "test"}

	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if _, exists := registry.Get("test-tool"); !exists {
		t.Error("Tool was not registered")
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()
	tool1 := &mockTool{name: "duplicate-tool", description: "test1"}
	tool2 := &mockTool{name: "duplicate-tool", description: "test2"}

	if err := registry.Register(tool1); err != nil {
		t.Fatalf("First Register() failed: %v", err)
	}

	if err := registry.Register(tool2); err == nil {
		t.Error("Second Register() should have failed for duplicate tool")
	}
}

func TestRegistry_GetFound(t *testing.T) {
	registry := NewRegistry()
	tool := &mockTool{name: "found-tool", description: "test"}

	_ = registry.Register(tool)

	foundTool, exists := registry.Get("found-tool")
	if !exists {
		t.Fatal("Get() did not find registered tool")
	}
	if foundTool == nil {
		t.Fatal("Get() returned nil tool")
	}
	if foundTool.Info().Name != "found-tool" {
		t.Errorf("Get() returned wrong tool: got %s, want found-tool", foundTool.Info().Name)
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := NewRegistry()

	_, exists := registry.Get("non-existent-tool")
	if exists {
		t.Error("Get() should not find non-existent tool")
	}
}

func TestRegistry_ExecuteCorrectTool(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	tool := newCustomExecuteTool("exec-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		return &ToolResult{Content: "executed correctly", IsError: false}, nil
	})

	_ = registry.Register(tool)

	result, err := registry.Execute(ctx, "exec-tool", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	if result.Content != "executed correctly" {
		t.Errorf("Execute() returned wrong content: got %s, want executed correctly", result.Content)
	}
	if result.IsError {
		t.Error("Execute() returned error result")
	}
}

func TestRegistry_ExecuteUnknown(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	_, err := registry.Execute(ctx, "unknown-tool", json.RawMessage(`{}`))
	if err == nil {
		t.Error("Execute() should have failed for unknown tool")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	if infos := registry.List(); len(infos) != 0 {
		t.Errorf("List() should return empty list, got %d items", len(infos))
	}

	tool1 := &mockTool{name: "tool1", description: "test1"}
	tool2 := &mockTool{name: "tool2", description: "test2"}
	tool3 := &mockTool{name: "tool3", description: "test3"}

	_ = registry.Register(tool1)
	_ = registry.Register(tool2)
	_ = registry.Register(tool3)

	infos := registry.List()
	if len(infos) != 3 {
		t.Fatalf("List() should return 3 items, got %d", len(infos))
	}

	names := make(map[string]bool)
	for _, info := range infos {
		names[info.Name] = true
	}

	for _, name := range []string{"tool1", "tool2", "tool3"} {
		if !names[name] {
			t.Errorf("List() did not include tool %s", name)
		}
	}
}

func TestRegistry_RegisterBatch(t *testing.T) {
	registry := NewRegistry()

	tool1 := &mockTool{name: "batch-tool1", description: "test1"}
	tool2 := &mockTool{name: "batch-tool2", description: "test2"}
	tool3 := &mockTool{name: "batch-tool3", description: "test3"}

	if err := registry.RegisterBatch(tool1, tool2, tool3); err != nil {
		t.Fatalf("RegisterBatch() failed: %v", err)
	}

	if infos := registry.List(); len(infos) != 3 {
		t.Errorf("RegisterBatch() should register 3 tools, got %d", len(infos))
	}
}

func TestRegistry_RegisterBatchWithDuplicate(t *testing.T) {
	registry := NewRegistry()

	tool1 := &mockTool{name: "dup-tool1", description: "test1"}
	tool2 := &mockTool{name: "dup-tool2", description: "test2"}
	tool3 := &mockTool{name: "dup-tool1", description: "test3"}

	if err := registry.RegisterBatch(tool1, tool2, tool3); err == nil {
		t.Error("RegisterBatch() should have failed with duplicate tool")
	}

	if infos := registry.List(); len(infos) != 2 {
		t.Errorf("RegisterBatch() should have registered only 2 tools, got %d", len(infos))
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			tool := &mockTool{
				name:        "concurrent-tool-" + string(rune('0'+n)),
				description: "test",
			}
			_ = registry.Register(tool)
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = registry.Execute(ctx, "concurrent-tool-"+string(rune('0'+n%10)), json.RawMessage(`{}`))
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = registry.Get("concurrent-tool-" + string(rune('0'+n%10)))
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.List()
		}()
	}

	wg.Wait()

	_ = registry.List()
}

func TestRegistry_ExecuteWithError(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	expectedErr := errors.New("execution failed")
	tool := newCustomExecuteTool("error-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		return nil, expectedErr
	})

	_ = registry.Register(tool)

	_, err := registry.Execute(ctx, "error-tool", nil)
	if err != expectedErr {
		t.Errorf("Execute() should return execution error, got: %v", err)
	}
}

func TestRegistry_ExecuteWithInput(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	input := json.RawMessage(`{"key":"value"}`)
	tool := newCustomExecuteTool("input-tool", func(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
		return &ToolResult{Content: string(input), IsError: false}, nil
	})

	_ = registry.Register(tool)

	result, err := registry.Execute(ctx, "input-tool", input)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		t.Fatalf("Failed to parse result content: %v", err)
	}

	if parsed["key"] != "value" {
		t.Errorf("Input was not correctly passed to tool: got %s, want value", parsed["key"])
	}
}