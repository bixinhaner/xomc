package definition

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// TestResolveSources_SingleDir 验证单目录扫描:仅扫 alarm-definitions/,跳过 sidecar/隐藏/非 xml,
// LoadedFrom 带 alarm-definitions/ 前缀,按 basename 字典序排序;不再合并 custom 目录。
func TestResolveSources_SingleDir(t *testing.T) {
	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, BuiltinDirSubdir)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	write := func(name string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(`<alarmModel/>`), 0o644))
	}
	write("ENB.xml")
	write("GNB.xml")
	write("MY.xml")
	write("MY.xml" + CustomMarkerSuffix) // sidecar 应跳过
	write(".hidden.xml")                 // 隐藏文件应跳过
	write("readme.txt")                  // 非 xml 应跳过

	// 旧 custom 目录即便存在也不应被合并扫描。
	customDir := filepath.Join(baseDir, "alarm-definitions-custom")
	require.NoError(t, os.MkdirAll(customDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "SHOULD_IGNORE.xml"), []byte(`<alarmModel/>`), 0o644))

	l := NewLoader(nil, appconfig.AlarmDefinitionLoaderConfig{}, baseDir, nil)
	sources, err := l.resolveSources()
	require.NoError(t, err)

	got := make([]string, 0, len(sources))
	for _, s := range sources {
		got = append(got, s.LoadedFrom)
	}
	assert.Equal(t, []string{
		"alarm-definitions/ENB.xml",
		"alarm-definitions/GNB.xml",
		"alarm-definitions/MY.xml",
	}, got)
}

// TestResolveSources_MissingDir 目录不存在 → 静默返回空,不报错。
func TestResolveSources_MissingDir(t *testing.T) {
	baseDir := t.TempDir()
	l := NewLoader(nil, appconfig.AlarmDefinitionLoaderConfig{}, baseDir, nil)
	sources, err := l.resolveSources()
	require.NoError(t, err)
	assert.Empty(t, sources)
}

func TestValidateAlarmIdentifiers(t *testing.T) {
	tests := []struct {
		name    string
		alarms  []xmlAlarm
		seen    map[string]string
		wantErr string
	}{
		{
			name:   "accepts unique identifiers",
			alarms: []xmlAlarm{{Identifier: "1001"}, {Identifier: "1002"}},
			seen:   map[string]string{"999": "ENB.xml"},
		},
		{
			name:    "rejects empty library",
			alarms:  []xmlAlarm{},
			seen:    map[string]string{},
			wantErr: "contains no alarm definitions",
		},
		{
			name:    "rejects missing identifier",
			alarms:  []xmlAlarm{{Identifier: ""}},
			seen:    map[string]string{},
			wantErr: "without identifier",
		},
		{
			name:    "rejects duplicate identifier in XML",
			alarms:  []xmlAlarm{{Identifier: "1001"}, {Identifier: "1001"}},
			seen:    map[string]string{},
			wantErr: "duplicated in NEW.xml",
		},
		{
			name:    "rejects identifier owned by another library",
			alarms:  []xmlAlarm{{Identifier: "1001"}},
			seen:    map[string]string{"1001": "ENB.xml"},
			wantErr: "conflicts with ENB.xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAlarmIdentifiers(tt.alarms, tt.seen, "NEW.xml")
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
