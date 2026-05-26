package agent

// AgentEventType 代理事件类型 💖
// 用于标识不同类型的代理事件，支持流式输出和状态跟踪
type AgentEventType string

const (
	// AgentEventText 文本输出事件
	AgentEventText AgentEventType = "text"

	// AgentEventThinking 思考事件
	AgentEventThinking AgentEventType = "thinking"

	// AgentEventToolCall 工具调用事件
	AgentEventToolCall AgentEventType = "tool_call"

	// AgentEventToolResult 工具执行结果事件
	AgentEventToolResult AgentEventType = "tool_result"

	// AgentEventComplete 代理完成事件
	AgentEventComplete AgentEventType = "complete"

	// AgentEventError 错误事件 🚨
	AgentEventError AgentEventType = "error"

	// AgentEventCompress 压缩事件
	AgentEventCompress AgentEventType = "compress"
)

// AgentEvent 代理事件结构体 📦
// 包含事件类型、内容、工具调用记录、代理结果和错误信息
type AgentEvent struct {
	Type      AgentEventType  // 事件类型 (´∀｀)
	Content   string          // 事件内容
	ToolCall  *ToolCallRecord // 工具调用记录（可选）
	Result    *AgentResult    // 代理结果（可选）
	Error     error           // 错误信息（可选）
}