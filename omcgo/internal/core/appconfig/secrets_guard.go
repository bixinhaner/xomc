package appconfig

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
)

// knownInsecureSecrets 收录绝不允许出现在生产环境的默认/示例/已泄露凭证。
// 这些值随 config.*.yaml / docker-compose.yml 进入 git 历史，等同公开，命中任一
// 即视为"未轮换"。值为人类可读的说明，用于拒绝启动时给出明确原因。
var knownInsecureSecrets = map[string]string{
	"omcgo123":   "PostgreSQL 默认密码",
	"minioadmin": "MinIO 默认 access/secret key",
	"dps":        "TR-069 STUN/ConnReq 默认共享密钥",
	"change-me-in-production-minimum-32-characters!!":        "JWT 占位密钥（dev 默认）",
	"omcgo-prod-jwt-secret-key-minimum-32-characters!!":      "JWT 硬编码密钥（prod 模板）",
	"8f7a9b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6": "JWT 硬编码密钥（docker-compose）",
}

// placeholderMarker 是 config.prod.yaml 中故意留下的占位符前缀。生产部署必须经
// 环境变量 / 密钥管理（Vault / K8s Secret）注入真实值覆盖；若启动时字段里仍含该
// 标记，说明凭证注入缺失。
const placeholderMarker = "REPLACE_ME"

// IsProductionEnv 当 OMCGO_ENV / GIN_MODE 指示运行在生产环境时返回 true。
// 判定与 cmd/app 的 validateJWTSecret 保持一致：空 OMCGO_ENV 视为 dev（与
// deployments/docker/entrypoint.sh 默认一致），dev/test/development 及
// GIN_MODE debug/test 均视为非生产。
func IsProductionEnv() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("OMCGO_ENV")))
	ginMode := strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE")))
	switch {
	case env == "" || env == "dev" || env == "development" || env == "test":
		return false
	case ginMode == "debug" || ginMode == "test":
		return false
	default:
		return true
	}
}

// SecretCheck 描述一个待校验的敏感配置项。Value 为原始字段值；当 IsDSN 为 true 时，
// 先从 DSN（如 postgres://user:pass@host/db）解析出密码部分再校验。
type SecretCheck struct {
	Field string
	Value string
	IsDSN bool
}

// GuardProductionSecrets 在生产环境下校验给定敏感项均已替换为真实凭证。
// dev/test 环境直接放行（返回 nil）。命中默认/示例/已泄露凭证或仍含 REPLACE_ME
// 占位符时返回汇总错误，调用方据此 Fatal 拒绝启动——把 validateJWTSecret 的
// "检测到默认值即拒启"思路扩展到 PG / MinIO / 共享密钥 / JWT 全量凭证。
//
// 设计上 guard 只对"明确不安全"的值报错（已知默认值 + REPLACE_ME 占位），不对
// 空值报错（空值交由各字段既有校验处理），以免误伤合法的非密码认证场景。
func GuardProductionSecrets(checks ...SecretCheck) error {
	if !IsProductionEnv() {
		return nil
	}
	var problems []string
	for _, c := range checks {
		v := c.Value
		if c.IsDSN {
			v = dsnPassword(c.Value)
		}
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if reason, bad := knownInsecureSecrets[v]; bad {
			problems = append(problems, fmt.Sprintf("%s = %s", c.Field, reason))
			continue
		}
		if strings.Contains(v, placeholderMarker) {
			problems = append(problems, fmt.Sprintf("%s 仍为 %s 占位符（未注入真实凭证）", c.Field, placeholderMarker))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf(
		"生产环境检测到 %d 项不安全凭证，拒绝启动（请经环境变量/密钥管理注入真实值并轮换已泄露密钥）:\n  - %s",
		len(problems), strings.Join(problems, "\n  - "))
}

// dsnPassword 从 URL 形态的 DSN 中解析出密码部分；解析失败或无密码时返回空串。
// 仅用于 guard 校验，不改变 DSN 本身的使用。
func dsnPassword(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return ""
	}
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return ""
	}
	pw, _ := u.User.Password()
	return pw
}
