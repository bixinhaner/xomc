package definition

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

func TestXMLAlarmSuggestions(t *testing.T) {
	t.Parallel()
	const raw = `<alarmModel neType="ENB"><alarms><alarm identifier="1001" cnSuggestion="检查电源&amp;链路&#10;然后复测" enSuggestion="Check power and link"/></alarms></alarmModel>`
	var doc xmlAlarmModel
	require.NoError(t, xml.Unmarshal([]byte(raw), &doc))
	require.Len(t, doc.Alarms, 1)
	require.Equal(t, "检查电源&链路\n然后复测", doc.Alarms[0].CnSuggestion)
	require.Equal(t, "Check power and link", doc.Alarms[0].EnSuggestion)
}

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
		name     string
		alarms   []xmlAlarm
		seen     map[string]seenAlarmIdentifier
		filename string
		neType   string
		wantErr  string
	}{
		{
			name:   "accepts unique identifiers",
			alarms: []xmlAlarm{{Identifier: "1001"}, {Identifier: "1002"}},
			seen: map[string]seenAlarmIdentifier{
				"999": {Filename: "ENB.xml", NeType: "ENB", Alarm: xmlAlarm{Identifier: "999"}},
			},
		},
		{
			name:    "rejects empty library",
			alarms:  []xmlAlarm{},
			seen:    map[string]seenAlarmIdentifier{},
			wantErr: "contains no alarm definitions",
		},
		{
			name:    "rejects missing identifier",
			alarms:  []xmlAlarm{{Identifier: ""}},
			seen:    map[string]seenAlarmIdentifier{},
			wantErr: "without identifier",
		},
		{
			name:    "rejects duplicate identifier in XML",
			alarms:  []xmlAlarm{{Identifier: "1001"}, {Identifier: "1001"}},
			seen:    map[string]seenAlarmIdentifier{},
			wantErr: "duplicated in NEW.xml",
		},
		{
			name:   "accepts unchanged GSM definitions retained in legacy ENB",
			alarms: []xmlAlarm{{Identifier: "60001"}},
			seen: map[string]seenAlarmIdentifier{
				"60001": {Filename: "ENB.xml", NeType: "ENB", Alarm: xmlAlarm{Identifier: "60001"}},
			},
			filename: "GSM.xml",
		},
		{
			name:   "rejects changed GSM definition retained in legacy ENB",
			alarms: []xmlAlarm{{Identifier: "60001", CnName: "新版"}},
			seen: map[string]seenAlarmIdentifier{
				"60001": {Filename: "ENB.xml", NeType: "ENB", Alarm: xmlAlarm{Identifier: "60001", CnName: "现网定制"}},
			},
			filename: "GSM.xml",
			wantErr:  "conflicts with ENB.xml",
		},
		{
			name:   "rejects legacy ENB file with customized ne type",
			alarms: []xmlAlarm{{Identifier: "60001"}},
			seen: map[string]seenAlarmIdentifier{
				"60001": {Filename: "ENB.xml", NeType: "CUSTOM", Alarm: xmlAlarm{Identifier: "60001"}},
			},
			filename: "GSM.xml",
			wantErr:  "conflicts with ENB.xml",
		},
		{
			name:   "rejects lowercase legacy ENB ne type",
			alarms: []xmlAlarm{{Identifier: "60001"}},
			seen: map[string]seenAlarmIdentifier{
				"60001": {Filename: "ENB.xml", NeType: "enb", Alarm: xmlAlarm{Identifier: "60001"}},
			},
			filename: "GSM.xml",
			wantErr:  "conflicts with ENB.xml",
		},
		{
			name:   "rejects lowercase current GSM ne type",
			alarms: []xmlAlarm{{Identifier: "60001"}},
			seen: map[string]seenAlarmIdentifier{
				"60001": {Filename: "ENB.xml", NeType: "ENB", Alarm: xmlAlarm{Identifier: "60001"}},
			},
			filename: "GSM.xml",
			neType:   "gsm",
			wantErr:  "conflicts with ENB.xml",
		},
		{
			name:   "rejects identifier owned by another library",
			alarms: []xmlAlarm{{Identifier: "1001"}},
			seen: map[string]seenAlarmIdentifier{
				"1001": {Filename: "ENB.xml", NeType: "ENB", Alarm: xmlAlarm{Identifier: "1001"}},
			},
			wantErr: "conflicts with ENB.xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.filename
			if filename == "" {
				filename = "NEW.xml"
			}
			neType := tt.neType
			if neType == "" {
				neType = strings.TrimSuffix(filename, filepath.Ext(filename))
			}
			err := validateAlarmIdentifiers(tt.alarms, tt.seen, filename, neType)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
