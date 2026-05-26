package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewChannel 测试创建 Channel。
func TestNewChannel(t *testing.T) {
	ch := NewChannel(10)
	assert.NotNil(t, ch, "Channel should not be nil")
	assert.NotNil(t, ch.subscribers, "Channel subscribers should not be nil")
	assert.Equal(t, 10, ch.buffer, "Channel buffer size should be 10")
}

// TestAgentMessage_FieldAccess 测试 AgentMessage 字段访问。
func TestAgentMessage_FieldAccess(t *testing.T) {
	msg := AgentMessage{
		From:    "agent-001",
		To:      "agent-002",
		Content: "Hello from agent-001",
		Data:    map[string]string{"key": "value"},
	}

	assert.Equal(t, "agent-001", msg.From, "From field should match")
	assert.Equal(t, "agent-002", msg.To, "To field should match")
	assert.Equal(t, "Hello from agent-001", msg.Content, "Content field should match")
	assert.NotNil(t, msg.Data, "Data field should not be nil")
}

// TestChannel_Subscribe 测试订阅功能。
func TestChannel_Subscribe(t *testing.T) {
	ch := NewChannel(10)

	sub := ch.Subscribe("agent-001")
	assert.NotNil(t, sub, "Subscription channel should not be nil")

	ch.mu.RLock()
	subs := ch.subscribers["agent-001"]
	ch.mu.RUnlock()
	assert.Equal(t, 1, len(subs), "Should have 1 subscriber for agent-001")
}

// TestChannel_Send_Receive 测试发送和接收消息。
func TestChannel_Send_Receive(t *testing.T) {
	ctx := context.Background()
	ch := NewChannel(10)

	sub := ch.Subscribe("agent-001")

	msg := AgentMessage{
		From:    "sender",
		To:      "agent-001",
		Content: "Test message",
		Data:    nil,
	}

	err := ch.Send(ctx, msg)
	assert.NoError(t, err, "Send should not return error")

	received, err := ch.Receive(ctx, sub)
	assert.NoError(t, err, "Receive should not return error")
	assert.Equal(t, msg.From, received.From, "Received message From should match")
	assert.Equal(t, msg.To, received.To, "Received message To should match")
	assert.Equal(t, msg.Content, received.Content, "Received message Content should match")
}

// TestChannel_Broadcast 测试广播消息。
func TestChannel_Broadcast(t *testing.T) {
	ctx := context.Background()
	ch := NewChannel(10)

	sub1 := ch.Subscribe("agent-001")
	sub2 := ch.Subscribe("agent-002")
	sub3 := ch.Subscribe("agent-003")

	msg := AgentMessage{
		From:    "broadcaster",
		To:      "",
		Content: "Broadcast message",
		Data:    nil,
	}

	err := ch.Broadcast(ctx, msg)
	assert.NoError(t, err, "Broadcast should not return error")

	received1, _ := ch.Receive(ctx, sub1)
	assert.Equal(t, msg.Content, received1.Content, "agent-001 should receive broadcast")

	received2, _ := ch.Receive(ctx, sub2)
	assert.Equal(t, msg.Content, received2.Content, "agent-002 should receive broadcast")

	received3, _ := ch.Receive(ctx, sub3)
	assert.Equal(t, msg.Content, received3.Content, "agent-003 should receive broadcast")
}

// TestChannel_ContextCancellation 测试上下文取消。
func TestChannel_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := NewChannel(10)

	sub := ch.Subscribe("agent-001")

	cancel()

	_, err := ch.Receive(ctx, sub)
	assert.Error(t, err, "Receive should return error when context is cancelled")
	assert.Equal(t, context.Canceled, err, "Error should be context.Canceled")
}

// TestChannel_SendWithTimeout 测试发送超时。
func TestChannel_SendWithTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := NewChannel(0)

	_ = ch.Subscribe("agent-001")

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	msg := AgentMessage{
		From:    "sender",
		To:      "agent-001",
		Content: "Test message",
		Data:    nil,
	}

	err := ch.Send(ctx, msg)
	assert.Error(t, err, "Send should return error when context is cancelled")
	assert.Equal(t, context.Canceled, err, "Error should be context.Canceled")
}

// TestChannel_Close 测试关闭 Channel。
func TestChannel_Close(t *testing.T) {
	ch := NewChannel(10)

	sub1 := ch.Subscribe("agent-001")
	sub2 := ch.Subscribe("agent-002")

	ch.Close()

	ch.mu.RLock()
	isEmpty := len(ch.subscribers) == 0
	ch.mu.RUnlock()
	assert.True(t, isEmpty, "Subscribers should be cleared after Close")

	_, ok := <-sub1
	assert.False(t, ok, "sub1 should be closed")

	_, ok = <-sub2
	assert.False(t, ok, "sub2 should be closed")
}

// TestChannel_MultipleSubscribersForSameAgent 测试同一 Agent 多个订阅者。
func TestChannel_MultipleSubscribersForSameAgent(t *testing.T) {
	ctx := context.Background()
	ch := NewChannel(10)

	sub1 := ch.Subscribe("agent-001")
	sub2 := ch.Subscribe("agent-001")

	msg := AgentMessage{
		From:    "sender",
		To:      "agent-001",
		Content: "Test message",
		Data:    nil,
	}

	err := ch.Send(ctx, msg)
	assert.NoError(t, err, "Send should not return error")

	received1, _ := ch.Receive(ctx, sub1)
	assert.Equal(t, msg.Content, received1.Content, "sub1 should receive message")

	received2, _ := ch.Receive(ctx, sub2)
	assert.Equal(t, msg.Content, received2.Content, "sub2 should receive message")
}

// TestChannel_AgentMessageWithData 测试带数据的消息。
func TestChannel_AgentMessageWithData(t *testing.T) {
	ctx := context.Background()
	ch := NewChannel(10)

	sub := ch.Subscribe("agent-001")

	msg := AgentMessage{
		From:    "agent-002",
		To:      "agent-001",
		Content: "Data message",
		Data: map[string]interface{}{
			"task_id":  "task-123",
			"priority": "high",
			"count":    42,
		},
	}

	err := ch.Send(ctx, msg)
	assert.NoError(t, err, "Send should not return error")

	received, _ := ch.Receive(ctx, sub)
	assert.Equal(t, msg.Data, received.Data, "Data field should match")

	data, ok := received.Data.(map[string]interface{})
	assert.True(t, ok, "Data should be a map")
	assert.Equal(t, "task-123", data["task_id"], "task_id should match")
	assert.Equal(t, "high", data["priority"], "priority should match")
	assert.Equal(t, int(42), data["count"], "count should be 42")
}

// TestChannel_ReceiveWithTimeout 测试接收超时。
func TestChannel_ReceiveWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	ch := NewChannel(10)
	sub := ch.Subscribe("agent-001")

	_, err := ch.Receive(ctx, sub)
	assert.Error(t, err, "Receive should return error when timeout")
	assert.Equal(t, context.DeadlineExceeded, err, "Error should be context.DeadlineExceeded")
}