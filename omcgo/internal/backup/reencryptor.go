// Package backup — KEK rotation re-encrypt service (T-0092).
//
// Reencryptor walks the backup MinIO bucket and migrates every encrypted
// file from its envelope-recorded kek_id to a target kek_id, by
// decrypt-then-encrypt under the same KeyProvider. The service drives the
// CLI binary cmd/backup-reencrypt; the same code is exercised by unit
// tests via the narrow consumer interfaces declared here (BucketLister,
// ObjectIO).
//
// Operational pattern:
//
//  1. Operator updates env to include both old (KEY_HISTORY) and new
//     (KEY + KEY_ID) KEKs and restarts the ACS deployment so live writes
//     land under the new KEK.
//  2. Operator runs `backup-reencrypt --target-kek-id=<new>` on the same
//     host. The tool walks the bucket, skips files already on the target,
//     decrypts the rest using KEKByID(envelope.kek_id), re-encrypts under
//     the active KEK (= target), and PUTs back.
//  3. After all files migrated, KEY_HISTORY can be removed and the old
//     KEK retired.
//
// Idempotency: re-running the tool naturally skips files already on the
// target — no state file or progress checkpoint is required.
//
// Cross-algorithm migration is out of scope (PRD §2.4): re-encrypt
// preserves the envelope's algo byte. Switching from GCM→CBC is a
// separate operation handled by future tooling.
package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
)

