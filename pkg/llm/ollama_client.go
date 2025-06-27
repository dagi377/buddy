package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// OllamaClient implements the LLMClient interface for Ollama
type OllamaClient struct {
	baseURL      string
	defaultModel string
	httpClient   *http.Client
	logger       interfaces.Logger
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(baseURL string, logger interfaces.Logger) *OllamaClient {
	return NewOllamaClientWithModel(baseURL, "deepseek-r1:latest", logger)
}

// NewOllamaClientWithModel creates a new Ollama client with a specific default model
func NewOllamaClientWithModel(baseURL, defaultModel string, logger interfaces.Logger) *OllamaClient {
	return &OllamaClient{
		baseURL:      baseURL,
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// Generate sends a request to Ollama and returns the response
func (c *OllamaClient) Generate(ctx context.Context, request interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	c.logger.WithFields(map[string]interface{}{
		"model":  request.Model,
		"prompt": request.Prompt[:min(100, len(request.Prompt))],
	}).Info("Sending request to Ollama")

	// Set default model if not specified
	if request.Model == "" {
		request.Model = c.defaultModel
	}

	// Prepare request body
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	// Parse response
	var llmResp interfaces.LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"model":    llmResp.Model,
		"response": llmResp.Response[:min(100, len(llmResp.Response))],
		"done":     llmResp.Done,
	}).Info("Received response from Ollama")

	return &llmResp, nil
}

// IsHealthy checks if Ollama is running and accessible
func (c *OllamaClient) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		c.logger.WithField("error", err).Error("Failed to create health check request")
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithField("error", err).Error("Ollama health check failed")
		return false
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK
	c.logger.WithField("healthy", healthy).Debug("Ollama health check completed")
	
	return healthy
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
