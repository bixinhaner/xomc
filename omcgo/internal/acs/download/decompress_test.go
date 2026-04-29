package download

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCompression(t *testing.T) {
	tests := []struct {
		path        string
		wantOK      bool
		wantFormat  string
		wantClean   string
	}{
		{"backup/2026/04/29/cfg.xml.gz", true, "gzip", "backup/2026/04/29/cfg.xml"},
		{"backup/cfg.xml.zst", true, "zstd", "backup/cfg.xml"},
		{"backup/cfg.xml.lz4", true, "lz4", "backup/cfg.xml"},
		{"backup/cfg.xml.bz2", true, "bzip2", "backup/cfg.xml"},
		{"backup/cfg.xml", false, "", "backup/cfg.xml"},
		{"firmware/v2.0.bin", false, "", "firmware/v2.0.bin"},
		{"backup/cfg.xml.gZ", false, "", "backup/cfg.xml.gZ"}, // case-sensitive
	}
	for _, tc := range tests {
		decomp, clean, ok := detectCompression(tc.path)
		assert.Equal(t, tc.wantOK, ok, "path=%s ok mismatch", tc.path)
		assert.Equal(t, tc.wantClean, clean, "path=%s clean mismatch", tc.path)
		if tc.wantOK {
			require.NotNil(t, decomp)
			assert.Equal(t, tc.wantFormat, decomp.Format())
		}
	}
}

// roundTripPlaintext compresses src into the named format and feeds the
// resulting bytes back through the matching Decompressor, asserting equality.
// Verifies that download decompression interoperates with the encoders shipped
// in T-0074 (gzip, zstd) and T-0077 (lz4, bzip2).
func roundTripPlaintext(t *testing.T, format string, plaintext []byte) {
	t.Helper()
	var compressed bytes.Buffer
	switch format {
	case "gzip":
		gz := gzip.NewWriter(&compressed)
		_, err := gz.Write(plaintext)
		require.NoError(t, err)
		require.NoError(t, gz.Close())
	case "zstd":
		enc, err := zstd.NewWriter(&compressed)
		require.NoError(t, err)
		_, err = enc.Write(plaintext)
		require.NoError(t, err)
		require.NoError(t, enc.Close())
	case "lz4":
		lzw := lz4.NewWriter(&compressed)
		_, err := lzw.Write(plaintext)
		require.NoError(t, err)
		require.NoError(t, lzw.Close())
	case "bzip2":
		bzw, err := bzip2.NewWriter(&compressed, &bzip2.WriterConfig{Level: 6})
		require.NoError(t, err)
		_, err = bzw.Write(plaintext)
		require.NoError(t, err)
		require.NoError(t, bzw.Close())
	default:
		t.Fatalf("unknown format %q", format)
	}

	decomp, _, ok := detectCompression("foo." + extFor(format))
	require.True(t, ok, "format=%s detect failed", format)

	rc, err := decomp.Wrap(bytes.NewReader(compressed.Bytes()))
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got, "format=%s round-trip mismatch", format)
}

func extFor(format string) string {
	switch format {
	case "gzip":
		return "gz"
	case "zstd":
		return "zst"
	case "lz4":
		return "lz4"
	case "bzip2":
		return "bz2"
	}
	return ""
}

func TestDecompressRoundTrip_allFormats(t *testing.T) {
	plaintext := []byte(strings.Repeat("backup config payload ", 256))
	for _, format := range []string{"gzip", "zstd", "lz4", "bzip2"} {
		format := format
		t.Run(format, func(t *testing.T) {
			roundTripPlaintext(t, format, plaintext)
		})
	}
}

// TestDecompressInvalidStream — a malformed compressed stream surfaces as a
// Wrap error (gzip/zstd only; lz4/bzip2 readers defer parse errors to Read).
func TestDecompressInvalidStream(t *testing.T) {
	garbage := bytes.NewReader([]byte("this is definitely not gzip"))
	d := gzipDecompressor{}
	_, err := d.Wrap(garbage)
	require.Error(t, err)
}

func TestDecompressMetrics_nilSafe(t *testing.T) {
	var m *DecompressMetrics
	m.RecordSuccess("gzip")    // must not panic on nil receiver
	m.RecordError("zstd", "open")
}
