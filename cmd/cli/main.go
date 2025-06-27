package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ai-agent-framework/pkg/agent"
	"github.com/spf13/cobra"
)

var (
	ollamaURL       string
	llmModel        string
	logLevel        string
	browserHeadless bool
	memoryType      string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "agent-cli",
		Short: "AI Agent Framework CLI",
		Long:  "Command line interface for the local AI agent framework",
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&ollamaURL, "ollama-url", "http://localhost:11434", "Ollama API URL")
	rootCmd.PersistentFlags().StringVar(&llmModel, "llm-model", "deepseek-r1:latest", "LLM model to use")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&browserHeadless, "headless", true, "Run browser in headless mode")
	rootCmd.PersistentFlags().StringVar(&memoryType, "memory-type", "memory", "Memory backend type")

	// Add commands
	rootCmd.AddCommand(planCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(executeCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func planCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan [goal]",
		Short: "Create a plan for the given goal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			goal := args[0]

			framework, err := createFramework()
			if err != nil {
				return fmt.Errorf("failed to create framework: %w", err)
			}

			ctx := context.Background()
			if err := framework.Start(ctx); err != nil {
				return fmt.Errorf("failed to start framework: %w", err)
			}
			defer framework.Stop(ctx)

			plan, err := framework.ExecuteGoal(ctx, goal)
			if err != nil {
				return fmt.Errorf("failed to create plan: %w", err)
			}

			fmt.Printf("Plan created successfully!\n")
			fmt.Printf("Plan ID: %s\n", plan.ID)
			fmt.Printf("Goal: %s\n", plan.Goal)
			fmt.Printf("Tasks: %d\n", len(plan.Tasks))
			fmt.Printf("Status: %s\n", plan.Status)

			fmt.Printf("\nTasks:\n")
			for i, task := range plan.Tasks {
				fmt.Printf("%d. [%s] %s\n", i+1, task.Type, task.Description)
			}

			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Get framework status",
		RunE: func(cmd *cobra.Command, args []string) error {
			framework, err := createFramework()
			if err != nil {
				return fmt.Errorf("failed to create framework: %w", err)
			}

			ctx := context.Background()
			if err := framework.Start(ctx); err != nil {
				return fmt.Errorf("failed to start framework: %w", err)
			}
			defer framework.Stop(ctx)

			status, err := framework.GetStatus(ctx)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			fmt.Printf("Framework Status:\n")
			for key, value := range status {
				fmt.Printf("  %s: %v\n", key, value)
			}

			return nil
		},
	}
}

func executeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "execute [goal]",
		Short: "Execute a goal and monitor progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			goal := args[0]

			framework, err := createFramework()
			if err != nil {
				return fmt.Errorf("failed to create framework: %w", err)
			}

			ctx := context.Background()
			if err := framework.Start(ctx); err != nil {
				return fmt.Errorf("failed to start framework: %w", err)
			}
			defer framework.Stop(ctx)

			fmt.Printf("Executing goal: %s\n", goal)

			plan, err := framework.ExecuteGoal(ctx, goal)
			if err != nil {
				return fmt.Errorf("failed to execute goal: %w", err)
			}

			fmt.Printf("Plan created and execution started!\n")
			fmt.Printf("Plan ID: %s\n", plan.ID)
			fmt.Printf("Tasks: %d\n", len(plan.Tasks))

			// In a real implementation, you'd monitor progress here
			fmt.Printf("Monitoring execution... (Press Ctrl+C to stop)\n")

			// Simple wait - in reality you'd subscribe to events
			<-ctx.Done()
			return ctx.Err()
		},
	}
}

func createFramework() (*agent.Framework, error) {
	config := &agent.Config{
		OllamaURL:       ollamaURL,
		LLMModel:        llmModel,
		LogLevel:        logLevel,
		BrowserHeadless: browserHeadless,
		MemoryType:      memoryType,
	}

	return agent.NewFramework(config)
}
