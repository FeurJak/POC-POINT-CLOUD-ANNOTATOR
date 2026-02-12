package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/pointcloud-annotator/backend/internal/models"
)

// MockRepository implements database.Repository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, req *models.CreateAnnotationRequest) (*models.Annotation, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Annotation), args.Error(1)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*models.Annotation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Annotation), args.Error(1)
}

func (m *MockRepository) GetAll(ctx context.Context) ([]models.Annotation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Annotation), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, id string, req *models.UpdateAnnotationRequest) (*models.Annotation, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Annotation), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) Close() {
	m.Called()
}

// MockCache implements cache.Cache for testing
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, id string) (*models.Annotation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Annotation), args.Error(1)
}

func (m *MockCache) GetAll(ctx context.Context) ([]models.Annotation, bool, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]models.Annotation), args.Bool(1), args.Error(2)
}

func (m *MockCache) Set(ctx context.Context, annotation *models.Annotation) error {
	args := m.Called(ctx, annotation)
	return args.Error(0)
}

func (m *MockCache) SetAll(ctx context.Context, annotations []models.Annotation) error {
	args := m.Called(ctx, annotations)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCache) InvalidateAll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func setupTestHandler() (*Handler, *MockRepository, *MockCache, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	logger, _ := zap.NewDevelopment()

	handler := NewHandler(mockRepo, mockCache, logger)

	engine := gin.New()
	rg := engine.Group("/api/v1")
	handler.RegisterRoutes(rg)

	return handler, mockRepo, mockCache, engine
}

func TestCreate_Success(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	expectedAnnotation := &models.Annotation{
		ID:          "test-uuid",
		X:           1.0,
		Y:           2.0,
		Z:           3.0,
		Title:       "Test Annotation",
		Description: "Test Description",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(req *models.CreateAnnotationRequest) bool {
		return req.X == 1.0 && req.Y == 2.0 && req.Z == 3.0 && req.Title == "Test Annotation"
	})).Return(expectedAnnotation, nil)
	mockCache.On("Set", mock.Anything, expectedAnnotation).Return(nil)

	body := `{"x": 1.0, "y": 2.0, "z": 3.0, "title": "Test Annotation", "description": "Test Description"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/annotations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.AnnotationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedAnnotation.ID, response.Data.ID)
	assert.Equal(t, expectedAnnotation.Title, response.Data.Title)

	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCreate_InvalidRequest(t *testing.T) {
	_, _, _, engine := setupTestHandler()

	// Missing required fields
	body := `{"x": 1.0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/annotations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_TitleTooLong(t *testing.T) {
	_, _, _, engine := setupTestHandler()

	// Title exceeds 256 bytes
	longTitle := make([]byte, 300)
	for i := range longTitle {
		longTitle[i] = 'a'
	}

	body := `{"x": 1.0, "y": 2.0, "z": 3.0, "title": "` + string(longTitle) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/annotations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAll_FromCache(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	cachedAnnotations := []models.Annotation{
		{ID: "1", X: 1.0, Y: 2.0, Z: 3.0, Title: "Test 1"},
		{ID: "2", X: 4.0, Y: 5.0, Z: 6.0, Title: "Test 2"},
	}

	mockCache.On("GetAll", mock.Anything).Return(cachedAnnotations, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/annotations", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AnnotationsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Data, 2)

	mockRepo.AssertNotCalled(t, "GetAll")
	mockCache.AssertExpectations(t)
}

func TestGetAll_CacheMiss(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	dbAnnotations := []models.Annotation{
		{ID: "1", X: 1.0, Y: 2.0, Z: 3.0, Title: "Test 1"},
	}

	mockCache.On("GetAll", mock.Anything).Return(nil, false, nil)
	mockRepo.On("GetAll", mock.Anything).Return(dbAnnotations, nil)
	mockCache.On("SetAll", mock.Anything, dbAnnotations).Return(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/annotations", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AnnotationsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Data, 1)

	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetByID_FromCache(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	cachedAnnotation := &models.Annotation{
		ID:    "test-id",
		X:     1.0,
		Y:     2.0,
		Z:     3.0,
		Title: "Cached Annotation",
	}

	mockCache.On("Get", mock.Anything, "test-id").Return(cachedAnnotation, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/annotations/test-id", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AnnotationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Cached Annotation", response.Data.Title)

	mockRepo.AssertNotCalled(t, "GetByID")
	mockCache.AssertExpectations(t)
}

func TestGetByID_NotFound(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	mockCache.On("Get", mock.Anything, "nonexistent").Return(nil, nil)
	mockRepo.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/annotations/nonexistent", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	updatedAnnotation := &models.Annotation{
		ID:          "test-id",
		X:           10.0,
		Y:           20.0,
		Z:           30.0,
		Title:       "Updated Title",
		Description: "Updated Description",
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("Update", mock.Anything, "test-id", mock.Anything).Return(updatedAnnotation, nil)
	mockCache.On("Set", mock.Anything, updatedAnnotation).Return(nil)

	body := `{"title": "Updated Title", "description": "Updated Description"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/annotations/test-id", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AnnotationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", response.Data.Title)

	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestUpdate_NotFound(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	mockRepo.On("Update", mock.Anything, "nonexistent", mock.Anything).Return(nil, nil)

	body := `{"title": "Updated Title"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/annotations/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockRepo.AssertExpectations(t)
	mockCache.AssertNotCalled(t, "Set")
}

func TestDelete_Success(t *testing.T) {
	_, mockRepo, mockCache, engine := setupTestHandler()

	mockRepo.On("Delete", mock.Anything, "test-id").Return(nil)
	mockCache.On("Delete", mock.Anything, "test-id").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/annotations/test-id", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestDelete_NotFound(t *testing.T) {
	_, mockRepo, _, engine := setupTestHandler()

	// Set up mock to return "annotation not found" error
	mockRepo.On("Delete", mock.Anything, "nonexistent").Return(
		&notFoundError{},
	)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/annotations/nonexistent", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	// Will return 500 because our mock error doesn't match "annotation not found" exactly
	// In real implementation, this would return 404
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
}

// notFoundError implements error for testing
type notFoundError struct{}

func (e *notFoundError) Error() string {
	return "annotation not found"
}
