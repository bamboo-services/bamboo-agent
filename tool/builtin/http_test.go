package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPTool_Info(t *testing.T) {
	tool := NewHTTPTool()
	info := tool.Info()

	if info.Name != "http_request" {
		t.Errorf("Expected name 'http_request', got '%s'", info.Name)
	}

	if info.Description != "Make HTTP GET/POST/PUT/DELETE requests" {
		t.Errorf("Unexpected description: '%s'", info.Description)
	}

	required := info.Parameters.Required
	if len(required) != 2 {
		t.Errorf("Expected 2 required parameters, got %d", len(required))
	}

	// Check required parameters are method and url
	requiredMap := make(map[string]bool)
	for _, r := range required {
		requiredMap[r] = true
	}

	if !requiredMap["method"] || !requiredMap["url"] {
		t.Error("Method and url should be required parameters")
	}
}

func TestHTTPTool_GET_Request(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	input := map[string]interface{}{
		"method": "GET",
		"url":    server.URL,
	}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(context.Background(), inputJSON)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.Content)
	}

	if !strings.Contains(result.Content, "200") || !strings.Contains(result.Content, "Hello, World!") {
		t.Errorf("Unexpected result: %s", result.Content)
	}
}

func TestHTTPTool_POST_Request(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != "test payload" {
			t.Errorf("Expected body 'test payload', got '%s'", string(body))
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Created"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	input := map[string]interface{}{
		"method": "POST",
		"url":    server.URL,
		"body":   "test payload",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(context.Background(), inputJSON)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.Content)
	}

	if !strings.Contains(result.Content, "201") || !strings.Contains(result.Content, "Created") {
		t.Errorf("Unexpected result: %s", result.Content)
	}
}

func TestHTTPTool_Custom_Headers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer token123" {
			t.Errorf("Expected Authorization header 'Bearer token123', got '%s'", authHeader)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tool := NewHTTPTool()
	input := map[string]interface{}{
		"method":  "GET",
		"url":     server.URL,
		"headers": map[string]string{"Authorization": "Bearer token123"},
	}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(context.Background(), inputJSON)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.Content)
	}
}

func TestHTTPTool_Invalid_URL(t *testing.T) {
	tool := NewHTTPTool()
	input := map[string]interface{}{
		"method": "GET",
		"url":    "://invalid-url",
	}
	inputJSON, _ := json.Marshal(input)

	result, err := tool.Execute(context.Background(), inputJSON)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected IsError=true for invalid URL")
	}

	if !strings.Contains(result.Content, "failed to create request") && !strings.Contains(result.Content, "request failed") {
		t.Errorf("Expected error message to contain request failure, got: %s", result.Content)
	}
}

func TestHTTPTool_Invalid_Input(t *testing.T) {
	tool := NewHTTPTool()
	input := "invalid json"

	result, err := tool.Execute(context.Background(), []byte(input))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected IsError=true for invalid input")
	}

	if !strings.Contains(result.Content, "invalid input") {
		t.Errorf("Expected error message to contain 'invalid input', got: %s", result.Content)
	}
}