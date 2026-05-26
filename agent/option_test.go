package agent

import (
	"context"
	"encoding/json"
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// mockLoopStrategy 是用于测试的简单 LoopStrategy 实现。
type mockLoopStrategy struct{}

func (m *mockLoopStrategy) Execute(ctx context.Context, core *agentCore, input string) (*AgentResult, error) {
	return &AgentResult{Content: "mock loop result", Iterations: 1}, nil
}

// mockCompressor 是用于测试的简单 ContextCompressor 实现。
type mockCompressor struct{}

func (m *mockCompressor) Compress(ctx context.Context, messages []bamboo.BambooMessage, maxTokens int64) ([]bamboo.BambooMessage, error) {
	return messages, nil
}

// testMockTool 是用于测试的简单工具实现。
type testMockTool struct {
	info   tool.ToolInfo
	result *tool.ToolResult
	err    error
}

func (m *testMockTool) Info() tool.ToolInfo { return m.info }
func (m *testMockTool) Execute(_ context.Context, _ json.RawMessage) (*tool.ToolResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// TestOption_WithSystemPrompt 测试 WithSystemPrompt 选项。
func TestOption_WithSystemPrompt(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentWithOptions(client, WithSystemPrompt("You are a helpful assistant."))

	core := agent.(*agentCore)
	if core.config.SystemPrompt != "You are a helpful assistant." {
		t.Errorf("expected SystemPrompt 'You are a helpful assistant.', got %q", core.config.SystemPrompt)
	}
}

// TestOption_WithConfig 测试 WithConfig 选项。
func TestOption_WithConfig(t *testing.T) {
	client := &mockBambooClient{}

	customConfig := AgentConfig{
		Model:              "custom-model",
		MaxTokens:          8192,
		MaxIterations:      20,
		MaxConcurrentTools: 5,
		MaxContextTokens:   200000,
	}

	agent := NewAgentWithOptions(client, WithConfig(customConfig))

	core := agent.(*agentCore)
	if core.config.Model != "custom-model" {
		t.Errorf("expected Model 'custom-model', got %q", core.config.Model)
	}
	if core.config.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens 8192, got %d", core.config.MaxTokens)
	}
	if core.config.MaxIterations != 20 {
		t.Errorf("expected MaxIterations 20, got %d", core.config.MaxIterations)
	}
	if core.config.MaxConcurrentTools != 5 {
		t.Errorf("expected MaxConcurrentTools 5, got %d", core.config.MaxConcurrentTools)
	}
	if core.config.MaxContextTokens != 200000 {
		t.Errorf("expected MaxContextTokens 200000, got %d", core.config.MaxContextTokens)
	}
}

// TestOption_WithMaxIterations 测试 WithMaxIterations 选项。
func TestOption_WithMaxIterations(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentWithOptions(client, WithMaxIterations(15))

	core := agent.(*agentCore)
	if core.config.MaxIterations != 15 {
		t.Errorf("expected MaxIterations 15, got %d", core.config.MaxIterations)
	}
}

// TestOption_WithMaxTokens 测试 WithMaxTokens 选项。
func TestOption_WithMaxTokens(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentWithOptions(client, WithMaxTokens(2048))

	core := agent.(*agentCore)
	if core.config.MaxTokens != 2048 {
		t.Errorf("expected MaxTokens 2048, got %d", core.config.MaxTokens)
	}
}

// TestOption_WithTemperature 测试 WithTemperature 选项。
func TestOption_WithTemperature(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentWithOptions(client, WithTemperature(0.9))

	core := agent.(*agentCore)
	if core.config.Temperature == nil {
		t.Fatal("expected Temperature to be set, got nil")
	}
	if *core.config.Temperature != 0.9 {
		t.Errorf("expected Temperature 0.9, got %f", *core.config.Temperature)
	}
}

// TestOption_WithMaxConcurrentTools 测试 WithMaxConcurrentTools 选项。
func TestOption_WithMaxConcurrentTools(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentWithOptions(client, WithMaxConcurrentTools(3))

	core := agent.(*agentCore)
	if core.config.MaxConcurrentTools != 3 {
		t.Errorf("expected MaxConcurrentTools 3, got %d", core.config.MaxConcurrentTools)
	}
}

// TestOption_WithLoopStrategy 测试 WithLoopStrategy 选项。
func TestOption_WithLoopStrategy(t *testing.T) {
	client := &mockBambooClient{}
	strategy := &mockLoopStrategy{}

	agent := NewAgentWithOptions(client, WithLoopStrategy(strategy))

	core := agent.(*agentCore)
	if core.config.LoopStrategy == nil {
		t.Fatal("expected LoopStrategy to be set, got nil")
	}
}

// TestOption_WithCompressor 测试 WithCompressor 选项。
func TestOption_WithCompressor(t *testing.T) {
	client := &mockBambooClient{}
	compressor := &mockCompressor{}

	agent := NewAgentWithOptions(client, WithCompressor(compressor))

	core := agent.(*agentCore)
	if core.config.Compressor == nil {
		t.Fatal("expected Compressor to be set, got nil")
	}
}

// TestOption_WithTools 测试 WithTools 选项。
func TestOption_WithTools(t *testing.T) {
	client := &mockBambooClient{}

	tool1 := &testMockTool{
		info:   tool.ToolInfo{Name: "tool1", Description: "first tool"},
		result: &tool.ToolResult{Content: "result1"},
	}
	tool2 := &testMockTool{
		info:   tool.ToolInfo{Name: "tool2", Description: "second tool"},
		result: &tool.ToolResult{Content: "result2"},
	}

	agent := NewAgentWithOptions(client, WithTools(tool1, tool2))

	core := agent.(*agentCore)
	tools := core.registry.List()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools registered, got %d", len(tools))
	}

	// Check that tool names match
	toolNames := make(map[string]bool)
	for _, info := range tools {
		toolNames[info.Name] = true
	}
	if !toolNames["tool1"] {
		t.Error("expected tool1 to be registered")
	}
	if !toolNames["tool2"] {
		t.Error("expected tool2 to be registered")
	}
}

