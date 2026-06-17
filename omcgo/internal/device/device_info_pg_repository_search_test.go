package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceListSearchFields(t *testing.T) {
	assert.Equal(t, []string{
		"d.serial_number",
		"d.site_name",
		"d.manufacturer",
		"d.model_name",
		"di.device_name",
		"di.address",
		"host(d.ip_address)",
		"di.mac",
		"di.pci",
	}, deviceListSearchFields())
}