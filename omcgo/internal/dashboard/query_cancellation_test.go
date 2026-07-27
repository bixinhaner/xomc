package dashboard

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogDashboardQueryFailure_DoesNotWarnForRequestCancellation(t *testing.T) {
	for _, queryErr := range []error{
		context.Canceled,
		context.DeadlineExceeded,
		fmt.Errorf("query dashboard: %w", context.Canceled),
		fmt.Errorf("query dashboard: %w", context.DeadlineExceeded),
	} {
		core, logs := observer.New(zap.WarnLevel)
		logDashboardQueryFailure(zap.New(core), "dashboard query failed", queryErr)
		assert.Zero(t, logs.Len(), "request cancellation must not inflate warning logs: %v", queryErr)
	}
}

func TestLogDashboardQueryFailure_WarnsForServiceFailure(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	logDashboardQueryFailure(zap.New(core), "dashboard query failed", fmt.Errorf("database unavailable"))
	assert.Equal(t, 1, logs.Len())
}
