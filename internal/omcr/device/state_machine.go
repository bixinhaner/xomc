package device

import (
	"fmt"

	"github.com/omcgo/omcgo/internal/common/model"
)

// validTransitions defines the allowed state transitions for a device.
var validTransitions = map[model.DeviceStatus][]model.DeviceStatus{
	model.DeviceDiscovered:   {model.DeviceRegistered, model.DeviceActive},
	model.DeviceRegistered:   {model.DeviceProvisioning, model.DeviceActive},
	model.DeviceProvisioning: {model.DeviceActive, model.DeviceRegistered},
	model.DeviceActive:       {model.DeviceMaintenance, model.DeviceOffline, model.DeviceDecommissioned},
	model.DeviceMaintenance:  {model.DeviceActive, model.DeviceDecommissioned},
	model.DeviceOffline:      {model.DeviceActive, model.DeviceDecommissioned},
}

// ValidateTransition checks whether a device state transition is allowed.
func ValidateTransition(current, target model.DeviceStatus) error {
	allowed, ok := validTransitions[current]
	if !ok {
		return fmt.Errorf("no transitions defined from state %q", current)
	}
	for _, s := range allowed {
		if s == target {
			return nil
		}
	}
	return fmt.Errorf("invalid transition from %q to %q", current, target)
}
