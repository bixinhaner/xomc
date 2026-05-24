# PRD — MML Catalog v2.3 Spec→DB Parser 工具（T-0169）

| 项 | 值 |
|---|---|
| PRD ID | F06-mml-catalog-spec-parser |
| Backlog | T-0169 |
| 功能域 | F06 (MML 控制台) + infra (omcctl) + admin (catalog 管理) |
| 优先级 | P2 |
| 状态 | DRAFT |
| 起草日 | 2026-05-24 |
| 起草人 | Claude（用户拍板 A 路径 — 见 T-0168 后续对话） |
| 关联 | T-0168 改造分析报告中识别出 parser 需求 |

---

## 1. 业务背景

### 1.1 痛点

- **规范文档驱动**：`omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md` (3485 行) 是 MML 命令与 path 映射的**权威源**。
- **现状脱节**：
  - DB `mml_commands` 表 190 条命令、`mml_command_sub_fields` 关联 ~ 2000+ 条（T-0123 落地）
  - `data/mml-catalog/cmcc-tdlte-v2.3.json` (16168 行) 是 v2.3 早期版本快照
  - spec md §R-2.4 71 个权威 `group_code` 经过 2026-05-21 v2 修订（新增「非可创建对象清单」+ 「同 group_code 合并」规则），与 DB 现状已经漂移
- **抽样发现真实 GAP**：
  - `Device.FaultMgmt.SupportedAlarm.{i}.*` 缺 MOD/ADD/RMV 3 个 op（仅有 LST）
  - `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*` 整组 4 op 全缺
- **人工补齐风险高**：
  - 全量 GAP 估算 10-30 条命令叶子 + 50-150 条 standardPath
  - 商用网管系统 10 万级设备规模，1 处拼写错误 = 全国蔓延 Fault 9005
  - 手工逐行 SQL 化错误率高，CI 难发现拼写漂移

### 1.2 解决方案

新增 omcctl 子命令 `omcctl mml import-spec-md`，自动化：

```
spec md  ─→  Parser  ─→  SpecCatalog (struct)
                                │
DB snapshot ─→ Loader ─┐        ↓
                       └──→  Differ  ─→  DiffReport (🔴/🟡/⚪)
                                          │
                                          ↓
                                ┌─────────┴───────────┐
                                ↓                     ↓
                       SQL Seed 生成器          JSON Catalog 生成器
                                ↓                     ↓
                  seed/000172_*.sql      data/mml-catalog/cmcc-tdlte-v2.3.json
```

## 2. 用户故事

### 2.1 MML 后端工程师 — 规范同步流水线
> 作为后端工程师，**当业务方更新规范文档（新增/修改命令组）时，我希望执行 `omcctl mml import-spec-md` 一条命令即可得到精确的差异报告 + 可直接 commit 的 seed/000172 + 同步更新的 catalog .json**，无需手工逐条 SQL 化和 diff，避免拼写错误。

### 2.2 SRE — 升级回归保护
> 作为 SRE，**我希望 parser 输出的 SQL 全部是 ON CONFLICT DO UPDATE 形态**（幂等），重启服务即应用变更，且 down 段精确清单可回滚，不污染 source='admin' 的用户自定义命令。

### 2.3 网管管理员 — 信任源可追溯
> 作为网管管理员，**我希望任何 catalog 变更都有 spec md 作为源凭据**（commit footer 引用 spec md path），便于排查"某命令为什么这样定义"。

## 3. 验收标准（Given-When-Then）

### GWT-1（Parser 解析正确性）
- **Given** spec md `cmcc-tdlte-southbound-data-model-v2.3.md` 文件
- **When** `omcctl mml import-spec-md --spec=<path> --dry-run`
- **Then**
  - 解析出 §R-2.4 表的 71 个 `(chapter, group_code, command_zh_name)` 三元组
  - 解析出 §SA-SR 各 H4 `#### 命令: <path>` 下的 path 集合（每条含 access / data_type / RW 标记）
  - 解析出 §R-3.2 非可创建对象黑名单
  - 71 条 command_zh_name 唯一性自检通过（命中违反 → 退出码非 0）

### GWT-2（DB Diff 准确性）
- **Given** parser 解析结果 + 当前 DB 状态
- **When** `omcctl mml import-spec-md --spec=<path> --diff-db=<dsn>`
- **Then**
  - 输出三类差异：🔴 缺失（new）/ 🟡 修改（existing target_paths 不一致）/ ⚪ 多余（DB 有但 spec 无；标记不删）
  - 抽样验证：
    - `SupportedAlarm` 应识别为 🔴 缺 MOD/ADD/RMV 3 条
    - `X2IpAddrMapInfo` 应识别为 🔴 全组 4 条 + 标准参数树缺 standardPath
    - 已对齐的 `DeviceInfo.SwUpgrade.*` 应识别为 ⚪ 无差异

