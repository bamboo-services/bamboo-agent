package agent

import (
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// Option 是配置 Agent 的函数式选项类型。
type Option func(*agentOptions)

// agentOptions 保存所有可配置参数。
type agentOptions struct {
	config       AgentConfig
	systemPrompt string
	tools        []tool.Tool
}

// WithSystemPrompt 设置系统提示词。
func WithSystemPrompt(prompt string) Option {
	return func(o *agentOptions) {
		o.systemPrompt = prompt
	}
}

// WithConfig 设置完整的 Agent 配置。
func WithConfig(config AgentConfig) Option {
	return func(o *agentOptions) {
		o.config = config
	}
}

// WithMaxIterations 设置最大循环迭代次数。
func WithMaxIterations(n int) Option {
	return func(o *agentOptions) {
		o.config.MaxIterations = n
	}
}

// WithMaxTokens 设置每次响应的最大 token 数。
func WithMaxTokens(n int64) Option {
	return func(o *agentOptions) {
		o.config.MaxTokens = n
	}
}

// WithTemperature 设置生成温度参数。
func WithTemperature(f float64) Option {
	return func(o *agentOptions) {
		o.config.Temperature = &f
	}
}

// WithMaxConcurrentTools 设置最大并发工具执行数量。
func WithMaxConcurrentTools(n int) Option {
	return func(o *agentOptions) {
		o.config.MaxConcurrentTools = n
	}
}

// WithLoopStrategy 设置循环策略。
func WithLoopStrategy(strategy LoopStrategy) Option {
	return func(o *agentOptions) {
		o.config.LoopStrategy = strategy
	}
}

// WithCompressor 设置上下文压缩器。
func WithCompressor(compressor ContextCompressor) Option {
	return func(o *agentOptions) {
		o.config.Compressor = compressor
	}
}

// WithTools 注册给定的工具。
func WithTools(tools ...tool.Tool) Option {
	return func(o *agentOptions) {
		o.tools = append(o.tools, tools...)
	}
}

// NewAgentWithOptions 使用函数式选项创建 Agent。
//
// 参数说明：
//   - client - BM-SDK 客户端
//   - opts - 可选的配置选项
//
// 返回：
//   - Agent - 配置好的 Agent 实例
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