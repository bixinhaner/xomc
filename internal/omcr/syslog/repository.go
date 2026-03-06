package syslog

import (
	"context"

	"github.com/omcgo/omcgo/internal/common/model"
)

// SyslogRepository defines the interface for system log and NE message log persistence.
type SyslogRepository interface {
	ListSystemLogs(ctx context.Context, filter SystemLogFilter) (*model.ListResponse[SystemLog], error)
	ListNEMessageLogs(ctx context.Context, filter NEMessageLogFilter) (*model.ListResponse[NEMessageLog], error)
}
