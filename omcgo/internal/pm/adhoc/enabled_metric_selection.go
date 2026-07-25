package adhoc

import (
	"context"
	"fmt"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/pm/indicator"
)

// EnabledMetricSelectionService owns the business rule that adhoc tasks may
// only use indicators enabled for the default operator.
type EnabledMetricSelectionService struct {
	repo indicator.EnabledIndicatorRepository
}

func NewEnabledMetricSelectionService(repo indicator.EnabledIndicatorRepository) *EnabledMetricSelectionService {
	if repo == nil {
		return nil
	}
	return &EnabledMetricSelectionService{repo: repo}
}

func (s *EnabledMetricSelectionService) ValidateTechnology(ctx context.Context, tech string, metricPaths []string) error {
	if s == nil || s.repo == nil || len(metricPaths) == 0 {
		return nil
	}
	if tech != "" {
		dt, err := indicatorDeviceTypeForTechnology(tech)
		if err != nil {
			return err
		}
		return indicator.ValidateEnabledIndicatorSelection(ctx, s.repo, dt, "default", metricPaths)
	}
	return indicator.ValidateEnabledIndicatorSelectionAnyDeviceType(ctx, s.repo,
		[]indicator.DeviceType{indicator.DeviceTypeENB, indicator.DeviceTypeGNB, indicator.DeviceTypeGSM},
		"default", metricPaths)
}

func indicatorDeviceTypeForTechnology(tech string) (indicator.DeviceType, error) {
	switch tech {
	case "lte":
		return indicator.DeviceTypeENB, nil
	case "nr":
		return indicator.DeviceTypeGNB, nil
	case "gsm":
		return indicator.DeviceTypeGSM, nil
	default:
		return "", fmt.Errorf("%w: unsupported technology %q", commonerrors.ErrInvalidInput, tech)
	}
}
