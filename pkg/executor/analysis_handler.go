package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// AnalysisTaskHandler handles analysis and data processing tasks
type AnalysisTaskHandler struct {
	logger      interfaces.Logger
	memoryStore interfaces.MemoryStore
}

// NewAnalysisTaskHandler creates a new analysis task handler
func NewAnalysisTaskHandler(logger interfaces.Logger, memoryStore interfaces.MemoryStore) *AnalysisTaskHandler {
	return &AnalysisTaskHandler{
		logger:      logger,
		memoryStore: memoryStore,
	}
}

// Handle executes an analysis task
func (h *AnalysisTaskHandler) Handle(ctx context.Context, task *interfaces.Task) error {
	h.logger.Info("Executing analysis task", map[string]interface{}{
		"task_id":     task.ID,
		"description": task.Description,
	})

	description := task.Description

	// Handle different types of analysis tasks
	if strings.Contains(strings.ToLower(description), "extract") && strings.Contains(strings.ToLower(description), "links") {
		return h.extractLinks(ctx, task)
	}

	if strings.Contains(strings.ToLower(description), "analyze") && strings.Contains(strings.ToLower(description), "content") {
		return h.analyzeContent(ctx, task)
	}

	// Generic analysis task
	h.logger.Info("Performing generic analysis", map[string]interface{}{
		"task_id":     task.ID,
		"description": description,
	})

	// Store analysis result in memory
	analysisResult := map[string]interface{}{
		"task_id":     task.ID,
		"type":        "analysis",
		"description": description,
		"status":      "completed",
		"timestamp":   ctx.Value("timestamp"),
	}

	key := fmt.Sprintf("analysis:%s", task.ID)
	if err := h.memoryStore.Store(ctx, key, analysisResult); err != nil {
		h.logger.Error("Failed to store analysis result", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to store analysis result: %w", err)
	}

	h.logger.Info("Analysis task completed", map[string]interface{}{
		"task_id": task.ID,
		"result":  "stored in memory",
	})

	return nil
}

// extractLinks simulates extracting links from search results
func (h *AnalysisTaskHandler) extractLinks(ctx context.Context, task *interfaces.Task) error {
	h.logger.Info("Extracting links from search results", map[string]interface{}{
		"task_id": task.ID,
	})

	// Simulate extracted links
	links := []string{
		"https://golang.org/",
		"https://go.dev/",
		"https://golang.org/doc/",
		"https://pkg.go.dev/",
		"https://golang.org/doc/tutorial/getting-started",
	}

	// Store links in memory
	key := fmt.Sprintf("extracted_links:%s", task.ID)
	if err := h.memoryStore.Store(ctx, key, links); err != nil {
		h.logger.Error("Failed to store extracted links", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to store extracted links: %w", err)
	}

	h.logger.Info("Links extracted and stored", map[string]interface{}{
		"task_id":    task.ID,
		"link_count": len(links),
		"links":      links,
	})

	return nil
}

// analyzeContent simulates analyzing webpage content
func (h *AnalysisTaskHandler) analyzeContent(ctx context.Context, task *interfaces.Task) error {
	h.logger.Info("Analyzing webpage content", map[string]interface{}{
		"task_id": task.ID,
	})

	// Simulate content analysis
	analysis := map[string]interface{}{
		"title":       "Go Programming Language",
		"description": "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
		"key_points": []string{
			"Statically typed, compiled programming language",
			"Designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson",
			"Syntactically similar to C, but with memory safety, garbage collection, structural typing, and CSP-style concurrency",
			"Fast compilation and execution",
			"Built-in support for concurrent programming",
		},
		"relevance": "high",
		"source":    "official Go website",
	}

	// Store analysis in memory
	key := fmt.Sprintf("content_analysis:%s", task.ID)
	if err := h.memoryStore.Store(ctx, key, analysis); err != nil {
		h.logger.Error("Failed to store content analysis", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to store content analysis: %w", err)
	}

	h.logger.Info("Content analysis completed and stored", map[string]interface{}{
		"task_id": task.ID,
		"title":   analysis["title"],
		"points":  len(analysis["key_points"].([]string)),
	})

	return nil
}

// CanHandle returns true if this handler can handle the given task type
func (h *AnalysisTaskHandler) CanHandle(taskType string) bool {
	return taskType == "analysis"
}
