# MML Script TXT Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 MML 脚本库改造成服务端权威校验的 TXT 导入式脚本库，保存不可变计划快照，并复用现有 `mml_tasks -> device_tasks -> Sequencer` 执行链路。

**Architecture:** 新增纯 Go TXT Parser、批量 Validator、Redis 一次性校验会话和事务化 Script Import Service。脚本执行只接受 `script_id + 调度/重试策略`，服务端从 `mml_scripts.plan_items` 复制任务快照；三套皮肤共用 `frontend-core` 契约，但分别实现符合各自组件体系的导入与执行界面。

**Tech Stack:** Go 1.25、Gin、pgx/v5、Squirrel、go-redis/v9、miniredis、PostgreSQL JSONB、React 19、TypeScript strict、React Query v5、Ant Design 5、v2 shadcn/Tailwind、v3 STARFORGE UI、Vitest、Testing Library。

## Global Constraints

- TXT 是脚本内容的唯一来源；内容页只读，内容更新必须重新导入完整 TXT。
- 仅接受 UTF-8 `.txt`，允许 UTF-8 BOM；空行和 `#` 注释忽略。
- 每条有效行必须使用 `命令;SN`，一行一个 SN；同一 SN 顺序由文件行序自动生成。
- 单脚本最多 200 个不同 SN、2000 条有效命令。
- 任意错误阻止保存；警告允许保存但必须展示并在执行时确认。
- 保存接口不接受浏览器提交的 `content`、`commands` 或 `plan_items`。
- 所有脚本统一使用 `execute_mode=device_bound`。
- 同一 SN 严格串行，不同 SN 并行；单行失败后继续该 SN 的下一行。
- 不迁移旧 MML 数据；只清理 `source='mml'` 的设备任务、`mml_tasks` 和 `mml_scripts`。
- 前端 API、Hooks、Types、错误码和分页进入 `frontend-core`；v1、v2、v3 业务能力一致。
- 数据库迁移提交前必须获取最新 `origin/main`、检查冲突和 Goose 编号，并运行 `bash omcgo/scripts/check-migrations.sh --strict`。
- 不修改、不提交用户提供且当前未跟踪的 `docs/design/MMLScript-模板导入校验与执行逻辑总结.md` 和 `docs/design/MMLTemplate.txt`。

## Progress Ledger

**Current checkpoint:** Task 6 complete
**Resume from:** Task 7, Step 1 — write execution snapshot failing tests
**Last verified commit:** `2c8a080f feat(mml): 完善 TXT 脚本导入接口复审问题`

- [x] Task 1 — 数据库契约与 Go 模型
- [x] Task 2 — TXT Parser
- [x] Task 3 — 批量命令/设备校验
- [x] Task 4 — Redis 一次性校验会话
- [x] Task 5 — 事务化脚本导入服务
- [x] Task 6 — 导入 HTTP API 与依赖注入
- [ ] Task 7 — 脚本执行预检与任务快照
- [ ] Task 8 — frontend-core 契约、API 与 Hooks
- [ ] Task 9 — v1 脚本库界面
- [ ] Task 10 — v2 脚本库界面
- [ ] Task 11 — v3 脚本库界面
- [ ] Task 12 — 旧入口收口与安全清理工具
- [ ] Task 13 — 全量验证、部署演练与文档收尾

## Resume Protocol

每次开始或恢复工作时严格执行：

1. 运行 `git status --short && git log -5 --oneline --decorate`，不得覆盖用户未提交内容。
2. 打开本文件 `Progress Ledger`，从 `Resume from` 指定步骤继续。
3. 运行上一已完成任务记录的验证命令；验证不通过时先修复回归，不跳到下一任务。
4. 开始任务前把 `Current checkpoint` 改成 `Task N in progress`、把 `Resume from` 改成当前步骤，并提交：

   ```bash
   git add docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
   git commit -m "chore(plan): 开始 MML TXT 导入任务 N"
   ```

5. 每完成一个红-绿测试循环，更新 `Resume from` 到下一个未完成步骤。
6. 任务验收通过后勾选任务，把验证命令与结果摘要追加到本节下方的 `Evidence Log`，再与该任务代码一起提交。
7. 不在本计划里写百分比；只使用 `not started`、`in progress`、`complete` 和可复现证据。

## Evidence Log

| Task | Commit | Verification |
|---|---|---|
| Design | `d44bacf5` | 设计已确认，实施计划从 Task 1 开始 |
| Task 1 | `e3f605e0..52909cb9` | 规格与质量复审通过；migration strict、PG 契约测试和 `go test ./internal/mml -count=1` 通过 |
| Task 2 | `43edbfa8..82e2c21c` | 规格与质量最终复审通过；聚焦 parser、MML 全包测试和 package build 通过，覆盖 BOM/CR/LF、终止换行 SHA、逐设备顺序、引号/大括号分隔符、200/2000 边界与有界错误累积。 |
| Task 3 | `d8885bbe..2b634644` | 规格与质量复审通过；批量 validator/PG 测试、MML package/build 和全量 Go 测试通过，覆盖 unsignedInt、boolean、可见自定义命令歧义与操作类型不匹配。 |
| Task 4 | `adc3bf6e..4ce47694` | miniredis 会话测试、MML 包全量测试及 app build 通过；覆盖随机 32-byte token、SHA-256 Redis key、15 分钟 TTL、用户绑定、过期、Claim/Release、并发双 Claim、consumed tombstone，以及 Redis Cluster 同槽位。 |
| Task 5 | `6fdfd51f..f23a9f17` | 规格与质量复审通过；服务/仓库聚焦测试、MML 包全量测试和 app build 通过，覆盖错误不发 token、warning 快照、Claim/Release/Finalize、跨 Redis/PG 幂等、替换版本冲突、Finalize 重试恢复、ValidatedAt 快照和 Creator 权限。 |
| Task 6 | `2c8a080f` | 规格与质量复审通过；handler focused tests、MML 包全量测试和 app build 通过，覆盖模板下载、5 路由、multipart 2MiB+64KiB、.txt/缺文件/413、鉴权/403、422 逐行问题、409 重放、严格 JSON、metadata-only 与 updated_at reload、Redis/PG 注入。 |

---

### Task 1: 数据库契约与 Go 模型

**Files:**
- Create: `omcgo/migrations/000015_redesign_mml_script_txt_import.sql`
- Modify: `omcgo/internal/mml/model.go`
- Modify: `omcgo/internal/mml/pg_repository.go`
- Modify: `omcgo/internal/mml/pg_repository_test.go`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `MMLScript.ImportSessionID`, `OriginalFilename`, `ContentSHA256`, `ValidationVersion`, `ValidatedAt`, `PlanItems`, `ValidationSummary`。
- Produces: `MMLTask.ScriptContentSHA256`, `ScriptValidationVersion`。
- Produces: PostgreSQL 列与 `scriptColumns` / `taskColumns` 一一对应。

