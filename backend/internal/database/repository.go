// Package database provides PostgreSQL database operations for annotations.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/pointcloud-annotator/backend/internal/config"
	"github.com/pointcloud-annotator/backend/internal/models"
)

// Repository defines the interface for annotation data operations.
type Repository interface {
	// Create creates a new annotation and returns its ID.
	Create(ctx context.Context, req *models.CreateAnnotationRequest) (*models.Annotation, error)

	// GetByID retrieves an annotation by its ID.
	GetByID(ctx context.Context, id string) (*models.Annotation, error)

	// GetAll retrieves all annotations.
	GetAll(ctx context.Context) ([]models.Annotation, error)

	// Update updates an existing annotation.
	Update(ctx context.Context, id string, req *models.UpdateAnnotationRequest) (*models.Annotation, error)

	// Delete removes an annotation by its ID.
	Delete(ctx context.Context, id string) error

	// Close closes the database connection.
	Close()
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(cfg *config.Config, logger *zap.Logger) (Repository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &PostgresRepository{
		pool:   pool,
		logger: logger,
	}

	if err := repo.migrate(ctx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Connected to PostgreSQL database")
	return repo, nil
}

// migrate creates the necessary database tables if they don't exist.
func (r *PostgresRepository) migrate(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS annotations (
			id UUID PRIMARY KEY,
			x DOUBLE PRECISION NOT NULL,
			y DOUBLE PRECISION NOT NULL,
			z DOUBLE PRECISION NOT NULL,
			title VARCHAR(256) NOT NULL,
			description VARCHAR(256) DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_annotations_created_at ON annotations(created_at);
	`

	_, err := r.pool.Exec(ctx, query)
	return err
}

// Create creates a new annotation.
func (r *PostgresRepository) Create(ctx context.Context, req *models.CreateAnnotationRequest) (*models.Annotation, error) {
	annotation := &models.Annotation{
		ID:          uuid.New().String(),
		X:           req.X,
		Y:           req.Y,
		Z:           req.Z,
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	query := `
		INSERT INTO annotations (id, x, y, z, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		annotation.ID,
		annotation.X,
		annotation.Y,
		annotation.Z,
		annotation.Title,
		annotation.Description,
		annotation.CreatedAt,
		annotation.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create annotation", zap.Error(err))
		return nil, fmt.Errorf("failed to create annotation: %w", err)
	}

	r.logger.Info("Created annotation", zap.String("id", annotation.ID))
	return annotation, nil
}

// GetByID retrieves an annotation by its ID.
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*models.Annotation, error) {
	query := `
		SELECT id, x, y, z, title, description, created_at, updated_at
		FROM annotations
		WHERE id = $1
	`

	var annotation models.Annotation
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&annotation.ID,
		&annotation.X,
		&annotation.Y,
		&annotation.Z,
		&annotation.Title,
		&annotation.Description,
		&annotation.CreatedAt,
		&annotation.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.Error("Failed to get annotation", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get annotation: %w", err)
	}

	return &annotation, nil
}

// GetAll retrieves all annotations.
func (r *PostgresRepository) GetAll(ctx context.Context) ([]models.Annotation, error) {
	query := `
		SELECT id, x, y, z, title, description, created_at, updated_at
		FROM annotations
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		r.logger.Error("Failed to get annotations", zap.Error(err))
		return nil, fmt.Errorf("failed to get annotations: %w", err)
	}
	defer rows.Close()

	var annotations []models.Annotation
	for rows.Next() {
		var annotation models.Annotation
		err := rows.Scan(
			&annotation.ID,
			&annotation.X,
			&annotation.Y,
			&annotation.Z,
			&annotation.Title,
			&annotation.Description,
			&annotation.CreatedAt,
			&annotation.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan annotation row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan annotation: %w", err)
		}
		annotations = append(annotations, annotation)
	}

	if annotations == nil {
		annotations = []models.Annotation{}
	}

	return annotations, nil
}

// Update updates an existing annotation.
func (r *PostgresRepository) Update(ctx context.Context, id string, req *models.UpdateAnnotationRequest) (*models.Annotation, error) {
	// First, get the existing annotation
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	// Apply updates
	if req.X != nil {
		existing.X = *req.X
	}
	if req.Y != nil {
		existing.Y = *req.Y
	}
	if req.Z != nil {
		existing.Z = *req.Z
	}
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	existing.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE annotations
		SET x = $2, y = $3, z = $4, title = $5, description = $6, updated_at = $7
		WHERE id = $1
	`

	_, err = r.pool.Exec(ctx, query,
		existing.ID,
		existing.X,
		existing.Y,
		existing.Z,
		existing.Title,
		existing.Description,
		existing.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update annotation", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to update annotation: %w", err)
	}

	r.logger.Info("Updated annotation", zap.String("id", id))
	return existing, nil
}

// Delete removes an annotation by its ID.
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM annotations WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete annotation", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to delete annotation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("annotation not found")
	}

	r.logger.Info("Deleted annotation", zap.String("id", id))
	return nil
}

// Close closes the database connection pool.
func (r *PostgresRepository) Close() {
	r.pool.Close()
	r.logger.Info("Closed database connection")
}
