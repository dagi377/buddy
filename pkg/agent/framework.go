package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-framework/pkg/interfaces"
	"github.com/ai-agent-framework/pkg/browser"
	"github.com/ai-agent-framework/pkg/eventbus"
	"github.com/ai-agent-framework/pkg/executor"
	"github.com/ai-agent-framework/pkg/langgraph"
	"github.com/ai-agent-framework/pkg/llm"
	"github.com/ai-agent-framework/pkg/logger"
	"github.com/ai-agent-framework/pkg/memory"
	"github.com/ai-agent-framework/pkg/planner"
)

// Framework implements the AgentFramework interface
type Framework struct {
	planner      interfaces.Planner
	executor     interfaces.TaskExecutor
	browserAgent interfaces.BrowserAgent
	memory       interfaces.MemoryStore
	langGraph    interfaces.LangGraphEngine
	llmClient    interfaces.LLMClient
	eventBus     interfaces.EventBus
	logger       interfaces.Logger
	
	// Configuration
	config *Config
	
	// Runtime state
	isRunning bool
}

// Config holds the framework configuration
type Config struct {
	OllamaURL      string
	LLMModel       string
	LogLevel       string
	BrowserHeadless bool
	MemoryType     string
}

// NewFramework creates a new agent framework with all components
func NewFramework(config *Config) (*Framework, error) {
	// Initialize logger
	logger := logger.NewLogrusLogger(config.LogLevel)
	
	// Initialize memory store
	var memoryStore interfaces.MemoryStore
	switch config.MemoryType {
	case "memory":
		memoryStore = memory.NewInMemoryStore(logger)
	default:
		memoryStore = memory.NewInMemoryStore(logger)
	}
	
	// Initialize event bus
	eventBus := eventbus.NewInMemoryEventBus(logger)
	
	// Initialize LLM client
	llmClient := llm.NewOllamaClientWithModel(config.OllamaURL, config.LLMModel, logger)
	
	// Initialize planner
	taskPlanner := planner.NewTaskPlanner(llmClient, memoryStore, logger)
	
	// Initialize browser agent
	browserAgent := browser.NewPlaywrightAgent(logger, config.BrowserHeadless)
	
	// Initialize task executor
	taskExecutor := executor.NewTaskExecutor(memoryStore, eventBus, logger)
	
	// Initialize LangGraph engine
	langGraphEngine := langgraph.NewLangGraphEngine(memoryStore, logger)
	
	framework := &Framework{
		planner:      taskPlanner,
		executor:     taskExecutor,
		browserAgent: browserAgent,
		memory:       memoryStore,
		langGraph:    langGraphEngine,
		llmClient:    llmClient,
		eventBus:     eventBus,
		logger:       logger,
		config:       config,
		isRunning:    false,
	}
	
	// Register task handlers
	framework.registerTaskHandlers()
	
	logger.Info("Agent framework initialized successfully")
	
	return framework, nil
}

// Start initializes and starts the agent framework
func (f *Framework) Start(ctx context.Context) error {
	f.logger.Info("Starting agent framework")
	
	// Check LLM health
	if !f.llmClient.IsHealthy(ctx) {
		return fmt.Errorf("LLM client is not healthy - ensure Ollama is running on %s", f.config.OllamaURL)
	}
	
	// Initialize browser agent
	if err := f.browserAgent.(*browser.PlaywrightAgent).Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize browser agent: %w", err)
	}
	
	// Start event monitoring
	f.startEventMonitoring(ctx)
	
	f.isRunning = true
	f.logger.Info("Agent framework started successfully")
	
	return nil
}

// Stop gracefully shuts down the agent framework
func (f *Framework) Stop(ctx context.Context) error {
	f.logger.Info("Stopping agent framework")
	
	// Close browser agent
	if err := f.browserAgent.Close(ctx); err != nil {
		f.logger.WithField("error", err).Warn("Failed to close browser agent")
	}
	
	// Clear memory if needed
	if err := f.memory.Clear(ctx); err != nil {
		f.logger.WithField("error", err).Warn("Failed to clear memory")
	}
	
	f.isRunning = false
	f.logger.Info("Agent framework stopped")
	
	return nil
}

