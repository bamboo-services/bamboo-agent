package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bamboo-services/bamboo-agent/tool"
)

// Bridge connects MCP server tools to the agent tool system.
type Bridge struct {
	client *Client
	tools  []MCPToolInfo
}

// NewBridge creates a new Bridge for the given MCP client.
func NewBridge(client *Client) *Bridge {
	return &Bridge{client: client}
}

// DiscoverAndConvert discovers tools from the MCP server and prepares them.
func (b *Bridge) DiscoverAndConvert(ctx context.Context) ([]tool.Tool, error) {
	tools, err := b.client.DiscoverTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover MCP tools: %w", err)
	}
	b.tools = tools

	result := make([]tool.Tool, len(tools))
	for i, t := range tools {
		result[i] = &mcpToolAdapter{
			info:   t,
			client: b.client,
		}
	}
	return result, nil
}

// AsTools returns all discovered MCP tools as tool.Tool interfaces.
// Must call DiscoverAndConvert first.
func (b *Bridge) AsTools() []tool.Tool {
	result := make([]tool.Tool, len(b.tools))
	for i, t := range b.tools {
		result[i] = &mcpToolAdapter{
			info:   t,
			client: b.client,
		}
	}
	return result
}

// mcpToolAdapter adapts an MCP tool to the tool.Tool interface.
type mcpToolAdapter struct {
	info   MCPToolInfo
	client *Client
}

// Info returns the tool metadata.
func (a *mcpToolAdapter) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        a.info.Name,
		Description: a.info.Description,
		Parameters: tool.InputSchema{
			Type:       "object",
			Properties: parseProperties(a.info.InputSchema),
		},
	}
}

// Execute calls the MCP tool.
func (a *mcpToolAdapter) Execute(ctx context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		return &tool.ToolResult{
			Content: fmt.Sprintf("invalid input: %v", err),
			IsError: true,
		}, nil
	}

	result, err := a.client.CallTool(ctx, a.info.Name, args)
	if err != nil {
		return &tool.ToolResult{
			Content: fmt.Sprintf("MCP tool call failed: %v", err),
			IsError: true,
		}, nil
	}

	// Combine content texts
	var content string
	for _, c := range result.Content {
		if c.Type == "text" && c.Text != "" {
			content += c.Text
		}
	}

	return &tool.ToolResult{
		Content: content,
		IsError: result.IsError,
	}, nil
}

// parseProperties extracts property definitions from the MCP input schema.
func parseProperties(schema json.RawMessage) map[string]tool.PropertyDef {
	if schema == nil {
		return nil
	}

	var raw struct {
		Type       string                            `json:"type"`
		Properties map[string]map[string]interface{} `json:"properties"`
		Required   []string                          `json:"required"`
	}
	if err := json.Unmarshal(schema, &raw); err != nil {
		return nil
	}

	// Return nil if no properties defined
	if raw.Properties == nil {
		return nil
	}

	props := make(map[string]tool.PropertyDef)
	for name, def := range raw.Properties {
		pd := tool.PropertyDef{}
		if t, ok := def["type"].(string); ok {
			pd.Type = t
		}
		if d, ok := def["description"].(string); ok {
			pd.Description = d
		}
		props[name] = pd
	}

	return props
}