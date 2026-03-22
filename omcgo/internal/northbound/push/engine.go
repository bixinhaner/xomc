package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"go.uber.org/zap"
)

// Target is the runtime representation of a push target.
type Target struct {
	ID             string   `json:"id"`
	URL            string   `json:"url"`
	AuthType       string   `json:"auth_type"`
	AuthToken      string   `json:"auth_token,omitempty"`
	DataTypes      []string `json:"data_types"`
	Format         string   `json:"format"`
	BatchSize      int      `json:"batch_size"`
	RetryCount     int      `json:"retry_count"`
	Enabled        bool     `json:"enabled"`
	SigningEnabled bool     `json:"signing_enabled"`
	SigningSecret  string   `json:"-"` // never expose in API responses
}

// Engine manages push targets and delivers events to external systems via HTTP.
type Engine struct {
	mu              sync.RWMutex
	targets         map[string]*Target
	circuitBreakers map[string]*reliability.CircuitBreaker
	client          *http.Client
	logger          *zap.Logger
	subs            []event.Subscription
	outboxRepo      OutboxRepository // nil means direct delivery (no outbox)
}

// NewEngine creates a new push Engine with initial targets loaded from config.
func NewEngine(cfgTargets []appconfig.PushTargetConfig, logger *zap.Logger) *Engine {
	e := &Engine{
		targets:         make(map[string]*Target),
		circuitBreakers: make(map[string]*reliability.CircuitBreaker),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}

	for _, ct := range cfgTargets {
		t := targetFromConfig(ct)
		e.targets[t.ID] = t
		e.circuitBreakers[t.ID] = reliability.NewCircuitBreaker(reliability.DefaultCircuitBreakerConfig())
	}

	return e
}

// SetOutboxRepo sets the outbox repository, enabling the outbox delivery pattern.
// When set, events are written to the outbox table and delivered by the OutboxWorker.
func (e *Engine) SetOutboxRepo(repo OutboxRepository) {
	e.outboxRepo = repo
}

// Subscribe registers event handlers for northbound/OSS events.
// When outbox is configured, events are enqueued rather than delivered directly.
func (e *Engine) Subscribe(eventBus event.EventBus) error {
	handler := e.handleEvent
	if e.outboxRepo != nil {
		handler = e.EnqueueEvent
	}

	subjects := []string{
		event.SubjectOSSAlarmForward,
		event.SubjectOSSPMExport,
		event.SubjectOSSConfigSnapshot,
	}

	for _, subject := range subjects {
		sub, err := eventBus.Subscribe(subject, handler)
		if err != nil {
			return fmt.Errorf("subscribe to %s: %w", subject, err)
		}
		e.subs = append(e.subs, sub)
	}

	mode := "direct"
	if e.outboxRepo != nil {
		mode = "outbox"
	}
	e.logger.Info("push engine subscribed to OSS events",
		zap.Int("targets", len(e.targets)),
		zap.String("mode", mode))
	return nil
}

// Close unsubscribes from all events.
func (e *Engine) Close() error {
	for _, sub := range e.subs {
		if err := sub.Unsubscribe(); err != nil {
			e.logger.Warn("unsubscribe error", zap.Error(err))
		}
	}
	e.subs = nil
	return nil
}

// AddTarget adds or updates a push target.
func (e *Engine) AddTarget(t *Target) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.targets[t.ID] = t
	if _, ok := e.circuitBreakers[t.ID]; !ok {
		e.circuitBreakers[t.ID] = reliability.NewCircuitBreaker(reliability.DefaultCircuitBreakerConfig())
	}
	e.logger.Info("push target added", zap.String("id", t.ID), zap.String("url", t.URL))
}

// RemoveTarget removes a push target by ID.
func (e *Engine) RemoveTarget(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.targets[id]; !ok {
		return false
	}
	delete(e.targets, id)
	delete(e.circuitBreakers, id)
	e.logger.Info("push target removed", zap.String("id", id))
	return true
}

// ListTargets returns a snapshot of all push targets.
func (e *Engine) ListTargets() []*Target {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*Target, 0, len(e.targets))
	for _, t := range e.targets {
		cp := *t
		result = append(result, &cp)
	}
	return result
}

