package license

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- PDF ---

func TestWriteSingleLicensePDF_Basic(t *testing.T) {
	licID := uuid.New()
	expiry := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	licensor := "OEM Corp"
	region := "east"
	deviceType := "pico"
	notes := "test license"

	lic := &License{
		ID:              licID,
		LicenseCode:     "LIC-EXP-001",
		LicenseName:     "Test Export License",
		ProductName:     "OMC Pro",
		LicenseType:     TypeSubscription,
		Status:          StatusActive,
		MaxDevices:      1000,
		UsedDevices:     250,
		GracePeriodDays: 30,
		Features:        json.RawMessage(`["pm","alarm"]`),
		IssueDate:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiryDate:      &expiry,
		Licensor:        &licensor,
		DeviceType:      &deviceType,
		Region:          &region,
		Notes:           &notes,
	}

	logs := []LicenseLog{
		{
			ID:        uuid.New(),
			LogType:   LogTypeActivate,
			Result:    LogResultSuccess,
			Details:   json.RawMessage(`{"summary":"activated"}`),
			CreatedAt: time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	err := WriteSingleLicensePDF(&buf, ExportContext{
		License:     lic,
		RecentLogs:  logs,
		GeneratedAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		GeneratedBy: "admin",
	})
	require.NoError(t, err)

	// PDF magic bytes start with "%PDF-"
	out := buf.Bytes()
	require.GreaterOrEqual(t, len(out), 100, "PDF should be at least 100 bytes")
	assert.Equal(t, []byte("%PDF-"), out[:5], "must start with PDF magic header")
}

func TestWriteSingleLicensePDF_NilLicense_Errors(t *testing.T) {
	var buf bytes.Buffer
	err := WriteSingleLicensePDF(&buf, ExportContext{License: nil})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "license is required")
}

func TestWriteSingleLicensePDF_PerpetualLicense(t *testing.T) {
	// ExpiryDate=nil → "(perpetual)" 文案
	lic := &License{
		ID:          uuid.New(),
		LicenseCode: "LIC-PERP-001",
		LicenseName: "Perpetual",
		ProductName: "OMC Core",
		LicenseType: TypePerpetual,
		Status:      StatusActive,
		MaxDevices:  100,
		IssueDate:   time.Now(),
		Features:    json.RawMessage(`[]`),
	}
	var buf bytes.Buffer
	err := WriteSingleLicensePDF(&buf, ExportContext{
		License:     lic,
		GeneratedAt: time.Now(),
	})
	require.NoError(t, err)
	assert.Greater(t, buf.Len(), 100)
}

func TestAsciiSafe(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"abc", "abc"},
		{"中文", "??"},
		{"Hello 世界", "Hello ??"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, asciiSafe(tt.in))
	}
}

// --- CSV ---

func TestWriteAllActiveCSV_HeaderAndRows(t *testing.T) {
	region := "east"
	licensor := "OEM"
	now := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	licenses := []*License{
		{
			ID:              uuid.New(),
			LicenseCode:     "LIC-CSV-001",
			LicenseName:     "Lic A",
			ProductName:     "Prod A",
			LicenseType:     TypeSubscription,
			Status:          StatusActive,
			MaxDevices:      500,
			UsedDevices:     100,
			GracePeriodDays: 7,
			IssueDate:       now,
			ExpiryDate:      &now,
			Licensor:        &licensor,
			Region:          &region,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:          uuid.New(),
			LicenseCode: "LIC-CSV-002",
			LicenseName: "Lic B",
			ProductName: "Prod B",
			LicenseType: TypePerpetual,
			Status:      StatusActive,
			MaxDevices:  100,
			IssueDate:   now,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	var buf bytes.Buffer
	err := WriteAllActiveCSV(&buf, licenses)
	require.NoError(t, err)

	body := buf.String()
	// UTF-8 BOM
	assert.True(t, strings.HasPrefix(body, "\xef\xbb\xbf"), "must start with UTF-8 BOM")

	// Strip BOM and parse
	bodyNoBOM := strings.TrimPrefix(body, "\xef\xbb\xbf")
	r := csv.NewReader(strings.NewReader(bodyNoBOM))
	rows, err := r.ReadAll()
	require.NoError(t, err)

	require.Len(t, rows, 3, "header + 2 rows")
	assert.Equal(t, "id", rows[0][0])
	assert.Equal(t, "license_code", rows[0][1])
	assert.Equal(t, "LIC-CSV-001", rows[1][1])
	assert.Equal(t, "LIC-CSV-002", rows[2][1])
	// 第一条 ExpiryDate=now → 格式化日期；第二条 nil → "(perpetual)"
	assert.NotEqual(t, "(perpetual)", rows[1][10])
	assert.Equal(t, "(perpetual)", rows[2][10])
}

func TestWriteAllActiveCSV_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteAllActiveCSV(&buf, nil)
	require.NoError(t, err)

	body := buf.String()
	assert.True(t, strings.HasPrefix(body, "\xef\xbb\xbf"))
	// 仅 header
	bodyNoBOM := strings.TrimPrefix(body, "\xef\xbb\xbf")
	rows, err := csv.NewReader(strings.NewReader(bodyNoBOM)).ReadAll()
	require.NoError(t, err)
	assert.Len(t, rows, 1, "header only")
}
