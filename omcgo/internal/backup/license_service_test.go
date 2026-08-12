package backup

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	devtask "github.com/omcgo/omcgo/internal/task"
)

type fakeLicenseRepo struct {
	mu   sync.Mutex
	rows map[string]*DeviceLicense
}

func newFakeLicenseRepo() *fakeLicenseRepo {
	return &fakeLicenseRepo{rows: make(map[string]*DeviceLicense)}
}

func (r *fakeLicenseRepo) Upsert(_ context.Context, lic *DeviceLicense) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *lic
	cp.AutoDispatchPending = true
	r.rows[lic.SerialNumber] = &cp
	return nil
}

func (r *fakeLicenseRepo) GetBySerialNumber(_ context.Context, sn string) (*DeviceLicense, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	lic := r.rows[sn]
	if lic == nil {
		return nil, nil
	}
	cp := *lic
	return &cp, nil
}

func (r *fakeLicenseRepo) BatchGetBySerialNumbers(ctx context.Context, sns []string) (map[string]*DeviceLicense, error) {
	out := make(map[string]*DeviceLicense)
	for _, sn := range sns {
		lic, err := r.GetBySerialNumber(ctx, sn)
		if err != nil {
			return nil, err
		}
		if lic != nil {
			out[sn] = lic
		}
	}
	return out, nil
}

func (r *fakeLicenseRepo) ClaimAutoDispatch(_ context.Context, sn string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	lic := r.rows[sn]
	if lic == nil || !lic.AutoDispatchPending {
		return false, nil
	}
	lic.AutoDispatchPending = false
	return true, nil
}

func (r *fakeLicenseRepo) ReleaseAutoDispatch(_ context.Context, sn string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if lic := r.rows[sn]; lic != nil {
		lic.AutoDispatchPending = true
	}
	return nil
}

func (r *fakeLicenseRepo) List(context.Context, LicenseFilter) ([]DeviceLicense, int64, error) {
	return nil, 0, nil
}
func (r *fakeLicenseRepo) Delete(_ context.Context, sn string) error {
	delete(r.rows, sn)
	return nil
}
func (r *fakeLicenseRepo) BatchDelete(context.Context, []string) ([]string, error) {
	return nil, nil
}

type fakeLicenseDeviceLookup struct {
	bySN map[string]*model.Device
}

func (f *fakeLicenseDeviceLookup) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	return f.bySN[sn], nil
}

type fakeLicenseTaskEnqueuer struct {
	mu       sync.Mutex
	requests []*devtask.CreateTaskRequest
	err      error
}

func (f *fakeLicenseTaskEnqueuer) CreateTask(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	cp := *req
	f.requests = append(f.requests, &cp)
	return &devtask.Task{}, nil
}

func (f *fakeLicenseTaskEnqueuer) GetQueueLength(context.Context, string) (int64, error) {
	return 0, nil
}

func newTestLicenseService(repo LicenseRepository, devices LicenseDeviceLookup, tasks devtask.Enqueuer) *LicenseService {
	return NewLicenseService(repo, &fakeMover{}, devices, tasks, LicenseBucketDefault, zap.NewNop())
}

func setLicenseTransferPolicy(t *testing.T, svc *LicenseService, policyValue string, capability transfercfg.HTTPSCapabilityStatus) *restoreCapabilityReader {
	t.Helper()
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: policyValue,
		Download: transfercfg.DownloadSettings{
			BaseURL:      "http://license-download.example.com:8080/proxy",
			HTTPSBaseURL: "https://license-download.example.com:9443/secure",
			Path:         "/smallcell/FileDownloadService",
		},
	}, nil)
	reader := &restoreCapabilityReader{status: capability}
	svc.SetTransferProvider(policy)
	svc.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, reader))
	return reader
}

func licenseQueuedURL(t *testing.T, req *devtask.CreateTaskRequest) string {
	t.Helper()
	var params struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(req.Params, &params))
	return params.URL
}

func TestLicenseImportAllowsUnknownDeviceAndKeepsPreinstallPending(t *testing.T) {
	repo := newFakeLicenseRepo()
	tasks := &fakeLicenseTaskEnqueuer{}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{}}, tasks)

	result, err := svc.ImportFromUpload(t.Context(), []LicenseImportItem{{
		FileName: "1202000534228JB0007.lic.json",
		Content:  []byte("license-body"),
	}}, "operator")

	require.NoError(t, err)
	require.Equal(t, []string{"1202000534228JB0007"}, result.Succeeded)
	require.Empty(t, result.Failed)
	stored, err := repo.GetBySerialNumber(t.Context(), "1202000534228JB0007")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.True(t, stored.AutoDispatchPending)
	assert.Empty(t, tasks.requests)
}

