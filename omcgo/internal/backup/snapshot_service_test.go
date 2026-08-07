package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ─────────────────────────────────────────────────────────────────────────
// fakes
// ─────────────────────────────────────────────────────────────────────────

type fakeSnapshotRepo struct {
	rows      map[string]*ConfigSnapshot
	upsertErr error
	upserts   []ConfigSnapshot
	mu        sync.Mutex
}

func newFakeSnapshotRepo() *fakeSnapshotRepo {
	return &fakeSnapshotRepo{rows: map[string]*ConfigSnapshot{}}
}

func (r *fakeSnapshotRepo) Upsert(_ context.Context, s *ConfigSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.upsertErr != nil {
		return r.upsertErr
	}
	cp := *s
	r.rows[s.SerialNumber] = &cp
	r.upserts = append(r.upserts, cp)
	return nil
}

func (r *fakeSnapshotRepo) GetBySerialNumber(_ context.Context, sn string) (*ConfigSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.rows[sn]; ok {
		cp := *v
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeSnapshotRepo) BatchGetBySerialNumbers(_ context.Context, sns []string) (map[string]*ConfigSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]*ConfigSnapshot{}
	for _, sn := range sns {
		if v, ok := r.rows[sn]; ok {
			cp := *v
			out[sn] = &cp
		}
	}
	return out, nil
}

func (r *fakeSnapshotRepo) List(_ context.Context, _ SnapshotFilter) ([]ConfigSnapshot, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]ConfigSnapshot, 0, len(r.rows))
	for _, v := range r.rows {
		items = append(items, *v)
	}
	return items, int64(len(items)), nil
}

func (r *fakeSnapshotRepo) Delete(_ context.Context, sn string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rows, sn)
	return nil
}

func (r *fakeSnapshotRepo) BatchDelete(_ context.Context, sns []string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleted := make([]string, 0, len(sns))
	for _, sn := range sns {
		if _, ok := r.rows[sn]; ok {
			delete(r.rows, sn)
			deleted = append(deleted, sn)
		}
	}
	return deleted, nil
}

// ----- mover (+ SnapshotSourceReader) -----

type putCall struct {
	bucket string
	object string
	body   []byte
	size   int64
}

type snapshotRemoveCall struct {
	bucket string
	object string
}

// fakeMover 同时实现 SnapshotMover 与 SnapshotSourceReader：objects 既被 PutObject
// 落地，也可由测试 seed 充当 promote 的源备份对象（#61）。
type fakeMover struct {
	putErr    error
	removeErr error
	readErr   error
	objects   map[string][]byte // "bucket/object" → bytes
	putCalls  []putCall
	rmCalls   []snapshotRemoveCall
	mu        sync.Mutex
}

// seed 预置一个源对象字节（promote 读取）。
func (m *fakeMover) seed(bucket, object string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	m.objects[bucket+"/"+object] = data
}

func (m *fakeMover) ReadObject(_ context.Context, bucket, object string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.readErr != nil {
		return nil, m.readErr
	}
	data, ok := m.objects[bucket+"/"+object]
	if !ok {
		return nil, fmt.Errorf("fake: source object not found %s/%s", bucket, object)
	}
	return append([]byte(nil), data...), nil
}

func (m *fakeMover) PutObject(_ context.Context, bucket, obj string, r io.Reader, size int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, _ := io.ReadAll(r)
	m.putCalls = append(m.putCalls, putCall{bucket, obj, body, size})
	if m.putErr != nil {
		return minio.UploadInfo{}, m.putErr
	}
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	m.objects[bucket+"/"+obj] = body
	return minio.UploadInfo{Bucket: bucket, Key: obj}, nil
}

func (m *fakeMover) RemoveObject(_ context.Context, bucket, obj string, _ minio.RemoveObjectOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rmCalls = append(m.rmCalls, snapshotRemoveCall{bucket, obj})
	delete(m.objects, bucket+"/"+obj)
	return m.removeErr
}

// ----- file lookup -----

type fakeFileLookup struct {
	bySN map[string][]BackupRestoreFile
	err  error
}

func (f *fakeFileLookup) ListBySerial(_ context.Context, sn string) ([]BackupRestoreFile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.bySN[sn], nil
}