- [ ] **Step 1: 获取最新主分支并锁定迁移编号**

  Run:

  ```bash
  git fetch origin main
  git rev-list --left-right --count HEAD...origin/main
  git merge-tree --write-tree --messages HEAD origin/main
  find omcgo/migrations -maxdepth 1 -name '*.sql' -print | sort -V | tail -5
  ```

  Expected: 无内容冲突；若 `000015` 已被占用，先把本计划内所有 `000015_redesign_mml_script_txt_import.sql` 替换为最新主分支之后的第一个空闲编号，再开始写迁移。

- [ ] **Step 2: 写迁移契约测试**

  在 `pg_repository_test.go` 增加真实 PG 可用时执行的断言：

  ```go
  func TestPgScriptRepository_TXTImportColumns(t *testing.T) {
      pool := newMMLTestPool(t)
      var count int
      err := pool.QueryRow(context.Background(), `
          SELECT count(*)
            FROM information_schema.columns
           WHERE table_schema='public' AND table_name='mml_scripts'
             AND column_name = ANY($1)`, []string{
          "import_session_id", "original_filename", "content_sha256", "validation_version",
          "validated_at", "plan_items", "validation_summary",
      }).Scan(&count)
      require.NoError(t, err)
      require.Equal(t, 7, count)
  }
  ```

- [ ] **Step 3: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/mml -run TestPgScriptRepository_TXTImportColumns -count=1 -v`
  Expected: FAIL，缺少新列；无数据库时测试按现有 `newMMLTestPool` 约定 SKIP。

- [ ] **Step 4: 编写迁移与模型**

  迁移 Up 必须先限定删除 MML 数据，再增加字段：

  ```sql
  -- +goose Up
  DELETE FROM device_tasks WHERE source = 'mml';
  DELETE FROM mml_tasks;
  DELETE FROM mml_scripts;

  ALTER TABLE mml_scripts
      ADD COLUMN import_session_id uuid NOT NULL,
      ADD COLUMN original_filename text NOT NULL DEFAULT '',
      ADD COLUMN content_sha256 text NOT NULL DEFAULT '',
      ADD COLUMN validation_version text NOT NULL DEFAULT '',
      ADD COLUMN validated_at timestamptz,
      ADD COLUMN plan_items jsonb NOT NULL DEFAULT '[]'::jsonb,
      ADD COLUMN validation_summary jsonb NOT NULL DEFAULT '{}'::jsonb;

  ALTER TABLE mml_tasks
      ADD COLUMN script_content_sha256 text NOT NULL DEFAULT '',
      ADD COLUMN script_validation_version text NOT NULL DEFAULT '';

  CREATE INDEX idx_mml_scripts_content_sha256 ON mml_scripts(content_sha256);
  CREATE UNIQUE INDEX uq_mml_scripts_import_session_id ON mml_scripts(import_session_id);
  CREATE INDEX idx_mml_scripts_plan_items_gin
      ON mml_scripts USING gin (plan_items jsonb_path_ops);
  ```

  Down 只撤销新列和索引，不恢复已删除历史数据；文件头必须明确此不可逆数据语义。

- [ ] **Step 5: 更新 Repository 扫描契约**

  `MMLScript` 新字段使用确定类型：

  ```go
  ImportSessionID    uuid.UUID     `json:"-"`
  OriginalFilename  string        `json:"original_filename"`
  ContentSHA256     string        `json:"content_sha256"`
  ValidationVersion string        `json:"validation_version"`
  ValidatedAt       *time.Time    `json:"validated_at,omitempty"`
  PlanItems         []MMLPlanItem `json:"plan_items"`
  ValidationSummary JSONMap       `json:"validation_summary"`
  ```

  `MMLTask` 增加两个字符串字段，并同步 `scriptColumns`、`taskColumns`、insert、update、scan。

- [ ] **Step 6: 运行迁移与仓库验证**

  Run:

  ```bash
  bash omcgo/scripts/check-migrations.sh --strict
  cd omcgo && go test ./internal/mml -run 'TestPgScriptRepository|TestPgTaskRepository' -count=1
  cd omcgo && go test ./internal/mml -count=1
  ```

  Expected: migration strict check PASS；MML 测试 PASS，真实 PG 不可用的集成测试只允许按既有规则 SKIP。

- [ ] **Step 7: 更新进度并提交**

  ```bash
  git add omcgo/migrations/000015_redesign_mml_script_txt_import.sql \
    omcgo/internal/mml/model.go omcgo/internal/mml/pg_repository.go \
    omcgo/internal/mml/pg_repository_test.go \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(mml): 建立 TXT 导入脚本数据契约"
  ```

### Task 2: TXT Parser

**Files:**
- Create: `omcgo/internal/mml/script_import_parser.go`
- Create: `omcgo/internal/mml/script_import_parser_test.go`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `ParseScriptTXT(raw []byte) (*ParsedScript, []ScriptIssue)`。
- Produces: `ParsedScript{NormalizedContent string, SHA256 string, Lines []ParsedScriptLine}`。
- Produces: `ParsedScriptLine{LineNo int, RawLine string, DeviceSN string, Order int, CommandCode string, OperationType string, Parameters map[string]string}`。

- [ ] **Step 1: 写解析器失败测试**

  ```go
  func TestParseScriptTXT_DerivesPerDeviceOrder(t *testing.T) {
      raw := []byte("\xef\xbb\xbf# note\r\nLST DEVICE_INFO;SN1\r\nMOD DEVICE_INFO:USER_LABEL=A;SN2\r\nLST DEVICE_INFO;SN2\r\n")
      got, issues := ParseScriptTXT(raw)
      require.Empty(t, issues)
      require.Equal(t, "# note\nLST DEVICE_INFO;SN1\nMOD DEVICE_INFO:USER_LABEL=A;SN2\nLST DEVICE_INFO;SN2\n", got.NormalizedContent)
      require.Equal(t, []int{1, 1, 2}, []int{got.Lines[0].Order, got.Lines[1].Order, got.Lines[2].Order})
      require.Equal(t, []int{2, 3, 4}, []int{got.Lines[0].LineNo, got.Lines[1].LineNo, got.Lines[2].LineNo})
  }
  ```

  同文件增加表驱动用例：空文件、非 UTF-8、缺少 SN、多 SN、两条命令拼一行、引号/大括号内分号、2001 行。

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/mml -run TestParseScriptTXT -count=1 -v`
  Expected: FAIL with `undefined: ParseScriptTXT`。

