package interfaces

import (
	"context"
	"time"
)

// Task represents a unit of work in the agent framework
type Task struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Parameters   map[string]interface{} `json:"parameters"`
	Status       TaskStatus             `json:"status"`
	Dependencies []string               `json:"dependencies"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Result       interface{}            `json:"result,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Plan represents a collection of tasks with dependencies
type Plan struct {
	ID        string     `json:"id"`
	Goal      string     `json:"goal"`
	Tasks     []Task     `json:"tasks"`
	Status    TaskStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// LLMRequest represents a request to the local LLM
type LLMRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// LLMResponse represents a response from the local LLM
type LLMResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Context  []int  `json:"context,omitempty"`
}

// BrowserAction represents an action to be performed in the browser
type BrowserAction struct {
	Type       string                 `json:"type"`
	Selector   string                 `json:"selector,omitempty"`
	Value      string                 `json:"value,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// StateTransition represents a state change in the LangGraph engine
type StateTransition struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Event     string                 `json:"event"`
	TaskID    string                 `json:"task_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Planner interface defines the task planning capabilities
type Planner interface {
	CreatePlan(ctx context.Context, goal string) (*Plan, error)
	UpdatePlan(ctx context.Context, planID string, feedback string) (*Plan, error)
	GetPlan(ctx context.Context, planID string) (*Plan, error)
}

// TaskExecutor interface defines task execution capabilities
type TaskExecutor interface {
	ExecuteTask(ctx context.Context, task *Task) error
	GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error)
	CancelTask(ctx context.Context, taskID string) error
	RegisterHandler(taskType string, handler TaskHandler)
}

// TaskHandler interface for specific task type handlers
type TaskHandler interface {
	Handle(ctx context.Context, task *Task) error
	CanHandle(taskType string) bool
}

// BrowserAgent interface defines browser automation capabilities
type BrowserAgent interface {
	Navigate(ctx context.Context, url string) error
	ExecuteAction(ctx context.Context, action BrowserAction) (interface{}, error)
	Screenshot(ctx context.Context) ([]byte, error)
	GetPageContent(ctx context.Context) (string, error)
	Close(ctx context.Context) error
}

// MemoryStore interface defines memory management capabilities
type MemoryStore interface {
	Store(ctx context.Context, key string, value interface{}) error
	Retrieve(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Clear(ctx context.Context) error
}

// LangGraphEngine interface defines state machine capabilities
type LangGraphEngine interface {
	CreateWorkflow(ctx context.Context, workflowID string, states []string) error
	AddTransition(ctx context.Context, workflowID string, from, to, event string) error
	TriggerEvent(ctx context.Context, workflowID string, event string, data map[string]interface{}) error
	GetCurrentState(ctx context.Context, workflowID string) (string, error)
	Subscribe(ctx context.Context, workflowID string) (<-chan StateTransition, error)
}

// LLMClient interface defines local LLM interaction capabilities
type LLMClient interface {
	Generate(ctx context.Context, request LLMRequest) (*LLMResponse, error)
	IsHealthy(ctx context.Context) bool
}

// Logger interface defines logging capabilities
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// EventBus interface defines pub/sub capabilities
type EventBus interface {
	Publish(ctx context.Context, topic string, data interface{}) error
	Subscribe(ctx context.Context, topic string) (<-chan interface{}, error)
	Unsubscribe(ctx context.Context, topic string, ch <-chan interface{}) error
}

// AgentFramework is the main interface that orchestrates all components
type AgentFramework interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	ExecuteGoal(ctx context.Context, goal string) (*Plan, error)
	GetStatus(ctx context.Context) (map[string]interface{}, error)
}
