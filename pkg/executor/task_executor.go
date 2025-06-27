package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// TaskExecutorImpl implements the TaskExecutor interface
type TaskExecutorImpl struct {
	handlers     map[string]interfaces.TaskHandler
	memory       interfaces.MemoryStore
	eventBus     interfaces.EventBus
	logger       interfaces.Logger
	mutex        sync.RWMutex
	runningTasks map[string]context.CancelFunc
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(memory interfaces.MemoryStore, eventBus interfaces.EventBus, logger interfaces.Logger) *TaskExecutorImpl {
	return &TaskExecutorImpl{
		handlers:     make(map[string]interfaces.TaskHandler),
		memory:       memory,
		eventBus:     eventBus,
		logger:       logger,
		runningTasks: make(map[string]context.CancelFunc),
	}
}

// RegisterHandler registers a handler for a specific task type
func (e *TaskExecutorImpl) RegisterHandler(taskType string, handler interfaces.TaskHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.handlers[taskType] = handler
	e.logger.WithField("task_type", taskType).Info("Registered task handler")
}

// ExecuteTask executes a task using the appropriate handler
func (e *TaskExecutorImpl) ExecuteTask(ctx context.Context, task *interfaces.Task) error {
	e.logger.WithFields(map[string]interface{}{
		"task_id":     task.ID,
		"task_type":   task.Type,
		"description": task.Description,
	}).Info("Starting task execution")

	// Check if handler exists
	e.mutex.RLock()
	handler, exists := e.handlers[task.Type]
	e.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for task type: %s", task.Type)
	}

	// Update task status to running
	task.Status = interfaces.TaskStatusRunning
	task.UpdatedAt = time.Now()

	if err := e.memory.Store(ctx, "task:"+task.ID, task); err != nil {
		e.logger.WithField("error", err).Warn("Failed to store task status")
	}

	// Publish task started event
	e.eventBus.Publish(ctx, "task.started", map[string]interface{}{
		"task_id": task.ID,
		"type":    task.Type,
	})

	// Create cancellable context for the task
	taskCtx, cancel := context.WithCancel(ctx)

	e.mutex.Lock()
	e.runningTasks[task.ID] = cancel
	e.mutex.Unlock()

	// Execute task in goroutine
	go func() {
		defer func() {
			e.mutex.Lock()
			delete(e.runningTasks, task.ID)
			e.mutex.Unlock()
		}()

		// Execute the task
		err := handler.Handle(taskCtx, task)

		// Update task status based on result
		if err != nil {
			task.Status = interfaces.TaskStatusFailed
			task.Error = err.Error()
			e.logger.WithFields(map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			}).Error("Task execution failed")

			// Publish task failed event
			e.eventBus.Publish(ctx, "task.failed", map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			})
		} else {
			task.Status = interfaces.TaskStatusCompleted
			e.logger.WithField("task_id", task.ID).Info("Task execution completed")

			// Publish task completed event
			e.eventBus.Publish(ctx, "task.completed", map[string]interface{}{
				"task_id": task.ID,
				"result":  task.Result,
			})
		}

		task.UpdatedAt = time.Now()

		// Store final task state
		if err := e.memory.Store(ctx, "task:"+task.ID, task); err != nil {
			e.logger.WithField("error", err).Warn("Failed to store final task status")
		}
	}()

	return nil
}

// GetTaskStatus returns the current status of a task
func (e *TaskExecutorImpl) GetTaskStatus(ctx context.Context, taskID string) (interfaces.TaskStatus, error) {
	data, err := e.memory.Retrieve(ctx, "task:"+taskID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve task: %w", err)
	}

	task, ok := data.(*interfaces.Task)
	if !ok {
		return "", fmt.Errorf("invalid task data in memory")
	}

	return task.Status, nil
}

// CancelTask cancels a running task
func (e *TaskExecutorImpl) CancelTask(ctx context.Context, taskID string) error {
	e.mutex.Lock()
	cancel, exists := e.runningTasks[taskID]
	e.mutex.Unlock()

	if !exists {
		return fmt.Errorf("task not running: %s", taskID)
	}

	// Cancel the task context
	cancel()

	// Update task status
	data, err := e.memory.Retrieve(ctx, "task:"+taskID)
	if err != nil {
		return fmt.Errorf("failed to retrieve task: %w", err)
	}

	task, ok := data.(*interfaces.Task)
	if !ok {
		return fmt.Errorf("invalid task data in memory")
	}

	task.Status = interfaces.TaskStatusCancelled
	task.UpdatedAt = time.Now()

	if err := e.memory.Store(ctx, "task:"+taskID, task); err != nil {
		e.logger.WithField("error", err).Warn("Failed to store cancelled task status")
	}

	// Publish task cancelled event
	e.eventBus.Publish(ctx, "task.cancelled", map[string]interface{}{
		"task_id": taskID,
	})

	e.logger.WithField("task_id", taskID).Info("Task cancelled")

	return nil
}

// GetRunningTasks returns a list of currently running task IDs
func (e *TaskExecutorImpl) GetRunningTasks() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var taskIDs []string
	for taskID := range e.runningTasks {
		taskIDs = append(taskIDs, taskID)
	}

	return taskIDs
}
