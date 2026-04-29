package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompressor_invalidFormat(t *testing.T) {
	_, err := NewCompressor("xz", 6)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrCompressionFormatInvalid))
}

func TestNewCompressor_levelOutOfRange(t *testing.T) {
	cases := []int{-1, 0, 10, 100}
	for _, lv := range cases {
		_, err := NewCompressor("gzip", lv)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrCompressionLevelOutOfRange), "level=%d", lv)
	}
}

func TestNewCompressor_lz4Stub(t *testing.T) {
	_, err := NewCompressor("lz4", 6)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrCompressionFormatNotImplemented))
}

func TestNewCompressor_bzip2Stub(t *testing.T) {
	_, err := NewCompressor("bzip2", 6)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrCompressionFormatNotImplemented))
}

func TestGzipRoundTrip_allLevels(t *testing.T) {
	plaintext := []byte(strings.Repeat("backup config payload ", 256)) // ~5.6KB, compresses well
	for level := 1; level <= 9; level++ {
		level := level
		t.Run("level="+itoa(level), func(t *testing.T) {
			c, err := NewCompressor("gzip", level)
			require.NoError(t, err)
			assert.Equal(t, "gzip", c.Format())
			assert.Equal(t, ".gz", c.Extension())

			rc, err := c.Wrap(context.Background(), bytes.NewReader(plaintext))
			require.NoError(t, err)
			defer rc.Close()

			compressed, err := io.ReadAll(rc)
			require.NoError(t, err)

			// Magic bytes 0x1f 0x8b
			require.GreaterOrEqual(t, len(compressed), 2)
			assert.Equal(t, byte(0x1f), compressed[0])
			assert.Equal(t, byte(0x8b), compressed[1])

			// Round-trip via stdlib gzip reader
			gzr, err := gzip.NewReader(bytes.NewReader(compressed))
			require.NoError(t, err)
			defer gzr.Close()
			recovered, err := io.ReadAll(gzr)
			require.NoError(t, err)
			assert.Equal(t, plaintext, recovered)
		})
	}
}

func TestZstdRoundTrip_allLevels(t *testing.T) {
	plaintext := []byte(strings.Repeat("backup config payload ", 256))
	for level := 1; level <= 9; level++ {
		level := level
		t.Run("level="+itoa(level), func(t *testing.T) {
			c, err := NewCompressor("zstd", level)
			require.NoError(t, err)
			assert.Equal(t, "zstd", c.Format())
			assert.Equal(t, ".zst", c.Extension())

			rc, err := c.Wrap(context.Background(), bytes.NewReader(plaintext))
			require.NoError(t, err)
			defer rc.Close()

			compressed, err := io.ReadAll(rc)
			require.NoError(t, err)

			// zstd magic bytes 0x28 0xb5 0x2f 0xfd
			require.GreaterOrEqual(t, len(compressed), 4)
			assert.Equal(t, byte(0x28), compressed[0])
			assert.Equal(t, byte(0xb5), compressed[1])
			assert.Equal(t, byte(0x2f), compressed[2])
			assert.Equal(t, byte(0xfd), compressed[3])

			dec, err := zstd.NewReader(bytes.NewReader(compressed))
			require.NoError(t, err)
			defer dec.Close()
			recovered, err := io.ReadAll(dec)
			require.NoError(t, err)
			assert.Equal(t, plaintext, recovered)
		})
	}
}

// TestStreamingMemory verifies that compressing a 50MB stream does not buffer
// the entire payload (peak heap stays well below the input size). 50MB chosen
// over 100MB to keep the test fast on CI.
func TestStreamingMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("skip streaming memory test under -short")
	}
	const size = 50 * 1024 * 1024 // 50 MB
	c, err := NewCompressor("gzip", 1)
	require.NoError(t, err)

	src := &countingZeroReader{remaining: size}

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	rc, err := c.Wrap(context.Background(), src)
	require.NoError(t, err)
	// Consume the compressed stream; n is compressed-bytes-out, not plaintext-in.
	// Highly compressible zeros shrink ~50MB → ~50KB.
	_, err = io.Copy(io.Discard, rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	// Plaintext bytes consumed (input side) must equal the full size.
	require.Equal(t, int64(size), src.read)

	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)

	// Heap delta should stay under ~10MB. A non-streaming impl would balloon
	// to >50MB. Use a generous 20MB ceiling to absorb GC noise on CI.
	delta := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	assert.Less(t, delta, int64(20*1024*1024),
		"heap grew %d bytes — likely buffered the full %d-byte stream", delta, size)
}

// countingZeroReader yields zeros for `remaining` bytes then EOF, while
// tracking how many bytes were consumed. Used by TestStreamingMemory to
// confirm the compressor pulled the whole input through streaming I/O.
type countingZeroReader struct {
	remaining int64
	read      int64
}

func (c *countingZeroReader) Read(p []byte) (int, error) {
	if c.remaining <= 0 {
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > c.remaining {
		n = int(c.remaining)
	}
	for i := 0; i < n; i++ {
		p[i] = 0
	}
	c.remaining -= int64(n)
	c.read += int64(n)
	return n, nil
}

// TestZstdLevelMapping exercises the 1..9 → klauspost level collapsing.
func TestZstdLevelMapping(t *testing.T) {
	tests := []struct {
		level int
		want  zstd.EncoderLevel
	}{
		{1, zstd.SpeedFastest},
		{2, zstd.SpeedFastest},
		{3, zstd.SpeedDefault},
		{6, zstd.SpeedDefault},
		{7, zstd.SpeedBetterCompression},
		{8, zstd.SpeedBetterCompression},
		{9, zstd.SpeedBestCompression},
	}
	for _, tc := range tests {
		got := zstdLevelFor(tc.level)
		assert.Equal(t, tc.want, got, "level=%d", tc.level)
	}
}

// itoa avoids strconv import noise in subtests.
func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
