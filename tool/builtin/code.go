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

// CodeExecTool executes code snippets in supported languages.
type CodeExecTool struct{}

// Info returns the tool metadata.
func (c *CodeExecTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "code_exec",
		Description: "Execute code snippets in Go or Python",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"language": {
					Type:        "string",
					Description: "Programming language: go, python",
					Enum:        []string{"go", "python"},
				},
				"code": {
					Type:        "string",
					Description: "The code to execute",
				},
				"timeout": {
					Type:        "number",
					Description: "Timeout in seconds (default: 30)",
				},
			},
			Required: []string{"language", "code"},
		},
	}
}

// Execute runs the code snippet.
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
