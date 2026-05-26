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

// HTTPTool 发起 HTTP 请求的工具。
//
// 提供支持 GET/POST/PUT/DELETE 等常用方法的 HTTP 客户端能力。
// 默认请求超时时间为 30 秒。
type HTTPTool struct {
	client *http.Client
}

// NewHTTPTool 创建默认配置的 HTTPTool。
//
// 初始化一个带有 30 秒超时的 HTTP 客户端。
//
// 返回：
//   - *HTTPTool - 新创建的 HTTP 工具实例
func NewHTTPTool() *HTTPTool {
	return &HTTPTool{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Info 返回工具的元数据信息。
//
// 返回工具的名称、描述和参数定义，供 AI Agent 识别和调用。
//
// 返回：
//   - tool.ToolInfo - 工具元数据，包含名称、描述和参数定义
func (h *HTTPTool) Info() tool.ToolInfo {
	return tool.ToolInfo{
		Name:        "http_request",
		Description: "发起 HTTP GET/POST/PUT/DELETE 请求",
		Parameters: tool.InputSchema{
			Type: "object",
			Properties: map[string]tool.PropertyDef{
				"method": {
					Type:        "string",
					Description: "HTTP 方法：GET、POST、PUT、DELETE",
				},
				"url": {
					Type:        "string",
					Description: "请求的目标 URL",
				},
				"headers": {
					Type:        "object",
					Description: "自定义请求头（键值对）",
				},
				"body": {
					Type:        "string",
					Description: "请求体（用于 POST/PUT）",
				},
			},
			Required: []string{"method", "url"},
		},
	}
}

// Execute 执行 HTTP 请求。
//
// 根据输入参数发起指定方法的 HTTP 请求，并返回响应结果。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - input - JSON 格式的请求参数，包含 method、url、headers、body 字段
//
// 返回：
//   - *tool.ToolResult - 请求结果，包含状态码和响应体
//   - error - 参数解析错误
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