package syslog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// SyslogService is the consumer-facing facade for the syslog domain.
//
// The interface intentionally stays small (1-3 high-level methods per concern)
// so that downstream callers can mock it without pulling the whole repository
// surface. The concrete *Service implementation delegates to SyslogRepository.
type SyslogService interface {
	// ListSystemLogs returns a paginated list of system logs filtered by the
	// provided criteria.
	ListSystemLogs(ctx context.Context, filter SystemLogFilter) (*model.ListResponse[SystemLog], error)

	// ListNEMessageLogs returns a paginated list of network-element message
	// logs filtered by the provided criteria.
	ListNEMessageLogs(ctx context.Context, filter NEMessageLogFilter) (*model.ListResponse[NEMessageLog], error)

	// ListNEMessagesByDevice is a convenience facade that queries NE message
	// logs for a single device within an optional time window.
	ListNEMessagesByDevice(ctx context.Context, deviceID uuid.UUID, window TimeWindow, list model.ListRequest) (*model.ListResponse[NEMessageLog], error)
}

// TimeWindow represents an optional time range used by service-layer queries.
// Both fields are optional; nil means "unbounded on this side".
type TimeWindow struct {
	Start *time.Time
	End   *time.Time
}

// Validate ensures Start <= End when both are provided.
func (w TimeWindow) Validate() error {
	if w.Start != nil && w.End != nil && w.End.Before(*w.Start) {
		return fmt.Errorf("invalid time window: end before start: %w", commonerrors.ErrInvalidInput)
	}
	return nil
}

// Service is the default implementation of SyslogService backed by a
// SyslogRepository.
type Service struct {
	repo   SyslogRepository
	logger *zap.Logger
}

// NewService constructs a syslog Service. It accepts an interface (repo) and
// returns a concrete *Service following the "accept interfaces, return
// structs" idiom.
func NewService(repo SyslogRepository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:   repo,
		logger: logger.Named("syslog"),
	}
}

// ListSystemLogs delegates to the repository after light validation.
func (s *Service) ListSystemLogs(ctx context.Context, filter SystemLogFilter) (*model.ListResponse[SystemLog], error) {
	if err := validateRange(filter.StartTime, filter.EndTime); err != nil {
		return nil, err
	}

	resp, err := s.repo.ListSystemLogs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list system logs: %w", err)
	}
	return resp, nil
}

// ListNEMessageLogs delegates to the repository after light validation.
func (s *Service) ListNEMessageLogs(ctx context.Context, filter NEMessageLogFilter) (*model.ListResponse[NEMessageLog], error) {
	if err := validateRange(filter.StartTime, filter.EndTime); err != nil {
		return nil, err
	}

	resp, err := s.repo.ListNEMessageLogs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list NE message logs: %w", err)
	}
	return resp, nil
}

// ListNEMessagesByDevice queries NE message logs for a single device within an
// optional time window. The deviceID is required; uuid.Nil yields a business
// error to fail fast at the boundary.
func (s *Service) ListNEMessagesByDevice(ctx context.Context, deviceID uuid.UUID, window TimeWindow, list model.ListRequest) (*model.ListResponse[NEMessageLog], error) {
	if deviceID == uuid.Nil {
		return nil, commonerrors.NewBusinessError(
			4001,
			"device id is required",
			commonerrors.ErrInvalidInput,
		)
	}
	if err := window.Validate(); err != nil {
		return nil, err
	}

	id := deviceID
	filter := NEMessageLogFilter{
		DeviceID:    &id,
		StartTime:   window.Start,
		EndTime:     window.End,
		ListRequest: list,
	}

	resp, err := s.repo.ListNEMessageLogs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list NE messages by device %s: %w", deviceID, err)
	}
	return resp, nil
}

// validateRange ensures start <= end when both pointers are non-nil.
func validateRange(start, end *time.Time) error {
	if start != nil && end != nil && end.Before(*start) {
		return fmt.Errorf("end_time before start_time: %w", commonerrors.ErrInvalidInput)
	}
	return nil
}

// Compile-time check that *Service satisfies SyslogService.
var _ SyslogService = (*Service)(nil)
