package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devcost-ai/internal/models"
	"devcost-ai/internal/repositories"
	"devcost-ai/pkg/logger"
)

// ResourcesHandler handles resource-related HTTP requests
type ResourcesHandler struct {
	repository *repositories.ResourceRepository
	logger     *logger.Logger
}

// NewResourcesHandler creates a new resources handler
func NewResourcesHandler(repository *repositories.ResourceRepository, log *logger.Logger) *ResourcesHandler {
	return &ResourcesHandler{
		repository: repository,
		logger:     log,
	}
}

// GetResourcesResponse represents the response structure for resources endpoint
type GetResourcesResponse struct {
	Success   bool                     `json:"success"`
	Resources []*models.Resource       `json:"resources"`
	Total     int64                    `json:"total"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"page_size"`
	HasNext   bool                     `json:"has_next"`
	Filters   *ResourceFiltersResponse `json:"filters,omitempty"`
}

// ResourceFiltersResponse represents the applied filters in response
type ResourceFiltersResponse struct {
	ResourceTypes []string          `json:"resource_types,omitempty"`
	Regions       []string          `json:"regions,omitempty"`
	States        []string          `json:"states,omitempty"`
	AccountIDs    []string          `json:"account_ids,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// GetResources handles GET /resources endpoint
func (h *ResourcesHandler) GetResources(c *gin.Context) {
	h.logger.Info("Handling GET /resources request")

	// Parse query parameters
	filters, err := h.parseQueryFilters(c)
	if err != nil {
		h.logger.Error("Failed to parse query filters",
			zap.Error(err),
			zap.String("query", c.Request.URL.RawQuery),
		)
		h.sendErrorResponse(c, http.StatusBadRequest, "INVALID_QUERY", "Invalid query parameters", err.Error())
		return
	}

	h.logger.Debug("Parsed query filters",
		zap.Strings("resource_types", filters.ResourceTypes),
		zap.Strings("regions", filters.Regions),
		zap.Strings("states", filters.States),
		zap.Strings("account_ids", filters.AccountIDs),
		zap.Any("tags", filters.Tags),
		zap.Int("limit", filters.Limit),
		zap.Int("offset", filters.Offset),
	)

	// Fetch resources from database
	resources, err := h.repository.GetResourcesByFilter(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to fetch resources from database",
			zap.Error(err),
		)
		h.sendErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch resources", err.Error())
		return
	}

	// Get total count for pagination
	total, err := h.repository.GetResourceCount(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get resource count",
			zap.Error(err),
		)
		// Continue with the response even if count fails
		total = int64(len(resources))
	}

	// Calculate pagination info
	page := 1
	if filters.Limit > 0 {
		page = (filters.Offset / filters.Limit) + 1
	}

	hasNext := false
	if filters.Limit > 0 {
		hasNext = int64(filters.Offset+filters.Limit) < total
	}

	// Prepare response
	response := &GetResourcesResponse{
		Success:   true,
		Resources: resources,
		Total:     total,
		Page:      page,
		PageSize:  filters.Limit,
		HasNext:   hasNext,
	}

	// Include filters in response if any were applied
	if h.hasActiveFilters(filters) {
		response.Filters = &ResourceFiltersResponse{
			ResourceTypes: filters.ResourceTypes,
			Regions:       filters.Regions,
			States:        filters.States,
			AccountIDs:    filters.AccountIDs,
			Tags:          filters.Tags,
			Limit:         filters.Limit,
			Offset:        filters.Offset,
		}
	}

	h.logger.Info("Successfully fetched resources",
		zap.Int64("total", total),
		zap.Int("returned", len(resources)),
		zap.Int("page", page),
		zap.Bool("has_next", hasNext),
	)

	c.JSON(http.StatusOK, response)
}

// GetResourceTypes handles GET /resources/types endpoint
func (h *ResourcesHandler) GetResourceTypes(c *gin.Context) {
	h.logger.Info("Handling GET /resources/types request")

	// Get resource statistics to extract types
	stats, err := h.repository.GetResourceStatistics(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get resource statistics",
			zap.Error(err),
		)
		h.sendErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get resource types", err.Error())
		return
	}

	// Extract resource types from statistics
	resourceTypes := make([]string, 0, len(stats.ByType))
	for resourceType := range stats.ByType {
		resourceTypes = append(resourceTypes, resourceType)
	}

	response := struct {
		Success       bool     `json:"success"`
		ResourceTypes []string `json:"resource_types"`
		Total         int64    `json:"total"`
	}{
		Success:       true,
		ResourceTypes: resourceTypes,
		Total:         int64(len(resourceTypes)),
	}

	h.logger.Info("Successfully fetched resource types",
		zap.Int("count", len(resourceTypes)),
	)

	c.JSON(http.StatusOK, response)
}

// GetRegions handles GET /resources/regions endpoint
func (h *ResourcesHandler) GetRegions(c *gin.Context) {
	h.logger.Info("Handling GET /resources/regions request")

	// Get resources and extract unique regions
	filter := &repositories.ResourceFilter{
		Limit: 1000, // Reasonable limit for regions
	}

	resources, err := h.repository.GetResourcesByFilter(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to fetch resources for regions",
			zap.Error(err),
		)
		h.sendErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get regions", err.Error())
		return
	}

	// Extract unique regions
	regionsMap := make(map[string]bool)
	for _, resource := range resources {
		if resource.Region != "" {
			regionsMap[resource.Region] = true
		}
	}

	regions := make([]string, 0, len(regionsMap))
	for region := range regionsMap {
		regions = append(regions, region)
	}

	response := struct {
		Success bool     `json:"success"`
		Regions []string `json:"regions"`
		Total   int      `json:"total"`
	}{
		Success: true,
		Regions: regions,
		Total:   len(regions),
	}

	h.logger.Info("Successfully fetched regions",
		zap.Int("count", len(regions)),
	)

	c.JSON(http.StatusOK, response)
}

// GetResourceByID handles GET /resources/:id endpoint
func (h *ResourcesHandler) GetResourceByID(c *gin.Context) {
	resourceID := c.Param("id")

	h.logger.Info("Handling GET /resources/:id request",
		zap.String("resource_id", resourceID),
	)

	if resourceID == "" {
		h.sendErrorResponse(c, http.StatusBadRequest, "MISSING_ID", "Resource ID is required", "")
		return
	}

	// Fetch resource from database
	resource, err := h.repository.GetResourceByResourceID(c.Request.Context(), resourceID)
	if err != nil {
		h.logger.Error("Failed to fetch resource by ID",
			zap.String("resource_id", resourceID),
			zap.Error(err),
		)

		if err.Error() == "resource with ID "+resourceID+" not found" {
			h.sendErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "Resource not found", "")
		} else {
			h.sendErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch resource", err.Error())
		}
		return
	}

	response := struct {
		Success  bool             `json:"success"`
		Resource *models.Resource `json:"resource"`
	}{
		Success:  true,
		Resource: resource,
	}

	h.logger.Info("Successfully fetched resource by ID",
		zap.String("resource_id", resourceID),
		zap.String("resource_type", string(resource.ResourceType)),
	)

	c.JSON(http.StatusOK, response)
}

// GetResourceStats handles GET /resources/stats endpoint
func (h *ResourcesHandler) GetResourceStats(c *gin.Context) {
	h.logger.Info("Handling GET /resources/stats request")

	// Get resource statistics
	stats, err := h.repository.GetResourceStatistics(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get resource statistics",
			zap.Error(err),
		)
		h.sendErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get resource statistics", err.Error())
		return
	}

	response := struct {
		Success    bool                             `json:"success"`
		Statistics *repositories.ResourceStatistics `json:"statistics"`
	}{
		Success:    true,
		Statistics: stats,
	}

	h.logger.Info("Successfully fetched resource statistics",
		zap.Int64("total_resources", stats.TotalCount),
		zap.Int64("type_count", stats.TypeCount),
	)

	c.JSON(http.StatusOK, response)
}

// parseQueryFilters parses query parameters into ResourceFilter
func (h *ResourcesHandler) parseQueryFilters(c *gin.Context) (*repositories.ResourceFilter, error) {
	filter := &repositories.ResourceFilter{
		Tags: make(map[string]string),
	}

	// Parse resource_type parameter
	if resourceTypes := c.QueryArray("resource_type"); len(resourceTypes) > 0 {
		filter.ResourceTypes = make([]models.ResourceType, 0, len(resourceTypes))
		for _, rt := range resourceTypes {
			resourceType := models.ResourceType(rt)
			if h.isValidResourceType(resourceType) {
				filter.ResourceTypes = append(filter.ResourceTypes, resourceType)
			} else {
				h.logger.Warn("Invalid resource type filter",
					zap.String("resource_type", rt),
				)
			}
		}
	}

	// Parse region parameter
	if regions := c.QueryArray("region"); len(regions) > 0 {
		filter.Regions = regions
	}

	// Parse state parameter
	if states := c.QueryArray("state"); len(states) > 0 {
		filter.States = make([]models.ResourceState, 0, len(states))
		for _, state := range states {
			resourceState := models.ResourceState(state)
			if h.isValidResourceState(resourceState) {
				filter.States = append(filter.States, resourceState)
			} else {
				h.logger.Warn("Invalid resource state filter",
					zap.String("state", state),
				)
			}
		}
	}

	// Parse account_id parameter
	if accountIDs := c.QueryArray("account_id"); len(accountIDs) > 0 {
		filter.AccountIDs = accountIDs
	}

	// Parse tag parameters (tag:key=value format)
	for key, values := range c.Request.URL.Query() {
		if len(key) > 4 && key[:4] == "tag:" {
			tagKey := key[4:] // Remove "tag:" prefix
			if len(values) > 0 {
				filter.Tags[tagKey] = values[0] // Take first value
			}
		}
	}

	// Parse limit parameter
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			h.logger.Warn("Invalid limit parameter",
				zap.String("limit", limitStr),
				zap.Error(err),
			)
		} else {
			// Validate limit range
			if limit > 0 && limit <= 1000 {
				filter.Limit = limit
			} else if limit > 1000 {
				filter.Limit = 1000 // Cap at 1000
				h.logger.Warn("Limit parameter capped at 1000",
					zap.Int("requested_limit", limit),
				)
			}
		}
	}

	// Parse offset parameter
	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			h.logger.Warn("Invalid offset parameter",
				zap.String("offset", offsetStr),
				zap.Error(err),
			)
		} else {
			if offset >= 0 {
				filter.Offset = offset
			} else {
				h.logger.Warn("Negative offset parameter ignored",
					zap.String("offset", offsetStr),
				)
			}
		}
	}

	// Set default limit if not provided
	if filter.Limit == 0 {
		filter.Limit = 50 // Default limit
	}

	return filter, nil
}

// isValidResourceType checks if a resource type is valid
func (h *ResourcesHandler) isValidResourceType(resourceType models.ResourceType) bool {
	validTypes := []models.ResourceType{
		models.ResourceTypeEC2,
		models.ResourceTypeRDS,
		models.ResourceTypeEBS,
		models.ResourceTypeLambda,
		models.ResourceTypeS3,
		models.ResourceTypeVPC,
		models.ResourceTypeSubnet,
		models.ResourceTypeEKS,
	}

	for _, validType := range validTypes {
		if resourceType == validType {
			return true
		}
	}
	return false
}

// isValidResourceState checks if a resource state is valid
func (h *ResourcesHandler) isValidResourceState(state models.ResourceState) bool {
	validStates := []models.ResourceState{
		models.ResourceStatePending,
		models.ResourceStateRunning,
		models.ResourceStateStopping,
		models.ResourceStateStopped,
		models.ResourceStateShuttingDown,
		models.ResourceStateTerminated,
		models.ResourceStateRebooting,
		models.ResourceStateAvailable,
		models.ResourceStateCreating,
		models.ResourceStateModifying,
		models.ResourceStateDeleting,
		models.ResourceStateBackupRestoring,
	}

	for _, validState := range validStates {
		if state == validState {
			return true
		}
	}
	return false
}

// hasActiveFilters checks if any filters are applied
func (h *ResourcesHandler) hasActiveFilters(filter *repositories.ResourceFilter) bool {
	return len(filter.ResourceTypes) > 0 ||
		len(filter.Regions) > 0 ||
		len(filter.States) > 0 ||
		len(filter.AccountIDs) > 0 ||
		len(filter.Tags) > 0 ||
		filter.Limit != 50 || // Default limit
		filter.Offset != 0
}

// sendErrorResponse sends a standardized error response
func (h *ResourcesHandler) sendErrorResponse(c *gin.Context, statusCode int, code, message, details string) {
	response := ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
		Details: details,
	}

	h.logger.Error("API error response",
		zap.Int("status_code", statusCode),
		zap.String("code", code),
		zap.String("message", message),
		zap.String("details", details),
	)

	c.JSON(statusCode, response)
}
