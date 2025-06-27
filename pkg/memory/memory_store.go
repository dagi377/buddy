package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// InMemoryStore implements the MemoryStore interface using in-memory storage
type InMemoryStore struct {
	data   map[string]interface{}
	mutex  sync.RWMutex
	logger interfaces.Logger
}

// NewInMemoryStore creates a new in-memory store
func NewInMemoryStore(logger interfaces.Logger) *InMemoryStore {
	return &InMemoryStore{
		data:   make(map[string]interface{}),
		logger: logger,
	}
}

// Store saves a value with the given key
func (m *InMemoryStore) Store(ctx context.Context, key string, value interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data[key] = value
	
	m.logger.WithFields(map[string]interface{}{
		"key":   key,
		"type":  fmt.Sprintf("%T", value),
	}).Debug("Stored value in memory")

	return nil
}

// Retrieve gets a value by key
func (m *InMemoryStore) Retrieve(ctx context.Context, key string) (interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	m.logger.WithFields(map[string]interface{}{
		"key":   key,
		"type":  fmt.Sprintf("%T", value),
	}).Debug("Retrieved value from memory")

	return value, nil
}

// Delete removes a value by key
func (m *InMemoryStore) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.data[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(m.data, key)
	
	m.logger.WithField("key", key).Debug("Deleted value from memory")

	return nil
}

// List returns all keys with the given prefix
func (m *InMemoryStore) List(ctx context.Context, prefix string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var keys []string
	for key := range m.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	m.logger.WithFields(map[string]interface{}{
		"prefix":    prefix,
		"key_count": len(keys),
	}).Debug("Listed keys from memory")

	return keys, nil
}

// Clear removes all stored values
func (m *InMemoryStore) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	count := len(m.data)
	m.data = make(map[string]interface{})
	
	m.logger.WithField("cleared_count", count).Info("Cleared all values from memory")

	return nil
}

// GetStats returns statistics about the memory store
func (m *InMemoryStore) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"total_keys": len(m.data),
		"type":       "in-memory",
	}
}
