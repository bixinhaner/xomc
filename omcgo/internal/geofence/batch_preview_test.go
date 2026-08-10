package geofence

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBindingInputsRejectsRawInputOverLimit(t *testing.T) {
	req := BindingInputRequest{DeviceSNs: make([]string, MaxBatchBindingInputs+1)}
	for i := range req.DeviceSNs {
		req.DeviceSNs[i] = fmt.Sprintf("SN-%04d", i)
	}

	_, err := normalizeBindingInputs(req)

	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestNormalizeBindingInputsTrimsAndDeduplicates(t *testing.T) {
	id := uuid.New()
	got, err := normalizeBindingInputs(BindingInputRequest{
		DeviceIDs: []uuid.UUID{id, id},
		DeviceSNs: []string{" SN001 ", "SN001", ""},
	})

	require.NoError(t, err)
	require.Equal(t, []BindingInput{
		{Key: "id:" + id.String(), Kind: BindingInputDeviceID, Value: id.String(), DeviceID: &id},
		{Key: "sn:SN001", Kind: BindingInputDeviceSN, Value: "SN001"},
	}, got)
}

func TestNormalizeBindingInputsRejectsSerialNumberLongerThanDeviceColumn(t *testing.T) {
	_, err := normalizeBindingInputs(BindingInputRequest{
		DeviceSNs: []string{strings.Repeat("S", MaxDeviceSerialNumberLength+1)},
	})

	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestNormalizeBindingInputsAcceptsMaxCharacterMultibyteSerialNumber(
	t *testing.T,
) {
	serialNumber := strings.Repeat("序", MaxDeviceSerialNumberLength)

	inputs, err := normalizeBindingInputs(BindingInputRequest{
		DeviceSNs: []string{serialNumber},
	})

	require.NoError(t, err)
	require.Equal(t, serialNumber, inputs[0].Value)
}

func TestBuildManualBindingPreviewClassifiesActiveRuleBindingAsMove(t *testing.T) {
	otherFenceID := uuid.New()
	otherBindingID := uuid.New()
	snapshot := enabledPolygonSnapshot(BindingCandidateFact{
		Input:                BindingInput{Key: "sn:SN001", Kind: BindingInputDeviceSN, Value: "SN001"},
		Device:               &DeviceIdentity{ID: uuid.New(), SerialNumber: "SN001", Carrier: "cmcc"},
		ActiveRuleBindingID:  &otherBindingID,
		ActiveRuleGeofenceID: &otherFenceID,
	})

	preview, err := buildManualBindingPreview(snapshot)

	require.NoError(t, err)
	require.Equal(t, BindingDecisionMove, preview.Items[0].Decision)
	require.Equal(t, ReasonReassigned, preview.Items[0].ReasonCode)
	require.Equal(t, otherBindingID, *preview.Items[0].SourceBindingID)
	require.Equal(t, otherFenceID, *preview.Items[0].SourceGeofenceID)
	require.Equal(t, 1, preview.MoveCount)
	require.Zero(t, preview.SkippedCount)
}

func TestManualBindingFingerprintIsOrderIndependentAndVisibilitySensitive(t *testing.T) {
	first := enabledPolygonSnapshot(eligibleFact("SN002"), eligibleFact("SN001"))
	second := enabledPolygonSnapshot(eligibleFact("SN001"), eligibleFact("SN002"))
	second.VisibilityDigest = first.VisibilityDigest

	a, err := buildManualBindingPreview(first)
	require.NoError(t, err)
	b, err := buildManualBindingPreview(second)
	require.NoError(t, err)
	require.Equal(t, a.PreviewFingerprint, b.PreviewFingerprint)

	second.VisibilityDigest = "different"
	c, err := buildManualBindingPreview(second)
	require.NoError(t, err)
	require.NotEqual(t, a.PreviewFingerprint, c.PreviewFingerprint)
}

func TestManualBindingDevicesResolvesSharedSNToDefinitionCarrierRegardlessOfUUIDOrder(t *testing.T) {
	devices := newManualBindingDevices()
	ctccID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	cmccID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	devices.add(&DeviceIdentity{ID: ctccID, SerialNumber: "SN001", Carrier: "ctcc"}, "cmcc")
	devices.add(&DeviceIdentity{ID: cmccID, SerialNumber: "SN001", Carrier: "cmcc"}, "cmcc")

	require.Equal(t, cmccID, devices.bySerialNumber["SN001"].ID)
}

func TestManualBindingDevicesKeepsDirectIDAcrossCarrierMismatch(t *testing.T) {
	devices := newManualBindingDevices()
	ctccID := uuid.New()

	devices.add(&DeviceIdentity{ID: ctccID, SerialNumber: "SN001", Carrier: "ctcc"}, "cmcc")

	require.Equal(t, ctccID, devices.byID[ctccID].ID)
}

func TestManualBindingFingerprintDistinguishesSuperadminAndEmptyGroupScopes(t *testing.T) {
	superadmin := enabledPolygonSnapshot(eligibleFact("SN001"))
	superadmin.VisibilityDigest = manualBindingVisibilityDigest(nil)
	emptyGroups := superadmin
	emptyGroups.VisibilityDigest = manualBindingVisibilityDigest([]uuid.UUID{})

	a, err := buildManualBindingPreview(superadmin)
	require.NoError(t, err)
	b, err := buildManualBindingPreview(emptyGroups)
	require.NoError(t, err)
	require.NotEqual(t, a.PreviewFingerprint, b.PreviewFingerprint)
}

func TestManualBindingFingerprintIncludesCanonicalSnapshotFacts(t *testing.T) {
	sameGeofenceID := uuid.New()
	otherGeofenceID := uuid.New()
	firstBindingID := uuid.New()
	secondBindingID := uuid.New()
	active := BindingStatusActive

	tests := []struct {
		name   string
		first  ManualBindSnapshot
		second ManualBindSnapshot
	}{
		{
			name: "definition status",
			first: func() ManualBindSnapshot {
				snapshot := enabledPolygonSnapshot(eligibleFact("SN001"))
				snapshot.Definition.Status = DefinitionStatusDisabled
				return snapshot
			}(),
			second: func() ManualBindSnapshot {
				snapshot := enabledPolygonSnapshot(eligibleFact("SN001"))
				snapshot.Definition.Status = DefinitionStatusArchived
				return snapshot
			}(),
		},
		{
			name: "device carrier",
			first: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.Device.Carrier = "ctcc"
				return enabledPolygonSnapshot(fact)
			}(),
			second: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.Device.Carrier = "cucc"
				return enabledPolygonSnapshot(fact)
			}(),
		},
		{
			name: "same geofence binding identity",
			first: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.SameGeofenceStatus = &active
				fact.SameGeofenceBindingID = &firstBindingID
				return enabledPolygonSnapshot(fact)
			}(),
			second: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.SameGeofenceStatus = &active
				fact.SameGeofenceBindingID = &secondBindingID
				return enabledPolygonSnapshot(fact)
			}(),
		},
		{
			name: "active rule binding identity",
			first: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.ActiveRuleBindingID = &firstBindingID
				fact.ActiveRuleGeofenceID = &otherGeofenceID
				return enabledPolygonSnapshot(fact)
			}(),
			second: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.ActiveRuleBindingID = &secondBindingID
				fact.ActiveRuleGeofenceID = &otherGeofenceID
				return enabledPolygonSnapshot(fact)
			}(),
		},
		{
			name: "active rule geofence identity",
			first: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.ActiveRuleBindingID = &firstBindingID
				fact.ActiveRuleGeofenceID = &otherGeofenceID
				return enabledPolygonSnapshot(fact)
			}(),
			second: func() ManualBindSnapshot {
				fact := eligibleFact("SN001")
				fact.ActiveRuleBindingID = &firstBindingID
				fact.ActiveRuleGeofenceID = &sameGeofenceID
				return enabledPolygonSnapshot(fact)
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := buildManualBindingPreview(tt.first)
			require.NoError(t, err)
			second, err := buildManualBindingPreview(tt.second)
			require.NoError(t, err)
			require.Equal(t, first.Items[0].Decision, second.Items[0].Decision)
			require.Equal(t, first.Items[0].ReasonCode, second.Items[0].ReasonCode)
			require.NotEqual(
				t,
				first.PreviewFingerprint,
				second.PreviewFingerprint,
			)
		})
	}
}

func TestPreviewManualBindingsRejectsEmptyGeofenceID(t *testing.T) {
	service := NewService(nil, nil)

	_, err := service.PreviewManualBindings(context.Background(), ManualBindingPreviewRequest{})

	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func enabledPolygonSnapshot(inputs ...BindingCandidateFact) ManualBindSnapshot {
	geofenceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("enabled-polygon-geofence"))
	versionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("enabled-polygon-version"))
	return ManualBindSnapshot{
		Definition: Definition{
			ID: geofenceID, Carrier: "cmcc", RuleType: RuleTypePolygonAllowZone,
			Status: DefinitionStatusEnabled, CurrentVersionID: &versionID,
			UpdatedAt: time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC),
		},
		Inputs:           inputs,
		VisibilityDigest: "visible-groups:all",
	}
}

func eligibleFact(serialNumber string) BindingCandidateFact {
	deviceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(serialNumber))
	return BindingCandidateFact{
		Input:  BindingInput{Key: "sn:" + serialNumber, Kind: BindingInputDeviceSN, Value: serialNumber},
		Device: &DeviceIdentity{ID: deviceID, SerialNumber: serialNumber, Carrier: "cmcc"},
	}
}
