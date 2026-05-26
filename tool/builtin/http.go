package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bamboo-services/bamboo-agent/tool"
)

// HTTPTool makes HTTP requests.
type HTTPTool struct {
	client *http.Client
}

// NewHTTPTool creates an HTTPTool with default client.
func NewHTTPTool() *HTTPTool {
	return &HTTPTool{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Info returns the tool metadata.
func (h *HTTPTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "http_request",
		Description: "Make HTTP GET/POST/PUT/DELETE requests",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"method": {
					Type:        "string",
					Description: "HTTP method: GET, POST, PUT, DELETE",
				},
				"url": {
					Type:        "string",
					Description: "The URL to request",
				},
				"headers": {
					Type:        "object",
					Description: "Custom headers (key-value pairs)",
				},
				"body": {
					Type:        "string",
					Description: "Request body (for POST/PUT)",
				},
			},
			Required: []string{"method", "url"},
		},
	}
}

// Execute makes the HTTP request.
func (h *HTTPTool) Execute(ctx context.Context, input json.RawMessage) (*tool.ToolResult, error) {
	var params struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}, nil
	}

	method := strings.ToUpper(params.Method)
	var bodyReader io.Reader
	if params.Body != "" {
		bodyReader = strings.NewReader(params.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, params.URL, bodyReader)
	if err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("failed to create request: %v", err), IsError: true}, nil
	}

	for k, v := range params.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return &tool.ToolResult{Content: fmt.Sprintf("request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	result := fmt.Sprintf("Status: %d %s\n\n%s", resp.StatusCode, resp.Status, string(body))
	return &tool.ToolResult{Content: result, IsError: false}, nil
}