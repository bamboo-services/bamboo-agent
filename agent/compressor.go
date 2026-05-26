package agent

import (
	"context"
	"fmt"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// ContextCompressor 定义上下文压缩器的接口。
//
// 定义了上下文压缩的核心能力，包括：
//   - Compress - 压缩消息历史，保留关键信息
type ContextCompressor interface {
	Compress(ctx context.Context, messages []bamboo.BambooMessage, maxTokens int64) ([]bamboo.BambooMessage, error)
}

// SummaryCompressor 基于 AI 总结的上下文压缩器实现。
//
// 通过调用 AI 模型生成对话摘要，实现消息历史的智能压缩。
type SummaryCompressor struct {
	client        bamboo.BambooClient
	keepRecent    int    // 保留最近几轮对话（默认 2，即 4 条消息）
	summaryPrompt string // 摘要生成提示词模板
}

// NewSummaryCompressor 创建基于 AI 总结的上下文压缩器。
//
// 使用默认配置初始化压缩器：
//   - keepRecent - 默认保留最近 2 轮对话（即 4 条消息）
//   - summaryPrompt - 使用预设的英文提示词生成摘要
//
// 参数说明：
//   - client - Bamboo 客户端，用于调用 AI 模型
//
// 返回：
//   - *SummaryCompressor - 新创建的压缩器实例
func NewSummaryCompressor(client bamboo.BambooClient) *SummaryCompressor {
	return &SummaryCompressor{
		client:        client,
		keepRecent:    2, // 2 轮 = 4 条消息（2 user + 2 assistant）
		summaryPrompt: "Please summarize the following conversation history concisely, preserving key facts, decisions, and context. Output only the summary:",
	}
}

// Compress 压缩消息历史，保留最近 N 轮对话并总结更早的消息。
//
// 压缩策略：
//   - 保留最近 N*2 条消息（N 轮对话）
//   - 将更早的消息通过 AI 总结为一条摘要
//   - 当消息总数 <= keepRecent*2 时，不做压缩直接返回
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - messages - 原始消息列表
//   - maxTokens - 最大 token 数限制（当前未使用）
//
// 返回：
//   - []bamboo.BambooMessage - 压缩后的消息列表（[摘要] + 最近消息）
//   - error - 执行错误，如 AI 调用失败
func (c *SummaryCompressor) Compress(ctx context.Context, messages []bamboo.BambooMessage, maxTokens int64) ([]bamboo.BambooMessage, error) {
	keepCount := c.keepRecent * 2 // 每轮 = 2 条消息

	if len(messages) <= keepCount {
		return messages, nil // 消息不足，无需压缩
	}

	// 分割：旧消息用于总结 + 最近消息保留
	oldMessages := messages[:len(messages)-keepCount]
	recentMessages := messages[len(messages)-keepCount:]

	// 从旧消息构建对话文本
	var conversationText string
	for _, msg := range oldMessages {
		conversationText += fmt.Sprintf("[%s]: ", msg.Role)
		for _, block := range msg.Content {
			if block.Type == bamboo.ContentBlockText {
				conversationText += block.Text + "\n"
			}
		}
	}

	// 使用 BambooClient 生成摘要
	summaryMessages := []bamboo.BambooMessage{
		{
			Role: bamboo.RoleUser,
			Content: []bamboo.ContentBlock{
				bamboo.NewTextBlock(c.summaryPrompt + "\n\n" + conversationText),
			},
		},
	}

	resp, err := c.client.Complete(ctx, summaryMessages, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// 从响应中提取文本内容
	var summaryText string
	for _, block := range resp.Content {
		if block.Type == bamboo.ContentBlockText {
			summaryText += block.Text
		}
	}

	// 构建摘要消息
	summaryMsg := bamboo.BambooMessage{
		Role: bamboo.RoleUser,
		Content: []bamboo.ContentBlock{
			bamboo.NewTextBlock("Previous conversation summary:\n" + summaryText),
		},
	}

	// 返回：[摘要] + 最近消息
	result := make([]bamboo.BambooMessage, 0, 1+len(recentMessages))
	result = append(result, summaryMsg)
	result = append(result, recentMessages...)

	return result, nil
}

// estimateTokens 估算消息列表的总 token 数。
//
// 使用字符数除以 4 作为近似估算值。
//
// 参数说明：
//   - messages - 消息列表
//
// 返回：
//   - int64 - 估算的 token 总数
func estimateTokens(messages []bamboo.BambooMessage) int64 {
	var totalChars int64
	for _, msg := range messages {
		for _, block := range msg.Content {
			if block.Type == bamboo.ContentBlockText {
				totalChars += int64(len(block.Text))
			}
		}
	}
	return totalChars / 4
}
