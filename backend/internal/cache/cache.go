// Package cache provides Redis caching operations for annotations.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/pointcloud-annotator/backend/internal/config"
	"github.com/pointcloud-annotator/backend/internal/models"
)

const (
	// Cache key prefixes
	annotationKeyPrefix = "annotation:"
	allAnnotationsKey   = "annotations:all"

	// Default TTL for cached items
	defaultTTL = 5 * time.Minute
)

// Cache defines the interface for caching operations.
type Cache interface {
	// Get retrieves an annotation from cache by ID.
	Get(ctx context.Context, id string) (*models.Annotation, error)

	// GetAll retrieves all cached annotations.
	GetAll(ctx context.Context) ([]models.Annotation, bool, error)

	// Set stores an annotation in cache.
	Set(ctx context.Context, annotation *models.Annotation) error

	// SetAll stores all annotations in cache.
	SetAll(ctx context.Context, annotations []models.Annotation) error

	// Delete removes an annotation from cache.
	Delete(ctx context.Context, id string) error

	// InvalidateAll removes all cached annotations.
	InvalidateAll(ctx context.Context) error

	// Close closes the cache connection.
	Close() error
}

// RedisCache implements Cache using Redis.
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
	ttl    time.Duration
}

// NewRedisCache creates a new Redis cache.
func NewRedisCache(cfg *config.Config, logger *zap.Logger) (Cache, error) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis cache")

	return &RedisCache{
		client: client,
		logger: logger,
		ttl:    defaultTTL,
	}, nil
}

// Get retrieves an annotation from cache by ID.
func (c *RedisCache) Get(ctx context.Context, id string) (*models.Annotation, error) {
	key := annotationKeyPrefix + id

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		c.logger.Warn("Failed to get from cache", zap.String("key", key), zap.Error(err))
		return nil, nil // Treat errors as cache miss
	}

	var annotation models.Annotation
	if err := json.Unmarshal(data, &annotation); err != nil {
		c.logger.Warn("Failed to unmarshal cached annotation", zap.Error(err))
		return nil, nil
	}

	c.logger.Debug("Cache hit", zap.String("key", key))
	return &annotation, nil
}

// GetAll retrieves all cached annotations.
func (c *RedisCache) GetAll(ctx context.Context) ([]models.Annotation, bool, error) {
	data, err := c.client.Get(ctx, allAnnotationsKey).Bytes()
	if err == redis.Nil {
		return nil, false, nil // Cache miss
	}
	if err != nil {
		c.logger.Warn("Failed to get all from cache", zap.Error(err))
		return nil, false, nil
	}

	var annotations []models.Annotation
	if err := json.Unmarshal(data, &annotations); err != nil {
		c.logger.Warn("Failed to unmarshal cached annotations", zap.Error(err))
		return nil, false, nil
	}

	c.logger.Debug("Cache hit for all annotations")
	return annotations, true, nil
}

// Set stores an annotation in cache.
func (c *RedisCache) Set(ctx context.Context, annotation *models.Annotation) error {
	key := annotationKeyPrefix + annotation.ID

	data, err := json.Marshal(annotation)
	if err != nil {
		c.logger.Warn("Failed to marshal annotation for cache", zap.Error(err))
		return err
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		c.logger.Warn("Failed to set cache", zap.String("key", key), zap.Error(err))
		return err
	}

	// Invalidate the "all" cache since data changed
	_ = c.InvalidateAll(ctx)

	c.logger.Debug("Cached annotation", zap.String("key", key))
	return nil
}

// SetAll stores all annotations in cache.
func (c *RedisCache) SetAll(ctx context.Context, annotations []models.Annotation) error {
	data, err := json.Marshal(annotations)
	if err != nil {
		c.logger.Warn("Failed to marshal annotations for cache", zap.Error(err))
		return err
	}

	if err := c.client.Set(ctx, allAnnotationsKey, data, c.ttl).Err(); err != nil {
		c.logger.Warn("Failed to set all cache", zap.Error(err))
		return err
	}

	c.logger.Debug("Cached all annotations", zap.Int("count", len(annotations)))
	return nil
}

// Delete removes an annotation from cache.
func (c *RedisCache) Delete(ctx context.Context, id string) error {
	key := annotationKeyPrefix + id

	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.Warn("Failed to delete from cache", zap.String("key", key), zap.Error(err))
		return err
	}

	// Invalidate the "all" cache since data changed
	_ = c.InvalidateAll(ctx)

	c.logger.Debug("Deleted from cache", zap.String("key", key))
	return nil
}

// InvalidateAll removes all cached annotations.
func (c *RedisCache) InvalidateAll(ctx context.Context) error {
	if err := c.client.Del(ctx, allAnnotationsKey).Err(); err != nil {
		c.logger.Warn("Failed to invalidate all cache", zap.Error(err))
		return err
	}
	return nil
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	c.logger.Info("Closing Redis connection")
	return c.client.Close()
}
