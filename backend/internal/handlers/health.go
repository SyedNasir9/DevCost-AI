package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devcost-ai/internal/db"
	"devcost-ai/pkg/logger"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db     db.Database
	logger *logger.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(database db.Database, log *logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:     database,
		logger: log,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// Check handles the health check endpoint
// @Summary Health check endpoint
// @Description Returns the health status of the DevCost AI service
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Service:   "devcost-ai",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Checks:    make(map[string]string),
	}

	// Check database connectivity
	if h.db != nil {
		if err := h.db.Health(c.Request.Context()); err != nil {
			response.Checks["database"] = "error: " + err.Error()
			response.Status = "degraded"
			h.logger.Error("Database health check failed", zap.Error(err))
		} else {
			response.Checks["database"] = "ok"
		}
	} else {
		response.Checks["database"] = "not_initialized"
	}

	// Determine HTTP status based on overall health
	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}