### GWT-3（SQL Seed 生成质量）
- **Given** DiffReport 与 `--out=seed/000172_*.sql` 参数
- **When** parser 跑生成
- **Then**
  - 生成的 SQL **本地 goose up 通过**（migrate v171→172）
  - **down 段精确**：仅 DELETE 本 seed INSERT 的 `command_code` 清单，不动 source='admin' 行
  - **重跑幂等**：连续 2 次 goose up 不报错（ON CONFLICT 生效）
  - JSON 字段中文转义正确（特别 logical_name_i18n / command_name_i18n）
  - 所有 `(command_id, standard_path_id)` 关联满足表唯一约束 uq_command_param

### GWT-4（JSON Catalog 同步）
- **Given** parser 同时生成 .json
- **When** `--out-json=data/mml-catalog/cmcc-tdlte-v2.3.json`
- **Then**
  - 新 .json 结构与现有 catalog .json schema 一致
  - 71 个 group_code 全部覆盖
  - 标识 generated_from + spec_md_hash + generated_at 元数据

### GWT-5（错误处理）
- **Given** spec md 有 markdown 表格语法错误 / §R-2.4 同名命令 / 引用未在 standard_params 的 path
- **When** parser 跑
- **Then**
  - 解析失败 → 退出码 1，stderr 输出精确行号
  - command_zh_name 重名 → 退出码 2，stderr 列出冲突项
  - standardPath 未在 standard_params 命中 → **不阻断**生成（按 §R-2.5.2 加入失败清单 stderr 输出），同时在 SQL 中加 INSERT INTO standard_params 补齐该 path

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 说明 |
|---|---|---|---|---|
| 规范文档 | v2.3 (本任务覆盖) | — (留 future) | — (留 future) | 本任务**仅处理 CMCC TD-LTE v2.3**；CTCC/CUCC 规范文档不在本任务范围 |
| spec md 结构 | §R-2.4 71 条 + §SA-SR 详情 | (假设同结构) | (假设同结构) | parser 设计支持通过 `--carrier=cmcc/ctcc/cucc` 切换 spec md 路径 |
| param_version | `cmcc-td-lte-v2.3` | `ctcc-*` | `cucc-*` | DB `mml_param_versions.version_code` 已支持多版本 |
| 命令分组 | chapter:SA..SR | (待业务方定义) | (待业务方定义) | 本任务不预制 CTCC/CUCC 分组 |

**结论**：本任务**仅覆盖 CMCC**；parser 设计为**多运营商可扩展**，通过 `--carrier` flag 切 spec 路径，但本期不实施 CTCC/CUCC。

## 5. 非目标（Out of Scope）

- **不解析 CTCC/CUCC 规范文档**（留独立 task）
- **不动 mml_command_groups schema**（18 chapter 一级结构保持，规范 v2.3 未增章节）
- **不实现 R-2.5 改造**（target_paths JSONB → tree_node_refs[] 软外键的 schema 演进）；仍写 target_paths JSONB，符合 DB 现状
- **不删 source='admin' 用户自定义命令**（⚪ 多余项标记不删）
- **不自动 commit/push**（parser 仅生成文件，由用户 commit）
- **不引入 markdown 库依赖**（用 stdlib regexp + bufio.Scanner 解析，避免第三方依赖膨胀）

## 6. 依赖

| 类型 | 标识 | 说明 |
|---|---|---|
| Backlog | T-0098 ✅ | standard_params / mml_categories / param_mappings schema 就绪 |
| Backlog | T-0123-P0 ✅ | mml_command_sub_fields (standard_path_id FK) schema 就绪 |
| Backlog | T-0123-P3 ✅ | mml admin Tab 已能消费本 parser 的产物 |
| 资产 | `omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md` | spec md 权威源 |
| 资产 | `omcgo/cmd/omcctl/` | cobra 子命令注册框架 |

无未结依赖。

## 7. 度量

- parser 解析速度：< 2s on 3485 行 spec md
- DB diff 速度：< 5s on 当前规模（2001 standard_params + 190 commands + 数千 sub_fields）
- 生成 SQL 大小：估算 ~ 200-500 KB（按 GAP 30 命令 + 200 path 估）
- 重跑幂等：连续 N 次 goose up 不报错（CI 反退化测试）

