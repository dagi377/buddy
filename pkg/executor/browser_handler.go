package executor

import (
	"context"
	"fmt"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// BrowserTaskHandler handles browser-related tasks
type BrowserTaskHandler struct {
	browserAgent interfaces.BrowserAgent
	logger       interfaces.Logger
}

// NewBrowserTaskHandler creates a new browser task handler
func NewBrowserTaskHandler(browserAgent interfaces.BrowserAgent, logger interfaces.Logger) *BrowserTaskHandler {
	return &BrowserTaskHandler{
		browserAgent: browserAgent,
		logger:       logger,
	}
}

// Handle executes a browser task
func (h *BrowserTaskHandler) Handle(ctx context.Context, task *interfaces.Task) error {
	h.logger.WithFields(map[string]interface{}{
		"task_id":     task.ID,
		"description": task.Description,
	}).Info("Handling browser task")

	// Parse task parameters
	actionType, ok := task.Parameters["action"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'action' parameter")
	}

	switch actionType {
	case "navigate":
		return h.handleNavigate(ctx, task)
	case "click":
		return h.handleClick(ctx, task)
	case "type":
		return h.handleType(ctx, task)
	case "extract":
		return h.handleExtract(ctx, task)
	case "screenshot":
		return h.handleScreenshot(ctx, task)
	case "wait":
		return h.handleWait(ctx, task)
	default:
		return fmt.Errorf("unsupported browser action: %s", actionType)
	}
}

// CanHandle checks if this handler can handle the given task type
func (h *BrowserTaskHandler) CanHandle(taskType string) bool {
	return taskType == "browser"
}

func (h *BrowserTaskHandler) handleNavigate(ctx context.Context, task *interfaces.Task) error {
	url, ok := task.Parameters["url"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'url' parameter for navigate action")
	}

	err := h.browserAgent.Navigate(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	task.Result = map[string]interface{}{
		"action": "navigate",
		"url":    url,
		"status": "success",
	}

	return nil
}

func (h *BrowserTaskHandler) handleClick(ctx context.Context, task *interfaces.Task) error {
	selector, ok := task.Parameters["selector"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'selector' parameter for click action")
	}

	action := interfaces.BrowserAction{
		Type:     "click",
		Selector: selector,
	}

	result, err := h.browserAgent.ExecuteAction(ctx, action)
	if err != nil {
		return fmt.Errorf("failed to click element %s: %w", selector, err)
	}

	task.Result = map[string]interface{}{
		"action":   "click",
		"selector": selector,
		"result":   result,
	}

	return nil
}

func (h *BrowserTaskHandler) handleType(ctx context.Context, task *interfaces.Task) error {
	selector, ok := task.Parameters["selector"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'selector' parameter for type action")
	}

	text, ok := task.Parameters["text"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'text' parameter for type action")
	}

	action := interfaces.BrowserAction{
		Type:     "type",
		Selector: selector,
		Value:    text,
	}

	result, err := h.browserAgent.ExecuteAction(ctx, action)
	if err != nil {
		return fmt.Errorf("failed to type in element %s: %w", selector, err)
	}

	task.Result = map[string]interface{}{
		"action":   "type",
		"selector": selector,
		"text":     text,
		"result":   result,
	}

	return nil
}

func (h *BrowserTaskHandler) handleExtract(ctx context.Context, task *interfaces.Task) error {
	selector, ok := task.Parameters["selector"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'selector' parameter for extract action")
	}

	extractType, ok := task.Parameters["extract_type"].(string)
	if !ok {
		extractType = "text" // default to text extraction
	}

	var action interfaces.BrowserAction
	switch extractType {
	case "text":
		action = interfaces.BrowserAction{
			Type:     "extract_text",
			Selector: selector,
		}
	case "attribute":
		attribute, ok := task.Parameters["attribute"].(string)
		if !ok {
			return fmt.Errorf("missing 'attribute' parameter for attribute extraction")
		}
		action = interfaces.BrowserAction{
			Type:     "extract_attribute",
			Selector: selector,
			Parameters: map[string]interface{}{
				"attribute": attribute,
			},
		}
	default:
		return fmt.Errorf("unsupported extract type: %s", extractType)
	}

	result, err := h.browserAgent.ExecuteAction(ctx, action)
	if err != nil {
		return fmt.Errorf("failed to extract from element %s: %w", selector, err)
	}

	task.Result = map[string]interface{}{
		"action":       "extract",
		"selector":     selector,
		"extract_type": extractType,
		"result":       result,
	}

	return nil
}

func (h *BrowserTaskHandler) handleScreenshot(ctx context.Context, task *interfaces.Task) error {
	screenshot, err := h.browserAgent.Screenshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to take screenshot: %w", err)
	}

	task.Result = map[string]interface{}{
		"action":     "screenshot",
		"size_bytes": len(screenshot),
		"format":     "png",
	}

	// Note: In a real implementation, you might want to save the screenshot
	// to a file or store it in a way that can be retrieved later

	return nil
}

func (h *BrowserTaskHandler) handleWait(ctx context.Context, task *interfaces.Task) error {
	selector, ok := task.Parameters["selector"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'selector' parameter for wait action")
	}

	timeout := 5000.0 // default 5 seconds
	if timeoutParam, ok := task.Parameters["timeout"]; ok {
		if timeoutFloat, ok := timeoutParam.(float64); ok {
			timeout = timeoutFloat
		}
	}

	action := interfaces.BrowserAction{
		Type:     "wait",
		Selector: selector,
		Parameters: map[string]interface{}{
			"timeout": timeout,
		},
	}

	result, err := h.browserAgent.ExecuteAction(ctx, action)
	if err != nil {
		return fmt.Errorf("failed to wait for element %s: %w", selector, err)
	}

	task.Result = map[string]interface{}{
		"action":   "wait",
		"selector": selector,
		"timeout":  timeout,
		"result":   result,
	}

	return nil
}
