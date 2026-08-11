package geofence

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// RuntimeMode is the hierarchical geofence operating mode.
type RuntimeMode string

const (
	RuntimeModeOff     RuntimeMode = "off"
	RuntimeModeObserve RuntimeMode = "observe"
	RuntimeModeEnforce RuntimeMode = "enforce"
)

type CarrierSetting struct {
	Carrier                     string      `json:"carrier"`
	Mode                        RuntimeMode `json:"mode"`
	EffectiveMode               RuntimeMode `json:"effective_mode"`
	DefaultBaselineRadiusMeters float64     `json:"default_baseline_radius_meters"`
	UpdatedBy                   *uuid.UUID  `json:"updated_by,omitempty"`
	UpdatedAt                   time.Time   `json:"updated_at"`
}

type Settings struct {
	SystemMode RuntimeMode      `json:"system_mode"`
	Carriers   []CarrierSetting `json:"carriers"`
}

// Availability is the minimal authenticated feature-discovery contract used
// by the GIS page. It intentionally exposes no carrier settings: the system
// switch controls whether the geofence entry exists at all, while API and
// button permissions continue to protect the management surface.
type Availability struct {
	Enabled bool `json:"enabled"`
}

func EffectiveRuntimeMode(systemMode, carrierMode RuntimeMode) RuntimeMode {
	systemRank, systemOK := runtimeModeRank(systemMode)
	carrierRank, carrierOK := runtimeModeRank(carrierMode)
	if !systemOK || !carrierOK {
		return RuntimeModeOff
	}
	if systemRank <= carrierRank {
		return systemMode
	}
	return carrierMode
}

func runtimeModeRank(mode RuntimeMode) (int, bool) {
	switch mode {
	case RuntimeModeOff:
		return 0, true
	case RuntimeModeObserve:
		return 1, true
	case RuntimeModeEnforce:
		return 2, true
	default:
		return 0, false
	}
}

func ValidateSettings(settings Settings) error {
	if _, ok := runtimeModeRank(settings.SystemMode); !ok {
		return fmt.Errorf("invalid geofence system mode %q: %w",
			settings.SystemMode, commonerrors.ErrInvalidInput)
	}
	if len(settings.Carriers) == 0 {
		return fmt.Errorf("at least one carrier setting is required: %w",
			commonerrors.ErrInvalidInput)
	}

	seen := make(map[string]struct{}, len(settings.Carriers))
	for _, setting := range settings.Carriers {
		carrier := strings.ToLower(strings.TrimSpace(setting.Carrier))
		if carrier == "" {
			return fmt.Errorf("carrier is required: %w", commonerrors.ErrInvalidInput)
		}
		if _, duplicate := seen[carrier]; duplicate {
			return fmt.Errorf("duplicate carrier %q: %w",
				carrier, commonerrors.ErrInvalidInput)
		}
		seen[carrier] = struct{}{}

		if _, ok := runtimeModeRank(setting.Mode); !ok {
			return fmt.Errorf("invalid geofence mode %q for carrier %q: %w",
				setting.Mode, carrier, commonerrors.ErrInvalidInput)
		}
		if setting.DefaultBaselineRadiusMeters <= 0 ||
			setting.DefaultBaselineRadiusMeters > 50000 {
			return fmt.Errorf(
				"default baseline radius for carrier %q must be within (0, 50000]: %w",
				carrier,
				commonerrors.ErrInvalidInput,
			)
		}
	}
	return nil
}