// ----- device lookup (snapshot-test only; restore_service_test.go has a
// different fakeDeviceLookup with knownSNs map) -----

type fakeSnapshotDeviceLookup struct {
	bySN map[string]*model.Device
}

func (f *fakeSnapshotDeviceLookup) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	return f.bySN[sn], nil
}

// ─────────────────────────────────────────────────────────────────────────
// constructors
// ─────────────────────────────────────────────────────────────────────────

func newTestSnapshotService(t *testing.T,
	repo SnapshotRepository, mover SnapshotMover,
	fileLookup SnapshotBackupFileLookup, deviceLookup SnapshotDeviceLookup,
) *SnapshotService {
	t.Helper()
	return NewSnapshotService(repo, mover, fileLookup, deviceLookup, "config-snapshots", zap.NewNop())
}

// newGCMEncryptor 构造一个测试用 AES-256-GCM Encryptor，既给 promote 注入解密器，
// 也用于在测试里把明文加密成"源备份对象"做往返。
func newGCMEncryptor(t *testing.T) Encryptor {
	t.Helper()
	enc, err := NewEncryptor("AES-256-GCM", newStaticKeyProvider(makeTestKey(t)))
	require.NoError(t, err)
	return enc
}

// md5HexOf 返回字节的十六进制 MD5（与 promote 写入 config_snapshots.MD5 同算法）。
func md5HexOf(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

// gzipBytes 用 gzip 压缩 plain，模拟上传侧 EnableCompression 落盘的字节。
func gzipBytes(t *testing.T, plain []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(plain)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

// findPut 在 mover.putCalls 里找写到指定 bucket/object 的那次调用。
func findPut(t *testing.T, mover *fakeMover, bucket, object string) putCall {
	t.Helper()
	for _, c := range mover.putCalls {
		if c.bucket == bucket && c.object == object {
			return c
		}
	}
	t.Fatalf("no PutObject to %s/%s found; putCalls=%+v", bucket, object, mover.putCalls)
	return putCall{}
}

// ─────────────────────────────────────────────────────────────────────────
// PromoteFromBackup tests（#61：解码管线 —— 读源对象 → 解密/解压 → 明文 PutObject）
// ─────────────────────────────────────────────────────────────────────────

func TestPromoteFromBackup_PlaintextSource(t *testing.T) {
	taskID := uuid.New()
	plain := []byte(`<?xml version="1.0"?><cfg>hello</cfg>`)
	srcRow := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-12345678-SN001.xml",
		ObjectPath:   "config_backup/backup/2026/05/22/backup-12345678-SN001.xml",
		FileSize:     int64(len(plain)),
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", "backup/2026/05/22/backup-12345678-SN001.xml", plain)
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {srcRow}}}
	dl := &fakeSnapshotDeviceLookup{bySN: map[string]*model.Device{
		"SN001": {DeviceName: "eNB-001", ProductClass: "FAP/BSC"},
	}}
	svc := newTestSnapshotService(t, repo, mover, fl, dl)
	svc.SetPromoteDecoder(mover, nil) // 明文源无需解密器

	require.NoError(t, svc.PromoteFromBackup(context.Background(), taskID, "SN001"))

	// 写到快照桶的就是明文（无 .enc/.gz 后缀）。
	put := findPut(t, mover, "config-snapshots", "SN001_CFG.xml")
	assert.Equal(t, plain, put.body, "snapshot object must be exact plaintext")

	row := repo.rows["SN001"]
	require.NotNil(t, row)
	assert.Equal(t, "SN001_CFG.xml", row.FileName)
	assert.Equal(t, "xml", row.FileExt)
	assert.Equal(t, "config-snapshots", row.ObjectBucket)
	assert.Equal(t, "SN001_CFG.xml", row.ObjectPath)
	assert.Equal(t, SnapshotSourceBackup, row.Source)
	require.NotNil(t, row.SourceTaskID)
	assert.Equal(t, taskID, *row.SourceTaskID)
	require.NotNil(t, row.EnbName)
	assert.Equal(t, "eNB-001", *row.EnbName)
	require.NotNil(t, row.ProductType)
	assert.Equal(t, "FAP/BSC", *row.ProductType)
	assert.EqualValues(t, len(plain), row.FileSize)
	// MD5 必须是明文哈希（#61 finding 3），而非源对象 ETag。
	require.NotNil(t, row.MD5)
	assert.Equal(t, md5HexOf(plain), *row.MD5)
}