- [ ] **Step 3: 实现纯解析器**

  核心入口必须无数据库、Redis、Gin 依赖：

  ```go
  const (
      MaxScriptDevices = 200
      MaxScriptLines   = 2000
      MaxScriptBytes   = 2 << 20
      ValidationVersion = "mml-txt-v1"
  )

  func ParseScriptTXT(raw []byte) (*ParsedScript, []ScriptIssue) {
      if len(raw) > MaxScriptBytes {
          return nil, []ScriptIssue{{Code: "MML_FILE_TOO_LARGE", Severity: IssueError}}
      }
      raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
      if !utf8.Valid(raw) {
          return nil, []ScriptIssue{{Code: "MML_FILE_ENCODING_INVALID", Severity: IssueError}}
      }
      normalized := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\r", "\n")
      orders := map[string]int{}
      lines := make([]ParsedScriptLine, 0)
      issues := make([]ScriptIssue, 0)
      for idx, physical := range strings.Split(normalized, "\n") {
          parsed, issue := parseScriptPhysicalLine(idx+1, physical)
          if issue != nil {
              issues = append(issues, *issue)
              continue
          }
          if parsed == nil {
              continue
          }
          orders[parsed.DeviceSN]++
          parsed.Order = orders[parsed.DeviceSN]
          lines = append(lines, *parsed)
      }
      if len(lines) == 0 && len(issues) == 0 {
          issues = append(issues, ScriptIssue{Code: "MML_FILE_EMPTY", Severity: IssueError})
      }
      if len(lines) > MaxScriptLines {
          issues = append(issues, ScriptIssue{Code: "MML_FILE_TOO_LARGE", Severity: IssueError})
      }
      canonical := strings.TrimRight(normalized, "\n") + "\n"
      digest := sha256.Sum256([]byte(canonical))
      return &ParsedScript{NormalizedContent: canonical, SHA256: hex.EncodeToString(digest[:]), Lines: lines}, issues
  }
  ```

  同文件完整实现 `parseScriptPhysicalLine`、顶层分隔符扫描和参数 token 扫描；双引号、单引号和大括号内部的 `;`、`,` 不得被当作顶层分隔符。`NormalizedContent` 保留注释和内部空行，只移除 BOM、统一换行并收敛文件末尾换行。

- [ ] **Step 4: 运行解析器测试**

  Run: `cd omcgo && go test ./internal/mml -run TestParseScriptTXT -count=1 -v`
  Expected: PASS，所有错误均携带稳定 code、原始 line_no 和 raw_line。

- [ ] **Step 5: 更新进度并提交**

  ```bash
  git add omcgo/internal/mml/script_import_parser.go \
    omcgo/internal/mml/script_import_parser_test.go \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(mml): 实现 TXT 脚本解析器"
  ```

### Task 3: 批量命令与设备校验

**Files:**
- Create: `omcgo/internal/mml/script_import_validator.go`
- Create: `omcgo/internal/mml/script_import_validator_test.go`
- Modify: `omcgo/internal/mml/repository.go`
- Modify: `omcgo/internal/mml/pg_repository.go`
- Modify: `omcgo/internal/mml/pg_repository_test.go`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `ScriptValidationRepository.LoadCommandsByCodes(ctx, codes)` 与 `LoadDevicesBySNs(ctx, sns)` 两个批量方法。
- Produces: `ValidateParsedScript(ctx, parsed, actor) (*ScriptValidationResult, error)`。
- Consumes: Task 2 的 `ParsedScript`。
- Produces: 规范化 `[]MMLPlanItem`、`ScriptValidationSummary` 和逐行 `[]ScriptIssue`。

- [ ] **Step 1: 写 Validator 失败测试**

  ```go
  func TestScriptImportValidator_BatchesAndRejectsUnknownParameter(t *testing.T) {
      repo := &fakeScriptValidationRepo{
          commands: map[string]MMLCommand{"MOD DEVICE_INFO": {CommandCode: "MOD DEVICE_INFO", OperationType: "MOD"}},
          params: map[string][]MMLParamRef{"MOD DEVICE_INFO": {{ParamCode: "USER_LABEL", IsWritable: true, ValueType: "string"}}},
          devices: map[string]*model.Device{"SN1": {SerialNumber: "SN1", ProductClass: "PC1"}},
      }
      parsed, parseIssues := ParseScriptTXT([]byte("MOD DEVICE_INFO:UNKNOWN=A;SN1\n"))
      require.Empty(t, parseIssues)
      result, err := NewScriptImportValidator(repo).Validate(context.Background(), parsed, ValidationActor{Username: "admin"})
      require.NoError(t, err)
      require.Equal(t, 1, repo.commandBatchCalls)
      require.Equal(t, 1, repo.deviceBatchCalls)
      require.Equal(t, "MML_PARAMETER_UNKNOWN", result.Issues[0].Code)
  }
  ```

  增加命令不存在、命令禁用、必填缺失、只读修改、类型/枚举/范围/正则、设备不存在、离线警告、危险命令警告、200 台边界用例。

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/mml -run TestScriptImportValidator -count=1 -v`
  Expected: FAIL with `undefined: NewScriptImportValidator`。

- [ ] **Step 3: 实现批量仓库与 Validator**

  仓库接口固定为：

  ```go
  type ScriptValidationRepository interface {
      LoadCommandsByCodes(ctx context.Context, codes []string, actor ValidationActor) (map[string]ValidationCommand, error)
      LoadDevicesBySNs(ctx context.Context, sns []string) (map[string]*model.Device, error)
  }
  ```

  `ValidationCommand` 聚合标准或当前用户可见自定义命令、参数约束、`RequireConfirm` 和目标路径。SQL 使用 `WHERE command_code = ANY($1)`、`WHERE serial_number = ANY($1)` 及批量 sub-field 查询。

- [ ] **Step 4: 将解析行编译为计划项**

  每条有效行输出：

  ```go
  MMLPlanItem{
      LineNo: line.LineNo,
      DeviceSN: line.DeviceSN,
      Order: line.Order,
      RawLine: line.RawLine,
      Command: map[string]interface{}{
          "command_code": command.CommandCode,
          "operation_type": command.OperationType,
          "rpc_method": command.RPCMethod,
          "parameters": stringMapToAny(line.Parameters),
          "param_refs": command.ParamRefs,
      },
  }
  ```

- [ ] **Step 5: 运行校验测试和查询次数测试**

  Run:

  ```bash
  cd omcgo && go test ./internal/mml -run 'TestScriptImportValidator|TestPgScriptValidationRepository' -count=1 -v
  ```

  Expected: PASS；2000 行相同命令和 200 个 SN 仍只调用一次命令批量加载、一次设备批量加载和一次参数批量加载。

- [ ] **Step 6: 更新进度并提交**

  ```bash
  git add omcgo/internal/mml/script_import_validator.go \
    omcgo/internal/mml/script_import_validator_test.go \
    omcgo/internal/mml/repository.go omcgo/internal/mml/pg_repository.go \
    omcgo/internal/mml/pg_repository_test.go \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(mml): 增加 TXT 脚本权威校验"
  ```

### Task 4: Redis 一次性校验会话

**Files:**
- Create: `omcgo/internal/mml/script_import_session.go`
- Create: `omcgo/internal/mml/script_import_session_test.go`
- Modify: `omcgo/internal/core/components/redisx/keys.go`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `ImportSessionStore.Put`, `Get`, `Claim`, `Release`, `Finalize`。
- Produces: Redis key `omc:mml:script-import:<sha256(token)>`，TTL 15 分钟。
- Consumes: Task 3 的 `ScriptValidationResult`。

- [ ] **Step 1: 写 miniredis 失败测试**

  ```go
  func TestRedisImportSessionStore_ClaimOnceAndBindUser(t *testing.T) {
      mr := miniredis.RunT(t)
      client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
      store := NewRedisImportSessionStore(client, 15*time.Minute)
      token, err := store.Put(context.Background(), "alice", &ImportSession{ContentSHA256: "abc"})
      require.NoError(t, err)
      _, err = store.Get(context.Background(), token, "bob")
      require.ErrorIs(t, err, ErrImportTokenOwnerMismatch)
      session, err := store.Claim(context.Background(), token, "alice", "request-1")
      require.NoError(t, err)
      require.NotEqual(t, uuid.Nil, session.ID)
      _, err = store.Claim(context.Background(), token, "alice", "request-2")
      require.ErrorIs(t, err, ErrImportTokenClaimed)
      require.NoError(t, store.Finalize(context.Background(), token, "alice", "request-1"))
      _, err = store.Claim(context.Background(), token, "alice", "request-1")
      require.ErrorIs(t, err, ErrImportTokenConsumed)
  }
  ```

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/mml -run TestRedisImportSessionStore -count=1 -v`
  Expected: FAIL with `undefined: NewRedisImportSessionStore`。

