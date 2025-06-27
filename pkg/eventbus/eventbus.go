package eventbus

import (
	"context"
	"sync"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// InMemoryEventBus implements the EventBus interface using channels
type InMemoryEventBus struct {
	subscribers map[string][]chan interface{}
	mutex       sync.RWMutex
	logger      interfaces.Logger
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(logger interfaces.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[string][]chan interface{}),
		logger:      logger,
	}
}

// Publish publishes data to all subscribers of a topic
func (e *InMemoryEventBus) Publish(ctx context.Context, topic string, data interface{}) error {
	e.mutex.RLock()
	subscribers, exists := e.subscribers[topic]
	e.mutex.RUnlock()

	if !exists {
		e.logger.WithField("topic", topic).Debug("No subscribers for topic")
		return nil
	}

	e.logger.WithFields(map[string]interface{}{
		"topic":            topic,
		"subscriber_count": len(subscribers),
	}).Debug("Publishing event")

	// Send to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- data:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel is full, skip this subscriber
			e.logger.WithField("topic", topic).Warn("Subscriber channel full, skipping")
		}
	}

	return nil
}

// Subscribe creates a channel to receive events for a topic
func (e *InMemoryEventBus) Subscribe(ctx context.Context, topic string) (<-chan interface{}, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Create buffered channel
	ch := make(chan interface{}, 10)

	// Add to subscribers
	if e.subscribers[topic] == nil {
		e.subscribers[topic] = make([]chan interface{}, 0)
	}
	e.subscribers[topic] = append(e.subscribers[topic], ch)

	e.logger.WithField("topic", topic).Info("New subscriber added")

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		e.Unsubscribe(ctx, topic, ch)
	}()

	return ch, nil
}

// Unsubscribe removes a channel from topic subscribers
func (e *InMemoryEventBus) Unsubscribe(ctx context.Context, topic string, ch <-chan interface{}) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	subscribers, exists := e.subscribers[topic]
	if !exists {
		return nil
	}

	// Find and remove the channel
	for i, subscriber := range subscribers {
		if subscriber == ch {
			e.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
			close(subscriber)
			e.logger.WithField("topic", topic).Info("Subscriber removed")
			break
		}
	}

	// Clean up empty topic
	if len(e.subscribers[topic]) == 0 {
		delete(e.subscribers, topic)
	}

	return nil
}

// GetTopics returns all active topics
func (e *InMemoryEventBus) GetTopics() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	topics := make([]string, 0, len(e.subscribers))
	for topic := range e.subscribers {
		topics = append(topics, topic)
	}

	return topics
}

// GetSubscriberCount returns the number of subscribers for a topic
func (e *InMemoryEventBus) GetSubscriberCount(topic string) int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	if subscribers, exists := e.subscribers[topic]; exists {
		return len(subscribers)
	}
	return 0
}
