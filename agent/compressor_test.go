package agent

import (
	"context"
	"strings"
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// mockBambooClient 是用于测试的 BambooClient 模拟实现。
type mockBambooClient struct {
	completeFunc func(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (*bamboo.Response, error)
}

func (m *mockBambooClient) Chat(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (<-chan bamboo.StreamEvent, error) {
	return nil, nil
}

func (m *mockBambooClient) Complete(ctx context.Context, messages []bamboo.BambooMessage, system string, config *bamboo.RequestConfig) (*bamboo.Response, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, messages, system, config)
	}
	return &bamboo.Response{
		Content: []bamboo.ContentBlock{
			bamboo.NewTextBlock("mock summary"),
		},
	}, nil
}

// buildMessages 构建指定数量的交替对话消息（user + assistant 成对）。
func buildMessages(count int) []bamboo.BambooMessage {
	messages := make([]bamboo.BambooMessage, 0, count)
	for i := range count {
		role := bamboo.RoleUser
		text := "user message"
		if i%2 == 1 {
			role = bamboo.RoleAssistant
			text = "assistant reply"
		}
		messages = append(messages, bamboo.BambooMessage{
			Role:    role,
			Content: []bamboo.ContentBlock{bamboo.NewTextBlock(text)},
		})
	}
	return messages
}

// TestSummaryCompressor_NoCompression 测试消息数量不足时不执行压缩
func TestSummaryCompressor_NoCompression(t *testing.T) {
	mock := &mockBambooClient{}
	compressor := NewSummaryCompressor(mock)

	// keepRecent=2 → keepCount=4，少于等于 4 条消息不压缩
	messages := buildMessages(4)

	result, err := compressor.Compress(context.Background(), messages, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 应该原样返回
	if len(result) != len(messages) {
		t.Errorf("expected %d messages, got %d", len(messages), len(result))
	}

	// 验证每条消息内容不变
	for i, msg := range result {
		if msg.Role != messages[i].Role {
			t.Errorf("message[%d] role changed: expected %s, got %s", i, messages[i].Role, msg.Role)
		}
	}
}

// TestSummaryCompressor_NoCompression_LessThanThreshold 测试消息远少于阈值时不压缩
func TestSummaryCompressor_NoCompression_LessThanThreshold(t *testing.T) {
	mock := &mockBambooClient{}
	compressor := NewSummaryCompressor(mock)

	messages := buildMessages(2)

	result, err := compressor.Compress(context.Background(), messages, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 messages unchanged, got %d", len(result))
	}
}

// TestSummaryCompressor_CompressesOldMessages 测试超过阈值时压缩旧消息
func TestSummaryCompressor_CompressesOldMessages(t *testing.T) {
	mock := &mockBambooClient{}
	compressor := NewSummaryCompressor(mock)

	// 8 条消息 > keepCount(4)，前 4 条应被总结
	messages := buildMessages(8)

	result, err := compressor.Compress(context.Background(), messages, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 期望结果：1 条摘要 + 4 条最近消息 = 5 条
	expectedLen := 1 + compressor.keepRecent*2
	if len(result) != expectedLen {
		t.Errorf("expected %d messages after compression, got %d", expectedLen, len(result))
	}

	// 第一条应该是摘要消息
	summaryMsg := result[0]
	if summaryMsg.Role != bamboo.RoleUser {
		t.Errorf("summary message should have user role, got %s", summaryMsg.Role)
	}
	if len(summaryMsg.Content) == 0 {
		t.Fatal("summary message should have content")
	}
	if !strings.Contains(summaryMsg.Content[0].Text, "Previous conversation summary:") {
		t.Errorf("summary message should contain prefix, got: %s", summaryMsg.Content[0].Text)
	}
	if !strings.Contains(summaryMsg.Content[0].Text, "mock summary") {
		t.Errorf("summary message should contain mock summary, got: %s", summaryMsg.Content[0].Text)
	}

	// 最近 4 条消息应该保持原样
	for i := 1; i < len(result); i++ {
		origIdx := len(messages) - compressor.keepRecent*2 + (i - 1)
		if result[i].Role != messages[origIdx].Role {
			t.Errorf("recent message[%d] role mismatch: expected %s, got %s", i, messages[origIdx].Role, result[i].Role)
		}
	}
}

// TestSummaryCompressor_SummaryContentFromOldMessages 测试摘要内容来自旧消息
func TestSummaryCompressor_SummaryContentFromOldMessages(t *testing.T) {
	var capturedPrompt string
	mock := &mockBambooClient{
		completeFunc: func(_ context.Context, messages []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (*bamboo.Response, error) {
			// 捕获发送给 AI 的消息内容
			if len(messages) > 0 && len(messages[0].Content) > 0 {
				capturedPrompt = messages[0].Content[0].Text
			}
			return &bamboo.Response{
				Content: []bamboo.ContentBlock{
					bamboo.NewTextBlock("AI-generated summary"),
				},
			}, nil
		},
	}
	compressor := NewSummaryCompressor(mock)

	// 6 条消息：前 2 条（1轮）应被总结，后 4 条（2轮）保留
	messages := buildMessages(6)

	result, err := compressor.Compress(context.Background(), messages, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证发送给 AI 的提示包含旧消息内容
	if !strings.Contains(capturedPrompt, compressor.summaryPrompt) {
		t.Error("captured prompt should contain summary prompt template")
	}
	if !strings.Contains(capturedPrompt, "[user]: user message") {
		t.Error("captured prompt should contain old user messages")
	}
	if !strings.Contains(capturedPrompt, "[assistant]: assistant reply") {
		t.Error("captured prompt should contain old assistant messages")
	}

	// 验证结果结构：[摘要] + 4 条最近
	if len(result) != 5 {
		t.Errorf("expected 5 messages (1 summary + 4 recent), got %d", len(result))
	}

	// 验证摘要包含 AI 生成的内容
	if !strings.Contains(result[0].Content[0].Text, "AI-generated summary") {
		t.Errorf("summary should contain 'AI-generated summary', got: %s", result[0].Content[0].Text)
	}
}

// TestSummaryCompressor_ClientError 测试 AI 客户端返回错误时的处理
func TestSummaryCompressor_ClientError(t *testing.T) {
	mock := &mockBambooClient{
		completeFunc: func(_ context.Context, _ []bamboo.BambooMessage, _ string, _ *bamboo.RequestConfig) (*bamboo.Response, error) {
			return nil, context.DeadlineExceeded
		},
	}
	compressor := NewSummaryCompressor(mock)

	messages := buildMessages(6)
	_, err := compressor.Compress(context.Background(), messages, 10000)
	if err == nil {
		t.Fatal("expected error when client fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to generate summary") {
		t.Errorf("error should wrap client error, got: %v", err)
	}
}

// TestEstimateTokens 测试 token 估算函数
func TestEstimateTokens(t *testing.T) {
	messages := []bamboo.BambooMessage{
		bamboo.NewUserMessage("12345678"), // 8 chars → 2 tokens
		bamboo.NewAssistantMessage("1234"), // 4 chars → 1 token
	}

	tokens := estimateTokens(messages)
	// 总共 12 字符 / 4 = 3 tokens
	if tokens != 3 {
		t.Errorf("expected 3 tokens, got %d", tokens)
	}
}

// TestSummaryCompressor_ExactBoundary 测试恰好等于阈值时不压缩
func TestSummaryCompressor_ExactBoundary(t *testing.T) {
	mock := &mockBambooClient{}
	compressor := NewSummaryCompressor(mock)

	// 恰好 keepRecent*2 = 4 条消息
	messages := buildMessages(4)

	result, err := compressor.Compress(context.Background(), messages, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 4 {
		t.Errorf("expected 4 messages unchanged at exact boundary, got %d", len(result))
	}
}

// TestSummaryCompressor_OneOverBoundary 测试刚超过阈值时压缩
func TestSummaryCompressor_OneOverBoundary(t *testing.T) {
	mock := &mockBambooClient{}
	compressor := NewSummaryCompressor(mock)

	// 5 条消息 > keepCount(4)，应触发压缩
	messages := buildMessages(5)

	result, err := compressor.Compress(context.Background(), messages, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 条摘要 + 4 条最近 = 5 条
	if len(result) != 5 {
		t.Errorf("expected 5 messages (1 summary + 4 recent), got %d", len(result))
	}
}
