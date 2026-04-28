package redact

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"password", true},
		{"Password", true},
		{"PASSWORD", true},
		{"pass_word", true}, // normalize strips '_' so this matches "password"
		{"refresh_token", true},
		{"refreshToken", true},
		{"refresh-token", true},
		{"X-API-Key", true},
		{"x_api_key", true},
		{"username", false},
		{"user_id", false},
		{"", false},
		{"snmp_community", true},
		{"device_password", true},
		{"private_key", true},
		{"PrivateKey", true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSensitive(tt.key))
		})
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"single char", "x", "*"},
		{"short - 5 chars", "12345", "*****"},
		{"at threshold - 6 chars", "abcdef", "******"},
		{"long - 7 chars", "abcdefg", "ab***fg"},
		{"long - secretvalue", "secretvalue", "se***ue"},
		{"long - 20 chars", "abcdefghij1234567890", "ab***90"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMaskString_NoLeakOfMiddle(t *testing.T) {
	// Defense check: long secrets must never leak more than 2+2 chars.
	in := "supersecretpassword12345"
	out := MaskString(in)
	assert.NotContains(t, out, "secret")
	assert.NotContains(t, out, "password")
	assert.NotContains(t, out, "ersec")
}

func TestMaskAny(t *testing.T) {
	assert.Equal(t, "se***ue", MaskAny("secretvalue"))
	assert.Equal(t, "se***ue", MaskAny([]byte("secretvalue")))
	assert.Equal(t, "***", MaskAny(12345))
	assert.Equal(t, "***", MaskAny(map[string]string{"k": "v"}))
	assert.Nil(t, MaskAny(nil))
}

func TestRedactMap_FlatSensitive(t *testing.T) {
	in := map[string]any{
		"username": "alice",
		"password": "supersecret",
		"token":    "abcdef123456",
		"age":      30,
	}
	out := RedactMap(in)
	assert.Equal(t, "alice", out["username"])
	assert.Equal(t, "su***et", out["password"])
	assert.Equal(t, "ab***56", out["token"])
	assert.Equal(t, 30, out["age"])
	// Input is NOT mutated.
	assert.Equal(t, "supersecret", in["password"])
}

func TestRedactMap_Nested(t *testing.T) {
	in := map[string]any{
		"user": "alice",
		"profile": map[string]any{
			"password":      "secret123",
			"refresh_token": "rt-abcdef-xyz",
		},
		"devices": []any{
			map[string]any{
				"sn":              "DEV001",
				"device_password": "dev-pwd-1234",
			},
			map[string]any{
				"sn":             "DEV002",
				"snmp_community": "public-but-secret",
			},
		},
	}
	out := RedactMap(in)

	profile, ok := out["profile"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "se***23", profile["password"])
	assert.Equal(t, "rt***yz", profile["refresh_token"])

	devices, ok := out["devices"].([]any)
	require.True(t, ok)
	require.Len(t, devices, 2)

	d0 := devices[0].(map[string]any)
	assert.Equal(t, "DEV001", d0["sn"])
	assert.Equal(t, "de***34", d0["device_password"])

	d1 := devices[1].(map[string]any)
	assert.Equal(t, "DEV002", d1["sn"])
	assert.Equal(t, "pu***et", d1["snmp_community"])
}

// TestRedactMap_TopLevelSensitiveKey verifies that when a top-level key
// is itself sensitive (e.g. "auth", "credentials"), the whole subtree is
// scrubbed via MaskAny.
func TestRedactMap_TopLevelSensitiveKey(t *testing.T) {
	in := map[string]any{
		"user": "alice",
		"auth": map[string]any{
			"password": "secret123",
		},
	}
	out := RedactMap(in)
	assert.Equal(t, "alice", out["user"])
	// MaskAny on a non-string returns the constant placeholder.
	assert.Equal(t, "***", out["auth"])
}

func TestRedactMap_NilSafe(t *testing.T) {
	assert.Nil(t, RedactMap(nil))
}

func TestRedactJSON_Object(t *testing.T) {
	in := []byte(`{"user":"alice","password":"supersecret","Authorization":"Bearer xyz123abcdef"}`)
	out := RedactJSON(in)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, "alice", got["user"])
	assert.Equal(t, "su***et", got["password"])
	assert.Equal(t, "Be***ef", got["Authorization"])

	// Plaintext must not appear in the byte stream.
	assert.NotContains(t, string(out), "supersecret")
	assert.NotContains(t, string(out), "xyz123abcdef")
}

