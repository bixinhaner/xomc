package provider

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

type stubSysConfigSavedHookRegistrar struct {
	hook admin.SysConfigSavedHook
}

type stubAppTransferSysConfigRepository struct {
	values map[string]string
}

type stubAppDeviceParameterByPathReader struct {
	seenPath string
	values   map[uuid.UUID]string
}

func (r stubAppTransferSysConfigRepository) GetByKey(
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

func (s *stubSysConfigSavedHookRegistrar) RegisterSavedHook(h admin.SysConfigSavedHook) {
	s.hook = h
}

func (r *stubAppDeviceParameterByPathReader) GetByPath(
	_ context.Context,
	deviceID uuid.UUID,
	path string,
) (*model.DeviceParameter, error) {
	r.seenPath = path
	value, ok := r.values[deviceID]
	if !ok {
		return nil, nil
	}
	return &model.DeviceParameter{
		DeviceID:       deviceID,
		ParameterPath:  path,
		ParameterValue: value,
	}, nil
}

func TestAppTransferPolicyWiringLoadsProtocolSnapshotFields(t *testing.T) {
	repository := stubAppTransferSysConfigRepository{values: map[string]string{
		transfercfg.Category + "." + transfercfg.KeyProtocolPolicy:       transfercfg.ProtocolPolicyPreferHTTPS,
		transfercfg.Category + "." + transfercfg.KeyHTTPSUploadBaseURL:   "https://app-upload.example.com",
		transfercfg.Category + "." + transfercfg.KeyHTTPSDownloadBaseURL: "https://app-download.example.com",
	}}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{}, newTransferSysConfigLookup(repository))

	snapshot := policy.Snapshot(context.Background())

	assert.Equal(t, transfercfg.ProtocolPolicyPreferHTTPS, snapshot.ProtocolPolicy)
	assert.Equal(t, "https://app-upload.example.com", snapshot.Upload.HTTPSBaseURL)
	assert.Equal(t, "https://app-download.example.com", snapshot.Download.HTTPSBaseURL)
}

func TestNewSoftwareTransferDefaultsUsesUpgradeBaseURLForFreshDeployHTTPFallback(t *testing.T) {
	defaults := newSoftwareTransferDefaults(appconfig.UpgradeConfig{
		ACSUploadBaseURL: " http://app.example.com:8080/base/ ",
	})
	policy := transfercfg.NewPolicy(defaults, nil)

	snapshot := policy.Snapshot(context.Background())

	assert.Equal(t, transfercfg.ProtocolPolicyForceHTTP, snapshot.ProtocolPolicy)
	assert.Equal(t, "http://app.example.com:8080/base", snapshot.Upload.BaseURL)
	assert.Equal(t, "/smallcell/FileUploadService", snapshot.Upload.Path)
	assert.Equal(t, "http://app.example.com:8080/base", snapshot.Download.BaseURL)
	assert.Equal(t, "/smallcell/FileDownloadService", snapshot.Download.Path)
}

func TestNewMRTransferDefaultsUsesSharedACSUploadBaseURLForFreshDeployHTTPFallback(t *testing.T) {
	defaults := newMRTransferDefaults(appconfig.UpgradeConfig{
		ACSUploadBaseURL: " http://acs-upload.example.com:8080/base/ ",
	})
	policy := transfercfg.NewPolicy(defaults, nil)

	snapshot := policy.Snapshot(context.Background())

	assert.Equal(t, transfercfg.ProtocolPolicyForceHTTP, snapshot.ProtocolPolicy)
	assert.Equal(t, "http://acs-upload.example.com:8080/base", snapshot.Upload.BaseURL)
	assert.Equal(t, "/smallcell/FileUploadService", snapshot.Upload.Path)
}

func TestRegisterTransferPolicyInvalidation_RefreshesAppSnapshotAfterSave(t *testing.T) {
	values := map[string]string{
		transfercfg.KeyProtocolPolicy:       transfercfg.ProtocolPolicyForceHTTP,
		transfercfg.KeyHTTPSUploadBaseURL:   "https://old-upload.example.com",
		transfercfg.KeyHTTPSDownloadBaseURL: "https://old-download.example.com",
	}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{}, func(_ context.Context, category, key string) (string, bool) {
		if category != transfercfg.Category {
			return "", false
		}
		value, ok := values[key]
		return value, ok
	})
	registrar := &stubSysConfigSavedHookRegistrar{}

	registerTransferPolicyInvalidation(registrar, policy)
	require.NotNil(t, registrar.hook)
	require.Equal(t, transfercfg.ProtocolPolicyForceHTTP, policy.Snapshot(context.Background()).ProtocolPolicy)

	values[transfercfg.KeyProtocolPolicy] = transfercfg.ProtocolPolicyPreferHTTPS
	values[transfercfg.KeyHTTPSUploadBaseURL] = "https://new-upload.example.com"
	values[transfercfg.KeyHTTPSDownloadBaseURL] = "https://new-download.example.com"
	registrar.hook(context.Background(), transfercfg.Category)

	refreshed := policy.Snapshot(context.Background())
	assert.Equal(t, transfercfg.ProtocolPolicyPreferHTTPS, refreshed.ProtocolPolicy)
	assert.Equal(t, "https://new-upload.example.com", refreshed.Upload.HTTPSBaseURL)
	assert.Equal(t, "https://new-download.example.com", refreshed.Download.HTTPSBaseURL)
}

func TestAppTransferAddressResolverWiringReadsDeviceHTTPSCapability(t *testing.T) {
	deviceID := uuid.New()
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Download: transfercfg.DownloadSettings{
			BaseURL:      "http://app-download.example.com",
			HTTPSBaseURL: "https://app-download.example.com",
		},
	}, nil)
	reader := &stubAppDeviceParameterByPathReader{values: map[uuid.UUID]string{
		deviceID: "true",
	}}

	resolver := newTransferAddressResolver(policy, reader)
	decision, err := resolver.Resolve(
		context.Background(),
		deviceID,
		transfercfg.TransferDirectionDownload,
	)

	require.NoError(t, err)
	assert.Equal(t, transfercfg.HTTPSCapabilityParameterPath, reader.seenPath)
	assert.Equal(t, transfercfg.TransferProtocolHTTPS, decision.Protocol)
	assert.Equal(t, "https://app-download.example.com", decision.BaseURL)
}