// GetTarget returns a push target by ID or nil if not found.
func (e *Engine) GetTarget(id string) *Target {
	e.mu.RLock()
	defer e.mu.RUnlock()
	t, ok := e.targets[id]
	if !ok {
		return nil
	}
	cp := *t
	return &cp
}

// GetCircuitBreaker returns the circuit breaker for a target, or nil if not found.
func (e *Engine) GetCircuitBreaker(targetID string) *reliability.CircuitBreaker {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.circuitBreakers[targetID]
}

// handleEvent dispatches an incoming event to all matching push targets.
func (e *Engine) handleEvent(ctx context.Context, evt event.Event) error {
	dataType := dataTypeFromSubject(evt.Subject)
	if dataType == "" {
		return nil
	}

	e.mu.RLock()
	targets := make([]*Target, 0)
	for _, t := range e.targets {
		if t.Enabled && matchesDataType(t.DataTypes, dataType) {
			cp := *t
			targets = append(targets, &cp)
		}
	}
	e.mu.RUnlock()

	for _, t := range targets {
		cb := e.GetCircuitBreaker(t.ID)
		if cb != nil {
			if err := cb.Allow(); err != nil {
				e.logger.Warn("push skipped: circuit breaker open",
					zap.String("target_id", t.ID),
					zap.String("subject", evt.Subject))
				continue
			}
		}

		if err := e.deliver(ctx, t, evt); err != nil {
			if cb != nil {
				cb.RecordFailure()
			}
			e.logger.Error("push delivery failed",
				zap.String("target_id", t.ID),
				zap.String("subject", evt.Subject),
				zap.Error(err))
		} else {
			if cb != nil {
				cb.RecordSuccess()
			}
		}
	}

	return nil
}

// deliver sends the event payload to a push target via HTTP POST with retry.
func (e *Engine) deliver(ctx context.Context, t *Target, evt event.Event) error {
	body, err := json.Marshal(map[string]interface{}{
		"event_id":  evt.ID,
		"subject":   evt.Subject,
		"payload":   json.RawMessage(evt.Payload),
		"timestamp": evt.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("marshal push payload: %w", err)
	}

	retries := t.RetryCount
	if retries < 1 {
		retries = 1
	}

	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, ...
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build push request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		switch t.AuthType {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+t.AuthToken)
		case "basic":
			req.Header.Set("Authorization", "Basic "+t.AuthToken)
		}

		// HMAC-SHA256 webhook signing for payload integrity and replay protection
		if t.SigningEnabled && t.SigningSecret != "" {
			ts := time.Now().Unix()
			sig := SignPayload(t.SigningSecret, ts, body)
			req.Header.Set("X-Webhook-Signature", "sha256="+sig)
			req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", ts))
		}

		resp, err := e.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("push HTTP request (attempt %d): %w", attempt+1, err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			e.logger.Debug("push delivered",
				zap.String("target_id", t.ID),
				zap.String("subject", evt.Subject),
				zap.Int("status", resp.StatusCode))
			return nil
		}

		lastErr = fmt.Errorf("push HTTP response (attempt %d): status %d", attempt+1, resp.StatusCode)
	}

	return fmt.Errorf("push delivery exhausted retries: %w", lastErr)
}

func dataTypeFromSubject(subject string) string {
	switch subject {
	case event.SubjectOSSAlarmForward:
		return "alarm"
	case event.SubjectOSSPMExport:
		return "pm"
	case event.SubjectOSSConfigSnapshot:
		return "config"
	default:
		return ""
	}
}

func matchesDataType(types []string, dt string) bool {
	for _, t := range types {
		if t == dt {
			return true
		}
	}
	return false
}

func targetFromConfig(ct appconfig.PushTargetConfig) *Target {
	return &Target{
		ID:             ct.ID,
		URL:            ct.URL,
		AuthType:       ct.AuthType,
		AuthToken:      ct.AuthToken,
		DataTypes:      ct.DataTypes,
		Format:         ct.Format,
		BatchSize:      ct.BatchSize,
		RetryCount:     ct.RetryCount,
		Enabled:        ct.Enabled,
		SigningEnabled: ct.SigningEnabled,
		SigningSecret:  ct.SigningSecret,
	}
}
