package upload

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// fakePolicyGetter satisfies backup.PolicyGetter for handler tests.
// Returning a fixed policy or a fixed error covers the relevant branches.
type fakePolicyGetter struct {
	policy *backup.BackupPolicy
	err    error
}

type deadlineRecordingResponseWriter struct {
	*httptest.ResponseRecorder
	readDeadlines  []time.Time
	writeDeadlines []time.Time
}

func (w *deadlineRecordingResponseWriter) SetReadDeadline(t time.Time) error {
	w.readDeadlines = append(w.readDeadlines, t)
	return nil
}

func (w *deadlineRecordingResponseWriter) SetWriteDeadline(t time.Time) error {
	w.writeDeadlines = append(w.writeDeadlines, t)
	return nil
}

func TestServeHTTP_ClearsDeadlinesForLargeFileUploads(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodPost, "/smallcell/FileUploadService", strings.NewReader("payload"))
	rec := &deadlineRecordingResponseWriter{ResponseRecorder: httptest.NewRecorder()}

	h.ServeHTTP(rec, req)

	require.NotEmpty(t, rec.readDeadlines)
	require.True(t, rec.readDeadlines[0].IsZero())
	require.NotEmpty(t, rec.writeDeadlines)
	require.True(t, rec.writeDeadlines[0].IsZero())
}

func (f *fakePolicyGetter) Get(_ context.Context) (*backup.BackupPolicy, error) {
	return f.policy, f.err
}

func newTestHandler(t *testing.T, getter backup.PolicyGetter) *Handler {
	t.Helper()
	h := &Handler{
		logger: zap.NewNop(),
	}
	h.SetCompression(getter, backup.NewPolicyMetrics(nil))
	return h
}

func TestMaybeWrapForCompression_unrelatedFileType(t *testing.T) {
	// RunningLog is neither FileTypeConfig (policy-gated backup compression)
	// nor FileTypePM/FileTypeMR (unconditional PM/MR compression) — must stay
	// plaintext.
	h := newTestHandler(t, &fakePolicyGetter{policy: enabledGzipPolicy()})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeRunningLog, "report.xml", strings.NewReader("payload"))
	assert.False(t, w.applied, "unrelated file type must not trigger compression")
}

func TestMaybeWrapForCompression_mrUnconditionalGzip(t *testing.T) {
	// MR shares PM's unconditional gzip treatment (2026-07-21): same
	// high-frequency, structured, repetitive shape, same compressPMUpload path.
	h := newTestHandler(t, nil)
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeMR, "report.xml", strings.NewReader("payload"))
	assert.True(t, w.applied, "MR file type must trigger unconditional gzip like PM")
	assert.Equal(t, "gzip", w.format)
}

func TestMaybeWrapForCompression_noPolicyGetter(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, "backup.xml", strings.NewReader("payload"))
	assert.False(t, w.applied, "nil policy getter must keep compression off")
}

func TestMaybeWrapForCompression_policyError(t *testing.T) {
	h := newTestHandler(t, &fakePolicyGetter{err: errors.New("DB outage")})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, "backup.xml", strings.NewReader("payload"))
	assert.False(t, w.applied, "policy lookup error must fall back to plaintext")
}

func TestMaybeWrapForCompression_disabled(t *testing.T) {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = false
	h := newTestHandler(t, &fakePolicyGetter{policy: pol})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, "backup.xml", strings.NewReader("payload"))
	assert.False(t, w.applied, "EnableCompression=false must keep plaintext")
}

// T-0077: lz4 / bzip2 now apply (no longer pass through). The pass-through
// tests were inverted — both formats now exercise the same applied=true path
// as gzip/zstd, exercised end-to-end by TestMaybeWrapForCompression_lz4Applied
// and _bzip2Applied below.

func TestMaybeWrapForCompression_lz4Applied(t *testing.T) {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = true
	pol.CompressionFormat = "lz4"
	h := newTestHandler(t, &fakePolicyGetter{policy: pol})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, "backup.xml", strings.NewReader("payload"))
	require.True(t, w.applied, "lz4 must apply now that T-0077 ships the real impl")
	defer w.body.Close()
	assert.Equal(t, "lz4", w.format)
	assert.Equal(t, ".lz4", w.ext)
}

