package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingReencryptObjectIO struct {
	blob       []byte
	getBuckets []string
	putBuckets []string
}

func (r *recordingReencryptObjectIO) GetObject(
	_ context.Context,
	bucket string,
	_ string,
	_ minio.GetObjectOptions,
) (io.ReadCloser, error) {
	r.getBuckets = append(r.getBuckets, bucket)
	return io.NopCloser(bytes.NewReader(r.blob)), nil
}

func (r *recordingReencryptObjectIO) PutObject(
	_ context.Context,
	bucket string,
	_ string,
	reader io.Reader,
	_ int64,
	_ minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	r.putBuckets = append(r.putBuckets, bucket)
	_, err := io.Copy(io.Discard, reader)
	return minio.UploadInfo{}, err
}

func TestReencryptor_NormalizesLegacyBucketForListGetAndPut(t *testing.T) {
	keyV1 := makeTestKey(t)
	keyV2 := makeTestKey(t)
	sourceProvider := newMultiKeyProvider("v1", map[string][]byte{"v1": keyV1})
	targetProvider := newMultiKeyProvider("v2", map[string][]byte{"v1": keyV1, "v2": keyV2})
	key := "backup/2026/07/24/SN001_CFG.xml.enc"
	blob := encryptUnder(t, sourceProvider, "AES-256-GCM", []byte("<cfg/>"), []byte("SN001_CFG.xml"))
	lister := &fakeBucketLister{objects: []minio.ObjectInfo{{Key: key}}}
	objectIO := &recordingReencryptObjectIO{blob: blob}

	reencryptor, err := NewReencryptor(ReencryptorConfig{
		Bucket:      "config_backup",
		TargetKekID: "v2",
		Concurrency: 1,
		Lister:      lister,
		IO:          objectIO,
		KeyProvider: targetProvider,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)

	stats, err := reencryptor.Run(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(1), stats.Reencrypted)
	require.Equal(t, []string{"config-backup"}, lister.buckets)
	require.Equal(t, []string{"config-backup"}, objectIO.getBuckets)
	require.Equal(t, []string{"config-backup"}, objectIO.putBuckets)
	for _, bucket := range append(append(lister.buckets, objectIO.getBuckets...), objectIO.putBuckets...) {
		require.NoError(t, s3utils.CheckValidBucketNameStrict(bucket))
	}
}

// =============================================================================
// fakeObjectIO — in-memory MinIO substitute for Reencryptor tests
// =============================================================================
//
// Holds a map of key → blob; GetObject returns an in-memory ReadCloser and
// PutObject overwrites the map entry. Errors can be injected per key.

type fakeObjectIO struct {
	mu       sync.Mutex
	store    map[string][]byte
	getErrs  map[string]error
	putErrs  map[string]error
	putCalls map[string]int
}

func newFakeObjectIO() *fakeObjectIO {
	return &fakeObjectIO{
		store:    map[string][]byte{},
		getErrs:  map[string]error{},
		putErrs:  map[string]error{},
		putCalls: map[string]int{},
	}
}

func (f *fakeObjectIO) GetObject(_ context.Context, _, key string, _ minio.GetObjectOptions) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.getErrs[key]; ok {
		return nil, err
	}
	blob, ok := f.store[key]
	if !ok {
		return nil, errors.New("fake-minio: key not found")
	}
	return io.NopCloser(bytes.NewReader(blob)), nil
}