// 核心回归（#61 finding 1）：加密 + 压缩的源备份对象，promote 后快照桶里必须是
// 可直接下发 CPE 的明文，且 AAD 不再错配。
func TestPromoteFromBackup_EncryptedCompressedSource_DecodesToPlaintext(t *testing.T) {
	taskID := uuid.New()
	enc := newGCMEncryptor(t)
	plain := []byte(`<?xml version="1.0"?><cfg><param>secret-value</param></cfg>`)

	// 模拟上传侧落盘：gzip 压缩 → AES-GCM 加密；on-disk key 带 .gz.enc 后缀，
	// AAD = 去掉 .enc 的 basename（与 acs/upload encAAD 契约一致）。
	srcKey := "backup/2026/05/22/backup-deadbeef-SN777.xml.gz.enc"
	gzBytes := gzipBytes(t, plain)
	aad := []byte("backup-deadbeef-SN777.xml.gz")
	blob, err := enc.Encrypt(gzBytes, aad)
	require.NoError(t, err)

	srcRow := BackupRestoreFile{
		SerialNumber: "SN777",
		FileName:     "backup-deadbeef-SN777.xml", // 事件 filename 无后缀
		ObjectPath:   "config_backup/" + srcKey,   // object_path 带全后缀
		FileSize:     int64(len(blob)),
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", srcKey, blob)
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN777": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, enc)

	require.NoError(t, svc.PromoteFromBackup(context.Background(), taskID, "SN777"))

	put := findPut(t, mover, "config-snapshots", "SN777_CFG.xml")
	assert.Equal(t, plain, put.body,
		"快照对象必须是解密+解压后的明文（否则恢复时 CPE 拿到不可用密文/压缩流）")

	row := repo.rows["SN777"]
	require.NotNil(t, row)
	assert.Equal(t, "SN777_CFG.xml", row.ObjectPath, "快照 key 无 .gz/.enc 后缀")
	assert.Equal(t, "xml", row.FileExt)
	assert.EqualValues(t, len(plain), row.FileSize, "FileSize 是明文大小")
	require.NotNil(t, row.MD5)
	assert.Equal(t, md5HexOf(plain), *row.MD5, "MD5 是明文配置指纹，非密文 ETag")
}

// 仅压缩（不加密）的源对象也应被解压成明文（EnableCompression 默认开启）。
func TestPromoteFromBackup_CompressedOnlySource_Decompresses(t *testing.T) {
	taskID := uuid.New()
	plain := []byte("just-config-bytes-zzzzzzzzzzzzzzzzzzzz")
	srcKey := "backup/cfg-SN9.xml.gz"
	srcRow := BackupRestoreFile{
		SerialNumber: "SN9",
		FileName:     "cfg-SN9.xml",
		ObjectPath:   "config_backup/" + srcKey,
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", srcKey, gzipBytes(t, plain))
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN9": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, nil) // 未加密，无需解密器

	require.NoError(t, svc.PromoteFromBackup(context.Background(), taskID, "SN9"))
	put := findPut(t, mover, "config-snapshots", "SN9_CFG.xml")
	assert.Equal(t, plain, put.body)
}

// 源是加密对象但未注入解密器 → 必须拒绝（不写不可解密的坏快照）。
func TestPromoteFromBackup_EncryptedSourceNoDecryptor_Rejected(t *testing.T) {
	taskID := uuid.New()
	srcKey := "backup/cfg-SN8.xml.enc"
	srcRow := BackupRestoreFile{
		SerialNumber: "SN8",
		FileName:     "cfg-SN8.xml",
		ObjectPath:   "config_backup/" + srcKey,
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", srcKey, []byte("OENC...ciphertext..."))
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN8": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, nil) // 没有解密器

	err := svc.PromoteFromBackup(context.Background(), taskID, "SN8")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionKeyUnavailable))
	assert.Empty(t, repo.upserts, "拒绝写坏快照")
	assert.Empty(t, mover.putCalls, "不应 PutObject")
}

// 错误的 AAD（被换名/被替换）→ 解密失败 → 不写快照（AEAD 反替换绑定生效）。
func TestPromoteFromBackup_TamperedAAD_DecryptFails(t *testing.T) {
	taskID := uuid.New()
	enc := newGCMEncryptor(t)
	// 用一个 basename 加密，但 object_path 用另一个 basename → AAD 对不上。
	blob, err := enc.Encrypt([]byte("payload"), []byte("ORIGINAL-NAME.xml"))
	require.NoError(t, err)
	srcKey := "backup/RENAMED-NAME.xml.enc" // AAD 会被推导为 RENAMED-NAME.xml
	srcRow := BackupRestoreFile{
		SerialNumber: "SN5",
		FileName:     "RENAMED-NAME.xml",
		ObjectPath:   "config_backup/" + srcKey,
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", srcKey, blob)
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN5": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, enc)

	err = svc.PromoteFromBackup(context.Background(), taskID, "SN5")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionAuthFailed))
	assert.Empty(t, repo.upserts)
}