func TestLicensePreinstallDispatchesOnceWhenDeviceIsOnline(t *testing.T) {
	const sn = "SN-PREINSTALL-1"
	repo := newFakeLicenseRepo()
	tasks := &fakeLicenseTaskEnqueuer{}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{
		sn: {SerialNumber: sn, IsOnline: true},
	}}, tasks)

	result, err := svc.ImportFromUpload(t.Context(), []LicenseImportItem{{
		FileName: sn + ".lic", Content: []byte("license-body"),
	}}, "operator")
	require.NoError(t, err)
	require.Empty(t, result.Failed)
	require.Len(t, tasks.requests, 1)
	assert.Equal(t, "Download", tasks.requests[0].Method)
	assert.Contains(t, tasks.requests[0].CommandKey, "LICENSE_PREINSTALL_")

	require.NoError(t, svc.DispatchPendingLicense(t.Context(), sn))
	assert.Len(t, tasks.requests, 1, "duplicate online events must not enqueue twice")
	stored, err := repo.GetBySerialNumber(t.Context(), sn)
	require.NoError(t, err)
	assert.False(t, stored.AutoDispatchPending)
}

func TestLicensePreinstallUsesHTTPSDownloadResolver(t *testing.T) {
	const sn = "SN-LIC-HTTPS"
	deviceID := uuid.New()
	repo := newFakeLicenseRepo()
	tasks := &fakeLicenseTaskEnqueuer{}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{
		sn: {ID: deviceID, SerialNumber: sn, IsOnline: true},
	}}, tasks)
	reader := setLicenseTransferPolicy(t, svc, transfercfg.ProtocolPolicyPreferHTTPS, transfercfg.HTTPSCapabilityEnabled)

	result, err := svc.ImportFromUpload(t.Context(), []LicenseImportItem{{
		FileName: sn + ".lic",
		Content:  []byte("license-body"),
	}}, "operator")

	require.NoError(t, err)
	require.Empty(t, result.Failed)
	require.Len(t, tasks.requests, 1)
	assert.Equal(t, 1, reader.calls)
	assert.Equal(t,
		"https://license-download.example.com:9443/secure/smallcell/FileDownloadService/device-licenses/SN-LIC-HTTPS.lic",
		licenseQueuedURL(t, tasks.requests[0]),
	)
	assert.Equal(t, "Download", tasks.requests[0].Method)
	assert.Contains(t, tasks.requests[0].CommandKey, "LICENSE_PREINSTALL_")
	assert.Contains(t, string(tasks.requests[0].Params), `"file_type":"License File"`)
	assert.Contains(t, string(tasks.requests[0].Params), `"target_file_name":"SN-LIC-HTTPS.lic"`)
}

func TestLicenseUpgradeUsesHTTPSDownloadResolverAndEncodesObjectPath(t *testing.T) {
	const sn = "SN-LIC-ENCODE"
	repo := newFakeLicenseRepo()
	repo.rows[sn] = &DeviceLicense{
		SerialNumber: sn,
		FileName:     "License A+B.lic",
		FileExt:      "lic",
		ObjectBucket: LicenseBucketDefault,
		ObjectPath:   "nested/License A+B.lic",
	}
	tasks := &fakeLicenseTaskEnqueuer{}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{
		sn: {ID: uuid.New(), SerialNumber: sn},
	}}, tasks)
	reader := setLicenseTransferPolicy(t, svc, transfercfg.ProtocolPolicyPreferHTTPS, transfercfg.HTTPSCapabilityEnabled)

	_, dispatched, missing, err := svc.DispatchLicenseUpgradeBySN(t.Context(), []string{sn}, "operator", uuid.MustParse("29900000-0000-4000-8000-000000000002"))

	require.NoError(t, err)
	require.Empty(t, missing)
	require.Equal(t, map[string]string{sn: "License A+B.lic"}, dispatched)
	require.Len(t, tasks.requests, 1)
	assert.Equal(t, 1, reader.calls)
	assert.Equal(t,
		"https://license-download.example.com:9443/secure/smallcell/FileDownloadService/device-licenses/nested/License%20A+B.lic",
		licenseQueuedURL(t, tasks.requests[0]),
	)
	assert.Contains(t, string(tasks.requests[0].Params), `"target_file_name":"License A+B.lic"`)
}

