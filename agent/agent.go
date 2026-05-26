package agent

import (
	"context"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// Agent is the core interface for an AI agent.
//
// It provides methods for running tasks (with or without streaming),
// managing tools, and configuring the system prompt.
type Agent interface {
	// Run executes an agent task and returns the final result.
	Run(ctx context.Context, input string) (*AgentResult, error)

	// Stream executes an agent task and returns events via channel.
	Stream(ctx context.Context, input string) (<-chan AgentEvent, error)

	// RunWithMessages executes using existing message history.
	RunWithMessages(ctx context.Context, messages []bamboo.BambooMessage) (*AgentResult, error)

	// AddTool registers a tool to the agent.
	AddTool(t tool.Tool) error

	// SetSystemPrompt updates the system prompt.
	SetSystemPrompt(prompt string)
}

// NewAgent creates a new agent with the given client and config.
//
// If config.LoopStrategy is nil, a default ReActLoop is used.
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

// Run executes an agent task and returns the final result.
// It delegates to the configured LoopStrategy (or the default ReActLoop).
func (a *agentCore) Run(ctx context.Context, input string) (*AgentResult, error) {
	strategy := a.config.LoopStrategy
	if strategy == nil {
		strategy = NewReActLoop()
	}
	return strategy.Execute(ctx, a, input)
}

// Stream executes an agent task and emits events through a channel.
// The channel is closed when the task finishes (or errors).
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

// RunWithMessages executes using an existing message history.
//
// It loads the provided messages into a fresh session, finds the last user
// message text, and runs the loop with that text as input.
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

// AddTool registers a tool in the agent's tool registry.
func (a *agentCore) AddTool(t tool.Tool) error {
	return a.registry.Register(t)
}

// SetSystemPrompt updates the system prompt in the agent's config.
func (a *agentCore) SetSystemPrompt(prompt string) {
	a.config.SystemPrompt = prompt
}