- [ ] **Step 3: 实现随机令牌、哈希 key 与原子消费**

  ```go
  type ImportSessionStore interface {
      Put(ctx context.Context, username string, session *ImportSession) (string, error)
      Get(ctx context.Context, token, username string) (*ImportSession, error)
      Claim(ctx context.Context, token, username, requestID string) (*ImportSession, error)
      Release(ctx context.Context, token, username, requestID string) error
      Finalize(ctx context.Context, token, username, requestID string) error
  }
  ```

  `Put` 为 session 生成 UUID，并使用 `crypto/rand` 生成 32 字节令牌；Redis 只保存令牌 SHA-256。`Claim` 使用 Lua 原子写入 requestID；`Release` 只释放同一 requestID；`Finalize` 删除会话并写入同 TTL 的 consumed tombstone。令牌过期、用户不匹配、已被其他请求 claim 和已消费分别映射稳定错误。

- [ ] **Step 4: 运行会话测试**

  Run: `cd omcgo && go test ./internal/mml -run TestRedisImportSessionStore -count=1 -v`
  Expected: PASS，TTL、越权、过期、claim/release、并发双 claim 和 consumed tombstone 均有断言。

- [ ] **Step 5: 更新进度并提交**

  ```bash
  git add omcgo/internal/mml/script_import_session.go \
    omcgo/internal/mml/script_import_session_test.go \
    omcgo/internal/core/components/redisx/keys.go \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(mml): 增加脚本导入校验会话"
  ```

### Task 5: 事务化脚本导入服务

**Files:**
- Create: `omcgo/internal/mml/script_import_service.go`
- Create: `omcgo/internal/mml/script_import_service_test.go`
- Modify: `omcgo/internal/mml/repository.go`
- Modify: `omcgo/internal/mml/pg_repository.go`
- Modify: `omcgo/internal/mml/pg_repository_test.go`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `ValidateScriptImport(ctx, username, filename string, raw []byte) (*ImportValidationResponse, error)`。
- Produces: `CreateScriptFromImport(ctx, username string, req SaveImportedScriptRequest) (*MMLScript, error)`。
- Produces: `ReplaceScriptFromImport(ctx, id, username string, req ReplaceImportedScriptRequest) (*MMLScript, error)`。

- [ ] **Step 1: 写服务失败测试**

  ```go
  func TestScriptImportService_DoesNotConsumeTokenWhenRepositoryFails(t *testing.T) {
      sessions := newFakeImportSessionStore(validSession())
      repo := &fakeScriptRepo{createErr: errors.New("pg unavailable")}
      svc := NewScriptImportService(repo, fakeValidator(), sessions, zap.NewNop())
      _, err := svc.Create(context.Background(), "alice", SaveImportedScriptRequest{ValidationToken: "token", ScriptName: "巡检"})
      require.ErrorContains(t, err, "pg unavailable")
      require.Equal(t, 1, sessions.claimCalls)
      require.Equal(t, 1, sessions.releaseCalls)
      require.Equal(t, 0, sessions.finalizeCalls)
  }
  ```

  增加错误结果阻止创建、令牌归属、名称必填、保存字段一致、重新导入版本冲突用例。

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/mml -run TestScriptImportService -count=1 -v`
  Expected: FAIL with `undefined: NewScriptImportService`。

- [ ] **Step 3: 增加 Repository 原子方法**

  ```go
  type ImportedScriptRepository interface {
      ScriptRepository
      CreateImported(ctx context.Context, script *MMLScript) error
      GetByImportSessionID(ctx context.Context, sessionID uuid.UUID) (*MMLScript, error)
      ReplaceImported(ctx context.Context, script *MMLScript, expectedUpdatedAt time.Time) error
      UpdateMetadata(ctx context.Context, id uuid.UUID, name, description string, tags []string) error
  }
  ```

  `ReplaceImported` 的 WHERE 同时包含 `id` 和 `updated_at`；0 行更新时先判定 ID 是否存在，再返回 not found 或 `ErrScriptVersionConflict`。

- [ ] **Step 4: 实现校验与保存编排**

  `Validate` 调用 Parser 和 Validator；有错误仍返回预览，但不生成可保存令牌。无错误时把全部权威结果写入 ImportSession。`Create` 先 `Claim`，把 session UUID 写入 `mml_scripts.import_session_id`，数据库失败时 `Release`；写库成功后 `Finalize`。若客户端因网络失败重试，先按唯一 session UUID 查询并返回同一脚本，保证跨 Redis/PostgreSQL 的幂等结果。

- [ ] **Step 5: 运行服务与仓库测试**

  Run:

  ```bash
  cd omcgo && go test ./internal/mml -run 'TestScriptImportService|TestPgScriptRepository' -count=1 -v
  ```

  Expected: PASS，创建失败不消费令牌、重复确认只产生一条脚本、并发替换返回 409 对应哨兵错误。

- [ ] **Step 6: 更新进度并提交**

  ```bash
  git add omcgo/internal/mml/script_import_service.go \
    omcgo/internal/mml/script_import_service_test.go \
    omcgo/internal/mml/repository.go omcgo/internal/mml/pg_repository.go \
    omcgo/internal/mml/pg_repository_test.go \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(mml): 支持原子导入 TXT 脚本"
  ```

### Task 6: 导入 HTTP API 与依赖注入

**Files:**
- Create: `omcgo/internal/mml/assets/MMLTemplate.txt`
- Create: `omcgo/internal/mml/script_import_template.go`
- Modify: `omcgo/internal/mml/handler.go`
- Modify: `omcgo/internal/mml/handler_test.go`
- Modify: `omcgo/cmd/app/provider/modules.go`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `GET /api/v1/mml/scripts/import/template`，响应 `text/plain; charset=utf-8` 和附件名 `MMLTemplate.txt`。
- Produces: `POST /api/v1/mml/scripts/import/validate`。
- Produces: `POST /api/v1/mml/scripts/import`。
- Produces: `POST /api/v1/mml/scripts/:id/import/validate` 与 `PUT /api/v1/mml/scripts/:id/import`。
- Produces: 基本信息更新接口只接受 `script_name`、`description`、`tags`。

- [ ] **Step 1: 写路由和 multipart 失败测试**

  ```go
  func TestHandler_ValidateScriptImport_RejectsNonTXT(t *testing.T) {
      h, router := newImportHandlerHarness(t)
      _ = h
      body, contentType := multipartFile(t, "file", "script.csv", []byte("LST DEVICE_INFO;SN1\n"))
      req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/import/validate", body)
      req.Header.Set("Content-Type", contentType)
      rec := httptest.NewRecorder()
      router.ServeHTTP(rec, req)
      require.Equal(t, http.StatusBadRequest, rec.Code)
      require.Contains(t, rec.Body.String(), "MML_FILE_TYPE_INVALID")
  }
  ```

  增加模板下载内容与响应头、缺文件、请求体超过 2 MiB、422 逐行错误、201 保存、409 重放、403 越权测试。

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/mml -run 'TestHandler_.*ScriptImport' -count=1 -v`
  Expected: FAIL，路由返回 404 或 handler 未定义。