func TestPromoteFromBackup_NVFile(t *testing.T) {
	taskID := uuid.New()
	plain := []byte("nv-config-blob")
	srcRow := BackupRestoreFile{
		SerialNumber: "SN_NV1",
		FileName:     "mib-home-fap.nv",
		ObjectPath:   "config_backup/backup/mib-home-fap.nv",
		FileSize:     int64(len(plain)),
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", "backup/mib-home-fap.nv", plain)
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN_NV1": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, nil)

	require.NoError(t, svc.PromoteFromBackup(context.Background(), taskID, "SN_NV1"))
	row := repo.rows["SN_NV1"]
	require.NotNil(t, row)
	assert.Equal(t, "SN_NV1_CFG.nv", row.FileName)
	assert.Equal(t, "nv", row.FileExt)
}

func TestPromoteFromBackup_NoSourceFile_ReturnsNotFound(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, nil)

	err := svc.PromoteFromBackup(context.Background(), uuid.New(), "SN404")
	require.Error(t, err)
	assert.Empty(t, mover.putCalls)
	assert.Empty(t, repo.upserts)
}

func TestPromoteFromBackup_PutObjectFails_DoesNotUpsert(t *testing.T) {
	taskID := uuid.New()
	srcKey := "backup/x.xml"
	srcRow := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-x-SN001.xml",
		ObjectPath:   "config_backup/" + srcKey,
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{putErr: errors.New("simulated minio failure")}
	mover.seed("config-backup", srcKey, []byte("<cfg/>"))
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, nil)

	err := svc.PromoteFromBackup(context.Background(), taskID, "SN001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated minio failure")
	assert.Empty(t, repo.upserts, "DB upsert should be skipped if PutObject fails")
}

func TestPromoteFromBackup_FileLookupNotConfigured(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil) // fileLookup=nil

	err := svc.PromoteFromBackup(context.Background(), uuid.New(), "SN001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPromoteNotConfigured))
}

