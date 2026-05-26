package builtin

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCodeExecTool_Info 测试 CodeExecTool 的 Info 方法返回正确的元数据。
func TestCodeExecTool_Info(t *testing.T) {
	tool := &CodeExecTool{}
	info := tool.Info()

	if info.Name != "code_exec" {
		t.Errorf("expected name 'code_exec', got %q", info.Name)
	}
	if info.Parameters.Type != "object" {
		t.Errorf("expected parameter type 'object', got %q", info.Parameters.Type)
	}

	// Verify language property has enum constraint
	langProp, ok := info.Parameters.Properties["language"]
	if !ok {
		t.Fatal("missing 'language' property")
	}
	if len(langProp.Enum) != 2 || langProp.Enum[0] != "go" || langProp.Enum[1] != "python" {
		t.Errorf("expected language enum [go, python], got %v", langProp.Enum)
	}

	// Verify required fields
	required := info.Parameters.Required
	hasLang, hasCode := false, false
	for _, r := range required {
		if r == "language" {
			hasLang = true
		}
		if r == "code" {
			hasCode = true
		}
	}
	if !hasLang || !hasCode {
		t.Errorf("expected required fields [language, code], got %v", required)
	}
}

// TestCodeExecTool_Python 测试执行 Python 代码。
func TestCodeExecTool_Python(t *testing.T) {
	// Skip if python3 is not available
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	tool := &CodeExecTool{}
	input, _ := json.Marshal(map[string]string{
		"language": "python",
		"code":     `print("hello from python")`,
	})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello from python") {
		t.Errorf("expected output to contain 'hello from python', got %q", result.Content)
	}
}

// TestCodeExecTool_Go 测试执行 Go 代码。
func TestCodeExecTool_Go(t *testing.T) {
	tool := &CodeExecTool{}
	input, _ := json.Marshal(map[string]string{
		"language": "go",
		"code": `package main

import "fmt"

func main() {
	fmt.Println("hello from go")
}`,
	})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello from go") {
		t.Errorf("expected output to contain 'hello from go', got %q", result.Content)
	}
}

// TestCodeExecTool_UnsupportedLanguage 测试不支持的语言返回错误。
func TestCodeExecTool_UnsupportedLanguage(t *testing.T) {
	tool := &CodeExecTool{}
	input, _ := json.Marshal(map[string]string{
		"language": "rust",
		"code":     `fn main() {}`,
	})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for unsupported language")
	}
	if !strings.Contains(result.Content, "unsupported language") {
		t.Errorf("expected 'unsupported language' message, got %q", result.Content)
	}
}

// TestCodeExecTool_InvalidCode 测试无效代码返回错误。
func TestCodeExecTool_InvalidCode(t *testing.T) {
	tool := &CodeExecTool{}
	input, _ := json.Marshal(map[string]string{
		"language": "python",
		"code":     `this is not valid python !!!`,
	})

	// Skip if python3 is not available
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for invalid code")
	}
}

// TestCodeExecTool_InvalidJSON 测试无效 JSON 返回错误。
func TestCodeExecTool_InvalidJSON(t *testing.T) {
	tool := &CodeExecTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid json`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
	if !strings.Contains(result.Content, "invalid input") {
		t.Errorf("expected 'invalid input' message, got %q", result.Content)
	}
}

// TestCodeExecTool_TempFilesCleanedUp 测试临时文件被正确清理。
func TestCodeExecTool_TempFilesCleanedUp(t *testing.T) {
	// Skip if python3 is not available
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	tool := &CodeExecTool{}

	// Execute with code that creates identifiable temp patterns
	input, _ := json.Marshal(map[string]string{
		"language": "python",
		"code":     `print("cleanup test")`,
	})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Content)
	}

	// Verify no bamboo-code-* temp directories remain
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "bamboo-code-") {
			t.Errorf("temp directory not cleaned up: %s/%s", tmpDir, entry.Name())
		}
	}
}
