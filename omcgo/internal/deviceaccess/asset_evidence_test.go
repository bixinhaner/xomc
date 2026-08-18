package deviceaccess

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAssetLookup struct {
	device            *AssetDeviceRecord
	registration      *AssetRegistrationRecord
	otherCarrierFound bool
	err               error
}

func (f *fakeAssetLookup) FindDevice(context.Context, string, string) (*AssetDeviceRecord, error) {
	return f.device, f.err
}

func (f *fakeAssetLookup) FindRegistration(context.Context, string, string) (*AssetRegistrationRecord, error) {
	return f.registration, f.err
}

func (f *fakeAssetLookup) ExistsUnderOtherCarrier(context.Context, string, string) (bool, error) {
	return f.otherCarrierFound, f.err
}

func TestAssetEvidenceResolverResolvesCurrentArchitectureSources(t *testing.T) {
	deviceID := uuid.New()
	registrationID := uuid.New()
	groupID := uuid.New()
	lastInform := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		lookup           *fakeAssetLookup
		wantSource       AssetEvidenceSource
		wantDeviceID     *uuid.UUID
		wantRegistration *uuid.UUID
	}{
		{
			name: "formal device with inform history",
			lookup: &fakeAssetLookup{device: &AssetDeviceRecord{
				ID: deviceID, Carrier: "cmcc", SerialNumber: "SN-001",
				GroupID: &groupID, SiteID: "SITE-1", LifecycleState: "commissioned",
				IsOnline: true, LastInformAt: &lastInform,
			}},
			wantSource:   AssetEvidenceSourceDevice,
			wantDeviceID: &deviceID,
		},
		{
			name: "batch preregistered device remains formal registered offline asset",
			lookup: &fakeAssetLookup{device: &AssetDeviceRecord{
				ID: deviceID, Carrier: "cmcc", SerialNumber: "SN-002",
				LifecycleState: "registered", IsOnline: false, LastInformAt: nil,
			}},
			wantSource:   AssetEvidenceSourceDevice,
			wantDeviceID: &deviceID,
		},
		{
			name: "legacy registration remains a separate asset source",
			lookup: &fakeAssetLookup{registration: &AssetRegistrationRecord{
				ID: registrationID, Carrier: "cmcc", SerialNumber: "SN-003",
				GroupID: &groupID, SiteName: "site-3", Status: "pending",
			}},
			wantSource:       AssetEvidenceSourceRegistration,
			wantRegistration: &registrationID,
		},
		{
			name: "linked device and registration are unified without merging tables",
			lookup: &fakeAssetLookup{
				device: &AssetDeviceRecord{
					ID: deviceID, Carrier: "cmcc", SerialNumber: "SN-004",
					GroupID: &groupID, SiteID: "SITE-4", LifecycleState: "registered",
				},
				registration: &AssetRegistrationRecord{
					ID: registrationID, Carrier: "cmcc", SerialNumber: "SN-004",
					DeviceID: &deviceID, GroupID: &groupID, SiteName: "SITE-4", Status: "pending",
				},
			},
			wantSource:       AssetEvidenceSourceDeviceAndRegistration,
			wantDeviceID:     &deviceID,
			wantRegistration: &registrationID,
		},
		{
			name:       "completely unknown identity has no asset evidence",
			lookup:     &fakeAssetLookup{},
			wantSource: AssetEvidenceSourceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewAssetEvidenceResolver(tt.lookup)

			got, err := resolver.Resolve(context.Background(), "cmcc", serialForLookup(tt.lookup))

			require.NoError(t, err)
			assert.Equal(t, tt.wantSource, got.Source)
			assert.Equal(t, tt.wantDeviceID, got.DeviceID)
			assert.Equal(t, tt.wantRegistration, got.RegistrationID)
		})
	}
}

func TestAssetEvidenceResolverRejectsOwnershipConflicts(t *testing.T) {
	deviceID := uuid.New()
	otherDeviceID := uuid.New()
	groupA := uuid.New()
	groupB := uuid.New()

	tests := []struct {
		name   string
		lookup *fakeAssetLookup
	}{
		{
			name: "same serial exists under another carrier",
			lookup: &fakeAssetLookup{
				otherCarrierFound: true,
			},
		},
		{
			name: "registration links another formal device",
			lookup: &fakeAssetLookup{
				device: &AssetDeviceRecord{
					ID: deviceID, Carrier: "cmcc", SerialNumber: "SN-CONFLICT",
				},
				registration: &AssetRegistrationRecord{
					ID: uuid.New(), Carrier: "cmcc", SerialNumber: "SN-CONFLICT", DeviceID: &otherDeviceID,
				},
			},
		},
		{
			name: "device and registration claim different groups",
			lookup: &fakeAssetLookup{
				device: &AssetDeviceRecord{
					ID: deviceID, Carrier: "cmcc", SerialNumber: "SN-CONFLICT", GroupID: &groupA,
				},
				registration: &AssetRegistrationRecord{
					ID: uuid.New(), Carrier: "cmcc", SerialNumber: "SN-CONFLICT",
					DeviceID: &deviceID, GroupID: &groupB,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewAssetEvidenceResolver(tt.lookup)

			_, err := resolver.Resolve(context.Background(), "cmcc", "SN-CONFLICT")

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrAssetOwnershipConflict)
		})
	}
}

func TestAssetEvidenceResolverWrapsLookupFailure(t *testing.T) {
	resolver := NewAssetEvidenceResolver(&fakeAssetLookup{err: errors.New("database unavailable")})

	_, err := resolver.Resolve(context.Background(), "cmcc", "SN-ERR")

	require.Error(t, err)
	assert.ErrorContains(t, err, "resolve asset evidence")
	assert.ErrorContains(t, err, "database unavailable")
}

func TestAssetEvidenceResolverProjectsIdentityAndRetirement(t *testing.T) {
	deviceID := uuid.New()
	resolver := NewAssetEvidenceResolver(&fakeAssetLookup{device: &AssetDeviceRecord{
		ID: deviceID, Carrier: "cmcc", SerialNumber: "SN-RETIRED",
		OUI: "48BF74", ProductClass: "FAP/BU1810", SoftwareVersion: "BM_2.0.4",
		LifecycleState: "decommissioned",
	}})

	evidence, err := resolver.Resolve(context.Background(), "cmcc", "SN-RETIRED")

	require.NoError(t, err)
	assert.Equal(t, "48BF74", evidence.ExpectedOUI)
	assert.Equal(t, "FAP/BU1810", evidence.ExpectedProductClass)
	assert.Equal(t, "BM_2.0.4", evidence.ExpectedSoftwareVersion)
	assert.True(t, evidence.AssetRetired)
}

func TestAssetEvidenceResolverTreatsExpiredRegistrationAsRetired(t *testing.T) {
	resolver := NewAssetEvidenceResolver(&fakeAssetLookup{registration: &AssetRegistrationRecord{
		ID: uuid.New(), Carrier: "cmcc", SerialNumber: "SN-EXPIRED", Status: "expired",
	}})

	evidence, err := resolver.Resolve(context.Background(), "cmcc", "SN-EXPIRED")

	require.NoError(t, err)
	assert.True(t, evidence.AssetRetired)
}

func serialForLookup(lookup *fakeAssetLookup) string {
	if lookup.device != nil {
		return lookup.device.SerialNumber
	}
	if lookup.registration != nil {
		return lookup.registration.SerialNumber
	}
	return "SN-UNKNOWN"
}