func TestMaybeWrapForCompression_bzip2Applied(t *testing.T) {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = true
	pol.CompressionFormat = "bzip2"
	h := newTestHandler(t, &fakePolicyGetter{policy: pol})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, "backup.xml", strings.NewReader("payload"))
	require.True(t, w.applied, "bzip2 must apply now that T-0077 ships the real impl")
	defer w.body.Close()
	assert.Equal(t, "bzip2", w.format)
	assert.Equal(t, ".bz2", w.ext)
}

func TestMaybeWrapForCompression_gzipApplied(t *testing.T) {
	plaintext := []byte(strings.Repeat("backup config payload ", 256))
	h := newTestHandler(t, &fakePolicyGetter{policy: enabledGzipPolicy()})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, "backup.xml", bytes.NewReader(plaintext))
	require.True(t, w.applied, "gzip compression must apply for FileTypeConfig + EnableCompression=true")
	defer w.body.Close()

	assert.Equal(t, "gzip", w.format)
	assert.Equal(t, ".gz", w.ext)

	compressed, err := io.ReadAll(w.body)
	require.NoError(t, err)

	// Sanity: gzip magic bytes
	require.GreaterOrEqual(t, len(compressed), 2)
	assert.Equal(t, byte(0x1f), compressed[0])
	assert.Equal(t, byte(0x8b), compressed[1])

	// Round-trip must recover plaintext
	gzr, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzr.Close()
	recovered, err := io.ReadAll(gzr)
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered)

	// bytesIn() reflects the plaintext we read through the counter.
	assert.Equal(t, int64(len(plaintext)), w.bytesIn())
}

func TestMaybeWrapForCompression_zstdApplied(t *testing.T) {
	plaintext := []byte(strings.Repeat("backup config payload ", 256))
	pol := backup.DefaultPolicy()
	pol.EnableCompression = true
	pol.CompressionFormat = "zstd"
	pol.CompressionLevel = 9

	h := newTestHandler(t, &fakePolicyGetter{policy: pol})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, "backup.xml", bytes.NewReader(plaintext))
	require.True(t, w.applied)
	defer w.body.Close()

	assert.Equal(t, "zstd", w.format)
	assert.Equal(t, ".zst", w.ext)

	compressed, err := io.ReadAll(w.body)
	require.NoError(t, err)

	dec, err := zstd.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer dec.Close()
	recovered, err := io.ReadAll(dec)
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered)
}

func TestMaybeWrapForCompression_pmUploadCompressesUnconditionally(t *testing.T) {
	plaintext := []byte(strings.Repeat("<counter name=\"x\">1</counter>", 256))
	// No policyGetter wired at all — PM compression must not depend on the
	// backup sys_configs policy machinery.
	h := &Handler{logger: zap.NewNop()}

	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypePM, "A20260718.0010-0800-0015-0800_48BF74.SN1.xml", bytes.NewReader(plaintext))
	require.True(t, w.applied, "PM upload must be gzip-compressed unconditionally")
	defer w.body.Close()

	assert.Equal(t, "gzip", w.format)
	assert.Equal(t, ".gz", w.ext)

	compressed, err := io.ReadAll(w.body)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(compressed), 2)
	assert.Equal(t, byte(0x1f), compressed[0], "must be valid gzip magic byte 1")
	assert.Equal(t, byte(0x8b), compressed[1], "must be valid gzip magic byte 2")
	assert.Less(t, len(compressed), len(plaintext), "repetitive PM XML must actually shrink")

	gzr, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzr.Close()
	recovered, err := io.ReadAll(gzr)
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered, "must round-trip back to the exact original bytes")
}

func TestMaybeWrapForCompression_pmUploadAlreadyGzipSkipsDoubleCompression(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}

	// issue #321: real CPEs / the simulator sometimes upload already-gzipped
	// PM files (filename ends in .gz). Must not wrap gzip-in-gzip.
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypePM, "A20260718.0010-0800-0015-0800_48BF74.SN1.xml.gz", strings.NewReader("already gzip bytes"))
	assert.False(t, w.applied, "PM filename already ending in .gz must not be re-compressed")
}

