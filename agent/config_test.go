package agent

import (
	"testing"
)

// TestDefaultConfig 测试默认配置的值。
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test all default values
	if config.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected Model to be 'claude-sonnet-4-20250514', got '%s'", config.Model)
	}

	if config.MaxTokens != 4096 {
		t.Errorf("Expected MaxTokens to be 4096, got %d", config.MaxTokens)
	}

	if config.MaxIterations != 10 {
		t.Errorf("Expected MaxIterations to be 10, got %d", config.MaxIterations)
	}

	if config.MaxConcurrentTools != 10 {
		t.Errorf("Expected MaxConcurrentTools to be 10, got %d", config.MaxConcurrentTools)
	}

	if config.MaxContextTokens != 180000 {
		t.Errorf("Expected MaxContextTokens to be 180000, got %d", config.MaxContextTokens)
	}

	// Test Temperature is nil (not set)
	if config.Temperature != nil {
		t.Errorf("Expected Temperature to be nil, got %v", *config.Temperature)
	}

	// Test SystemPrompt is empty string
	if config.SystemPrompt != "" {
		t.Errorf("Expected SystemPrompt to be empty string, got '%s'", config.SystemPrompt)
	}

	// Test LoopStrategy is nil (interface type, not set in defaults)
	if config.LoopStrategy != nil {
		t.Errorf("Expected LoopStrategy to be nil, got %v", config.LoopStrategy)
	}

	// Test Compressor is nil (interface type, not set in defaults)
	if config.Compressor != nil {
		t.Errorf("Expected Compressor to be nil, got %v", config.Compressor)
	}
}