package agent

import (
	"context"
	"encoding/json"
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// Compile-time check: agentCore must satisfy the Agent interface.
func TestAgentInterfaceSatisfaction(t *testing.T) {
	var _ Agent = (*agentCore)(nil)
}

// TestAgent_AddTool verifies that AddTool registers a tool in the registry.
func TestAgent_AddTool(t *testing.T) {
	mockClient := &mockBambooClient{}

	agent := NewAgent(mockClient, DefaultConfig())

	mt := &mockTool{
		info:   tool.ToolInfo{Name: "test_tool", Description: "a test tool"},
		result: &tool.ToolResult{Content: "ok"},
	}

	if err := agent.AddTool(mt); err != nil {
		t.Fatalf("AddTool returned error: %v", err)
	}

	// Verify tool is registered by checking the internal registry.
	core := agent.(*agentCore)
	tools := core.registry.List()
	if len(tools) != 1 {
		t.Fatalf("expected 1 registered tool, got %d", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", tools[0].Name)
	}
}

// TestAgent_SetSystemPrompt verifies that SetSystemPrompt updates the config.
func TestAgent_SetSystemPrompt(t *testing.T) {
	mockClient := &mockBambooClient{}

	agent := NewAgent(mockClient, DefaultConfig())
	core := agent.(*agentCore)

	if core.config.SystemPrompt != "" {
		t.Errorf("expected empty system prompt initially, got %q", core.config.SystemPrompt)
	}

	agent.SetSystemPrompt("You are a helpful assistant.")

	if core.config.SystemPrompt != "You are a helpful assistant." {
		t.Errorf("expected system prompt 'You are a helpful assistant.', got %q", core.config.SystemPrompt)
	}
}

// TestAgent_NewAgent_DefaultStrategy verifies NewAgent works with nil LoopStrategy.
func TestAgent_NewAgent_DefaultStrategy(t *testing.T) {
	mockClient := &mockBambooClient{}
	cfg := DefaultConfig()
	// LoopStrategy is nil by default in DefaultConfig.

	agent := NewAgent(mockClient, cfg)
	core := agent.(*agentCore)

	if core.client == nil {
		t.Error("expected client to be set")
	}
	if core.session == nil {
		t.Error("expected session to be created")
	}
	if core.registry == nil {
		t.Error("expected registry to be created")
	}
	if core.executor == nil {
		t.Error("expected executor to be created")
	}
}

// TestAgent_Run_TextResponse verifies Run delegates to LoopStrategy and returns result.
func TestAgent_Run_TextResponse(t *testing.T) {
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			ch := make(chan bamboo.StreamEvent, 10)
			go func() {
				defer close(ch)
				pushTextResponse(ch, "Hello from agent!")
			}()
			return ch, nil
		},
	}

	agent := NewAgent(mockClient, DefaultConfig())
	result, err := agent.Run(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "Hello from agent!" {
		t.Errorf("expected content 'Hello from agent!', got %q", result.Content)
	}
	if result.Iterations != 1 {
		t.Errorf("expected 1 iteration, got %d", result.Iterations)
	}
}

// TestAgent_Run_WithToolCall verifies Run handles tool calls through the agent.
func TestAgent_Run_WithToolCall(t *testing.T) {
	callCount := 0
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			callCount++
			ch := make(chan bamboo.StreamEvent, 20)
			go func() {
				defer close(ch)
				if callCount == 1 {
					pushToolUseResponse(ch, "call_1", "weather", `{"city":"Tokyo"}`)
				} else {
					pushTextResponse(ch, "The weather in Tokyo is sunny.")
				}
			}()
			return ch, nil
		},
	}

	agent := NewAgent(mockClient, DefaultConfig())
	agent.AddTool(&mockTool{
		info:   tool.ToolInfo{Name: "weather", Description: "gets weather"},
		result: &tool.ToolResult{Content: "sunny, 25°C"},
	})

	result, err := agent.Run(context.Background(), "What's the weather in Tokyo?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "The weather in Tokyo is sunny." {
		t.Errorf("expected 'The weather in Tokyo is sunny.', got %q", result.Content)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Name != "weather" {
		t.Errorf("expected tool 'weather', got %q", result.ToolCalls[0].Name)
	}
	if result.ToolCalls[0].Result != "sunny, 25°C" {
		t.Errorf("expected result 'sunny, 25°C', got %q", result.ToolCalls[0].Result)
	}
}

// TestAgent_Stream verifies Stream emits events correctly.
func TestAgent_Stream(t *testing.T) {
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			ch := make(chan bamboo.StreamEvent, 10)
			go func() {
				defer close(ch)
				pushTextResponse(ch, "streamed response")
			}()
			return ch, nil
		},
	}

	agent := NewAgent(mockClient, DefaultConfig())
	ch, err := agent.Stream(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var events []AgentEvent
	for evt := range ch {
		events = append(events, evt)
	}

	// Expect: text event + complete event
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Type != AgentEventText {
		t.Errorf("event[0] type should be text, got %s", events[0].Type)
	}
	if events[0].Content != "streamed response" {
		t.Errorf("event[0] content should be 'streamed response', got %q", events[0].Content)
	}

	if events[1].Type != AgentEventComplete {
		t.Errorf("event[1] type should be complete, got %s", events[1].Type)
	}
	if events[1].Result == nil {
		t.Fatal("event[1] result should not be nil")
	}
}

