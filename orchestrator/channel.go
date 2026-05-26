package orchestrator

import (
	"context"
	"sync"
)

// AgentMessage 保存 Agent 之间传递的消息数据。
//
// 用于多 Agent 协作场景下的消息通信，支持：
//   - 点对点消息传递
//   - 广播消息
//   - 任意结构化数据附加
type AgentMessage struct {
	// From 是发送者 Agent 或任务 ID。
	From string

	// To 是接收者 Agent 或任务 ID（空表示广播）。
	To string

	// Content 是消息文本内容。
	Content string

	// Data 携带任意结构化数据（可选）。
	Data interface{}
}

// Channel 提供 Agent 之间的消息传递能力。
//
// 基于 channel 的发布-订阅模式，支持：
//   - 点对点消息发送
//   - 广播消息
//   - 并发安全的订阅管理
//
// 使用前通过 `NewChannel` 创建实例，通过 `Subscribe` 订阅消息，
// 通过 `Send` 或 `Broadcast` 发送消息。
type Channel struct {
	mu          sync.RWMutex
	subscribers map[string][]chan AgentMessage
	buffer      int
}

// NewChannel 创建指定缓冲区大小的新 Channel。
//
// 缓冲区大小影响消息发送的阻塞行为：
//   - buffer 为 0：发送方会阻塞，直到接收方读取消息
//   - buffer 大于 0：缓冲区未满时发送方不会阻塞
//
// 参数说明：
//   - buffer - 消息通道的缓冲区大小
//
// 返回：
//   - *Channel - 新创建的 Channel 实例
func NewChannel(buffer int) *Channel {
	return &Channel{
		subscribers: make(map[string][]chan AgentMessage),
		buffer:      buffer,
	}
}

// Subscribe 为指定的 Agent ID 注册消息订阅。
//
// 每次调用都会返回一个新的消息接收通道，允许多个订阅者接收同一 ID 的消息。
//
// 参数说明：
//   - agentID - Agent 或任务 ID，用于标识订阅者
//
// 返回：
//   - <-chan AgentMessage - 消息接收通道
func (c *Channel) Subscribe(agentID string) <-chan AgentMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan AgentMessage, c.buffer)
	c.subscribers[agentID] = append(c.subscribers[agentID], ch)
	return ch
}

// Send 向指定的 Agent 发送单条消息。
//
// 消息会发送到所有订阅了 `msg.To` ID 的通道。如果接收方通道已满且
// 超过 context 超时时间，返回错误。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - msg - 要发送的消息
//
// 返回：
//   - error - 上下文取消或超时时返回错误，否则返回 nil
func (c *Channel) Send(ctx context.Context, msg AgentMessage) error {
	c.mu.RLock()
	subs := c.subscribers[msg.To]
	c.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- msg:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Broadcast 向所有订阅者广播消息。
//
// 消息会发送到所有已注册的订阅者通道。如果任一接收方通道已满且
// 超过 context 超时时间，返回错误。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - msg - 要广播的消息
//
// 返回：
//   - error - 上下文取消或超时时返回错误，否则返回 nil
func (c *Channel) Broadcast(ctx context.Context, msg AgentMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, subs := range c.subscribers {
		for _, ch := range subs {
			select {
			case ch <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

// Receive 阻塞等待接收通道中的消息。
//
// 当通道有消息时立即返回，如果上下文取消则返回错误。
//
// 参数说明：
//   - ctx - 上下文，用于取消和超时控制
//   - sub - 消息接收通道
//
// 返回：
//   - AgentMessage - 接收到的消息
//   - error - 上下文取消时返回错误，否则返回 nil
func (c *Channel) Receive(ctx context.Context, sub <-chan AgentMessage) (AgentMessage, error) {
	select {
	case msg := <-sub:
		return msg, nil
	case <-ctx.Done():
		return AgentMessage{}, ctx.Err()
	}
}

// Close 关闭所有订阅者通道并清空订阅者列表。
//
// 调用后，所有订阅者通道将收到零值并关闭，不能再用于接收消息。
// 这是一个幂等操作，多次调用不会有副作用。
func (c *Channel) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, subs := range c.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	c.subscribers = make(map[string][]chan AgentMessage)
}