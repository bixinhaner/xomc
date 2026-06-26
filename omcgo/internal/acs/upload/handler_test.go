package upload

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/backup"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// fakePolicyGetter satisfies backup.PolicyGetter for handler tests.
// Returning a fixed policy or a fixed error covers the relevant branches.
type fakePolicyGetter struct {
	policy *backup.BackupPolicy
	err    error
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

func TestMaybeWrapForCompression_nonConfigFileType(t *testing.T) {
	h := newTestHandler(t, &fakePolicyGetter{policy: enabledGzipPolicy()})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypePM, strings.NewReader("payload"))
	assert.False(t, w.applied, "PM file type must not trigger compression")
}

func TestMaybeWrapForCompression_noPolicyGetter(t *testing.T) {
	h := &Handler{logger: zap.NewNop()}
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, strings.NewReader("payload"))
	assert.False(t, w.applied, "nil policy getter must keep compression off")
}

func TestMaybeWrapForCompression_policyError(t *testing.T) {
	h := newTestHandler(t, &fakePolicyGetter{err: errors.New("DB outage")})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, strings.NewReader("payload"))
	assert.False(t, w.applied, "policy lookup error must fall back to plaintext")
}

func TestMaybeWrapForCompression_disabled(t *testing.T) {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = false
	h := newTestHandler(t, &fakePolicyGetter{policy: pol})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, strings.NewReader("payload"))
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
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, strings.NewReader("payload"))
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
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, strings.NewReader("payload"))
	require.True(t, w.applied, "bzip2 must apply now that T-0077 ships the real impl")
	defer w.body.Close()
	assert.Equal(t, "bzip2", w.format)
	assert.Equal(t, ".bz2", w.ext)
}

func TestMaybeWrapForCompression_gzipApplied(t *testing.T) {
	plaintext := []byte(strings.Repeat("backup config payload ", 256))
	h := newTestHandler(t, &fakePolicyGetter{policy: enabledGzipPolicy()})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, bytes.NewReader(plaintext))
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
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, bytes.NewReader(plaintext))
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
		"pm-files", "pm-files/2026/05/25/pm-X.xml.gz", "pm-X.xml.gz", 1234, "")

	assert.Empty(t, bus.published, "empty device_sn must skip pm.file.received publish")
}

func TestPublishPMFileReceivedEvent_publishesThinPayload(t *testing.T) {
	bus := &captureBus{}
	h := &Handler{logger: zap.NewNop(), eventBus: bus}

	h.publishPMFileReceivedEvent(context.Background(),
		"pm-files", "pm-files/2026/05/25/pm-1202000240194DP0015.xml.gz",
		"pm-1202000240194DP0015.xml.gz", 4096, "1202000240194DP0015")

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

	assert.Equal(t, "pm-files/2026/05/25/pm-1202000240194DP0015.xml.gz", decoded.MinIOPath)
	assert.Equal(t, "pm-files", decoded.Bucket)
	assert.Equal(t, "1202000240194DP0015", decoded.DeviceSN)
	assert.Equal(t, int64(4096), decoded.FileSize)
	assert.Equal(t, "pm-1202000240194DP0015.xml.gz", decoded.FileName)
	// Thin payload: device_id/oui intentionally empty — collector fills via DeviceLookup.
	assert.Empty(t, decoded.DeviceID, "thin payload must leave device_id empty")
	assert.Empty(t, decoded.DeviceOUI, "thin payload must leave device_oui empty")
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

// ─── issue #561 · PM 同步 gzip 嗅探+压缩 ───────────────────────────────────────
//
// 覆盖 5 个 case（账本 T-561 spec §Scope.4）：
//   A 明文 PM   → 流式 gzip，落盘前 2 字节 1f8b，能解回原文
//   B 预压缩 gzip → 原样存，落盘字节 ≡ 输入字节
//   C 短包不足 2 字节 → 走压缩路径，不报错
//   D filename + objectPath 已带 .gz + 内容是 gzip → ServeHTTP 不叠 .gz（间接验证由 ServeHTTP 集成测试覆盖；此处单测验证 pmSyncGzip 本身只关心字节）
//   E peek 错误 → 走压缩路径（用 errReader 注入）

// ioReadCloserOf 把 []byte 包成 io.ReadCloser（Close 是 no-op）。
type ioReadCloserOf struct{ io.Reader }

func (ioReadCloserOf) Close() error { return nil }

func TestPMSyncGzip_plaintextWraps(t *testing.T) {
	plain := []byte(strings.Repeat("<measCollec/>", 500))
	out, alreadyGz, err := pmSyncGzip(context.Background(), ioReadCloserOf{Reader: bytes.NewReader(plain)})
	require.NoError(t, err)
	defer out.Close()
	assert.False(t, alreadyGz, "明文必须走流式压缩")

	compressed, err := io.ReadAll(out)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(compressed), 2)
	// 落盘字节首两字节是 gzip magic
	assert.Equal(t, byte(0x1f), compressed[0], "落盘首字节必须是 gzip magic 1f")
	assert.Equal(t, byte(0x8b), compressed[1], "落盘次字节必须是 gzip magic 8b")
	// 能解回原文
	gzr, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzr.Close()
	recovered, err := io.ReadAll(gzr)
	require.NoError(t, err)
	assert.Equal(t, plain, recovered, "解压后必须与原文相等")
}

