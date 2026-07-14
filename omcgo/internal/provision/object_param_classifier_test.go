package provision

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── isObjectPath ───────────────────────────────────────────────────────────

func TestIsObjectPath_Cases(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		// scalar — 无尾点的叶子参数
		{"scalar_leaf_deviceinfo", "Device.DeviceInfo.SerialNumber", false},
		{"scalar_leaf_system_mode", "Dev.System.Mode", false},
		{"scalar_no_dot", "NoDot", false},
		{"empty_string", "", false},

		// object — 尾点 "." 的对象前缀（CPE 应枚举实例）
		{"object_wifi_ssid", "Device.WiFi.SSID.", true},
		{"object_lte_cell", "Device.X_VENDOR.LteCell.", true},
		{"object_nested", "Device.Services.FAPService.", true},

		// 边界：单个 "." 也算 object（虽不太可能出现，行为定义清楚）
		{"single_dot", ".", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isObjectPath(c.path))
		})
	}
}

// ── classifyPrefixes 分组顺序稳定 ────────────────────────────────────────

func TestClassifyPrefixes_PreservesOrder(t *testing.T) {
	input := []string{
		"Device.DeviceInfo.SerialNumber", // scalar #0
		"Device.WiFi.SSID.",              // object #0
		"Device.System.Mode",             // scalar #1
		"Device.X_LteCell.",              // object #1
		"Device.DeviceInfo.Manufacturer", // scalar #2
	}
	scalars, objects := classifyPrefixes(input)
	assert.Equal(t, []string{
		"Device.DeviceInfo.SerialNumber",
		"Device.System.Mode",
		"Device.DeviceInfo.Manufacturer",
	}, scalars)
	assert.Equal(t, []string{
		"Device.WiFi.SSID.",
		"Device.X_LteCell.",
	}, objects)
}

func TestClassifyPrefixes_AllScalar(t *testing.T) {
	input := []string{"A.B.C", "A.B.D", "E"}
	scalars, objects := classifyPrefixes(input)
	assert.Equal(t, input, scalars)
	assert.Empty(t, objects)
}

func TestClassifyPrefixes_AllObject(t *testing.T) {
	input := []string{"Device.WiFi.", "Device.LteCell."}
	scalars, objects := classifyPrefixes(input)
	assert.Empty(t, scalars)
	assert.Equal(t, input, objects)
}

func TestClassifyPrefixes_Empty(t *testing.T) {
	scalars, objects := classifyPrefixes(nil)
	assert.Empty(t, scalars)
	assert.Empty(t, objects)
}

func TestIsExpandedInstanceObjectPath_Cases(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"expanded_instance", "DeviceGSM.Bts.1.", true},
		{"expanded_large_instance", "DeviceGSM.Bts.256.", true},
		{"plain_object", "DeviceGSM.Bts.", false},
		{"placeholder_object", "DeviceGSM.Bts.{i}.", false},
		{"scalar_numeric_leaf", "DeviceGSM.Bts.1.Enable", false},
		{"non_numeric_tail", "DeviceGSM.Bts.foo.", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isExpandedInstanceObjectPath(c.path))
		})
	}
}

// ── buildGPVBatches: scalar 批量 + object 自适应分批 ────────────────────────

func TestBuildGPVBatches_MixedScalarAndObject(t *testing.T) {
	// 5 scalar + 3 object, batchSize=2
	// 期望：scalar 拆成 3 批 (2+2+1)；普通 object 单发；实例级 object 合批
	input := []string{
		"s1", "obj1.", "s2", "DeviceGSM.Bts.1.", "s3", "DeviceGSM.Bts.2.", "s4", "s5",
	}
	batches := buildGPVBatches(input, 2)

	want := [][]string{
		{"s1", "s2"},
		{"s3", "s4"},
		{"s5"},
		{"obj1."},
		{"DeviceGSM.Bts.1.", "DeviceGSM.Bts.2."},
	}
	assert.Equal(t, want, batches)
}

func TestBuildGPVBatches_AllScalar_NormalBatching(t *testing.T) {
	input := []string{"s1", "s2", "s3", "s4", "s5"}
	batches := buildGPVBatches(input, 3)
	want := [][]string{
		{"s1", "s2", "s3"},
		{"s4", "s5"},
	}
	assert.Equal(t, want, batches)
}

