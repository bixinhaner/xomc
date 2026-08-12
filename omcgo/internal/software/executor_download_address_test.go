package software

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type countingHTTPSCapabilityReader struct {
	status transfercfg.HTTPSCapabilityStatus
	reads  int
}

func (r *countingHTTPSCapabilityReader) ReadHTTPSCapability(_ context.Context, _ uuid.UUID) transfercfg.HTTPSCapabilityStatus {
	r.reads++
	return r.status
}

type failingDownloadAddressResolver struct {
	err error
}

func (r failingDownloadAddressResolver) Resolve(
	context.Context,
	uuid.UUID,
	transfercfg.TransferDirection,
) (transfercfg.AddressDecision, error) {
	return transfercfg.AddressDecision{}, r.err
}

type downloadAddressHarness struct {
	exec       *UpgradeExecutor
	redis      *redis.Client
	miniredis  *miniredis.Miniredis
	subRepo    *recordingSubTaskRepo
	queued     []devtask.CreateTaskRequest
	deviceID   uuid.UUID
	deviceSN   string
	subTaskID  uuid.UUID
	parentTask uuid.UUID
}

func newDownloadAddressHarness(t *testing.T) *downloadAddressHarness {
	t.Helper()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	h := &downloadAddressHarness{
		redis:      redisClient,
		miniredis:  mr,
		subRepo:    &recordingSubTaskRepo{},
		deviceID:   uuid.New(),
		deviceSN:   "SN-IMG-001",
		subTaskID:  uuid.New(),
		parentTask: uuid.New(),
	}
	deviceRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			require.Equal(t, h.deviceID, id)
			return &model.Device{
				ID:              h.deviceID,
				SerialNumber:    h.deviceSN,
				Status:          model.DeviceActive,
				FirmwareVersion: "V1.0.0",
			}, nil
		},
	}
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			h.queued = append(h.queued, *req)
			return devtask.NewTask(req), nil
		},
	}

	h.exec = NewUpgradeExecutor(
		&svcMockTaskRepo{},
		h.subRepo,
		deviceRepo,
		&svcMockFirmwareRepo{},
		cmdQueue,
		nil,
		redisClient,
		&svcMockEventBus{},
		zap.NewNop(),
	)
	return h
}

func (h *downloadAddressHarness) run(fw *FirmwareVersion) {
	h.exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID:         h.subTaskID,
		TaskID:     h.parentTask,
		DeviceID:   h.deviceID,
		FirmwareID: ptrUUID(uuid.New()),
	}, fw, false, "")
}

func (h *downloadAddressHarness) runResumeMode(fw *FirmwareVersion) {
	h.exec.executeOne(context.Background(), &UpgradeSubTask{
		ID:         h.subTaskID,
		TaskID:     h.parentTask,
		DeviceID:   h.deviceID,
		FirmwareID: ptrUUID(uuid.New()),
	}, fw, false, "", false)
}

func (h *downloadAddressHarness) queuedDownloadURL(t *testing.T) string {
	t.Helper()
	require.Len(t, h.queued, 1)
	require.Equal(t, "Download", h.queued[0].Method)
	var params map[string]any
	require.NoError(t, json.Unmarshal(h.queued[0].Params, &params))
	urlValue, ok := params["url"].(string)
	require.True(t, ok)
	return urlValue
}

func validIMGFirmware() *FirmwareVersion {
	return &FirmwareVersion{
		ID:        uuid.New(),
		Version:   "V2.0.0",
		FileName:  "fw.bin",
		FileSize:  4096,
		FileType:  FileTypeIMG,
		MinIOPath: "img/product/V2.0.0/fw.bin",
		SHA256Val: "deadbeefdeadbeef",
		MD5Val:    "abc123",
	}
}

func newDownloadPolicy(policy string) *transfercfg.Policy {
	return transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: policy,
		Download: transfercfg.DownloadSettings{
			BaseURL:      "http://download.example.com:8080/proxy",
			HTTPSBaseURL: "https://download.example.com:9443/secure-proxy",
			Path:         "/smallcell/FileDownloadService",
		},
	}, nil)
}

func TestExecuteOneIMGDownload_ForceHTTPDoesNotReadCapability(t *testing.T) {
	h := newDownloadAddressHarness(t)
	policy := newDownloadPolicy(transfercfg.ProtocolPolicyForceHTTP)
	reader := &countingHTTPSCapabilityReader{status: transfercfg.HTTPSCapabilityEnabled}
	h.exec.SetTransferProvider(policy)
	h.exec.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, reader))

	h.run(validIMGFirmware())

	assert.Equal(t, 0, reader.reads)
	assert.Equal(t,
		"http://download.example.com:8080/proxy/smallcell/FileDownloadService/firmware/img/product/V2.0.0/fw.bin",
		h.queuedDownloadURL(t),
	)
}

