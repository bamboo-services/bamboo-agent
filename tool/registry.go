package tool

import (
	"context"
	"fmt"
	"sync"
)

// Registry 管理所有已注册的工具，支持线程安全的注册、查找和执行
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建一个空的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册一个工具到注册表中
// 如果工具名称已存在，返回错误
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

// RegisterBatch 批量注册多个工具
// 如果任何一个工具注册失败，立即停止并返回错误
func (r *Registry) RegisterBatch(tools ...Tool) error {
	for _, tool := range tools {
		if err := r.Register(tool); err != nil {
			return err
		}
	}
	return nil
}

// Get 根据名称查找工具
// 返回工具和一个布尔值表示是否找到
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// Execute 执行指定名称的工具
// 如果工具不存在，返回错误
func (r *Registry) Execute(ctx context.Context, name string, input []byte) (*ToolResult, error) {
	tool, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool.Execute(ctx, input)
}

// List 返回所有已注册工具的信息
func (r *Registry) List() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		infos = append(infos, tool.Info())
	}
	return infos
}