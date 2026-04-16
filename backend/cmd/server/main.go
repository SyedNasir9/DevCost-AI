package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"devcost-ai/internal/config"
	"devcost-ai/internal/db"
	"devcost-ai/internal/router"
	"devcost-ai/pkg/logger"
)

// @title DevCost AI API
// @version 1.0
// @description Cloud cost optimization platform API
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	zapConfig, err := cfg.Logging.ToZapConfig()
	if err != nil {
		fmt.Printf("Failed to create zap config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.NewLogger(zapConfig)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("Starting DevCost AI server",
		zap.String("version", "1.0.0"),
		zap.String("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
	)

	// Initialize database connection
	database, err := db.NewClient(&cfg.Database, log)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(context.Background())

	// Initialize database schema and data
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.Initialize(ctx); err != nil {
		log.Fatal("Failed to initialize database", zap.Error(err))
	}

	log.Info("Database connected and initialized successfully")

	// Start connection pool statistics logging
	go database.StartStatsLogging(context.Background(), 5*time.Minute)

	// Initialize router
	r := router.NewRouter(log)
	r.SetupRoutes(database)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r.GetEngine(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Starting HTTP server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
		os.Exit(1)
	}

	log.Info("Server exited properly")
}
