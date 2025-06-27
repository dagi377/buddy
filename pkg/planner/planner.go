package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ai-agent-framework/pkg/interfaces"
)

// TaskPlanner implements the Planner interface
type TaskPlanner struct {
	llmClient interfaces.LLMClient
	memory    interfaces.MemoryStore
	logger    interfaces.Logger
}

// NewTaskPlanner creates a new task planner
func NewTaskPlanner(llmClient interfaces.LLMClient, memory interfaces.MemoryStore, logger interfaces.Logger) *TaskPlanner {
	return &TaskPlanner{
		llmClient: llmClient,
		memory:    memory,
		logger:    logger,
	}
}

// CreatePlan breaks down a goal into executable tasks
func (p *TaskPlanner) CreatePlan(ctx context.Context, goal string) (*interfaces.Plan, error) {
	p.logger.WithField("goal", goal).Info("Creating plan")

	// Generate plan using LLM
	prompt := p.buildPlanningPrompt(goal)
	
	llmReq := interfaces.LLMRequest{
		Model:  "llama3",
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2000,
		},
	}

	resp, err := p.llmClient.Generate(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Parse the LLM response into tasks
	tasks, err := p.parsePlanResponse(resp.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan response: %w", err)
	}

	// Create plan object
	plan := &interfaces.Plan{
		ID:        uuid.New().String(),
		Goal:      goal,
		Tasks:     tasks,
		Status:    interfaces.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store plan in memory
	if err := p.memory.Store(ctx, "plan:"+plan.ID, plan); err != nil {
		p.logger.WithField("error", err).Warn("Failed to store plan in memory")
	}

	p.logger.WithFields(map[string]interface{}{
		"plan_id":    plan.ID,
		"task_count": len(tasks),
	}).Info("Plan created successfully")

	return plan, nil
}

// UpdatePlan modifies an existing plan based on feedback
func (p *TaskPlanner) UpdatePlan(ctx context.Context, planID string, feedback string) (*interfaces.Plan, error) {
	p.logger.WithFields(map[string]interface{}{
		"plan_id":  planID,
		"feedback": feedback,
	}).Info("Updating plan")

	// Retrieve existing plan
	plan, err := p.GetPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve plan: %w", err)
	}

	// Generate updated plan using LLM
	prompt := p.buildUpdatePrompt(plan, feedback)
	
	llmReq := interfaces.LLMRequest{
		Model:  "llama3",
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2000,
		},
	}

	resp, err := p.llmClient.Generate(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate updated plan: %w", err)
	}

	// Parse the updated tasks
	updatedTasks, err := p.parsePlanResponse(resp.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated plan response: %w", err)
	}

	// Update plan
	plan.Tasks = updatedTasks
	plan.UpdatedAt = time.Now()

	// Store updated plan
	if err := p.memory.Store(ctx, "plan:"+plan.ID, plan); err != nil {
		p.logger.WithField("error", err).Warn("Failed to store updated plan in memory")
	}

	p.logger.WithField("plan_id", planID).Info("Plan updated successfully")

	return plan, nil
}

// GetPlan retrieves a plan by ID
func (p *TaskPlanner) GetPlan(ctx context.Context, planID string) (*interfaces.Plan, error) {
	data, err := p.memory.Retrieve(ctx, "plan:"+planID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve plan: %w", err)
	}

	plan, ok := data.(*interfaces.Plan)
	if !ok {
		return nil, fmt.Errorf("invalid plan data in memory")
	}

	return plan, nil
}

// buildPlanningPrompt creates a prompt for the LLM to generate a plan
func (p *TaskPlanner) buildPlanningPrompt(goal string) string {
	return fmt.Sprintf(`You are an AI task planner. Break down the following goal into specific, executable tasks.

Goal: %s

Please provide a JSON response with the following structure:
{
  "tasks": [
    {
      "type": "browser|script|api|analysis",
      "description": "Clear description of what to do",
      "parameters": {
        "key": "value"
      },
      "dependencies": ["task_id_1", "task_id_2"]
    }
  ]
}

Task types:
- browser: Web browser automation (navigation, clicking, form filling)
- script: Execute a script or command
- api: Make API calls
- analysis: Analyze data or content

Make sure tasks are:
1. Specific and actionable
2. Properly ordered with dependencies
3. Include all necessary parameters
4. Realistic and achievable

Response:`, goal)
}

// buildUpdatePrompt creates a prompt for updating an existing plan
func (p *TaskPlanner) buildUpdatePrompt(plan *interfaces.Plan, feedback string) string {
	planJSON, _ := json.MarshalIndent(plan.Tasks, "", "  ")
	
	return fmt.Sprintf(`You are an AI task planner. Update the following plan based on the feedback provided.

Original Goal: %s
Current Plan: %s

Feedback: %s

Please provide an updated JSON response with the same structure as the original plan.
Consider the feedback and modify, add, or remove tasks as necessary.

Response:`, plan.Goal, string(planJSON), feedback)
}

// parsePlanResponse parses the LLM response into tasks
func (p *TaskPlanner) parsePlanResponse(response string) ([]interfaces.Task, error) {
	// Extract JSON from response (LLM might include extra text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1
	
	if jsonStart == -1 || jsonEnd == 0 {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[jsonStart:jsonEnd]

	// Parse JSON
	var planData struct {
		Tasks []struct {
			Type         string                 `json:"type"`
			Description  string                 `json:"description"`
			Parameters   map[string]interface{} `json:"parameters"`
			Dependencies []string               `json:"dependencies"`
		} `json:"tasks"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &planData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to Task objects
	tasks := make([]interfaces.Task, len(planData.Tasks))
	now := time.Now()

	for i, taskData := range planData.Tasks {
		tasks[i] = interfaces.Task{
			ID:           uuid.New().String(),
			Type:         taskData.Type,
			Description:  taskData.Description,
			Parameters:   taskData.Parameters,
			Status:       interfaces.TaskStatusPending,
			Dependencies: taskData.Dependencies,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	return tasks, nil
}