## 8. 风险

| ID | 风险 | 等级 | 缓解 |
|---|---|---|---|
| R-NEW-T0169-1 | Markdown 表格非严格结构（缺失列 / 列宽偏移）导致 parser 误判 | 中 | 加 schema validator：每张表必须 6/7 列；缺列 → 退出码 3 + 精确行号；fixture md 片段单测覆盖 |
| R-NEW-T0169-2 | §R-2.4 71 条 vs §SA-SR 详情不一致（spec 自身漂移） | 中 | parser 内置 cross-check：71 条权威表 vs SA-SR H4 标题集 → mismatch 输出 warning 列表 |
| R-NEW-T0169-3 | 生成的 SQL 列名/约束与 schema 不一致（CLAUDE.md §5.5.4 历史踩坑） | 低 | parser 内置 schema 常量（与 migrations/000095/000058 对齐），单测覆盖 SQL 模板 |
| R-NEW-T0169-4 | 跨章节 group_code 合并（v2.3 新引入）误处理 | 中 | parser 显式处理 §R-2.4 "SH→SF 合并 1 条" 注释，单测验证 |

## 9. 实施路线

| 阶段 | 制品 | 工作量 |
|---|---|---|
| S2 设计 | 本 PRD 附录"## 设计备忘" | 0.2d |
| S3 实施 | `omcgo/internal/mml/specparser/` 包（parser + differ + sql_gen + json_gen + tests） + `omcgo/cmd/omcctl/cmd_mml_import.go` 子命令 + 真实跑通 spec md 生成 seed/000172 | 1.0d |
| S4 验证 | go test -race / omcctl 跑通 / migrate up+down / 抽样核对生成的 SQL | 0.2d |
| S5 审查 | self-review + DoD | 0.05d |
| S6 提交 | 单 commit（parser 工具 + spec md 引用 + 生成的 seed） | 0.05d |
| S7 收尾 | backlog 回写 + 删除 T-0168 留下的 sample seed | 0.05d |
| **合计** | | **~1.5d** |

---

## 设计备忘（S2 阶段产出 — 2026-05-24）

### S2.1 模块组织

```
omcgo/
├── cmd/
│   └── omcctl/
│       └── cmd_mml_import.go        # 新增：import-spec-md cobra 子命令
└── internal/
    └── mml/
        └── specparser/              # 新增：parser 独立包
            ├── parser.go            # Markdown → SpecCatalog
            ├── parser_test.go       # fixture md 片段单测
            ├── differ.go            # SpecCatalog × DB → DiffReport
            ├── differ_test.go
            ├── sql_gen.go           # DiffReport → seed SQL
            ├── sql_gen_test.go
            ├── json_gen.go          # SpecCatalog → catalog JSON
            ├── json_gen_test.go
            ├── types.go             # 所有数据结构
            └── fixtures/
                ├── spec_sample.md   # 测试用 spec md 片段
                └── expected_diff.json
```

### S2.2 核心数据结构

