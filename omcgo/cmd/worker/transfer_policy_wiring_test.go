package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubWorkerTransferSysConfigRepository struct {
	values map[string]string
}

type stubWorkerDeviceParameterByPathReader struct {
	values map[uuid.UUID]string
	calls  int
}

func (r *stubWorkerDeviceParameterByPathReader) GetByPath(
	_ context.Context,
	deviceID uuid.UUID,
	path string,
) (*model.DeviceParameter, error) {
	r.calls++
	if path != transfercfg.HTTPSCapabilityParameterPath {
		return nil, nil
	}
	value, ok := r.values[deviceID]
	if !ok {
		return nil, nil
	}
	return &model.DeviceParameter{
		ParameterPath:  path,
		ParameterValue: value,
	}, nil
}

func (r stubWorkerTransferSysConfigRepository) GetByKey(
	_ context.Context,
	category string,
	key string,
) (*admin.SysConfig, error) {
	value, ok := r.values[category+"."+key]
	if !ok {
		return nil, nil
	}
	return &admin.SysConfig{Category: category, Key: key, Value: value}, nil
}

func TestWorkerTransferPolicyWiringLoadsProtocolSnapshotFields(t *testing.T) {
	repository := stubWorkerTransferSysConfigRepository{values: map[string]string{
		transfercfg.Category + "." + transfercfg.KeyProtocolPolicy:       transfercfg.ProtocolPolicyPreferHTTPS,
		transfercfg.Category + "." + transfercfg.KeyHTTPSUploadBaseURL:   "https://worker-upload.example.com",
		transfercfg.Category + "." + transfercfg.KeyHTTPSDownloadBaseURL: "https://worker-download.example.com",
	}}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{}, newTransferSysConfigLookup(repository))

	snapshot := policy.Snapshot(context.Background())

	assert.Equal(t, transfercfg.ProtocolPolicyPreferHTTPS, snapshot.ProtocolPolicy)
	assert.Equal(t, "https://worker-upload.example.com", snapshot.Upload.HTTPSBaseURL)
	assert.Equal(t, "https://worker-download.example.com", snapshot.Download.HTTPSBaseURL)
}

func TestNewWorkerTransferDefaultsUsesPMUploadTemplateForFreshDeployHTTPFallback(t *testing.T) {
	defaults := newWorkerTransferDefaults(appconfig.PMConfig{
		UploadURLTemplate: "http://worker.example.com:7557/proxy/smallcell/FileUploadService?fileType=PM&filename=",
	})
	policy := transfercfg.NewPolicy(defaults, nil)

	snapshot := policy.Snapshot(context.Background())

	assert.Equal(t, transfercfg.ProtocolPolicyForceHTTP, snapshot.ProtocolPolicy)
	assert.Equal(t, "http://worker.example.com:7557", snapshot.Upload.BaseURL)
	assert.Equal(t, "/proxy/smallcell/FileUploadService", snapshot.Upload.Path)
}

func TestWorkerTransferAddressResolverWiringReadsDeviceHTTPSCapability(t *testing.T) {
	deviceID := uuid.New()
	reader := &stubWorkerDeviceParameterByPathReader{values: map[uuid.UUID]string{
		deviceID: "true",
	}}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Upload: transfercfg.UploadSettings{
			BaseURL:      "http://worker-upload.example.com:8080",
			HTTPSBaseURL: "https://worker-upload.example.com:8443",
		},
	}, nil)
	resolver := newTransferAddressResolver(policy, reader)

	decision, err := resolver.Resolve(
		context.Background(),
		deviceID,
		transfercfg.TransferDirectionUpload,
	)

	require.NoError(t, err)
	assert.Equal(t, "https://worker-upload.example.com:8443", decision.BaseURL)
	assert.Equal(t, transfercfg.TransferProtocolHTTPS, decision.Protocol)
	assert.Equal(t, 1, reader.calls)
}
