package langgraph

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ai-agent-framework/pkg/interfaces"
)

// WorkflowState represents the state of a workflow
type WorkflowState struct {
	ID           string                 `json:"id"`
	CurrentState string                 `json:"current_state"`
	States       []string               `json:"states"`
	Transitions  map[string]map[string]string `json:"transitions"` // from -> event -> to
	Data         map[string]interface{} `json:"data"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// LangGraphEngineImpl implements the LangGraphEngine interface
type LangGraphEngineImpl struct {
	workflows   map[string]*WorkflowState
	subscribers map[string][]chan interfaces.StateTransition
	memory      interfaces.MemoryStore
	logger      interfaces.Logger
	mutex       sync.RWMutex
}

// NewLangGraphEngine creates a new LangGraph engine
func NewLangGraphEngine(memory interfaces.MemoryStore, logger interfaces.Logger) *LangGraphEngineImpl {
	return &LangGraphEngineImpl{
		workflows:   make(map[string]*WorkflowState),
		subscribers: make(map[string][]chan interfaces.StateTransition),
		memory:      memory,
		logger:      logger,
	}
}

// CreateWorkflow creates a new workflow with the specified states
func (e *LangGraphEngineImpl) CreateWorkflow(ctx context.Context, workflowID string, states []string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if len(states) == 0 {
		return fmt.Errorf("workflow must have at least one state")
	}

	workflow := &WorkflowState{
		ID:           workflowID,
		CurrentState: states[0], // Start with the first state
		States:       states,
		Transitions:  make(map[string]map[string]string),
		Data:         make(map[string]interface{}),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Initialize transitions map for each state
	for _, state := range states {
		workflow.Transitions[state] = make(map[string]string)
	}

	e.workflows[workflowID] = workflow

	// Store in memory
	if err := e.memory.Store(ctx, "workflow:"+workflowID, workflow); err != nil {
		e.logger.WithField("error", err).Warn("Failed to store workflow in memory")
	}

	e.logger.WithFields(map[string]interface{}{
		"workflow_id":   workflowID,
		"initial_state": workflow.CurrentState,
		"state_count":   len(states),
	}).Info("Created new workflow")

	return nil
}

// AddTransition adds a state transition rule
func (e *LangGraphEngineImpl) AddTransition(ctx context.Context, workflowID string, from, to, event string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	workflow, exists := e.workflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	// Validate states exist
	if !e.stateExists(workflow, from) {
		return fmt.Errorf("source state does not exist: %s", from)
	}
	if !e.stateExists(workflow, to) {
		return fmt.Errorf("target state does not exist: %s", to)
	}

	// Add transition
	workflow.Transitions[from][event] = to
	workflow.UpdatedAt = time.Now()

	// Update in memory
	if err := e.memory.Store(ctx, "workflow:"+workflowID, workflow); err != nil {
		e.logger.WithField("error", err).Warn("Failed to update workflow in memory")
	}

	e.logger.WithFields(map[string]interface{}{
		"workflow_id": workflowID,
		"from":        from,
		"to":          to,
		"event":       event,
	}).Info("Added workflow transition")

	return nil
}

// TriggerEvent triggers a state transition based on an event
func (e *LangGraphEngineImpl) TriggerEvent(ctx context.Context, workflowID string, event string, data map[string]interface{}) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	workflow, exists := e.workflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	currentState := workflow.CurrentState
	
	// Check if transition exists for current state and event
	nextState, exists := workflow.Transitions[currentState][event]
	if !exists {
		return fmt.Errorf("no transition defined for state '%s' with event '%s'", currentState, event)
	}

	// Create state transition
	transition := interfaces.StateTransition{
		From:      currentState,
		To:        nextState,
		Event:     event,
		TaskID:    workflowID,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Update workflow state
	workflow.CurrentState = nextState
	workflow.UpdatedAt = time.Now()

	// Merge data if provided
	if data != nil {
		for key, value := range data {
			workflow.Data[key] = value
		}
	}

	// Update in memory
	if err := e.memory.Store(ctx, "workflow:"+workflowID, workflow); err != nil {
		e.logger.WithField("error", err).Warn("Failed to update workflow state in memory")
	}

	// Notify subscribers
	e.notifySubscribers(workflowID, transition)

	e.logger.WithFields(map[string]interface{}{
		"workflow_id": workflowID,
		"from":        currentState,
		"to":          nextState,
		"event":       event,
	}).Info("State transition triggered")

	return nil
}

// GetCurrentState returns the current state of a workflow
func (e *LangGraphEngineImpl) GetCurrentState(ctx context.Context, workflowID string) (string, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	workflow, exists := e.workflows[workflowID]
	if !exists {
		return "", fmt.Errorf("workflow not found: %s", workflowID)
	}

	return workflow.CurrentState, nil
}

// Subscribe creates a channel to receive state transition notifications
func (e *LangGraphEngineImpl) Subscribe(ctx context.Context, workflowID string) (<-chan interfaces.StateTransition, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Check if workflow exists
	if _, exists := e.workflows[workflowID]; !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	// Create channel for notifications
	ch := make(chan interfaces.StateTransition, 10) // Buffered channel

	// Add to subscribers
	if e.subscribers[workflowID] == nil {
		e.subscribers[workflowID] = make([]chan interfaces.StateTransition, 0)
	}
	e.subscribers[workflowID] = append(e.subscribers[workflowID], ch)

	e.logger.WithField("workflow_id", workflowID).Info("New subscriber added")

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		e.unsubscribe(workflowID, ch)
	}()

	return ch, nil
}

// GetWorkflow returns the complete workflow state
func (e *LangGraphEngineImpl) GetWorkflow(ctx context.Context, workflowID string) (*WorkflowState, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	workflow, exists := e.workflows[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	// Return a copy to prevent external modification
	workflowCopy := *workflow
	return &workflowCopy, nil
}

// Helper methods

func (e *LangGraphEngineImpl) stateExists(workflow *WorkflowState, state string) bool {
	for _, s := range workflow.States {
		if s == state {
			return true
		}
	}
	return false
}

func (e *LangGraphEngineImpl) notifySubscribers(workflowID string, transition interfaces.StateTransition) {
	subscribers, exists := e.subscribers[workflowID]
	if !exists {
		return
	}

	// Send notification to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- transition:
		default:
			// Channel is full, skip this subscriber
			e.logger.WithField("workflow_id", workflowID).Warn("Subscriber channel full, skipping notification")
		}
	}
}

func (e *LangGraphEngineImpl) unsubscribe(workflowID string, ch chan interfaces.StateTransition) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	subscribers, exists := e.subscribers[workflowID]
	if !exists {
		return
	}

	// Remove channel from subscribers
	for i, subscriber := range subscribers {
		if subscriber == ch {
			e.subscribers[workflowID] = append(subscribers[:i], subscribers[i+1:]...)
			close(ch)
			break
		}
	}

	e.logger.WithField("workflow_id", workflowID).Info("Subscriber removed")
}
