package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/bamboo-services/bamboo-agent/tool"
)

// CodeExecTool 在支持的语言中执行代码片段。
//
// 支持的语言包括 Go 和 Python，每种语言都使用安全的临时目录执行。
// 可以指定超时时间，默认为 30 秒。
type CodeExecTool struct{}

// Info 返回工具的元数据信息。
//
// 返回：
//   - tool.ToolInfo - 包含工具名称、描述、参数schema等信息
func (c *CodeExecTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "code_exec",
		Description: "在支持的语言中执行代码片段（Go 或 Python）",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"language": {
					Type:        "string",
					Description: "编程语言：go 或 python",
					Enum:        []string{"go", "python"},
				},
				"code": {
					Type:        "string",
					Description: "要执行的代码",
				},
				"timeout": {
					Type:        "number",
					Description: "超时时间（秒），默认为 30",
				},
			},
			Required: []string{"language", "code"},
		},
	}
}

// Execute 运行代码片段。
//
// 根据指定的语言调用相应的执行器，支持超时控制。
// 执行失败时返回错误信息，执行成功时返回标准输出。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - input - JSON 格式的输入参数，包含 language、code 和可选的 timeout
//
// 返回：
//   - *tool.ToolResult - 执行结果，包含输出内容或错误信息
//   - error - 执行错误（仅内部错误，执行失败通过 ToolResult.IsError 标识）
func (c *CodeExecTool) Execute(ctx context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var params struct {
		Language string  `json:"language"`
		Code     string  `json:"code"`
		Timeout  float64 `json:"timeout"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout * float64(time.Second))
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch params.Language {
	case "go":
		return c.execGo(ctx, params.Code)
	case "python":
		return c.execPython(ctx, params.Code)
	default:
		return &tool.ToolResult{
			Content: fmt.Sprintf("unsupported language: %s (supported: go, python)", params.Language),
			IsError: true,
		}, nil
	}
}

// execGo 在临时目录中执行 Go 代码。
//
// 将代码写入临时文件并使用 `go run` 执行。
// 执行后自动清理临时目录。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - code - 要执行的 Go 代码
//
// 返回：
//   - *tool.ToolResult - 执行结果，包含标准输出或错误信息
//   - error - 内部错误（如创建临时目录失败）
func (c *CodeExecTool) execGo(ctx context.Context, code string) (*tool.ToolResult, error) {
	tmpDir, err := os.MkdirTemp("", "bamboo-code-*")
	if err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("failed to create temp dir: %v", err), IsError: true}, nil
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcFile, []byte(code), 0o644); err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("failed to write source: %v", err), IsError: true}, nil
	}

	cmd := exec.CommandContext(ctx, "go", "run", srcFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String()

	if err != nil {
		return &tool.ToolResult{
			Content: fmt.Sprintf("execution failed: %v\n%s", err, stderr.String()),
			IsError: true,
		}, nil
	}

	return &tool.ToolResult{Content: output, IsError: false}, nil
}

// execPython 在临时目录中执行 Python 代码。
//
// 将代码写入临时文件并使用 `python3` 执行。
// 执行后自动清理临时目录。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - code - 要执行的 Python 代码
//
// 返回：
//   - *tool.ToolResult - 执行结果，包含标准输出或错误信息
//   - error - 内部错误（如创建临时目录失败）
func (c *CodeExecTool) execPython(ctx context.Context, code string) (*tool.ToolResult, error) {
	tmpDir, err := os.MkdirTemp("", "bamboo-code-*")
	if err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("failed to create temp dir: %v", err), IsError: true}, nil
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "script.py")
	if err := os.WriteFile(srcFile, []byte(code), 0o644); err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("failed to write source: %v", err), IsError: true}, nil
	}

	cmd := exec.CommandContext(ctx, "python3", srcFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String()

	if err != nil {
		return &tool.ToolResult{
			Content: fmt.Sprintf("execution failed: %v\n%s", err, stderr.String()),
			IsError: true,
		}, nil
	}

	return &tool.ToolResult{Content: output, IsError: false}, nil
}
