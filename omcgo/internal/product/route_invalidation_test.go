package product

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestProductRouteFieldsChanged_IgnoresDisplayFields(t *testing.T) {
	before := &Product{
		Name:                "old-name",
		Description:         "old description",
		IndicatorDeviceType: "ENB",
		IndicatorPlatform:   "BLQ",
	}
	after := &Product{
		Name:                "new-name",
		Description:         "new description",
		IndicatorDeviceType: "enb",
		IndicatorPlatform:   "BLQ",
	}

	assert.False(t, productRouteFieldsChanged(before, after),
		"display-only edits and device type case changes must not invalidate KPI route")
}

func TestProductRouteFieldsChanged_DetectsDeviceTypeAndPlatform(t *testing.T) {
	tests := []struct {
		name   string
		before Product
		after  Product
	}{
		{
			name:   "device type",
			before: Product{IndicatorDeviceType: "ENB", IndicatorPlatform: "BLQ"},
			after:  Product{IndicatorDeviceType: "GSM", IndicatorPlatform: "BLQ"},
		},
		{
			name:   "platform",
			before: Product{IndicatorDeviceType: "ENB", IndicatorPlatform: "BLQ"},
			after:  Product{IndicatorDeviceType: "ENB", IndicatorPlatform: "BSC"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, productRouteFieldsChanged(&tc.before, &tc.after))
		})
	}
}

func TestHandlerInvalidateRouteCache_InvokesInvalidatorWithTrigger(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	var got []RouteInvalidationTrigger
	h.SetRouteInvalidator(func(_ context.Context, trigger RouteInvalidationTrigger) error {
		got = append(got, trigger)
		return nil
	})

	h.invalidateRouteCache(context.Background(), RouteInvalidationTriggerProductWrite)

	require.Equal(t, []RouteInvalidationTrigger{RouteInvalidationTriggerProductWrite}, got)
}

func TestHandlerInvalidateRouteCache_ErrorDoesNotPanic(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	h.SetRouteInvalidator(func(context.Context, RouteInvalidationTrigger) error {
		return errors.New("redis down")
	})

	assert.NotPanics(t, func() {
		h.invalidateRouteCache(context.Background(), RouteInvalidationTriggerProductWrite)
	})
}

func TestRouteFieldsMapChanged(t *testing.T) {
	before := map[string]ProductRouteFields{
		"p1": normalizeProductRouteFields("ENB", "BLQ"),
		"p2": normalizeProductRouteFields("GSM", "BSC"),
	}

	assert.False(t, RouteFieldsMapChanged(before, map[string]ProductRouteFields{
		"p1": normalizeProductRouteFields("enb", "BLQ"),
		"p2": normalizeProductRouteFields("gsm", "BSC"),
	}))
	assert.True(t, RouteFieldsMapChanged(before, map[string]ProductRouteFields{
		"p1": normalizeProductRouteFields("GNB", "BLQ"),
		"p2": normalizeProductRouteFields("GSM", "BSC"),
	}))
	assert.True(t, RouteFieldsMapChanged(before, map[string]ProductRouteFields{
		"p1": normalizeProductRouteFields("ENB", "BLQ"),
	}))
}
