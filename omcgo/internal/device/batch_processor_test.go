package device

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// TestPrepareDeviceUpdate_DerivesIPFromConnectionRequestURL
//
// Issue #440 回归保障：周期性 Inform 走 InformHandler.handlePeriodic 的批处理路径
// 时，prepareDeviceUpdate 漏掉「从 ConnectionRequestURL 派生 IPAddress」这一步，
// 导致 TR-069 设备已上线 N 小时仍然 ip_address = "" → 列表/详情 IP 列空白。
//
// 此测试锁定批路径必须与 DeviceService.UpdateFromInform / RegisterFromInform
// 同口径派生 IP，且 udpAddr（NAT/UDP）优先级保持不变。
func TestPrepareDeviceUpdate_DerivesIPFromConnectionRequestURL(t *testing.T) {
	tests := []struct {
		name       string
		params     []tr069.ParameterValueStruct
		initialIP  string
		wantIP     string
		wantNATted bool
	}{
		{
			name: "non-NAT device: IP derived from ConnectionRequestURL host",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.24.224.38:7547"},
			},
			initialIP:  "",
			wantIP:     "172.24.224.38",
			wantNATted: false,
		},
		{
			name: "NAT device: udpAddr overrides ConnectionRequestURL host",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://10.0.0.1:7547"},
				{Name: "Device.ManagementServer.UDPConnectionRequestAddress", Value: "203.0.113.5:9999"},
			},
			initialIP:  "10.0.0.1",
			wantIP:     "203.0.113.5",
			wantNATted: true,
		},
		{
			name: "invalid UDP address keeps ConnectionRequestURL host without NAT",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.17.1.14:7547/AB48F15B575B64B43D3F8830CFCF8277"},
				{Name: "Device.ManagementServer.UDPConnectionRequestAddress", Value: "0.0.0.0"},
				{Name: "Device.ManagementServer.STUNServerPort", Value: "3478"},
			},
			initialIP:  "",
			wantIP:     "172.17.1.14",
			wantNATted: false,
		},
		{
			name: "invalid UDP address ignores cached STUN port",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.17.1.14:7547/AB48F15B575B64B43D3F8830CFCF8277"},
				{Name: "Device.ManagementServer.UDPConnectionRequestAddress", Value: "0.0.0.0"},
			},
			initialIP:  "",
			wantIP:     "172.17.1.14",
			wantNATted: false,
		},
		{
			name: "missing ConnectionRequestURL preserves existing summary",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"},
			},
			initialIP:  "192.168.1.100",
			wantIP:     "192.168.1.100",
			wantNATted: false,
		},
		{
			name: "empty ConnectionRequestURL preserves existing summary",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.ManagementServer.ConnectionRequestURL", Value: ""},
			},
			initialIP:  "192.168.1.101",
			wantIP:     "192.168.1.101",
			wantNATted: false,
		},
		{
			name: "invalid ConnectionRequestURL preserves existing summary",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "://invalid"},
			},
			initialIP:  "192.168.1.102",
			wantIP:     "192.168.1.102",
			wantNATted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &model.Device{
				ID:           uuid.New(),
				SerialNumber: "TEST-SN-001",
				IPAddress:    tt.initialIP,
				Status:       model.DeviceActive,
			}
			inform := &tr069.InformMessage{
				DeviceId: tr069.DeviceId{
					Manufacturer: "BAICELLS",
					OUI:          "48BF74",
					ProductClass: "FAP/mBS31001/SC",
					SerialNumber: "TEST-SN-001",
				},
				Event:         []tr069.EventStruct{{EventCode: "2 PERIODIC"}},
				ParameterList: tt.params,
			}

			_, _ = prepareDeviceUpdate(device, inform)

			assert.Equal(t, tt.wantIP, device.IPAddress, "IPAddress")
			assert.Equal(t, tt.wantNATted, device.NatDetected, "NatDetected")
		})
	}
}