func TestRedactJSON_Nested(t *testing.T) {
	in := []byte(`{"data":{"creds":{"api_key":"k-1234567890","name":"n1"}}}`)
	out := RedactJSON(in)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))

	data := got["data"].(map[string]any)
	creds := data["creds"].(map[string]any)
	assert.Equal(t, "n1", creds["name"])
	assert.Equal(t, "k-***90", creds["api_key"])
	assert.NotContains(t, string(out), "1234567890")
}

func TestRedactJSON_InvalidJSON_Passthrough(t *testing.T) {
	in := []byte(`not json`)
	out := RedactJSON(in)
	assert.Equal(t, in, out)
}

func TestRedactJSON_Empty(t *testing.T) {
	assert.Equal(t, []byte{}, RedactJSON([]byte{}))
	assert.Nil(t, RedactJSON(nil))
}

func TestRedactJSON_Array(t *testing.T) {
	in := []byte(`[{"password":"abcdefgh"},{"name":"n"}]`)
	out := RedactJSON(in)

	var arr []map[string]any
	require.NoError(t, json.Unmarshal(out, &arr))
	require.Len(t, arr, 2)
	assert.Equal(t, "ab***gh", arr[0]["password"])
	assert.Equal(t, "n", arr[1]["name"])
}

func TestZapField_String_Sensitive(t *testing.T) {
	f := String("password", "supersecret")
	assert.Equal(t, "password", f.Key)
	assert.Equal(t, zapcore.StringType, f.Type)
	assert.Equal(t, "su***et", f.String)
}

func TestZapField_String_NonSensitive(t *testing.T) {
	f := String("user", "alice")
	assert.Equal(t, "user", f.Key)
	assert.Equal(t, "alice", f.String)
}

func TestZapField_Stringp_Nil(t *testing.T) {
	f := Stringp("password", nil)
	// zap.Stringp with nil yields an empty string field — we only need
	// to make sure it doesn't panic and doesn't leak.
	assert.Equal(t, "password", f.Key)
}

func TestZapField_Stringp_Sensitive(t *testing.T) {
	v := "topsecret"
	f := Stringp("token", &v)
	assert.Equal(t, "token", f.Key)
	assert.Equal(t, "to***et", f.String)
}

func TestZapField_Any_Sensitive(t *testing.T) {
	f := Any("password", "supersecret")
	assert.Equal(t, "password", f.Key)
	// zap.Any of string -> StringType
	assert.Equal(t, zapcore.StringType, f.Type)
	assert.Equal(t, "su***et", f.String)
}

func TestZapField_Any_NonSensitive_PreservesType(t *testing.T) {
	f := Any("count", 42)
	assert.Equal(t, "count", f.Key)
	assert.Equal(t, zapcore.Int64Type, f.Type)
	assert.Equal(t, int64(42), f.Integer)
}

// captureCore returns a logger that captures fields into a slice for inspection.
func captureCore() (*zap.Logger, *observedCore) {
	c := &observedCore{}
	return zap.New(c), c
}

type observedCore struct {
	entries []capturedEntry
}

type capturedEntry struct {
	msg    string
	fields []zapcore.Field
}

func (c *observedCore) Enabled(zapcore.Level) bool { return true }
func (c *observedCore) With(fields []zapcore.Field) zapcore.Core {
	return c // simplistic — fields applied here would be merged on Write
}
func (c *observedCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(e, c)
}
func (c *observedCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	c.entries = append(c.entries, capturedEntry{msg: e.Message, fields: fields})
	return nil
}
func (c *observedCore) Sync() error { return nil }

func TestZapIntegration_LoggerNeverEmitsPlaintext(t *testing.T) {
	logger, core := captureCore()
	logger.Info("login",
		zap.String("user", "alice"),
		String("password", "supersecret"),
		String("token", "Bearer xyz123"),
	)
	require.Len(t, core.entries, 1)
	entry := core.entries[0]
	for _, f := range entry.fields {
		assert.NotEqual(t, "supersecret", f.String, "plaintext password must not be in field %q", f.Key)
		assert.NotContains(t, f.String, "xyz123", "plaintext token must not appear in field %q", f.Key)
	}
}
