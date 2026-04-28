package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestRedactString_MasksSensitive(t *testing.T) {
	f := RedactString("password", "supersecret")
	assert.Equal(t, "password", f.Key)
	assert.Equal(t, zapcore.StringType, f.Type)
	assert.Equal(t, "su***et", f.String)
}

func TestRedactString_PassesThroughNonSensitive(t *testing.T) {
	f := RedactString("user", "alice")
	assert.Equal(t, "user", f.Key)
	assert.Equal(t, "alice", f.String)
}

func TestRedactStringp_NilSafe(t *testing.T) {
	f := RedactStringp("password", nil)
	assert.Equal(t, "password", f.Key)
}

func TestRedactStringp_MasksSensitive(t *testing.T) {
	v := "topsecret"
	f := RedactStringp("token", &v)
	assert.Equal(t, "token", f.Key)
	assert.Equal(t, "to***et", f.String)
}

func TestRedactAny_MasksSensitive(t *testing.T) {
	f := RedactAny("password", "supersecret")
	assert.Equal(t, "password", f.Key)
	assert.Equal(t, zapcore.StringType, f.Type)
	assert.Equal(t, "su***et", f.String)
}

func TestRedactAny_PreservesNonSensitiveTypes(t *testing.T) {
	f := RedactAny("count", 42)
	assert.Equal(t, "count", f.Key)
	assert.Equal(t, zapcore.Int64Type, f.Type)
	assert.Equal(t, int64(42), f.Integer)
}
