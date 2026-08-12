package transfercfg

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDeviceParameterByPathReader struct {
	parameter *model.DeviceParameter
	err       error
	calls     int
	path      string
}

func (s *stubDeviceParameterByPathReader) GetByPath(
	_ context.Context,
	_ uuid.UUID,
	path string,
) (*model.DeviceParameter, error) {
	s.calls++
	s.path = path
	return s.parameter, s.err
}

func TestDeviceParameterHTTPSCapabilityReader_StrictParsing(t *testing.T) {
	tests := []struct {
		name      string
		parameter *model.DeviceParameter
		err       error
		want      HTTPSCapabilityStatus
	}{
		{name: "missing", want: HTTPSCapabilityUnknown},
		{name: "read error", err: errors.New("database unavailable"), want: HTTPSCapabilityReadError},
		{name: "true", parameter: &model.DeviceParameter{ParameterValue: "  true  "}, want: HTTPSCapabilityEnabled},
		{name: "false", parameter: &model.DeviceParameter{ParameterValue: "false"}, want: HTTPSCapabilityDisabled},
		{name: "empty", parameter: &model.DeviceParameter{ParameterValue: "  "}, want: HTTPSCapabilityUnknown},
		{name: "uppercase true is other", parameter: &model.DeviceParameter{ParameterValue: "TRUE"}, want: HTTPSCapabilityOtherValue},
		{name: "title true is other", parameter: &model.DeviceParameter{ParameterValue: "True"}, want: HTTPSCapabilityOtherValue},
		{name: "uppercase false is other", parameter: &model.DeviceParameter{ParameterValue: "FALSE"}, want: HTTPSCapabilityOtherValue},
		{name: "one is not true", parameter: &model.DeviceParameter{ParameterValue: "1"}, want: HTTPSCapabilityOtherValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &stubDeviceParameterByPathReader{parameter: tt.parameter, err: tt.err}
			reader := NewDeviceParameterHTTPSCapabilityReader(repository)

			got := reader.ReadHTTPSCapability(context.Background(), uuid.New())

			assert.Equal(t, tt.want, got)
			require.Equal(t, 1, repository.calls)
			assert.Equal(t, HTTPSCapabilityParameterPath, repository.path)
		})
	}
}
