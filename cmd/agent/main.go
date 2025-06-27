package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ai-agent-framework/pkg/agent"
	"github.com/ai-agent-framework/pkg/interfaces"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration from environment variables
	config := &agent.Config{
		OllamaURL:       getEnv("OLLAMA_URL", "http://localhost:11434"),
		LLMModel:        getEnv("LLM_MODEL", "deepseek-r1:latest"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		BrowserHeadless: getEnvBool("BROWSER_HEADLESS", true),
		MemoryType:      getEnv("MEMORY_TYPE", "memory"),
	}

	// Create agent framework
	framework, err := agent.NewFramework(config)
	if err != nil {
		log.Fatalf("Failed to create agent framework: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the framework
	if err := framework.Start(ctx); err != nil {
		log.Fatalf("Failed to start agent framework: %v", err)
	}

	// Setup REST API
	router := setupRouter(framework)

	// Start HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Println("Starting HTTP server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := framework.Stop(ctx); err != nil {
		log.Printf("Failed to stop framework: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(framework interfaces.AgentFramework) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		status, err := framework.GetStatus(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, status)
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Execute goal endpoint
		v1.POST("/goals", func(c *gin.Context) {
			var request struct {
				Goal string `json:"goal" binding:"required"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			plan, err := framework.ExecuteGoal(c.Request.Context(), request.Goal)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"plan_id": plan.ID,
				"goal":    plan.Goal,
				"tasks":   len(plan.Tasks),
				"status":  plan.Status,
			})
		})

		// Get framework status
		v1.GET("/status", func(c *gin.Context) {
			status, err := framework.GetStatus(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, status)
		})
	}

	return router
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