// 解码管线（sourceReader）未注入 → 拒绝走老的原样复制路径（会产生坏快照）。
func TestPromoteFromBackup_DecoderNotConfigured(t *testing.T) {
	taskID := uuid.New()
	srcRow := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-x-SN001.xml",
		ObjectPath:   "config_backup/backup/x.xml",
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	// 不调 SetPromoteDecoder

	err := svc.PromoteFromBackup(context.Background(), taskID, "SN001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPromoteNotConfigured))
	assert.Empty(t, repo.upserts)
}

func TestPromoteFromBackup_RejectsInvalidExtension(t *testing.T) {
	taskID := uuid.New()
	srcKey := "foo/backup.weirdext"
	srcRow := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup.weirdext", // 既不是 xml 也不是 nv
		ObjectPath:   "config_backup/" + srcKey,
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", srcKey, []byte("whatever"))
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, nil)

	err := svc.PromoteFromBackup(context.Background(), taskID, "SN001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidConfigFileExt))
	assert.Empty(t, mover.putCalls, "must not PutObject when normalization fails")
}

func TestPromoteFromBackup_FallbackToLatestWhenTaskIDMismatch(t *testing.T) {
	taskID := uuid.New()
	// 两行：第一行（最新）task_id 与请求不一致，仍应被选中作为回退。
	row1 := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-newer-SN001.xml",
		ObjectPath:   "config_backup/backup/newer.xml",
		TaskID:       ptrStr(uuid.New().String()), // 不匹配
	}
	row2 := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-older-SN001.xml",
		ObjectPath:   "config_backup/backup/older.xml",
		TaskID:       ptrStr(uuid.New().String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	mover.seed("config-backup", "backup/newer.xml", []byte("<newer/>"))
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {row1, row2}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)
	svc.SetPromoteDecoder(mover, nil)

	require.NoError(t, svc.PromoteFromBackup(context.Background(), taskID, "SN001"))
	put := findPut(t, mover, "config-snapshots", "SN001_CFG.xml")
	assert.Equal(t, []byte("<newer/>"), put.body, "应选中最新行 newer.xml 作为回退源")
}

// ─────────────────────────────────────────────────────────────────────────
// ImportFromUpload tests
// ─────────────────────────────────────────────────────────────────────────

func TestImportFromUpload_AllowsUnknownDevice(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil,
		&fakeSnapshotDeviceLookup{bySN: map[string]*model.Device{}})

	res, err := svc.ImportFromUpload(t.Context(), []SnapshotImportItem{{
		FileName: "NOT-REGISTERED_CFG.xml",
		Content:  []byte("<config/>")},
	}, "alice")

	require.NoError(t, err)
	require.Equal(t, []string{"NOT-REGISTERED"}, res.Succeeded)
	require.Empty(t, res.Failed)
	stored, err := repo.GetBySerialNumber(t.Context(), "NOT-REGISTERED")
	require.NoError(t, err)
	require.NotNil(t, stored)
}

func TestImportFromUpload_AllSucceed(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	items := []SnapshotImportItem{
		{FileName: "SN001_CFG.xml", Content: []byte("<config/>")},
		{FileName: "SN002_CFG.nv", Content: []byte{0x01, 0x02, 0x03}},
	}
	res, err := svc.ImportFromUpload(context.Background(), items, "alice")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"SN001", "SN002"}, res.Succeeded)
	assert.Empty(t, res.Failed)

	require.Len(t, mover.putCalls, 2)
	require.Len(t, repo.upserts, 2)
	for _, row := range repo.upserts {
		assert.Equal(t, SnapshotSourceManualUpload, row.Source)
		require.NotNil(t, row.UpdateBy)
		assert.Equal(t, "alice", *row.UpdateBy)
		assert.NotEmpty(t, row.MD5)
	}
}

func TestImportFromUpload_PartialFailure_InvalidName(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	items := []SnapshotImportItem{
		{FileName: "SN001_CFG.xml", Content: []byte("ok")},
		{FileName: "weird.txt", Content: []byte("bad")},
		{FileName: "SN003_CFG.nv", Content: []byte("nvdata")},
	}
	res, err := svc.ImportFromUpload(context.Background(), items, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"SN001", "SN003"}, res.Succeeded)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, "weird.txt", res.Failed[0].FileName)
	assert.Equal(t, ImportErrInvalidName, res.Failed[0].ErrorCode)
	assert.Contains(t, res.Failed[0].Message, "weird.txt")

	// Failed item should not produce MinIO Put or DB Upsert.
	assert.Len(t, mover.putCalls, 2)
	assert.Len(t, repo.upserts, 2)
}

func TestImportFromUpload_EmptyBodyRejected(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.xml", Content: nil}}, "")
	require.NoError(t, err)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, ImportErrEmptyBody, res.Failed[0].ErrorCode)
	assert.Empty(t, mover.putCalls)
	assert.Empty(t, repo.upserts)
}

