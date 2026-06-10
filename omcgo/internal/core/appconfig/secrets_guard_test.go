package appconfig

import (
	"strings"
	"testing"
)

func TestIsProductionEnv(t *testing.T) {
	cases := []struct {
		name    string
		env     string
		ginMode string
		want    bool
	}{
		{"empty is dev", "", "", false},
		{"dev", "dev", "", false},
		{"development", "development", "", false},
		{"test", "test", "", false},
		{"gin debug overrides", "prod", "debug", false},
		{"gin test overrides", "prod", "test", false},
		{"prod", "prod", "", true},
		{"production", "production", "", true},
		{"staging treated as prod", "staging", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OMCGO_ENV", tc.env)
			t.Setenv("GIN_MODE", tc.ginMode)
			if got := IsProductionEnv(); got != tc.want {
				t.Fatalf("IsProductionEnv()=%v, want %v", got, tc.want)
			}
		})
	}
}

// 失败路径：生产环境下默认/占位/已泄露凭证必须被拒绝。
func TestGuardProductionSecrets_RejectsInsecure(t *testing.T) {
	t.Setenv("OMCGO_ENV", "prod")
	t.Setenv("GIN_MODE", "")

	cases := []struct {
		name  string
		check SecretCheck
	}{
		{"pg default password in dsn", SecretCheck{Field: "db.dsn", Value: "postgres://omcgo:omcgo123@postgres:5432/omcgo?sslmode=disable", IsDSN: true}},
		{"minio default", SecretCheck{Field: "minio.access_key", Value: "minioadmin"}},
		{"stun default secret", SecretCheck{Field: "stun.shared_secret", Value: "dps"}},
		{"prod hardcoded jwt", SecretCheck{Field: "jwt.secret", Value: "omcgo-prod-jwt-secret-key-minimum-32-characters!!"}},
		{"compose hardcoded jwt", SecretCheck{Field: "jwt.secret", Value: "8f7a9b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6"}},
		{"replace_me placeholder", SecretCheck{Field: "minio.secret_key", Value: "REPLACE_ME_MINIO_SECRET_KEY"}},
		{"replace_me in dsn", SecretCheck{Field: "db.dsn", Value: "postgres://omcgo:REPLACE_ME_DB_PASSWORD@postgres:5432/omcgo", IsDSN: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := GuardProductionSecrets(tc.check)
			if err == nil {
				t.Fatalf("expected guard to reject %q, got nil", tc.check.Value)
			}
		})
	}
}

// 失败路径：多项问题应汇总在同一条错误里全部列出。
func TestGuardProductionSecrets_AggregatesAllProblems(t *testing.T) {
	t.Setenv("OMCGO_ENV", "prod")
	t.Setenv("GIN_MODE", "")

	err := GuardProductionSecrets(
		SecretCheck{Field: "db.dsn", Value: "postgres://omcgo:omcgo123@postgres:5432/omcgo", IsDSN: true},
		SecretCheck{Field: "minio.access_key", Value: "minioadmin"},
		SecretCheck{Field: "stun.shared_secret", Value: "dps"},
	)
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}
	for _, field := range []string{"db.dsn", "minio.access_key", "stun.shared_secret"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("aggregated error missing field %q: %v", field, err)
		}
	}
}

// 成功路径：dev/test 环境一律放行，即便填的是默认凭证（便于本地开发）。
func TestGuardProductionSecrets_SkipsInDev(t *testing.T) {
	t.Setenv("OMCGO_ENV", "dev")
	t.Setenv("GIN_MODE", "")

	err := GuardProductionSecrets(
		SecretCheck{Field: "db.dsn", Value: "postgres://omcgo:omcgo123@postgres:5432/omcgo", IsDSN: true},
		SecretCheck{Field: "minio.access_key", Value: "minioadmin"},
	)
	if err != nil {
		t.Fatalf("dev env must skip guard, got error: %v", err)
	}
}

// 成功路径：生产环境下真实凭证应通过。
func TestGuardProductionSecrets_AcceptsRealSecrets(t *testing.T) {
	t.Setenv("OMCGO_ENV", "prod")
	t.Setenv("GIN_MODE", "")

	err := GuardProductionSecrets(
		SecretCheck{Field: "db.dsn", Value: "postgres://omcgo:S0me-Str0ng-P@ss-9f2c@postgres:5432/omcgo?sslmode=require", IsDSN: true},
		SecretCheck{Field: "minio.access_key", Value: "AKIA7REALKEYxyz"},
		SecretCheck{Field: "minio.secret_key", Value: "z9x8c7v6b5n4m3-real-secret-value-here"},
		SecretCheck{Field: "jwt.secret", Value: "a-genuinely-unique-production-jwt-secret-7d3f9"},
		SecretCheck{Field: "stun.shared_secret", Value: "f3a9c1e7b2d4-unique-stun-secret"},
		SecretCheck{Field: "empty.optional", Value: ""}, // 空值不报错
	)
	if err != nil {
		t.Fatalf("real secrets in prod must pass, got error: %v", err)
	}
}

func TestDSNPassword(t *testing.T) {
	cases := []struct {
		dsn  string
		want string
	}{
		{"postgres://omcgo:omcgo123@postgres:5432/omcgo?sslmode=disable", "omcgo123"},
		{"postgres://user@host/db", ""},   // no password
		{"", ""},                          // empty
		{"not a url at all %%%", ""},      // unparseable → empty
		{"postgres://u:p%40ss@host/db", "p@ss"}, // percent-encoded password
	}
	for _, tc := range cases {
		if got := dsnPassword(tc.dsn); got != tc.want {
			t.Errorf("dsnPassword(%q)=%q, want %q", tc.dsn, got, tc.want)
		}
	}
}
