package tool

// InputSchema 定义工具输入参数的 schema。
//
// 符合 JSON Schema 规范，用于描述工具的输入结构。
type InputSchema struct {
	// Type 是参数类型，通常为 "object"。
	Type string `json:"type"`

	// Properties 定义参数对象的属性键值对。
	Properties map[string]PropertyDef `json:"properties,omitempty"`

	// Required 列出必需参数的名称数组。
	Required []string `json:"required,omitempty"`
}

// PropertyDef 定义输入 schema 中的单个属性。
//
// 描述参数的类型、描述和可选值。
type PropertyDef struct {
	// Type 是参数的数据类型，如 "string"、"number"、"boolean" 等。
	Type string `json:"type"`

	// Description 是参数的中文描述。
	Description string `json:"description,omitempty"`

	// Enum 定义参数的可选值列表，限制输入范围。
	Enum []string `json:"enum,omitempty"`
}