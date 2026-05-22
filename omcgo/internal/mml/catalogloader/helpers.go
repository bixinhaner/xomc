package catalogloader

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// rpcMethodFor 映射 operationType → RPC 方法名。
func rpcMethodFor(op string) string {
	switch op {
	case "LST":
		return "GetParameterValues"
	case "MOD":
		return "SetParameterValues"
	case "ADD":
		return "AddObject"
	case "RMV":
		return "DeleteObject"
	}
	return ""
}

// normalizeForCommandCode 把对象路径模板 → SQL/MML 友好的标识符。
//
//	"Device.DeviceInfo.*"                  → "Device_DeviceInfo"
//	"Device.FaultMgmt.CurrentAlarm.{i}.*"  → "Device_FaultMgmt_CurrentAlarm_i"
func normalizeForCommandCode(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r == '.', r == '{', r == '}', r == '*', r == ' ':
			if len(out) > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
		default:
			out = append(out, r)
		}
	}
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	return string(out)
}

// ensureParamVersion 确保 catalog 对应的 mml_param_versions 行存在。
// FK：mml_command_groups.param_version → mml_param_versions.version_code。
// 派生规则与 upsertChapterGroupV2 保持一致：version_code = "{carrier}-{tech}-v2.3"。
// 注：specVersion 在文件中是 "cmcc-tdlte-v2.3" 类自描述串，不直接做 version_code。
func ensureParamVersion(ctx context.Context, tx pgx.Tx, carrier, tech, specVersion string) error {
	versionCode := fmt.Sprintf("%s-%s-v2.3", carrier, tech)
	versionName := fmt.Sprintf("%s %s %s", carrier, tech, specVersion)
	_, err := tx.Exec(ctx, `
		INSERT INTO mml_param_versions (
			id, version_code, version_name, description, is_active, source, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, true, 'standard', NOW(), NOW()
		)
		ON CONFLICT (version_code) DO UPDATE SET
			version_name = EXCLUDED.version_name,
			updated_at = NOW();
	`, versionCode, versionName, "Auto-created by mml-catalog loader for spec "+specVersion)
	if err != nil {
		return fmt.Errorf("upsert param_version %s: %w", versionCode, err)
	}
	return nil
}
