package orchestrator

import (
	"context"
	"sync"
)

// AgentMessage represents a message sent between agents.
type AgentMessage struct {
	// From is the sender agent/task ID.
	From string
	// To is the recipient agent/task ID (empty for broadcast).
	To string
	// Content is the message payload.
	Content string
	// Data carries arbitrary structured data (optional).
	Data interface{}
}

// Channel provides message passing between agents.
type Channel struct {
	mu          sync.RWMutex
	subscribers map[string][]chan AgentMessage
	buffer      int
}

// NewChannel creates a new Channel with the given buffer size.
func NewChannel(buffer int) *Channel {
	return &Channel{
		subscribers: make(map[string][]chan AgentMessage),
		buffer:      buffer,
	}
}

// Subscribe registers a subscriber for the given topic/agent ID.
func (c *Channel) Subscribe(agentID string) <-chan AgentMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan AgentMessage, c.buffer)
	c.subscribers[agentID] = append(c.subscribers[agentID], ch)
	return ch
}

// Send sends a message to a specific agent.
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

// Broadcast sends a message to all subscribers.
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

// Receive blocks until a message is received on the given channel.
func (c *Channel) Receive(ctx context.Context, sub <-chan AgentMessage) (AgentMessage, error) {
	select {
	case msg := <-sub:
		return msg, nil
	case <-ctx.Done():
		return AgentMessage{}, ctx.Err()
	}
}

// Close closes all subscriber channels.
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