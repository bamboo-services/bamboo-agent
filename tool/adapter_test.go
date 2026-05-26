package tool

import (
	"reflect"
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

func TestBambooAdapter_ToBambooTool(t *testing.T) {
	adapter := NewBambooAdapter()

	tests := []struct {
		name string
		info ToolInfo
		want bamboo.Tool
	}{
		{
			name: "simple tool with no parameters",
			info: ToolInfo{
				Name:        "test_tool",
				Description: "A simple test tool",
				Parameters: InputSchema{
					Type:       "object",
					Properties: map[string]PropertyDef{},
					Required:   []string{},
				},
			},
			want: bamboo.Tool{
				Name:        "test_tool",
				Description: "A simple test tool",
				InputSchema: bamboo.InputSchema{
					Type:       "object",
					Properties: map[string]bamboo.PropertyDef{},
					Required:   []string{},
				},
			},
		},
		{
			name: "tool with complex parameters",
			info: ToolInfo{
				Name:        "complex_tool",
				Description: "A tool with complex parameters",
				Parameters: InputSchema{
					Type: "object",
					Properties: map[string]PropertyDef{
						"query": {
							Type:        "string",
							Description: "Search query",
						},
						"limit": {
							Type:        "integer",
							Description: "Result limit",
						},
						"sort": {
							Type: "string",
							Enum: []string{"asc", "desc"},
						},
					},
					Required: []string{"query"},
				},
			},
			want: bamboo.Tool{
				Name:        "complex_tool",
				Description: "A tool with complex parameters",
				InputSchema: bamboo.InputSchema{
					Type: "object",
					Properties: map[string]bamboo.PropertyDef{
						"query": {
							Type:        "string",
							Description: "Search query",
						},
						"limit": {
							Type:        "integer",
							Description: "Result limit",
						},
						"sort": {
							Type: "string",
							Enum: []string{"asc", "desc"},
						},
					},
					Required: []string{"query"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.ToBambooTool(tt.info)

			// Check Name
			if got.Name != tt.want.Name {
				t.Errorf("ToBambooTool() Name = %v, want %v", got.Name, tt.want.Name)
			}

			// Check Description
			if got.Description != tt.want.Description {
				t.Errorf("ToBambooTool() Description = %v, want %v", got.Description, tt.want.Description)
			}

			// Check InputSchema Type
			if got.InputSchema.Type != tt.want.InputSchema.Type {
				t.Errorf("ToBambooTool() InputSchema.Type = %v, want %v", got.InputSchema.Type, tt.want.InputSchema.Type)
			}

			// Check InputSchema Properties
			if !reflect.DeepEqual(got.InputSchema.Properties, tt.want.InputSchema.Properties) {
				t.Errorf("ToBambooTool() InputSchema.Properties = %v, want %v", got.InputSchema.Properties, tt.want.InputSchema.Properties)
			}

			// Check InputSchema Required
			if !reflect.DeepEqual(got.InputSchema.Required, tt.want.InputSchema.Required) {
				t.Errorf("ToBambooTool() InputSchema.Required = %v, want %v", got.InputSchema.Required, tt.want.InputSchema.Required)
			}
		})
	}
}

func TestBambooAdapter_ToBambooTools(t *testing.T) {
	adapter := NewBambooAdapter()

	infos := []ToolInfo{
		{
			Name:        "tool1",
			Description: "First tool",
			Parameters: InputSchema{
				Type:       "object",
				Properties: map[string]PropertyDef{},
				Required:   []string{},
			},
		},
		{
			Name:        "tool2",
			Description: "Second tool",
			Parameters: InputSchema{
				Type:       "object",
				Properties: map[string]PropertyDef{},
				Required:   []string{},
			},
		},
	}

	got := adapter.ToBambooTools(infos)

	if len(got) != len(infos) {
		t.Fatalf("ToBambooTools() returned %d tools, want %d", len(got), len(infos))
	}

	for i, tool := range got {
		if tool.Name != infos[i].Name {
			t.Errorf("ToBambooTools()[%d].Name = %v, want %v", i, tool.Name, infos[i].Name)
		}
		if tool.Description != infos[i].Description {
			t.Errorf("ToBambooTools()[%d].Description = %v, want %v", i, tool.Description, infos[i].Description)
		}
	}
}

func TestBambooAdapter_EmptyLists(t *testing.T) {
	adapter := NewBambooAdapter()

	// Test with empty slice
	emptyInfos := []ToolInfo{}
	got := adapter.ToBambooTools(emptyInfos)

	if got == nil {
		t.Error("ToBambooTools() with empty slice returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("ToBambooTools() with empty slice returned %d items, want 0", len(got))
	}

	// Test with nil properties
	infoWithNilProps := ToolInfo{
		Name:        "test",
		Description: "Test",
		Parameters: InputSchema{
			Type:       "object",
			Properties: nil,
			Required:   nil,
		},
	}

	gotTool := adapter.ToBambooTool(infoWithNilProps)

	if gotTool.InputSchema.Properties == nil {
		t.Error("ToBambooTool() InputSchema.Properties should be initialized map, not nil")
	}
	if gotTool.InputSchema.Required == nil {
		t.Error("ToBambooTool() InputSchema.Required should be initialized slice, not nil")
	}
}

func TestConvertInputSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema InputSchema
		want   bamboo.InputSchema
	}{
		{
			name: "empty schema",
			schema: InputSchema{
				Type:       "object",
				Properties: map[string]PropertyDef{},
				Required:   []string{},
			},
			want: bamboo.InputSchema{
				Type:       "object",
				Properties: map[string]bamboo.PropertyDef{},
				Required:   []string{},
			},
		},
		{
			name: "schema with single property",
			schema: InputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"param1": {
						Type:        "string",
						Description: "A parameter",
					},
				},
				Required: []string{"param1"},
			},
			want: bamboo.InputSchema{
				Type: "object",
				Properties: map[string]bamboo.PropertyDef{
					"param1": {
						Type:        "string",
						Description: "A parameter",
					},
				},
				Required: []string{"param1"},
			},
		},
		{
			name: "schema with multiple required fields",
			schema: InputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"a": {Type: "string"},
					"b": {Type: "integer"},
					"c": {Type: "boolean"},
				},
				Required: []string{"a", "b", "c"},
			},
			want: bamboo.InputSchema{
				Type: "object",
				Properties: map[string]bamboo.PropertyDef{
					"a": {Type: "string"},
					"b": {Type: "integer"},
					"c": {Type: "boolean"},
				},
				Required: []string{"a", "b", "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertInputSchema(tt.schema)

			if got.Type != tt.want.Type {
				t.Errorf("convertInputSchema() Type = %v, want %v", got.Type, tt.want.Type)
			}

			if !reflect.DeepEqual(got.Properties, tt.want.Properties) {
				t.Errorf("convertInputSchema() Properties = %v, want %v", got.Properties, tt.want.Properties)
			}

			if !reflect.DeepEqual(got.Required, tt.want.Required) {
				t.Errorf("convertInputSchema() Required = %v, want %v", got.Required, tt.want.Required)
			}
		})
	}
}