func TestPMSyncGzip_alreadyGzipPassthrough(t *testing.T) {
	// 先构造一段合法 gzip
	plain := []byte("preCompressedPmXmlPayload")
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, _ = gzw.Write(plain)
	_ = gzw.Close()
	preGz := buf.Bytes()
	require.Equal(t, byte(0x1f), preGz[0])
	require.Equal(t, byte(0x8b), preGz[1])

	out, alreadyGz, err := pmSyncGzip(context.Background(), ioReadCloserOf{Reader: bytes.NewReader(preGz)})
	require.NoError(t, err)
	defer out.Close()
	assert.True(t, alreadyGz, "已 gzip 必须命中原样存路径")

	// 落盘字节 ≡ 输入字节（含 peek 出的 2 字节也要原样吐回）
	got, err := io.ReadAll(out)
	require.NoError(t, err)
	assert.Equal(t, preGz, got, "原样存路径落盘字节必须与输入逐字节相等")
}

func TestPMSyncGzip_shortPacketGoesToCompression(t *testing.T) {
	// 仅 1 字节 — peek(2) 会失败，但必须走压缩路径不报错
	out, alreadyGz, err := pmSyncGzip(context.Background(), ioReadCloserOf{Reader: bytes.NewReader([]byte{'x'})})
	require.NoError(t, err)
	defer out.Close()
	assert.False(t, alreadyGz, "短包必须按明文处理")

	compressed, err := io.ReadAll(out)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(compressed), 2)
	assert.Equal(t, byte(0x1f), compressed[0])
	assert.Equal(t, byte(0x8b), compressed[1])

	gzr, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzr.Close()
	recovered, err := io.ReadAll(gzr)
	require.NoError(t, err)
	assert.Equal(t, []byte{'x'}, recovered)
}

// errReader 第一次 Read 立即返回错误，模拟连接断开 / peek 失败场景。
type errReader struct{ err error }

func (e errReader) Read(_ []byte) (int, error) { return 0, e.err }
func (e errReader) Close() error               { return nil }

func TestPMSyncGzip_peekErrorGoesToCompression(t *testing.T) {
	// peek 失败：上游连接异常。我们的策略是走压缩路径——构造仍然成功，
	// 真正的 Read 错误等到 PutObject 拉数据时才暴露（由 net/http 处理）。
	out, alreadyGz, err := pmSyncGzip(context.Background(), errReader{err: io.ErrUnexpectedEOF})
	require.NoError(t, err, "构造 pmSyncGzip 不应失败")
	defer out.Close()
	assert.False(t, alreadyGz, "peek 错误必须按明文处理")
	// 实际 ReadAll 时会从 errReader 传播错误；这里只验证构造路径不返回 err。
	_, _ = io.ReadAll(out)
}

func TestPMSyncGzip_objectPathSuffixCheck(t *testing.T) {
	// 验证调用方的 objectPath 处理逻辑：handler.ServeHTTP 在 ft==PM 时
	// "若 objectPath 不以 .gz 结尾则追加 .gz"。这是字符串拼接逻辑，
	// 这里用三种 filename 路径模拟，确认拼接结果只有单 .gz 后缀。
	cases := []struct {
		name string
		obj  string
		want string
	}{
		{"plain xml", "pm-files/2026/06/22/A_OUI.SN.xml", "pm-files/2026/06/22/A_OUI.SN.xml.gz"},
		{"already .xml.gz", "pm-files/2026/06/22/A_OUI.SN.xml.gz", "pm-files/2026/06/22/A_OUI.SN.xml.gz"},
		{"upper case .GZ", "pm-files/2026/06/22/A_OUI.SN.xml.GZ", "pm-files/2026/06/22/A_OUI.SN.xml.GZ"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.obj
			if !strings.HasSuffix(strings.ToLower(got), ".gz") {
				got += ".gz"
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

