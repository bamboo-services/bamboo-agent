package agent

import (
	"context"
	"encoding/json"
	"fmt"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// LoopStrategy defines the iteration strategy for an agent's execution cycle.
type LoopStrategy interface {
	Execute(ctx context.Context, core *agentCore, input string) (*AgentResult, error)
}

// agentCore is the minimal internal state needed by LoopStrategy.
type agentCore struct {
	client   bamboo.BambooClient
	config   AgentConfig
	session  *Session
	registry *tool.Registry
	executor *tool.ToolExecutor
}

// ReActLoop implements the Reason-Act iteration strategy:
//  1. Send messages to AI (streaming)
//  2. If AI returns tool_use → execute tools → append results → loop
//  3. If AI returns end_turn → return final result
//  4. If max iterations reached → force stop
type ReActLoop struct{}

func NewReActLoop() *ReActLoop {
	return &ReActLoop{}
}

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
