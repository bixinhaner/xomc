package geofence

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const MapBoundsQueryOrder = "minLng,maxLng,minLat,maxLat"

type MapBounds struct {
	MinLongitude float64 `json:"min_longitude"`
	MaxLongitude float64 `json:"max_longitude"`
	MinLatitude  float64 `json:"min_latitude"`
	MaxLatitude  float64 `json:"max_latitude"`
}

func (b MapBounds) Validate() error {
	if !isFiniteMapCoordinate(b.MinLongitude) ||
		!isFiniteMapCoordinate(b.MaxLongitude) ||
		!isFiniteMapCoordinate(b.MinLatitude) ||
		!isFiniteMapCoordinate(b.MaxLatitude) {
		return fmt.Errorf("map bounds must be finite: %w", commonerrors.ErrInvalidInput)
	}
	if b.MinLongitude < -180 || b.MaxLongitude > 180 ||
		b.MinLatitude < -90 || b.MaxLatitude > 90 {
		return fmt.Errorf("map bounds are outside WGS84: %w", commonerrors.ErrInvalidInput)
	}
	if b.MinLongitude >= b.MaxLongitude || b.MinLatitude >= b.MaxLatitude {
		return fmt.Errorf("map bounds must be ordered: %w", commonerrors.ErrInvalidInput)
	}
	return nil
}

func ParseMapBounds(raw string) (*MapBounds, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf(
			"bounds must use %s: %w",
			MapBoundsQueryOrder,
			commonerrors.ErrInvalidInput,
		)
	}
	values := make([]float64, len(parts))
	for index, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || !isFiniteMapCoordinate(value) {
			return nil, fmt.Errorf(
				"bounds value %d is invalid: %w",
				index,
				commonerrors.ErrInvalidInput,
			)
		}
		values[index] = value
	}
	bounds := &MapBounds{
		MinLongitude: values[0],
		MaxLongitude: values[1],
		MinLatitude:  values[2],
		MaxLatitude:  values[3],
	}
	if err := bounds.Validate(); err != nil {
		return nil, err
	}
	return bounds, nil
}

func isFiniteMapCoordinate(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

type MapDefinitionFilter struct {
	Bounds        *MapBounds
	Carrier       string
	Status        DefinitionStatus
	Name          string
	VisibleGroups []uuid.UUID
}

type MapDefinition struct {
	Definition     Definition `json:"definition"`
	CurrentVersion *Version   `json:"current_version,omitempty"`
}

type MapDefinitionList struct {
	Items []MapDefinition `json:"items"`
}
