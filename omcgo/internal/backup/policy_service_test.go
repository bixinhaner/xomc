package backup

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// mockPolicyRepo implements PolicyRepository for unit tests.
type mockPolicyRepo struct {
	current  *BackupPolicy   // nil = empty table
	upsertFn func(ctx context.Context, p *BackupPolicy) error
}

func (m *mockPolicyRepo) Get(_ context.Context) (*BackupPolicy, error) {
	if m.current == nil {
		return nil, commonerrors.ErrNotFound
	}
	cp := *m.current
	return &cp, nil
}

func (m *mockPolicyRepo) Upsert(ctx context.Context, p *BackupPolicy) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, p)
	}
	if m.current == nil {
		p.ID = uuid.New()
	} else {
		p.ID = m.current.ID
	}
	m.current = p
	return nil
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestPolicyService_Get_EmptyReturnsDefaults(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{}, zap.NewNop())
	got, err := svc.Get(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)

	want := DefaultPolicy()
	assert.Equal(t, want.RetentionDays, got.RetentionDays)
	assert.Equal(t, want.StorageBackend, got.StorageBackend)
	assert.Equal(t, want.AlertThresholdPercent, got.AlertThresholdPercent)
	assert.Equal(t, want.EnableEncryption, got.EnableEncryption)
}

func TestPolicyService_Get_ExistingRoundTrips(t *testing.T) {
	id := uuid.New()
	repo := &mockPolicyRepo{current: &BackupPolicy{
		ID:                    id,
		RetentionDays:         60,
		MaxBackupCount:        200,
		MinBackupCount:        5,
		AutoCleanup:           true,
		CleanupTime:           "02:30",
		CleanupDayOfWeek:      1,
		KeepLastN:             10,
		EnableCompression:     true,
		CompressionLevel:      9,
		CompressionFormat:     "zstd",
		StorageBackend:        "ftp",
		LocalPath:             "/var/backup/omc",
		MaxStorageGB:          1000,
		EnableEncryption:      true,
		EncryptionAlgorithm:   "AES-256-GCM",
		AlertOnFailure:        true,
		AlertEmail:            "ops@example.com",
		AlertThresholdPercent: 90,
	}}
	svc := NewPolicyService(repo, zap.NewNop())
	got, err := svc.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, 60, got.RetentionDays)
	assert.Equal(t, "zstd", got.CompressionFormat)
	assert.Equal(t, "ftp", got.StorageBackend)
	assert.True(t, got.EnableEncryption)
}

// ---------------------------------------------------------------------------
// Update — validation table-driven
// ---------------------------------------------------------------------------

