// Package download — on-the-fly decompression for backup files (T-0072).
//
// Backup files are stored compressed (.gz/.zst/.lz4/.bz2) by T-0074 + T-0077,
// but most CPEs cannot decompress on their own when CWMP Download tells them
// "this is a Vendor Configuration File". The download handler therefore
// detects compressed extensions and wraps the MinIO object stream with the
// matching decompressor before writing the response body, mirroring T-0074's
// streaming compression on the upload side.
package download

import (
	stdbzip2 "compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Decompressor wraps a compressed reader and returns a plaintext reader.
// Format() label is used for Prometheus metrics; Cleaner trims the relevant
// extension off the file name so CPE / browsers see the original ".xml".
type Decompressor interface {
	Wrap(src io.Reader) (io.ReadCloser, error)
	Format() string
}

// detectCompression inspects objectPath's extension and returns the matching
// Decompressor + the file name with that extension stripped. ok=false signals
// "no compression detected; pass through unchanged".
func detectCompression(objectPath string) (Decompressor, string, bool) {
	switch {
	case strings.HasSuffix(objectPath, ".gz"):
		return gzipDecompressor{}, strings.TrimSuffix(objectPath, ".gz"), true
	case strings.HasSuffix(objectPath, ".zst"):
		return zstdDecompressor{}, strings.TrimSuffix(objectPath, ".zst"), true
	case strings.HasSuffix(objectPath, ".lz4"):
		return lz4Decompressor{}, strings.TrimSuffix(objectPath, ".lz4"), true
	case strings.HasSuffix(objectPath, ".bz2"):
		return bzip2Decompressor{}, strings.TrimSuffix(objectPath, ".bz2"), true
	default:
		return nil, objectPath, false
	}
}

type gzipDecompressor struct{}

func (gzipDecompressor) Format() string { return "gzip" }
func (gzipDecompressor) Wrap(src io.Reader) (io.ReadCloser, error) {
	r, err := gzip.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	return r, nil
}

type zstdDecompressor struct{}

func (zstdDecompressor) Format() string { return "zstd" }
func (zstdDecompressor) Wrap(src io.Reader) (io.ReadCloser, error) {
	r, err := zstd.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("zstd reader: %w", err)
	}
	return zstdReadCloser{r}, nil
}

// zstdReadCloser adapts klauspost zstd.Decoder.Close() (no return value) to
// io.ReadCloser (which requires Close() error).
type zstdReadCloser struct{ *zstd.Decoder }

func (z zstdReadCloser) Close() error { z.Decoder.Close(); return nil }

type lz4Decompressor struct{}

func (lz4Decompressor) Format() string { return "lz4" }
func (lz4Decompressor) Wrap(src io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(lz4.NewReader(src)), nil
}

type bzip2Decompressor struct{}

func (bzip2Decompressor) Format() string { return "bzip2" }
func (bzip2Decompressor) Wrap(src io.Reader) (io.ReadCloser, error) {
	// stdlib compress/bzip2 has only a Reader (no Writer); ideal here.
	return io.NopCloser(stdbzip2.NewReader(src)), nil
}
