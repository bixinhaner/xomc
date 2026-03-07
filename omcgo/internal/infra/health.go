package infra

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ComponentHealth represents the health status of a single component.
type ComponentHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // healthy, unhealthy, degraded
	Latency string `json:"latency"`
	Error   string `json:"error,omitempty"`
}

type healthCheck struct {
	name string
	fn   func(ctx context.Context) error
}

// HealthChecker aggregates health checks from multiple components.
type HealthChecker struct {
	mu     sync.RWMutex
	checks []healthCheck
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// Register adds a health check function for a named component.
func (h *HealthChecker) Register(name string, check func(ctx context.Context) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, healthCheck{name: name, fn: check})
}

// CheckAll runs all registered health checks concurrently and returns results.
func (h *HealthChecker) CheckAll(ctx context.Context) []ComponentHealth {
	h.mu.RLock()
	checks := make([]healthCheck, len(h.checks))
	copy(checks, h.checks)
	h.mu.RUnlock()

	results := make([]ComponentHealth, len(checks))
	var wg sync.WaitGroup

	for i, check := range checks {
		wg.Add(1)
		go func(idx int, c healthCheck) {
			defer wg.Done()
			start := time.Now()
			err := c.fn(ctx)
			latency := time.Since(start)

			result := ComponentHealth{
				Name:    c.name,
				Status:  "healthy",
				Latency: latency.String(),
			}
			if err != nil {
				result.Status = "unhealthy"
				result.Error = err.Error()
			}
			results[idx] = result
		}(i, check)
	}

	wg.Wait()
	return results
}

// IsHealthy returns true if all components are healthy.
func (h *HealthChecker) IsHealthy(ctx context.Context) bool {
	for _, result := range h.CheckAll(ctx) {
		if result.Status != "healthy" {
			return false
		}
	}
	return true
}

// StatusCode returns 200 if healthy, 503 if unhealthy.
func (h *HealthChecker) StatusCode(ctx context.Context) int {
	if h.IsHealthy(ctx) {
		return 200
	}
	return 503
}

// String returns a human-readable summary.
func (h *HealthChecker) String(ctx context.Context) string {
	results := h.CheckAll(ctx)
	out := "Health Check Results:\n"
	for _, r := range results {
		status := "OK"
		if r.Status != "healthy" {
			status = fmt.Sprintf("FAIL (%s)", r.Error)
		}
		out += fmt.Sprintf("  %-15s %s [%s]\n", r.Name, status, r.Latency)
	}
	return out
}