func TestPolicyService_Update_ValidationMatrix(t *testing.T) {
	good := DefaultPolicy()

	tests := []struct {
		name       string
		mutate     func(p *BackupPolicy)
		expectErr  bool
		errSubstr  string
	}{
		{"defaults pass", func(_ *BackupPolicy) {}, false, ""},
		{"retention too low", func(p *BackupPolicy) { p.RetentionDays = 0 }, true, "retention_days"},
		{"retention too high", func(p *BackupPolicy) { p.RetentionDays = 4000 }, true, "retention_days"},
		{"max < min", func(p *BackupPolicy) { p.MinBackupCount = 200 }, true, "min_backup_count cannot exceed"},
		{"cleanup_time bad format", func(p *BackupPolicy) { p.CleanupTime = "25:99" }, true, "cleanup_time"},
		{"cleanup_time non-zero-pad", func(p *BackupPolicy) { p.CleanupTime = "9:00" }, true, "cleanup_time"},
		{"cleanup_day too low", func(p *BackupPolicy) { p.CleanupDayOfWeek = -2 }, true, "cleanup_day_of_week"},
		{"cleanup_day too high", func(p *BackupPolicy) { p.CleanupDayOfWeek = 7 }, true, "cleanup_day_of_week"},
		{"compression level out of range", func(p *BackupPolicy) { p.CompressionLevel = 0 }, true, "compression_level"},
		{"unknown compression format", func(p *BackupPolicy) { p.CompressionFormat = "xz" }, true, "compression_format"},
		{"unknown storage backend", func(p *BackupPolicy) { p.StorageBackend = "blob" }, true, "storage_backend"},
		{"unknown encryption algorithm (review HIGH-2)", func(p *BackupPolicy) { p.EncryptionAlgorithm = "ROT13" }, true, "encryption_algorithm"},
		{"local backend missing local_path (review HIGH-4)", func(p *BackupPolicy) { p.StorageBackend = "local"; p.LocalPath = "" }, true, "local_path"},
		{"ftp backend missing ftp_config_id (review HIGH-4)", func(p *BackupPolicy) { p.StorageBackend = "ftp"; p.FTPConfigID = nil }, true, "ftp_config_id"},
		{"sftp backend missing ftp_config_id (review HIGH-4)", func(p *BackupPolicy) { p.StorageBackend = "sftp"; p.FTPConfigID = nil }, true, "ftp_config_id"},
		{"alert threshold too low", func(p *BackupPolicy) { p.AlertThresholdPercent = 49 }, true, "alert_threshold"},
		{"alert threshold too high", func(p *BackupPolicy) { p.AlertThresholdPercent = 96 }, true, "alert_threshold"},
		{"bad email when alert on", func(p *BackupPolicy) { p.AlertOnFailure = true; p.AlertEmail = "not-an-email" }, true, "alert_email"},
		{"empty email ok when alert on", func(p *BackupPolicy) { p.AlertOnFailure = true; p.AlertEmail = "" }, false, ""},
		// T-0077: lz4 + bzip2 are now first-class implementations and accepted
		// regardless of EnableCompression value (the prior T-0074 reject guard
		// was removed in policy_service.go alongside this assertion change).
		{"lz4 + enable_compression=true now accepted (T-0077)", func(p *BackupPolicy) {
			p.EnableCompression = true
			p.CompressionFormat = "lz4"
		}, false, ""},
		{"bzip2 + enable_compression=true now accepted (T-0077)", func(p *BackupPolicy) {
			p.EnableCompression = true
			p.CompressionFormat = "bzip2"
		}, false, ""},
		// T-0075: encryption-algorithm stub rejection (mirror lz4/bzip2 T-0074 pattern).
		{"AES-256-CBC + enable_encryption=true rejected (T-0075)", func(p *BackupPolicy) {
			p.EnableEncryption = true
			p.EncryptionAlgorithm = "AES-256-CBC"
		}, true, "AES-256-CBC not yet implemented"},
		{"ChaCha20-Poly1305 + enable_encryption=true rejected (T-0075)", func(p *BackupPolicy) {
			p.EnableEncryption = true
			p.EncryptionAlgorithm = "ChaCha20-Poly1305"
		}, true, "ChaCha20-Poly1305 not yet implemented"},
		// AES-256-GCM + EnableEncryption=true ACCEPTED when no KeyProvider wired
		// (validation skips KEK check; UI Persisted-Tag is the only signal).
		{"AES-256-GCM + enable_encryption=true accepted with no key provider (T-0075)", func(p *BackupPolicy) {
			p.EnableEncryption = true
			p.EncryptionAlgorithm = "AES-256-GCM"
		}, false, ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := *good // shallow copy is fine; no slices
			tt.mutate(&p)
			svc := NewPolicyService(&mockPolicyRepo{}, zap.NewNop())
			_, err := svc.Update(context.Background(), &p)
			if tt.expectErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
					"expected ErrInvalidInput, got %v", err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPolicyService_Update_NilPolicy(t *testing.T) {
	svc := NewPolicyService(&mockPolicyRepo{}, zap.NewNop())
	_, err := svc.Update(context.Background(), nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestPolicyService_Update_PreservesIdOnExisting(t *testing.T) {
	existingID := uuid.New()
	repo := &mockPolicyRepo{current: &BackupPolicy{
		ID:                    existingID,
		RetentionDays:         30,
		MaxBackupCount:        100,
		MinBackupCount:        3,
		CleanupTime:           "03:00",
		KeepLastN:             5,
		CompressionLevel:      6,
		CompressionFormat:     "gzip",
		StorageBackend:        "local",
		MaxStorageGB:          500,
		EncryptionAlgorithm:   "AES-256-GCM",
		AlertThresholdPercent: 80,
	}}
	svc := NewPolicyService(repo, zap.NewNop())

	new := DefaultPolicy()
	new.RetentionDays = 90 // change one field
	got, err := svc.Update(context.Background(), new)
	require.NoError(t, err)
	assert.Equal(t, existingID, got.ID, "Update should preserve existing id (singleton)")
	assert.Equal(t, 90, got.RetentionDays)
}

func TestPolicyService_Update_UpsertError(t *testing.T) {
	repo := &mockPolicyRepo{
		upsertFn: func(_ context.Context, _ *BackupPolicy) error {
			return errors.New("simulated DB outage")
		},
	}
	svc := NewPolicyService(repo, zap.NewNop())
	_, err := svc.Update(context.Background(), DefaultPolicy())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated DB outage")
}

// TestValidatePolicy_AESGCM_KEKUnavailable verifies the T-0075 PUT-time
// guard: when KeyProvider.Available()=false, EnableEncryption=true with
// AES-256-GCM is rejected — operator must either set OMC_BACKUP_ENCRYPTION_KEY
// or disable encryption.
func TestValidatePolicy_AESGCM_KEKUnavailable(t *testing.T) {
	repo := &mockPolicyRepo{}
	svc := NewPolicyService(repo, zap.NewNop())
	svc.SetKeyProvider(newStaticKeyProvider(nil)) // unavailable

	p := *DefaultPolicy()
	p.EnableEncryption = true
	p.EncryptionAlgorithm = "AES-256-GCM"

	_, err := svc.Update(context.Background(), &p)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
	assert.Contains(t, err.Error(), "encryption key not configured")
	assert.Contains(t, err.Error(), EnvBackupEncryptionKey)
}

// TestValidatePolicy_AESGCM_KEKAvailable confirms the happy path: same
// policy with a wired-and-available KeyProvider passes validation.
func TestValidatePolicy_AESGCM_KEKAvailable(t *testing.T) {
	repo := &mockPolicyRepo{}
	svc := NewPolicyService(repo, zap.NewNop())
	key := make([]byte, kekSize)
	for i := range key {
		key[i] = 0x42
	}
	svc.SetKeyProvider(newStaticKeyProvider(key))

	p := *DefaultPolicy()
	p.EnableEncryption = true
	p.EncryptionAlgorithm = "AES-256-GCM"

	_, err := svc.Update(context.Background(), &p)
	require.NoError(t, err, "AES-256-GCM with available KEK must pass")
}
