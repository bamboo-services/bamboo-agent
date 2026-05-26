package orchestrator

import (
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/agent"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// AgentBuilder provides a fluent API for constructing agents.
type AgentBuilder struct {
	client       bamboo.BambooClient
	config       *agent.AgentConfig
	systemPrompt string
	tools        []tool.Tool
	strategy     agent.LoopStrategy
}

// NewAgentBuilder creates a new AgentBuilder.
func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{}
}

// WithClient sets the BambooClient.
func (b *AgentBuilder) WithClient(client bamboo.BambooClient) *AgentBuilder {
	b.client = client
	return b
}

// WithConfig sets the AgentConfig.
func (b *AgentBuilder) WithConfig(config agent.AgentConfig) *AgentBuilder {
	b.config = &config
	return b
}

// WithSystemPrompt sets the system prompt.
func (b *AgentBuilder) WithSystemPrompt(prompt string) *AgentBuilder {
	b.systemPrompt = prompt
	return b
}

// WithTools adds tools to be registered.
func (b *AgentBuilder) WithTools(tools ...tool.Tool) *AgentBuilder {
	b.tools = append(b.tools, tools...)
	return b
}

// WithLoopStrategy sets the loop strategy.
func (b *AgentBuilder) WithLoopStrategy(strategy agent.LoopStrategy) *AgentBuilder {
	b.strategy = strategy
	return b
}

// Build creates the Agent instance.
func (b *AgentBuilder) Build() agent.Agent {
	if b.client == nil {
		panic("AgentBuilder: BambooClient is required, call WithClient() first")
	}

	config := agent.DefaultConfig()
	if b.config != nil {
		config = *b.config
	}
	if b.systemPrompt != "" {
		config.SystemPrompt = b.systemPrompt
	}
	if b.strategy != nil {
		config.LoopStrategy = b.strategy
	}

	ag := agent.NewAgent(b.client, config)

	for _, t := range b.tools {
		_ = ag.AddTool(t)
	}

	return ag
}