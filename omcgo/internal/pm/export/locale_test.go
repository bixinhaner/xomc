package export

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestWithExportLocale_PreservesParamsAndAddsEnglish(t *testing.T) {
	raw, err := withExportLocale([]byte(`{"task_id":"11111111-1111-1111-1111-111111111111"}`), appcontext.LocaleEN)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", got["task_id"])
	assert.Equal(t, "en-US", got["locale"])
}

func TestWithExportLocale_EmptyParamsBecomesObject(t *testing.T) {
	raw, err := withExportLocale(nil, appcontext.LocaleZH)
	require.NoError(t, err)
	assert.JSONEq(t, `{"locale":"zh-CN"}`, string(raw))
}

func TestWithExportLocale_RejectsInvalidParams(t *testing.T) {
	_, err := withExportLocale([]byte(`not-json`), appcontext.LocaleEN)
	require.Error(t, err)
}

func TestWithExportLocale_RejectsNullParams(t *testing.T) {
	_, err := withExportLocale([]byte(`null`), appcontext.LocaleEN)
	require.Error(t, err)
}

func TestExportLocale(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want appcontext.Locale
	}{
		{name: "english", raw: []byte(`{"locale":"en-US"}`), want: appcontext.LocaleEN},
		{name: "chinese", raw: []byte(`{"locale":"zh-CN"}`), want: appcontext.LocaleZH},
		{name: "legacy missing", raw: []byte(`{}`), want: appcontext.LocaleZH},
		{name: "invalid locale", raw: []byte(`{"locale":"unknown"}`), want: appcontext.LocaleZH},
		{name: "invalid json", raw: []byte(`not-json`), want: appcontext.LocaleZH},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, exportLocale(tt.raw))
		})
	}
}