// ObjectIO is the narrow MinIO contract Reencryptor consumes for byte-level
// object reads + writes. The CLI adapts *minio.Client to this stream-oriented
// interface; consumer-side definition keeps tests and callers small.
type ObjectIO interface {
	GetObject(ctx context.Context, bucket, key string, opts minio.GetObjectOptions) (io.ReadCloser, error)
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

// reencryptResult is the outcome label written into the metrics counter.
// Treated as an internal enum-string to keep label cardinality bounded.
type reencryptResult string

const (
	resultReencrypted          reencryptResult = "reencrypted"
	resultSkipAlreadyTarget    reencryptResult = "skip_already_target"
	resultSkipNotEncrypted     reencryptResult = "skip_not_encrypted"
	resultSkipEnvelopeInvalid  reencryptResult = "skip_envelope_invalid"
	resultSkipSourceKEKMissing reencryptResult = "skip_source_kek_unavailable"
	resultSkipMinIOError       reencryptResult = "skip_minio_error"
	resultFailed               reencryptResult = "failed"
	resultReencryptedDryRun    reencryptResult = "reencrypted_dry_run"

	// Cap re-encrypt downloads to plaintext-cap + envelope overhead. Mirrors
	// encryption.go encMaxPlaintext (64 MiB) plus a generous header pad.
	reencryptMaxObjectSize = encMaxPlaintext + 4096
)

// ReencryptStats aggregates per-outcome counts across a Run invocation.
// Returned to the caller for log / exit-code use; metrics carry the same
// information broken down by result label.
type ReencryptStats struct {
	Reencrypted              int64
	ReencryptedDryRun        int64
	SkipAlreadyTarget        int64
	SkipNotEncrypted         int64
	SkipEnvelopeInvalid      int64
	SkipSourceKEKUnavailable int64
	SkipMinIOError           int64
	Failed                   int64
}

// Total returns the sum across all outcome buckets — used by progress logs.
func (s ReencryptStats) Total() int64 {
	return s.Reencrypted + s.ReencryptedDryRun + s.SkipAlreadyTarget +
		s.SkipNotEncrypted + s.SkipEnvelopeInvalid + s.SkipSourceKEKUnavailable +
		s.SkipMinIOError + s.Failed
}

// ReencryptorConfig wires Reencryptor dependencies. All fields are
// required except DryRun, ProgressEvery, Metrics and Logger (which have
// safe defaults).
type ReencryptorConfig struct {
	Bucket        string
	TargetKekID   string
	Concurrency   int  // clamped to [1, 32]
	DryRun        bool // when true, skip PutObject; report what would change
	ProgressEvery int  // log progress every N processed (default 100; 0 = no progress)
	Lister        BucketLister
	IO            ObjectIO
	KeyProvider   KeyProvider
	Metrics       *PolicyMetrics // nil-safe
	Logger        *zap.Logger    // nil-safe (zap.NewNop)
}

// Reencryptor holds the active configuration and per-run mutable state.
// One instance per Run invocation; not thread-safe across multiple Runs
// (the worker pool inside Run is itself thread-safe).
type Reencryptor struct {
	cfg ReencryptorConfig
}

// NewReencryptor validates ReencryptorConfig and returns a ready service.
// The clamping for Concurrency (1..32) happens here so callers can't
// accidentally launch a 1000-worker run that would saturate MinIO.
func NewReencryptor(cfg ReencryptorConfig) (*Reencryptor, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("reencryptor: bucket is required")
	}
	cfg.Bucket = appconfig.NormalizeConfigBackupBucket(cfg.Bucket)
	if cfg.TargetKekID == "" && cfg.KeyProvider != nil {
		// Operator may want to migrate INTO empty-id (legacy slot).
		// Allow but require explicit empty string — already the zero value.
	}
	if cfg.Lister == nil {
		return nil, errors.New("reencryptor: bucket lister is required")
	}
	if cfg.IO == nil {
		return nil, errors.New("reencryptor: object I/O is required")
	}
	if cfg.KeyProvider == nil {
		return nil, errors.New("reencryptor: key provider is required")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.Concurrency > 32 {
		cfg.Concurrency = 32
	}
	if cfg.ProgressEvery < 0 {
		cfg.ProgressEvery = 0
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Reencryptor{cfg: cfg}, nil
}

// Run walks the configured bucket and re-encrypts each non-target file.
// Errors from individual files are tallied into stats but never abort
// the run; the only paths that return a non-nil error are:
//   - ListObjects stream error (terminal — bucket reachable but stream
//     dies mid-walk)
//   - ctx cancellation (returns ctx.Err() so the caller knows it was
//     interrupted vs. completed naturally)
func (r *Reencryptor) Run(ctx context.Context) (ReencryptStats, error) {
	startedAt := time.Now()
	r.cfg.Logger.Info("reencrypt run started",
		zap.String("bucket", r.cfg.Bucket),
		zap.String("target_kek_id", r.cfg.TargetKekID),
		zap.Int("concurrency", r.cfg.Concurrency),
		zap.Bool("dry_run", r.cfg.DryRun),
	)

	// Buffered worker pool: keys arrive on `keys`; workers consume +
	// process. Stats counters use atomics so the scatter-gather across
	// workers is lock-free.
	keys := make(chan string, r.cfg.Concurrency)
	var stats ReencryptStats
	var processed int64

	var wg sync.WaitGroup
	for i := 0; i < r.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range keys {
				start := time.Now()
				outcome := r.reencryptOne(ctx, key)
				r.cfg.Metrics.ObserveReencryptDuration(time.Since(start).Seconds())
				r.cfg.Metrics.RecordReencryptOutcome(string(outcome))
				atomicAddOutcome(&stats, outcome)
				n := atomic.AddInt64(&processed, 1)
				if r.cfg.ProgressEvery > 0 && n%int64(r.cfg.ProgressEvery) == 0 {
					r.cfg.Logger.Info("reencrypt progress",
						zap.Int64("processed", n),
						zap.Duration("elapsed", time.Since(startedAt)),
					)
				}
			}
		}()
	}

	// Producer: stream keys from the bucket lister into the channel.
	// On ctx cancel or list-side error, close the channel so workers
	// drain cleanly.
	var listErr error
	for obj := range r.cfg.Lister.ListObjects(ctx, r.cfg.Bucket, minio.ListObjectsOptions{
		Recursive: true,
	}) {
		if obj.Err != nil {
			listErr = fmt.Errorf("list bucket: %w", obj.Err)
			break
		}
		select {
		case keys <- obj.Key:
		case <-ctx.Done():
			listErr = ctx.Err()
			break
		}
		if ctx.Err() != nil {
			listErr = ctx.Err()
			break
		}
	}
	close(keys)
	wg.Wait()

	r.cfg.Logger.Info("reencrypt run completed",
		zap.Int64("processed", processed),
		zap.Int64("reencrypted", stats.Reencrypted),
		zap.Int64("dry_run", stats.ReencryptedDryRun),
		zap.Int64("skip_already_target", stats.SkipAlreadyTarget),
		zap.Int64("skip_not_encrypted", stats.SkipNotEncrypted),
		zap.Int64("skip_envelope_invalid", stats.SkipEnvelopeInvalid),
		zap.Int64("skip_source_kek_unavailable", stats.SkipSourceKEKUnavailable),
		zap.Int64("skip_minio_error", stats.SkipMinIOError),
		zap.Int64("failed", stats.Failed),
		zap.Duration("elapsed", time.Since(startedAt)),
		zap.Error(listErr),
	)
	return stats, listErr
}

