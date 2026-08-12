package transfercfg

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

const HTTPSCapabilityParameterPath = "Device.Https.HttpsEnable"

type HTTPSCapabilityStatus string

const (
	HTTPSCapabilityNotRead    HTTPSCapabilityStatus = "not_read"
	HTTPSCapabilityEnabled    HTTPSCapabilityStatus = "enabled"
	HTTPSCapabilityDisabled   HTTPSCapabilityStatus = "disabled"
	HTTPSCapabilityUnknown    HTTPSCapabilityStatus = "unknown"
	HTTPSCapabilityReadError  HTTPSCapabilityStatus = "read_error"
	HTTPSCapabilityOtherValue HTTPSCapabilityStatus = "other_value"
)

type DeviceParameterByPathReader interface {
	GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
}

type HTTPSCapabilityReader interface {
	ReadHTTPSCapability(ctx context.Context, deviceID uuid.UUID) HTTPSCapabilityStatus
}

type DeviceParameterHTTPSCapabilityReader struct {
	repository DeviceParameterByPathReader
}

func NewDeviceParameterHTTPSCapabilityReader(repository DeviceParameterByPathReader) *DeviceParameterHTTPSCapabilityReader {
	return &DeviceParameterHTTPSCapabilityReader{repository: repository}
}

func (r *DeviceParameterHTTPSCapabilityReader) ReadHTTPSCapability(
	ctx context.Context,
	deviceID uuid.UUID,
) HTTPSCapabilityStatus {
	if r == nil || r.repository == nil {
		return HTTPSCapabilityUnknown
	}

	parameter, err := r.repository.GetByPath(ctx, deviceID, HTTPSCapabilityParameterPath)
	if err != nil {
		return HTTPSCapabilityReadError
	}
	if parameter == nil {
		return HTTPSCapabilityUnknown
	}

	value := strings.TrimSpace(parameter.ParameterValue)
	switch value {
	case "true":
		return HTTPSCapabilityEnabled
	case "false":
		return HTTPSCapabilityDisabled
	case "":
		return HTTPSCapabilityUnknown
	default:
		return HTTPSCapabilityOtherValue
	}
}
