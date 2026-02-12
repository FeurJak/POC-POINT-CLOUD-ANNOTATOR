// Package handler provides the business logic handlers for annotation operations.
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/pointcloud-annotator/backend/internal/cache"
	"github.com/pointcloud-annotator/backend/internal/database"
	"github.com/pointcloud-annotator/backend/internal/models"
)

// Handler provides HTTP handlers for annotation operations.
type Handler struct {
	repo   database.Repository
	cache  cache.Cache
	logger *zap.Logger
}

// NewHandler creates a new annotation handler.
func NewHandler(repo database.Repository, cache cache.Cache, logger *zap.Logger) *Handler {
	return &Handler{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// RegisterRoutes registers the handler routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/annotations", h.Create)
	rg.GET("/annotations", h.GetAll)
	rg.GET("/annotations/:id", h.GetByID)
	rg.PUT("/annotations/:id", h.Update)
	rg.PATCH("/annotations/:id", h.Update)
	rg.DELETE("/annotations/:id", h.Delete)
}

// Create handles the creation of a new annotation.
// @Summary Create annotation
// @Description Create a new point cloud annotation
// @Tags annotations
// @Accept json
// @Produce json
// @Param annotation body models.CreateAnnotationRequest true "Annotation data"
// @Success 201 {object} models.AnnotationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/annotations [post]
func (h *Handler) Create(c *gin.Context) {
	var req models.CreateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create request", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Validate title length (max 256 bytes)
	if len(req.Title) > 256 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "title exceeds maximum length of 256 bytes",
		})
		return
	}

	ctx := context.Background()
	annotation, err := h.repo.Create(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to create annotation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to create annotation",
		})
		return
	}

	// Cache the new annotation
	_ = h.cache.Set(ctx, annotation)

	c.JSON(http.StatusCreated, models.AnnotationResponse{Data: *annotation})
}

// GetAll handles retrieving all annotations.
// @Summary Get all annotations
// @Description Retrieve all point cloud annotations
// @Tags annotations
// @Produce json
// @Success 200 {object} models.AnnotationsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/annotations [get]
func (h *Handler) GetAll(c *gin.Context) {
	ctx := context.Background()

	// Try cache first
	annotations, found, err := h.cache.GetAll(ctx)
	if err == nil && found {
		h.logger.Debug("Returning cached annotations")
		c.JSON(http.StatusOK, models.AnnotationsResponse{Data: annotations})
		return
	}

	// Cache miss, get from database
	annotations, err = h.repo.GetAll(ctx)
	if err != nil {
		h.logger.Error("Failed to get annotations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve annotations",
		})
		return
	}

	// Update cache
	_ = h.cache.SetAll(ctx, annotations)

	c.JSON(http.StatusOK, models.AnnotationsResponse{Data: annotations})
}

// GetByID handles retrieving a single annotation by ID.
// @Summary Get annotation by ID
// @Description Retrieve a specific annotation by its ID
// @Tags annotations
// @Produce json
// @Param id path string true "Annotation ID"
// @Success 200 {object} models.AnnotationResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/annotations/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	// Try cache first
	annotation, err := h.cache.Get(ctx, id)
	if err == nil && annotation != nil {
		h.logger.Debug("Returning cached annotation", zap.String("id", id))
		c.JSON(http.StatusOK, models.AnnotationResponse{Data: *annotation})
		return
	}

	// Cache miss, get from database
	annotation, err = h.repo.GetByID(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get annotation", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve annotation",
		})
		return
	}

	if annotation == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "annotation not found",
		})
		return
	}

	// Update cache
	_ = h.cache.Set(ctx, annotation)

	c.JSON(http.StatusOK, models.AnnotationResponse{Data: *annotation})
}

// Update handles updating an existing annotation.
// @Summary Update annotation
// @Description Update an existing annotation
// @Tags annotations
// @Accept json
// @Produce json
// @Param id path string true "Annotation ID"
// @Param annotation body models.UpdateAnnotationRequest true "Updated annotation data"
// @Success 200 {object} models.AnnotationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/annotations/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Validate title length if provided
	if req.Title != nil && len(*req.Title) > 256 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "title exceeds maximum length of 256 bytes",
		})
		return
	}

	ctx := context.Background()
	annotation, err := h.repo.Update(ctx, id, &req)
	if err != nil {
		h.logger.Error("Failed to update annotation", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to update annotation",
		})
		return
	}

	if annotation == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "annotation not found",
		})
		return
	}

	// Update cache
	_ = h.cache.Set(ctx, annotation)

	c.JSON(http.StatusOK, models.AnnotationResponse{Data: *annotation})
}

// Delete handles deleting an annotation.
// @Summary Delete annotation
// @Description Delete an annotation by ID
// @Tags annotations
// @Produce json
// @Param id path string true "Annotation ID"
// @Success 204 "No Content"
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/annotations/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	err := h.repo.Delete(ctx, id)
	if err != nil {
		if err.Error() == "annotation not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "annotation not found",
			})
			return
		}

		h.logger.Error("Failed to delete annotation", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to delete annotation",
		})
		return
	}

	// Remove from cache
	_ = h.cache.Delete(ctx, id)

	c.Status(http.StatusNoContent)
}
