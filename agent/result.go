package agent

import (
	"encoding/json"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// ToolCallRecord 工具调用记录 🛠️
// 记录每次工具调用的详细信息，包括输入参数和执行结果
type ToolCallRecord struct {
	ID      string          // 工具调用唯一标识
	Name    string          // 工具名称
	Input   json.RawMessage // 工具输入参数（JSON格式）
	Result  string          // 工具执行结果
	IsError bool            // 是否为错误结果
}

// AgentResult 代理执行结果 ✨
// 包含代理执行的完整信息，包括消息、工具调用和资源使用
type AgentResult struct {
	Content    string              // 最终内容输出
	Messages   []bamboo.BambooMessage // 消息历史记录
	ToolCalls  []ToolCallRecord    // 工具调用记录列表
	Usage      bamboo.Usage        // Token使用统计
	Iterations int                 // 迭代次数
}