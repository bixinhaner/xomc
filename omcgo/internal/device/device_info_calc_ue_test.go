package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcUECount(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   int
	}{
		{
			name:   "标准路径有值",
			params: map[string]string{"Device.DeviceInfo.UE_Count": "5"},
			want:   5,
		},
		{
			name: "优先使用标准路径，忽略厂商扩展",
			params: map[string]string{
				"Device.DeviceInfo.UE_Count":                              "3",
				"Device.Services.FAPService.1.X_COM_ConnectedUECount": "7",
			},
			want: 3,
		},
		{
			name: "标准路径为空，降级到厂商扩展",
			params: map[string]string{
				"Device.Services.FAPService.1.X_COM_ConnectedUECount": "2",
			},
			want: 2,
		},
		{
			name:   "两个路径都没有，返回0",
			params: map[string]string{"Device.DeviceInfo.UpTime": "1000"},
			want:   0,
		},
		{
			name:   "参数为空字符串，返回0",
			params: map[string]string{"Device.DeviceInfo.UE_Count": ""},
			want:   0,
		},
		{
			name:   "非数字字符串，返回0",
			params: map[string]string{"Device.DeviceInfo.UE_Count": "N/A"},
			want:   0,
		},
		{
			name:   "负数，返回0",
			params: map[string]string{"Device.DeviceInfo.UE_Count": "-1"},
			want:   0,
		},
		{
			name:   "零值，返回0",
			params: map[string]string{"Device.DeviceInfo.UE_Count": "0"},
			want:   0,
		},
		{
			name:   "带空格的数字",
			params: map[string]string{"Device.DeviceInfo.UE_Count": " 4 "},
			want:   4,
		},
		{
			name:   "空map",
			params: map[string]string{},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcUECount(tt.params)
			assert.Equal(t, tt.want, got)
		})
	}
}
