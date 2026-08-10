package geofence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type CandidateDevice struct {
	ID           uuid.UUID `json:"id"`
	SerialNumber string    `json:"serial_number"`
	Name         string    `json:"name,omitempty"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
}

type GeofenceCandidatePreview struct {
	GeofenceID uuid.UUID         `json:"geofence_id"`
	VersionID  uuid.UUID         `json:"version_id"`
	Inside     []CandidateDevice `json:"inside"`
	Outside    []CandidateDevice `json:"outside"`
	NoLocation []CandidateDevice `json:"no_location"`
}

type GeofenceCandidateReader interface {
	ListGeofenceCandidates(context.Context, string, []uuid.UUID) ([]CandidateDevice, error)
}

func (s *Service) PreviewGeofenceCandidates(
	ctx context.Context,
	geofenceID uuid.UUID,
	visibleGroups []uuid.UUID,
) (GeofenceCandidatePreview, error) {
	if geofenceID == uuid.Nil {
		return GeofenceCandidatePreview{}, fmt.Errorf("geofence is required: %w", commonerrors.ErrInvalidInput)
	}
	if s.candidateReader == nil {
		return GeofenceCandidatePreview{}, fmt.Errorf("geofence candidate reader is not configured")
	}
	definition, err := s.repository.GetDefinition(ctx, geofenceID)
	if err != nil {
		return GeofenceCandidatePreview{}, fmt.Errorf("get geofence for candidate preview: %w", err)
	}
	if definition == nil {
		return GeofenceCandidatePreview{}, commonerrors.ErrNotFound
	}
	if definition.CurrentVersionID == nil {
		return GeofenceCandidatePreview{}, fmt.Errorf("geofence has no published version: %w", commonerrors.ErrInvalidInput)
	}
	version, err := s.repository.GetVersion(ctx, *definition.CurrentVersionID)
	if err != nil {
		return GeofenceCandidatePreview{}, fmt.Errorf("get geofence version for candidate preview: %w", err)
	}
	if version == nil || version.Status != VersionStatusPublished {
		return GeofenceCandidatePreview{}, fmt.Errorf("geofence version is not published: %w", commonerrors.ErrInvalidInput)
	}
	devices, err := s.candidateReader.ListGeofenceCandidates(ctx, definition.Carrier, visibleGroups)
	if err != nil {
		return GeofenceCandidatePreview{}, fmt.Errorf("list geofence candidate devices: %w", err)
	}
	preview := GeofenceCandidatePreview{
		GeofenceID: geofenceID,
		VersionID:  version.ID,
		Inside:     make([]CandidateDevice, 0),
		Outside:    make([]CandidateDevice, 0),
		NoLocation: make([]CandidateDevice, 0),
	}
	for _, device := range devices {
		if device.Latitude == nil || device.Longitude == nil {
			preview.NoLocation = append(preview.NoLocation, device)
			continue
		}
		distance, err := signedBoundaryDistance(
			definition.RuleType,
			version.GeometryJSON,
			*device.Longitude,
			*device.Latitude,
		)
		if err != nil {
			return GeofenceCandidatePreview{}, fmt.Errorf("evaluate candidate device: %w", err)
		}
		if distance <= 0 {
			preview.Inside = append(preview.Inside, device)
		} else {
			preview.Outside = append(preview.Outside, device)
		}
	}
	return preview, nil
}
