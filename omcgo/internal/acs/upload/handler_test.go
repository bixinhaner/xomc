package upload

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/backup"
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

func TestMaybeWrapForCompression_lz4PassThrough(t *testing.T) {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = true
	pol.CompressionFormat = "lz4"
	h := newTestHandler(t, &fakePolicyGetter{policy: pol})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, strings.NewReader("payload"))
	assert.False(t, w.applied, "lz4 (not implemented) must pass through")
}

func TestMaybeWrapForCompression_bzip2PassThrough(t *testing.T) {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = true
	pol.CompressionFormat = "bzip2"
	h := newTestHandler(t, &fakePolicyGetter{policy: pol})
	w := h.maybeWrapForCompression(context.Background(), tr069.FileTypeConfig, strings.NewReader("payload"))
	assert.False(t, w.applied, "bzip2 (not implemented) must pass through")
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

func enabledGzipPolicy() *backup.BackupPolicy {
	pol := backup.DefaultPolicy()
	pol.EnableCompression = true
	pol.CompressionFormat = "gzip"
	pol.CompressionLevel = 6
	return pol
}
