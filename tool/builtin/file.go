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

// fileReadInput 是 FileReadTool 的输入参数。
//
// 用于传入文件路径，指定要读取的文件位置。
type fileReadInput struct {
	Path string `json:"path"` // 文件的绝对路径
}

// FileReadTool 提供文件读取能力。
//
// 用于读取指定路径的文件内容，支持错误处理。
type FileReadTool struct{}

// Info 返回 FileReadTool 的元信息。
//
// 返回工具名称、描述和参数定义，用于工具注册和 AI 调用。
//
// 返回：
//   - tool.ToolInfo - 工具元信息，包含名称、描述和参数定义
func (t FileReadTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "file_read",
		Description: "读取指定路径的文件内容。",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"path": {
					Type:        "string",
					Description: "要读取的文件的绝对路径。",
				},
			},
			Required: []string{"path"},
		},
	}
}

// Execute 执行文件读取操作。
//
// 解析输入参数，读取文件内容并返回。如果文件不存在或无权限访问，返回相应的错误信息。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制（当前未使用）
//   - input - JSON 格式的输入参数，包含 `path` 字段
//
// 返回：
//   - *tool.ToolResult - 执行结果，包含文件内容或错误信息
//   - error - 解析错误或执行错误
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

// fileWriteInput 是 FileWriteTool 的输入参数。
//
// 用于传入文件路径和内容，指定要写入的文件位置和写入内容。
type fileWriteInput struct {
	Path    string `json:"path"`    // 文件的绝对路径
	Content string `json:"content"` // 要写入文件的内容
}

// FileWriteTool 提供文件写入能力。
//
// 用于将内容写入指定路径的文件，自动创建不存在的父目录，支持错误处理。
type FileWriteTool struct{}

// Info 返回 FileWriteTool 的元信息。
//
// 返回工具名称、描述和参数定义，用于工具注册和 AI 调用。
//
// 返回：
//   - tool.ToolInfo - 工具元信息，包含名称、描述和参数定义
func (t FileWriteTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "file_write",
		Description: "将内容写入指定路径的文件，若父目录不存在则自动创建。",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"path": {
					Type:        "string",
					Description: "要写入的文件的绝对路径。",
				},
				"content": {
					Type:        "string",
					Description: "要写入文件的内容。",
				},
			},
			Required: []string{"path", "content"},
		},
	}
}

// Execute 执行文件写入操作。
//
// 解析输入参数，自动创建父目录，将内容写入文件并返回结果。如果权限不足或写入失败，返回相应的错误信息。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制（当前未使用）
//   - input - JSON 格式的输入参数，包含 `path` 和 `content` 字段
//
// 返回：
//   - *tool.ToolResult - 执行结果，包含成功信息或错误信息
//   - error - 解析错误或执行错误
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

// fileSearchInput 是 FileSearchTool 的输入参数。
//
// 用于传入文件路径和搜索模式，指定要搜索的文件和要匹配的文本模式。
type fileSearchInput struct {
	Path    string `json:"path"`    // 要搜索的文件的绝对路径
	Pattern string `json:"pattern"` // 搜索模式，在文件中查找匹配的文本
}

// FileSearchTool 提供文件内容搜索能力。
//
// 用于在指定文件中搜索匹配的文本模式，返回匹配行及行号，支持错误处理。
type FileSearchTool struct{}

// Info 返回 FileSearchTool 的元信息。
//
// 返回工具名称、描述和参数定义，用于工具注册和 AI 调用。
//
// 返回：
//   - tool.ToolInfo - 工具元信息，包含名称、描述和参数定义
func (t FileSearchTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "file_search",
		Description: "在文件中搜索匹配的文本模式，返回匹配行及行号。",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"path": {
					Type:        "string",
					Description: "要搜索的文件的绝对路径。",
				},
				"pattern": {
					Type:        "string",
					Description: "在文件中搜索的文本模式。",
				},
			},
			Required: []string{"path", "pattern"},
		},
	}
}

// Execute 执行文件搜索操作。
//
// 解析输入参数，打开文件并逐行搜索匹配模式，返回匹配行及行号。如果文件不存在或无权限访问，返回相应的错误信息。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制（当前未使用）
//   - input - JSON 格式的输入参数，包含 `path` 和 `pattern` 字段
//
// 返回：
//   - *tool.ToolResult - 执行结果，包含匹配行（格式为 `行号: 内容`）或错误信息
//   - error - 解析错误或执行错误
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


