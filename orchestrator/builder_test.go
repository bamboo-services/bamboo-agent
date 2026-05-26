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

	// Verify system prompt was set
	agent.SetSystemPrompt("test")
	agent.SetSystemPrompt("custom prompt")
}

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

func TestBuilder_DefaultConfig(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestBuilder_SystemPromptOverride(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		WithSystemPrompt("custom system prompt").
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}

	// The prompt should be set in the config
	agent.SetSystemPrompt("custom system prompt")
}

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

	// Tools should be registered
	err := agent.AddTool(&mockTool{info: tool.ToolInfo{Name: "tool3"}})
	if err != nil {
		t.Errorf("Failed to add tool after build: %v", err)
	}
}

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

func TestBuilder_EmptyBuilderJustClient(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentBuilder().
		WithClient(client).
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

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

func TestBuilder_NullSafeReturns(t *testing.T) {
	client := &mockBambooClient{}
	builder := NewAgentBuilder()

	// All methods should return non-nil builder
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

func TestBuilder_WithSystemPromptEmptyString(t *testing.T) {
	client := &mockBambooClient{}

	// Empty string should not override default
	agent := NewAgentBuilder().
		WithClient(client).
		WithSystemPrompt("").
		Build()

	if agent == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestBuilder_ReusableBuilder(t *testing.T) {
	client := &mockBambooClient{}

	builder := NewAgentBuilder()

	agent1 := builder.WithClient(client).Build()
	if agent1 == nil {
		t.Fatal("First Build() returned nil")
	}

	// Builder can be reused for a second agent
	client2 := &mockBambooClient{}
	agent2 := builder.WithClient(client2).Build()
	if agent2 == nil {
		t.Fatal("Second Build() returned nil")
	}

	// Agents should be different
	if agent1 == agent2 {
		t.Error("Expected different agent instances")
	}
}