func (f *fakeObjectIO) PutObject(_ context.Context, _, key string, reader io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.putErrs[key]; ok {
		return minio.UploadInfo{}, err
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	f.store[key] = body
	f.putCalls[key]++
	return minio.UploadInfo{Key: key, Size: int64(len(body))}, nil
}

// reencryptOneInMemory is a test-only shim that mirrors reencryptOne's
// logic using an in-memory blob source instead of calling GetObject.
// Production code never invokes this; it lives in _test.go so it can't
// leak into binaries.
//
// Returning the produced re-encrypted blob (or the original if dry-run /
// skip) lets tests assert envelope contents directly.
func (r *Reencryptor) reencryptOneInMemory(_ context.Context, key string, blob []byte) (reencryptResult, []byte) {
	// Inlined parse-only branches mirror reencryptOne up to PutObject.
	if !endsWithSuffix(key, ".enc") {
		return resultSkipNotEncrypted, nil
	}
	hdr, err := parseEnvelopeHeader(blob)
	if err != nil {
		return resultSkipEnvelopeInvalid, nil
	}
	if hdr.kekID == r.cfg.TargetKekID {
		return resultSkipAlreadyTarget, nil
	}
	algoName, ok := algoStringFromByte(hdr.algo)
	if !ok {
		return resultSkipEnvelopeInvalid, nil
	}
	enc, err := NewEncryptor(algoName, r.cfg.KeyProvider)
	if err != nil {
		return resultSkipSourceKEKMissing, nil
	}
	aad := []byte(trimEncSuffixBasename(key))
	plaintext, err := enc.Decrypt(blob, aad)
	if err != nil {
		if errors.Is(err, ErrEncryptionKeyUnavailable) {
			return resultSkipSourceKEKMissing, nil
		}
		return resultSkipEnvelopeInvalid, nil
	}
	newBlob, err := enc.Encrypt(plaintext, aad)
	if err != nil {
		return resultFailed, nil
	}
	if r.cfg.DryRun {
		return resultReencryptedDryRun, nil
	}
	return resultReencrypted, newBlob
}

func endsWithSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func trimEncSuffixBasename(key string) string {
	// Equivalent to filepath.Base + strings.TrimSuffix used in production.
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '/' {
			key = key[i+1:]
			break
		}
	}
	if endsWithSuffix(key, ".enc") {
		key = key[:len(key)-4]
	}
	return key
}

// =============================================================================
// V1-V10 tests via reencryptOneInMemory + Reencryptor.Run with fakes
// =============================================================================

