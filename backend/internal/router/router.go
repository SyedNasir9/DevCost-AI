package router

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"devcost-ai/internal/db"
	"devcost-ai/internal/handlers"
	"devcost-ai/internal/repositories"
	"devcost-ai/internal/services"
	"devcost-ai/pkg/logger"
)

// Router holds the Gin router and dependencies
type Router struct {
	engine *gin.Engine
	logger *logger.Logger
}

// NewRouter creates a new Gin router with middleware
func NewRouter(log *logger.Logger) *Router {
	// Set Gin mode based on environment
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	engine := gin.New()

	// Add middleware
	engine.Use(gin.Recovery())
	engine.Use(loggingMiddleware(log))
	engine.Use(corsMiddleware())

	return &Router{
		engine: engine,
		logger: log,
	}
}

// SetupRoutes configures all application routes
func (r *Router) SetupRoutes(database db.Database) {
	pool := database.GetPool()

	// Initialize repositories
	resourceRepo := repositories.NewResourceRepository(pool, r.logger)
	recommendationRepo := repositories.NewRecommendationRepository(pool, r.logger)
	actionRepo := repositories.NewActionRepository(pool, r.logger)

	// Initialize services
	wasteService := services.NewWasteDetectionService(r.logger, nil, services.DefaultWasteDetectionConfig())
	recommendationService := services.NewRecommendationService(r.logger, recommendationRepo)

	// For simulation service, we need an interface adapter
	simulationService := services.NewSimulationService(r.logger, recommendationService, nil)

	// AI Service (optional - based on config)
	aiConfig := services.AIConfig{
		Enabled: os.Getenv("AI_ENABLED") == "true",
		BaseURL: getEnvOrDefault("AI_BASE_URL", "http://localhost:11434"),
		Model:   getEnvOrDefault("AI_MODEL", "llama3.2"),
		Timeout: 30 * time.Second,
	}
	aiService := services.NewAIService(aiConfig)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(database, r.logger)
	resourcesHandler := handlers.NewResourcesHandler(resourceRepo, r.logger)
	wasteHandler := handlers.NewWasteHandler(wasteService, r.logger)
	recommendationsHandler := handlers.NewRecommendationsHandler(recommendationService, r.logger)
	actionHandler := handlers.NewActionHandler(nil, actionRepo, r.logger) // Pipeline initialized later if needed
	aiHandler := handlers.NewAIHandler(aiService, r.logger)

	// Health check endpoint (no auth required)
	r.engine.GET("/health", healthHandler.Check)

	// API version 1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Resource routes
		resources := v1.Group("/resources")
		{
			resources.GET("", resourcesHandler.GetResources)
			resources.GET("/stats", resourcesHandler.GetResourceStats)
			resources.GET("/:id", resourcesHandler.GetResourceByID)
		}

		// Waste detection routes
		waste := v1.Group("/waste")
		{
			waste.GET("", wasteHandler.GetWaste)
		}

		// Recommendations routes
		recommendations := v1.Group("/recommendations")
		{
			recommendations.GET("", recommendationsHandler.GetRecommendations)
			recommendations.GET("/:id", recommendationsHandler.GetRecommendationByID)
		}

		// Actions routes
		actions := v1.Group("/actions")
		{
			actions.GET("", actionHandler.ListActions)
			actions.GET("/:id", actionHandler.GetAction)
			actions.POST("/execute", actionHandler.ExecuteActions)
		}

		// AI routes (optional)
		ai := v1.Group("/ai")
		{
			ai.POST("/analyze", aiHandler.Analyze)
			ai.POST("/explain", aiHandler.Explain)
			ai.POST("/anomaly", aiHandler.DetectAnomaly)
		}
	}

	// Slack integration routes
	slack := r.engine.Group("/slack")
	{
		slackHandler := handlers.NewSlackHandler(
			r.logger,
			wasteService,
			recommendationService,
			nil, // ActionPipeline - optional
			simulationService,
			aiService,
			os.Getenv("SLACK_BOT_TOKEN"),
		)
		slack.POST("/command", slackHandler.HandleSlashCommand)
		slack.POST("/webhook", slackHandler.HandleSlackWebhook)
	}

	// Swagger documentation
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.logger.Info("All routes configured successfully")
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// loggingMiddleware adds structured logging to each request
func loggingMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()

		// Get status code
		statusCode := c.Writer.Status()

		// Build full path
		if raw != "" {
			path = path + "?" + raw
		}

		// Log the request
		log.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("ip", clientIP),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
