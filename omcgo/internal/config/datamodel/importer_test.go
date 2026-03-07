package datamodel

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePayload(t *testing.T) {
	tests := []struct {
		name     string
		payload  importPayload
		wantErrs int
	}{
		{
			name: "valid carrier_default payload",
			payload: importPayload{
				Carrier:       "cmcc",
				Technology:    "lte",
				Version:       "V2.3",
				ParameterTree: []byte(`{"Device":{}}`),
			},
			wantErrs: 0,
		},
		{
			name: "valid product payload",
			payload: importPayload{
				Carrier:       "cmcc",
				Technology:    "lte",
				Version:       "V2.3",
				OUI:           "00E0FC",
				ProductClass:  "SmallCell-LTE",
				ParameterTree: []byte(`{"Device":{}}`),
			},
			wantErrs: 0,
		},
		{
			name: "missing carrier",
			payload: importPayload{
				Technology:    "lte",
				Version:       "V2.3",
				ParameterTree: []byte(`{"Device":{}}`),
			},
			wantErrs: 1,
		},
		{
			name: "invalid carrier",
			payload: importPayload{
				Carrier:       "unknown",
				Technology:    "lte",
				Version:       "V2.3",
				ParameterTree: []byte(`{"Device":{}}`),
			},
			wantErrs: 1,
		},
		{
			name: "missing technology",
			payload: importPayload{
				Carrier:       "cmcc",
				Version:       "V2.3",
				ParameterTree: []byte(`{"Device":{}}`),
			},
			wantErrs: 1,
		},
		{
			name: "missing version",
			payload: importPayload{
				Carrier:       "cmcc",
				Technology:    "lte",
				ParameterTree: []byte(`{"Device":{}}`),
			},
			wantErrs: 1,
		},
		{
			name: "missing parameter_tree",
			payload: importPayload{
				Carrier:    "cmcc",
				Technology: "lte",
				Version:    "V2.3",
			},
			wantErrs: 1,
		},
		{
			name: "product_class without oui",
			payload: importPayload{
				Carrier:       "cmcc",
				Technology:    "lte",
				Version:       "V2.3",
				ProductClass:  "SmallCell-LTE",
				ParameterTree: []byte(`{"Device":{}}`),
			},
			wantErrs: 1,
		},
		{
			name:     "all fields missing",
			payload:  importPayload{},
			wantErrs: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validatePayload(&tt.payload)
			assert.Len(t, errs, tt.wantErrs, "errors: %v", errs)
		})
	}
}

func TestDetermineScope(t *testing.T) {
	tests := []struct {
		oui          string
		productClass string
		want         string
	}{
		{"00E0FC", "SmallCell-LTE", "product"},
		{"00E0FC", "", "oui"},
		{"", "", "carrier_default"},
	}

	for _, tt := range tests {
		scope := determineScope(tt.oui, tt.productClass)
		assert.Equal(t, tt.want, string(scope))
	}
}

func TestCountParameters(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		count int
	}{
		{"single leaf", `"value"`, 1},
		{"flat object", `{"a": 1, "b": 2}`, 2},
		{"nested object", `{"a": {"b": 1, "c": 2}, "d": 3}`, 3},
		{"array", `[1, 2, 3]`, 3},
		{"empty object", `{}`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v interface{}
			err := json.Unmarshal([]byte(tt.json), &v)
			assert.NoError(t, err)
			assert.Equal(t, tt.count, countParameters(v))
		})
	}
}

func TestValidateImport(t *testing.T) {
	imp := NewDataModelImporter(nil, nil)

	t.Run("valid payload", func(t *testing.T) {
		payload := `{"carrier":"cmcc","technology":"lte","version":"V2.3","parameter_tree":{"Device":{}}}`
		result, err := imp.ValidateImport(nil, strings.NewReader(payload))
		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "carrier_default", result.Scope)
		assert.Empty(t, result.Errors)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		result, err := imp.ValidateImport(nil, strings.NewReader("not json"))
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("missing required fields", func(t *testing.T) {
		result, err := imp.ValidateImport(nil, strings.NewReader(`{}`))
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})
}
