package agent

import (
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// TestAgentEventTypeValues 验证所有事件类型常量都是非空字符串 ✨
func TestAgentEventTypeValues(t *testing.T) {
	tests := []struct {
		name  string
		event AgentEventType
	}{
		{"Text", AgentEventText},
		{"Thinking", AgentEventThinking},
		{"ToolCall", AgentEventToolCall},
		{"ToolResult", AgentEventToolResult},
		{"Complete", AgentEventComplete},
		{"Error", AgentEventError},
		{"Compress", AgentEventCompress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.event) == "" {
				t.Errorf("AgentEventType %s should not be empty", tt.name)
			}
		})
	}
}

// TestToolCallRecordFields 测试 ToolCallRecord 字段访问 🛠️
func TestToolCallRecordFields(t *testing.T) {
	record := ToolCallRecord{
		ID:      "test-123",
		Name:    "test_tool",
		Input:   []byte(`{"key": "value"}`),
		Result:  "success",
		IsError: false,
	}

	if record.ID != "test-123" {
		t.Errorf("Expected ID 'test-123', got '%s'", record.ID)
	}

	if record.Name != "test_tool" {
		t.Errorf("Expected Name 'test_tool', got '%s'", record.Name)
	}

	if string(record.Input) != `{"key": "value"}` {
		t.Errorf("Input mismatch: %s", string(record.Input))
	}

	if record.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", record.Result)
	}

	if record.IsError {
		t.Errorf("Expected IsError false, got true")
	}

	// 测试错误场景
	errorRecord := ToolCallRecord{
		ID:      "error-456",
		IsError: true,
	}

	if !errorRecord.IsError {
		t.Errorf("Expected IsError true, got false")
	}
}

// TestAgentResultFields 测试 AgentResult 字段访问和零值 ✨
func TestAgentResultFields(t *testing.T) {
	// 测试有值的 AgentResult
	result := AgentResult{
		Content: "test content",
		Messages: []bamboo.BambooMessage{
			{Role: bamboo.RoleUser, Content: []bamboo.ContentBlock{bamboo.NewTextBlock("hello")}},
		},
		ToolCalls: []ToolCallRecord{
			{ID: "call-1", Name: "tool1"},
		},
		Usage: bamboo.Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
		Iterations: 3,
	}

	if result.Content != "test content" {
		t.Errorf("Expected Content 'test content', got '%s'", result.Content)
	}

	if len(result.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result.Messages))
	}

	if len(result.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(result.ToolCalls))
	}

	if result.Usage.InputTokens != 10 {
		t.Errorf("Expected InputTokens 10, got %d", result.Usage.InputTokens)
	}

	if result.Usage.OutputTokens != 20 {
		t.Errorf("Expected OutputTokens 20, got %d", result.Usage.OutputTokens)
	}

	if result.Iterations != 3 {
		t.Errorf("Expected Iterations 3, got %d", result.Iterations)
	}

	// 测试零值
	zeroResult := AgentResult{}

	if zeroResult.Content != "" {
		t.Errorf("Expected zero Content to be empty, got '%s'", zeroResult.Content)
	}

	if zeroResult.Messages != nil {
		t.Errorf("Expected zero Messages to be nil")
	}

	if zeroResult.ToolCalls != nil {
		t.Errorf("Expected zero ToolCalls to be nil")
	}

	if zeroResult.Iterations != 0 {
		t.Errorf("Expected zero Iterations to be 0, got %d", zeroResult.Iterations)
	}
}

// TestAgentEventFields 测试 AgentEvent 结构体字段 📦
func TestAgentEventFields(t *testing.T) {
	toolCall := &ToolCallRecord{ID: "call-1", Name: "test_tool"}
	result := &AgentResult{Content: "test result"}

	event := AgentEvent{
		Type:     AgentEventComplete,
		Content:  "event content",
		ToolCall: toolCall,
		Result:   result,
	}

	if event.Type != AgentEventComplete {
		t.Errorf("Expected Type AgentEventComplete, got %s", event.Type)
	}

	if event.Content != "event content" {
		t.Errorf("Expected Content 'event content', got '%s'", event.Content)
	}

	if event.ToolCall == nil {
		t.Error("Expected ToolCall to be non-nil")
	}

	if event.Result == nil {
		t.Error("Expected Result to be non-nil")
	}
}