package agent

import (
	"context"
	"encoding/json"
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

type streamingMockBambooClient struct {
	mockBambooClient
	chatFunc func(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error)
}

func (m *streamingMockBambooClient) Chat(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, messages, system, config)
	}
	ch := make(chan bamboo.StreamEvent)
	close(ch)
	return ch, nil
}

func (m *streamingMockBambooClient) Complete(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (*bamboo.Response, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, messages, system, config)
	}
	return &bamboo.Response{Content: []bamboo.ContentBlock{bamboo.NewTextBlock("mock")}}, nil
}

func pushTextResponse(ch chan<- bamboo.StreamEvent, text string) {
	ch <- bamboo.StreamEvent{
		Type:       bamboo.EventContentBlockStart,
		Index:      0,
		ContentBlock: &bamboo.ContentBlock{Type: bamboo.ContentBlockText},
	}
	ch <- bamboo.StreamEvent{
		Type:  bamboo.EventContentBlockDelta,
		Index: 0,
		Delta: &bamboo.StreamDelta{Type: bamboo.DeltaTextDelta, Text: text},
	}
	ch <- bamboo.StreamEvent{
		Type:  bamboo.EventContentBlockStop,
		Index: 0,
	}
	ch <- bamboo.StreamEvent{
		Type: bamboo.EventMessageDelta,
		Delta: &bamboo.MessageDelta{StopReason: bamboo.FinishReasonEndTurn},
		Usage: &bamboo.Usage{InputTokens: 10, OutputTokens: 20},
	}
	ch <- bamboo.StreamEvent{Type: bamboo.EventMessageStop}
}

func pushToolUseResponse(ch chan<- bamboo.StreamEvent, toolID, toolName, toolInput string) {
	ch <- bamboo.StreamEvent{
		Type:  bamboo.EventContentBlockStart,
		Index: 0,
		ContentBlock: &bamboo.ContentBlock{
			Type: bamboo.ContentBlockToolUse,
			ID:   toolID,
			Name: toolName,
		},
	}
	ch <- bamboo.StreamEvent{
		Type:  bamboo.EventContentBlockDelta,
		Index: 0,
		Delta: &bamboo.StreamDelta{Type: bamboo.DeltaInputJSON, PartialJSON: toolInput},
	}
	ch <- bamboo.StreamEvent{
		Type:  bamboo.EventContentBlockStop,
		Index: 0,
	}
	ch <- bamboo.StreamEvent{
		Type: bamboo.EventMessageDelta,
		Delta: &bamboo.MessageDelta{StopReason: bamboo.FinishReasonToolUse},
		Usage: &bamboo.Usage{InputTokens: 15, OutputTokens: 25},
	}
	ch <- bamboo.StreamEvent{Type: bamboo.EventMessageStop}
}

type mockTool struct {
	info   tool.ToolInfo
	result *tool.ToolResult
	err    error
}

func (m *mockTool) Info() tool.ToolInfo { return m.info }
func (m *mockTool) Execute(_ context.Context, _ json.RawMessage) (*tool.ToolResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// TestReActLoop_TextOnlyResponse 测试仅返回文本的响应。
func TestReActLoop_TextOnlyResponse(t *testing.T) {
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			ch := make(chan bamboo.StreamEvent, 10)
			go func() {
				defer close(ch)
				pushTextResponse(ch, "Hello! How can I help?")
			}()
			return ch, nil
		},
	}

	registry := tool.NewRegistry()
	session := NewSession(registry)
	executor := tool.NewToolExecutor(registry, 10)

	core := &agentCore{
		client:   mockClient,
		config:   DefaultConfig(),
		session:  session,
		registry: registry,
		executor: executor,
	}

	loop := NewReActLoop()
	result, err := loop.Execute(context.Background(), core, "Hi there")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "Hello! How can I help?" {
		t.Errorf("expected content 'Hello! How can I help?', got %q", result.Content)
	}
	if result.Iterations != 1 {
		t.Errorf("expected 1 iteration, got %d", result.Iterations)
	}
	if len(result.ToolCalls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(result.ToolCalls))
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("expected input tokens 10, got %d", result.Usage.InputTokens)
	}

	messages := session.GetMessages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", len(messages))
	}
	if messages[0].Role != bamboo.RoleUser {
		t.Errorf("message[0] role should be user, got %s", messages[0].Role)
	}
	if messages[1].Role != bamboo.RoleAssistant {
		t.Errorf("message[1] role should be assistant, got %s", messages[1].Role)
	}
}

