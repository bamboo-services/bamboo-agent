package tool

import (
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// BambooAdapter 将内部工具类型转换为 bamboo SDK 类型。
//
// 用于将本地的 ToolInfo 等类型转换为 BM-SDK 中定义的 bamboo.Tool 类型。
type BambooAdapter struct{}

// NewBambooAdapter 创建一个 BambooAdapter 实例。
//
// 返回：
//   - *BambooAdapter - 新创建的适配器实例
func NewBambooAdapter() *BambooAdapter {
	return &BambooAdapter{}
}

// ToBambooTool 将 ToolInfo 转换为 bamboo.Tool。
//
// 将本地的工具信息结构转换为 BM-SDK 中定义的工具格式，
// 包括工具名称、描述和输入参数结构。
//
// 参数说明：
//   - info - 本地工具信息
//
// 返回：
//   - bamboo.Tool - BM-SDK 工具格式
func (a *BambooAdapter) ToBambooTool(info ToolInfo) bamboo.Tool {
	return bamboo.Tool{
		Name:        info.Name,
		Description: info.Description,
		InputSchema: convertInputSchema(info.Parameters),
	}
}

// ToBambooTools 将多个 ToolInfo 转换为 bamboo.Tool 切片。
//
// 批量转换本地工具信息为 BM-SDK 工具格式。
//
// 参数说明：
//   - infos - 本地工具信息列表
//
// 返回：
//   - []bamboo.Tool - BM-SDK 工具格式切片
func (a *BambooAdapter) ToBambooTools(infos []ToolInfo) []bamboo.Tool {
	tools := make([]bamboo.Tool, len(infos))
	for i, info := range infos {
		tools[i] = a.ToBambooTool(info)
	}
	return tools
}

// convertInputSchema 将 tool.InputSchema 转换为 bamboo.InputSchema。
//
// 将本地的输入参数结构转换为 BM-SDK 中定义的输入参数格式。
// 注意：tool.PropertyDef 不包含 Items 字段，因此该字段会被设置为 nil。
//
// 参数说明：
//   - schema - 本地输入参数结构
//
// 返回：
//   - bamboo.InputSchema - BM-SDK 输入参数格式
func convertInputSchema(schema InputSchema) bamboo.InputSchema {
	props := make(map[string]bamboo.PropertyDef, len(schema.Properties))
	for k, v := range schema.Properties {
		props[k] = bamboo.PropertyDef{
			Type:        v.Type,
			Description: v.Description,
			Enum:        v.Enum,
			// Items field is left nil as tool.PropertyDef doesn't have it
		}
	}
	required := schema.Required
	if required == nil {
		required = []string{}
	}
	return bamboo.InputSchema{
		Type:       schema.Type,
		Properties: props,
		Required:   required,
	}
}