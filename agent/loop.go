package agent

import (
	"context"
	"encoding/json"
	"fmt"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// LoopStrategy 定义 Agent 执行周期的迭代策略。
//
// 定义了执行策略的核心能力，包括：
//   - Execute - 执行 Agent 任务并返回最终结果
type LoopStrategy interface {
	Execute(ctx context.Context, core *agentCore, input string) (*AgentResult, error)
}

// agentCore 是 LoopStrategy 需要的最小内部状态。
//
// 用于在 LoopStrategy 执行过程中传递必要的上下文信息，包含：
//   - client - BM-SDK 客户端，用于 AI 交互
//   - config - Agent 配置参数
//   - session - 会话消息历史
//   - registry - 工具注册表
//   - executor - 工具执行器
type agentCore struct {
	client   bamboo.BambooClient
	config   AgentConfig
	session  *Session
	registry *tool.Registry
	executor *tool.ToolExecutor
}

// ReActLoop 实现 Reason-Act 迭代策略。
//
// 执行流程如下：
//   - 向 AI 发送消息（流式）
//   - 如果 AI 返回 tool_use → 执行工具 → 追加结果 → 继续循环
//   - 如果 AI 返回 end_turn → 返回最终结果
//   - 如果达到最大迭代次数 → 强制停止
type ReActLoop struct{}

// NewReActLoop 创建一个新的 ReActLoop 实例。
//
// 返回：
//   - *ReActLoop - ReActLoop 实例
func NewReActLoop() *ReActLoop {
	return &ReActLoop{}
}

// Execute 执行 Agent 任务并返回最终结果。
//
// 使用 ReAct 迭代策略执行任务，支持工具调用和流式输出。
// 在每次迭代中检查上下文长度，必要时进行压缩。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - core - agentCore 实例，包含执行所需的客户端、配置和状态
//   - input - 用户输入文本
//
// 返回：
//   - *AgentResult - 执行结果，包含生成内容、工具调用记录和使用量
//   - error - 执行错误，如上下文取消、AI 调用失败或工具执行失败
func (l *ReActLoop) Execute(ctx context.Context, core *agentCore, input string) (*AgentResult, error) {
	core.session.AppendMessage(bamboo.NewUserMessage(input))

	var allToolCalls []ToolCallRecord
	var lastContent string
	var usage bamboo.Usage
	totalIterations := 0

	for i := 0; i < core.config.MaxIterations; i++ {
		totalIterations = i + 1

		if core.config.Compressor != nil && core.config.MaxContextTokens > 0 {
			messages := core.session.GetMessages()
			estimated := estimateTokens(messages)
			if estimated > core.config.MaxContextTokens {
				compressed, err := core.config.Compressor.Compress(ctx, messages, core.config.MaxContextTokens)
				if err == nil {
					core.session.Clear()
					for _, msg := range compressed {
						core.session.AppendMessage(msg)
					}
				}
			}
		}

		messages := core.session.GetMessages()
		bambooTools := tool.NewBambooAdapter().ToBambooTools(core.registry.List())

		config := &bamboo.RequestConfig{
			Model:     core.config.Model,
			MaxTokens: core.config.MaxTokens,
		}
		if core.config.Temperature != nil {
			config.Temperature = core.config.Temperature
		}
		if len(bambooTools) > 0 {
			config.Tools = bambooTools
		}

		stream, err := core.client.Chat(ctx, messages, core.config.SystemPrompt, config)
		if err != nil {
			return nil, fmt.Errorf("AI chat failed: %w", err)
		}

		var textContent string
		type toolCallAcc struct {
			ID    string
			Name  string
			Input string
		}
		var toolCalls []struct {
			ID    string
			Name  string
			Input json.RawMessage
		}
		var currentToolCall *toolCallAcc
		var stopReason bamboo.FinishReason

		for event := range stream {
			switch event.Type {
			case bamboo.EventContentBlockStart:
				if event.ContentBlock != nil && event.ContentBlock.Type == bamboo.ContentBlockToolUse {
					currentToolCall = &toolCallAcc{
						ID:   event.ContentBlock.ID,
						Name: event.ContentBlock.Name,
					}
				}

			case bamboo.EventContentBlockDelta:
				if event.Delta != nil {
					switch d := event.Delta.(type) {
					case *bamboo.StreamDelta:
						switch d.Type {
						case bamboo.DeltaTextDelta:
							textContent += d.Text
						case bamboo.DeltaInputJSON:
							if currentToolCall != nil {
								currentToolCall.Input += d.PartialJSON
							}
						}
					}
				}

			case bamboo.EventContentBlockStop:
				if currentToolCall != nil {
					toolCalls = append(toolCalls, struct {
						ID    string
						Name  string
						Input json.RawMessage
					}{
						ID:    currentToolCall.ID,
						Name:  currentToolCall.Name,
						Input: json.RawMessage(currentToolCall.Input),
					})
					currentToolCall = nil
				}

			case bamboo.EventMessageDelta:
				if event.Delta != nil {
					if md, ok := event.Delta.(*bamboo.MessageDelta); ok {
						stopReason = md.StopReason
					}
				}

			case bamboo.EventError:
				if event.Error != nil {
					return nil, fmt.Errorf("stream error: %s", event.Error.Message)
				}
			}

			if event.Usage != nil {
				usage = *event.Usage
			}
		}

		_ = stopReason

		assistantContent := make([]bamboo.ContentBlock, 0, 1+len(toolCalls))
		if textContent != "" {
			assistantContent = append(assistantContent, bamboo.NewTextBlock(textContent))
			lastContent = textContent
		}
		for _, tc := range toolCalls {
			assistantContent = append(assistantContent, bamboo.ContentBlock{
				Type:  bamboo.ContentBlockToolUse,
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Input,
			})
		}
		core.session.AppendMessage(bamboo.BambooMessage{
			Role:    bamboo.RoleAssistant,
			Content: assistantContent,
		})

		if len(toolCalls) == 0 {
			return &AgentResult{
				Content:    lastContent,
				Messages:   core.session.GetMessages(),
				ToolCalls:  allToolCalls,
				Usage:      usage,
				Iterations: totalIterations,
			}, nil
		}

		inputs := make([]tool.ToolCallInput, len(toolCalls))
		for i, tc := range toolCalls {
			inputs[i] = tool.ToolCallInput{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Input,
			}
		}

		outputs, err := core.executor.ExecuteAll(ctx, inputs)
		if err != nil {
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}

		toolResultContent := make([]bamboo.ContentBlock, 0, len(outputs))
		for idx, out := range outputs {
			var content string
			var isError bool
			if out.Error != nil {
				content = out.Error.Error()
				isError = true
			} else if out.Result != nil {
				content = out.Result.Content
				isError = out.Result.IsError
			}

			toolResultContent = append(toolResultContent, bamboo.NewToolResultBlock(out.ID, content, isError))

			allToolCalls = append(allToolCalls, ToolCallRecord{
				ID:      out.ID,
				Name:    toolCalls[idx].Name,
				Input:   toolCalls[idx].Input,
				Result:  content,
				IsError: isError,
			})
		}

		core.session.AppendMessage(bamboo.BambooMessage{
			Role:    bamboo.RoleUser,
			Content: toolResultContent,
		})
	}

	return &AgentResult{
		Content:    lastContent,
		Messages:   core.session.GetMessages(),
		ToolCalls:  allToolCalls,
		Usage:      usage,
		Iterations: totalIterations,
	}, nil
}
