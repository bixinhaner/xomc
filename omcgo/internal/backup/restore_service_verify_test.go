package backup

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	devicemodel "github.com/omcgo/omcgo/internal/core/model"
)

// ── Task 1 dispatch 侧：下发时记录期望指纹（expected_hash / hash_algo）──

// TestCreate_RecordsExpectedHash 验证按路径恢复在建 restore_task 行时记下期望
// 指纹（= 下发文件 MD5），供恢复后主动校验比对。
func TestCreate_RecordsExpectedHash(t *testing.T) {
	svc, repo, _ := newSvc(t, []string{"SN001"}, true)
	content := []byte("config-bytes-甲")
	sum := md5.Sum(content)
	wantMD5 := hex.EncodeToString(sum[:])
	svc.SetObjectReader(&fakeObjReader{content: content})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/cfg.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "alice")
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	require.NotNil(t, repo.created[0].ExpectedHash)
	assert.Equal(t, wantMD5, *repo.created[0].ExpectedHash)
	require.NotNil(t, repo.created[0].HashAlgo)
	assert.Equal(t, RestoreHashMD5, *repo.created[0].HashAlgo)
}

// TestCreate_NoObjReader_NoExpectedHash 验证未注入 objReader（无从算指纹）时
// 不记 expected_hash，留给编排器停在 downloaded。
func TestCreate_NoObjReader_NoExpectedHash(t *testing.T) {
	svc, repo, _ := newSvc(t, []string{"SN001"}, true)
	// 不调 SetObjectReader → computeSourceMD5 返回空串。
	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/cfg.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "alice")
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Nil(t, repo.created[0].ExpectedHash)
}

// TestCreateBySnapshot_RecordsExpectedHashFromSnapshotMD5 验证按快照恢复用
// config_snapshots.md5（明文指纹）作为期望指纹基线。
func TestCreateBySnapshot_RecordsExpectedHashFromSnapshotMD5(t *testing.T) {
	svc, repo, _ := newSvc(t, []string{"SN001"}, true)
	snap := makeSnap("SN001")
	fp := "snapshot-md5-fingerprint"
	snap.MD5 = &fp
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{"SN001": snap}})

	res, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001"}}, "alice")
	require.NoError(t, err)
	require.NotNil(t, res.Task)
	require.Len(t, repo.created, 1)
	require.NotNil(t, repo.created[0].ExpectedHash)
	assert.Equal(t, fp, *repo.created[0].ExpectedHash)
}

// ── Task 3：跨版本检查在 CreateBySnapshot 里的接线（block 跳过设备）──

// fakeDeviceLookupWithFW 返回带固件版本的设备，用于跨版本检查测试。
type fakeDeviceLookupWithFW struct {
	devices map[string]*devicemodel.Device
}

func (f *fakeDeviceLookupWithFW) GetBySerialNumber(_ context.Context, sn string) (*devicemodel.Device, error) {
	if d, ok := f.devices[sn]; ok {
		return d, nil
	}
	return nil, nil
}

// snapWithVersion 构造一个带来源版本的快照（跨版本检查的源版本来源）。
func snapWithVersion(sn, ver string) *ConfigSnapshot {
	s := makeSnap(sn)
	v := ver
	s.SourceVersion = &v
	return s
}

// TestCreateBySnapshot_CrossVersionBlock_SkipsDevice 验证 Block 策略下源版本与目标
// 设备固件版本不一致时跳过该设备，不下发 Download。
func TestCreateBySnapshot_CrossVersionBlock_SkipsDevice(t *testing.T) {
	repo := &mockRestoreRepo{}
	enq := &fakeEnqueuer{}
	devRepo := &fakeDeviceLookupWithFW{devices: map[string]*devicemodel.Device{
		"SN001": {SerialNumber: "SN001", FirmwareVersion: "2.0.0"}, // 目标固件
	}}
	svc := NewRestoreService(repo, devRepo, enq, &fakeStater{exists: true}, NewRestoreMetrics(nil), zap.NewNop())
	// 快照来源版本 1.0.0 ≠ 目标 2.0.0 → Block 命中。
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{"SN001": snapWithVersion("SN001", "1.0.0")}})
	svc.SetObjectReader(&fakeObjReader{content: []byte("x")})
	svc.SetCrossVersionChecker(NewCrossVersionChecker(CrossVersionBlock, nil, zap.NewNop()))

	res, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001"}}, "alice")
	require.NoError(t, err)
	require.NotNil(t, res.Task)
	assert.Empty(t, enq.requests, "block 命中应跳过设备，不下发 Download")
}

// TestCreateBySnapshot_CrossVersionWarn_StillDispatches 验证 warn_audit 策略下
// 版本不一致仅告警/审计，仍照常下发。
func TestCreateBySnapshot_CrossVersionWarn_StillDispatches(t *testing.T) {
	repo := &mockRestoreRepo{}
	enq := &fakeEnqueuer{}
	devRepo := &fakeDeviceLookupWithFW{devices: map[string]*devicemodel.Device{
		"SN001": {SerialNumber: "SN001", FirmwareVersion: "2.0.0"},
	}}
	sink := &memAuditSink{}
	svc := NewRestoreService(repo, devRepo, enq, &fakeStater{exists: true}, NewRestoreMetrics(nil), zap.NewNop())
	setRestoreTransferPolicy(t, svc, "force_http", "not_read")
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{"SN001": snapWithVersion("SN001", "1.0.0")}})
	svc.SetObjectReader(&fakeObjReader{content: []byte("x")})
	svc.SetCrossVersionChecker(NewCrossVersionChecker(CrossVersionWarnAudit, sink, zap.NewNop()))

	_, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001"}}, "alice")
	require.NoError(t, err)
	require.Len(t, enq.requests, 1, "warn_audit 不阻断下发")
	require.Len(t, sink.events, 1, "不一致应写一条审计")
	assert.Equal(t, "1.0.0", sink.events[0].SourceVersion)
	assert.Equal(t, "2.0.0", sink.events[0].TargetVersion)
}

// TestCreateBySnapshot_NoCrossVersionChecker_DispatchesNormally 验证未装检查器时
// 恢复照常下发（向后兼容）。
func TestCreateBySnapshot_NoCrossVersionChecker_DispatchesNormally(t *testing.T) {
	svc, _, enq := newSvc(t, []string{"SN001"}, true)
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{"SN001": makeSnap("SN001")}})
	svc.SetObjectReader(&fakeObjReader{content: []byte("x")})

	_, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001"}}, "alice")
	require.NoError(t, err)
	require.Len(t, enq.requests, 1)
}
