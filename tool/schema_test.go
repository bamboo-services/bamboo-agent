package tool

import (
	"encoding/json"
	"testing"
)

// TestInputSchemaRoundTrip 测试 InputSchema 的 JSON 序列化与反序列化。
func TestInputSchemaRoundTrip(t *testing.T) {
	original := InputSchema{
		Type: "object",
		Properties: map[string]PropertyDef{
			"name": {
				Type:        "string",
				Description: "User name",
			},
			"age": {
				Type:        "integer",
				Description: "User age",
			},
			"status": {
				Type:        "string",
				Description: "User status",
				Enum:        []string{"active", "inactive"},
			},
		},
		Required: []string{"name", "age"},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal InputSchema: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled InputSchema
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal InputSchema: %v", err)
	}

	// Verify equals original
	if unmarshaled.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", unmarshaled.Type, original.Type)
	}

	if len(unmarshaled.Properties) != len(original.Properties) {
		t.Errorf("Properties length mismatch: got %d, want %d", len(unmarshaled.Properties), len(original.Properties))
	}

	for key, prop := range original.Properties {
		unmarshaledProp, ok := unmarshaled.Properties[key]
		if !ok {
			t.Errorf("Property %q missing in unmarshaled schema", key)
			continue
		}
		if unmarshaledProp.Type != prop.Type {
			t.Errorf("Property %q Type mismatch: got %q, want %q", key, unmarshaledProp.Type, prop.Type)
		}
		if unmarshaledProp.Description != prop.Description {
			t.Errorf("Property %q Description mismatch: got %q, want %q", key, unmarshaledProp.Description, prop.Description)
		}
		if len(unmarshaledProp.Enum) != len(prop.Enum) {
			t.Errorf("Property %q Enum length mismatch: got %d, want %d", key, len(unmarshaledProp.Enum), len(prop.Enum))
		}
		for i, enum := range prop.Enum {
			if unmarshaledProp.Enum[i] != enum {
				t.Errorf("Property %q Enum[%d] mismatch: got %q, want %q", key, i, unmarshaledProp.Enum[i], enum)
			}
		}
	}

	if len(unmarshaled.Required) != len(original.Required) {
		t.Errorf("Required length mismatch: got %d, want %d", len(unmarshaled.Required), len(original.Required))
	}
	for i, req := range original.Required {
		if unmarshaled.Required[i] != req {
			t.Errorf("Required[%d] mismatch: got %q, want %q", i, unmarshaled.Required[i], req)
		}
	}
}

// TestPropertyDefRoundTrip 测试 PropertyDef 的 JSON 序列化与反序列化。
func TestPropertyDefRoundTrip(t *testing.T) {
	original := PropertyDef{
		Type:        "string",
		Description: "User status",
		Enum:        []string{"active", "inactive"},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal PropertyDef: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled PropertyDef
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal PropertyDef: %v", err)
	}

	// Verify equals original
	if unmarshaled.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", unmarshaled.Type, original.Type)
	}
	if unmarshaled.Description != original.Description {
		t.Errorf("Description mismatch: got %q, want %q", unmarshaled.Description, original.Description)
	}
	if len(unmarshaled.Enum) != len(original.Enum) {
		t.Errorf("Enum length mismatch: got %d, want %d", len(unmarshaled.Enum), len(original.Enum))
	}
	for i, enum := range original.Enum {
		if unmarshaled.Enum[i] != enum {
			t.Errorf("Enum[%d] mismatch: got %q, want %q", i, unmarshaled.Enum[i], enum)
		}
	}
}

// TestInputSchemaOmitemptyFields 测试 InputSchema 的 omitempty 字段行为。
func TestInputSchemaOmitemptyFields(t *testing.T) {
	tests := []struct {
		name     string
		schema   InputSchema
		wantJSON string
	}{
		{
			name: "All fields present",
			schema: InputSchema{
				Type:       "object",
				Properties: map[string]PropertyDef{"name": {Type: "string"}},
				Required:   []string{"name"},
			},
			wantJSON: `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`,
		},
		{
			name:     "Only Type field",
			schema:   InputSchema{Type: "string"},
			wantJSON: `{"type":"string"}`,
		},
		{
			name: "Type and Properties only",
			schema: InputSchema{
				Type:       "object",
				Properties: map[string]PropertyDef{"name": {Type: "string"}},
			},
			wantJSON: `{"type":"object","properties":{"name":{"type":"string"}}}`,
		},
		{
			name:     "Type and Required only",
			schema:   InputSchema{Type: "object", Required: []string{"name"}},
			wantJSON: `{"type":"object","required":["name"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.schema)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			gotJSON := string(data)
			if gotJSON != tt.wantJSON {
				t.Errorf("JSON mismatch:\ngot  %q\nwant %q", gotJSON, tt.wantJSON)
			}
		})
	}
}

// TestPropertyDefOmitemptyFields 测试 PropertyDef 的 omitempty 字段行为。
func TestPropertyDefOmitemptyFields(t *testing.T) {
	tests := []struct {
		name     string
		prop     PropertyDef
		wantJSON string
	}{
		{
			name:     "Only Type field",
			prop:     PropertyDef{Type: "string"},
			wantJSON: `{"type":"string"}`,
		},
		{
			name:     "Type and Description",
			prop:     PropertyDef{Type: "string", Description: "A name"},
			wantJSON: `{"type":"string","description":"A name"}`,
		},
		{
			name:     "Type and Enum",
			prop:     PropertyDef{Type: "string", Enum: []string{"a", "b"}},
			wantJSON: `{"type":"string","enum":["a","b"]}`,
		},
		{
			name:     "All fields",
			prop:     PropertyDef{Type: "string", Description: "A name", Enum: []string{"a", "b"}},
			wantJSON: `{"type":"string","description":"A name","enum":["a","b"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.prop)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			gotJSON := string(data)
			if gotJSON != tt.wantJSON {
				t.Errorf("JSON mismatch:\ngot  %q\nwant %q", gotJSON, tt.wantJSON)
			}
		})
	}
}