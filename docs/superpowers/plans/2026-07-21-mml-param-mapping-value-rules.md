# MML Param Mapping Value Rules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `param-mappings/*.xml` 中的 min/max、defaultValue、validationPattern 和枚举规则刷新到数据库，并让 MML 控制台按当前参数模型提供选择和提交校验。

**Architecture:** 模型级规则继续存储在 `param_mappings`，通过 `standard_path` 与 `standard_params`/`mml_command_sub_fields` 对齐；标准树提供公共元数据和范围回退，参数模型规则优先。XML Loader 负责 builtin 全量刷新、custom 保留、discovered 对账和缓存刷新；控制台 API 输出结构化规则，webcode 负责输入控件和提交校验。

**Tech Stack:** Go、pgx、Squirrel、PostgreSQL/Goose、React、TypeScript、Ant Design、Vitest。

## Global Constraints

- 所有 Git 操作在仓库根目录或本隔离 worktree 执行，不能修改主工作区已有无关文件。
- 后端遵循 `handler -> service -> repository/model` 分层，SQL 使用 pgx/Squirrel，错误使用 `%w` 包装。
- 前端共享类型/API 放 `omcmb/frontend-core`，页面和交互放 `omcmb/webcode`。
- 不把产品模型差异化规则写入全局 `standard_params`。
- migration 使用主库 schema 当前序号 `000003`，必须包含 Goose Up/Down 和幂等 DDL。
- 每个行为先写失败测试并确认 RED，再写最小实现并确认 GREEN。

---

### Task 1: XML 规则模型与枚举规范化

**Files:**

- Modify: `omcgo/internal/config/parammodel/model.go`
- Modify: `omcgo/internal/config/parammodel/loader.go`
- Test: `omcgo/internal/config/parammodel/loader_test.go` 或同包新增测试文件

**Interfaces:**

- Produces `ParamMapping.DefaultValue`、`ParamMapping.ValidationPattern` 和 XML entry 对应字段。
- Produces a single enum normalization helper used by loader insert paths and tests.

- [ ] **Step 1: Write the failing test**

覆盖：

```go
func TestNormalizeEnumCSV_ReplacesFullWidthComma(t *testing.T) {
    got := normalizeEnumCSV("PSK，SIM，CERT，OTHER")
    require.Equal(t, "PSK,SIM,CERT,OTHER", got)
}

func TestXMLParamEntryCarriesValueRules(t *testing.T) {
    // unmarshal one parameter containing min/max/default/pattern/enum attrs
    // and assert every attribute is retained.
}
```

- [ ] **Step 2: Run the focused test and confirm RED**

```bash
cd omcgo && go test ./internal/config/parammodel -run 'TestNormalizeEnumCSV|TestXMLParamEntryCarriesValueRules'
```

Expected: FAIL because the new fields/helper do not exist.

- [ ] **Step 3: Implement the minimal model/parser changes**

Add `DefaultValue *string` and `ValidationPattern *string` to `ParamMapping`; add XML attributes to `xmlParamEntry`; add `normalizeEnumCSV` that trims the value and converts full-width commas without changing regex text. Use normalized values only when inserting enum fields.

- [ ] **Step 4: Run focused tests**

```bash
cd omcgo && go test ./internal/config/parammodel -run 'TestNormalizeEnumCSV|TestXMLParamEntryCarriesValueRules'
```

Expected: PASS.

### Task 2: Schema、Loader 刷新、repository 与 discovered 对账

**Files:**

- Create: `omcgo/migrations/000003_mml_param_mapping_value_rules.sql`
- Modify: `omcgo/internal/config/parammodel/loader.go`
- Modify: `omcgo/internal/config/parammodel/pg_repository.go`
- Modify: `omcgo/internal/config/parammodel/intersect.go`
- Modify: `omcgo/internal/config/parammodel/model.go`
- Test: `omcgo/internal/config/parammodel/loader_test.go`
- Test: `omcgo/internal/config/parammodel/pg_repository_test.go` 或现有 repository 测试

**Interfaces:**

- `param_mappings` 和 `discovered_param_mappings` 暴露 nullable `default_value` 与 `validation_pattern`。
- `ListMappingsByParamModel`、discovered 读写、intersect 行和 reconcile 更新携带所有取值规则。

- [ ] **Step 1: Write failing persistence/reconcile tests**

断言包含 min、max、defaultValue、validationPattern、枚举值和标签的 mapping 能完整写入；reload 会替换旧 builtin 值但保留 custom；discovered 会更新默认规则和 min/max，且保留 override 标记保护的范围值。

- [ ] **Step 2: Run focused tests and confirm RED**

```bash
cd omcgo && go test ./internal/config/parammodel -run 'Test.*(Reload|Mapping|Discovered|Enum|ValueRule)'
```

Expected: FAIL on missing columns/fields or missing behavior.

- [ ] **Step 3: Add migration 000003**

