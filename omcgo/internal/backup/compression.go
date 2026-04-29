// Package backup — backup file compression (T-0074 / R-102 followup).
//
// Compressor wraps an io.Reader of plaintext bytes and returns an io.ReadCloser
// of compressed bytes. The actual compression happens streaming via io.Pipe so
// large backup files (100 MB+) do not buffer in memory.
//
// Integration point: acs/upload/handler.go ServeHTTP wraps the inbound CPE
// upload r.Body with a Compressor when policy.EnableCompression=true and the
// file type is FileTypeConfig (CWMP "3" Vendor Configuration File).
//
// Algorithm coverage:
//   - gzip:  full, stdlib compress/gzip                            (T-0074)
//   - zstd:  full, github.com/klauspost/compress/zstd              (T-0074)
//   - lz4:   full, github.com/pierrec/lz4/v4 (frame format)        (T-0077)
//   - bzip2: full, github.com/dsnet/compress/bzip2                 (T-0077)
package backup

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Sentinel errors callers compare with errors.Is.
var (
	ErrCompressionFormatInvalid        = errors.New("compression format must be one of gzip|bzip2|lz4|zstd")
	ErrCompressionLevelOutOfRange      = errors.New("compression level out of range [1,9]")
	ErrCompressionFormatNotImplemented = errors.New("compression format not implemented in current build")
)

// Compressor wraps a plaintext reader and returns a streaming compressed reader.
type Compressor interface {
	// Wrap reads from src and returns a ReadCloser of the compressed stream.
	// The provided ctx aborts the internal pump goroutine when cancelled, so
	// the caller can bound goroutine lifetime by the request scope (e.g. when
	// the HTTP handler returns early because PutObject failed).
	// Closing the returned ReadCloser flushes the compressor and finalizes the
	// stream. The implementation closes src if it implements io.Closer.
	Wrap(ctx context.Context, src io.Reader) (io.ReadCloser, error)
	// Format returns the canonical algorithm name (gzip|zstd|lz4|bzip2).
	Format() string
	// Extension returns the file-name suffix to append after compression
	// (".gz" | ".zst" | ".lz4" | ".bz2"), including the leading dot.
	Extension() string
}

// NewCompressor returns the Compressor matching format/level. format must be
// one of gzip|bzip2|lz4|zstd; level must be in [1,9].
//
// ErrCompressionFormatNotImplemented is retained as a sentinel for any future
// algorithm whose dep is gated behind a build tag — currently no algorithm
// returns it.
func NewCompressor(format string, level int) (Compressor, error) {
	if level < 1 || level > 9 {
		return nil, fmt.Errorf("level=%d: %w", level, ErrCompressionLevelOutOfRange)
	}
	switch format {
	case "gzip":
		return &gzipCompressor{level: level}, nil
	case "zstd":
		return &zstdCompressor{level: zstdLevelFor(level)}, nil
	case "lz4":
		return &lz4Compressor{level: lz4LevelFor(level)}, nil
	case "bzip2":
		return &bzip2Compressor{level: level}, nil
	default:
		return nil, fmt.Errorf("format=%q: %w", format, ErrCompressionFormatInvalid)
	}
}

// gzipCompressor — stdlib compress/gzip.
type gzipCompressor struct {
	level int
}

func (c *gzipCompressor) Format() string    { return "gzip" }
func (c *gzipCompressor) Extension() string { return ".gz" }

func (c *gzipCompressor) Wrap(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	gzw, err := gzip.NewWriterLevel(pw, c.level)
	if err != nil {
		_ = pw.Close()
		return nil, fmt.Errorf("gzip writer level=%d: %w", c.level, err)
	}
	go pumpAndClose(ctx, src, gzw, pw)
	return pr, nil
}

// zstdCompressor — github.com/klauspost/compress/zstd.
//
// Each Wrap creates a fresh encoder. Encoder pooling is a future optimization
// (PRD §9.10 待定点) but adds complexity for marginal MVP gain.
type zstdCompressor struct {
	level zstd.EncoderLevel
}

func (c *zstdCompressor) Format() string    { return "zstd" }
func (c *zstdCompressor) Extension() string { return ".zst" }

func (c *zstdCompressor) Wrap(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	enc, err := zstd.NewWriter(pw, zstd.WithEncoderLevel(c.level))
	if err != nil {
		_ = pw.Close()
		return nil, fmt.Errorf("zstd writer level=%d: %w", c.level, err)
	}
	go pumpAndClose(ctx, src, enc, pw)
	return pr, nil
}