func TestLicenseUpgradeForceHTTPDoesNotReadCapability(t *testing.T) {
	const sn = "SN-LIC-FORCE"
	repo := newFakeLicenseRepo()
	repo.rows[sn] = &DeviceLicense{
		SerialNumber: sn,
		FileName:     sn + ".lic",
		FileExt:      "lic",
		ObjectBucket: LicenseBucketDefault,
		ObjectPath:   sn + ".lic",
	}
	tasks := &fakeLicenseTaskEnqueuer{}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{
		sn: {ID: uuid.New(), SerialNumber: sn},
	}}, tasks)
	reader := setLicenseTransferPolicy(t, svc, transfercfg.ProtocolPolicyForceHTTP, transfercfg.HTTPSCapabilityEnabled)

	_, dispatched, missing, err := svc.DispatchLicenseUpgradeBySN(t.Context(), []string{sn}, "operator", uuid.MustParse("29900000-0000-4000-8000-000000000001"))

	require.NoError(t, err)
	require.Empty(t, missing)
	require.Equal(t, map[string]string{sn: sn + ".lic"}, dispatched)
	require.Len(t, tasks.requests, 1)
	assert.Equal(t, 0, reader.calls)
	assert.Equal(t,
		"http://license-download.example.com:8080/proxy/smallcell/FileDownloadService/device-licenses/SN-LIC-FORCE.lic",
		licenseQueuedURL(t, tasks.requests[0]),
	)
	assert.Contains(t, tasks.requests[0].CommandKey, "LICENSE_UPGRADE_")
}

func TestLicenseDownloadResolverErrorFailsClosedWithoutEnqueue(t *testing.T) {
	const sn = "SN-LIC-FAIL"
	repo := newFakeLicenseRepo()
	repo.rows[sn] = &DeviceLicense{
		SerialNumber: sn,
		FileName:     sn + ".lic",
		FileExt:      "lic",
		ObjectBucket: LicenseBucketDefault,
		ObjectPath:   sn + ".lic",
	}
	tasks := &fakeLicenseTaskEnqueuer{}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{
		sn: {ID: uuid.New(), SerialNumber: sn},
	}}, tasks)
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Download: transfercfg.DownloadSettings{
			BaseURL:      "http://license-download.example.com:8080",
			HTTPSBaseURL: "https://license-download.example.com:9443",
			Path:         "/smallcell/FileDownloadService",
		},
	}, nil)
	svc.SetTransferProvider(policy)
	svc.SetDownloadAddressResolver(failingTransferResolver{err: errors.New("resolver down")})

	_, dispatched, missing, err := svc.DispatchLicenseUpgradeBySN(t.Context(), []string{sn}, "operator", uuid.New())

	require.NoError(t, err)
	require.Empty(t, missing)
	assert.Empty(t, dispatched)
	assert.Empty(t, tasks.requests, "resolver failure must not enqueue stale or naked Download URL")
}

func TestLicensePreinstallEnqueueFailureRemainsRetryable(t *testing.T) {
	const sn = "SN-PREINSTALL-RETRY"
	repo := newFakeLicenseRepo()
	tasks := &fakeLicenseTaskEnqueuer{err: errors.New("queue unavailable")}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{
		sn: {SerialNumber: sn, IsOnline: true},
	}}, tasks)

	result, err := svc.ImportFromUpload(t.Context(), []LicenseImportItem{{
		FileName: sn + ".lic", Content: []byte("license-body"),
	}}, "operator")
	require.NoError(t, err)
	require.Empty(t, result.Failed, "durable preinstall must not fail because immediate delivery failed")
	stored, err := repo.GetBySerialNumber(t.Context(), sn)
	require.NoError(t, err)
	assert.True(t, stored.AutoDispatchPending)

	tasks.err = nil
	require.NoError(t, svc.DispatchPendingLicense(t.Context(), sn))
	assert.Len(t, tasks.requests, 1)
}

func TestLicensePreinstallFirmwareChangeCoversSuppressedOnlineEvent(t *testing.T) {
	const sn = "SN-FIRMWARE-ONLINE"
	repo := newFakeLicenseRepo()
	repo.rows[sn] = &DeviceLicense{
		SerialNumber: sn, FileName: sn + ".lic", FileExt: "lic",
		ObjectBucket: LicenseBucketDefault, ObjectPath: sn + ".lic",
		AutoDispatchPending: true,
	}
	tasks := &fakeLicenseTaskEnqueuer{}
	svc := newTestLicenseService(repo, &fakeLicenseDeviceLookup{bySN: map[string]*model.Device{
		sn: {SerialNumber: sn, IsOnline: true},
	}}, tasks)
	subscriber := NewLicensePreinstallSubscriber(svc, zap.NewNop())
	evt, err := event.NewEvent(event.SubjectDeviceFirmwareChanged, device.DeviceFirmwareChangedEvent{
		SerialNumber: sn, BecameOnline: true,
	})
	require.NoError(t, err)

	require.NoError(t, subscriber.handleFirmwareChanged(t.Context(), evt))
	assert.Len(t, tasks.requests, 1)
}
