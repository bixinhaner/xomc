package device

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetAntennaSectors_UsesAntennaParameterGroup(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := newFakeDeviceRepo()
	deviceRepo.devices[deviceID] = &model.Device{ID: deviceID}
	paramRepo := newFakeParamRepo()
	service := NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())

	paramRepo.groupParams[deviceID] = map[string][]model.DeviceParameter{"antenna": {
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.AntennaInfo.Azimuth", ParameterValue: "120"},
	}}

	sectors, err := service.GetAntennaSectors(context.Background(), deviceID)
	require.NoError(t, err)
	require.Len(t, sectors, 1)
	require.NotNil(t, sectors[0].Azimuth)
	require.Equal(t, 120.0, *sectors[0].Azimuth)
	require.True(t, sectors[0].DirectionAvailable)
	require.Equal(t, "antenna", paramRepo.lastGroup)
}

type memoryAntennaPlanRepo struct {
	plans []AntennaSectorPlan
}

func (r *memoryAntennaPlanRepo) ListByDevice(_ context.Context, deviceID uuid.UUID) ([]AntennaSectorPlan, error) {
	result := make([]AntennaSectorPlan, 0, len(r.plans))
	for _, plan := range r.plans {
		if plan.DeviceID == deviceID {
			result = append(result, plan)
		}
	}
	return result, nil
}

func (r *memoryAntennaPlanRepo) Upsert(_ context.Context, plan AntennaSectorPlan) error {
	for i := range r.plans {
		if r.plans[i].DeviceID == plan.DeviceID && r.plans[i].SectorNumber == plan.SectorNumber {
			r.plans[i] = plan
			return nil
		}
	}
	r.plans = append(r.plans, plan)
	return nil
}

func TestUpdateAntennaSectorPlan_SavesLocallyAndRecalculates(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := newFakeDeviceRepo()
	deviceRepo.devices[deviceID] = &model.Device{ID: deviceID}
	paramRepo := newFakeParamRepo()
	planRepo := &memoryAntennaPlanRepo{}
	service := NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())
	service.SetAntennaSectorPlanRepository(planRepo)

	azimuth, height, downtilt := 120.0, 18.0, 6.0
	horizontal, vertical := 65.0, 8.0
	sector, err := service.UpdateAntennaSectorPlan(context.Background(), deviceID, 1, UpdateAntennaSectorPlanRequest{
		Azimuth:             &azimuth,
		AntennaHeight:       &height,
		MechanicalDowntilt:  &downtilt,
		HorizontalBeamwidth: &horizontal,
		VerticalBeamwidth:   &vertical,
	})

	require.NoError(t, err)
	require.True(t, sector.CoverageAvailable)
	require.Equal(t, 120.0, *sector.Azimuth)
	require.Len(t, planRepo.plans, 1)
}
