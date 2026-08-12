package transfercfg

import (
	"context"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicy_Snapshot_UsesDefaultsWhenDBKeysMissing(t *testing.T) {
	defaults := DefaultsFromACSConfig(appconfig.ACSConfig{
		Upload: appconfig.UploadConfig{
			BaseURL:     "http://bootstrap.example.com/",
			Path:        "smallcell/FileUploadService",
			Username:    "upload-user",
			Password:    "upload-pass",
			MaxFileSize: 1024,
		},
		Download: appconfig.DownloadConfig{
			BaseURL:  "http://bootstrap.example.com/",
			Path:     "smallcell/FileDownloadService",
			Username: "download-user",
			Password: "download-pass",
		},
	})

	policy := NewPolicy(defaults, nil)
	snap := policy.Snapshot(context.Background())

	assert.Equal(t, ProtocolPolicyForceHTTP, snap.ProtocolPolicy)
	assert.Equal(t, "http://bootstrap.example.com", snap.Upload.BaseURL)
	assert.Equal(t, "/smallcell/FileUploadService", snap.Upload.Path)
	assert.Equal(t, int64(1024), snap.Upload.MaxFileSize)
	assert.Equal(t, "download-user", snap.Download.Username)
	assert.Equal(t, "/smallcell/FileDownloadService", snap.Download.Path)
}

func TestPolicy_Snapshot_DBOverrideWins(t *testing.T) {
	lookup := func(_ context.Context, category, key string) (string, bool) {
		values := map[string]string{
			Category + "." + KeyProtocolPolicy:       ProtocolPolicyPreferHTTPS,
			Category + "." + KeyUploadBaseURL:        "http://db-upload.example.com/",
			Category + "." + KeyHTTPSUploadBaseURL:   "https://db-upload.example.com/",
			Category + "." + KeyUploadPath:           "custom-upload",
			Category + "." + KeyUploadUsername:       "db-up-user",
			Category + "." + KeyUploadPassword:       "db-up-pass",
			Category + "." + KeyUploadMaxFileSize:    "2048",
			Category + "." + KeyDownloadBaseURL:      "http://db-download.example.com/",
			Category + "." + KeyHTTPSDownloadBaseURL: "https://db-download.example.com/",
			Category + "." + KeyDownloadPath:         "custom-download",
			Category + "." + KeyDownloadUsername:     "db-down-user",
			Category + "." + KeyDownloadPassword:     "db-down-pass",
		}
		value, ok := values[category+"."+key]
		return value, ok
	}

	policy := NewPolicy(Snapshot{
		Upload: UploadSettings{
			BaseURL:     "http://bootstrap-upload.example.com",
			Path:        "/smallcell/FileUploadService",
			Username:    "bootstrap-up-user",
			Password:    "bootstrap-up-pass",
			MaxFileSize: 1024,
		},
		Download: DownloadSettings{
			BaseURL:  "http://bootstrap-download.example.com",
			Path:     "/smallcell/FileDownloadService",
			Username: "bootstrap-down-user",
			Password: "bootstrap-down-pass",
		},
	}, lookup)

	snap := policy.Snapshot(context.Background())
	require.Equal(t, ProtocolPolicyPreferHTTPS, snap.ProtocolPolicy)
	require.Equal(t, "http://db-upload.example.com", snap.Upload.BaseURL)
	assert.Equal(t, "https://db-upload.example.com", snap.Upload.HTTPSBaseURL)
	assert.Equal(t, "/custom-upload", snap.Upload.Path)
	assert.Equal(t, "db-up-user", snap.Upload.Username)
	assert.Equal(t, "db-up-pass", snap.Upload.Password)
	assert.Equal(t, int64(2048), snap.Upload.MaxFileSize)
	assert.Equal(t, "http://db-download.example.com", snap.Download.BaseURL)
	assert.Equal(t, "https://db-download.example.com", snap.Download.HTTPSBaseURL)
	assert.Equal(t, "/custom-download", snap.Download.Path)
	assert.Equal(t, "db-down-user", snap.Download.Username)
	assert.Equal(t, "db-down-pass", snap.Download.Password)
}

func TestPolicy_Snapshot_BlankDBOverrideFallsBackToDefaults(t *testing.T) {
	lookup := func(_ context.Context, category, key string) (string, bool) {
		values := map[string]string{
			Category + "." + KeyProtocolPolicy:       "   ",
			Category + "." + KeyUploadBaseURL:        "   ",
			Category + "." + KeyHTTPSUploadBaseURL:   " ",
			Category + "." + KeyUploadPath:           "",
			Category + "." + KeyUploadUsername:       " ",
			Category + "." + KeyUploadPassword:       "",
			Category + "." + KeyDownloadBaseURL:      " ",
			Category + "." + KeyHTTPSDownloadBaseURL: "",
			Category + "." + KeyDownloadPath:         "",
			Category + "." + KeyDownloadUsername:     "  ",
			Category + "." + KeyDownloadPassword:     "",
		}
		value, ok := values[category+"."+key]
		return value, ok
	}

	policy := NewPolicy(Snapshot{
		Upload: UploadSettings{
			BaseURL:     "http://bootstrap-upload.example.com",
			Path:        "/smallcell/FileUploadService",
			Username:    "bootstrap-up-user",
			Password:    "bootstrap-up-pass",
			MaxFileSize: 1024,
		},
		Download: DownloadSettings{
			BaseURL:  "http://bootstrap-download.example.com",
			Path:     "/smallcell/FileDownloadService",
			Username: "bootstrap-down-user",
			Password: "bootstrap-down-pass",
		},
	}, lookup)

	snap := policy.Snapshot(context.Background())
	assert.Equal(t, ProtocolPolicyForceHTTP, snap.ProtocolPolicy)
	assert.Equal(t, "http://bootstrap-upload.example.com", snap.Upload.BaseURL)
	assert.Empty(t, snap.Upload.HTTPSBaseURL)
	assert.Equal(t, "/smallcell/FileUploadService", snap.Upload.Path)
	assert.Equal(t, "bootstrap-up-user", snap.Upload.Username)
	assert.Equal(t, "bootstrap-up-pass", snap.Upload.Password)
	assert.Equal(t, "http://bootstrap-download.example.com", snap.Download.BaseURL)
	assert.Empty(t, snap.Download.HTTPSBaseURL)
	assert.Equal(t, "/smallcell/FileDownloadService", snap.Download.Path)
	assert.Equal(t, "bootstrap-down-user", snap.Download.Username)
	assert.Equal(t, "bootstrap-down-pass", snap.Download.Password)
}

func TestPolicy_Snapshot_ReloadsProtocolFieldsAfterCacheExpiry(t *testing.T) {
	values := map[string]string{
		KeyProtocolPolicy:       ProtocolPolicyForceHTTP,
		KeyHTTPSUploadBaseURL:   "https://old-upload.example.com",
		KeyHTTPSDownloadBaseURL: "https://old-download.example.com",
	}
	policy := NewPolicy(Snapshot{}, func(_ context.Context, category, key string) (string, bool) {
		if category != Category {
			return "", false
		}
		value, ok := values[key]
		return value, ok
	})

	initial := policy.Snapshot(context.Background())
	require.Equal(t, ProtocolPolicyForceHTTP, initial.ProtocolPolicy)

	values[KeyProtocolPolicy] = ProtocolPolicyPreferHTTPS
	values[KeyHTTPSUploadBaseURL] = "https://new-upload.example.com"
	values[KeyHTTPSDownloadBaseURL] = "https://new-download.example.com"
	policy.cache.Load().expiresAt = time.Now().Add(-time.Second)

	refreshed := policy.Snapshot(context.Background())
	assert.Equal(t, ProtocolPolicyPreferHTTPS, refreshed.ProtocolPolicy)
	assert.Equal(t, "https://new-upload.example.com", refreshed.Upload.HTTPSBaseURL)
	assert.Equal(t, "https://new-download.example.com", refreshed.Download.HTTPSBaseURL)
}