func TestBambooAdapter_PropertyDefConversion(t *testing.T) {
	adapter := NewBambooAdapter()

	info := ToolInfo{
		Name:        "test",
		Description: "Test",
		Parameters: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDef{
				"full_property": {
					Type:        "string",
					Description: "Full property with all fields",
					Enum:        []string{"option1", "option2", "option3"},
				},
				"minimal_property": {
					Type: "integer",
				},
			},
			Required: []string{"full_property", "minimal_property"},
		},
	}

	got := adapter.ToBambooTool(info)

	// Verify full property conversion
	fullProp := got.InputSchema.Properties["full_property"]
	if fullProp.Type != "string" {
		t.Errorf("PropertyDef.Type = %v, want string", fullProp.Type)
	}
	if fullProp.Description != "Full property with all fields" {
		t.Errorf("PropertyDef.Description = %v, want 'Full property with all fields'", fullProp.Description)
	}
	if !reflect.DeepEqual(fullProp.Enum, []string{"option1", "option2", "option3"}) {
		t.Errorf("PropertyDef.Enum = %v, want [option1 option2 option3]", fullProp.Enum)
	}

	// Verify minimal property conversion
	minimalProp := got.InputSchema.Properties["minimal_property"]
	if minimalProp.Type != "integer" {
		t.Errorf("PropertyDef.Type = %v, want integer", minimalProp.Type)
	}
	if minimalProp.Description != "" {
		t.Errorf("PropertyDef.Description = %v, want empty string", minimalProp.Description)
	}
	if minimalProp.Enum != nil {
		t.Errorf("PropertyDef.Enum = %v, want nil", minimalProp.Enum)
	}
}