func TestExecuteOneIMGDownload_PreferHTTPSUsesHTTPSWhenCapabilityTrue(t *testing.T) {
	h := newDownloadAddressHarness(t)
	policy := newDownloadPolicy(transfercfg.ProtocolPolicyPreferHTTPS)
	reader := &countingHTTPSCapabilityReader{status: transfercfg.HTTPSCapabilityEnabled}
	h.exec.SetTransferProvider(policy)
	h.exec.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, reader))

	h.run(validIMGFirmware())

	assert.Equal(t, 1, reader.reads)
	assert.Equal(t,
		"https://download.example.com:9443/secure-proxy/smallcell/FileDownloadService/firmware/img/product/V2.0.0/fw.bin",
		h.queuedDownloadURL(t),
	)
}

func TestExecuteOneIMGDownload_PreferHTTPSFallsBackToHTTPWhenCapabilityNotEnabled(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status transfercfg.HTTPSCapabilityStatus
	}{
		{name: "false", status: transfercfg.HTTPSCapabilityDisabled},
		{name: "unknown", status: transfercfg.HTTPSCapabilityUnknown},
		{name: "read error", status: transfercfg.HTTPSCapabilityReadError},
		{name: "other", status: transfercfg.HTTPSCapabilityOtherValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newDownloadAddressHarness(t)
			policy := newDownloadPolicy(transfercfg.ProtocolPolicyPreferHTTPS)
			reader := &countingHTTPSCapabilityReader{status: tc.status}
			h.exec.SetTransferProvider(policy)
			h.exec.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, reader))

			h.run(validIMGFirmware())

			assert.Equal(t, 1, reader.reads)
			assert.Equal(t,
				"http://download.example.com:8080/proxy/smallcell/FileDownloadService/firmware/img/product/V2.0.0/fw.bin",
				h.queuedDownloadURL(t),
			)
		})
	}
}

func TestExecuteOneIMGDownload_PreservesProxyPrefixPortAndPathEncoding(t *testing.T) {
	h := newDownloadAddressHarness(t)
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Download: transfercfg.DownloadSettings{
			BaseURL:      "http://192.0.2.10:8080/reverse-proxy",
			HTTPSBaseURL: "https://[2001:db8::10]:9443/reverse-proxy",
			Path:         "/smallcell/FileDownloadService",
		},
	}, nil)
	reader := &countingHTTPSCapabilityReader{status: transfercfg.HTTPSCapabilityEnabled}
	h.exec.SetTransferProvider(policy)
	h.exec.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, reader))
	fw := validIMGFirmware()
	fw.MinIOPath = "img/product A/V2.0.0/固 件.bin"
	fw.FileName = "固 件.bin"

	h.run(fw)

	assert.Equal(t,
		"https://[2001:db8::10]:9443/reverse-proxy/smallcell/FileDownloadService/firmware/img/product%20A/V2.0.0/%E5%9B%BA%20%E4%BB%B6.bin",
		h.queuedDownloadURL(t),
	)
}

func TestExecuteOneIMGDownload_DoesNotDoubleFirmwareBucketForLegacyMinIOPath(t *testing.T) {
	h := newDownloadAddressHarness(t)
	policy := newDownloadPolicy(transfercfg.ProtocolPolicyForceHTTP)
	h.exec.SetTransferProvider(policy)
	h.exec.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, nil))
	fw := validIMGFirmware()
	fw.MinIOPath = "firmware/SmallCell/V1.0.0/firmware.bin"

	h.run(fw)

	assert.Equal(t,
		"http://download.example.com:8080/proxy/smallcell/FileDownloadService/firmware/SmallCell/V1.0.0/firmware.bin",
		h.queuedDownloadURL(t),
	)
}

func TestExecuteOneIMGDownload_ResolverErrorFailsClosedWithoutCreatingDeviceTask(t *testing.T) {
	h := newDownloadAddressHarness(t)
	h.exec.SetDownloadAddressResolver(failingDownloadAddressResolver{err: errors.New("https base unavailable")})

	h.run(validIMGFirmware())

	assert.Empty(t, h.queued)
	failure, ok := h.subRepo.lastFailure()
	require.True(t, ok)
	assert.Equal(t, FailureInternalError, failure.code)
	assert.Contains(t, failure.msg, "invalid download URL")
	assert.False(t, h.miniredis.Exists(upgradeDeviceLockKey(h.deviceSN)))
}

