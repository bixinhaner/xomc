package snmp

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Engine fans out a single AlarmEvent to every enabled TrapTarget known to
// the registry. It is the only public entry point used by future production
// glue (T-0017): once alarm.Engine emits an alarm, the consumer adapter
// constructs an AlarmEvent and calls Engine.Process.
//
// In the skeleton stage Engine is constructed and exercised in tests but is
// NOT yet wired to alarm.Engine, NATS, or the REST router. See doc.go for
// the isolation contract.
type Engine struct {
	sender   Sender
	registry TargetRegistry
	mapper   AlarmMapper
	logger   *zap.Logger

	// perSendTimeout caps how long Engine.Process waits on each Sender.Send.
	// It is a defensive guardrail — Sender implementations should also honour
	// ctx — but in skeleton stage we keep both belts and braces because the
	// gosnmp UDP layer can occasionally hang on broken sockets.
	perSendTimeout time.Duration
}

// EngineOption configures Engine construction.
type EngineOption func(*Engine)

// WithMapper overrides the default mapper. T-0017 will use this hook to plug
// in a Carrier-aware mapper that delegates to internal/carrier.Carrier.
func WithMapper(m AlarmMapper) EngineOption {
	return func(e *Engine) {
		if m != nil {
			e.mapper = m
		}
	}
}

// WithPerSendTimeout overrides the default per-target send timeout (5s).
func WithPerSendTimeout(d time.Duration) EngineOption {
	return func(e *Engine) {
		if d > 0 {
			e.perSendTimeout = d
		}
	}
}

// NewEngine constructs an Engine. sender and registry are mandatory; logger
// may be nil (a no-op logger is substituted).
func NewEngine(sender Sender, registry TargetRegistry, logger *zap.Logger, opts ...EngineOption) (*Engine, error) {
	if sender == nil {
		return nil, errors.New("snmp: NewEngine requires a Sender")
	}
	if registry == nil {
		return nil, errors.New("snmp: NewEngine requires a TargetRegistry")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	e := &Engine{
		sender:         sender,
		registry:       registry,
		mapper:         DefaultAlarmMapper(),
		logger:         logger,
		perSendTimeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e, nil
}

// Process delivers alarm to every enabled target concurrently and returns one
// SendResult per attempted target. The slice is empty (not nil) when the
// registry has no enabled targets.
//
// Process never returns an error itself — per-target failures are surfaced in
// SendResult.Err so the caller can route success and failure through
// independent observability channels (Prometheus / log / outbox).
//
// Concurrency model: one goroutine per enabled target, fan-in via WaitGroup +
// channel. This keeps a slow / unreachable target from blocking healthy ones.
func (e *Engine) Process(ctx context.Context, alarm *AlarmEvent) []SendResult {
	if alarm == nil {
		return nil
	}

	targets := e.registry.ListEnabled()
	if len(targets) == 0 {
		e.logger.Debug("snmp engine: no enabled targets, skipping",
			zap.String("alarm_id", alarm.AlarmID),
		)
		return []SendResult{}
	}

	vars, err := e.mapper.MapAlarmToTrapPDU(alarm)
	if err != nil {
		e.logger.Warn("snmp engine: mapper failed, all targets reported as failed",
			zap.String("alarm_id", alarm.AlarmID),
			zap.Error(err),
		)
		results := make([]SendResult, 0, len(targets))
		for _, t := range targets {
			results = append(results, SendResult{
				TargetID: t.ID,
				OSSName:  t.OSSName,
				Success:  false,
				Err:      err,
			})
		}
		return results
	}

	resultsCh := make(chan SendResult, len(targets))
	var wg sync.WaitGroup
	wg.Add(len(targets))

	for _, t := range targets {
		go func(target *TrapTarget) {
			defer wg.Done()
			resultsCh <- e.sendOne(ctx, target, vars, alarm)
		}(t)
	}
	wg.Wait()
	close(resultsCh)

	out := make([]SendResult, 0, len(targets))
	for r := range resultsCh {
		out = append(out, r)
	}
	return out
}

// sendOne performs a single delivery attempt under a bounded timeout and
// converts the outcome to a SendResult. Errors from the Sender are passed
// through verbatim so observability layers can categorise them.
func (e *Engine) sendOne(ctx context.Context, target *TrapTarget, vars []Variable, alarm *AlarmEvent) SendResult {
	timeout := e.perSendTimeout
	if target.Timeout > 0 && target.Timeout < timeout {
		timeout = target.Timeout
	}
	sendCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := e.sender.Send(sendCtx, target, vars)
	latency := time.Since(start)

	res := SendResult{
		TargetID: target.ID,
		OSSName:  target.OSSName,
		Success:  err == nil,
		Err:      err,
		Latency:  latency,
	}

	if err != nil {
		e.logger.Warn("snmp engine: send failed",
			zap.String("alarm_id", alarm.AlarmID),
			zap.String("target_id", target.ID),
			zap.String("oss_name", target.OSSName),
			zap.Duration("latency", latency),
			zap.Error(err),
		)
	} else {
		e.logger.Debug("snmp engine: send ok",
			zap.String("alarm_id", alarm.AlarmID),
			zap.String("target_id", target.ID),
			zap.String("oss_name", target.OSSName),
			zap.Duration("latency", latency),
		)
	}
	return res
}
