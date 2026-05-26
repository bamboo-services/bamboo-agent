package agent

// LoopStrategy 在 loop.go 中定义
// ContextCompressor 在 compressor.go 中定义

// AgentConfig 保存 Agent 的所有配置参数。
//
// 通过 `DefaultConfig` 获取默认值，或使用 Functional Options 模式覆盖。
type AgentConfig struct {
	// Model 是使用的模型名称。
	Model string

	// MaxTokens 是单次 AI 调用的最大 token 数。
	MaxTokens int64

	// Temperature 是 AI 生成的温度参数（0.0-1.0）。
	Temperature *float64

	// SystemPrompt 是系统提示词，用于设定 Agent 角色。
	SystemPrompt string

	// MaxIterations 是 ReAct 循环的最大迭代次数。
	MaxIterations int

	// LoopStrategy 是循环策略，nil 时使用默认 ReActLoop。
	LoopStrategy LoopStrategy

	// MaxContextTokens 是上下文的最大 token 数。
	MaxContextTokens int64

	// Compressor 是上下文压缩器，nil 时禁用压缩。
	Compressor ContextCompressor

	// MaxConcurrentTools 是并发执行的最大工具数量。
	MaxConcurrentTools int
}

// DefaultConfig 返回合理的默认配置。
//
// 返回：
//   - AgentConfig - 包含默认配置值的结构体
func DefaultConfig() AgentConfig {
	return AgentConfig{
		Model:              "claude-sonnet-4-20250514",
		MaxTokens:          4096,
		MaxIterations:      10,
		MaxConcurrentTools: 10,
		MaxContextTokens:   180000,
	}
}