// ExecuteGoal creates a plan for the goal and executes it
func (f *Framework) ExecuteGoal(ctx context.Context, goal string) (*interfaces.Plan, error) {
	f.logger.WithField("goal", goal).Info("Executing goal")
	
	if !f.isRunning {
		return nil, fmt.Errorf("framework is not running")
	}
	
	// Create plan
	plan, err := f.planner.CreatePlan(ctx, goal)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	
	// Create workflow for plan execution
	workflowID := "plan:" + plan.ID
	states := []string{"pending", "running", "completed", "failed"}
	
	if err := f.langGraph.CreateWorkflow(ctx, workflowID, states); err != nil {
		f.logger.WithField("error", err).Warn("Failed to create workflow")
	}
	
	// Add workflow transitions
	transitions := map[string]map[string]string{
		"pending":   {"start": "running"},
		"running":   {"complete": "completed", "fail": "failed"},
		"completed": {},
		"failed":    {"retry": "pending"},
	}
	
	for from, events := range transitions {
		for event, to := range events {
			f.langGraph.AddTransition(ctx, workflowID, from, to, event)
		}
	}
	
	// Start plan execution
	go f.executePlan(ctx, plan)
	
	// Trigger workflow start
	f.langGraph.TriggerEvent(ctx, workflowID, "start", map[string]interface{}{
		"plan_id": plan.ID,
		"goal":    goal,
	})
	
	return plan, nil
}

// GetStatus returns the current status of the framework
func (f *Framework) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	status := map[string]interface{}{
		"running":     f.isRunning,
		"llm_healthy": f.llmClient.IsHealthy(ctx),
		"timestamp":   time.Now(),
	}
	
	// Add memory stats if available
	if memStore, ok := f.memory.(*memory.InMemoryStore); ok {
		status["memory"] = memStore.GetStats()
	}
	
	// Add event bus stats if available
	if eventBus, ok := f.eventBus.(*eventbus.InMemoryEventBus); ok {
		status["event_topics"] = eventBus.GetTopics()
	}
	
	return status, nil
}

// registerTaskHandlers registers handlers for different task types
func (f *Framework) registerTaskHandlers() {
	// Register task handlers
	browserHandler := executor.NewBrowserTaskHandler(f.browserAgent, f.logger)
	f.executor.RegisterHandler("browser", browserHandler)
	
	// Register script handler
	scriptHandler := executor.NewScriptTaskHandler(f.logger)
	f.executor.RegisterHandler("script", scriptHandler)
	
	// Register analysis handler
	analysisHandler := executor.NewAnalysisTaskHandler(f.logger, f.memory)
	f.executor.RegisterHandler("analysis", analysisHandler)
	
	f.logger.Info("Task handlers registered")
}

// executePlan executes all tasks in a plan
func (f *Framework) executePlan(ctx context.Context, plan *interfaces.Plan) {
	f.logger.WithField("plan_id", plan.ID).Info("Starting plan execution")
	
	workflowID := "plan:" + plan.ID
	
	// Execute tasks (simplified - in reality you'd handle dependencies)
	for _, task := range plan.Tasks {
		f.logger.WithFields(map[string]interface{}{
			"plan_id": plan.ID,
			"task_id": task.ID,
			"type":    task.Type,
		}).Info("Executing task")
		
		if err := f.executor.ExecuteTask(ctx, &task); err != nil {
			f.logger.WithFields(map[string]interface{}{
				"plan_id": plan.ID,
				"task_id": task.ID,
				"error":   err.Error(),
			}).Error("Task execution failed")
			
			// Trigger workflow failure
			f.langGraph.TriggerEvent(ctx, workflowID, "fail", map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			})
			return
		}
	}
	
	// All tasks completed successfully
	f.langGraph.TriggerEvent(ctx, workflowID, "complete", map[string]interface{}{
		"plan_id": plan.ID,
	})
	
	f.logger.WithField("plan_id", plan.ID).Info("Plan execution completed")
}

// startEventMonitoring starts monitoring framework events
func (f *Framework) startEventMonitoring(ctx context.Context) {
	// Subscribe to task events
	taskEvents, err := f.eventBus.Subscribe(ctx, "task.*")
	if err != nil {
		f.logger.WithField("error", err).Error("Failed to subscribe to task events")
		return
	}
	
	go func() {
		for {
			select {
			case event := <-taskEvents:
				f.logger.WithField("event", event).Debug("Received task event")
			case <-ctx.Done():
				return
			}
		}
	}()
}
