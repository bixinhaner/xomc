package dashboard

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func isDashboardRequestCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func logDashboardQueryFailure(logger *zap.Logger, message string, err error, fields ...zap.Field) {
	logDashboardQueryFailureAt(logger, zap.WarnLevel, message, err, fields...)
}

func logDashboardQueryError(logger *zap.Logger, message string, err error, fields ...zap.Field) {
	logDashboardQueryFailureAt(logger, zap.ErrorLevel, message, err, fields...)
}

func logDashboardQueryFailureAt(logger *zap.Logger, level zapcore.Level, message string, err error, fields ...zap.Field) {
	if logger == nil || isDashboardRequestCancellation(err) {
		return
	}
	if checked := logger.Check(level, message); checked != nil {
		checked.Write(append(fields, zap.Error(err))...)
	}
}