func TestCountingReader(t *testing.T) {
	src := strings.NewReader("hello world") // 11 bytes
	c := &countingReader{r: src}

	buf := make([]byte, 5)
	n, err := c.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(5), c.n.Load())

	rest, err := io.ReadAll(c)
	require.NoError(t, err)
	assert.Equal(t, " world", string(rest))
	assert.Equal(t, int64(11), c.n.Load())
}

func TestDeriveUploadFilename_FaultLogUsesSNAndMillisecondTimestamp(t *testing.T) {
	got := deriveUploadFilename("RL", "dbc91d19-6364-4d3f-97bc-ae5d0b6d17f3", "SN-ABC")

	assert.Regexp(t, `^fault-SN-ABC-\d{17}\.tar\.gz$`, got)
	assert.NotContains(t, got, "dbc91d19", "fault log file name should not expose task id")
}

func TestBuildUploadObjectPath_FaultLogOmitsTaskSubdir(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 34, 56, 789*int(time.Millisecond), time.UTC)

	got := buildUploadObjectPath(
		tr069.FileTypeFaultLog,
		"fault",
		now,
		"dbc91d19-6364-4d3f-97bc-ae5d0b6d17f3",
		"fault-SN-ABC-20260715123456789.tar.gz",
	)

	assert.Equal(t, "fault/2026/07/15/fault-SN-ABC-20260715123456789.tar.gz", got)
	assert.NotContains(t, got, "/dbc91d19/", "fault log path should be readable without task-id directory")
}

func TestBuildUploadObjectPath_RunningLogOmitsTaskSubdir(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 34, 56, 0, time.UTC)

	got := buildUploadObjectPath(
		tr069.FileTypeRunningLog,
		"running",
		now,
		"dbc91d19-6364-4d3f-97bc-ae5d0b6d17f3",
		"runtime-dbc91d19-SN-ABC.tar.gz",
	)

	assert.Equal(t, "running/2026/07/15/runtime-dbc91d19-SN-ABC.tar.gz", got)
}

func TestBuildUploadObjectPath_ConfigBackupKeepsTaskSubdir(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 34, 56, 0, time.UTC)

	got := buildUploadObjectPath(
		tr069.FileTypeConfig,
		"backup",
		now,
		"dbc91d19-6364-4d3f-97bc-ae5d0b6d17f3",
		"SN-ABC_CFG.xml",
	)

	assert.Equal(t, "backup/2026/07/15/dbc91d19/SN-ABC_CFG.xml", got)
}

// captureBus records every published event for assertions.
type captureBus struct {
	mu        sync.Mutex
	published []struct {
		subject string
		evt     event.Event
	}
	publishErr error
}