func TestImportFromUpload_PutObjectFails(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{putErr: errors.New("network down")}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.xml", Content: []byte("ok")}}, "")
	require.NoError(t, err)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, ImportErrPutObject, res.Failed[0].ErrorCode)
	assert.Equal(t, "SN001", res.Failed[0].SerialNumber)
	assert.Empty(t, repo.upserts, "DB upsert must not run when MinIO Put fails")
}

func TestImportFromUpload_UpsertFails_CompensatesRemoveObject(t *testing.T) {
	repo := newFakeSnapshotRepo()
	repo.upsertErr = errors.New("DB exploded")
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.xml", Content: []byte("ok")}}, "")
	require.NoError(t, err)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, ImportErrUpsert, res.Failed[0].ErrorCode)

	require.Len(t, mover.putCalls, 1)
	require.Len(t, mover.rmCalls, 1, "compensating RemoveObject must run after upsert failure")
	assert.Equal(t, mover.putCalls[0].bucket, mover.rmCalls[0].bucket)
	assert.Equal(t, mover.putCalls[0].object, mover.rmCalls[0].object)
}

func TestImportFromUpload_PutObjectBodyBytesMatch(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	payload := bytes.Repeat([]byte{0xAB}, 1024)
	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.nv", Content: payload}}, "")
	require.NoError(t, err)
	assert.Empty(t, res.Failed)

	require.Len(t, mover.putCalls, 1)
	assert.Equal(t, payload, mover.putCalls[0].body)
	assert.EqualValues(t, len(payload), mover.putCalls[0].size)
}

// ─────────────────────────────────────────────────────────────────────────
// Delete tests
// ─────────────────────────────────────────────────────────────────────────

func TestDelete_RemovesMinIOAndDB(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	// 先种一条
	require.NoError(t, repo.Upsert(context.Background(), &ConfigSnapshot{
		SerialNumber: "SN001",
		FileName:     "SN001_CFG.xml",
		FileExt:      "xml",
		ObjectBucket: "config-snapshots",
		ObjectPath:   "SN001_CFG.xml",
		Source:       SnapshotSourceManualUpload,
	}))

	require.NoError(t, svc.Delete(context.Background(), "SN001"))
	assert.NotContains(t, repo.rows, "SN001")
	require.Len(t, mover.rmCalls, 1)
	assert.Equal(t, "config-snapshots", mover.rmCalls[0].bucket)
	assert.Equal(t, "SN001_CFG.xml", mover.rmCalls[0].object)
}

func TestDelete_MinIOFailureStillDeletesDB(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{removeErr: errors.New("minio offline")}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)
	require.NoError(t, repo.Upsert(context.Background(), &ConfigSnapshot{
		SerialNumber: "SN001",
		FileName:     "SN001_CFG.xml",
		FileExt:      "xml",
		ObjectBucket: "config-snapshots",
		ObjectPath:   "SN001_CFG.xml",
		Source:       SnapshotSourceManualUpload,
	}))

	require.NoError(t, svc.Delete(context.Background(), "SN001"))
	assert.NotContains(t, repo.rows, "SN001", "DB delete must proceed even when MinIO Remove fails")
}

func TestDelete_NonExistentSN_NoOp(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	require.NoError(t, svc.Delete(context.Background(), "ghost"))
	assert.Empty(t, mover.rmCalls, "no MinIO Remove if row doesn't exist")
}

// ─────────────────────────────────────────────────────────────────────────
// BatchGet / read proxies
// ─────────────────────────────────────────────────────────────────────────

func TestBatchGetBySerialNumbers_MissingReportedByOmission(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)
	require.NoError(t, repo.Upsert(context.Background(), &ConfigSnapshot{
		SerialNumber: "SN001",
		FileName:     "SN001_CFG.xml", FileExt: "xml",
		ObjectBucket: "config-snapshots", ObjectPath: "SN001_CFG.xml",
		Source: SnapshotSourceManualUpload,
	}))

	out, err := svc.BatchGetBySerialNumbers(context.Background(),
		[]string{"SN001", "SN_MISSING"})
	require.NoError(t, err)
	assert.Contains(t, out, "SN001")
	assert.NotContains(t, out, "SN_MISSING",
		"BatchGet must omit missing SNs so caller can identify reject set")
}

// ─────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────

func ptrStr(s string) *string { return &s }