func TestExecuteOneIMGDownload_ResumeModeKeepsLegacyRelativeURLAndSkipsResolver(t *testing.T) {
	h := newDownloadAddressHarness(t)
	h.exec.SetDownloadAddressResolver(failingDownloadAddressResolver{err: errors.New("must not be called")})
	fw := validIMGFirmware()

	h.runResumeMode(fw)

	assert.Equal(t, "firmware/"+fw.MinIOPath, h.queuedDownloadURL(t))
}

func TestExecuteOneIMGDownload_CanceledResolverStillReleasesDeviceLock(t *testing.T) {
	h := newDownloadAddressHarness(t)
	h.exec.SetDownloadAddressResolver(failingDownloadAddressResolver{err: context.Canceled})

	h.run(validIMGFirmware())

	assert.Empty(t, h.queued)
	assert.False(t, h.miniredis.Exists(upgradeDeviceLockKey(h.deviceSN)))
}

func TestExecuteOneIMGDownload_MissingHTTPSBaseFailsClosedWithoutCreatingDeviceTask(t *testing.T) {
	h := newDownloadAddressHarness(t)
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: transfercfg.ProtocolPolicyPreferHTTPS,
		Download: transfercfg.DownloadSettings{
			BaseURL: "http://download.example.com",
			Path:    "/smallcell/FileDownloadService",
		},
	}, nil)
	reader := &countingHTTPSCapabilityReader{status: transfercfg.HTTPSCapabilityEnabled}
	h.exec.SetTransferProvider(policy)
	h.exec.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, reader))

	h.run(validIMGFirmware())

	assert.Empty(t, h.queued)
	failure, ok := h.subRepo.lastFailure()
	require.True(t, ok)
	assert.Equal(t, FailureInternalError, failure.code)
	assert.Contains(t, failure.msg, "invalid download URL")
	assert.False(t, h.miniredis.Exists(upgradeDeviceLockKey(h.deviceSN)))
}

func TestExecuteOneIMGDownload_URLConstructionErrorFailsClosedWithoutCreatingDeviceTask(t *testing.T) {
	h := newDownloadAddressHarness(t)
	policy := newDownloadPolicy(transfercfg.ProtocolPolicyForceHTTP)
	h.exec.SetTransferProvider(policy)
	h.exec.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, nil))
	fw := validIMGFirmware()
	fw.MinIOPath = "img//bad.bin"

	h.run(fw)

	assert.Empty(t, h.queued)
	failure, ok := h.subRepo.lastFailure()
	require.True(t, ok)
	assert.Equal(t, FailureInternalError, failure.code)
	assert.Contains(t, failure.msg, "invalid download URL")
	assert.False(t, h.miniredis.Exists(upgradeDeviceLockKey(h.deviceSN)))
}

func TestExecuteOneIMGDownload_VerificationFailureDoesNotResolveAddress(t *testing.T) {
	h := newDownloadAddressHarness(t)
	h.exec.SetDownloadAddressResolver(failingDownloadAddressResolver{err: errors.New("must not be called")})
	fw := validIMGFirmware()
	fw.SHA256Val = ""
	fw.MD5Val = ""

	h.run(fw)

	assert.Empty(t, h.queued)
	failure, ok := h.subRepo.lastFailure()
	require.True(t, ok)
	assert.Equal(t, FailureIntegrityCheck, failure.code)
}

func TestExecuteOneDownload_NonIMGKeepsLegacyRelativeURLAndSkipsResolver(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fileType FileType
		path     string
	}{
		{name: "patch", fileType: FileTypePATCH, path: "patch/product/V2.0.0/patch.bin"},
		{name: "fpga", fileType: FileTypeFPGA, path: "fpga/product/V2.0.0/fpga.bin"},
		{name: "ap", fileType: FileTypeAP, path: "ap/product/V2.0.0/ap.bin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newDownloadAddressHarness(t)
			h.exec.SetDownloadAddressResolver(failingDownloadAddressResolver{err: errors.New("must not be called")})
			fw := validIMGFirmware()
			fw.FileType = tc.fileType
			fw.MinIOPath = tc.path

			h.run(fw)

			assert.Equal(t, "firmware/"+tc.path, h.queuedDownloadURL(t))
		})
	}
}