func (c *captureBus) Publish(_ context.Context, subject string, evt event.Event) error {
	if c.publishErr != nil {
		return c.publishErr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.published = append(c.published, struct {
		subject string
		evt     event.Event
	}{subject, evt})
	return nil
}

func (c *captureBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (c *captureBus) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (c *captureBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return nil, nil
}

func (c *captureBus) Close() error { return nil }

func TestPublishPMFileReceivedEvent_emptySNSkipsPublish(t *testing.T) {
	bus := &captureBus{}
	h := &Handler{logger: zap.NewNop(), eventBus: bus}

	h.publishPMFileReceivedEvent(context.Background(),
		"pm-files", "pm-files/2026/05/25/pm-X.xml", "pm-X.xml", 1234, "")

	assert.Empty(t, bus.published, "empty device_sn must skip pm.file.received publish")
}

func TestPublishPMFileReceivedEvent_publishesThinPayload(t *testing.T) {
	bus := &captureBus{}
	h := &Handler{logger: zap.NewNop(), eventBus: bus}

	h.publishPMFileReceivedEvent(context.Background(),
		"pm-files", "pm-files/2026/05/25/pm-1202000240194DP0015.xml",
		"pm-1202000240194DP0015.xml", 4096, "1202000240194DP0015")

	require.Len(t, bus.published, 1)
	got := bus.published[0]
	assert.Equal(t, event.SubjectPMFileReceived, got.subject)

	// Decode payload via the wire-format the collector uses.
	var decoded struct {
		MinIOPath string `json:"minio_path"`
		Bucket    string `json:"bucket"`
		DeviceSN  string `json:"device_sn"`
		FileSize  int64  `json:"file_size"`
		FileName  string `json:"file_name"`
		DeviceID  string `json:"device_id"`
		DeviceOUI string `json:"device_oui"`
	}
	require.NoError(t, got.evt.DecodePayload(&decoded))

	assert.Equal(t, "pm-files/2026/05/25/pm-1202000240194DP0015.xml", decoded.MinIOPath)
	assert.Equal(t, "pm-files", decoded.Bucket)
	assert.Equal(t, "1202000240194DP0015", decoded.DeviceSN)
	assert.Equal(t, int64(4096), decoded.FileSize)
	assert.Equal(t, "pm-1202000240194DP0015.xml", decoded.FileName)
	// Thin payload: device_id/oui intentionally empty — collector fills via DeviceLookup.
	assert.Empty(t, decoded.DeviceID, "thin payload must leave device_id empty")
	assert.Empty(t, decoded.DeviceOUI, "thin payload must leave device_oui empty")
}

func TestPublishLogFileReceivedEvent_UsesQuerySNForDeviceSuppliedFaultLogName(t *testing.T) {
	bus := &captureBus{}
	h := &Handler{logger: zap.NewNop(), eventBus: bus}

	h.publishLogFileReceivedEvent(context.Background(),
		"logs",
		"fault/2026/07/15/fault-E8F2971A3DC921A03D3E4FD4A0C1-20260715153900123.tar.gz",
		"fault-E8F2971A3DC921A03D3E4FD4A0C1-20260715153900123.tar.gz",
		string(tr069.FileTypeFaultLog),
		1317251,
		"E8F2971A3DC921A03D3E4FD4A0C1",
		"dbc91d19-6364-4d3f-97bc-ae5d0b6d17f3",
	)

	require.Len(t, bus.published, 1)
	got := bus.published[0]
	assert.Equal(t, event.SubjectLogFileReceived, got.subject)

	var decoded struct {
		Bucket     string `json:"bucket"`
		ObjectPath string `json:"object_path"`
		FileName   string `json:"file_name"`
		FileType   string `json:"file_type"`
		FileSize   int64  `json:"file_size"`
		TaskID8    string `json:"task_id8"`
		DeviceSN   string `json:"device_sn"`
	}
	require.NoError(t, got.evt.DecodePayload(&decoded))
	assert.Equal(t, "logs", decoded.Bucket)
	assert.Equal(t, "fault/2026/07/15/fault-E8F2971A3DC921A03D3E4FD4A0C1-20260715153900123.tar.gz", decoded.ObjectPath)
	assert.Equal(t, "fault-E8F2971A3DC921A03D3E4FD4A0C1-20260715153900123.tar.gz", decoded.FileName)
	assert.Equal(t, string(tr069.FileTypeFaultLog), decoded.FileType)
	assert.Equal(t, int64(1317251), decoded.FileSize)
	assert.Equal(t, "dbc91d19", decoded.TaskID8)
	assert.Equal(t, "E8F2971A3DC921A03D3E4FD4A0C1", decoded.DeviceSN)
}

func TestPublishLogFileReceivedEvent_FaultLogPrefersQueryIdentityOverFilename(t *testing.T) {
	bus := &captureBus{}
	h := &Handler{logger: zap.NewNop(), eventBus: bus}

	h.publishLogFileReceivedEvent(context.Background(),
		"logs",
		"fault/2026/07/15/fault-12345678-SN-FROM-NAME.tar.gz",
		"fault-12345678-SN-FROM-NAME.tar.gz",
		string(tr069.FileTypeFaultLog),
		1024,
		"SN-FROM-QUERY",
		"abcdef01-6364-4d3f-97bc-ae5d0b6d17f3",
	)

	require.Len(t, bus.published, 1)
	var decoded struct {
		TaskID8  string `json:"task_id8"`
		DeviceSN string `json:"device_sn"`
	}
	require.NoError(t, bus.published[0].evt.DecodePayload(&decoded))
	assert.Equal(t, "abcdef01", decoded.TaskID8)
	assert.Equal(t, "SN-FROM-QUERY", decoded.DeviceSN)
}

func TestExtractDeviceSNFromPMFilename(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "Baicells real CPE format (Beijing time window, .xml)",
			filename: "A20260525.1630+0800-1645+0800_48BF74.1202000240194DP0015.xml",
			want:     "1202000240194DP0015",
		},
		{
			name:     "Baicells real CPE format with .xml.gz",
			filename: "A20260525.1630+0800-1645+0800_48BF74.1202000240194DP0015.xml.gz",
			want:     "1202000240194DP0015",
		},
		{
			name:     "cpe_simulator pm-{SN}.xml.gz pattern",
			filename: "pm-1202000240194DP0015.xml.gz",
			want:     "1202000240194DP0015",
		},
		{
			name:     "cpe_simulator pm-{SN}.xml pattern (no compression)",
			filename: "pm-G1-VALIDATE-001.xml",
			want:     "G1-VALIDATE-001",
		},
		// #364: BSC / 2G GSM real-CPE dialect — 3GPP A-form without an {OUI} vendor
		// tag (no underscore-OUI segment). The SN is the last dot-segment.
		{
			name:     "BSC/GSM A-form without OUI tag (.xml)",
			filename: "A20260615.0000-0015.B54DEF3B1F059A9077B8998FD274.xml",
			want:     "B54DEF3B1F059A9077B8998FD274",
		},
		{
			name:     "BSC/GSM A-form without OUI tag (.xml.gz)",
			filename: "A20260615.0000-0015.B54DEF3B1F059A9077B8998FD274.xml.gz",
			want:     "B54DEF3B1F059A9077B8998FD274",
		},
		// #364: vendor PM-prefix dialects (uppercase / underscore / lowercase underscore).
		{
			name:     "vendor PM_{SN}.xml dialect",
			filename: "PM_B54DEF3B1F059A9077B8998FD274.xml",
			want:     "B54DEF3B1F059A9077B8998FD274",
		},
		{
			name:     "vendor PM-{SN}.xml.gz dialect",
			filename: "PM-B54DEF3B1F059A9077B8998FD274.xml.gz",
			want:     "B54DEF3B1F059A9077B8998FD274",
		},
		{
			name:     "simulator pm_{SN}.xml underscore dialect",
			filename: "pm_B54DEF3B1F059A9077B8998FD274.xml",
			want:     "B54DEF3B1F059A9077B8998FD274",
		},
		{
			name:     "unknown pattern returns empty (caller will WARN-skip)",
			filename: "random-filename.xml",
			want:     "",
		},
		// #364: bare SN and SN-prefix forms remain ambiguous (no vendor marker) and
		// are intentionally NOT matched — the safe path for those is the ?sn= URL query.
		{
			name:     "bare {SN}.xml stays unmatched (ambiguous, use ?sn=)",
			filename: "B54DEF3B1F059A9077B8998FD274.xml",
			want:     "",
		},
		{
			name:     "SN-prefix {SN}_date.xml stays unmatched (ambiguous, use ?sn=)",
			filename: "B54DEF3B1F059A9077B8998FD274_20260615.xml",
			want:     "",
		},
		{
			name:     "single-segment A{x}.xml stays unmatched (needs >=2 dots)",
			filename: "Anything.xml",
			want:     "",
		},
		{
			name:     "empty filename",
			filename: "",
			want:     "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, extractDeviceSNFromPMFilename(tc.filename))
		})
	}
}