// TestAgent_Stream_WithError verifies Stream emits error event on failure.
func TestAgent_Stream_WithError(t *testing.T) {
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			return nil, context.DeadlineExceeded
		},
	}

	agent := NewAgent(mockClient, DefaultConfig())
	ch, err := agent.Stream(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error creating stream: %v", err)
	}

	var events []AgentEvent
	for evt := range ch {
		events = append(events, evt)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event (error), got %d", len(events))
	}
	if events[0].Type != AgentEventError {
		t.Errorf("expected error event, got %s", events[0].Type)
	}
	if events[0].Error == nil {
		t.Error("expected non-nil error in event")
	}
}

// TestAgent_RunWithMessages verifies RunWithMessages with pre-loaded messages.
func TestAgent_RunWithMessages(t *testing.T) {
	var capturedMessages []bamboo.BambooMessage
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, messages []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			capturedMessages = messages
			ch := make(chan bamboo.StreamEvent, 10)
			go func() {
				defer close(ch)
				pushTextResponse(ch, "I understand your history.")
			}()
			return ch, nil
		},
	}

	agent := NewAgent(mockClient, DefaultConfig())

	messages := []bamboo.BambooMessage{
		bamboo.NewUserMessage("First question"),
		bamboo.NewAssistantMessage("First answer"),
		bamboo.NewUserMessage("Follow-up question"),
	}

	result, err := agent.RunWithMessages(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "I understand your history." {
		t.Errorf("expected 'I understand your history.', got %q", result.Content)
	}

	// The loop should have received at least the user message.
	if len(capturedMessages) < 1 {
		t.Fatal("expected at least 1 message sent to AI")
	}
}

// TestAgent_RunWithMessages_EmptyHistory verifies RunWithMessages with no user messages.
func TestAgent_RunWithMessages_EmptyHistory(t *testing.T) {
	mockClient := &mockBambooClient{}

	agent := NewAgent(mockClient, DefaultConfig())

	messages := []bamboo.BambooMessage{
		bamboo.NewAssistantMessage("No user message here"),
	}

	result, err := agent.RunWithMessages(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return with zero iterations since no user input found.
	if result.Iterations != 0 {
		t.Errorf("expected 0 iterations, got %d", result.Iterations)
	}
}

// TestAgent_AddTool_Duplicate verifies duplicate tool registration returns error.
func TestAgent_AddTool_Duplicate(t *testing.T) {
	mockClient := &mockBambooClient{}
	agent := NewAgent(mockClient, DefaultConfig())

	mt := &mockTool{
		info:   tool.ToolInfo{Name: "dup_tool", Description: "duplicate"},
		result: &tool.ToolResult{Content: "ok"},
	}

	if err := agent.AddTool(mt); err != nil {
		t.Fatalf("first AddTool should succeed: %v", err)
	}

	if err := agent.AddTool(mt); err == nil {
		t.Fatal("second AddTool with same name should return error")
	}
}

// TestAgent_Run_UsesSystemPrompt verifies the system prompt is passed through.
func TestAgent_Run_UsesSystemPrompt(t *testing.T) {
	var capturedSystem string
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, system string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			capturedSystem = system
			ch := make(chan bamboo.StreamEvent, 10)
			go func() {
				defer close(ch)
				pushTextResponse(ch, "ack")
			}()
			return ch, nil
		},
	}

	cfg := DefaultConfig()
	cfg.SystemPrompt = "You are a pirate."

	agent := NewAgent(mockClient, cfg)
	_, err := agent.Run(context.Background(), "ahoy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedSystem != "You are a pirate." {
		t.Errorf("expected system prompt 'You are a pirate.', got %q", capturedSystem)
	}
}

// TestAgent_Run_WithCustomStrategy verifies a custom LoopStrategy is used.
func TestAgent_Run_WithCustomStrategy(t *testing.T) {
	mockClient := &mockBambooClient{}

	strategy := &customTestStrategy{}
	cfg := DefaultConfig()
	cfg.LoopStrategy = strategy

	agent := NewAgent(mockClient, cfg)
	result, err := agent.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strategy.called {
		t.Error("expected custom strategy Execute to be called")
	}
	if strategy.capturedInput != "test input" {
		t.Errorf("expected input 'test input', got %q", strategy.capturedInput)
	}
	if result.Content != "custom result" {
		t.Errorf("expected 'custom result', got %q", result.Content)
	}
}

// customTestStrategy is a test double for LoopStrategy.
type customTestStrategy struct {
	called         bool
	capturedInput  string
	capturedCore   *agentCore
}

func (s *customTestStrategy) Execute(_ context.Context, core *agentCore, input string) (*AgentResult, error) {
	s.called = true
	s.capturedInput = input
	s.capturedCore = core
	return &AgentResult{
		Content:    "custom result",
		Messages:   []bamboo.BambooMessage{},
		Iterations: 1,
	}, nil
}

// Ensure unused imports are satisfied (json.RawMessage used by mockTool).
var _ json.RawMessage
