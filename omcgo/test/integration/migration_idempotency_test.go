package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// 本测试归属 issue #24（seed baseline 幂等性）。它守护两条不变量：
//   1. seed/000001_init_seed.sql 的全部 public 部署数据 INSERT 都带
//      `ON CONFLICT DO NOTHING`（静态检查，无需数据库，CI 永远运行）；
//   2. 把这些 public INSERT 在已建好 schema 的库上重复前向应用，第二次不报错、
//      行数稳定（DB 检查，未设 OMCGO_TEST_DB_DSN 时 t.Skip）。
//
// _timescaledb_catalog.* 的 INSERT 故意不带 ON CONFLICT（靠文件头尾的
// timescaledb_pre/post_restore 括号 + DISABLE/ENABLE TRIGGER 还原约定保证一致性），
// 因此既不静态断言它们带 ON CONFLICT，也不在 DB 检查里重灌它们。

// seedFilePath 返回 consolidated seed baseline 的路径（相对测试包目录）。
func seedFilePath() string {
	return filepath.Join("..", "..", "migrations", "seed", "000001_init_seed.sql")
}

// publicInsert 描述从 seed 文件解析出的一条 INSERT INTO public.* 语句。
type publicInsert struct {
	table     string // 不含 schema 前缀的表名，如 "users"
	sql       string // 完整 SQL（含结尾 ON CONFLICT DO NOTHING;）
	hasOnConf bool   // 是否带 ON CONFLICT DO NOTHING
	startLine int    // 起始行号（1-based，便于报错定位）
}

var (
	insertStartRe = regexp.MustCompile(`^\s*INSERT INTO public\.(\w+)\s`)
	// onConflictEndRe 匹配已加固语句的终止行：独立成行的 `ON CONFLICT ... DO NOTHING;`。
	onConflictEndRe = regexp.MustCompile(`(?i)^\s*ON CONFLICT\b.*\bDO NOTHING\s*;\s*$`)
	// rowEndRe 匹配未加固语句的 pg_dump 行末 `);`（最后一行数据行直接收尾的情况）。
	rowEndRe = regexp.MustCompile(`\);\s*$`)
)

// parsePublicInserts 扫描 seed 文件，提取所有 INSERT INTO public.* 语句。
// 每条语句从 `INSERT INTO public.X` 行起；终止行为以下两者之一（取先命中者）：
//   - 独立成行的 `ON CONFLICT ... DO NOTHING;`（已加固语句，幂等）；
//   - pg_dump 的行末 `);`（未加固语句的最后一行数据行）。
//
// 末行数据行以 `)` 结尾（无分号）后接 ON CONFLICT 终止行的形态由前者捕获，
// 故先判 onConflictEndRe、再判 rowEndRe。
func parsePublicInserts(t *testing.T) []publicInsert {
	t.Helper()

	raw, err := os.ReadFile(seedFilePath())
	if err != nil {
		t.Fatalf("read seed file %s: %v", seedFilePath(), err)
	}
	lines := strings.Split(string(raw), "\n")

	var inserts []publicInsert
	for i := 0; i < len(lines); i++ {
		m := insertStartRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		start := i

		end := i
		hasOnConf := false
		for end < len(lines) {
			if onConflictEndRe.MatchString(lines[end]) {
				hasOnConf = true
				break
			}
			// 未加固语句：数据行以 `);` 直接收尾，且其后不是 ON CONFLICT 行。
			if rowEndRe.MatchString(lines[end]) {
				next := end + 1
				if next >= len(lines) || !onConflictEndRe.MatchString(lines[next]) {
					break
				}
			}
			end++
		}

		inserts = append(inserts, publicInsert{
			table:     m[1],
			sql:       strings.Join(lines[start:end+1], "\n"),
			hasOnConf: hasOnConf,
			startLine: start + 1,
		})
		i = end // 跳过已消费的行
	}
	return inserts
}

// TestSeedBaselineHasOnConflict 静态守护：seed baseline 的每条 public 部署 INSERT
// 都带 ON CONFLICT DO NOTHING。无需数据库，CI 中始终执行。
func TestSeedBaselineHasOnConflict(t *testing.T) {
	inserts := parsePublicInserts(t)
	if len(inserts) == 0 {
		t.Fatalf("解析 %s 未发现任何 INSERT INTO public.*，解析逻辑或文件可能已漂移", seedFilePath())
	}

	for _, ins := range inserts {
		if !ins.hasOnConf {
			t.Errorf("public.%s 的 INSERT（起始行 %d）缺少 ON CONFLICT DO NOTHING——"+
				"会破坏 baseline 全新库重复前向应用的幂等性", ins.table, ins.startLine)
		}
	}

	t.Logf("校验通过：%d 条 INSERT INTO public.* 均带 ON CONFLICT DO NOTHING", len(inserts))
}

// TestSeedBaselineIdempotent DB 守护：在已建好 schema 的库上把每条 public INSERT
// 重复前向应用，断言第二次不报错且行数稳定。未设 OMCGO_TEST_DB_DSN 时跳过。
//
// 前置条件：OMCGO_TEST_DB_DSN 指向一个已应用 000001_init_schema.sql 的库
// （与既有集成测试一致，schema 由 scripts/integration_test.sh 预先迁好）。
func TestSeedBaselineIdempotent(t *testing.T) {
	pool := SetupTestDB(t) // OMCGO_TEST_DB_DSN 未设则 t.Skip

	inserts := parsePublicInserts(t)
	if len(inserts) == 0 {
		t.Fatalf("解析 %s 未发现任何 INSERT INTO public.*", seedFilePath())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, ins := range inserts {
		ins := ins
		t.Run(ins.table, func(t *testing.T) {
			if !ins.hasOnConf {
				t.Fatalf("public.%s 缺少 ON CONFLICT DO NOTHING，重复应用必然失败", ins.table)
			}

			countSQL := fmt.Sprintf("SELECT count(*) FROM public.%s", ins.table)

			// 第一次应用：把 baseline 数据灌入（已存在则因 ON CONFLICT 跳过）。
			if _, err := pool.Exec(ctx, ins.sql); err != nil {
				t.Fatalf("public.%s 首次应用失败: %v", ins.table, err)
			}
			var before int64
			if err := pool.QueryRow(ctx, countSQL).Scan(&before); err != nil {
				t.Fatalf("public.%s 计数失败: %v", ins.table, err)
			}

			// 第二次应用：核心断言——重复前向应用不报错。
			if _, err := pool.Exec(ctx, ins.sql); err != nil {
				t.Fatalf("public.%s 重复应用报错（幂等性破坏）: %v", ins.table, err)
			}
			var after int64
			if err := pool.QueryRow(ctx, countSQL).Scan(&after); err != nil {
				t.Fatalf("public.%s 重复应用后计数失败: %v", ins.table, err)
			}

			if before != after {
				t.Errorf("public.%s 行数在重复应用后发生变化: %d → %d（应稳定）", ins.table, before, after)
			}
		})
	}
}
