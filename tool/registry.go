package tool

import (
	"context"
	"fmt"
	"sync"
)

// Registry 管理所有已注册的工具，支持线程安全的注册、查找和执行。
//
// 使用读写锁（sync.RWMutex）保证并发安全，支持单个工具注册、批量注册、
// 按名称查找、执行工具和列出所有工具信息。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建一个空的工具注册表。
//
// 返回一个初始化完成的 `Registry` 实例，内部工具映射表已创建完成。
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册一个工具到注册表中。
//
// 如果工具名称已存在，返回错误。注册操作是线程安全的。
//
// 参数说明：
//   - tool - 要注册的工具实例
//
// 返回：
//   - error - 注册错误，如工具名称已存在
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Info().Name
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool '%s' is already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// RegisterBatch 批量注册多个工具。
//
// 如果任何一个工具注册失败，立即停止并返回错误。
//
// 参数说明：
//   - tools - 要批量注册的工具列表
//
// 返回：
//   - error - 注册错误，如某个工具名称已存在
func (r *Registry) RegisterBatch(tools ...Tool) error {
	for _, tool := range tools {
		if err := r.Register(tool); err != nil {
			return err
		}
	}
	return nil
}

// Get 根据名称查找工具。
//
// 查找操作是线程安全的（读锁）。
//
// 参数说明：
//   - name - 工具名称
//
// 返回：
//   - Tool - 找到的工具实例
//   - bool - 是否找到（true 为找到，false 为未找到）
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// Execute 执行指定名称的工具。
//
// 如果工具不存在，返回错误。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - name - 工具名称
//   - input - 工具输入参数（JSON 字节数组）
//
// 返回：
//   - *ToolResult - 工具执行结果
//   - error - 执行错误，如工具不存在或执行失败
func (r *Registry) Execute(ctx context.Context, name string, input []byte) (*ToolResult, error) {
	tool, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool.Execute(ctx, input)
}

// List 返回所有已注册工具的信息。
//
// 返回的信息列表顺序不确定，调用方不应依赖顺序。
//
// 返回：
//   - []ToolInfo - 所有已注册工具的信息列表
func (r *Registry) List() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		infos = append(infos, tool.Info())
	}
	return infos
}