```go
// SpecCatalog 是解析 spec md 的结果集
type SpecCatalog struct {
    Version    string              // "cmcc-td-lte-v2.3"
    SourceMD   string              // spec md 绝对路径
    SourceHash string              // spec md sha256
    Groups     []*SpecGroup        // 71 个 group（按 SA→SR 顺序）
    Blacklist  []string            // §R-3.2 非可创建对象 group_code 清单
}

type SpecGroup struct {
    Chapter         string              // "SA" / "SB" / ...
    GroupCode       string              // "Device.DeviceInfo.*"
    CommandZhName   string              // "设备基本信息"（§R-2.4 权威）
    HasInstance     bool                // group_code 含 {i} → 可派生 ADD/RMV（叠加 Blacklist 判断）
    Paths           []*SpecPath
}

type SpecPath struct {
    StandardPath  string  // "Device.DeviceInfo.UserLabel"
    ParamName     string  // "UserLabel"
    ChineseName   string  // "用户友好名"
    Access        string  // "READ_ONLY" / "READ_WRITE" / "WRITE_ONLY"
    DataType      string  // "string" / "unsignedInt" / "boolean" / ...
    MinValue      *int64  // 类型字符串中如 unsignedInt[1:5] 提取的下界
    MaxValue      *int64  // 同上上界
    MaxLength     *int    // string(64) 中的 64
}

// DBSnapshot 现状（从 PG 查或从 backup .sql 读）
type DBSnapshot struct {
    StandardParams        map[string]*StandardParamRow      // standard_path → row
    Commands              map[string]*MMLCommandRow         // command_code → row
    CommandSubFields      map[string]map[string]*SubFieldRow // command_code → standard_path → row
}

// DiffReport 三类差异
type DiffReport struct {
    NewStandardParams     []*SpecPath                       // 🔴 spec 有 DB 无
    NewCommands           []*SpecCommand                    // 🔴 spec 派生命令 DB 无
    UpdatedCommands       []*UpdatedCommand                 // 🟡 target_paths 不一致
    NewSubFieldLinks      []*SubFieldLink                   // 🔴 关联缺失
    OrphanCommands        []string                          // ⚪ DB 有但 spec 无（不删，仅标）
    OrphanLinks           []string                          // ⚪ 关联多余
    Summary               DiffSummary
}

type SpecCommand struct {
    GroupCode       string
    OperationType   string    // LST / MOD / ADD / RMV
    CommandCode     string    // 派生：op + " " + logical_code，如 "LST X2_IP_ADDR_MAP"
    LogicalCode     string    // 从 group_code 派生：去 "." 转 "_" 大写
    CommandZhName   string
    TargetPaths     []string  // standardPath 列表（LST=全部，MOD=仅 RW，ADD/RMV=object_name）
    RPCMethod       string    // GetParameterValues / SetParameterValues / AddObject / DeleteObject
}
```

### S2.3 Parser 算法（关键步骤）

```
1. 打开 spec md，按行读
2. 状态机识别四种段落：
   - §R-2.4 表格 (line 134-206 起，markdown table format) → 填充 71 个 SpecGroup 框架
   - §R-3.2 非可创建对象清单（pattern: line 含 "non-creatable" 或 "非可创建"）→ 填 Blacklist
   - §SA-SR 章节（pattern: `^## S[A-R] - `）→ 切到对应 SpecGroup
   - `#### 命令: <path> 📖📝` H4 → 切到对应 group 的 path 收集
3. 在 H4 下读 markdown table（7 列：# / TR-098 / TR-181 / 参数名 / 中文名 / 权限 / 类型）
   - 提取 TR-181 列 → SpecPath.StandardPath
   - 权限列 "📝 RW" → READ_WRITE / "📖 R" → READ_ONLY
   - 类型列 "unsignedInt[1:5]" → DataType=unsignedInt, MinValue=1, MaxValue=5
   - 类型列 "string(64)" → DataType=string, MaxLength=64
4. Cross-check：§R-2.4 表的 71 个 group_code 必须每个在 §SA-SR 中能找到 H4 标题；mismatch → warning
5. 71 条 CommandZhName 唯一性 check → 重名 → exit 2
```

### S2.4 Differ 算法

```
对 SpecCatalog.Groups 每个 group：
  1. 派生 op 集合：
     - LST: 恒生成
     - MOD: 若 group.Paths 含至少一条 Access in (READ_WRITE, WRITE_ONLY)
     - ADD/RMV: 若 group.HasInstance && group.GroupCode 不在 Blacklist
  2. 对每个 op 派生 SpecCommand：
     - LogicalCode = group_code 去 "Device." 前缀 + 替换 "{i}." → "" + "." → "_" + 大写
     - CommandCode = op + " " + LogicalCode
     - TargetPaths:
       - LST: group.Paths 全部 standardPath
       - MOD: group.Paths Filter Access ∈ (READ_WRITE, WRITE_ONLY) 的 standardPath
       - ADD/RMV: [group.GroupCode 去 "{i}.*" 后的 object_name]
  3. 对比 DBSnapshot.Commands[CommandCode]:
     - 不存在 → NewCommands +1
     - target_paths 不一致 (set equality) → UpdatedCommands +1
  4. 对每个 (SpecCommand, standardPath) 组合 vs DBSnapshot.CommandSubFields → NewSubFieldLinks
对 SpecCatalog.Groups 每个 standardPath vs DBSnapshot.StandardParams → NewStandardParams