- [ ] **Step 3: 实现 Handler 与稳定错误映射**

  上传 handler 必须先限制请求体：

  ```go
  c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxScriptBytes+64*1024)
  file, header, err := c.Request.FormFile("file")
  ```

  从鉴权上下文获取 username，不读取表单中的 creator。错误响应包含稳定 `code`、`message`、`issues`；日志只记录 filename、size、sha256、issue codes，不记录脚本内容。

  `script_import_template.go` 使用 `//go:embed assets/MMLTemplate.txt` 提供唯一模板字节；模板包含注释、单 SN 行规则以及标准/自定义 LST、MOD、ADD、RMV 示例。下载 handler 禁止从工作目录动态读文件。

- [ ] **Step 4: 注入 Redis Session Store 和 Import Service**

  在 `modules.go` 使用现有 `c.Redis`：

  ```go
  importSessions := mml.NewRedisImportSessionStore(c.Redis, 15*time.Minute)
  importValidator := mml.NewScriptImportValidator(mml.NewPgScriptValidationRepository(c.PgPool))
  importService := mml.NewScriptImportService(mmlScriptRepo, importValidator, importSessions, logger)
  mmlService.SetScriptImportService(importService)
  ```

- [ ] **Step 5: 运行 Handler、MML 包和构建验证**

  Run:

  ```bash
  cd omcgo && go test ./internal/mml -run 'TestHandler_.*ScriptImport' -count=1 -v
  cd omcgo && go test ./internal/mml -count=1
  cd omcgo && go build ./cmd/app
  ```

  Expected: 全部 PASS；请求体超限返回 413，业务校验错误返回 422。

- [ ] **Step 6: 更新进度并提交**

  ```bash
  git add omcgo/internal/mml/assets/MMLTemplate.txt \
    omcgo/internal/mml/script_import_template.go \
    omcgo/internal/mml/handler.go omcgo/internal/mml/handler_test.go \
    omcgo/cmd/app/provider/modules.go \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(mml): 暴露 TXT 脚本导入接口"
  ```

### Task 7: 脚本执行预检与任务快照

**Files:**
- Create: `omcgo/internal/mml/script_execution.go`
- Create: `omcgo/internal/mml/script_execution_test.go`
- Modify: `omcgo/internal/mml/handler.go`
- Modify: `omcgo/internal/mml/handler_test.go`
- Modify: `omcgo/internal/mml/service.go`
- Modify: `omcgo/internal/mml/scheduler.go`
- Modify: `omcgo/internal/mml/scheduler_test.go`
- Modify: `omcgo/internal/mml/sequencer_test.go`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `POST /api/v1/mml/scripts/:id/executions`。
- Produces: `CreateScriptExecution(ctx, scriptID, actor, req) (*MMLTask, *ScriptValidationResult, error)`。
- Consumes: Task 3 Validator、Task 1 脚本计划字段和现有 `ExecuteCommand`。

- [ ] **Step 1: 写执行快照失败测试**

  ```go
  func TestCreateScriptExecution_UsesStoredPlanNotClientCommands(t *testing.T) {
      script := importedScriptFixture("sha-a", []MMLPlanItem{planItem(2, "SN1", 1, "LST DEVICE_INFO")})
      svc, tasks := newScriptExecutionHarness(script)
      task, validation, err := svc.CreateScriptExecution(context.Background(), script.ID, "alice", ScriptExecutionRequest{
          TaskName: "巡检", ExecuteType: ExecuteImmediate,
      })
      require.NoError(t, err)
      require.Empty(t, validation.Errors())
      require.Equal(t, script.PlanItems, task.PlanItems)
      require.Equal(t, "sha-a", task.ScriptContentSHA256)
      require.Equal(t, 1, tasks.createCalls)
  }
  ```

  增加动态错误阻断、警告需 `ConfirmWarnings`、定时触发再次预检、脚本替换不影响已有任务快照、前一行失败继续下一行测试。

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/mml -run 'TestCreateScriptExecution|TestScheduler_.*Preflight|TestSequencer_.*Continue' -count=1 -v`
  Expected: FAIL，新入口和预检未定义。

- [ ] **Step 3: 实现执行请求与预检**

  ```go
  type ScriptExecutionRequest struct {
      TaskName           string      `json:"task_name"`
      ExecuteType        ExecuteType `json:"execute_type"`
      ScheduledAt        *string     `json:"scheduled_at"`
      PeriodStart        *string     `json:"period_start"`
      PeriodEnd          *string     `json:"period_end"`
      PeriodTime         string      `json:"period_time"`
      OfflineRetry       bool        `json:"offline_retry"`
      OfflineRetryWait   int         `json:"offline_retry_wait"`
      FailedRetry        bool        `json:"failed_retry"`
      FailedRetryCount   int         `json:"failed_retry_count"`
      FailedRetryInterval int        `json:"failed_retry_interval"`
      ConfirmWarnings    bool        `json:"confirm_warnings"`
  }
  ```

  Service 从脚本记录重建 `ExecuteRequest{ExecuteMode:"device_bound", PlanItems:clone(script.PlanItems)}`，不从 HTTP 接受命令和 SN。动态错误返回 422；只有警告且未确认返回 409 和 warnings。

- [ ] **Step 4: 调度触发前再次预检**

  Scheduler 对 scheduled/periodic 实例加载脚本快照对应命令和设备，运行动态检查；阻断错误把本次实例标记 failed 并记录逐行问题，不能静默保持 pending。

- [ ] **Step 5: 运行执行链测试**

  Run:

  ```bash
  cd omcgo && go test ./internal/mml -run 'TestCreateScriptExecution|TestScheduler|TestSequencer|TestFanout' -count=1
  cd omcgo && go test ./internal/mml -count=1
  ```

  Expected: PASS；同 SN 串行、跨 SN 并行、失败继续和计划快照均有断言。

- [ ] **Step 6: 更新进度并提交**

  ```bash
  git add omcgo/internal/mml/script_execution.go omcgo/internal/mml/script_execution_test.go \
    omcgo/internal/mml/handler.go omcgo/internal/mml/handler_test.go \
    omcgo/internal/mml/service.go omcgo/internal/mml/scheduler.go \
    omcgo/internal/mml/scheduler_test.go omcgo/internal/mml/sequencer_test.go \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(mml): 从导入脚本创建执行快照"
  ```

### Task 8: frontend-core 契约、API 与 Hooks

**Files:**
- Modify: `omcmb/frontend-core/src/types/mml.ts`
- Modify: `omcmb/frontend-core/src/services/api/mmlApi.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useMML.ts`
- Create: `omcmb/frontend-core/src/services/api/__tests__/mmlScriptImportApi.test.ts`
- Create: `omcmb/frontend-core/src/hooks/api/__tests__/useMMLScriptImport.test.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: `MMLScriptImportValidation`, `MMLScriptIssue`, `MMLScriptValidationSummary`, `MMLScriptExecutionInput`。
- Produces: `mmlApi.validateScriptImport`, `createImportedScript`, `validateScriptReplacement`, `replaceImportedScript`, `createScriptExecution`。
- Produces: `mmlApi.downloadScriptImportTemplate`，从服务端取得 Blob 和响应文件名。
- Produces: 对应 React Query mutation hooks。

