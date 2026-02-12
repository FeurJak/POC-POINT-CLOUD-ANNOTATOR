package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnnotation_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	annotation := Annotation{
		ID:          "test-uuid",
		X:           1.5,
		Y:           2.5,
		Z:           3.5,
		Title:       "Test Annotation",
		Description: "Test Description",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Marshal to JSON
	data, err := json.Marshal(annotation)
	assert.NoError(t, err)

	// Unmarshal back
	var unmarshaled Annotation
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, annotation.ID, unmarshaled.ID)
	assert.Equal(t, annotation.X, unmarshaled.X)
	assert.Equal(t, annotation.Y, unmarshaled.Y)
	assert.Equal(t, annotation.Z, unmarshaled.Z)
	assert.Equal(t, annotation.Title, unmarshaled.Title)
	assert.Equal(t, annotation.Description, unmarshaled.Description)
}

func TestCreateAnnotationRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateAnnotationRequest
		valid   bool
	}{
		{
			name: "valid request",
			request: CreateAnnotationRequest{
				X:           1.0,
				Y:           2.0,
				Z:           3.0,
				Title:       "Valid Title",
				Description: "Description",
			},
			valid: true,
		},
		{
			name: "valid request without description",
			request: CreateAnnotationRequest{
				X:     1.0,
				Y:     2.0,
				Z:     3.0,
				Title: "Valid Title",
			},
			valid: true,
		},
		{
			name: "empty title",
			request: CreateAnnotationRequest{
				X:     1.0,
				Y:     2.0,
				Z:     3.0,
				Title: "",
			},
			valid: false,
		},
		{
			name: "negative coordinates are valid",
			request: CreateAnnotationRequest{
				X:     -100.5,
				Y:     -200.5,
				Z:     -300.5,
				Title: "Negative Coords",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - title must not be empty
			isValid := tt.request.Title != ""
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestUpdateAnnotationRequest_PartialUpdate(t *testing.T) {
	// Test that optional fields can be nil
	newTitle := "New Title"
	newX := 10.0

	request := UpdateAnnotationRequest{
		Title: &newTitle,
		X:     &newX,
		// Y, Z, Description are nil - should not be updated
	}

	assert.NotNil(t, request.Title)
	assert.Equal(t, "New Title", *request.Title)
	assert.NotNil(t, request.X)
	assert.Equal(t, 10.0, *request.X)
	assert.Nil(t, request.Y)
	assert.Nil(t, request.Z)
	assert.Nil(t, request.Description)
}

func TestAnnotationResponse_Structure(t *testing.T) {
	annotation := Annotation{
		ID:    "test-id",
		Title: "Test",
	}

	response := AnnotationResponse{
		Data: annotation,
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	// Check that it has the correct structure
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Contains(t, parsed, "data")
}

func TestAnnotationsResponse_EmptyList(t *testing.T) {
	response := AnnotationsResponse{
		Data: []Annotation{},
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	dataField, ok := parsed["data"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, dataField, 0)
}

func TestErrorResponse_Structure(t *testing.T) {
	response := ErrorResponse{
		Error:   "not_found",
		Message: "Annotation not found",
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, "not_found", parsed["error"])
	assert.Equal(t, "Annotation not found", parsed["message"])
}
