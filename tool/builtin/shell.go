package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/bamboo-services/bamboo-agent/tool"
)

// ShellTool 执行 Shell 命令并返回结果。
//
// 支持 stdout 和 stderr 输出，以及超时控制。
// 默认超时时间为 30 秒。
type ShellTool struct{}

// Info 返回工具的元数据信息。
//
// 返回：
//   - tool.ToolInfo - 包含工具名称、描述和参数定义
func (s *ShellTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "shell",
		Description: "执行 Shell 命令并返回 stdout 和 stderr",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"command": {
					Type:        "string",
					Description: "要执行的 Shell 命令",
				},
				"timeout": {
					Type:        "number",
					Description: "超时时间，单位秒（默认：30）",
				},
			},
			Required: []string{"command"},
		},
	}
}

// Execute 执行 Shell 命令并返回结果。
//
// 在指定的超时时间内执行命令，返回 stdout 和 stderr 的组合输出。
// 如果命令超时或执行失败，返回错误信息。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - input - JSON 格式的参数，包含 command 和可选的 timeout
//
// 返回：
//   - *tool.ToolResult - 执行结果，包含命令输出或错误信息
//   - error - 参数解析错误
func (s *ShellTool) Execute(ctx context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var params struct {
		Command string  `json:"command"`
		Timeout float64 `json:"timeout"`
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

	cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "STDERR:\n" + stderr.String()
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &tool.ToolResult{
				Content: fmt.Sprintf("command timed out after %v\n%s", timeout, output),
				IsError: true,
			}, nil
		}
		return &tool.ToolResult{
			Content: fmt.Sprintf("command failed: %v\n%s", err, output),
			IsError: true,
		}, nil
	}

	return &tool.ToolResult{Content: output, IsError: false}, nil
}