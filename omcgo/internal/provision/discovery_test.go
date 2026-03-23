package provision

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_filterMultiInstanceObjects(t *testing.T) {
	tests := []struct {
		name         string
		input        []string
		wantFiltered []string
		wantSkipped  int
	}{
		{
			name: "多实例只保留最小编号",
			input: []string{
				"Device.Services.FAPService.1.",
				"Device.Services.FAPService.2.",
				"Device.Services.FAPService.3.",
			},
			wantFiltered: []string{"Device.Services.FAPService.1."},
			wantSkipped:  2,
		},
		{
			name: "混合多实例和普通子对象",
			input: []string{
				"Device.Services.FAPService.1.",
				"Device.Services.FAPService.2.",
				"Device.ManagementServer.",
			},
			wantFiltered: []string{
				"Device.ManagementServer.",
				"Device.Services.FAPService.1.",
			},
			wantSkipped: 1,
		},
		{
			name: "无多实例不过滤",
			input: []string{
				"Device.ManagementServer.",
				"Device.DeviceInfo.",
			},
			wantFiltered: []string{
				"Device.ManagementServer.",
				"Device.DeviceInfo.",
			},
			wantSkipped: 0,
		},
		{
			name: "不同父路径下各自独立过滤",
			input: []string{
				"Device.Services.FAPService.1.",
				"Device.Services.FAPService.2.",
				"Device.Ethernet.Interface.1.",
				"Device.Ethernet.Interface.2.",
				"Device.Ethernet.Interface.3.",
			},
			wantFiltered: []string{
				"Device.Ethernet.Interface.1.",
				"Device.Services.FAPService.1.",
			},
			wantSkipped: 3,
		},
		{
			name: "实例编号不从1开始",
			input: []string{
				"Device.Services.FAPService.5.",
				"Device.Services.FAPService.10.",
			},
			wantFiltered: []string{"Device.Services.FAPService.5."},
			wantSkipped:  1,
		},
		{
			name: "单个实例不过滤",
			input: []string{
				"Device.Services.FAPService.1.",
			},
			wantFiltered: []string{"Device.Services.FAPService.1."},
			wantSkipped:  0,
		},
		{
			name: "数字和非数字子节点共存",
			input: []string{
				"Device.Services.FAPService.1.",
				"Device.Services.FAPService.2.",
				"Device.Services.FAPService.CellConfig.",
			},
			wantFiltered: []string{
				"Device.Services.FAPService.CellConfig.",
				"Device.Services.FAPService.1.",
			},
			wantSkipped: 1,
		},
		{
			name:         "空列表",
			input:        []string{},
			wantFiltered: []string{},
			wantSkipped:  0,
		},
		{
			name: "顶层对象不受影响",
			input: []string{
				"Device.",
			},
			wantFiltered: []string{"Device."},
			wantSkipped:  0,
		},
		{
			name: "嵌套多实例各层独立",
			input: []string{
				"Device.Services.FAPService.1.CellConfig.1.",
				"Device.Services.FAPService.1.CellConfig.2.",
				"Device.Services.FAPService.1.CellConfig.3.",
			},
			wantFiltered: []string{
				"Device.Services.FAPService.1.CellConfig.1.",
			},
			wantSkipped: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFiltered, gotSkipped := filterMultiInstanceObjects(tt.input)

			// Sort both slices for stable comparison.
			sort.Strings(gotFiltered)
			sort.Strings(tt.wantFiltered)

			assert.Equal(t, tt.wantFiltered, gotFiltered, "filtered paths")
			assert.Equal(t, tt.wantSkipped, gotSkipped, "skipped count")
		})
	}
}

func Test_splitLastSegment(t *testing.T) {
	tests := []struct {
		path        string
		wantParent  string
		wantSegment string
	}{
		{"Device.Services.FAPService.1.", "Device.Services.FAPService", "1"},
		{"Device.ManagementServer.", "Device", "ManagementServer"},
		{"Device.", "", "Device"},
		{"Device.Services.FAPService.CellConfig.", "Device.Services.FAPService", "CellConfig"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			parent, segment := splitLastSegment(tt.path)
			assert.Equal(t, tt.wantParent, parent)
			assert.Equal(t, tt.wantSegment, segment)
		})
	}
}
