package specparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const specFixture = `
# Title

### R-2.4 命令中文名权威表

| # | 章节 | group_code | 命令中文名 |
|--:|------|-----------|------------|
| 1 | SA | ` + "`Device.DeviceInfo.*`" + ` | 设备基本信息 |
| 2 | SD | ` + "`Device.FaultMgmt.SupportedAlarm.{i}.*`" + ` | 支持告警类型 |
| 3 | SF | ` + "`Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*`" + ` | X2 接口 IP 映射 |

### R-3.2 非可创建对象清单

- ` + "`Device.X.NoCreatable.{i}.*`" + `

## SA - DeviceInfo

#### 命令: Device.DeviceInfo.* 📖📝

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | ` + "`IGD.DeviceInfo.UserLabel`" + ` | ` + "`Device.DeviceInfo.UserLabel`" + ` | UserLabel | 用户友好名 | 📝 RW | string |
| 2 | ` + "`IGD.DeviceInfo.UpTime`" + ` | ` + "`Device.DeviceInfo.UpTime`" + ` | UpTime | 运行时间 | 📖 R | unsignedInt[1:5] |
| 3 | ` + "`IGD.DeviceInfo.Stage`" + ` | ` + "`Device.DeviceInfo.Stage`" + ` | Stage | 阶段 | 📖 R | unsignedInt[1, 5, 10, 20] |

## SD - FaultMgmt

#### 命令: Device.FaultMgmt.SupportedAlarm.{i}.* 📖📝

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | ` + "`IGD.X`" + ` | ` + "`Device.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism`" + ` | ReportingMechanism | 上报机制 | 📝 RW | string(32) |
`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestParseMarkdown_R24Table_Happy(t *testing.T) {
	cat, err := ParseMarkdown(writeFixture(t, specFixture), "test-v1")
	require.NoError(t, err)
	require.Len(t, cat.Groups, 3, "应解析 §R-2.4 三行")

	assert.Equal(t, "SA", cat.Groups[0].Chapter)
	assert.Equal(t, "Device.DeviceInfo.*", cat.Groups[0].GroupCode)
	assert.Equal(t, "设备基本信息", cat.Groups[0].CommandZhName)
	assert.False(t, cat.Groups[0].HasInstance)

	assert.Equal(t, "支持告警类型", cat.Groups[1].CommandZhName)
	assert.True(t, cat.Groups[1].HasInstance, "含 {i} 应 HasInstance=true")
}

func TestParseMarkdown_R32Blacklist(t *testing.T) {
	cat, err := ParseMarkdown(writeFixture(t, specFixture), "test-v1")
	require.NoError(t, err)
	assert.Contains(t, cat.Blacklist, "Device.X.NoCreatable.{i}.*")
}

func TestParseMarkdown_PathTableParsing(t *testing.T) {
	cat, err := ParseMarkdown(writeFixture(t, specFixture), "test-v1")
	require.NoError(t, err)
	g := cat.Groups[0]
	require.Len(t, g.Paths, 3)

	assert.Equal(t, "Device.DeviceInfo.UserLabel", g.Paths[0].StandardPath)
	assert.Equal(t, AccessReadWrite, g.Paths[0].Access)
	assert.Equal(t, "string", g.Paths[0].DataType)

	assert.Equal(t, "unsignedInt", g.Paths[1].DataType)
	require.NotNil(t, g.Paths[1].MinValue)
	require.NotNil(t, g.Paths[1].MaxValue)
	assert.Equal(t, int64(1), *g.Paths[1].MinValue)
	assert.Equal(t, int64(5), *g.Paths[1].MaxValue)
}

func TestParseMarkdown_TypeEnumBracketStripped(t *testing.T) {
	// regression: spec 中 "unsignedInt[1, 5, 10, 20]" 枚举形式不应让 DataType 超出 VARCHAR(16)
	cat, err := ParseMarkdown(writeFixture(t, specFixture), "test-v1")
	require.NoError(t, err)
	g := cat.Groups[0]
	require.Len(t, g.Paths, 3)
	dt := g.Paths[2].DataType
	assert.LessOrEqual(t, len(dt), 16, "DataType 不应超过 VARCHAR(16)")
	assert.Equal(t, "unsignedInt", dt)
}

func TestParseMarkdown_DuplicateZhName(t *testing.T) {
	dup := strings.Replace(specFixture, "X2 接口 IP 映射", "设备基本信息", 1)
	_, err := ParseMarkdown(writeFixture(t, dup), "test-v1")
	require.Error(t, err)
	var dupErr *ErrDuplicateZhName
	assert.ErrorAs(t, err, &dupErr)
}

func TestParseMarkdown_StringLengthParsing(t *testing.T) {
	cat, err := ParseMarkdown(writeFixture(t, specFixture), "test-v1")
	require.NoError(t, err)
	g := cat.Groups[1] // SD - FaultMgmt
	require.Len(t, g.Paths, 1)
	require.NotNil(t, g.Paths[0].MaxLength)
	assert.Equal(t, 32, *g.Paths[0].MaxLength)
}