// TestReActLoop_ToolCallAndLoop 测试工具调用和循环。
func TestReActLoop_ToolCallAndLoop(t *testing.T) {
	callCount := 0
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			callCount++
			ch := make(chan bamboo.StreamEvent, 20)
			go func() {
				defer close(ch)
				if callCount == 1 {
					pushToolUseResponse(ch, "call_1", "calculator", `{"expr":"2+2"}`)
				} else {
					pushTextResponse(ch, "The result is 4.")
				}
			}()
			return ch, nil
		},
	}

	registry := tool.NewRegistry()
	registry.Register(&mockTool{
		info:   tool.ToolInfo{Name: "calculator", Description: "does math"},
		result: &tool.ToolResult{Content: "4"},
	})

	session := NewSession(registry)
	executor := tool.NewToolExecutor(registry, 10)

	core := &agentCore{
		client:   mockClient,
		config:   DefaultConfig(),
		session:  session,
		registry: registry,
		executor: executor,
	}

	loop := NewReActLoop()
	result, err := loop.Execute(context.Background(), core, "What is 2+2?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "The result is 4." {
		t.Errorf("expected final content 'The result is 4.', got %q", result.Content)
	}
	if result.Iterations != 2 {
		t.Errorf("expected 2 iterations, got %d", result.Iterations)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Name != "calculator" {
		t.Errorf("expected tool name 'calculator', got %q", result.ToolCalls[0].Name)
	}
	if result.ToolCalls[0].Result != "4" {
		t.Errorf("expected tool result '4', got %q", result.ToolCalls[0].Result)
	}
	if result.ToolCalls[0].IsError {
		t.Error("tool call should not be an error")
	}

	messages := session.GetMessages()
	if len(messages) != 4 {
		t.Fatalf("expected 4 messages (user + assistant[w/tool] + tool_result + assistant), got %d", len(messages))
	}
}

// TestReActLoop_MaxIterations 测试最大迭代次数限制。
func TestReActLoop_MaxIterations(t *testing.T) {
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			ch := make(chan bamboo.StreamEvent, 20)
			go func() {
				defer close(ch)
				pushToolUseResponse(ch, "call_loop", "loop_tool", `{"n":1}`)
			}()
			return ch, nil
		},
	}

	registry := tool.NewRegistry()
	registry.Register(&mockTool{
		info:   tool.ToolInfo{Name: "loop_tool", Description: "always loops"},
		result: &tool.ToolResult{Content: "looped"},
	})

	session := NewSession(registry)
	executor := tool.NewToolExecutor(registry, 10)

	cfg := DefaultConfig()
	cfg.MaxIterations = 3

	core := &agentCore{
		client:   mockClient,
		config:   cfg,
		session:  session,
		registry: registry,
		executor: executor,
	}

	loop := NewReActLoop()
	result, err := loop.Execute(context.Background(), core, "keep looping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Iterations != 3 {
		t.Errorf("expected 3 iterations (max), got %d", result.Iterations)
	}
	if len(result.ToolCalls) != 3 {
		t.Errorf("expected 3 tool calls (one per iteration), got %d", len(result.ToolCalls))
	}
}

// TestReActLoop_ChatError 测试 Chat 错误处理。
func TestReActLoop_ChatError(t *testing.T) {
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			return nil, context.DeadlineExceeded
		},
	}

	registry := tool.NewRegistry()
	session := NewSession(registry)
	executor := tool.NewToolExecutor(registry, 10)

	core := &agentCore{
		client:   mockClient,
		config:   DefaultConfig(),
		session:  session,
		registry: registry,
		executor: executor,
	}

	loop := NewReActLoop()
	_, err := loop.Execute(context.Background(), core, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestReActLoop_StreamError 测试流错误处理。
func TestReActLoop_StreamError(t *testing.T) {
	mockClient := &streamingMockBambooClient{
		chatFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
			ch := make(chan bamboo.StreamEvent, 5)
			go func() {
				defer close(ch)
				ch <- bamboo.StreamEvent{
					Type:  bamboo.EventError,
					Error: bamboo.NewBambooError("api_error", "something broke"),
				}
			}()
			return ch, nil
		},
	}

	registry := tool.NewRegistry()
	session := NewSession(registry)
	executor := tool.NewToolExecutor(registry, 10)

	core := &agentCore{
		client:   mockClient,
		config:   DefaultConfig(),
		session:  session,
		registry: registry,
		executor: executor,
	}

	loop := NewReActLoop()
	_, err := loop.Execute(context.Background(), core, "test")
	if err == nil {
		t.Fatal("expected error from stream error event, got nil")
	}
}
