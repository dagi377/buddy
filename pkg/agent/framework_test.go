package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFramework(t *testing.T) {
	config := &Config{
		OllamaURL:       "http://localhost:11434",
		LogLevel:        "info",
		BrowserHeadless: true,
		MemoryType:      "memory",
	}

	framework, err := NewFramework(config)
	require.NoError(t, err)
	assert.NotNil(t, framework)
	assert.Equal(t, config, framework.config)
	assert.False(t, framework.isRunning)
}

func TestFrameworkStatus(t *testing.T) {
	config := &Config{
		OllamaURL:       "http://localhost:11434",
		LogLevel:        "info",
		BrowserHeadless: true,
		MemoryType:      "memory",
	}

	framework, err := NewFramework(config)
	require.NoError(t, err)

	ctx := context.Background()
	status, err := framework.GetStatus(ctx)
	require.NoError(t, err)

	assert.Contains(t, status, "running")
	assert.Contains(t, status, "llm_healthy")
	assert.Contains(t, status, "timestamp")
	assert.False(t, status["running"].(bool))
}

func TestFrameworkLifecycle(t *testing.T) {
	config := &Config{
		OllamaURL:       "http://localhost:11434",
		LogLevel:        "debug",
		BrowserHeadless: true,
		MemoryType:      "memory",
	}

	framework, err := NewFramework(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test stop without start (should not error)
	err = framework.Stop(ctx)
	assert.NoError(t, err)

	// Note: Start test would require Ollama to be running
	// In a real test environment, you'd mock the LLM client
}
