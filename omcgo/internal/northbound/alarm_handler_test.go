package northbound

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

// 北向告警导出的级别入参解析需同时接受 5 位字典码（31001~31004）与历史 1~4 小编号，
// 否则 OSS 用字典码筛级别会落默认分支返回 0、过滤静默失效返回全部告警（issue #306，
// 与 #219/#272 同根）。返回 canonical 1~4，仓储层 severityAliases 会双向展开匹配库内码。
func TestParseSeverity_AcceptsDictionaryAndLegacyCodes(t *testing.T) {
	cases := []struct {
		in   string
		want model.AlarmSeverity
	}{
		// 5 位字典码（系统落库与对外暴露的正式编码）
		{"31001", model.AlarmCritical},
		{"31002", model.AlarmMajor},
		{"31003", model.AlarmMinor},
		{"31004", model.AlarmWarning},
		// 历史 1~4 小编号（向后兼容）
		{"1", model.AlarmCritical},
		{"2", model.AlarmMajor},
		{"3", model.AlarmMinor},
		{"4", model.AlarmWarning},
		// 非法值落默认分支
		{"", 0},
		{"99", 0},
		{"abc", 0},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, parseSeverity(c.in))
		})
	}
}
