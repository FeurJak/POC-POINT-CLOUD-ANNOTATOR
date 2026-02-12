// Package models contains the data models for the application.
package models

import (
	"time"
)

// Annotation represents a point cloud annotation with its 3D position and metadata.
type Annotation struct {
	ID          string    `json:"id"`
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Z           float64   `json:"z"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateAnnotationRequest represents the request body for creating an annotation.
type CreateAnnotationRequest struct {
	X           float64 `json:"x" binding:"required"`
	Y           float64 `json:"y" binding:"required"`
	Z           float64 `json:"z" binding:"required"`
	Title       string  `json:"title" binding:"required,max=256"`
	Description string  `json:"description" binding:"max=256"`
}

// UpdateAnnotationRequest represents the request body for updating an annotation.
type UpdateAnnotationRequest struct {
	X           *float64 `json:"x,omitempty"`
	Y           *float64 `json:"y,omitempty"`
	Z           *float64 `json:"z,omitempty"`
	Title       *string  `json:"title,omitempty" binding:"omitempty,max=256"`
	Description *string  `json:"description,omitempty" binding:"omitempty,max=256"`
}

// AnnotationResponse wraps a single annotation in the API response.
type AnnotationResponse struct {
	Data Annotation `json:"data"`
}

// AnnotationsResponse wraps multiple annotations in the API response.
type AnnotationsResponse struct {
	Data []Annotation `json:"data"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