- [ ] **Step 1: 写 API 映射失败测试**

  ```ts
  it('sends multipart validate request and maps snake_case issues', async () => {
    mock.onPost('/mml/scripts/import/validate').reply(200, {
      validation_token: 'token',
      summary: { effective_lines: 1, device_count: 1, error_count: 0, warning_count: 0 },
      plan_items: [{ line_no: 1, device_sn: 'SN1', order: 1, command: { command_code: 'LST DEVICE_INFO' } }],
      issues: [],
    })
    const result = await mmlApi.validateScriptImport(new File(['LST DEVICE_INFO;SN1'], 'script.txt'))
    expect(result.validationToken).toBe('token')
    expect(result.planItems[0].deviceSn).toBe('SN1')
  })
  ```

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcmb/webcode && npm run test -- --run ../frontend-core/src/services/api/__tests__/mmlScriptImportApi.test.ts`
  Expected: FAIL，API 方法或类型不存在。

- [ ] **Step 3: 实现共享类型和 API**

  `validateScriptImport(file)` 使用 `FormData`；不得手动设置 multipart boundary。`downloadScriptImportTemplate()` 以 Blob 请求服务端模板端点。保存请求只发送 token 和基本信息。执行请求不包含 `commands`、`deviceSns` 或 `planItems`。

- [ ] **Step 4: 实现 Hooks 与缓存失效**

  ```ts
  export function useValidateMMLScriptImport() {
    return useMutation({ mutationFn: (file: File) => api.validateScriptImport(file) })
  }

  export function useCreateImportedMMLScript() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: (input: MMLImportedScriptCreateInput) => api.createImportedScript(input),
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] }),
    })
  }

  export function useReplaceImportedMMLScript() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ id, input }: { id: string; input: MMLImportedScriptReplaceInput }) =>
        api.replaceImportedScript(id, input),
      onSuccess: (_, vars) => Promise.all([
        queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] }),
        queryClient.invalidateQueries({ queryKey: ['mml', 'scripts', vars.id] }),
      ]),
    })
  }

  export function useCreateMMLScriptExecution() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ id, input }: { id: string; input: MMLScriptExecutionInput }) =>
        api.createScriptExecution(id, input),
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] }),
    })
  }
  ```

- [ ] **Step 5: 运行共享层测试与类型检查**

  Run:

  ```bash
  cd omcmb/webcode && npm run test -- --run ../frontend-core/src/services/api/__tests__/mmlScriptImportApi.test.ts ../frontend-core/src/hooks/api/__tests__/useMMLScriptImport.test.tsx
  cd omcmb && npm run typecheck
  ```

  Expected: tests PASS；三皮肤 typecheck 和 skin parity PASS。

- [ ] **Step 6: 更新进度并提交**

  ```bash
  git add omcmb/frontend-core/src/types/mml.ts \
    omcmb/frontend-core/src/services/api/mmlApi.ts \
    omcmb/frontend-core/src/hooks/api/useMML.ts \
    omcmb/frontend-core/src/services/api/__tests__/mmlScriptImportApi.test.ts \
    omcmb/frontend-core/src/hooks/api/__tests__/useMMLScriptImport.test.tsx \
    omcmb/frontend-core/src/i18n/zh-CN/index.ts omcmb/frontend-core/src/i18n/en-US/index.ts \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(frontend-core): 接入 MML TXT 导入契约"
  ```

### Task 9: v1 脚本库界面

**Files:**
- Create: `omcmb/webcode/src/pages/mml/ScriptTask/ScriptImportModal.tsx`
- Create: `omcmb/webcode/src/pages/mml/ScriptTask/ScriptImportPreview.tsx`
- Create: `omcmb/webcode/src/pages/mml/ScriptTask/ScriptExecutionDrawer.tsx`
- Create: `omcmb/webcode/src/pages/mml/ScriptTask/__tests__/ScriptImportModal.test.tsx`
- Create: `omcmb/webcode/src/pages/mml/ScriptTask/__tests__/ScriptExecutionDrawer.test.tsx`
- Modify: `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Consumes: Task 8 Hooks 和类型。
- Produces: v1 新增、只读详情、重新导入、下载 TXT 和执行配置界面。

