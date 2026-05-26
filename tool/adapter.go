package tool

import (
	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

// BambooAdapter converts internal tool types to bamboo SDK types
type BambooAdapter struct{}

// NewBambooAdapter creates a new BambooAdapter instance
func NewBambooAdapter() *BambooAdapter {
	return &BambooAdapter{}
}

// ToBambooTool converts ToolInfo to bamboo.Tool
func (a *BambooAdapter) ToBambooTool(info ToolInfo) bamboo.Tool {
	return bamboo.Tool{
		Name:        info.Name,
		Description: info.Description,
		InputSchema: convertInputSchema(info.Parameters),
	}
}

// ToBambooTools converts multiple ToolInfo to bamboo.Tool slice
func (a *BambooAdapter) ToBambooTools(infos []ToolInfo) []bamboo.Tool {
	tools := make([]bamboo.Tool, len(infos))
	for i, info := range infos {
		tools[i] = a.ToBambooTool(info)
	}
	return tools
}

// convertInputSchema converts tool.InputSchema to bamboo.InputSchema
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