// atomicAddOutcome increments the stats counter for the given outcome
// using atomic ops so concurrent worker writes are safe without a mutex.
// We accept the small bookkeeping awkwardness of one switch-case in
// exchange for lock-free progress accounting.
func atomicAddOutcome(stats *ReencryptStats, outcome reencryptResult) {
	switch outcome {
	case resultReencrypted:
		atomic.AddInt64(&stats.Reencrypted, 1)
	case resultReencryptedDryRun:
		atomic.AddInt64(&stats.ReencryptedDryRun, 1)
	case resultSkipAlreadyTarget:
		atomic.AddInt64(&stats.SkipAlreadyTarget, 1)
	case resultSkipNotEncrypted:
		atomic.AddInt64(&stats.SkipNotEncrypted, 1)
	case resultSkipEnvelopeInvalid:
		atomic.AddInt64(&stats.SkipEnvelopeInvalid, 1)
	case resultSkipSourceKEKMissing:
		atomic.AddInt64(&stats.SkipSourceKEKUnavailable, 1)
	case resultSkipMinIOError:
		atomic.AddInt64(&stats.SkipMinIOError, 1)
	case resultFailed:
		atomic.AddInt64(&stats.Failed, 1)
	}
}

// reencryptOne handles one bucket object end-to-end. Errors are
// classified into reencryptResult outcomes and never propagated to the
// caller — Run aggregates outcomes; per-file errors don't abort the run.
func (r *Reencryptor) reencryptOne(ctx context.Context, key string) reencryptResult {
	if !strings.HasSuffix(key, ".enc") {
		return resultSkipNotEncrypted
	}

	obj, err := r.cfg.IO.GetObject(ctx, r.cfg.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		r.cfg.Logger.Warn("reencrypt: GetObject failed", zap.String("key", key), zap.Error(err))
		return resultSkipMinIOError
	}
	defer obj.Close()
	blob, err := io.ReadAll(io.LimitReader(obj, reencryptMaxObjectSize))
	if err != nil {
		r.cfg.Logger.Warn("reencrypt: read object body failed", zap.String("key", key), zap.Error(err))
		return resultSkipMinIOError
	}

	hdr, err := parseEnvelopeHeader(blob)
	if err != nil {
		r.cfg.Logger.Warn("reencrypt: envelope parse failed",
			zap.String("key", key), zap.Error(err))
		return resultSkipEnvelopeInvalid
	}
	if hdr.kekID == r.cfg.TargetKekID {
		return resultSkipAlreadyTarget
	}

	algoName, ok := algoStringFromByte(hdr.algo)
	if !ok {
		r.cfg.Logger.Warn("reencrypt: unsupported algo byte",
			zap.String("key", key), zap.Uint8("algo", hdr.algo))
		return resultSkipEnvelopeInvalid
	}
	enc, err := NewEncryptor(algoName, r.cfg.KeyProvider)
	if err != nil {
		r.cfg.Logger.Warn("reencrypt: NewEncryptor failed",
			zap.String("key", key), zap.String("algo", algoName), zap.Error(err))
		return resultSkipSourceKEKMissing
	}

	// AAD = basename minus ".enc" suffix — mirrors ACS upload/download
	// AAD computation (T-0085 review HIGH-1 fix).
	aad := []byte(strings.TrimSuffix(filepath.Base(key), ".enc"))

	plaintext, err := enc.Decrypt(blob, aad)
	if err != nil {
		// Could be source KEK missing OR genuine tamper. Both are
		// "do not touch the file" — let the operator investigate.
		r.cfg.Logger.Warn("reencrypt: decrypt failed",
			zap.String("key", key),
			zap.String("source_kek_id", hdr.kekID),
			zap.Error(err))
		if errors.Is(err, ErrEncryptionKeyUnavailable) {
			return resultSkipSourceKEKMissing
		}
		return resultSkipEnvelopeInvalid
	}

	newBlob, err := enc.Encrypt(plaintext, aad)
	if err != nil {
		r.cfg.Logger.Warn("reencrypt: re-encrypt failed",
			zap.String("key", key), zap.Error(err))
		return resultFailed
	}

	if r.cfg.DryRun {
		r.cfg.Logger.Info("reencrypt: dry-run would replace",
			zap.String("key", key),
			zap.String("source_kek_id", hdr.kekID),
			zap.String("target_kek_id", r.cfg.TargetKekID))
		return resultReencryptedDryRun
	}

	_, err = r.cfg.IO.PutObject(ctx, r.cfg.Bucket, key,
		bytes.NewReader(newBlob), int64(len(newBlob)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		r.cfg.Logger.Warn("reencrypt: PutObject failed",
			zap.String("key", key), zap.Error(err))
		return resultFailed
	}
	return resultReencrypted
}

// algoStringFromByte translates an envelope algo byte to the canonical
// NewEncryptor algorithm name. Returns ok=false for unrecognised bytes
// so the caller can record skip_envelope_invalid.
func algoStringFromByte(b byte) (string, bool) {
	switch b {
	case encAlgoGCM:
		return "AES-256-GCM", true
	case encAlgoCBC:
		return "AES-256-CBC", true
	case encAlgoChaCha20:
		return "ChaCha20-Poly1305", true
	default:
		return "", false
	}
}
