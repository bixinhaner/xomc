package alarm

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// SharedEmailTransport is the temporary compatibility seam used while legacy
// notify_email rules are migrated. The transport is owned by notification and
// uses the same app configuration as the reliable delivery worker.
type SharedEmailTransport interface {
	Send(context.Context, []string, string, string) error
}

type SharedEmailDispatcher struct {
	transport SharedEmailTransport
	logger    *zap.Logger
	metrics   *EmailMetrics
}

func NewSharedEmailDispatcher(
	transport SharedEmailTransport,
	logger *zap.Logger,
	metrics *EmailMetrics,
) *SharedEmailDispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SharedEmailDispatcher{transport: transport, logger: logger, metrics: metrics}
}

func (d *SharedEmailDispatcher) Dispatch(ctx context.Context, to []string, subject, body string) error {
	if d == nil || d.transport == nil {
		return fmt.Errorf("dispatch legacy alarm email: shared transport is required")
	}
	if err := validateEmailParams("shared-transport", to, subject); err != nil {
		d.recordFailure()
		return err
	}
	if err := d.transport.Send(ctx, to, subject, body); err != nil {
		d.recordFailure()
		d.logger.Warn("legacy alarm email dispatch failed", zap.Error(err))
		return fmt.Errorf("dispatch legacy alarm email through notification transport: %w", err)
	}
	d.recordSuccess()
	return nil
}

func (d *SharedEmailDispatcher) recordSuccess() {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues("success").Inc()
	}
}

func (d *SharedEmailDispatcher) recordFailure() {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues("failure").Inc()
	}
}
