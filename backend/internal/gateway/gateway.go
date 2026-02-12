// Package gateway provides the API gateway that routes requests to handlers.
package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/pointcloud-annotator/backend/internal/config"
)

// Gateway provides the API gateway functionality.
type Gateway struct {
	cfg        *config.Config
	logger     *zap.Logger
	httpClient *http.Client
}

// NewGateway creates a new API gateway.
func NewGateway(cfg *config.Config, logger *zap.Logger) *Gateway {
	return &Gateway{
		cfg:    cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterRoutes registers the gateway routes on the given router group.
func (g *Gateway) RegisterRoutes(rg *gin.RouterGroup) {
	// Proxy all annotation routes to the handler service
	rg.Any("/annotations", g.proxyToHandler)
	rg.Any("/annotations/*path", g.proxyToHandler)
}

// proxyToHandler forwards requests to the handler service.
func (g *Gateway) proxyToHandler(c *gin.Context) {
	// Build the target URL
	targetURL, err := url.Parse(g.cfg.HandlerURL)
	if err != nil {
		g.logger.Error("Invalid handler URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "configuration_error",
			"message": "invalid handler URL configuration",
		})
		return
	}

	// Construct the path
	path := c.Request.URL.Path
	if extraPath := c.Param("path"); extraPath != "" {
		// The path already includes /annotations, so we use the full path
	}
	targetURL.Path = path
	targetURL.RawQuery = c.Request.URL.RawQuery

	g.logger.Debug("Proxying request",
		zap.String("method", c.Request.Method),
		zap.String("target", targetURL.String()),
	)

	// Read the request body
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, err = io.ReadAll(c.Request.Body)
		if err != nil {
			g.logger.Error("Failed to read request body", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "failed to read request body",
			})
			return
		}
	}

	// Create the proxy request
	proxyReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		c.Request.Method,
		targetURL.String(),
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		g.logger.Error("Failed to create proxy request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to create proxy request",
		})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set content type if body exists
	if len(bodyBytes) > 0 && proxyReq.Header.Get("Content-Type") == "" {
		proxyReq.Header.Set("Content-Type", "application/json")
	}

	// Execute the request
	resp, err := g.httpClient.Do(proxyReq)
	if err != nil {
		g.logger.Error("Failed to proxy request", zap.Error(err))

		// Check if it's a connection error
		if strings.Contains(err.Error(), "connection refused") {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_unavailable",
				"message": "handler service is not available",
			})
			return
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "proxy_error",
			"message": "failed to reach handler service",
		})
		return
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		g.logger.Error("Failed to read response body", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to read response",
		})
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Return the response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// HealthCheck returns a health check handler.
func (g *Gateway) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"role":    g.cfg.Role,
		"service": "point-cloud-annotator",
	})
}