使用 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS ... text` 为两张映射表新增四列；Down 删除本迁移创建的四列。

- [ ] **Step 4: Extend batch insert and repository scans**

将 `default_value` 和 `validation_pattern` 加入 Squirrel insert、默认 mapping SELECT/Scan、discovered SELECT/Scan 和 discovered upsert。

- [ ] **Step 5: Extend discovered reconciliation and intersect inheritance**

discovered 更新时同步 enum values/labels、mirror、default value、validation pattern；保留现有 min/max 设备 override 行为；`buildIntersectRow` 复制所有新字段。

- [ ] **Step 6: Run focused tests and migration checks**

```bash
cd omcgo && go test ./internal/config/parammodel -run 'Test.*(Reload|Mapping|Discovered|Enum|ValueRule)'
bash scripts/check-migrations.sh
```

Expected: focused tests PASS and migration numbering check PASS.

### Task 3: 参数模型 enriched 查询与 MML API 契约

**Files:**

- Modify: `omcgo/internal/mml/admin_repository.go`
- Modify: `omcgo/internal/mml/console_service.go`
- Modify: `omcgo/internal/mml/console_service_test.go` 和/或 `admin_repository_test.go`
- Modify: `omcmb/frontend-core/src/types/mmlConsole.ts`
- Modify: `omcmb/frontend-core/src/services/api/mmlApi.ts`

**Interfaces:**

- Backend `SubFieldDTO` 暴露 `default_value`、`validation_pattern` 和 `enum_options`。
- `ListEnrichedByCommand` 使用解析出的 `param_model_id` 读取模型规则，同时保留 admin fallback。

- [ ] **Step 1: Write failing backend tests**

增加 service 测试，证明 product-class 请求将 ParamModel ID 传入 enriched lookup，并返回模型级 min/max/default/pattern/enum；增加 admin-context 测试，证明标准树回退仍有效。

- [ ] **Step 2: Run focused tests and confirm RED**

```bash
cd omcgo && go test ./internal/mml -run 'Test.*(SubField|ParamModel|ValueRule|Enriched)'
```

Expected: FAIL because service currently passes nil and DTO lacks fields.

- [ ] **Step 3: Implement model-aware SQL enrichment**

按 `sp.standard_path` 和 `param_model_id` 增加模型 mapping lookup；模型 min/max 非空时优先，否则使用标准树 min/max；由规范化 CSV 构建 enum options，缺少 label 时使用 value。

- [ ] **Step 4: Pass ParamModelID from ConsoleService**

在 product-class 和 device-key 分支保存解析出的 ID，传入 `ListEnrichedByCommand`，保留已有 supported-path 交集过滤。

- [ ] **Step 5: Map the API contract into frontend-core**

增加 backend/camelCase 类型，并映射 `validationPattern`、`enumOptions`、模型级 default/min/max。

- [ ] **Step 6: Run focused tests**

```bash
cd omcgo && go test ./internal/mml -run 'Test.*(SubField|ParamModel|ValueRule|Enriched)'
cd ../omcmb && npm run typecheck
```

如果 frontend 测试脚本名称不同，以 `package.json` 的现有脚本为准并记录实际命令。

### Task 4: 控制台枚举控件与提交校验

**Files:**

- Modify: `omcmb/webcode/src/pages/mml/Console/types.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/adapters.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/modParamValidation.ts`
- Modify: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx`
- Test: `omcmb/webcode/src/pages/mml/Console/modParamValidation.test.ts`
- Test: `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx`

**Interfaces:**

- `CommandParamPath` 携带 `enumOptions` 和 `validationPattern`。
- `validateModParamValue` 按 required、enum、type/range/length、pattern 顺序校验。

- [ ] **Step 1: Write failing validation tests**

```ts
expect(validateModParamValue(pathWithEnum, 'invalid')?.code).toBe('enumValue');
expect(validateModParamValue(pathWithPattern, 'abc')?.code).toBe('pattern');
expect(validateModParamValue(pathWithRange, '101')?.code).toBe('maxValue');
```

增加 modal 测试，证明枚举 path 渲染 Select、defaultValue 被选中、BOOLEAN 仍提交 true/false，并且校验错误只在提交后出现。

- [ ] **Step 2: Run focused tests and confirm RED**

运行现有前端测试脚本中对应两个测试文件。Expected: FAIL because new fields and validation codes are absent.

- [ ] **Step 3: Implement adapter and validation changes**

将 enum options 和 validation pattern 映射到 `CommandParamPath`；增加 enum/pattern 错误码；安全解析 slash-delimited JavaScript regex，非法 pattern 返回校验错误而不是抛异常；保留现有 min/max 和字符串长度校验。

- [ ] **Step 4: Implement modal rendering and initialization**

枚举 Select 优先于通用 Input；保留 BOOLEAN normalization；初始化优先使用 defaultValue，再走兼容 fallback；规则提示继续只在提交后显示。

- [ ] **Step 5: Run focused frontend tests and typecheck**

```bash
cd omcmb && npm run typecheck
```

Expected: focused tests PASS and typecheck exits 0.

### Task 5: 全量验证与 XML/数据库对账

**Files:**

- 只修改前述文件（若验证发现缺陷）。
- Optional Create: `docs/qa-report/mml-param-mapping-value-rules-20260721.md` 记录实测对账证据。

- [ ] **Step 1: Run backend build and complete test suite**

```bash
cd omcgo && go build ./... && go test ./...
```

- [ ] **Step 2: Run frontend typecheck and relevant tests**

```bash
cd omcmb && npm run typecheck
```

- [ ] **Step 3: Run XML-to-database reconciliation checks**

应用 migration 并在测试数据库 reload param-model 字典后，对 BLQ、BSC、MLQ、ENB_DEFAULT_098、ENB_DEFAULT_181 逐模型/路径比较 min/max/default/pattern/enum 数量和典型值，确认旧 builtin 规则不残留且 custom 行未删除。

- [ ] **Step 4: Verify browser behavior**

打开带产品上下文的 MML 控制台，检查真实 sub-field API 和 DOM：枚举 path 使用 Select、BOOLEAN 使用 true/false Select、默认值生效、非法输入只在提交后显示错误。

- [ ] **Step 5: Run final diff checks**

```bash
git diff --check
git status --short --branch
git diff --stat
```

记录环境限制，不把未执行的验证写成已通过。
