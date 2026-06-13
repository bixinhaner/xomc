package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gz(t *testing.T, plain string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write([]byte(plain))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestMaybeGunzip(t *testing.T) {
	const xml = `<?xml version="1.0"?><measCollecFile/>`

	tests := []struct {
		name        string
		input       []byte
		wantApplied bool
		wantOut     string
		wantErr     bool
	}{
		{
			name:        "plaintext xml passes through unchanged",
			input:       []byte(xml),
			wantApplied: false,
			wantOut:     xml,
		},
		{
			name:        "gzip stream is decompressed",
			input:       gz(t, xml),
			wantApplied: true,
			wantOut:     xml,
		},
		{
			name:        "empty stream passes through (not gzip)",
			input:       []byte{},
			wantApplied: false,
			wantOut:     "",
		},
		{
			name:        "single byte passes through (too short for magic)",
			input:       []byte{0x1f},
			wantApplied: false,
			wantOut:     "\x1f",
		},
		{
			name:        "first magic byte only, second differs → passthrough",
			input:       []byte{0x1f, 0x00, 0x01},
			wantApplied: false,
			wantOut:     "\x1f\x00\x01",
		},
		{
			name:    "gzip magic but corrupt body → error",
			input:   []byte{0x1f, 0x8b, 0x08, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, applied, err := MaybeGunzip(bytes.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantApplied, applied)
			got, rerr := io.ReadAll(out)
			require.NoError(t, rerr)
			assert.Equal(t, tt.wantOut, string(got))
		})
	}
}

func TestIsGzip(t *testing.T) {
	assert.True(t, IsGzip([]byte{0x1f, 0x8b, 0x08}))
	assert.False(t, IsGzip([]byte{0x1f, 0x00}))
	assert.False(t, IsGzip([]byte{0x1f}))
	assert.False(t, IsGzip(nil))
	assert.False(t, IsGzip([]byte("<?xml")))
}
