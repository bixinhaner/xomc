package indicator

import (
	"context"
	"fmt"
	"slices"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

var ErrDisabledIndicatorSelected = fmt.Errorf("%w: disabled indicators selected", commonerrors.ErrInvalidInput)

func ValidateEnabledIndicatorSelection(ctx context.Context, repo EnabledIndicatorRepository, dt DeviceType, operatorCode string, indicatorIDs []string) error {
	if repo == nil || len(indicatorIDs) == 0 {
		return nil
	}
	enabled, err := repo.List(ctx, dt, operatorCode)
	if err != nil {
		return fmt.Errorf("list enabled indicators: %w", err)
	}
	enabledSet := make(map[string]struct{}, len(enabled))
	for _, id := range enabled {
		enabledSet[id] = struct{}{}
	}
	disabled := disabledIndicatorIDs(indicatorIDs, enabledSet)
	if len(disabled) > 0 {
		return fmt.Errorf("%w: %v", ErrDisabledIndicatorSelected, disabled)
	}
	return nil
}

func ValidateEnabledIndicatorSelectionAnyDeviceType(ctx context.Context, repo EnabledIndicatorRepository, deviceTypes []DeviceType, operatorCode string, indicatorIDs []string) error {
	if repo == nil || len(indicatorIDs) == 0 {
		return nil
	}
	enabledSet := map[string]struct{}{}
	for _, dt := range deviceTypes {
		enabled, err := repo.List(ctx, dt, operatorCode)
		if err != nil {
			return fmt.Errorf("list enabled indicators for %s: %w", dt, err)
		}
		for _, id := range enabled {
			enabledSet[id] = struct{}{}
		}
	}
	disabled := disabledIndicatorIDs(indicatorIDs, enabledSet)
	if len(disabled) > 0 {
		return fmt.Errorf("%w: %v", ErrDisabledIndicatorSelected, disabled)
	}
	return nil
}

func disabledIndicatorIDs(indicatorIDs []string, enabledSet map[string]struct{}) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, id := range indicatorIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := enabledSet[id]; !ok {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}