func TestPublishPMFileReceivedEvent_publishErrorDoesNotPanic(t *testing.T) {
	// Defensive: even if NATS is down, publish failure must not bring down
	// the upload request (5xx on a successful MinIO write is worse than a
	// silently-skipped event — the CPE will re-upload on next cycle).
	bus := &captureBus{publishErr: errors.New("NATS down")}
	h := &Handler{logger: zap.NewNop(), eventBus: bus}

	// Should log error but not panic.
	assert.NotPanics(t, func() {
		h.publishPMFileReceivedEvent(context.Background(),
			"pm-files", "obj/path.xml.gz", "f.xml.gz", 1, "SN1")
	})
	_ = time.Millisecond // keep import alive
}

func enabledGzipPolicy() *backup.BackupPolicy {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = true
	pol.CompressionFormat = "gzip"
	pol.CompressionLevel = 6
	return pol
}

func TestPMUploadEventKeepsPlainXMLPath(t *testing.T) {
	bus := &captureBus{}
	h := &Handler{logger: zap.NewNop(), eventBus: bus}

	h.publishPMFileReceivedEvent(context.Background(),
		"pm-files", "pm/2026/07/03/A_48BF74.1202000240194DP0015.xml",
		"A_48BF74.1202000240194DP0015.xml", 2048, "1202000240194DP0015")

	require.Len(t, bus.published, 1)
	var decoded struct {
		MinIOPath string `json:"minio_path"`
		FileName  string `json:"file_name"`
	}
	require.NoError(t, bus.published[0].evt.DecodePayload(&decoded))
	assert.Equal(t, "pm/2026/07/03/A_48BF74.1202000240194DP0015.xml", decoded.MinIOPath)
	assert.Equal(t, "A_48BF74.1202000240194DP0015.xml", decoded.FileName)
	assert.NotContains(t, decoded.MinIOPath, ".xml.gz", "ACS must not publish a pre-ingest gzip path for plaintext PM XML")
}

