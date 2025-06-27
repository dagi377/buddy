package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// ScriptTaskHandler handles script execution tasks
type ScriptTaskHandler struct {
	logger interfaces.Logger
}

// NewScriptTaskHandler creates a new script task handler
func NewScriptTaskHandler(logger interfaces.Logger) *ScriptTaskHandler {
	return &ScriptTaskHandler{
		logger: logger,
	}
}

// Handle executes a script task
func (h *ScriptTaskHandler) Handle(ctx context.Context, task *interfaces.Task) error {
	h.logger.Info("Executing script task", map[string]interface{}{
		"task_id":     task.ID,
		"description": task.Description,
	})

	// For now, we'll simulate script execution by logging the task
	// In a real implementation, you might execute shell commands or custom scripts
	description := task.Description

	// Simple script simulation based on description
	if strings.Contains(strings.ToLower(description), "open") && strings.Contains(strings.ToLower(description), "search engine") {
		h.logger.Info("Simulating opening search engine", map[string]interface{}{
			"task_id": task.ID,
			"action":  "open_search_engine",
		})

		// Simulate opening a browser to a search engine
		// In a real scenario, this might trigger a browser action
		return nil
	}

	// For other script tasks, we can try to execute them as shell commands
	// This is a simplified implementation - in production you'd want more security
	if strings.HasPrefix(description, "run:") {
		command := strings.TrimPrefix(description, "run:")
		command = strings.TrimSpace(command)

		h.logger.Info("Executing shell command", map[string]interface{}{
			"task_id": task.ID,
			"command": command,
		})

		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		output, err := cmd.CombinedOutput()

		if err != nil {
			h.logger.Error("Script execution failed", map[string]interface{}{
				"task_id": task.ID,
				"command": command,
				"error":   err.Error(),
				"output":  string(output),
			})
			return fmt.Errorf("script execution failed: %w", err)
		}

		h.logger.Info("Script executed successfully", map[string]interface{}{
			"task_id": task.ID,
			"command": command,
			"output":  string(output),
		})

		return nil
	}

	// For other script tasks, just log and mark as completed
	h.logger.Info("Script task completed (simulated)", map[string]interface{}{
		"task_id":     task.ID,
		"description": description,
	})

	return nil
}

// CanHandle returns true if this handler can handle the given task type
func (h *ScriptTaskHandler) CanHandle(taskType string) bool {
	return taskType == "script"
}
