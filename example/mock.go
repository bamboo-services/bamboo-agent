package main

import (
	"context"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// mockBambooClient 模拟 AI 服务端的流式响应。
//
// 通过 chatFunc 闭包模拟不同轮次的 AI 行为：
//   - 返回 tool_use 触发工具调用
//   - 返回文本内容作为最终回复
//
// 默认行为返回空流，设置 chatFunc 后可自定义每轮响应。
type mockBambooClient struct {
	// chatFunc 定义每次 Chat 调用的行为。
	// 闭包可捕获外部状态（如调用计数器）来模拟多轮对话。
	chatFunc func(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error)
}

// Chat 发起流式对话，委托给 chatFunc 或返回空流。
func (m *mockBambooClient) Chat(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, messages, system, config)
	}
	ch := make(chan bamboo.StreamEvent)
	close(ch)
	return ch, nil
}

// Complete 发起非流式对话，返回默认模拟响应。
func (m *mockBambooClient) Complete(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (*bamboo.Response, error) {
	return &bamboo.Response{
		Content: []bamboo.ContentBlock{bamboo.NewTextBlock("mock response")},
	}, nil
}

// pushTextResponse 向流中推送文本响应的完整事件序列。
//
// 模拟 AI 返回纯文本回复时的标准事件流：
//
//	content_block_start → content_block_delta(text) → content_block_stop
//	→ message_delta(end_turn) → message_stop
func pushTextResponse(ch chan<- bamboo.StreamEvent, text string) {
	ch <- bamboo.StreamEvent{
		Type:         bamboo.EventContentBlockStart,
		Index:        0,
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
		Type:  bamboo.EventMessageDelta,
		Delta: &bamboo.MessageDelta{StopReason: bamboo.FinishReasonEndTurn},
		Usage: &bamboo.Usage{InputTokens: 10, OutputTokens: 20},
	}
	ch <- bamboo.StreamEvent{Type: bamboo.EventMessageStop}
}

// pushToolUseResponse 向流中推送工具调用响应的完整事件序列。
//
// 模拟 AI 请求调用工具时的标准事件流：
//
//	content_block_start(tool_use) → content_block_delta(input_json) → content_block_stop
//	→ message_delta(tool_use) → message_stop
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
		Type:  bamboo.EventMessageDelta,
		Delta: &bamboo.MessageDelta{StopReason: bamboo.FinishReasonToolUse},
		Usage: &bamboo.Usage{InputTokens: 15, OutputTokens: 25},
	}
	ch <- bamboo.StreamEvent{Type: bamboo.EventMessageStop}
}
