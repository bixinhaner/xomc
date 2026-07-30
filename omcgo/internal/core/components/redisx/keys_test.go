package redisx_test

import (
	"strings"
	"testing"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
)

// TestKeyBuilder_Composition 覆盖所有公有构造方法的命名前缀与拼接顺序，
// 目的不是穷举所有值，而是把"命名空间 → prefix"这张映射表钉死，避免有人
// 无意中改了字面量导致线上扫 key 的运维脚本失效。
func TestKeyBuilder_Composition(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		// ACS
		{"ACSSession", redisx.Keys.ACSSession("S1"), "acs:session:id:S1"},
		{"ACSHeartbeat", redisx.Keys.ACSHeartbeat("SN1"), "acs:heartbeat:SN1"},
		{"ACSConnReqPending", redisx.Keys.ACSConnReqPending("SN1"), "acs:connreq:pending:SN1"},
		{"ACSContinuousWake", redisx.Keys.ACSContinuousWake("SN1"), "acs:continuous_wake:SN1"},
		{"ACSSTUN", redisx.Keys.ACSSTUN("SN1"), "acs:stun:SN1"},
		{"ACSTaskQueue", redisx.Keys.ACSTaskQueue("SN1"), "acs:taskq:SN1"},
		{"ACSTaskDetail", redisx.Keys.ACSTaskDetail("T1"), "acs:task:T1"},
		{"ACSCWMP2Task", redisx.Keys.ACSCWMP2Task("H1"), "acs:cwmp2task:H1"},
		{"ACSTaskQueuePrefix", redisx.Keys.ACSTaskQueuePrefix(), "acs:taskq:"},
		{"ACSTaskQueuePattern", redisx.Keys.ACSTaskQueuePattern(), "acs:taskq:*"},
		{"ACSTaskTransitionPending", redisx.Keys.ACSTaskTransitionPending(), "acs:task:transition:pending"},

		// datamodel
		{"DataModelCacheVersion", redisx.Keys.DataModelCacheVersion(), "datamodel:cache_version"},
		{"DataModelPattern", redisx.Keys.DataModelPattern(), "datamodel:*"},
		{"DataModelProduct", redisx.Keys.DataModelProduct("cmcc", "lte", "OUI", "PC"), "datamodel:product:cmcc:lte:OUI:PC"},
		{"DataModelOUI", redisx.Keys.DataModelOUI("cmcc", "lte", "OUI"), "datamodel:oui:cmcc:lte:OUI"},
		{"DataModelDefault", redisx.Keys.DataModelDefault("cmcc", "lte"), "datamodel:default:cmcc:lte"},
		{"DataModelResolve", redisx.Keys.DataModelResolve("cmcc", "lte", "OUI", "PC"), "datamodel:resolve:cmcc:lte:OUI:PC"},

		// alarm / reboot
		{"AlarmActive", redisx.Keys.AlarmActive("SN1"), "alarm:active:SN1"},
		{"RebootAbnormal", redisx.Keys.RebootAbnormal("SN1"), "reboot:abnormal:SN1"},

		// device / provision / upload
		{"DeviceSN", redisx.Keys.DeviceSN("SN1"), "device:sn:SN1"},
		{"ProvisionSyncPlan", redisx.Keys.ProvisionSyncPlan("SN1"), "provision:sync_plan:SN1"},
		{"UploadSession", redisx.Keys.UploadSession("SN1", "CK1"), "upload:session:SN1:CK1"},

		// admin
		{"AuthCaptcha", redisx.Keys.AuthCaptcha("ID1"), "auth:captcha:ID1"},
		{"AuthFailed", redisx.Keys.AuthFailed("alice"), "auth:failed:alice"},
		{"PermVisibleGroups", redisx.Keys.PermVisibleGroups("U1"), "perm:visible_groups:U1"},
		{"CasbinPolicyChannel", redisx.Keys.CasbinPolicyChannel(), "casbin:policy:reload"},

		// sse
		{"SSEPending", redisx.Keys.SSEPending("U1"), "sse:pending:U1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

// TestKeyBuilder_Prefixes 校验 *Prefix 返回值是对应 key 的严格前缀，供
// TrimPrefix 等业务代码放心使用。
func TestKeyBuilder_Prefixes(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		key    string
	}{
		{"ACSSession", redisx.Keys.ACSSessionPrefix(), redisx.Keys.ACSSession("X")},
		{"ACSHeartbeat", redisx.Keys.ACSHeartbeatPrefix(), redisx.Keys.ACSHeartbeat("X")},
		{"ACSSTUN", redisx.Keys.ACSSTUNPrefix(), redisx.Keys.ACSSTUN("X")},
		{"ACSTaskQueue", redisx.Keys.ACSTaskQueuePrefix(), redisx.Keys.ACSTaskQueue("X")},
		{"ACSTaskDetail", redisx.Keys.ACSTaskDetailPrefix(), redisx.Keys.ACSTaskDetail("X")},
		{"ACSCWMP2Task", redisx.Keys.ACSCWMP2TaskPrefix(), redisx.Keys.ACSCWMP2Task("X")},
		{"ProvisionSyncPlan", redisx.Keys.ProvisionSyncPlanPrefix(), redisx.Keys.ProvisionSyncPlan("X")},
		{"AuthCaptcha", redisx.Keys.AuthCaptchaPrefix(), redisx.Keys.AuthCaptcha("X")},
		{"AuthFailed", redisx.Keys.AuthFailedPrefix(), redisx.Keys.AuthFailed("X")},
		{"PermVisibleGroups", redisx.Keys.PermVisibleGroupsPrefix(), redisx.Keys.PermVisibleGroups("X")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.HasPrefix(tc.key, tc.prefix) {
				t.Fatalf("%s: key %q not prefixed by %q", tc.name, tc.key, tc.prefix)
			}
		})
	}
}
