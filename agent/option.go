package agent

import (
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// Option is a functional option for configuring an Agent.
type Option func(*agentOptions)

// agentOptions holds all configurable parameters.
type agentOptions struct {
	config       AgentConfig
	systemPrompt string
	tools        []tool.Tool
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(o *agentOptions) {
		o.systemPrompt = prompt
	}
}

// WithConfig sets the full agent config.
func WithConfig(config AgentConfig) Option {
	return func(o *agentOptions) {
		o.config = config
	}
}

// WithMaxIterations sets the maximum loop iterations.
func WithMaxIterations(n int) Option {
	return func(o *agentOptions) {
		o.config.MaxIterations = n
	}
}

// WithMaxTokens sets the maximum tokens per response.
func WithMaxTokens(n int64) Option {
	return func(o *agentOptions) {
		o.config.MaxTokens = n
	}
}

// WithTemperature sets the temperature.
func WithTemperature(f float64) Option {
	return func(o *agentOptions) {
		o.config.Temperature = &f
	}
}

// WithMaxConcurrentTools sets max concurrent tool executions.
func WithMaxConcurrentTools(n int) Option {
	return func(o *agentOptions) {
		o.config.MaxConcurrentTools = n
	}
}

// WithLoopStrategy sets the loop strategy.
func WithLoopStrategy(strategy LoopStrategy) Option {
	return func(o *agentOptions) {
		o.config.LoopStrategy = strategy
	}
}

// WithCompressor sets the context compressor.
func WithCompressor(compressor ContextCompressor) Option {
	return func(o *agentOptions) {
		o.config.Compressor = compressor
	}
}

// WithTools registers the given tools.
func WithTools(tools ...tool.Tool) Option {
	return func(o *agentOptions) {
		o.tools = append(o.tools, tools...)
	}
}

// NewAgentWithOptions creates an Agent using functional options.
func NewAgentWithOptions(client bamboo.BambooClient, opts ...Option) Agent {
	o := &agentOptions{
		config: DefaultConfig(),
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.systemPrompt != "" {
		o.config.SystemPrompt = o.systemPrompt
	}

	agent := NewAgent(client, o.config)

	for _, t := range o.tools {
		_ = agent.AddTool(t)
	}

	return agent
}