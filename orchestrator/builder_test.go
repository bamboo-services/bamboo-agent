package orchestrator

import (
	"context"
	"encoding/json"
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/agent"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// mockBambooClient implements bamboo.BambooClient for testing
type mockBambooClient struct{}

func (m *mockBambooClient) Chat(ctx context.Context, messages []bamboo.BambooMessage, systemPrompt string, config *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
	ch := make(chan bamboo.StreamEvent, 1)
	close(ch)
	return ch, nil
}

func (m *mockBambooClient) Complete(ctx context.Context, messages []bamboo.BambooMessage, systemPrompt string, config *bamboo.RequestConfig) (*bamboo.Response, error) {
	return &bamboo.Response{}, nil
}

// mockTool implements tool.Tool for testing
type mockTool struct {
	info tool.ToolInfo
}

func (t *mockTool) Info() tool.ToolInfo {
	return t.info
}

func (t *mockTool) Execute(ctx context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	return &tool.ToolResult{Content: "ok"}, nil
}

// TestBuilder_FullBuild 测试完整构建 Agent。
func TestBuilder_FullBuild(t *testing.T) {
	client := &mockBambooClient{}
	mockTool1 := &mockTool{info: tool.ToolInfo{Name: "tool1"}}
	mockTool2 := &mockTool{info: tool.ToolInfo{Name: "tool2"}}

	agent := NewAgentBuilder().
		WithClient(client).
		WithConfig(agent.AgentConfig{
			Model:         "test-model",
			MaxTokens:     1000,
			MaxIterations: 5,
		}).
		WithSystemPrompt("custom prompt").
		WithTools(mockTool1, mockTool2).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}

	agent.SetSystemPrompt("test")
	agent.SetSystemPrompt("custom prompt")
}

// TestBuilder_MissingClientPanics 测试缺少 Client 时panic。
func TestBuilder_MissingClientPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic when client is missing")
		}
		if r != "AgentBuilder: BambooClient is required, call WithClient() first" {
			t.Errorf("Unexpected panic message: %v", r)
		}
	}()

	NewAgentBuilder().Build()
}

// TestBuilder_DefaultConfig 测试默认配置。
func TestBuilder_DefaultConfig(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

// TestBuilder_SystemPromptOverride 测试系统提示覆盖。
func TestBuilder_SystemPromptOverride(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		WithSystemPrompt("custom system prompt").
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}

	agent.SetSystemPrompt("custom system prompt")
}

// TestBuilder_ToolsRegistration 测试工具注册。
func TestBuilder_ToolsRegistration(t *testing.T) {
	client := &mockBambooClient{}
	mockTool1 := &mockTool{info: tool.ToolInfo{Name: "tool1"}}
	mockTool2 := &mockTool{info: tool.ToolInfo{Name: "tool2"}}

	agent := NewAgentBuilder().
		WithClient(client).
		WithTools(mockTool1).
		WithTools(mockTool2).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}

	err := agent.AddTool(&mockTool{info: tool.ToolInfo{Name: "tool3"}})
	if err != nil {
		t.Errorf("Failed to add tool after build: %v", err)
	}
}

// TestBuilder_ChainedFluentCalls 测试链式调用。
func TestBuilder_ChainedFluentCalls(t *testing.T) {
	client := &mockBambooClient{}
	mockTool := &mockTool{info: tool.ToolInfo{Name: "tool"}}

	agent := NewAgentBuilder().
		WithClient(client).
		WithSystemPrompt("prompt").
		WithTools(mockTool).
		WithConfig(agent.AgentConfig{MaxIterations: 3}).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

// TestBuilder_EmptyBuilderJustClient 测试仅设置 Client。
func TestBuilder_EmptyBuilderJustClient(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

// TestBuilder_MultipleWithTools 测试多次 WithTools。
func TestBuilder_MultipleWithTools(t *testing.T) {
	client := &mockBambooClient{}
	mockTool1 := &mockTool{info: tool.ToolInfo{Name: "tool1"}}
	mockTool2 := &mockTool{info: tool.ToolInfo{Name: "tool2"}}
	mockTool3 := &mockTool{info: tool.ToolInfo{Name: "tool3"}}

	agent := NewAgentBuilder().
		WithClient(client).
		WithTools(mockTool1, mockTool2).
		WithTools(mockTool3).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

// TestBuilder_ConfigOverride 测试配置覆盖。
func TestBuilder_ConfigOverride(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		WithConfig(agent.AgentConfig{
			Model:         "custom-model",
			MaxTokens:     2000,
			MaxIterations: 20,
		}).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

// TestBuilder_NullSafeReturns 测试空安全返回。
func TestBuilder_NullSafeReturns(t *testing.T) {
	client := &mockBambooClient{}
	builder := NewAgentBuilder()

	if builder.WithClient(client) == nil {
		t.Error("WithClient() returned nil")
	}
	if builder.WithConfig(agent.AgentConfig{}) == nil {
		t.Error("WithConfig() returned nil")
	}
	if builder.WithSystemPrompt("test") == nil {
		t.Error("WithSystemPrompt() returned nil")
	}
	if builder.WithTools() == nil {
		t.Error("WithTools() returned nil")
	}
}

// TestBuilder_WithSystemPromptEmptyString 测试空字符串提示。
func TestBuilder_WithSystemPromptEmptyString(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		WithSystemPrompt("").
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

// TestBuilder_ReusableBuilder 测试可重用 Builder。
func TestBuilder_ReusableBuilder(t *testing.T) {
	client := &mockBambooClient{}

	builder := NewAgentBuilder()

	agent1 := builder.WithClient(client).Build()
	if agent1 == nil {
		t.Fatal("First Build() returned nil")
	}

	client2 := &mockBambooClient{}
	agent2 := builder.WithClient(client2).Build()
	if agent2 == nil {
		t.Fatal("Second Build() returned nil")
	}

	if agent1 == agent2 {
		t.Error("Expected different agent instances")
	}
}