// Package main is the entry point for the point cloud annotator service.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/pointcloud-annotator/backend/internal/cache"
	"github.com/pointcloud-annotator/backend/internal/config"
	"github.com/pointcloud-annotator/backend/internal/database"
	"github.com/pointcloud-annotator/backend/internal/gateway"
	"github.com/pointcloud-annotator/backend/internal/handler"
)

func main() {
	// Parse command line flags
	role := flag.String("role", "", "Service role: gateway or handler (overrides SERVICE_ROLE env var)")
	port := flag.String("port", "", "Server port (overrides SERVER_PORT env var)")
	flag.Parse()

	// Override environment variables if flags are provided
	if *role != "" {
		os.Setenv("SERVICE_ROLE", *role)
	}
	if *port != "" {
		os.Setenv("SERVER_PORT", *port)
	}

	app := fx.New(
		fx.Provide(
			config.New,
			newLogger,
			newGinEngine,
		),
		fx.Invoke(startServer),
	)

	app.Run()
}

// newLogger creates a new zap logger based on the environment.
func newLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg.IsDevelopment() {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

// newGinEngine creates and configures a new Gin engine.
func newGinEngine(cfg *config.Config) *gin.Engine {
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	// CORS middleware
	engine.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	return engine
}

// startServer starts the HTTP server based on the configured role.
func startServer(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger, engine *gin.Engine) error {
	logger.Info("Starting service",
		zap.String("role", cfg.Role),
		zap.String("port", cfg.ServerPort),
	)

	// Setup API versioned routes
	apiV1 := engine.Group("/api/v1")

	// Health check endpoint
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"role":    cfg.Role,
			"service": "point-cloud-annotator",
		})
	})

	var repo database.Repository
	var cacheClient cache.Cache

	if cfg.IsHandler() {
		// Handler mode: connect to database and cache, register handlers
		var err error
		repo, err = database.NewPostgresRepository(cfg, logger)
		if err != nil {
			logger.Fatal("Failed to connect to database", zap.Error(err))
			return err
		}

		cacheClient, err = cache.NewRedisCache(cfg, logger)
		if err != nil {
			logger.Fatal("Failed to connect to Redis", zap.Error(err))
			return err
		}

		h := handler.NewHandler(repo, cacheClient, logger)
		h.RegisterRoutes(apiV1)

		logger.Info("Handler routes registered")
	} else {
		// Gateway mode: setup proxy to handler
		gw := gateway.NewGateway(cfg, logger)
		gw.RegisterRoutes(apiV1)

		logger.Info("Gateway routes registered",
			zap.String("handler_url", cfg.HandlerURL),
		)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: engine,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("Server starting", zap.String("addr", server.Addr))
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("Server failed", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Server shutting down")

			if repo != nil {
				repo.Close()
			}
			if cacheClient != nil {
				_ = cacheClient.Close()
			}

			return server.Shutdown(ctx)
		},
	})

	return nil
}
