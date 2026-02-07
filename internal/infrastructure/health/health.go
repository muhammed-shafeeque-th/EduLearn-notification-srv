package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	database "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/database/gorm"
	"go.uber.org/zap"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

type HealthCheck struct {
	Name     string       `json:"name"`
	Status   HealthStatus `json:"status"`
	Message  string       `json:"message,omitempty"`
	Duration string       `json:"duration,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Checks    map[string]HealthCheck `json:"checks"`
	Uptime    string                 `json:"uptime"`
}

type HealthChecker struct {
	db          *database.DB
	redisClient ports.Cache
	logger      *zap.Logger
	startTime   time.Time
	serviceName string
	version     string
	mu          sync.RWMutex
}

func NewHealthChecker(db *database.DB, redisClient ports.Cache, logger *zap.Logger, serviceName, version string) *HealthChecker {
	return &HealthChecker{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		startTime:   time.Now(),
		serviceName: serviceName,
		version:     version,
	}
}

func (h *HealthChecker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	h.handleHealthCheck(w, r, h.checkLiveness)
}

func (h *HealthChecker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	h.handleHealthCheck(w, r, h.checkReadiness)
}

func (h *HealthChecker) handleHealthCheck(w http.ResponseWriter, r *http.Request, checkFunc func() map[string]HealthCheck) {
	w.Header().Set("Content-Type", "application/json")

	checks := checkFunc()
	overallStatus := h.determineOverallStatus(checks)

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Service:   h.serviceName,
		Version:   h.version,
		Checks:    checks,
		Uptime:    time.Since(h.startTime).String(),
	}

	statusCode := http.StatusOK
	if overallStatus == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == StatusDegraded {
		statusCode = http.StatusOK // Degraded is still considered OK for load balancers
	}

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *HealthChecker) checkLiveness() map[string]HealthCheck {
	checks := make(map[string]HealthCheck)

	// Basic liveness check - just check if the service is running
	checks["service"] = HealthCheck{
		Name:    "service",
		Status:  StatusHealthy,
		Message: "Service is running",
	}

	return checks
}

func (h *HealthChecker) checkReadiness() map[string]HealthCheck {
	checks := make(map[string]HealthCheck)

	// Database connectivity check
	checks["database"] = h.checkDatabase()

	// Redis connectivity check
	checks["redis"] = h.checkRedis()

	// Memory usage check
	checks["memory"] = h.checkMemory()

	return checks
}

func (h *HealthChecker) checkDatabase() HealthCheck {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result int
	err := h.db.Gorm().WithContext(ctx).Raw("SELECT 1").Scan(&result).Error

	duration := time.Since(start)

	if err != nil {
		h.logger.Error("Database health check failed", zap.Error(err))
		return HealthCheck{
			Name:     "database",
			Status:   StatusUnhealthy,
			Message:  "Database connection failed",
			Duration: duration.String(),
			Error:    err.Error(),
		}
	}

	return HealthCheck{
		Name:     "database",
		Status:   StatusHealthy,
		Message:  "Database connection successful",
		Duration: duration.String(),
	}
}

func (h *HealthChecker) checkRedis() HealthCheck {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.redisClient.Ping(ctx)

	duration := time.Since(start)

	if err != nil {
		h.logger.Error("Redis health check failed", zap.Error(err))
		return HealthCheck{
			Name:     "redis",
			Status:   StatusUnhealthy,
			Message:  "Redis connection failed",
			Duration: duration.String(),
			Error:    err.Error(),
		}
	}

	return HealthCheck{
		Name:     "redis",
		Status:   StatusHealthy,
		Message:  "Redis connection successful",
		Duration: duration.String(),
	}
}

func (h *HealthChecker) checkMemory() HealthCheck {
	// This is a basic memory check - in production you might want to use runtime.ReadMemStats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Consider memory usage unhealthy if it's above 90% of system memory
	// This is a simplified check - adjust based on your requirements
	memoryUsagePercent := float64(m.Alloc) / float64(m.Sys) * 100

	if memoryUsagePercent > 90 {
		return HealthCheck{
			Name:    "memory",
			Status:  StatusDegraded,
			Message: fmt.Sprintf("High memory usage: %.2f%%", memoryUsagePercent),
		}
	}

	return HealthCheck{
		Name:    "memory",
		Status:  StatusHealthy,
		Message: fmt.Sprintf("Memory usage: %.2f%%", memoryUsagePercent),
	}
}

func (h *HealthChecker) determineOverallStatus(checks map[string]HealthCheck) HealthStatus {
	hasUnhealthy := false
	hasDegraded := false

	for _, check := range checks {
		switch check.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// StartHealthServer starts the health check HTTP server
func (h *HealthChecker) StartHealthServer(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.LivenessHandler)
	mux.HandleFunc("/ready", h.ReadinessHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	h.logger.Info("Health check server starting", zap.String("port", port))
	return server.ListenAndServe()
}
