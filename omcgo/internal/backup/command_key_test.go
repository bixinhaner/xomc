package backup

import "testing"

// 防回归：ParseCommandKey 必须同时识别两种生成路径
// 1. 规范格式 BuildRestoreCommandKey → `{cellCode}_RESTORE_{taskID8}`
// 2. restore_service.go:449 捷径   → `CONFIG_RESTORE_{taskID8}_{sn}`
// 失败时 TransferComplete 会被 "skipped_not_ours" 静默丢弃，整个 restore_task 超时。
func TestParseCommandKey_RestoreVariants(t *testing.T) {
	cases := []struct {
		name      string
		ck        string
		wantKind  CommandKeyKind
		wantPref  string
	}{
		{"规范 hex 在末尾", "site01_RESTORE_140571b0", CommandKeyKindRestore, "140571b0"},
		{"捷径 hex 在中间", "CONFIG_RESTORE_140571b0_1202000314204KT0079", CommandKeyKindRestore, "140571b0"},
		{"规范 backup hex 末尾", "site01_BACKUP_deadbeef", CommandKeyKindBackup, "deadbeef"},
		{"捷径 backup hex 中间", "CONFIG_BACKUP_deadbeef_SN001", CommandKeyKindBackup, "deadbeef"},
		{"非法 hex 长度", "x_RESTORE_zzz", CommandKeyKindUnknown, ""},
		{"完全无关 ck", "mr-open-abc-sn1", CommandKeyKindUnknown, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotKind, gotPref := ParseCommandKey(tc.ck)
			if gotKind != tc.wantKind || gotPref != tc.wantPref {
				t.Fatalf("ParseCommandKey(%q) = (%q, %q), want (%q, %q)",
					tc.ck, gotKind, gotPref, tc.wantKind, tc.wantPref)
			}
		})
	}
}
