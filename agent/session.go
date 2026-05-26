package agent

import (
	"sync"

	bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
	"github.com/bamboo-services/bamboo-agent/tool"
)

// Session 管理内存中的对话历史
//
// 提供线程安全的消息追加、获取和清空功能
type Session struct {
	mu       sync.RWMutex
	messages []bamboo.BambooMessage
	tools    *tool.Registry
}

// NewSession 创建一个新的会话，关联指定的工具注册表
func NewSession(tools *tool.Registry) *Session {
	return &Session{
		messages: make([]bamboo.BambooMessage, 0),
		tools:    tools,
	}
}

// AppendMessage 将消息追加到对话历史中
func (s *Session) AppendMessage(msg bamboo.BambooMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

// GetMessages 返回对话历史中所有消息的副本
// 返回副本以防止外部修改影响会话状态
func (s *Session) GetMessages() []bamboo.BambooMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]bamboo.BambooMessage, len(s.messages))
	copy(result, s.messages)
	return result
}

// Clear 清空对话历史
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = s.messages[:0]
}

// Tools 返回与此会话关联的工具注册表
func (s *Session) Tools() *tool.Registry {
	return s.tools
}