package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bamboo-services/bamboo-agent/tool"
)

// fileReadInput 定义 FileReadTool 的输入参数
type fileReadInput struct {
	Path string `json:"path"`
}

// FileReadTool 读取指定路径的文件内容
type FileReadTool struct{}

// Info 返回 FileReadTool 的元信息
func (t FileReadTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "file_read",
		Description: "Read the content of a file at the specified path.",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"path": {
					Type:        "string",
					Description: "The absolute path of the file to read.",
				},
			},
			Required: []string{"path"},
		},
	}
}

// Execute 执行文件读取操作
func (t FileReadTool) Execute(_ context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var in fileReadInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	data, err := os.ReadFile(in.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &tool.ToolResult{Content: fmt.Sprintf("file not found: %s", in.Path), IsError: true}, nil
		}
		if os.IsPermission(err) {
			return &tool.ToolResult{Content: fmt.Sprintf("permission denied: %s", in.Path), IsError: true}, nil
		}
		return &tool.ToolResult{Content: fmt.Sprintf("failed to read file: %s", err.Error()), IsError: true}, nil
	}

	return &tool.ToolResult{Content: string(data)}, nil
}

// fileWriteInput 定义 FileWriteTool 的输入参数
type fileWriteInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FileWriteTool 将内容写入指定路径的文件
type FileWriteTool struct{}

// Info 返回 FileWriteTool 的元信息
func (t FileWriteTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "file_write",
		Description: "Write content to a file at the specified path. Creates parent directories if they do not exist.",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"path": {
					Type:        "string",
					Description: "The absolute path of the file to write.",
				},
				"content": {
					Type:        "string",
					Description: "The content to write to the file.",
				},
			},
			Required: []string{"path", "content"},
		},
	}
}

// Execute 执行文件写入操作
func (t FileWriteTool) Execute(_ context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var in fileWriteInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// 确保父目录存在
	dir := filepath.Dir(in.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if os.IsPermission(err) {
			return &tool.ToolResult{Content: fmt.Sprintf("permission denied: %s", dir), IsError: true}, nil
		}
		return &tool.ToolResult{Content: fmt.Sprintf("failed to create directory: %s", err.Error()), IsError: true}, nil
	}

	if err := os.WriteFile(in.Path, []byte(in.Content), 0o644); err != nil {
		if os.IsPermission(err) {
			return &tool.ToolResult{Content: fmt.Sprintf("permission denied: %s", in.Path), IsError: true}, nil
		}
		return &tool.ToolResult{Content: fmt.Sprintf("failed to write file: %s", err.Error()), IsError: true}, nil
	}

	return &tool.ToolResult{Content: "File written successfully"}, nil
}

// fileSearchInput 定义 FileSearchTool 的输入参数
type fileSearchInput struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
}

// FileSearchTool 在指定文件中搜索匹配的内容行
type FileSearchTool struct{}

// Info 返回 FileSearchTool 的元信息
func (t FileSearchTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "file_search",
		Description: "Search for a pattern in a file and return matching lines with line numbers.",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"path": {
					Type:        "string",
					Description: "The absolute path of the file to search.",
				},
				"pattern": {
					Type:        "string",
					Description: "The search pattern to look for in the file.",
				},
			},
			Required: []string{"path", "pattern"},
		},
	}
}

// Execute 执行文件搜索操作
func (t FileSearchTool) Execute(_ context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var in fileSearchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	f, err := os.Open(in.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &tool.ToolResult{Content: fmt.Sprintf("file not found: %s", in.Path), IsError: true}, nil
		}
		if os.IsPermission(err) {
			return &tool.ToolResult{Content: fmt.Sprintf("permission denied: %s", in.Path), IsError: true}, nil
		}
		return &tool.ToolResult{Content: fmt.Sprintf("failed to open file: %s", err.Error()), IsError: true}, nil
	}
	defer f.Close()

	var matches []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if strings.Contains(scanner.Text(), in.Pattern) {
			matches = append(matches, fmt.Sprintf("%d: %s", lineNum, scanner.Text()))
		}
	}

	if err := scanner.Err(); err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("failed to read file: %s", err.Error()), IsError: true}, nil
	}

	if len(matches) == 0 {
		return &tool.ToolResult{Content: fmt.Sprintf("no matches found for pattern: %s", in.Pattern)}, nil
	}

	return &tool.ToolResult{Content: strings.Join(matches, "\n")}, nil
}


