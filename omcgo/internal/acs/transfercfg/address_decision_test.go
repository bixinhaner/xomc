package transfercfg

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticSnapshotProvider struct {
	snapshot Snapshot
}

func TestAddressResolver_EnabledDeviceFailsClosedForMissingOrInvalidHTTPSBaseURL(t *testing.T) {
	tests := []struct {
		name      string
		direction TransferDirection
		configure func(*Snapshot)
	}{
		{
			name:      "upload missing",
			direction: TransferDirectionUpload,
			configure: func(snapshot *Snapshot) { snapshot.Upload.HTTPSBaseURL = "" },
		},
		{
			name:      "upload uses HTTP scheme",
			direction: TransferDirectionUpload,
			configure: func(snapshot *Snapshot) { snapshot.Upload.HTTPSBaseURL = "http://upload.example.com" },
		},
		{
			name:      "download missing",
			direction: TransferDirectionDownload,
			configure: func(snapshot *Snapshot) { snapshot.Download.HTTPSBaseURL = "" },
		},
		{
			name:      "download uses HTTP scheme",
			direction: TransferDirectionDownload,
			configure: func(snapshot *Snapshot) { snapshot.Download.HTTPSBaseURL = "http://download.example.com" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := Snapshot{
				ProtocolPolicy: ProtocolPolicyPreferHTTPS,
				Upload: UploadSettings{
					BaseURL:      "http://upload.example.com",
					HTTPSBaseURL: "https://upload.example.com",
				},
				Download: DownloadSettings{
					BaseURL:      "http://download.example.com",
					HTTPSBaseURL: "https://download.example.com",
				},
			}
			tt.configure(&snapshot)
			resolver := NewAddressResolver(
				staticSnapshotProvider{snapshot: snapshot},
				&stubHTTPSCapabilityReader{status: HTTPSCapabilityEnabled},
			)

			decision, err := resolver.Resolve(context.Background(), uuid.New(), tt.direction)

			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrHTTPSBaseURLUnavailable))
			assert.Empty(t, decision.BaseURL)
			assert.NotContains(t, err.Error(), "http://")
		})
	}
}

func (p staticSnapshotProvider) Snapshot(context.Context) Snapshot {
	return p.snapshot
}

type stubHTTPSCapabilityReader struct {
	status HTTPSCapabilityStatus
	calls  int
}

func (r *stubHTTPSCapabilityReader) ReadHTTPSCapability(context.Context, uuid.UUID) HTTPSCapabilityStatus {
	r.calls++
	return r.status
}

func TestAddressResolver_ForceHTTPShortCircuitsCapabilityReader(t *testing.T) {
	tests := []struct {
		name      string
		direction TransferDirection
		wantURL   string
	}{
		{name: "upload", direction: TransferDirectionUpload, wantURL: "http://upload.example.com/proxy"},
		{name: "download", direction: TransferDirectionDownload, wantURL: "http://download.example.com/proxy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &stubHTTPSCapabilityReader{status: HTTPSCapabilityEnabled}
			resolver := NewAddressResolver(staticSnapshotProvider{snapshot: Snapshot{
				ProtocolPolicy: ProtocolPolicyForceHTTP,
				Upload:         UploadSettings{BaseURL: "http://upload.example.com/proxy"},
				Download:       DownloadSettings{BaseURL: "http://download.example.com/proxy"},
			}}, reader)

			decision, err := resolver.Resolve(context.Background(), uuid.New(), tt.direction)

			require.NoError(t, err)
			assert.Equal(t, tt.direction, decision.Direction)
			assert.Equal(t, TransferProtocolHTTP, decision.Protocol)
			assert.Equal(t, tt.wantURL, decision.BaseURL)
			assert.Equal(t, HTTPSCapabilityNotRead, decision.Capability)
			assert.Equal(t, AddressReasonForceHTTP, decision.Reason)
			assert.Zero(t, reader.calls)
		})
	}
}

func TestAddressResolver_StrategyCapabilityDirectionMatrix(t *testing.T) {
	type capabilityCase struct {
		status HTTPSCapabilityStatus
		reason AddressDecisionReason
	}
	capabilities := []capabilityCase{
		{status: HTTPSCapabilityEnabled, reason: AddressReasonHTTPSCapabilityEnabled},
		{status: HTTPSCapabilityDisabled, reason: AddressReasonHTTPSCapabilityDisabled},
		{status: HTTPSCapabilityUnknown, reason: AddressReasonHTTPSCapabilityUnknown},
		{status: HTTPSCapabilityReadError, reason: AddressReasonHTTPSCapabilityReadError},
		{status: HTTPSCapabilityOtherValue, reason: AddressReasonHTTPSCapabilityOtherValue},
	}
	directions := []TransferDirection{TransferDirectionUpload, TransferDirectionDownload}
	policies := []string{ProtocolPolicyForceHTTP, ProtocolPolicyPreferHTTPS}
	snapshot := Snapshot{
		Upload: UploadSettings{
			BaseURL:      "http://upload.example.com/proxy",
			HTTPSBaseURL: "https://upload.example.com/proxy",
		},
		Download: DownloadSettings{
			BaseURL:      "http://download.example.com/proxy",
			HTTPSBaseURL: "https://download.example.com/proxy",
		},
	}

	for _, policy := range policies {
		for _, capability := range capabilities {
			for _, direction := range directions {
				name := policy + "/" + string(capability.status) + "/" + string(direction)
				t.Run(name, func(t *testing.T) {
					snapshot.ProtocolPolicy = policy
					reader := &stubHTTPSCapabilityReader{status: capability.status}
					resolver := NewAddressResolver(staticSnapshotProvider{snapshot: snapshot}, reader)

					decision, err := resolver.Resolve(context.Background(), uuid.New(), direction)

					require.NoError(t, err)
					assert.Equal(t, direction, decision.Direction)
					if policy == ProtocolPolicyForceHTTP {
						assert.Equal(t, TransferProtocolHTTP, decision.Protocol)
						assert.Equal(t, HTTPSCapabilityNotRead, decision.Capability)
						assert.Equal(t, AddressReasonForceHTTP, decision.Reason)
						assert.Zero(t, reader.calls)
						return
					}

					assert.Equal(t, 1, reader.calls)
					assert.Equal(t, capability.status, decision.Capability)
					assert.Equal(t, capability.reason, decision.Reason)
					if capability.status == HTTPSCapabilityEnabled {
						assert.Equal(t, TransferProtocolHTTPS, decision.Protocol)
						assert.Contains(t, decision.BaseURL, "https://")
					} else {
						assert.Equal(t, TransferProtocolHTTP, decision.Protocol)
						assert.Contains(t, decision.BaseURL, "http://")
					}
				})
			}
		}
	}
}