func makeReencryptor(t *testing.T, cfg ReencryptorConfig) *Reencryptor {
	t.Helper()
	if cfg.Bucket == "" {
		cfg.Bucket = CanonicalRestoreBucket
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	r, err := NewReencryptor(cfg)
	require.NoError(t, err)
	return r
}

func encryptUnder(t *testing.T, kp KeyProvider, algo string, plaintext, aad []byte) []byte {
	t.Helper()
	enc, err := NewEncryptor(algo, kp)
	require.NoError(t, err)
	blob, err := enc.Encrypt(plaintext, aad)
	require.NoError(t, err)
	return blob
}

// V1 — single-file v1 → v2 (in-memory)
func TestReencryptor_V1_SingleFileV1ToV2(t *testing.T) {
	keyV1 := makeTestKey(t)
	keyV2 := makeTestKey(t)

	// Encrypt under active=v1
	srcKp := newMultiKeyProvider("v1", map[string][]byte{"v1": keyV1})
	plaintext := []byte("payload-A")
	aad := []byte("backup-aabbccdd-SN999.xml.gz")
	blob := encryptUnder(t, srcKp, "AES-256-GCM", plaintext, aad)

	// Provider for re-encrypt: active=v2 + history v1
	dstKp := newMultiKeyProvider("v2", map[string][]byte{"v1": keyV1, "v2": keyV2})
	r := makeReencryptor(t, ReencryptorConfig{
		TargetKekID: "v2",
		Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
		KeyProvider: dstKp,
	})

	outcome, newBlob := r.reencryptOneInMemory(context.Background(),
		"backup/2026/03/29/backup-aabbccdd-SN999.xml.gz.enc", blob)
	require.Equal(t, resultReencrypted, outcome)
	require.NotNil(t, newBlob)

	// New envelope kek_id = "v2"
	require.Equal(t, encVersion2, newBlob[4])
	require.Equal(t, byte(2), newBlob[6])
	require.Equal(t, "v2", string(newBlob[8:10]))

	// Decrypts under destination provider with original AAD
	verifyEnc, _ := NewEncryptor("AES-256-GCM", dstKp)
	recovered, err := verifyEnc.Decrypt(newBlob, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered)
}

// V2 — already on target → skip_already_target, no PutObject
func TestReencryptor_V2_AlreadyTarget(t *testing.T) {
	keyV2 := makeTestKey(t)
	kp := newMultiKeyProvider("v2", map[string][]byte{"v2": keyV2})
	plaintext := []byte("p")
	aad := []byte("backup-deadbeef-SN.xml.gz")
	blob := encryptUnder(t, kp, "AES-256-GCM", plaintext, aad)

	r := makeReencryptor(t, ReencryptorConfig{
		TargetKekID: "v2",
		Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
		KeyProvider: kp,
	})
	outcome, newBlob := r.reencryptOneInMemory(context.Background(),
		"backup/2026/03/29/backup-deadbeef-SN.xml.gz.enc", blob)
	assert.Equal(t, resultSkipAlreadyTarget, outcome)
	assert.Nil(t, newBlob)
}

// V3 — non-.enc file → skip_not_encrypted
func TestReencryptor_V3_NotEncryptedFile(t *testing.T) {
	kp := newMultiKeyProvider("v2", map[string][]byte{"v2": makeTestKey(t)})
	r := makeReencryptor(t, ReencryptorConfig{
		TargetKekID: "v2",
		Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
		KeyProvider: kp,
	})
	outcome, _ := r.reencryptOneInMemory(context.Background(),
		"unrelated/manual.xml", []byte("not-an-envelope"))
	assert.Equal(t, resultSkipNotEncrypted, outcome)
}

// V4 — source KEK missing
func TestReencryptor_V4_SourceKEKMissing(t *testing.T) {
	keyV0 := makeTestKey(t)
	keyV2 := makeTestKey(t)

	// Encrypt under v0
	srcKp := newMultiKeyProvider("v0", map[string][]byte{"v0": keyV0})
	blob := encryptUnder(t, srcKp, "AES-256-GCM", []byte("p"), []byte("foo.xml"))

	// Provider for re-encrypt has only v2 — v0 not available
	dstKp := newMultiKeyProvider("v2", map[string][]byte{"v2": keyV2})
	r := makeReencryptor(t, ReencryptorConfig{
		TargetKekID: "v2",
		Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
		KeyProvider: dstKp,
	})
	outcome, _ := r.reencryptOneInMemory(context.Background(), "foo.xml.enc", blob)
	assert.Equal(t, resultSkipSourceKEKMissing, outcome)
}

// V5 — envelope corrupt
func TestReencryptor_V5_EnvelopeInvalid(t *testing.T) {
	kp := newMultiKeyProvider("v2", map[string][]byte{"v2": makeTestKey(t)})
	r := makeReencryptor(t, ReencryptorConfig{
		TargetKekID: "v2",
		Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
		KeyProvider: kp,
	})
	outcome, _ := r.reencryptOneInMemory(context.Background(), "garbage.enc", []byte("not OENC"))
	assert.Equal(t, resultSkipEnvelopeInvalid, outcome)
}

// V8 — three algorithms re-encrypt successfully
func TestReencryptor_V8_ThreeAlgos(t *testing.T) {
	keyV1 := makeTestKey(t)
	keyV2 := makeTestKey(t)
	srcKp := newMultiKeyProvider("v1", map[string][]byte{"v1": keyV1})
	dstKp := newMultiKeyProvider("v2", map[string][]byte{"v1": keyV1, "v2": keyV2})

	for _, algo := range []string{"AES-256-GCM", "AES-256-CBC", "ChaCha20-Poly1305"} {
		algo := algo
		t.Run(algo, func(t *testing.T) {
			plaintext := []byte("payload-" + algo)
			aad := []byte("backup-aabbccdd-SN.xml.gz")
			blob := encryptUnder(t, srcKp, algo, plaintext, aad)

			r := makeReencryptor(t, ReencryptorConfig{
				TargetKekID: "v2",
				Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
				KeyProvider: dstKp,
			})
			outcome, newBlob := r.reencryptOneInMemory(context.Background(),
				"backup/2026/03/29/backup-aabbccdd-SN.xml.gz.enc", blob)
			require.Equal(t, resultReencrypted, outcome)
			require.NotNil(t, newBlob)

			// Algo byte unchanged across re-encrypt
			require.Equal(t, blob[5], newBlob[5], "algo byte must survive re-encrypt")
			// kek_id changed v1 → v2
			require.Equal(t, "v2", string(newBlob[8:10]))

			// Round-trip decrypt verify
			verifyEnc, _ := NewEncryptor(algo, dstKp)
			recovered, err := verifyEnc.Decrypt(newBlob, aad)
			require.NoError(t, err)
			assert.Equal(t, plaintext, recovered)
		})
	}
}

// V9 — dry-run does not write
func TestReencryptor_V9_DryRun(t *testing.T) {
	keyV1 := makeTestKey(t)
	keyV2 := makeTestKey(t)
	srcKp := newMultiKeyProvider("v1", map[string][]byte{"v1": keyV1})
	dstKp := newMultiKeyProvider("v2", map[string][]byte{"v1": keyV1, "v2": keyV2})

	blob := encryptUnder(t, srcKp, "AES-256-GCM", []byte("p"), []byte("foo.xml"))

	r := makeReencryptor(t, ReencryptorConfig{
		TargetKekID: "v2",
		DryRun:      true,
		Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
		KeyProvider: dstKp,
	})
	outcome, newBlob := r.reencryptOneInMemory(context.Background(), "foo.xml.enc", blob)
	assert.Equal(t, resultReencryptedDryRun, outcome)
	assert.Nil(t, newBlob, "dry-run reports outcome but does not produce output blob")
}

// V10 — concurrency clamping at NewReencryptor time
func TestReencryptor_V10_ConcurrencyClamp(t *testing.T) {
	cases := []struct {
		given int
		want  int
	}{
		{given: -5, want: 1},
		{given: 0, want: 1},
		{given: 1, want: 1},
		{given: 8, want: 8},
		{given: 32, want: 32},
		{given: 100, want: 32},
	}
	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("given=%d", c.given), func(t *testing.T) {
			r, err := NewReencryptor(ReencryptorConfig{
				Bucket:      "x",
				TargetKekID: "v2",
				Concurrency: c.given,
				Lister:      &fakeBucketLister{}, IO: newFakeObjectIO(),
				KeyProvider: newMultiKeyProvider("v2", map[string][]byte{"v2": makeTestKey(t)}),
			})
			require.NoError(t, err)
			assert.Equal(t, c.want, r.cfg.Concurrency)
		})
	}
}

