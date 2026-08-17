package querytemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, serialNumber string) (*model.Device, error)
}

type DeviceScopeValidator struct {
	permissions authz.VisibleGroupsResolver
	devices     DeviceLookup
	groups      authz.GroupReader
}

func NewDeviceScopeValidator(
	permissions authz.VisibleGroupsResolver,
	devices DeviceLookup,
	groups authz.GroupReader,
) *DeviceScopeValidator {
	return &DeviceScopeValidator{permissions: permissions, devices: devices, groups: groups}
}

func (v *DeviceScopeValidator) ValidatePayload(
	ctx context.Context,
	payload []byte,
	userID uuid.UUID,
	isSuperAdmin bool,
) error {
	if v == nil || v.permissions == nil || v.devices == nil {
		return nil
	}
	var request struct {
		DeviceSNs     []string `json:"device_sns"`
		RegularReport struct {
			Enabled bool `json:"enabled"`
		} `json:"regular_report"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return fmt.Errorf("%w: parse device scope: %v", commonerrors.ErrInvalidInput, err)
	}
	if !request.RegularReport.Enabled {
		return nil
	}
	if len(request.DeviceSNs) == 0 {
		return fmt.Errorf("%w: regular report requires at least one device", commonerrors.ErrInvalidInput)
	}
	visibleGroups, err := v.permissions.GetUserVisibleGroupIDs(ctx, userID, isSuperAdmin)
	if err != nil {
		return fmt.Errorf("resolve KPI template device scope: %w", err)
	}
	seen := make(map[string]struct{}, len(request.DeviceSNs))
	for _, rawSN := range request.DeviceSNs {
		sn := strings.TrimSpace(rawSN)
		if sn == "" {
			return fmt.Errorf("%w: device_sns contains an empty value", commonerrors.ErrInvalidInput)
		}
		if _, exists := seen[sn]; exists {
			continue
		}
		seen[sn] = struct{}{}
		device, err := v.devices.GetBySerialNumber(ctx, sn)
		if err != nil || device == nil {
			return fmt.Errorf("%w: device %q is unavailable", commonerrors.ErrInvalidInput, sn)
		}
		if err := authz.AuthorizeDeviceAccess(ctx, v.groups, device.ID, visibleGroups); err != nil {
			return err
		}
	}
	return nil
}
