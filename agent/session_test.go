package agent

import (
	"testing"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// TestNewSession 测试创建新会话。
func TestNewSession(t *testing.T) {
	registry := tool.NewRegistry()
	session := NewSession(registry)

	if session == nil {
		t.Fatal("NewSession returned nil")
	}

	// 验证消息历史为空
	messages := session.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected empty messages, got %d messages", len(messages))
	}

	// 验证工具注册表正确设置
	if session.Tools() != registry {
		t.Error("Tools registry not correctly set")
	}
}

// TestNewSessionWithNilRegistry 测试使用 nil 工具注册表创建会话。
func TestNewSessionWithNilRegistry(t *testing.T) {
	session := NewSession(nil)

	if session == nil {
		t.Fatal("NewSession with nil registry returned nil")
	}

	// nil 工具注册表也是合法的
	if session.Tools() != nil {
		t.Error("Expected nil tools registry, got non-nil")
	}
}

// TestAppendMessage 测试追加消息到历史记录。
func TestAppendMessage(t *testing.T) {
	registry := tool.NewRegistry()
	session := NewSession(registry)

	// 创建测试消息
	msg := bamboo.NewUserMessage("Hello")

	session.AppendMessage(msg)
	messages := session.GetMessages()

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != bamboo.RoleUser {
		t.Errorf("Expected RoleUser, got %v", messages[0].Role)
	}
}

// TestGetMessages 测试获取所有消息。
func TestGetMessages(t *testing.T) {
	registry := tool.NewRegistry()
	session := NewSession(registry)

	// 添加多条消息
	msgs := []bamboo.BambooMessage{
		bamboo.NewUserMessage("User message"),
		bamboo.NewAssistantMessage("Assistant message"),
		bamboo.NewUserMessage("Another user message"),
	}

	for _, msg := range msgs {
		session.AppendMessage(msg)
	}

	messages := session.GetMessages()

	if len(messages) != len(msgs) {
		t.Errorf("Expected %d messages, got %d", len(msgs), len(messages))
	}

	// 验证每条消息的内容
	for i, msg := range msgs {
		if messages[i].Role != msg.Role {
			t.Errorf("Message %d: expected role %v, got %v", i, msg.Role, messages[i].Role)
		}
	}
}

// TestGetMessagesReturnsCopy 测试 GetMessages 返回副本。
func TestGetMessagesReturnsCopy(t *testing.T) {
	registry := tool.NewRegistry()
	session := NewSession(registry)

	// 添加一条消息
	msg := bamboo.NewUserMessage("Original")
	session.AppendMessage(msg)

	// 获取消息副本
	messages := session.GetMessages()

	// 修改副本
	messages[0] = bamboo.NewUserMessage("Modified")

	// 再次获取消息，应该仍然是原始内容
	messagesAgain := session.GetMessages()

	if len(messagesAgain[0].Content) == 0 {
		t.Fatal("Expected content block")
	}
	if messagesAgain[0].Content[0].Text != "Original" {
		t.Errorf("GetMessages did not return a copy. Expected 'Original', got '%s'", messagesAgain[0].Content[0].Text)
	}
}

// TestClear 测试清空消息历史。
func TestClear(t *testing.T) {
	registry := tool.NewRegistry()
	session := NewSession(registry)

	// 添加多条消息
	for i := 0; i < 5; i++ {
		msg := bamboo.NewUserMessage(string(rune('a' + i)))
		session.AppendMessage(msg)
	}

	// 验证消息已添加
	messages := session.GetMessages()
	if len(messages) != 5 {
		t.Errorf("Expected 5 messages before clear, got %d", len(messages))
	}

	// 清空消息
	session.Clear()

	// 验证消息已清空
	messages = session.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}

// TestMultipleAppendsPreserveOrder 测试多次追加保持顺序。
func TestMultipleAppendsPreserveOrder(t *testing.T) {
	registry := tool.NewRegistry()
	session := NewSession(registry)

	// 添加多条消息，交替使用 User 和 Assistant
	msgs := []bamboo.BambooMessage{
		bamboo.NewUserMessage("Message 1"),
		bamboo.NewAssistantMessage("Response 1"),
		bamboo.NewUserMessage("Message 2"),
		bamboo.NewAssistantMessage("Response 2"),
		bamboo.NewUserMessage("Message 3"),
	}

	for _, msg := range msgs {
		session.AppendMessage(msg)
	}

	messages := session.GetMessages()

	if len(messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(messages))
	}

	// 验证顺序正确
	expectedOrder := []bamboo.MessageRole{
		bamboo.RoleUser,
		bamboo.RoleAssistant,
		bamboo.RoleUser,
		bamboo.RoleAssistant,
		bamboo.RoleUser,
	}

	for i, expected := range expectedOrder {
		if messages[i].Role != expected {
			t.Errorf("Message %d: expected role %v, got %v", i, expected, messages[i].Role)
		}
	}
}