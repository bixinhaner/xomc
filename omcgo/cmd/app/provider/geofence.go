package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/geofence"
)

type thirdPartyLocationStore struct {
	devices      device.DeviceReader
	observations device.LocationObservationRepository
}

type geofenceCandidateReader struct {
	devices device.DeviceReader
}

func (r geofenceCandidateReader) ListGeofenceCandidates(
	ctx context.Context,
	carrierCode string,
	visibleGroups []uuid.UUID,
) ([]geofence.CandidateDevice, error) {
	carrier := model.CarrierCode(carrierCode)
	items := make([]geofence.CandidateDevice, 0)
	for page := 1; ; page++ {
		result, err := r.devices.List(ctx, device.DeviceFilter{
			Carrier:       &carrier,
			VisibleGroups: visibleGroups,
			ListRequest:   model.ListRequest{Page: page, PageSize: 1000},
		})
		if err != nil {
			return nil, fmt.Errorf("list devices for geofence candidates: %w", err)
		}
		for _, item := range result.Items {
			items = append(items, geofence.CandidateDevice{
				ID: item.ID, SerialNumber: item.SerialNumber, Name: item.DeviceName,
				Latitude: item.Latitude, Longitude: item.Longitude,
			})
		}
		if len(result.Items) == 0 || len(items) >= int(result.Total) {
			return items, nil
		}
	}
}

func (s thirdPartyLocationStore) SaveThirdPartyLocation(
	ctx context.Context,
	item geofence.ThirdPartyLocationDevice,
	observedAt time.Time,
	longitude float64,
	latitude float64,
) error {
	dev, err := s.devices.GetBySerialNumber(ctx, item.SerialNumber)
	if err != nil {
		return fmt.Errorf("device lookup failed")
	}
	if dev == nil {
		return fmt.Errorf("device not found in OMC system")
	}
	_, err = s.observations.SaveLatestWithOutbox(
		ctx,
		dev.ID,
		device.ReportedLocation{
			Latitude:   latitude,
			Longitude:  longitude,
			ObservedAt: observedAt,
			ReceivedAt: time.Now().UTC(),
			SourcePath: "third_party:fence/batchUpdateDeviceLocation",
		},
	)
	if err != nil {
		if errors.Is(err, device.ErrStaleLocationObservation) {
			return fmt.Errorf("stale updateTime")
		}
		if errors.Is(err, device.ErrLocationSourceNotAllowed) {
			return fmt.Errorf("device is not configured for external positioning")
		}
		return fmt.Errorf("location update failed")
	}
	return nil
}

func initGeofenceModule(c *Container) error {
	repository := geofence.NewPgRepository(c.PgPool)
	settingsRepository := geofence.NewPgSettingsRepository(
		storage.NewPoolDB(c.PgPool),
	)
	service := geofence.NewService(
		repository,
		settingsRepository,
	)
	service.SetThirdPartyLocationTimezoneProvider(c.SystemTimezone)
	service.SetThirdPartyLocationStore(thirdPartyLocationStore{
		devices:      c.DeviceRepo,
		observations: device.NewPgLocationObservationRepository(c.PgPool),
	})
	service.SetThirdPartyLocationBatchRepository(
		geofence.NewPgThirdPartyLocationBatchRepository(c.PgPool),
	)
	service.SetGeofenceCandidateReader(geofenceCandidateReader{devices: c.DeviceRepo})
	service.SetGeofenceControlActionReader(
		geofence.NewPgControlActionRepository(c.PgPool),
	)
	c.GeofenceService = service
	handler := geofence.NewHandler(service, c.Logger.Named("geofence"))
	if c.PermService != nil {
		handler.SetPermissionService(c.PermService)
	}
	c.GeofenceHandler = handler
	return nil
}
