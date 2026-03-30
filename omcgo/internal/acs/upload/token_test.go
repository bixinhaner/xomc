package upload

import (
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test TokenManager — JWT token generation and validation
// ---------------------------------------------------------------------------

func Test_TokenManager_GenerateAndValidate(t *testing.T) {
	tm := NewTokenManager("test-secret-key", 5*time.Minute)

	token, err := tm.Generate("SN001", "cmd-key-1", "4", "pm_20260322.xml")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := tm.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "SN001", claims.DeviceSN)
	assert.Equal(t, "cmd-key-1", claims.CommandKey)
	assert.Equal(t, "4", claims.FileType)
	assert.Equal(t, "pm_20260322.xml", claims.TargetFileName)
	assert.Equal(t, "cmd-key-1", claims.ID)
}

func Test_TokenManager_Validate_ExpiredToken(t *testing.T) {
	tm := NewTokenManager("test-secret", -1*time.Second) // already expired

	token, err := tm.Generate("SN001", "cmd-key-1", "4", "file.xml")
	require.NoError(t, err)

	_, err = tm.Validate(token)
	assert.Error(t, err, "expired token should fail validation")
	assert.Contains(t, err.Error(), "parse token")
}

func Test_TokenManager_Validate_WrongSecret(t *testing.T) {
	tmGen := NewTokenManager("secret-1", 5*time.Minute)
	tmVal := NewTokenManager("secret-2", 5*time.Minute)

	token, err := tmGen.Generate("SN001", "cmd-key-1", "4", "file.xml")
	require.NoError(t, err)

	_, err = tmVal.Validate(token)
	assert.Error(t, err, "token signed with different secret should fail")
}

func Test_TokenManager_Validate_InvalidTokenString(t *testing.T) {
	tm := NewTokenManager("test-secret", 5*time.Minute)

	_, err := tm.Validate("not-a-valid-jwt-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse token")
}

func Test_TokenManager_Validate_EmptyToken(t *testing.T) {
	tm := NewTokenManager("test-secret", 5*time.Minute)

	_, err := tm.Validate("")
	assert.Error(t, err)
}

func Test_TokenManager_Generate_DifferentTokensPerCall(t *testing.T) {
	tm := NewTokenManager("test-secret", 5*time.Minute)

	token1, err := tm.Generate("SN001", "cmd-1", "4", "file1.xml")
	require.NoError(t, err)

	token2, err := tm.Generate("SN001", "cmd-2", "4", "file2.xml")
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2, "different parameters should produce different tokens")
}

// ---------------------------------------------------------------------------
// Test normalizeFileType + storage.BucketAndCategory routing
// ---------------------------------------------------------------------------

func Test_normalizeFileType(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected tr069.FileType
	}{
		{"PM numeric", "4", tr069.FileTypePM},
		{"PM string", "PM", tr069.FileTypePM},
		{"MR numeric", "5", tr069.FileTypeMR},
		{"MR string", "MR", tr069.FileTypeMR},
		{"Log numeric", "6", tr069.FileTypeRunningLog},
		{"LOG uppercase", "LOG", tr069.FileTypeRunningLog},
		{"DataModel numeric", "11", tr069.FileTypeDataModel},
		{"DataModel string", "PARAMETER MODEL", tr069.FileTypeDataModel},
		{"Firmware", "1", tr069.FileTypeFirmware},
		{"unknown defaults to log", "99", tr069.FileTypeRunningLog},
		{"empty defaults to log", "", tr069.FileTypeRunningLog},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeFileType(tt.raw))
		})
	}
}

func Test_BucketAndCategory_Routing(t *testing.T) {
	buckets := testBuckets()

	tests := []struct {
		name           string
		fileType       string
		expectedBucket string
	}{
		{"PM numeric", "4", buckets.PMFiles},
		{"PM string", "PM", buckets.PMFiles},
		{"MR numeric", "5", buckets.MRFiles},
		{"MR string", "MR", buckets.MRFiles},
		{"Log numeric", "6", buckets.Logs},
		{"LOG uppercase", "LOG", buckets.Logs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ft := normalizeFileType(tt.fileType)
			bucket, _ := storage.BucketAndCategory(ft, buckets)
			assert.Equal(t, tt.expectedBucket, bucket)
		})
	}
}

// ---------------------------------------------------------------------------
// Test Uploader helper methods — bucketForFileType, objectPath
// ---------------------------------------------------------------------------

func Test_Uploader_bucketForFileType(t *testing.T) {
	u := &Uploader{buckets: testBuckets()}

	tests := []struct {
		name     string
		fileType string
		expected string
	}{
		{"type 4 PM", "4", "omc-pm-files"},
		{"type 5 MR", "5", "omc-mr-files"},
		{"type 6 Log", "6", "omc-logs"},
		{"unknown defaults to PM", "9", "omc-pm-files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, u.bucketForFileType(tt.fileType))
		})
	}
}

func Test_Uploader_objectPath_Format(t *testing.T) {
	u := &Uploader{}
	path := u.objectPath("SN001", "pm_file.xml")

	// Format: {YYYY}/{MM}/{DD}/{deviceSN}/{filename}
	assert.Contains(t, path, "SN001")
	assert.Contains(t, path, "pm_file.xml")
	assert.Regexp(t, `\d{4}/\d{2}/\d{2}/SN001/pm_file.xml`, path)
}

// ---------------------------------------------------------------------------
// Test SessionStore key format
// ---------------------------------------------------------------------------

func Test_SessionStore_KeyFormat(t *testing.T) {
	store := &SessionStore{}
	key := store.key("SN001", "cmd-1")
	assert.Equal(t, "upload:session:SN001:cmd-1", key)
}

func Test_SessionStore_KeyFormat_SpecialChars(t *testing.T) {
	store := &SessionStore{}
	key := store.key("device/sn", "cmd:key")
	assert.Equal(t, "upload:session:device/sn:cmd:key", key)
}

// ---------------------------------------------------------------------------
// Test Session struct JSON marshaling
// ---------------------------------------------------------------------------

func Test_Session_Struct(t *testing.T) {
	now := time.Now()
	s := Session{
		DeviceSN:       "SN001",
		CommandKey:     "cmd-1",
		FileType:       "4",
		Bucket:         "omc-pm-files",
		ObjectPath:     "pm/2026/03/22/file.xml",
		FileSize:       1024,
		UploadedAt:     now,
		TargetFileName: "file.xml",
	}

	assert.Equal(t, "SN001", s.DeviceSN)
	assert.Equal(t, "cmd-1", s.CommandKey)
	assert.Equal(t, int64(1024), s.FileSize)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func testBuckets() appconfig.BucketConfig {
	return appconfig.BucketConfig{
		PMFiles: "omc-pm-files",
		MRFiles: "omc-mr-files",
		Logs:    "omc-logs",
	}
}