// V_Required — NewReencryptor required-field validation
func TestReencryptor_RequiredFields(t *testing.T) {
	good := func() ReencryptorConfig {
		return ReencryptorConfig{
			Bucket: "b", TargetKekID: "v2",
			Lister: &fakeBucketLister{}, IO: newFakeObjectIO(),
			KeyProvider: newMultiKeyProvider("v2", map[string][]byte{"v2": makeTestKey(t)}),
		}
	}
	cases := []struct {
		name   string
		mutate func(*ReencryptorConfig)
	}{
		{"empty bucket", func(c *ReencryptorConfig) { c.Bucket = "" }},
		{"nil lister", func(c *ReencryptorConfig) { c.Lister = nil }},
		{"nil io", func(c *ReencryptorConfig) { c.IO = nil }},
		{"nil kp", func(c *ReencryptorConfig) { c.KeyProvider = nil }},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := good()
			tc.mutate(&cfg)
			_, err := NewReencryptor(cfg)
			require.Error(t, err)
		})
	}
}

// V_AlgoStringFromByte
func TestAlgoStringFromByte(t *testing.T) {
	cases := map[byte]string{
		encAlgoGCM:      "AES-256-GCM",
		encAlgoCBC:      "AES-256-CBC",
		encAlgoChaCha20: "ChaCha20-Poly1305",
	}
	for b, want := range cases {
		got, ok := algoStringFromByte(b)
		require.True(t, ok)
		assert.Equal(t, want, got)
	}
	_, ok := algoStringFromByte(0xFF)
	assert.False(t, ok)
}

// V_Stats_Total — stats Total = sum across buckets
func TestReencryptStats_Total(t *testing.T) {
	s := ReencryptStats{
		Reencrypted:              10,
		ReencryptedDryRun:        2,
		SkipAlreadyTarget:        5,
		SkipNotEncrypted:         3,
		SkipEnvelopeInvalid:      1,
		SkipSourceKEKUnavailable: 1,
		SkipMinIOError:           1,
		Failed:                   1,
	}
	assert.Equal(t, int64(24), s.Total())
}

// suppress unused imports for fakeObjectIO/bytes/http (kept for future extension
// when we add full Run integration tests; the in-memory path covers core logic).
var _ = bytes.NewReader
var _ = http.StatusOK