// newTestDeduper 起一个 miniredis 支撑的真实 event.Deduper，用于校验去重的
// SETNX+TTL 行为（而不是 mock 掉 Redis 交互）。
func newTestDeduper(t *testing.T) *event.Deduper {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return event.NewDeduper(rdb, time.Hour, nil)
}

func TestCheckPMUploadDuplicate_FirstSeenNotDuplicate(t *testing.T) {
	h := &Handler{logger: zap.NewNop(), pmDedup: newTestDeduper(t)}

	skip := h.checkPMUploadDuplicate(context.Background(), "SN1", "A20260718.0010-0800-0015-0800_48BF74.SN1.xml")

	assert.False(t, skip, "first time seeing this (device_sn, filename) must not be treated as duplicate")
}

func TestCheckPMUploadDuplicate_RepeatIsDuplicate(t *testing.T) {
	h := &Handler{logger: zap.NewNop(), pmDedup: newTestDeduper(t)}
	ctx := context.Background()
	sn, filename := "SN1", "A20260718.0010-0800-0015-0800_48BF74.SN1.xml"

	require.False(t, h.checkPMUploadDuplicate(ctx, sn, filename), "first call must pass through")
	assert.True(t, h.checkPMUploadDuplicate(ctx, sn, filename), "second call for the same device_sn+filename must be flagged as duplicate")
}

func TestCheckPMUploadDuplicate_DifferentFilenameNotDuplicate(t *testing.T) {
	h := &Handler{logger: zap.NewNop(), pmDedup: newTestDeduper(t)}
	ctx := context.Background()

	require.False(t, h.checkPMUploadDuplicate(ctx, "SN1", "A20260718.0010-0800-0015-0800_48BF74.SN1.xml"))
	// Next reporting period is a different filename — must not be suppressed
	// by the previous period's dedup key.
	assert.False(t, h.checkPMUploadDuplicate(ctx, "SN1", "A20260718.0015-0800-0020-0800_48BF74.SN1.xml"))
}

func TestCheckPMUploadDuplicate_NilDeduperNeverSkips(t *testing.T) {
	h := &Handler{logger: zap.NewNop()} // pmDedup not wired (nil-safe default)

	assert.False(t, h.checkPMUploadDuplicate(context.Background(), "SN1", "f.xml"),
		"without a wired deduper, PM uploads must never be treated as duplicate")
}

func TestCheckPMUploadDuplicate_EmptySNNeverSkips(t *testing.T) {
	h := &Handler{logger: zap.NewNop(), pmDedup: newTestDeduper(t)}

	assert.False(t, h.checkPMUploadDuplicate(context.Background(), "", "f.xml"),
		"empty device_sn can't form a reliable dedup key, must fail open")
}

// fakeBackpressureGate 让测试可以精确控制 Acquire() 是否放行，不依赖真实
// /proc/pressure/io 或磁盘水位。
type fakeBackpressureGate struct {
	allow    bool
	reason   string
	accepted []time.Time
}