func TestBuildGPVBatches_AllPlainObject_EachIsOwnBatch(t *testing.T) {
	input := []string{"obj1.", "obj2.", "obj3."}
	batches := buildGPVBatches(input, 50)
	want := [][]string{
		{"obj1."},
		{"obj2."},
		{"obj3."},
	}
	assert.Equal(t, want, batches)
}

func TestBuildGPVBatches_ExpandedObjects_Within5MBBudgetSingleBatch(t *testing.T) {
	input := make([]string, 0, 256)
	for i := 1; i <= 256; i++ {
		input = append(input, "DeviceGSM.Bts."+strconv.Itoa(i)+".")
	}
	batches := buildGPVBatches(input, 50)

	require.Len(t, batches, 1)
	assert.Len(t, batches[0], 256)
	for _, batch := range batches {
		assert.LessOrEqual(t, estimatedGPVNATSPayloadBytes(batch), gpvNATSPayloadBudgetBytes)
		assert.Less(t, estimatedGPVNATSPayloadBytes(batch), natsMaxPayloadBytes)
	}
}

func TestBuildGPVBatches_ExpandedObjects_Over5MBBudgetSplits(t *testing.T) {
	input := make([]string, 0, 400)
	for i := 1; i <= 400; i++ {
		input = append(input, "DeviceGSM.Bts."+strconv.Itoa(i)+".")
	}
	batches := buildGPVBatches(input, 50)

	assert.Greater(t, len(batches), 1)
	for _, batch := range batches {
		assert.LessOrEqual(t, estimatedGPVNATSPayloadBytes(batch), gpvNATSPayloadBudgetBytes)
		assert.Less(t, estimatedGPVNATSPayloadBytes(batch), natsMaxPayloadBytes)
	}
	assert.Len(t, batches[0], maxExpandedObjectPrefixesPerGPV())
}

func TestBuildGPVBatches_ExpandedObjects_FlushBeforePlainObject(t *testing.T) {
	input := []string{
		"DeviceGSM.Bts.1.",
		"DeviceGSM.Bts.2.",
		"DeviceGSM.Msc.",
		"DeviceGSM.Bts.3.",
	}
	batches := buildGPVBatches(input, 50)

	assert.Equal(t, [][]string{
		{"DeviceGSM.Bts.1.", "DeviceGSM.Bts.2."},
		{"DeviceGSM.Msc."},
		{"DeviceGSM.Bts.3."},
	}, batches)
}

func TestBuildGPVBatches_Empty(t *testing.T) {
	batches := buildGPVBatches(nil, 10)
	assert.Nil(t, batches)
}

func TestBuildGPVBatches_ZeroBatchSize_DefaultsTo50(t *testing.T) {
	// 51 scalar，batchSize=0 → 默认 50 → 2 批 (50 + 1)
	input := make([]string, 51)
	for i := range input {
		input[i] = "scalar"
	}
	batches := buildGPVBatches(input, 0)
	assert.Len(t, batches, 2)
	assert.Len(t, batches[0], 50)
	assert.Len(t, batches[1], 1)
}

// ── 真实路径形态回归（防止 basePrefix 输出格式变更后这里失效） ───────────────

func TestBuildGPVBatches_RealisticPathShapes(t *testing.T) {
	// 模拟 BLQ 真机：典型 Path B 同步会拉到的混合形态
	input := []string{
		// scalar
		"Device.DeviceInfo.SerialNumber",
		"Device.DeviceInfo.Manufacturer",
		"Device.DeviceInfo.SoftwareVersion",
		// object 前缀（basePrefix 已截到 "."）
		"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.",
		"Device.WiFi.SSID.",
	}
	batches := buildGPVBatches(input, 10)

	// 3 scalar 一批 + 2 object 单独
	assert.Equal(t, [][]string{
		{
			"Device.DeviceInfo.SerialNumber",
			"Device.DeviceInfo.Manufacturer",
			"Device.DeviceInfo.SoftwareVersion",
		},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."},
		{"Device.WiFi.SSID."},
	}, batches)
}
