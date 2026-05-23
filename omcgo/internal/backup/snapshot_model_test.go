package backup

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSnapshotFileName(t *testing.T) {
	tests := []struct {
		name      string
		rawName   string
		sn        string
		wantName  string
		wantExt   string
		wantErrIs error
	}{
		{
			name:     "backup chain raw name → normalized",
			rawName:  "backup-a1b2c3d4-SN001.xml",
			sn:       "SN001",
			wantName: "SN001_CFG.xml",
			wantExt:  "xml",
		},
		{
			name:     "uppercase ext is lowercased",
			rawName:  "anything.XML",
			sn:       "SN001",
			wantName: "SN001_CFG.xml",
			wantExt:  "xml",
		},
		{
			name:     "nv ext supported",
			rawName:  "backup-xyz-SN002.nv",
			sn:       "SN002",
			wantName: "SN002_CFG.nv",
			wantExt:  "nv",
		},
		{
			name:     "uppercase NV is lowercased",
			rawName:  "config.NV",
			sn:       "SN003",
			wantName: "SN003_CFG.nv",
			wantExt:  "nv",
		},
		{
			name:      "empty serial number",
			rawName:   "anything.xml",
			sn:        "",
			wantErrIs: ErrEmptySerialNumber,
		},
		{
			name:      "empty file name",
			rawName:   "",
			sn:        "SN001",
			wantErrIs: ErrEmptyFileName,
		},
		{
			name:      "unknown ext (cfg)",
			rawName:   "backup.cfg",
			sn:        "SN001",
			wantErrIs: ErrInvalidConfigFileExt,
		},
		{
			name:      "no extension",
			rawName:   "backup-noext",
			sn:        "SN001",
			wantErrIs: ErrInvalidConfigFileExt,
		},
		{
			name:      "json ext rejected",
			rawName:   "data.json",
			sn:        "SN001",
			wantErrIs: ErrInvalidConfigFileExt,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotExt, err := NormalizeSnapshotFileName(tt.rawName, tt.sn)
			if tt.wantErrIs != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErrIs),
					"expected error chain to include %v, got %v", tt.wantErrIs, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantExt, gotExt)
		})
	}
}

func TestValidateImportFileName(t *testing.T) {
	tests := []struct {
		name       string
		fileName   string
		expectedSN string
		wantSN     string
		wantExt    string
		wantErrIs  error
	}{
		{
			name:       "canonical xml",
			fileName:   "SN001_CFG.xml",
			expectedSN: "SN001",
			wantSN:     "SN001",
			wantExt:    "xml",
		},
		{
			name:       "canonical nv",
			fileName:   "SN002_CFG.nv",
			expectedSN: "SN002",
			wantSN:     "SN002",
			wantExt:    "nv",
		},
		{
			name:       "uppercase ext is normalized",
			fileName:   "SN003_CFG.XML",
			expectedSN: "SN003",
			wantSN:     "SN003",
			wantExt:    "xml",
		},
		{
			name:       "no expected SN: accept any prefix",
			fileName:   "DeviceXYZ_CFG.xml",
			expectedSN: "",
			wantSN:     "DeviceXYZ",
			wantExt:    "xml",
		},
		{
			name:      "empty file name",
			fileName:  "",
			wantErrIs: ErrEmptyFileName,
		},
		{
			name:      "missing _CFG suffix",
			fileName:  "SN001.xml",
			wantErrIs: ErrInvalidConfigFileName,
		},
		{
			name:      "wrong ext",
			fileName:  "SN001_CFG.cfg",
			wantErrIs: ErrInvalidConfigFileName,
		},
		{
			name:      "path traversal in name",
			fileName:  "../etc/passwd_CFG.xml",
			wantErrIs: ErrInvalidConfigFileName,
		},
		{
			name:      "path separator in name",
			fileName:  "dir/SN001_CFG.xml",
			wantErrIs: ErrInvalidConfigFileName,
		},
		{
			name:       "SN prefix mismatch",
			fileName:   "SN001_CFG.xml",
			expectedSN: "SN002",
			wantErrIs:  ErrFileNameSerialMismatch,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSN, gotExt, err := ValidateImportFileName(tt.fileName, tt.expectedSN)
			if tt.wantErrIs != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErrIs),
					"expected error chain to include %v, got %v", tt.wantErrIs, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSN, gotSN)
			assert.Equal(t, tt.wantExt, gotExt)
		})
	}
}

func TestSnapshotObjectPath(t *testing.T) {
	assert.Equal(t, "SN001_CFG.xml", SnapshotObjectPath("SN001", "xml"))
	assert.Equal(t, "SN002_CFG.nv", SnapshotObjectPath("SN002", "nv"))
}
