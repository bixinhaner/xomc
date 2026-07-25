package querytemplate

import (
	"context"
	"fmt"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/pm/indicator"
)

// EnabledMetricPayloadService owns the business rule that saved query
// templates may only reference indicators enabled for the default operator.
type EnabledMetricPayloadService struct {
	repo indicator.EnabledIndicatorRepository
}

func NewEnabledMetricPayloadService(repo indicator.EnabledIndicatorRepository) *EnabledMetricPayloadService {
	if repo == nil {
		return nil
	}
	return &EnabledMetricPayloadService{repo: repo}
}

func (s *EnabledMetricPayloadService) ValidatePayload(ctx context.Context, payload []byte) error {
	if s == nil || s.repo == nil {
		return nil
	}
	metricPaths, err := payloadStringArray(payload, "metric_paths")
	if err != nil {
		return err
	}
	if len(metricPaths) == 0 {
		return nil
	}
	deviceType, err := payloadString(payload, "device_type")
	if err != nil {
		return err
	}
	if deviceType != "" {
		dt, err := indicator.ParseDeviceType(deviceType)
		if err != nil {
			return fmt.Errorf("%w: invalid device_type", commonerrors.ErrInvalidInput)
		}
		return indicator.ValidateEnabledIndicatorSelection(ctx, s.repo, dt, "default", metricPaths)
	}
	return indicator.ValidateEnabledIndicatorSelectionAnyDeviceType(ctx, s.repo,
		[]indicator.DeviceType{indicator.DeviceTypeENB, indicator.DeviceTypeGNB, indicator.DeviceTypeGSM},
		"default", metricPaths)
}
