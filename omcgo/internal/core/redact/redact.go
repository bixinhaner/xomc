// Package redact provides helpers for masking sensitive data (passwords,
// tokens, API keys, device credentials) before they are written to logs or
// returned in error responses.
//
// It defines:
//
//   - SensitiveKeys: the canonical list of field names treated as secret
//     (case-insensitive match).
//   - MaskString: replace a string value with a fixed mask, preserving
//     length information for short strings or showing head/tail for longer
//     ones.
//   - RedactMap: walk a map[string]any recursively and mask every value
//     whose key is sensitive.
//   - RedactJSON: decode JSON, redact, re-encode (used for log/error
//     bodies that arrive as bytes).
//   - String / NamedError: helpers built on go.uber.org/zap.Field that
//     hide sensitive values at the field-construction site, so that
//     callers can drop in `redact.String("password", pwd)` exactly where
//     they would have written `zap.String(...)`.
//
// The package is intentionally allocation-light and dependency-free apart
// from `go.uber.org/zap`, which is already required by the logger.
package redact

import (
	"encoding/json"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SensitiveKeys lists the field names that must be masked when logged or
// returned in error responses. Matching is case-insensitive and tolerates
// common separators ("_", "-").
//
// The list is deliberately conservative: it captures the names actually
// used in this codebase (TR-069 device passwords, ACS auth tokens, JWT
// refresh tokens, SNMP community strings, etc.) plus the standard set of
// aliases recommended by OWASP Logging Cheat Sheet.
var SensitiveKeys = []string{
	"password", "passwd", "pwd",
	"secret",
	"token", "access_token", "refresh_token", "id_token", "jwt", "bearer",
	"api_key", "apikey", "x-api-key",
	"authorization", "auth",
	"private_key", "privatekey",
	"credential", "credentials",
	"device_password", "device_secret",
	"snmp_community", "community",
	"client_secret",
	"session_key", "session_token",
}

// IsSensitive reports whether the given key matches any name in
// SensitiveKeys. The comparison is case-insensitive and ignores `_` / `-`.
func IsSensitive(key string) bool {
	if key == "" {
		return false
	}
	norm := normalize(key)
	for _, sk := range SensitiveKeys {
		if normalize(sk) == norm {
			return true
		}
	}
	return false
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// maskFiller is the rune used to build masks. Exposed as a constant so
// tests can rely on a stable value.
const maskFiller = "*"

// shortMaskThreshold is the cutoff below which MaskString returns a fully
// masked string. Above it, MaskString shows two head + two tail
// characters with a fixed three-character ellipsis in the middle.
const shortMaskThreshold = 6

// MaskString returns a redacted form of s suitable for logs.
//
// Rules:
//   - Empty input → "" (we don't want to add visible characters).
//   - Length ≤ shortMaskThreshold → all characters become '*' (length
//     preserved so that "short password" and "12-char password" still
//     produce visibly different masks at a glance).
//   - Otherwise → first 2 + "***" + last 2 characters (e.g.
//     "secretvalue" → "se***ue"). This preserves enough signal to
//     correlate logs across requests without leaking the middle.
func MaskString(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= shortMaskThreshold {
		return strings.Repeat(maskFiller, len(s))
	}
	return s[:2] + strings.Repeat(maskFiller, 3) + s[len(s)-2:]
}

// MaskAny returns the masked form of v if v is a string; otherwise it
// returns the constant placeholder "***" so we never leak structured
// secrets (numbers used as PINs, []byte tokens, etc.).
func MaskAny(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		return MaskString(t)
	case []byte:
		return MaskString(string(t))
	default:
		return "***"
	}
}

// RedactMap returns a new map where every value whose key matches
// SensitiveKeys (case-insensitive) is replaced via MaskAny. Nested maps
// and slices are walked recursively. The input is NOT mutated.
//
// The function is defensive against cycles introduced by callers: it
// follows the project rule "never mutate inputs" and copies as it goes,
// so a caller-introduced cycle would be visible as repeated traversal
// but not infinite recursion in practice (Go map literals cannot self-
// reference). Callers should not pass cyclic structures.
func RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if IsSensitive(k) {
			out[k] = MaskAny(v)
			continue
		}
		out[k] = redactValue(v)
	}
	return out
}

func redactValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return RedactMap(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = redactValue(item)
		}
		return out
	default:
		return v
	}
}

// RedactJSON decodes b as JSON, walks the structure to mask sensitive
// keys, and re-encodes. If b is not valid JSON the input is returned
// unchanged (the caller is presumably writing arbitrary bytes; we should
// not corrupt them).
func RedactJSON(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return b
	}
	cleaned := redactValue(v)
	out, err := json.Marshal(cleaned)
	if err != nil {
		return b
	}
	return out
}

// String is a drop-in replacement for zap.String that masks the value if
// the field is named after a sensitive key. Use it like:
//
//	logger.Info("login attempt",
//	    zap.String("user", user),
//	    redact.String("password", pwd),
//	)
//
// Unlike a zapcore.Encoder hook, this approach is explicit and keeps the
// existing logger plumbing untouched.
func String(key, value string) zap.Field {
	if IsSensitive(key) {
		return zap.String(key, MaskString(value))
	}
	return zap.String(key, value)
}

// Stringp is the *string variant of String.
func Stringp(key string, value *string) zap.Field {
	if value == nil {
		return zap.Stringp(key, nil)
	}
	if IsSensitive(key) {
		masked := MaskString(*value)
		return zap.String(key, masked)
	}
	return zap.Stringp(key, value)
}

// Any is a drop-in replacement for zap.Any. If the field is sensitive we
// substitute MaskAny(value); otherwise we delegate to zap.Any so structured
// types are still encoded faithfully.
func Any(key string, value any) zap.Field {
	if IsSensitive(key) {
		return zap.Any(key, MaskAny(value))
	}
	return zap.Any(key, value)
}

// FieldEncoder wraps a zapcore.ObjectEncoder so that any sensitive key
// added through it is masked. It is provided for callers that build
// zap.Object values from external structs and cannot easily switch every
// AddString site to redact.String.
//
// Usage:
//
//	type loginPayload struct {
//	    User     string
//	    Password string
//	}
//	func (p loginPayload) MarshalLogObject(enc zapcore.ObjectEncoder) error {
//	    re := redact.NewFieldEncoder(enc)
//	    re.AddString("user", p.User)
//	    re.AddString("password", p.Password) // masked automatically
//	    return nil
//	}
type FieldEncoder struct {
	zapcore.ObjectEncoder
}

// NewFieldEncoder wraps enc with sensitive-field masking.
func NewFieldEncoder(enc zapcore.ObjectEncoder) *FieldEncoder {
	return &FieldEncoder{ObjectEncoder: enc}
}

// AddString masks the value if key is sensitive, then forwards to the
// underlying encoder.
func (e *FieldEncoder) AddString(key, value string) {
	if IsSensitive(key) {
		e.ObjectEncoder.AddString(key, MaskString(value))
		return
	}
	e.ObjectEncoder.AddString(key, value)
}

// AddByteString masks the value if key is sensitive.
func (e *FieldEncoder) AddByteString(key string, value []byte) {
	if IsSensitive(key) {
		e.ObjectEncoder.AddString(key, MaskString(string(value)))
		return
	}
	e.ObjectEncoder.AddByteString(key, value)
}

// AddReflected masks the value if key is sensitive. For non-sensitive
// keys it forwards to the underlying encoder, preserving existing error
// handling.
func (e *FieldEncoder) AddReflected(key string, value any) error {
	if IsSensitive(key) {
		return e.ObjectEncoder.AddReflected(key, MaskAny(value))
	}
	return e.ObjectEncoder.AddReflected(key, value)
}
