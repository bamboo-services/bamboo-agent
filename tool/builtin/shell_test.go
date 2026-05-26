package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestShellTool_Info 测试 ShellTool 的 Info 方法返回正确的元数据。
func TestShellTool_Info(t *testing.T) {
	shell := &ShellTool{}
	info := shell.Info()

	if info.Name != "shell" {
		t.Errorf("Expected name 'shell', got '%s'", info.Name)
	}

	if info.Description != "执行 Shell 命令并返回 stdout 和 stderr" {
		t.Errorf("Unexpected description: %s", info.Description)
	}

	if info.Parameters.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", info.Parameters.Type)
	}

	// Check required fields
	if len(info.Parameters.Required) != 1 || info.Parameters.Required[0] != "command" {
		t.Errorf("Expected required field 'command', got %v", info.Parameters.Required)
	}

	// Check properties
	commandProp, ok := info.Parameters.Properties["command"]
	if !ok {
		t.Fatal("Missing 'command' property")
	}
	if commandProp.Type != "string" {
		t.Errorf("Expected 'command' type 'string', got '%s'", commandProp.Type)
	}

	timeoutProp, ok := info.Parameters.Properties["timeout"]
	if !ok {
		t.Fatal("Missing 'timeout' property")
	}
	if timeoutProp.Type != "number" {
		t.Errorf("Expected 'timeout' type 'number', got '%s'", timeoutProp.Type)
	}
}

// TestShellTool_Execute_SimpleCommand 测试执行简单的 echo 命令。
func TestShellTool_Execute_SimpleCommand(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	input, _ := json.Marshal(map[string]interface{}{
		"command": "echo hello",
	})

	result, err := shell.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.Content)
	}

	if !strings.Contains(result.Content, "hello") {
		t.Errorf("Expected output to contain 'hello', got: %s", result.Content)
	}
}

// TestShellTool_Execute_WithArguments 测试执行带参数的命令。
func TestShellTool_Execute_WithArguments(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	input, _ := json.Marshal(map[string]interface{}{
		"command": "ls -la",
	})

	result, err := shell.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.Content)
	}

	// ls should output something
	if result.Content == "" {
		t.Error("Expected non-empty output from ls")
	}
}

// TestShellTool_Execute_FailingCommand 测试失败的命令返回 IsError=true。
func TestShellTool_Execute_FailingCommand(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	input, _ := json.Marshal(map[string]interface{}{
		"command": "exit 1",
	})

	result, err := shell.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("Expected IsError=true for failing command")
	}

	if !strings.Contains(result.Content, "command failed") {
		t.Errorf("Expected error message to contain 'command failed', got: %s", result.Content)
	}
}

// TestShellTool_Execute_WithTimeout 测试命令可以在超时时间内完成。
func TestShellTool_Execute_WithTimeout(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	// Use a short timeout but execute a fast command
	input, _ := json.Marshal(map[string]interface{}{
		"command": "echo quick",
		"timeout": 5.0,
	})

	result, err := shell.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success within timeout, got error: %s", result.Content)
	}

	if !strings.Contains(result.Content, "quick") {
		t.Errorf("Expected output to contain 'quick', got: %s", result.Content)
	}
}

// TestShellTool_Execute_TimeoutExceeded 测试超时的命令返回错误。
func TestShellTool_Execute_TimeoutExceeded(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	// Sleep longer than timeout
	input, _ := json.Marshal(map[string]interface{}{
		"command": "sleep 2",
		"timeout": 0.5, // 0.5 seconds
	})

	start := time.Now()
	result, err := shell.Execute(ctx, input)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("Expected IsError=true for timeout")
	}

	if !strings.Contains(result.Content, "timed out") {
		t.Errorf("Expected error message to contain 'timed out', got: %s", result.Content)
	}

	// Should timeout within reasonable time (allow some margin)
	if duration > 2*time.Second {
		t.Errorf("Timeout took too long: %v (expected ~0.5s)", duration)
	}
}

// TestShellTool_Execute_InvalidJSON 测试无效 JSON 返回 IsError=true。
func TestShellTool_Execute_InvalidJSON(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	invalidJSON := json.RawMessage("{invalid json}")

	result, err := shell.Execute(ctx, invalidJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("Expected IsError=true for invalid JSON")
	}

	if !strings.Contains(result.Content, "invalid input") {
		t.Errorf("Expected error message to contain 'invalid input', got: %s", result.Content)
	}
}

// TestShellTool_Execute_StderrCaptured 测试捕获 stderr 输出。
func TestShellTool_Execute_StderrCaptured(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	// Command that writes to stderr
	input, _ := json.Marshal(map[string]interface{}{
		"command": "echo 'error message' >&2",
	})

	result, err := shell.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "STDERR") {
		t.Errorf("Expected output to contain 'STDERR', got: %s", result.Content)
	}

	if !strings.Contains(result.Content, "error message") {
		t.Errorf("Expected output to contain 'error message', got: %s", result.Content)
	}
}

// TestShellTool_Execute_DefaultTimeout 测试默认超时为 30 秒。
func TestShellTool_Execute_DefaultTimeout(t *testing.T) {
	shell := &ShellTool{}
	ctx := context.Background()

	// Command without timeout parameter
	input, _ := json.Marshal(map[string]interface{}{
		"command": "echo test",
	})

	start := time.Now()
	result, err := shell.Execute(ctx, input)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.Content)
	}

	// Should complete quickly (not 30 seconds)
	if duration > 5*time.Second {
		t.Errorf("Default timeout seems to be blocking: %v", duration)
	}
}