package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	// Parse task parameters or infer from description
	actionType, ok := task.Parameters["action"].(string)
	if !ok {
		// Try to infer action from task description
		actionType = h.inferActionFromDescription(task.Description)
		if actionType == "" {
			return fmt.Errorf("could not determine browser action from description: %s", task.Description)
		}

		// Set inferred parameters
		if task.Parameters == nil {
			task.Parameters = make(map[string]interface{})
		}
		task.Parameters["action"] = actionType
		h.setParametersFromDescription(task)
	}

	// Special handling for search-related tasks that need navigation first
	if actionType == "type" && h.needsNavigation(task.Description) {
		// Ensure we navigate to Google first
		err := h.ensureGoogleNavigation(ctx)
		if err != nil {
			return fmt.Errorf("failed to navigate to Google before search: %w", err)
		}
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

	// Save the task result
	if err := h.saveTaskResult(ctx, task, task.Result); err != nil {
		h.logger.WithField("error", err).Warn("Failed to save task result")
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

	// Save the task result
	if err := h.saveTaskResult(ctx, task, task.Result); err != nil {
		h.logger.WithField("error", err).Warn("Failed to save task result")
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

	// Save the task result
	if err := h.saveTaskResult(ctx, task, task.Result); err != nil {
		h.logger.WithField("error", err).Warn("Failed to save task result")
	}

	// If this was a search action, submit the search by pressing Enter
	if h.isSearchAction(task.Description) {
		h.logger.Info("Submitting search by pressing Enter")

		// Press Enter to submit the search
		enterAction := interfaces.BrowserAction{
			Type:     "type",
			Selector: selector,
			Value:    "\n", // Press Enter
		}

		_, err := h.browserAgent.ExecuteAction(ctx, enterAction)
		if err != nil {
			h.logger.WithField("error", err).Warn("Failed to submit search with Enter")
		} else {
			h.logger.Info("Search submitted successfully via Enter key")
		}

		// Wait a moment for the search results to load
		time.Sleep(3 * time.Second)
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

	// Save the task result
	if err := h.saveTaskResult(ctx, task, task.Result); err != nil {
		h.logger.WithField("error", err).Warn("Failed to save task result")
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

// inferActionFromDescription attempts to determine the browser action from task description
func (h *BrowserTaskHandler) inferActionFromDescription(description string) string {
	desc := strings.ToLower(description)

	// Navigation patterns
	if strings.Contains(desc, "navigate") || strings.Contains(desc, "go to") || strings.Contains(desc, "visit") || strings.Contains(desc, "open") || strings.Contains(desc, "launch") || strings.Contains(desc, "start") {
		return "navigate"
	}

	// Search patterns
	if strings.Contains(desc, "search for") || strings.Contains(desc, "find") {
		return "type" // Assume search involves typing
	}

	// Click patterns
	if strings.Contains(desc, "click") || strings.Contains(desc, "press") || strings.Contains(desc, "select") {
		return "click"
	}

	// Type patterns
	if strings.Contains(desc, "type") || strings.Contains(desc, "enter") || strings.Contains(desc, "input") {
		return "type"
	}

	// Extract patterns
	if strings.Contains(desc, "extract") || strings.Contains(desc, "get") || strings.Contains(desc, "scrape") {
		return "extract"
	}

	// Screenshot patterns
	if strings.Contains(desc, "screenshot") || strings.Contains(desc, "capture") {
		return "screenshot"
	}

	// Wait patterns
	if strings.Contains(desc, "wait") || strings.Contains(desc, "pause") {
		return "wait"
	}

	// Default to navigate for most browser tasks
	if strings.Contains(desc, "page") || strings.Contains(desc, "site") || strings.Contains(desc, "website") || strings.Contains(desc, "browser") || strings.Contains(desc, "chrome") {
		return "navigate"
	}

	return ""
}

// setParametersFromDescription sets task parameters based on the description
func (h *BrowserTaskHandler) setParametersFromDescription(task *interfaces.Task) {
	desc := strings.ToLower(task.Description)
	action := task.Parameters["action"].(string)

	switch action {
	case "navigate":
		// Try to extract URL from description
		if strings.Contains(desc, "google") {
			task.Parameters["url"] = "https://www.google.com"
		} else if strings.Contains(desc, "search") {
			task.Parameters["url"] = "https://www.google.com"
		} else {
			// Default URL if none specified
			task.Parameters["url"] = "https://www.google.com"
		}

	case "type":
		// Extract search query from description
		var query string

		if strings.Contains(desc, "search for") {
			// Extract text after "search for"
			parts := strings.Split(desc, "search for")
			if len(parts) > 1 {
				query = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(desc, "find") {
			// Extract text after "find"
			parts := strings.Split(desc, "find")
			if len(parts) > 1 {
				query = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(desc, "enter") {
			// Extract text between quotes for "enter 'text'" patterns
			start := strings.Index(desc, "'")
			if start != -1 {
				end := strings.Index(desc[start+1:], "'")
				if end != -1 {
					query = desc[start+1 : start+1+end]
				}
			}
			// Fallback: extract text after "enter"
			if query == "" {
				parts := strings.Split(desc, "enter")
				if len(parts) > 1 {
					// Remove common suffix like "into the search bar"
					text := strings.TrimSpace(parts[1])
					text = strings.Replace(text, "into the search bar", "", -1)
					text = strings.Replace(text, "in the search box", "", -1)
					text = strings.Replace(text, "into search", "", -1)
					query = strings.TrimSpace(text)
				}
			}
		} else if strings.Contains(desc, "type") {
			// Extract text after "type"
			parts := strings.Split(desc, "type")
			if len(parts) > 1 {
				// Look for quoted text first
				text := parts[1]
				start := strings.Index(text, "'")
				if start != -1 {
					end := strings.Index(text[start+1:], "'")
					if end != -1 {
						query = text[start+1 : start+1+end]
					}
				}
				// Fallback to everything after "type"
				if query == "" {
					query = strings.TrimSpace(text)
				}
			}
		}

		// Set the extracted query
		if query != "" {
			task.Parameters["text"] = query
		} else {
			// Default search query if we can't extract one
			task.Parameters["text"] = "cafes near leaside"
		}

		// Use more robust selector for Google search input
		task.Parameters["selector"] = "textarea[name='q'], input[name='q'], input[aria-label*='Search'], textarea[aria-label*='Search'], #APjFqb, .gLFyf, input[title='Search'], textarea[title='Search']"

	case "click":
		// Set default selector for common elements
		if strings.Contains(desc, "search") {
			task.Parameters["selector"] = "input[type='submit'], button[type='submit'], .search-button, input[value*='Search'], button[aria-label*='Search']"
		} else {
			task.Parameters["selector"] = "button, a, input[type='submit']"
		}

	case "extract":
		// Set default extraction parameters
		task.Parameters["selector"] = "h1, h2, h3, .title, .result"
		task.Parameters["extract_type"] = "text"

	case "screenshot":
		// Set default screenshot parameters
		task.Parameters["filename"] = fmt.Sprintf("screenshot_%s.png", task.ID)

	case "wait":
		// Set default wait parameters
		task.Parameters["timeout"] = 5000.0 // 5 seconds
		task.Parameters["selector"] = "body"
	}
}

// needsNavigation checks if a task description suggests it needs navigation first
func (h *BrowserTaskHandler) needsNavigation(description string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "search") || strings.Contains(desc, "enter") || strings.Contains(desc, "type")
}

// ensureGoogleNavigation makes sure we're on Google before performing search actions
func (h *BrowserTaskHandler) ensureGoogleNavigation(ctx context.Context) error {
	// Try to get current page content to see if we're already on Google
	content, err := h.browserAgent.GetPageContent(ctx)
	if err == nil && strings.Contains(strings.ToLower(content), "google") {
		h.logger.Info("Already on Google page, skipping navigation")
		return nil
	}

	h.logger.Info("Navigating to Google for search task")
	return h.browserAgent.Navigate(ctx, "https://www.google.com")
}

// isSearchAction checks if a task description suggests it's a search action
func (h *BrowserTaskHandler) isSearchAction(description string) bool {
	desc := strings.ToLower(description)
	return strings.Contains(desc, "search") || strings.Contains(desc, "enter") || strings.Contains(desc, "type")
}

// saveTaskResult saves the task result to a timestamped folder in the results directory
func (h *BrowserTaskHandler) saveTaskResult(ctx context.Context, task *interfaces.Task, result interface{}) error {
	// Create results directory with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	resultsDir := filepath.Join("results", timestamp)

	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		h.logger.WithField("error", err).Warn("Failed to create results directory")
		return err
	}

	// Take a screenshot of the final state
	screenshot, err := h.browserAgent.Screenshot(ctx)
	if err != nil {
		h.logger.WithField("error", err).Warn("Failed to take final screenshot")
	} else {
		screenshotPath := filepath.Join(resultsDir, "final_screenshot.png")
		if err := os.WriteFile(screenshotPath, screenshot, 0644); err != nil {
			h.logger.WithField("error", err).Warn("Failed to save final screenshot")
		} else {
			h.logger.WithField("path", screenshotPath).Info("Final screenshot saved")
		}
	}

	// Get page content
	pageContent, err := h.browserAgent.GetPageContent(ctx)
	if err != nil {
		h.logger.WithField("error", err).Warn("Failed to get page content")
	} else {
		contentPath := filepath.Join(resultsDir, "page_content.html")
		if err := os.WriteFile(contentPath, []byte(pageContent), 0644); err != nil {
			h.logger.WithField("error", err).Warn("Failed to save page content")
		} else {
			h.logger.WithField("path", contentPath).Info("Page content saved")
		}
	}

	// Create result metadata
	resultData := map[string]interface{}{
		"task_id":        task.ID,
		"task_type":      task.Type,
		"description":    task.Description,
		"status":         task.Status,
		"result":         result,
		"parameters":     task.Parameters,
		"timestamp":      time.Now().Format(time.RFC3339),
		"execution_time": time.Since(task.CreatedAt).String(),
	}

	// Save result metadata as JSON
	resultPath := filepath.Join(resultsDir, "task_result.json")
	resultJSON, err := json.MarshalIndent(resultData, "", "  ")
	if err != nil {
		h.logger.WithField("error", err).Warn("Failed to marshal result data")
		return err
	}

	if err := os.WriteFile(resultPath, resultJSON, 0644); err != nil {
		h.logger.WithField("error", err).Warn("Failed to save result metadata")
		return err
	}

	h.logger.WithFields(map[string]interface{}{
		"task_id":     task.ID,
		"results_dir": resultsDir,
		"result_path": resultPath,
	}).Info("Task result saved successfully")

	return nil
}
