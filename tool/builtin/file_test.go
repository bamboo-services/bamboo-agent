package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bamboo-services/bamboo-agent/tool"
)

// =============================================================================
// FileReadTool Tests
// =============================================================================

// TestFileReadTool_Info 测试 FileReadTool 的 Info 方法返回正确的元数据。
func TestFileReadTool_Info(t *testing.T) {
	tool := FileReadTool{}
	info := tool.Info()

	if info.Name != "file_read" {
		t.Errorf("expected name 'file_read', got '%s'", info.Name)
	}
	if info.Parameters.Type != "object" {
		t.Errorf("expected parameters type 'object', got '%s'", info.Parameters.Type)
	}
	if _, ok := info.Parameters.Properties["path"]; !ok {
		t.Error("expected 'path' property in parameters")
	}
	if len(info.Parameters.Required) != 1 || info.Parameters.Required[0] != "path" {
		t.Errorf("expected required ['path'], got %v", info.Parameters.Required)
	}
}

// TestFileReadTool_ReadsExistingFile 测试读取已存在的文件。
func TestFileReadTool_ReadsExistingFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	content := "hello, world!"

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := FileReadTool{}
	input, _ := json.Marshal(map[string]string{"path": filePath})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Content)
	}
	if result.Content != content {
		t.Errorf("expected content '%s', got '%s'", content, result.Content)
	}
}

// TestFileReadTool_NonExistentFile 测试读取不存在的文件返回错误。
func TestFileReadTool_NonExistentFile(t *testing.T) {
	tool := FileReadTool{}
	input, _ := json.Marshal(map[string]string{"path": "/nonexistent/path/file.txt"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for non-existent file")
	}
}

// TestFileReadTool_InvalidJSON 测试无效 JSON 输入返回错误。
func TestFileReadTool_InvalidJSON(t *testing.T) {
	tool := FileReadTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

// =============================================================================
// FileWriteTool Tests
// =============================================================================

// TestFileWriteTool_Info 测试 FileWriteTool 的 Info 方法返回正确的元数据。
func TestFileWriteTool_Info(t *testing.T) {
	tool := FileWriteTool{}
	info := tool.Info()

	if info.Name != "file_write" {
		t.Errorf("expected name 'file_write', got '%s'", info.Name)
	}
	if info.Parameters.Type != "object" {
		t.Errorf("expected parameters type 'object', got '%s'", info.Parameters.Type)
	}
	if _, ok := info.Parameters.Properties["path"]; !ok {
		t.Error("expected 'path' property in parameters")
	}
	if _, ok := info.Parameters.Properties["content"]; !ok {
		t.Error("expected 'content' property in parameters")
	}
	if len(info.Parameters.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(info.Parameters.Required))
	}
}

// TestFileWriteTool_WritesFile 测试写入文件并验证内容。
func TestFileWriteTool_WritesFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "output.txt")
	content := "written content"

	tool := FileWriteTool{}
	input, _ := json.Marshal(map[string]string{"path": filePath, "content": content})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Content)
	}
	if result.Content != "File written successfully" {
		t.Errorf("expected success message, got: '%s'", result.Content)
	}

	// 验证文件实际被写入
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected file content '%s', got '%s'", content, string(data))
	}
}

// TestFileWriteTool_CreatesDirectory 测试自动创建目录并写入文件。
func TestFileWriteTool_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sub", "dir", "output.txt")
	content := "nested write"

	tool := FileWriteTool{}
	input, _ := json.Marshal(map[string]string{"path": filePath, "content": content})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Content)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected file content '%s', got '%s'", content, string(data))
	}
}

// TestFileWriteTool_InvalidJSON 测试无效 JSON 输入返回错误。
func TestFileWriteTool_InvalidJSON(t *testing.T) {
	tool := FileWriteTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

// =============================================================================
// FileSearchTool Tests
// =============================================================================

// TestFileSearchTool_Info 测试 FileSearchTool 的 Info 方法返回正确的元数据。
func TestFileSearchTool_Info(t *testing.T) {
	tool := FileSearchTool{}
	info := tool.Info()

	if info.Name != "file_search" {
		t.Errorf("expected name 'file_search', got '%s'", info.Name)
	}
	if info.Parameters.Type != "object" {
		t.Errorf("expected parameters type 'object', got '%s'", info.Parameters.Type)
	}
	if _, ok := info.Parameters.Properties["path"]; !ok {
		t.Error("expected 'path' property in parameters")
	}
	if _, ok := info.Parameters.Properties["pattern"]; !ok {
		t.Error("expected 'pattern' property in parameters")
	}
	if len(info.Parameters.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(info.Parameters.Required))
	}
}

// TestFileSearchTool_FindsMatchingLines 测试搜索并返回匹配的行。
func TestFileSearchTool_FindsMatchingLines(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "search.txt")
	content := "line one\nline two\nanother line\nfinal line two"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := FileSearchTool{}
	input, _ := json.Marshal(map[string]string{"path": filePath, "pattern": "two"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Content)
	}

	expected := "2: line two\n4: final line two"
	if result.Content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result.Content)
	}
}

// TestFileSearchTool_NoMatches 测试无匹配时返回空结果。
func TestFileSearchTool_NoMatches(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "search.txt")
	content := "line one\nline two"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := FileSearchTool{}
	input, _ := json.Marshal(map[string]string{"path": filePath, "pattern": "xyz_not_found"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected no error for no-match case, got: %s", result.Content)
	}
}

// TestFileSearchTool_NonExistentFile 测试搜索不存在的文件返回错误。
func TestFileSearchTool_NonExistentFile(t *testing.T) {
	tool := FileSearchTool{}
	input, _ := json.Marshal(map[string]string{"path": "/nonexistent/file.txt", "pattern": "test"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for non-existent file")
	}
}

// TestFileSearchTool_InvalidJSON 测试无效 JSON 输入返回错误。
func TestFileSearchTool_InvalidJSON(t *testing.T) {
	tool := FileSearchTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

// =============================================================================
// Interface compliance checks
// =============================================================================

// TestToolInterfaceCompliance 测试所有文件工具都实现了 tool.Tool 接口。
func TestToolInterfaceCompliance(t *testing.T) {
	// 确保所有工具都实现了 tool.Tool 接口
	var _ tool.Tool = FileReadTool{}
	var _ tool.Tool = FileWriteTool{}
	var _ tool.Tool = FileSearchTool{}
}
