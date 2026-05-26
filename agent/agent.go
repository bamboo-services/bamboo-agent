package agent

import (
	"context"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// Agent 是 AI Agent 的核心接口。
//
// 定义了 Agent 的完整生命周期，包括：
//   - 任务执行（Run / Stream）
//   - 历史消息复用（RunWithMessages）
//   - 工具注册（AddTool）
//   - 系统提示词配置（SetSystemPrompt）
type Agent interface {
	// Run 执行一个 Agent 任务并返回最终结果。
	Run(ctx context.Context, input string) (*AgentResult, error)

	// Stream 执行一个 Agent 任务并通过 channel 返回事件。
	Stream(ctx context.Context, input string) (<-chan AgentEvent, error)

	// RunWithMessages 使用现有的消息历史执行。
	RunWithMessages(ctx context.Context, messages []bamboo.BambooMessage) (*AgentResult, error)

	// AddTool 向 Agent 注册一个工具。
	AddTool(t tool.Tool) error

	// SetSystemPrompt 更新系统提示词。
	SetSystemPrompt(prompt string)
}

// NewAgent 使用给定的客户端和配置创建一个新的 Agent。
//
// 如果 config.LoopStrategy 为 nil，则使用默认的 ReActLoop。
func NewAgent(client bamboo.BambooClient, config AgentConfig) Agent {
	registry := tool.NewRegistry()
	session := NewSession(registry)
	executor := tool.NewToolExecutor(registry, config.MaxConcurrentTools)

	return &agentCore{
		client:   client,
		config:   config,
		session:  session,
		registry: registry,
		executor: executor,
	}
}

// Run 执行一个 Agent 任务并返回最终结果。
//
// 委托给配置的 LoopStrategy（或默认的 ReActLoop）执行。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - input - 用户输入文本
//
// 返回：
//   - *AgentResult - 执行结果，包含生成内容和工具调用记录
//   - error - 执行错误，如上下文取消或 AI 调用失败
func (a *agentCore) Run(ctx context.Context, input string) (*AgentResult, error) {
	strategy := a.config.LoopStrategy
	if strategy == nil {
		strategy = NewReActLoop()
	}
	return strategy.Execute(ctx, a, input)
}

// Stream 执行一个 Agent 任务并通过 channel 发送事件。
//
// 当任务完成（或出错）时，channel 会被关闭。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - input - 用户输入文本
//
// 返回：
//   - <-chan AgentEvent - 事件 channel，包含流式输出事件
//   - error - 创建 channel 或执行错误
func (a *agentCore) Stream(ctx context.Context, input string) (<-chan AgentEvent, error) {
	ch := make(chan AgentEvent, 64)

	go func() {
		defer close(ch)

		result, err := a.Run(ctx, input)
		if err != nil {
			ch <- AgentEvent{
				Type:  AgentEventError,
				Error: err,
			}
			return
		}

		if result.Content != "" {
			ch <- AgentEvent{
				Type:    AgentEventText,
				Content: result.Content,
			}
		}

		for i := range result.ToolCalls {
			ch <- AgentEvent{
				Type:     AgentEventToolCall,
				ToolCall: &result.ToolCalls[i],
			}
		}

		ch <- AgentEvent{
			Type:   AgentEventComplete,
			Result: result,
		}
	}()

	return ch, nil
}

// RunWithMessages 使用现有的消息历史执行。
//
// 将提供的消息加载到新的会话中，找到最后一条用户消息文本，
// 并使用该文本作为输入运行循环。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - messages - 消息历史列表
//
// 返回：
//   - *AgentResult - 执行结果，包含生成内容和工具调用记录
//   - error - 执行错误，如上下文取消或 AI 调用失败
func (a *agentCore) RunWithMessages(ctx context.Context, messages []bamboo.BambooMessage) (*AgentResult, error) {
	a.session.Clear()
	for _, msg := range messages {
		a.session.AppendMessage(msg)
	}

	// Find the last user message text to use as input for the loop.
	var lastUserInput string
	msgs := a.session.GetMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == bamboo.RoleUser {
			for _, block := range msgs[i].Content {
				if block.Type == bamboo.ContentBlockText {
					lastUserInput = block.Text
					break
				}
			}
			break
		}
	}

	if lastUserInput == "" {
		return &AgentResult{
			Messages:   a.session.GetMessages(),
			Iterations: 0,
		}, nil
	}

	// Clear session so the loop starts fresh and appends the user message itself.
	a.session.Clear()
	strategy := a.config.LoopStrategy
	if strategy == nil {
		strategy = NewReActLoop()
	}
	return strategy.Execute(ctx, a, lastUserInput)
}

// AddTool 向 Agent 的工具注册表中注册一个工具。
//
// 参数说明：
//   - t - 要注册的工具
//
// 返回：
//   - error - 注册错误，如工具名称冲突或参数验证失败
func (a *agentCore) AddTool(t tool.Tool) error {
	return a.registry.Register(t)
}

// SetSystemPrompt 更新 Agent 配置中的系统提示词。
//
// 参数说明：
//   - prompt - 新的系统提示词
func (a *agentCore) SetSystemPrompt(prompt string) {
	a.config.SystemPrompt = prompt
}