- [ ] **Step 1: 写 v1 交互失败测试**

  ```tsx
  it('keeps save disabled when server validation has errors', async () => {
    render(<ScriptImportModal open onClose={vi.fn()} />)
    await userEvent.upload(screen.getByLabelText('选择 TXT'), new File(['BAD'], 'bad.txt', { type: 'text/plain' }))
    expect(await screen.findByText('MML_LINE_FORMAT_INVALID')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '确认保存' })).toBeDisabled()
    expect(screen.queryByRole('textbox', { name: '脚本内容' })).not.toBeInTheDocument()
  })
  ```

  增加警告允许保存、重新上传替换预览、只读详情、执行警告二次确认测试。

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcmb/webcode && npm run test -- --run src/pages/mml/ScriptTask/__tests__`
  Expected: FAIL，新组件不存在。

- [ ] **Step 3: 实现导入和只读预览**

  页面字段固定为基本信息、TXT 上传、模板下载、文件摘要、统计卡、按行预览、错误/警告过滤、错误报告下载。移除 `CommandSelectModal`、内容 TextArea 和模式选择器。

- [ ] **Step 4: 实现执行 Drawer**

  Drawer 只发送任务名称、执行方式、时间和重试策略。服务端返回 warnings 时显示确认 Modal，再以 `confirmWarnings=true` 重试；errors 保持 Drawer 打开并显示逐行结果。

- [ ] **Step 5: 运行 v1 测试和类型检查**

  Run:

  ```bash
  cd omcmb/webcode && npm run test -- --run src/pages/mml/ScriptTask/__tests__
  cd omcmb/webcode && npm run typecheck
  ```

  Expected: PASS，页面中没有在线脚本内容编辑器。

- [ ] **Step 6: 更新进度并提交**

  ```bash
  git add omcmb/webcode/src/pages/mml/ScriptTask \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(frontend): 重构 v1 MML 脚本导入界面"
  ```

### Task 10: v2 脚本库界面

**Files:**
- Create: `omcmb/webcode-v2/src/pages/mml/components/ScriptImportDialog.tsx`
- Create: `omcmb/webcode-v2/src/pages/mml/components/ScriptExecutionDialog.tsx`
- Create: `omcmb/webcode-v2/src/pages/mml/__tests__/ScriptImportDialog.test.tsx`
- Modify: `omcmb/webcode-v2/src/pages/mml/ScriptTask.tsx`
- Modify: `omcmb/webcode-v2/package.json`
- Create: `omcmb/webcode-v2/vitest.config.ts`
- Modify: `omcmb/package-lock.json`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Consumes: Task 8 Hooks 和与 v1 相同的字段/状态语义。
- Produces: v2 shadcn/Tailwind 版本，不复制解析或 API 逻辑。

- [ ] **Step 1: 写 v2 能力对齐失败测试**

  测试必须断言：TXT 上传、模板下载、错误过滤、只读预览、重新导入、执行方式、警告确认均存在，且错误时保存禁用。

- [ ] **Step 2: 建立 v2 测试入口并确认业务测试失败**

  在 v2 增加 `"test": "vitest run"`，并在 `vitest.config.ts` 配置 React、`@`/`@core` alias、`environment: 'jsdom'` 和 Testing Library setup。添加与 webcode 相同版本的 `vitest`、`jsdom`、`@testing-library/react`、`@testing-library/user-event` 后运行 `npm install --package-lock-only`。

  Run: `cd omcmb/webcode-v2 && npm run test -- --run src/pages/mml/__tests__/ScriptImportDialog.test.tsx`
  Expected: FAIL，新导入 Dialog 不存在。

- [ ] **Step 3: 实现 v2 页面**

  使用 v2 的 `Dialog`、`Button`、`Table` 和表单组件；业务判断只消费共享 hook 返回的 `status/issues/summary`，不得调用 `parseMmlScriptPlan`。

- [ ] **Step 4: 运行 v2 验证**

  Run:

  ```bash
  cd omcmb/webcode-v2 && npm run test -- --run src/pages/mml/__tests__/ScriptImportDialog.test.tsx
  cd omcmb/webcode-v2 && npm run typecheck
  ```

  Expected: PASS。

- [ ] **Step 5: 更新进度并提交**

  ```bash
  git add omcmb/webcode-v2/src/pages/mml omcmb/webcode-v2/package.json \
    omcmb/webcode-v2/vitest.config.ts omcmb/package-lock.json \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(frontend): 补齐 v2 MML 脚本导入"
  ```

### Task 11: v3 脚本库界面

**Files:**
- Create: `omcmb/webcode-v3/src/pages/mml/script/ScriptImportDialog.tsx`
- Create: `omcmb/webcode-v3/src/pages/mml/script/ScriptExecutionDialog.tsx`
- Create: `omcmb/webcode-v3/src/pages/mml/script/__tests__/ScriptImportDialog.test.tsx`
- Modify: `omcmb/webcode-v3/src/pages/mml/script/index.tsx`
- Modify: `omcmb/webcode-v3/package.json`
- Create: `omcmb/webcode-v3/vitest.config.ts`
- Modify: `omcmb/package-lock.json`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Consumes: Task 8 Hooks 和与 v1/v2 相同的字段/状态语义。
- Produces: v3 STARFORGE 版本，不复制解析或 API 逻辑。

- [ ] **Step 1: 写 v3 能力对齐失败测试**

  测试断言与 Task 10 完全相同的业务能力，并额外断言上传/校验 loading 状态使用 v3 可访问状态文本，而不是只靠动画。

- [ ] **Step 2: 建立 v3 测试入口并确认业务测试失败**

  按 Task 10 的精确版本增加 `test` script、Vitest/jsdom/Testing Library devDependencies 和 v3 alias 配置，然后运行 `npm install --package-lock-only`。

  Run: `cd omcmb/webcode-v3 && npm run test -- --run src/pages/mml/script/__tests__/ScriptImportDialog.test.tsx`
  Expected: FAIL，新导入 Dialog 不存在。

- [ ] **Step 3: 实现 v3 页面**

  使用 v3 GlassPanel/NeonButton 视觉体系；保留文件名、摘要、统计、逐行问题和执行策略的完整可读文本。

- [ ] **Step 4: 运行 v3 和三皮肤一致性验证**

  Run:

  ```bash
  cd omcmb/webcode-v3 && npm run test -- --run src/pages/mml/script/__tests__/ScriptImportDialog.test.tsx
  cd omcmb/webcode-v3 && npm run typecheck
  cd omcmb && npm run skin-parity
  ```

  Expected: PASS，三皮肤路由和菜单集合一致。

- [ ] **Step 5: 更新进度并提交**

  ```bash
  git add omcmb/webcode-v3/src/pages/mml/script omcmb/webcode-v3/package.json \
    omcmb/webcode-v3/vitest.config.ts omcmb/package-lock.json \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "feat(frontend): 补齐 v3 MML 脚本导入"
  ```

### Task 12: 旧入口收口与安全清理工具

**Files:**
- Modify: `omcgo/internal/mml/handler.go`
- Modify: `omcgo/internal/mml/handler_test.go`
- Modify: `omcgo/internal/task/redis_queue.go`
- Modify: `omcgo/internal/task/redis_queue_test.go`
- Modify: `omcgo/cmd/omcctl/mml.go`
- Create: `omcgo/cmd/omcctl/mml_reset_scripts_test.go`
- Modify: `omcmb/webcode/src/pages/mml/components/ScriptTaskDrawer.tsx`
- Modify: `omcmb/webcode/src/pages/mml/TaskRecord/index.tsx`
- Modify: `omcmb/frontend-core/src/utils/mmlScriptPlanParser.ts`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Removes: 浏览器直接提交脚本 `content/commands/plan_items` 的脚本保存入口。
- Produces: `RedisTaskQueue.PurgeBySource(ctx, task.TaskSourceMML, dryRun)`。
- Produces: `omcctl mml reset-script-data --redis-addr localhost:6379 --dry-run|--apply`，只清理 Redis MML 项；PostgreSQL 由 Task 1 迁移清理。

- [ ] **Step 1: 写 Redis 来源过滤失败测试**

  ```go
  func TestRedisTaskQueue_PurgeBySource_LeavesNonMMLTasks(t *testing.T) {
      q, _ := newRedisQueueWithMini(t)
      require.NoError(t, q.Push(context.Background(), &Task{ID: "m1", DeviceSN: "SN1", Source: TaskSourceMML}))
      require.NoError(t, q.Push(context.Background(), &Task{ID: "a1", DeviceSN: "SN1", Source: TaskSourceAPI}))
      result, err := q.PurgeBySource(context.Background(), TaskSourceMML, false)
      require.NoError(t, err)
      require.Equal(t, int64(1), result.Deleted)
      require.NotNil(t, mustGetTask(t, q, "a1"))
  }
  ```

- [ ] **Step 2: 运行测试确认失败**

  Run: `cd omcgo && go test ./internal/task -run TestRedisTaskQueue_PurgeBySource -count=1 -v`
  Expected: FAIL，`PurgeBySource` 不存在。

- [ ] **Step 3: 实现安全 Redis 清理**

  使用 SCAN 遍历设备队列 key，pipeline 读取任务详情，只对 `task.Source == source` 的 ID 执行 `ZREM`、详情 hash 删除和 CWMP 映射删除。`dryRun=true` 只返回 matched，不修改 Redis。禁止 `KEYS` 和 `FLUSHDB`。

- [ ] **Step 4: 增加 omcctl 双确认命令**

  `--dry-run` 为默认；`--apply` 必须同时要求 `--confirm DELETE-MML-RUNTIME`。输出 matched/deleted/skipped/error 数量并返回非零退出码表示部分失败。

- [ ] **Step 5: 删除旧脚本创建和直接上传任务入口**

  后端普通 `POST /mml/scripts` 不再注册；`PUT /mml/scripts/:id` 只更新基本信息。任务记录页移除“上传脚本直接建任务”，改为跳转脚本库；MML Console 的单命令即时执行保留，但不得保存为脚本。删除三个皮肤对 `parseMmlScriptPlan` 的脚本保存依赖；工具文件若无其他消费者则连同测试删除。

- [ ] **Step 6: 运行收口验证**

  Run:

  ```bash
  cd omcgo && go test ./internal/task -run TestRedisTaskQueue_PurgeBySource -count=1
  cd omcgo && go test ./internal/mml -run 'TestHandler_.*Script' -count=1
  cd omcgo && go test ./cmd/omcctl -run TestMMLResetScripts -count=1
  cd omcmb && npm run typecheck
  rg -n "createScript\(|parseMmlScriptPlan" omcmb/webcode*/src/pages/mml omcmb/frontend-core/src
  ```

  Expected: tests/typecheck PASS；`rg` 只允许非脚本库的兼容测试或明确保留的 Console 解析用途，不得命中三皮肤脚本新增/编辑界面。

- [ ] **Step 7: 更新进度并提交**

  ```bash
  git add omcgo/internal/mml/handler.go omcgo/internal/mml/handler_test.go \
    omcgo/internal/task/redis_queue.go omcgo/internal/task/redis_queue_test.go \
    omcgo/cmd/omcctl/mml.go omcgo/cmd/omcctl/mml_reset_scripts_test.go \
    omcmb/webcode/src/pages/mml/components/ScriptTaskDrawer.tsx \
    omcmb/webcode/src/pages/mml/TaskRecord/index.tsx \
    omcmb/frontend-core/src/utils/mmlScriptPlanParser.ts \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "refactor(mml): 收口脚本入口并增加安全清理"
  ```

### Task 13: 全量验证、部署演练与文档收尾

**Files:**
- Create: `omcgo/internal/mml/testdata/import-valid.txt`
- Create: `omcgo/internal/mml/testdata/import-invalid.txt`
- Create: `omcgo/scripts/e2e_mml_script_import.sh`
- Modify: `deployments/docker/README.md`
- Modify: `docs/design/mml-script-txt-import-redesign-20260710.md`
- Modify: `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`

**Interfaces:**
- Produces: 可重复执行的 TXT 导入、保存、执行、结果追溯 E2E。
- Produces: 部署前 dry-run、Redis MML 清理、迁移、健康检查和回滚限制说明。

- [ ] **Step 1: 编写 E2E 失败脚本**

  脚本必须验证：非法文件返回 422 且不能保存；合法文件取得 token 并保存；同 token 重放返回 409；执行只使用脚本快照；结果包含 `plan_line_no`、`plan_device_sn`、`plan_order` 和 `script_content_sha256`。

- [ ] **Step 2: 在本地栈运行 E2E 并记录首次结果**

  Run: `bash omcgo/scripts/e2e_mml_script_import.sh`
  Expected on first run: 至少一个断言 FAIL；记录具体端点与响应，不接受空跑。

- [ ] **Step 3: 修复 E2E 暴露的问题并复跑**

  Run: `bash omcgo/scripts/e2e_mml_script_import.sh`
  Expected: 所有断言 PASS，输出 `MML script import E2E: PASS`。

- [ ] **Step 4: 执行完整质量门**

  Run:

  ```bash
  bash omcgo/scripts/check-migrations.sh --strict
  cd omcgo && go build ./...
  cd omcgo && go test ./...
  cd omcmb && npm run skin-parity
  cd omcmb && npm run typecheck
  cd omcmb/webcode && npm run test
  cd omcmb/webcode-v2 && npm run test
  cd omcmb/webcode-v3 && npm run test
  ```

  Expected: 所有命令 exit 0；测试输出 0 failures。环境导致的真实 PG 集成测试 SKIP 必须列入 Evidence Log，不能写成 PASS。

- [ ] **Step 5: 浏览器验收三皮肤**

  分别访问三套皮肤的 `/mml/script`：导入合法 TXT、导入非法 TXT、重新导入、下载 TXT、执行警告确认、查看任务结果。每套记录 URL、脚本 ID、任务 ID、网络响应和页面截图；不得出现 pageerror、404、空白预览或可编辑脚本文本框。

- [ ] **Step 6: 演练全量切换**

  顺序执行：停止 MML 新建和调度；`omcctl mml reset-script-data --dry-run`；记录数量；使用 `--apply --confirm DELETE-MML-RUNTIME` 清理 Redis；应用迁移；启动服务；检查 `device_tasks WHERE source <> 'mml'` 数量未变化；运行 E2E 和健康检查。

- [ ] **Step 7: 更新文档与最终进度**

  将实际 API、迁移文件名、错误码、部署命令和验证证据同步到设计稿及部署 README。把 Task 13 勾选完成，设置：

  ```text
  Current checkpoint: All tasks complete
  Resume from: Final review and delivery
  ```

- [ ] **Step 8: 最新主分支冲突门与提交**

  Run:

  ```bash
  git fetch origin main
  git rev-list --left-right --count HEAD...origin/main
  git merge-tree --write-tree --messages HEAD origin/main
  bash omcgo/scripts/check-migrations.sh --strict
  ```

  Expected: 无未解决冲突，迁移编号与最新主分支无碰撞。

  Commit:

  ```bash
  git add omcgo/internal/mml/testdata omcgo/scripts/e2e_mml_script_import.sh \
    deployments/docker/README.md docs/design/mml-script-txt-import-redesign-20260710.md \
    docs/superpowers/plans/2026-07-10-mml-script-txt-import.md
  git commit -m "test(mml): 完成 TXT 脚本导入端到端验收"
  ```

## Final Delivery Gate

只有以下证据同时存在才能声明完成：

- Progress Ledger 13 个任务全部勾选，Evidence Log 每项有 commit 和验证命令。
- `git status --short` 只包含用户原有未跟踪参考资料，不包含本功能未提交改动。
- 后端 build/test、前端 skin-parity/typecheck/test、migration strict check、E2E 全部有当轮输出。
- 三皮肤真实浏览器验收均有 URL、脚本 ID、任务 ID 和截图。
- Redis dry-run 与 apply 数量一致，非 MML `device_tasks` 数量切换前后不变。
- 提交前已对最新 `origin/main` 做 divergence、merge-tree 和迁移编号检查。
