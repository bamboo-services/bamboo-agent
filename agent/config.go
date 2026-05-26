package agent

// LoopStrategy is defined in loop.go
// ContextCompressor is defined in compressor.go

// AgentConfig holds all configuration parameters for the agent
type AgentConfig struct {
	Model              string
	MaxTokens          int64
	Temperature        *float64
	SystemPrompt       string
	MaxIterations      int
	LoopStrategy       LoopStrategy       // Interface, implemented later
	MaxContextTokens   int64
	Compressor         ContextCompressor  // Interface, implemented later
	MaxConcurrentTools int
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() AgentConfig {
	return AgentConfig{
		Model:              "claude-sonnet-4-20250514",
		MaxTokens:          4096,
		MaxIterations:      10,
		MaxConcurrentTools: 10,
		MaxContextTokens:   180000,
	}
}