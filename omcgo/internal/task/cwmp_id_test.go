package task

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GenerateCWMPID_Format(t *testing.T) {
	methods := []string{
		"GetParameterValues",
		"SetParameterValues",
		"GetParameterNames",
		"Download",
		"Reboot",
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			id := GenerateCWMPID(method)

			assert.True(t, strings.HasPrefix(id, cwmpIDPrefix),
				"CWMP ID should start with prefix %q, got %q", cwmpIDPrefix, id)
			assert.Contains(t, id, method,
				"CWMP ID should contain method name")

			// Format: ID:intrnl.unset.id.{Method}{timestamp}.{random}
			rest := strings.TrimPrefix(id, cwmpIDPrefix)
			lastDot := strings.LastIndex(rest, ".")
			require.NotEqual(t, -1, lastDot, "should have a dot separating timestamp and random")

			// Random part should be 8 digits
			randomPart := rest[lastDot+1:]
			assert.Len(t, randomPart, 8, "random part should be 8 digits")
		})
	}
}

func Test_GenerateCWMPID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateCWMPID("GetParameterValues")
		assert.False(t, ids[id], "CWMP ID should be unique, got duplicate: %s", id)
		ids[id] = true
	}
}

func Test_ParseCWMPID_Success(t *testing.T) {
	tests := []struct {
		name           string
		cwmpID         string
		expectedMethod string
		wantOK         bool
	}{
		{
			name:           "GetParameterValues",
			cwmpID:         "ID:intrnl.unset.id.GetParameterValues1772248515.45470049",
			expectedMethod: "GetParameterValues",
			wantOK:         true,
		},
		{
			name:           "SetParameterValues",
			cwmpID:         "ID:intrnl.unset.id.SetParameterValues1772248515.12345678",
			expectedMethod: "SetParameterValues",
			wantOK:         true,
		},
		{
			name:           "Download",
			cwmpID:         "ID:intrnl.unset.id.Download1772248515.99999999",
			expectedMethod: "Download",
			wantOK:         true,
		},
		{
			name:           "Reboot",
			cwmpID:         "ID:intrnl.unset.id.Reboot1772248515.00000001",
			expectedMethod: "Reboot",
			wantOK:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, timestamp, ok := ParseCWMPID(tt.cwmpID)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.expectedMethod, method)
			assert.Greater(t, timestamp, int64(0), "timestamp should be positive")
		})
	}
}

func Test_ParseCWMPID_Roundtrip(t *testing.T) {
	methods := []string{"GetParameterValues", "SetParameterValues", "Reboot", "Download"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			id := GenerateCWMPID(method)
			parsedMethod, timestamp, ok := ParseCWMPID(id)

			require.True(t, ok, "ParseCWMPID should succeed for generated ID")
			assert.Equal(t, method, parsedMethod)
			assert.Greater(t, timestamp, int64(0))
		})
	}
}

func Test_ParseCWMPID_InvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		cwmpID string
	}{
		{"empty string", ""},
		{"wrong prefix", "WRONG:prefix.GetParameterValues1234.5678"},
		{"no dot separator", "ID:intrnl.unset.id.GetParameterValues1234"},
		{"only prefix", "ID:intrnl.unset.id."},
		{"no method name - digits only", "ID:intrnl.unset.id.1234.5678"},
		{"random garbage", "some-random-string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, timestamp, ok := ParseCWMPID(tt.cwmpID)
			assert.False(t, ok, "should return false for invalid input %q", tt.cwmpID)
			assert.Empty(t, method)
			assert.Equal(t, int64(0), timestamp)
		})
	}
}

func Test_CWMPIDHash_ShortID(t *testing.T) {
	// IDs <= 32 chars should be returned as-is
	shortIDs := []string{
		"abc",
		"12345678901234567890123456789012", // exactly 32 chars
		"",
		"a",
	}

	for _, id := range shortIDs {
		t.Run(id, func(t *testing.T) {
			hash := CWMPIDHash(id)
			assert.Equal(t, id, hash, "short IDs should be returned as-is")
		})
	}
}

func Test_CWMPIDHash_LongID(t *testing.T) {
	longID := "ID:intrnl.unset.id.GetParameterValues1772248515705.45470049"
	require.Greater(t, len(longID), 32, "test ID should be > 32 chars")

	hash := CWMPIDHash(longID)
	assert.NotEqual(t, longID, hash, "long IDs should be hashed")
	// Hash is hex-encoded 16 bytes = 32 hex chars
	assert.Len(t, hash, 32, "hash should be 32 hex characters")
}

func Test_CWMPIDHash_Consistent(t *testing.T) {
	id := "ID:intrnl.unset.id.GetParameterValues1772248515705.45470049"
	hash1 := CWMPIDHash(id)
	hash2 := CWMPIDHash(id)
	assert.Equal(t, hash1, hash2, "same input should produce same hash")
}

func Test_CWMPIDHash_DifferentInputs(t *testing.T) {
	id1 := "ID:intrnl.unset.id.GetParameterValues1772248515705.45470049"
	id2 := "ID:intrnl.unset.id.SetParameterValues1772248515705.45470049"
	hash1 := CWMPIDHash(id1)
	hash2 := CWMPIDHash(id2)
	assert.NotEqual(t, hash1, hash2, "different inputs should produce different hashes")
}

func Test_generateUUID_Format(t *testing.T) {
	uuid := generateUUID()

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
	parts := strings.Split(uuid, "-")
	require.Len(t, parts, 5, "UUID should have 5 parts separated by hyphens")
	assert.Len(t, parts[0], 8)
	assert.Len(t, parts[1], 4)
	assert.Len(t, parts[2], 4)
	assert.Len(t, parts[3], 4)
	assert.Len(t, parts[4], 12)

	// Version 4: third section starts with '4'
	assert.Equal(t, byte('4'), parts[2][0], "UUID v4 third section should start with '4'")
}

func Test_generateUUID_Unique(t *testing.T) {
	uuids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uuid := generateUUID()
		assert.False(t, uuids[uuid], "UUID should be unique, got duplicate: %s", uuid)
		uuids[uuid] = true
	}
}

func Test_randomInt64_Range(t *testing.T) {
	for i := 0; i < 100; i++ {
		n := randomInt64(100)
		assert.GreaterOrEqual(t, n, int64(0))
		assert.Less(t, n, int64(100))
	}
}

func Test_randomInt64_ZeroMax(t *testing.T) {
	assert.Equal(t, int64(0), randomInt64(0))
	assert.Equal(t, int64(0), randomInt64(-1))
}

func Test_parseInt64_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0", 0},
		{"1", 1},
		{"1772248515", 1772248515},
		{"99999999", 99999999},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseInt64(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_parseInt64_StopsAtNonDigit(t *testing.T) {
	// parseInt64 stops at first non-digit character
	result, err := parseInt64("123abc")
	require.NoError(t, err)
	assert.Equal(t, int64(123), result)
}
