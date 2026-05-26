package tool

import (
	"context"
	"encoding/json"
	"testing"
)

// mockTool 用于测试工具接口
type mockTool struct {
	name        string
	description string
}

func (m *mockTool) Info() ToolInfo {
	return ToolInfo{
		Name:        m.name,
		Description: m.description,
		Parameters: InputSchema{
			Type:       "object",
			Properties: map[string]PropertyDef{},
			Required:   []string{},
		},
	}
}

func (m *mockTool) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	return &ToolResult{
		Content: "mock tool executed successfully",
		IsError: false,
	}, nil
}

func TestToolInterface(t *testing.T) {
	// 创建 mockTool 实例
	mock := &mockTool{
		name:        "test_tool",
		description: "A test tool for unit testing",
	}

	// 验证它实现了 Tool 接口
	var _ Tool = mock

	// 测试 Info() 方法
	info := mock.Info()
	if info.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", info.Name)
	}
	if info.Description != "A test tool for unit testing" {
		t.Errorf("Expected description 'A test tool for unit testing', got '%s'", info.Description)
	}
	if info.Parameters.Type != "object" {
		t.Errorf("Expected parameters type 'object', got '%s'", info.Parameters.Type)
	}
}

func TestToolExecute(t *testing.T) {
	mock := &mockTool{
		name:        "test_tool",
		description: "A test tool for unit testing",
	}

	ctx := context.Background()
	input := json.RawMessage(`{"test": "input"}`)

	result, err := mock.Execute(ctx, input)
	if err != nil {
		t.Errorf("Execute returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("Expected IsError=false, got true")
	}
	if result.Content != "mock tool executed successfully" {
		t.Errorf("Expected content 'mock tool executed successfully', got '%s'", result.Content)
	}
}

func TestToolResult(t *testing.T) {
	result := &ToolResult{
		Content: "test content",
		IsError: false,
	}

	if result.Content != "test content" {
		t.Errorf("Expected Content 'test content', got '%s'", result.Content)
	}
	if result.IsError != false {
		t.Errorf("Expected IsError false, got true")
	}
}

func TestToolInfo(t *testing.T) {
	info := ToolInfo{
		Name:        "tool_name",
		Description: "tool description",
		Parameters: InputSchema{
			Type:       "object",
			Properties: map[string]PropertyDef{},
			Required:   []string{},
		},
	}

	if info.Name != "tool_name" {
		t.Errorf("Expected Name 'tool_name', got '%s'", info.Name)
	}
	if info.Description != "tool description" {
		t.Errorf("Expected Description 'tool description', got '%s'", info.Description)
	}
}

func TestInputSchema(t *testing.T) {
	schema := InputSchema{
		Type: "object",
		Properties: map[string]PropertyDef{
			"param1": {
				Type:        "string",
				Description: "A test parameter",
			},
		},
		Required: []string{"param1"},
	}

	if schema.Type != "object" {
		t.Errorf("Expected Type 'object', got '%s'", schema.Type)
	}
	if len(schema.Properties) != 1 {
		t.Errorf("Expected 1 property, got %d", len(schema.Properties))
	}
	if len(schema.Required) != 1 || schema.Required[0] != "param1" {
		t.Errorf("Expected Required to contain 'param1', got %v", schema.Required)
	}
}