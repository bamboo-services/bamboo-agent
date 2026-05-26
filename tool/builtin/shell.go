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

// ShellTool executes shell commands.
type ShellTool struct{}

// Info returns the tool metadata.
func (s *ShellTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "shell",
		Description: "Execute a shell command and return stdout and stderr",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"command": {
					Type:        "string",
					Description: "The shell command to execute",
				},
				"timeout": {
					Type:        "number",
					Description: "Timeout in seconds (default: 30)",
				},
			},
			Required: []string{"command"},
		},
	}
}

// Execute runs the shell command.
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