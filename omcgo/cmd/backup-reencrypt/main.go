// Command backup-reencrypt walks the OMC backup MinIO bucket and migrates
// every encrypted file from its envelope-recorded kek_id to a target
// kek_id. Designed for one-shot operator use after a KEK rotation has
// been provisioned via OMC_BACKUP_ENCRYPTION_KEY{,_ID,_HISTORY} env vars.
//
// Usage:
//
//	OMC_BACKUP_ENCRYPTION_KEY=<new-hex> \
//	OMC_BACKUP_ENCRYPTION_KEY_ID=v2 \
//	OMC_BACKUP_ENCRYPTION_KEY_HISTORY="v1=<old-hex>" \
//	OMC_MINIO_ENDPOINT=minio:9000 \
//	OMC_MINIO_ACCESS_KEY=... OMC_MINIO_SECRET_KEY=... \
//	OMC_MINIO_BUCKET=config_backup \
//	backup-reencrypt --target-kek-id=v2 --concurrency=4
//
// Operates idempotently: re-running skips files already on the target
// kek_id, so interrupted runs can be resumed by re-invoking.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/backup"
)

const (
	envMinIOEndpoint  = "OMC_MINIO_ENDPOINT"
	envMinIOAccessKey = "OMC_MINIO_ACCESS_KEY"
	envMinIOSecretKey = "OMC_MINIO_SECRET_KEY"
	envMinIOUseSSL    = "OMC_MINIO_USE_SSL"
	envMinIOBucket    = "OMC_MINIO_BUCKET"
)

func main() {
	var (
		targetKekID   string
		concurrency   int
		dryRun        bool
		progressEvery int
		bucketFlag    string
	)

	rootCmd := &cobra.Command{
		Use:   "backup-reencrypt",
		Short: "Re-encrypt all OMC backup files in a MinIO bucket to a target KEK",
		Long: `Walks the OMC backup bucket and migrates every encrypted file from its
envelope-recorded kek_id to the target kek_id, by decrypt-then-encrypt
under the same KeyProvider. Idempotent — already-on-target files are
skipped without download.

Required env vars:
  OMC_BACKUP_ENCRYPTION_KEY          active KEK (64 hex chars)
  OMC_BACKUP_ENCRYPTION_KEY_ID       active KEK id (must equal --target-kek-id)
  OMC_BACKUP_ENCRYPTION_KEY_HISTORY  retired KEKs as "id=hex;id=hex"
  OMC_MINIO_ENDPOINT                 e.g. "minio:9000"
  OMC_MINIO_ACCESS_KEY / SECRET_KEY  MinIO credentials
  OMC_MINIO_USE_SSL                  "true" or "false"
  OMC_MINIO_BUCKET                   defaults to "config_backup"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), targetKekID, concurrency, dryRun, progressEvery, bucketFlag)
		},
	}
	rootCmd.Flags().StringVar(&targetKekID, "target-kek-id", "", "Target KEK id (required; must match OMC_BACKUP_ENCRYPTION_KEY_ID)")
	rootCmd.Flags().IntVar(&concurrency, "concurrency", 4, "Worker count (clamped 1..32)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Walk bucket but skip PutObject")
	rootCmd.Flags().IntVar(&progressEvery, "progress-every", 100, "Log progress every N processed (0 = no progress)")
	rootCmd.Flags().StringVar(&bucketFlag, "bucket", "", "Override OMC_MINIO_BUCKET")

	// SIGTERM / SIGINT handling at the cobra context level.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, targetKekID string, concurrency int, dryRun bool, progressEvery int, bucketFlag string) error {
	if targetKekID == "" {
		return fmt.Errorf("--target-kek-id is required")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	// KeyProvider from env vars (T-0087 multi-version mode).
	kp, err := backup.NewEnvKeyProvider()
	if err != nil {
		return fmt.Errorf("init key provider: %w", err)
	}
	if !kp.Available() {
		return fmt.Errorf("KEK not configured; set %s + %s", "OMC_BACKUP_ENCRYPTION_KEY", "OMC_BACKUP_ENCRYPTION_KEY_ID")
	}
	// Sanity: target KEK must be reachable. Fail fast before bucket walk.
	if _, err := kp.KEKByID(ctx, targetKekID); err != nil {
		return fmt.Errorf("target kek_id=%q not configured (check OMC_BACKUP_ENCRYPTION_KEY_ID + KEY_HISTORY): %w",
			targetKekID, err)
	}

	mc, err := minioFromEnv()
	if err != nil {
		return fmt.Errorf("init minio: %w", err)
	}

	bucket := bucketFlag
	if bucket == "" {
		bucket = os.Getenv(envMinIOBucket)
	}
	if bucket == "" {
		bucket = backup.CanonicalRestoreBucket
	}

	r, err := backup.NewReencryptor(backup.ReencryptorConfig{
		Bucket:        bucket,
		TargetKekID:   targetKekID,
		Concurrency:   concurrency,
		DryRun:        dryRun,
		ProgressEvery: progressEvery,
		Lister:        mc,
		IO:            mc,
		KeyProvider:   kp,
		Logger:        logger,
	})
	if err != nil {
		return fmt.Errorf("init reencryptor: %w", err)
	}

	stats, err := r.Run(ctx)
	logger.Info("backup-reencrypt summary",
		zap.String("target_kek_id", targetKekID),
		zap.Bool("dry_run", dryRun),
		zap.Int64("reencrypted", stats.Reencrypted),
		zap.Int64("dry_run_count", stats.ReencryptedDryRun),
		zap.Int64("skip_already_target", stats.SkipAlreadyTarget),
		zap.Int64("skip_not_encrypted", stats.SkipNotEncrypted),
		zap.Int64("skip_envelope_invalid", stats.SkipEnvelopeInvalid),
		zap.Int64("skip_source_kek_unavailable", stats.SkipSourceKEKUnavailable),
		zap.Int64("skip_minio_error", stats.SkipMinIOError),
		zap.Int64("failed", stats.Failed),
	)
	if err != nil {
		return err
	}
	if stats.Failed > 0 {
		return fmt.Errorf("%d files failed to re-encrypt; see logs for details", stats.Failed)
	}
	return nil
}

// minioFromEnv constructs a MinIO client purely from env vars. Mirrors
// internal/core/components/minio.NewMinIOClient but stays standalone so
// this binary doesn't pull in viper / appconfig.
func minioFromEnv() (*miniogo.Client, error) {
	endpoint := os.Getenv(envMinIOEndpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("%s not set", envMinIOEndpoint)
	}
	access := os.Getenv(envMinIOAccessKey)
	secret := os.Getenv(envMinIOSecretKey)
	useSSL := os.Getenv(envMinIOUseSSL) == "true"
	return miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: useSSL,
	})
}