// pumpAndClose copies src → compressor → pipe writer in a goroutine, then
// closes both the compressor and the pipe so the reader sees EOF / first error.
//
// ctx cancellation aborts the copy: when the upload handler returns early
// (e.g. PutObject failed), the request ctx is cancelled and the pump goroutine
// exits without waiting for src to drain. ctxReader interrupts blocking reads
// on src by returning the ctx error mid-Read.
//
// CloseWithError(nil) is equivalent to Close, so passing copyErr unconditionally
// is safe and propagates any partial-write failures to the consumer.
func pumpAndClose(ctx context.Context, src io.Reader, compressor io.WriteCloser, pw *io.PipeWriter) {
	_, copyErr := io.Copy(compressor, &ctxReader{ctx: ctx, r: src})
	closeErr := compressor.Close()
	finalErr := copyErr
	if finalErr == nil {
		finalErr = closeErr
	}
	_ = pw.CloseWithError(finalErr)
	if rc, ok := src.(io.Closer); ok {
		_ = rc.Close()
	}
}

// ctxReader wraps src so each Read first checks ctx cancellation. This is
// best-effort interruption; an in-flight Read on a blocking socket still
// has to return on its own (Go's net/http server closes r.Body on handler
// exit, which unblocks reads — so the worst-case lifetime is bounded by
// the request, not the goroutine).
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *ctxReader) Read(p []byte) (int, error) {
	if err := cr.ctx.Err(); err != nil {
		return 0, err
	}
	return cr.r.Read(p)
}

// zstdLevelFor maps OMC's 1..9 scale onto klauspost zstd's encoder levels.
//
// klauspost/compress/zstd exposes 4 named levels (Fastest, Default, Better,
// Best) — we collapse the 1..9 input across them to keep policy.compression_level
// semantics stable across formats.
func zstdLevelFor(level int) zstd.EncoderLevel {
	switch {
	case level <= 2:
		return zstd.SpeedFastest
	case level <= 6:
		return zstd.SpeedDefault
	case level <= 8:
		return zstd.SpeedBetterCompression
	default:
		return zstd.SpeedBestCompression
	}
}

// lz4Compressor — github.com/pierrec/lz4/v4. Default frame format (compatible
// with the `lz4` CLI tool), not block format.
type lz4Compressor struct {
	level lz4.CompressionLevel
}

func (c *lz4Compressor) Format() string    { return "lz4" }
func (c *lz4Compressor) Extension() string { return ".lz4" }

func (c *lz4Compressor) Wrap(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	lzw := lz4.NewWriter(pw)
	if err := lzw.Apply(lz4.CompressionLevelOption(c.level)); err != nil {
		_ = pw.Close()
		return nil, fmt.Errorf("lz4 writer level=%v: %w", c.level, err)
	}
	go pumpAndClose(ctx, src, lzw, pw)
	return pr, nil
}

// lz4LevelFor maps OMC's 1..9 scale to pierrec/lz4 named levels.
// pierrec exposes Fast (level 0) plus Level1..Level9; OMC's 1..9 maps directly.
func lz4LevelFor(level int) lz4.CompressionLevel {
	switch level {
	case 1:
		return lz4.Level1
	case 2:
		return lz4.Level2
	case 3:
		return lz4.Level3
	case 4:
		return lz4.Level4
	case 5:
		return lz4.Level5
	case 6:
		return lz4.Level6
	case 7:
		return lz4.Level7
	case 8:
		return lz4.Level8
	case 9:
		return lz4.Level9
	default:
		// Defensive — NewCompressor already rejects out-of-range; fallback to
		// Level3 (pierrec's documented sweet spot).
		return lz4.Level3
	}
}

// bzip2Compressor — github.com/dsnet/compress/bzip2.
//
// bzip2 is slow vs zstd/lz4 but ships in this build for parity with the
// schema CHECK constraint and for legacy integrations that ship `bunzip2` on
// the consumer side. Restore (T-0072) decompresses via stdlib compress/bzip2.
type bzip2Compressor struct {
	level int // 1..9 — accepted natively
}

func (c *bzip2Compressor) Format() string    { return "bzip2" }
func (c *bzip2Compressor) Extension() string { return ".bz2" }

func (c *bzip2Compressor) Wrap(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	bzw, err := bzip2.NewWriter(pw, &bzip2.WriterConfig{Level: c.level})
	if err != nil {
		_ = pw.Close()
		return nil, fmt.Errorf("bzip2 writer level=%d: %w", c.level, err)
	}
	go pumpAndClose(ctx, src, bzw, pw)
	return pr, nil
}