func (f *fakeBackpressureGate) Acquire() (bool, string) {
	if f.allow {
		return true, ""
	}
	return false, f.reason
}
func (f *fakeBackpressureGate) Release()              {}
func (f *fakeBackpressureGate) RecordRejected(string) {}
func (f *fakeBackpressureGate) RecordAccepted(at time.Time) {
	f.accepted = append(f.accepted, at)
}

type fakeUploadObjectStore struct {
	err  error
	puts int
}

func (f *fakeUploadObjectStore) PutObject(
	_ context.Context,
	_, _ string,
	body io.Reader,
	size int64,
	_ minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	f.puts++
	if f.err != nil {
		return minio.UploadInfo{}, f.err
	}
	_, _ = io.Copy(io.Discard, body)
	return minio.UploadInfo{Size: size}, nil
}

// TestServeHTTP_PMUpload_BackpressureRejectionDoesNotPoisonDedup 是 2026-07-20
// omc78 压测环境实测复现的严重 bug 的回归测试：修复前 ServeHTTP 先做 PM 去重检查
// （FirstTime 用 SETNX 把 (device_sn, filename) 标记为"已见过"，24h TTL），再做
// 背压检查；背压生效期间收到的第一次上传会被去重标记为已见过之后才被背压拒收
// （503），导致设备按 TR-069 语义重传时，重传请求被去重短路直接回 200 OK——
// 设备以为上传成功，但这份 PM 文件从未真正落盘/入库，且 24h 内该 (device_sn,
// filename) 都无法再重传成功，等价于背压窗口内的 PM 文件被静默永久丢弃。
//
// 正确顺序应该是：背压检查在前，去重检查在后——背压拒收的请求必须完全不触碰
// 去重层，去重标记只应该在请求真正被接纳、准备落盘时才打上。
func TestServeHTTP_PMUpload_BackpressureRejectionDoesNotPoisonDedup(t *testing.T) {
	gate := &fakeBackpressureGate{allow: false, reason: "resource_pressure"}
	h := &Handler{
		logger:       zap.NewNop(),
		pmDedup:      newTestDeduper(t),
		backpressure: gate,
	}
	const sn = "SN1"
	const filename = "A20260718.0010-0800-0015-0800_48BF74.SN1.xml"

	// 第一次上传：背压生效中，必须被拒收 503。
	req := httptest.NewRequest(http.MethodPost,
		"/smallcell/FileUploadService?fileType=PM&filename="+filename+"&sn="+sn, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code,
		"backpressure must reject the PM upload with 503")
	assert.Empty(t, gate.accepted)

	// 关键断言：被背压拒收的这次请求，绝不能把去重键标记为"已见过"——
	// 否则设备按 TR-069 语义重传时会被去重层错误地当成"已处理过的重复"直接吃掉。
	skip := h.checkPMUploadDuplicate(context.Background(), sn, filename)
	assert.False(t, skip,
		"a request rejected by backpressure must never mark the dedup key; "+
			"otherwise the device's retry gets silently swallowed and the PM file is lost for the 24h TTL")
}

func TestServeHTTP_RecordAcceptedOnlyAfterSuccessfulPMObjectWrite(t *testing.T) {
	tests := []struct {
		name         string
		fileType     string
		storeErr     error
		wantStatus   int
		wantAccepted int
	}{
		{name: "PM storage failure", fileType: "PM", storeErr: errors.New("minio unavailable"), wantStatus: http.StatusInternalServerError},
		{name: "non PM success", fileType: "LOG", wantStatus: http.StatusOK},
		{name: "PM success", fileType: "PM", wantStatus: http.StatusOK, wantAccepted: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate := &fakeBackpressureGate{allow: true}
			store := &fakeUploadObjectStore{err: tt.storeErr}
			h := &Handler{
				logger: zap.NewNop(), minioClient: store,
				buckets:      appconfig.BucketConfig{PMFiles: "pm-files", Logs: "logs"},
				backpressure: gate,
			}
			h.SetCompression(nil, backup.NewPolicyMetrics(nil))
			req := httptest.NewRequest(http.MethodPost,
				"/smallcell/FileUploadService?fileType="+tt.fileType+"&filename=report.xml&sn=SN1",
				strings.NewReader("payload"))
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantAccepted, len(gate.accepted))
			assert.Equal(t, 1, store.puts)
		})
	}
}