// TestOption_NoOptions 测试不使用任何选项时的默认行为。
func TestOption_NoOptions(t *testing.T) {
	client := &mockBambooClient{}

	agent := NewAgentWithOptions(client)

	core := agent.(*agentCore)
	defaults := DefaultConfig()

	// Verify default values are used
	if core.config.Model != defaults.Model {
		t.Errorf("expected default Model %q, got %q", defaults.Model, core.config.Model)
	}
	if core.config.MaxTokens != defaults.MaxTokens {
		t.Errorf("expected default MaxTokens %d, got %d", defaults.MaxTokens, core.config.MaxTokens)
	}
	if core.config.MaxIterations != defaults.MaxIterations {
		t.Errorf("expected default MaxIterations %d, got %d", defaults.MaxIterations, core.config.MaxIterations)
	}
	if core.config.MaxConcurrentTools != defaults.MaxConcurrentTools {
		t.Errorf("expected default MaxConcurrentTools %d, got %d", defaults.MaxConcurrentTools, core.config.MaxConcurrentTools)
	}
	if core.config.MaxContextTokens != defaults.MaxContextTokens {
		t.Errorf("expected default MaxContextTokens %d, got %d", defaults.MaxContextTokens, core.config.MaxContextTokens)
	}
}

// TestOption_ChainedOptions 测试链式使用多个选项。
func TestOption_ChainedOptions(t *testing.T) {
	client := &mockBambooClient{}
	strategy := &mockLoopStrategy{}
	compressor := &mockCompressor{}

	temp := 0.8

	agent := NewAgentWithOptions(
		client,
		WithSystemPrompt("Custom prompt"),
		WithMaxIterations(25),
		WithMaxTokens(1024),
		WithTemperature(temp),
		WithMaxConcurrentTools(7),
		WithLoopStrategy(strategy),
		WithCompressor(compressor),
	)

	core := agent.(*agentCore)

	if core.config.SystemPrompt != "Custom prompt" {
		t.Errorf("expected SystemPrompt 'Custom prompt', got %q", core.config.SystemPrompt)
	}
	if core.config.MaxIterations != 25 {
		t.Errorf("expected MaxIterations 25, got %d", core.config.MaxIterations)
	}
	if core.config.MaxTokens != 1024 {
		t.Errorf("expected MaxTokens 1024, got %d", core.config.MaxTokens)
	}
	if core.config.Temperature == nil {
		t.Fatal("expected Temperature to be set, got nil")
	}
	if *core.config.Temperature != 0.8 {
		t.Errorf("expected Temperature 0.8, got %f", *core.config.Temperature)
	}
	if core.config.MaxConcurrentTools != 7 {
		t.Errorf("expected MaxConcurrentTools 7, got %d", core.config.MaxConcurrentTools)
	}
	if core.config.LoopStrategy == nil {
		t.Error("expected LoopStrategy to be set")
	}
	if core.config.Compressor == nil {
		t.Error("expected Compressor to be set")
	}
}

// TestOption_WithConfigAndIndividualOptions 测试配置和单个选项的优先级。
func TestOption_WithConfigAndIndividualOptions(t *testing.T) {
	client := &mockBambooClient{}

	// WithConfig should apply first, then individual options can override
	customConfig := AgentConfig{
		Model:              "base-model",
		MaxTokens:          4096,
		MaxIterations:      10,
		MaxConcurrentTools: 10,
	}

	agent := NewAgentWithOptions(
		client,
		WithConfig(customConfig),
		WithMaxTokens(8192),  // Override MaxTokens
		WithMaxIterations(20), // Override MaxIterations
	)

	core := agent.(*agentCore)

	// Base model and MaxConcurrentTools should remain from WithConfig
	if core.config.Model != "base-model" {
		t.Errorf("expected Model 'base-model', got %q", core.config.Model)
	}
	if core.config.MaxConcurrentTools != 10 {
		t.Errorf("expected MaxConcurrentTools 10, got %d", core.config.MaxConcurrentTools)
	}

	// Overridden values should take precedence
	if core.config.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens 8192 (overridden), got %d", core.config.MaxTokens)
	}
	if core.config.MaxIterations != 20 {
		t.Errorf("expected MaxIterations 20 (overridden), got %d", core.config.MaxIterations)
	}
}

// TestOption_WithSystemPromptOverridesConfig 测试 SystemPrompt 选项覆盖配置。
func TestOption_WithSystemPromptOverridesConfig(t *testing.T) {
	client := &mockBambooClient{}

	config := AgentConfig{
		SystemPrompt: "Config prompt",
	}

	// WithSystemPrompt should override config's SystemPrompt
	agent := NewAgentWithOptions(
		client,
		WithConfig(config),
		WithSystemPrompt("Option prompt"),
	)

	core := agent.(*agentCore)
	if core.config.SystemPrompt != "Option prompt" {
		t.Errorf("expected SystemPrompt 'Option prompt' (from option), got %q", core.config.SystemPrompt)
	}
}