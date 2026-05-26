package orchestrator

import (
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/agent"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// AgentBuilder 提供流畅的 API 用于构建 Agent 实例。
//
// 支持链式调用配置 BambooClient、AgentConfig、系统提示词、工具列表和循环策略，
// 最后通过 Build() 方法创建完整的 Agent 实例。
type AgentBuilder struct {
	// client 是 BM-SDK 客户端，用于 AI 交互。
	client bamboo.BambooClient

	// config 是 Agent 配置，nil 时使用默认配置。
	config *agent.AgentConfig

	// systemPrompt 是系统提示词，用于设定 Agent 角色。
	systemPrompt string

	// tools 是待注册的工具列表。
	tools []tool.Tool

	// strategy 是循环策略，nil 时使用默认 ReActLoop。
	strategy agent.LoopStrategy
}

// NewAgentBuilder 创建一个新的 AgentBuilder 实例。
//
// 返回初始化为空的构建器，需要通过 WithXXX 方法配置后调用 Build()。
func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{}
}

// WithClient 设置 BambooClient。
//
// 参数说明：
//   - client - BM-SDK 客户端
//
// 返回：当前构建器实例，支持链式调用
func (b *AgentBuilder) WithClient(client bamboo.BambooClient) *AgentBuilder {
	b.client = client
	return b
}

// WithConfig 设置 AgentConfig。
//
// 参数说明：
//   - config - Agent 配置（传入值会被复制）
//
// 返回：当前构建器实例，支持链式调用
func (b *AgentBuilder) WithConfig(config agent.AgentConfig) *AgentBuilder {
	b.config = &config
	return b
}

// WithSystemPrompt 设置系统提示词。
//
// 参数说明：
//   - prompt - 系统提示词文本
//
// 返回：当前构建器实例，支持链式调用
func (b *AgentBuilder) WithSystemPrompt(prompt string) *AgentBuilder {
	b.systemPrompt = prompt
	return b
}

// WithTools 添加待注册的工具。
//
// 参数说明：
//   - tools - 工具列表，可变参数
//
// 返回：当前构建器实例，支持链式调用
func (b *AgentBuilder) WithTools(tools ...tool.Tool) *AgentBuilder {
	b.tools = append(b.tools, tools...)
	return b
}

// WithLoopStrategy 设置循环策略。
//
// 参数说明：
//   - strategy - LoopStrategy 实现
//
// 返回：当前构建器实例，支持链式调用
func (b *AgentBuilder) WithLoopStrategy(strategy agent.LoopStrategy) *AgentBuilder {
	b.strategy = strategy
	return b
}

// Build 创建 Agent 实例。
//
// 必须先调用 WithClient() 设置客户端，否则会 panic。
// 会合并配置、注册工具、应用系统提示词和循环策略。
//
// 返回：配置完成的 Agent 实例
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