⚪ Orphan check：DBSnapshot 中 source='standard' 但 SpecCatalog 不覆盖 → OrphanCommands
```

### S2.5 SQL 生成约束（避免 CLAUDE.md §5.5 历史踩坑）

| 约束 | 实施 |
|---|---|
| **§5.5.4 INSERT 列与 schema 匹配** | parser 内置 schema 常量字符串，与 migrations/000058、000095 对齐；SQL 模板用 Go const 而非动态拼接 |
| **§5.5.6 UUID 格式** | 不预生成 UUID 字面量；全部用 `gen_random_uuid()` 或 `(SELECT id FROM ...)` 子查询 |
| **§5.5.7 ON CONFLICT** | 全部 INSERT 加 `ON CONFLICT (uniq_key) DO UPDATE SET ... updated_at=NOW()` |
| **§5.5.8 Down 完整性** | down 段精确列出 INSERT 的 command_code 清单（parser 输出 .down.sql 旁路文件） |
| **§5.5.4 JSON 转义** | JSON 字段（command_name_i18n / logical_name_i18n / label_i18n）用 Go encoding/json marshal 然后 SQL 字符串转义（双重单引号） |

### S2.6 omcctl 子命令设计

```bash
omcctl mml import-spec-md \
  --spec=<path>                # spec md 路径（必填）
  --diff-db=<dsn>              # PG DSN（可选；不提供则 dry-run，不查 DB）
  --carrier=cmcc               # 默认 cmcc；future 支持 ctcc/cucc
  --version=cmcc-td-lte-v2.3   # 写入 mml_param_versions.version_code
  --out=<seed_path>            # 输出 seed SQL（必填）
  --out-json=<json_path>       # 输出 catalog JSON（可选）
  --dry-run                    # 不写文件，仅打印 diff summary
  -v                           # verbose 输出
```

### S2.7 Carrier 差异点

**本任务无运营商差异**（仅处理 CMCC v2.3）。但 parser 设计为多运营商可扩展（`--carrier=` flag），future 任务可零修改加 CTCC/CUCC spec。

### S2.8 观测埋点

- 不引入 Prometheus 指标（一次性 CLI 工具）
- 全程 stderr 结构化日志（zap）：
  - INFO: parser 进度（解析 §R-2.4 完成 / 解析 §SA-SR 完成 / diff 完成 / SQL 生成完成）
  - WARN: 非阻塞警告（§R-2.5.2 失败清单 / cross-check mismatch）
  - ERROR: 阻塞错误（schema validator 失败 / 唯一性自检失败）+ 退出码

### S2.9 测试设计

```
parser_test.go:
  - TestParser_R24Table_Happy        # 71 条权威表正确解析
  - TestParser_R24Table_DuplicateName # 同名 → exit 2
  - TestParser_SAR_PathTable         # H4 命令表正确解析
  - TestParser_TypeWithRange         # "unsignedInt[1:5]" 解析
  - TestParser_AccessIcon            # 📖 R / 📝 RW 识别
  - TestParser_R32Blacklist          # §R-3.2 非可创建清单识别

differ_test.go:
  - TestDiffer_NewCommand_X2         # X2IpAddrMap 全缺识别为 4 op NEW
  - TestDiffer_NewOps_SupportedAlarm # SupportedAlarm 现有 LST，识别 MOD/ADD/RMV NEW
  - TestDiffer_UpdatedTargetPaths    # 已存在命令 target_paths 不一致 → Updated
  - TestDiffer_OrphanCommand         # DB source='standard' 但 spec 无 → Orphan
  - TestDiffer_OrphanLink            # sub_field 多余链接

sql_gen_test.go:
  - TestSQLGen_InsertStandardParam   # 列名 + ON CONFLICT 正确
  - TestSQLGen_InsertCommand_UUIDRef # group_id 用 (SELECT id FROM ...)
  - TestSQLGen_InsertSubField_JOIN   # sub_field INSERT 用 JOIN 查 id
  - TestSQLGen_DownSection           # down 段精确 DELETE
  - TestSQLGen_I18nEscape            # 中文 JSON 转义

集成测试（手工）：
  - 跑 omcctl mml import-spec-md 真实 spec md → 生成 seed/000172_*.sql
  - migrate up → 验证 row count 增量符合 diff report
  - migrate down → 验证 row count 回退
  - migrate up → 验证幂等（重跑无 error）
```

### S2.10 出口门核查

| 出口门 | 状态 |
|---|---|
| 接口契约明确 | ✅ S2.2 数据结构 + S2.6 CLI flag |
| 迁移草案 | ✅ S2.5 SQL 约束 + 输出文件 seed/000172_*.sql |
| Carrier 差异点列出 | ✅ S2.7 无差异 |
| 观测埋点名字列出 | ✅ S2.8 |
| 待定点 < 3 | ✅ 0 待定点 |

---

**版本历史**：
- 2026-05-24 v1.0 起草（S0 七要素 + S2 设计备忘合并）
