# MML 老交互恢复方案 v2

> **状态**：✅ **APPROVED** 2026-05-14（7 项主决策 + 3 项收尾决策 均已审决）
> **日期**：2026-05-14
> **作者**：架构
> **关联**：v1 同名（已废弃）、Sprint A `docs/design/mml-rebuild-plan-20260513.md`

---

## 0. v2 关键变更（相对 v1）

| # | v1 假设 | v2 修订（依据用户审核回复） |
|---|--------|----------------------------|
| 1 | "同类型合并"=允许 UI 队列里串多条 LST | **= 一个 logical command 本身就打包多个 sub-field**（`LST DEVICE_INFO` 内含 Model Name / IP / MAC 等 14 项；它们**不是** 14 条命令）。Model Name / IP / MAC 是 **sub-field**，不是 command。 |
| 2 | 引入第二份 `mml-command-catalog.xml` 资产 | **不引入新 XML**。利用现有 `mml_commands` / `mml_param_groups` / `mml_params` 表（Sprint A Loader 已写入），通过 **admin UI 增删改 + 一次性 SQL 迁移**完成 catalog 整理 |
| 3 | catalog 分运营商目录 `cmcc/ctcc/cucc/` | **通用一份**，不按运营商拆 |
| 4 | ParameterPath Command 视图可合并到 MMLEditor 高级模式 | **保留双 Tab**（Control Panel / ParameterPath Command），不合并 |
| 5 | 自定义命令（PrivateTemplate / PublicTemplate）放二期 | **现有 `mml_custom_commands` 已就绪**，只做集成 |
| 6 | 提到"如方案推进超预算可回退 Sprint B" | **不回退**，向前推进 |
| 7 | XML 启动期 Loader 不变 | **将 standard-model.xml 一次性导入 DB 表**，禁用启动期 reloader（DB 成为单一权威源） |
| 8 | 多语言只在 group/command 层 | **全字段多语言**（group name / command logical name / sub-field label / constraint text / explanation）。当前要求中英两种，预留扩展。 |
| 9 | 未明确 path 元数据 | **明确加 5 类元数据**到 `mml_params`：access_type / is_object / supports_add / supports_delete / change_applies。**前端 UI 据此差异化呈现**：只读 path 灰显、可写 path 显输入框、可删 object 显删除图标等 |

---

## 0.5 运营商差异矩阵 + 非目标（S0 集中要素）

### 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 备注 |
|------|------|------|------|------|
| MML catalog 内容 | 通用一份 | 通用一份 | 通用一份 | §0 决策 #3 锁定，**无差异** |
| 命令分组层级 | 通用 | 通用 | 通用 | BSC Configuration / eNB Configuration / gNB Configuration 跨运营商共享 |
| sub-field 路径 | 通用（standardPath） | 通用 | 通用 | 任何 privatePath 差异走 T-0098 `parammodel.Translator` 透明翻译，不在 MML 层处理 |
| RBAC 权限 | `mml.catalog.manage` 单点 | 同 | 同 | §12 决策 3 锁定 |
| 多语言文案 | zh-CN / en-US 全字段 | 同 | 同 | §3 P3 锁定，未来运营商如需"小语种"挂 `*_i18n` JSONB 即可 |
| 默认勾选行为 | LST 全勾选 / MOD 仅 READ_WRITE | 同 | 同 | §1.3 / §1.4 实测对齐 |

**结论**：本方案三家运营商**完全一致**，catalog 通用化是核心决策（§0 #3）。若未来某运营商需要差异（如 CMCC 专有命令），通过 admin UI 创建 source='admin' 行 + i18n 元数据即可；不引入 `cmcc/ctcc/cucc/` 目录分裂。

### 非目标（明确不做）

1. **不在协议层合并 RPC**：多条 statement 各自走独立 GPV/SPV，不在 ACS 内合并多个 statement 的 sub-field 到一次 RPC（用户决策 #1 锁定语义）
2. **不引入新 XML catalog 资产**：废弃 v1 设计的 `mml-command-catalog.xml`（§0 #2）
3. **不分运营商目录**：catalog 不拆 `cmcc/ctcc/cucc/`（§0 #3）
4. **不合并 Control Panel 与 ParameterPath Command 视图**：保持双 Tab（§0 #4 / #5）
5. **不回退 Sprint A/B 既有能力**：scheduler / sequencer / fanout / result_aggregator / script_parser 全保留（§0 #6）
6. **不实施真 LDAP/SSO RBAC 改造**：复用现有 RBAC 体系，新增单一权限点 `mml.catalog.manage` 即可（§12 决策 3）
7. **不做命令执行历史回滚 UI**：task 表已记录历史，本方案不在交互层做回滚（v1 §10 沿用）
8. **不实施 ACS 端协议改造**：GPV/SPV/AddObject/DeleteObject 既有，本方案只重排 payload
9. **不破坏 standard-model.xml 文件本身**：XML 保留作 import 工具的输入与回溯证据，运行期不读
10. **不在本任务做客户化/二次开发 plugin 架构**：catalog 由 admin UI 管理足够；插件式扩展不在范围

### 度量（验收硬指标）

复用 §11 DoD 清单作为度量：

- **数据层**（§11.1）：6 条字段/表存在性 + 1 条迁移可逆 + 1 条多语言覆盖率 ≥60%
- **API 层**（§11.2）：5 条端点单测 + 1 条权限 403 + tree 数据正确性
- **前端**（§11.3）：10 条交互验收（勾选 / textbox 双向 / OnReboot 二次确认 / READ_ONLY 过滤等）
- **E2E**（§11.4）：226 + 14 = 240 断言全过 + webcode-v2/v3 typecheck PASS

---

## 1. 老系统交互全景（playwright 实测，与 v1 一致）

登录 `http://172.21.175.129:8081/sys/login/userLoad.htm`（admin / OMC@123456）→ Maintenance > MML：

### 1.1 三栏 + Step 1-4

```
┌───────────────┬────────────────────┬──────────────────────────────┐
│ Step 1        │ Step 2             │ Step 3                       │
│ 设备列表       │ MML List 命令树    │ Control Panel / ParamPath 视图│
│ ☐ F4F1F7...   │ ▼ BSC Configuration│ [LST DEVICE_INFO:lstId={...}]│
│ ☐ 0D59FB...   │   ▼ Basic Info     │  ☑ Model Name     Device.... │
│ ☐ 1EBECA...   │     • LST DEV...   │  ☑ System Uptime  Device.... │
│ Batch Input   │     • MOD DEV...   │  ...                         │
│               │   ▶ BTS            │           [DO]               │
│               │ ▼ Customized       │ ── Result (Step 4) ──────────│
└───────────────┴────────────────────┴──────────────────────────────┘
```

### 1.2 命令树结构

```
MML List
├── BSC Configuration                  ← 一级 group
│   ├── Basic Info                     ← 二级 group
│   │   ├── Device info(LST DEVICE_INFO)   ← 命令叶子
│   │   └── Device info(MOD DEVICE_INFO)
│   ├── BTS
│   │   ├── BTS Info(LST/MOD/ADD/RMV BTS_INFO)
│   │   ├── Trx(LST/MOD/ADD/RMV BTS_TRX)
│   │   └── Ts(LST/MOD BTS_TS)
│   ├── Msc / Cs7 / Mgw / DNS / keepalived / Handover / Network
└── Customized
    ├── PrivateTemplate                ← `mml_custom_commands` (scope='private')
    │   ├── admin → 123 / test / ...
    │   └── wangyunqi123 → ...
    └── PublicTemplate                 ← `mml_custom_commands` (scope='public')
```

### 1.3 LST 子字段勾选式（实测样本：Device info）

顶部 MML textbox：

```
LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_SYS_TIME,LTE_GSM_IP,
LTE_GSM_MAC,LTE_GSM_SOFTWARE,LTE_GSM_HARDWARE,LTE_GSM_MME_STATUS,
DEVICEGSM_MCC,DEVICEGSM_MNC,LTE_BTSNUM,BSC_ENCRYPTION,
DEVICEGSM_TIMERNETT3212,DEVICEGSM_NRIBITLEN,DEVICEGSM_NRINULLADD};
```

下方 14 行勾选列表（每行：☑ 显示名 + MML 内部 code + TR-069 path），默认全勾，勾选 / textbox 双向绑定。

### 1.4 MOD 输入式 + range 提示

7 行 input，每行有 `range:001-999` / `Integer,range:0-1` 等约束文本，仅暴露 READ_WRITE 子字段。

### 1.5 MML 字符串语法

```
<OP> <COMMAND_CODE>[:<paramKey>={<subFieldCode>,...}][:<field>=<val>,...];
```

- 多条用 `;` 分隔可串行执行（textbox 上方原文提示）
- LST：`lstId={CODE,...}`
- MOD：`field=value,...`
- ADD / RMV：参照子字段集合 / instance index

---

## 2. 现状诊断（昨天 Sprint A 之后的回归）

| 层 | 现状 | 问题 |
|----|------|------|
| **DB** | `mml_command_params_rel` 已 DROP（000090），命令 → sub-field 关系只剩 `target_paths JSONB` | 丢了 (mml_code, label_i18n, sort_order, default_selected, is_required) 等 sub-field 决定 UI 形态的关键字段 |
| **DB** | `mml_params` 只有 is_writable boolean | 没区分 param vs object；没 supports_add / supports_delete / change_applies；UI 无法差异化呈现 |
| **DB** | `mml_param_versions` 仍存 STANDARD 单条 + 多个 product 版本 | 老 24 条产品版本数据已硬删，但版本号体系还在 |
| **Loader** | `mmlstandardloader` 启动期读 XML → UPSERT | 用户要求"一次性入库，启动期不再读 XML" |
| **后端** | Handler 缺：group 树、sub-fields、render/parse MML、批量 statements 执行；admin CRUD（catalog 管理）整体缺失 | UI 没法绑定 |
| **前端** | `ParamPathPanel` + `ParamFormRenderer`（适合 path 编辑 / 通用表单） | 不是老系统的"勾选式 + 输入式 + MML 双绑"形态 |
| **前端** | 没有 `MmlEditor` / `SubFieldChecklist` / `SubFieldInputList` 组件 | 三栏第 3 列没"对的"组件 |

---

## 3. 设计原则（v2）

| # | 原则 | 含义 |
|---|------|------|
| P1 | **DB 是 catalog 的唯一权威源** | XML 仅作一次性导入；运行期不再读 XML；不引入新 XML 资产 |
| P2 | **复用现有表 + 最小列增量** | 复活 `mml_command_params_rel`（重命名 `mml_command_sub_fields`）；扩展 `mml_params`、`mml_commands`、`mml_param_groups` 各加 3-5 列 |
| P3 | **多语言贯穿全字段** | `*_i18n JSONB` 在 group / command / sub-field / constraint / explanation 全部存在；当前注入 zh-CN、en-US 两种 |
| P4 | **元数据驱动 UI 渲染** | sub-field 上 5 类 flag（writable / object / can_add / can_delete / change_applies）决定前端用 input / checkbox / "+" / "🗑" / 提示 badge 哪一种 |
| P5 | **管理界面是一等公民** | Catalog 通过 admin UI 增删改：建/删 group、改 command 名称、调整 sub-field 顺序、设可见性。提供 CRUD API |
| P6 | **同类型合并 = 一命令多 sub-field** | 不在协议层合 GPV/SPV，不在 UI 队列合并；保持 Sprint A 的"一命令一 op_type"结构，但**单条 LST 命令本身就内含 N 个 sub-field**，渲染时一次性 GPV 这 N 个 path |
| P7 | **批量 = 设备 fanout + statements 串联** | 多设备 × 多条 statement 拓扑：N × M 个 device_task；statement 内部按 sub-field 集成一次 RPC |
| P8 | **Sprint A/B 既有能力保留** | scheduler / sequencer / fanout / result_aggregator / script / custom commands 不破坏；payload schema 兼容老调用方 |

---

## 4. 数据模型（v2 最终版）

### 4.1 扩展 `mml_params`（path 元数据 + i18n 强化）

```sql
ALTER TABLE mml_params
    -- TR-069 标准的 access 三态（替代 is_writable 单 bool；is_writable 派生为 generated column）
    ADD COLUMN IF NOT EXISTS access_type VARCHAR(20) NOT NULL DEFAULT 'READ_ONLY'
        CHECK (access_type IN ('READ_ONLY','READ_WRITE','WRITE_ONLY')),

    -- 该 row 描述的是 object 节点（path 以 . 结尾）还是 leaf param
    ADD COLUMN IF NOT EXISTS is_object BOOLEAN NOT NULL DEFAULT false,

    -- 仅 is_object=true 有效：是否支持 AddObject 创建实例
    ADD COLUMN IF NOT EXISTS supports_add BOOLEAN NOT NULL DEFAULT false,

    -- 仅 is_object=true 有效：是否支持 DeleteObject 删除实例
    ADD COLUMN IF NOT EXISTS supports_delete BOOLEAN NOT NULL DEFAULT false,

    -- 生效时机：Immediate / OnReboot
    ADD COLUMN IF NOT EXISTS change_applies VARCHAR(20) NOT NULL DEFAULT 'Immediate',

    -- 范围约束的 UI 提示文本（多语言），如 "Integer, range:0-1" / "整数，范围 0-1"
    ADD COLUMN IF NOT EXISTS constraint_text_i18n JSONB NOT NULL DEFAULT '{}',

    -- 是否允许 admin 在 catalog 管理界面删除此 row（保护内置标准 path）
    ADD COLUMN IF NOT EXISTS catalog_protected BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_mml_params_access_type ON mml_params(access_type);
CREATE INDEX IF NOT EXISTS idx_mml_params_is_object ON mml_params(is_object);
```

**字段语义对照**：

| 列 | XML 源 | UI 用途 |
|----|--------|---------|
| `access_type` | `<param access="READ_ONLY">` | LST：只读浅灰；MOD：只读项过滤掉、只显 READ_WRITE 输入 |
| `is_object` | `<object>` vs `<param>` | object 表示是 instance 容器，UI 显 "+" 添加 / 列出实例 |
| `supports_add` | `is_object && access_type='READ_WRITE'` | 控制 "+" 按钮显隐 |
| `supports_delete` | 同上 + 业务规则 | 控制实例行的 "🗑" 显隐 |
| `change_applies` | `<param changeApplies="OnReboot">` | MOD 提交前 toast："本字段生效需重启"，并要求二次确认 |
| `constraint_text_i18n` | 综合 min/max/type 渲染 | MOD 输入框右侧的灰字提示 |
| `catalog_protected` | 标准 import 默认 true | admin UI 上"删除"按钮在 protected 行禁用 |

### 4.2 复活 `mml_command_sub_fields`（替代被 DROP 的 `mml_command_params_rel`）

```sql
CREATE TABLE mml_command_sub_fields (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id       UUID NOT NULL REFERENCES mml_commands(id) ON DELETE CASCADE,
    param_id         UUID NOT NULL REFERENCES mml_params(id)   ON DELETE RESTRICT,

    -- 老系统 MML 字符串内部使用的 code（命令上下文相关，可与 param_code 不同）
    -- 如 path=Device.DeviceInfo.X_COM_MODULE_TYPE，在 DEVICE_INFO 命令里 mml_code=LTE_GSM_MODEL_NAME
    mml_code         VARCHAR(100) NOT NULL,

    -- 在该命令中显示的 sub-field 标签（与 mml_params.name_i18n 区分；可覆盖 param 默认 label）
    label_i18n       JSONB NOT NULL DEFAULT '{}',

    -- LST：默认是否勾选
    default_selected BOOLEAN NOT NULL DEFAULT true,

    -- MOD/ADD：本字段是否必填
    is_required      BOOLEAN NOT NULL DEFAULT false,

    sort_order       INT NOT NULL DEFAULT 0,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_command_mml_code UNIQUE (command_id, mml_code),
    CONSTRAINT uq_command_param    UNIQUE (command_id, param_id)
);

CREATE INDEX idx_mml_command_sub_fields_command ON mml_command_sub_fields(command_id, sort_order);
CREATE INDEX idx_mml_command_sub_fields_param   ON mml_command_sub_fields(param_id);
```

**关键设计点**：

- 一条 sub-field row = (command, param) 组合 + 命令上下文的 mml_code 和 label 覆盖
- `mml_code` 仅在该命令内唯一（unique constraint 是 `(command_id, mml_code)`）
- `param_id` 指向 `mml_params` —— path / value_type / access 等通用属性走 join 取
- `label_i18n` 是命令上下文的别名（如 `Device.DeviceInfo.X_COM_MME_Status` 在 DEVICE_INFO 命令叫 "MSC Status"，可能在其他命令叫别的）；为空时 fallback 到 `mml_params.name_i18n`

### 4.3 扩展 `mml_commands`（逻辑命令分组）

```sql
ALTER TABLE mml_commands
    -- 老系统命令叶子前缀（去掉 op 的"裸"逻辑名）
    -- 例：command_code = 'LST_DEVICE_INFO' 时，logical_code = 'DEVICE_INFO'，logical_name_i18n = {"en":"Device info","zh":"设备信息"}
    ADD COLUMN IF NOT EXISTS logical_code VARCHAR(100),
    ADD COLUMN IF NOT EXISTS logical_name_i18n JSONB NOT NULL DEFAULT '{}',

    -- 是否标准导入（区分 admin 自建 vs XML 导入）；admin UI 上控制"删除"按钮显隐
    ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'admin',  -- standard / admin
    ADD COLUMN IF NOT EXISTS catalog_protected BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_mml_commands_logical_code ON mml_commands(logical_code);
```

**派生关系**：

```
mml_commands.command_code         = 'LST_DEVICE_INFO'    (Sprint A 原命名)
mml_commands.logical_code         = 'DEVICE_INFO'         (NEW，去掉 op 前缀)
mml_commands.logical_name_i18n    = {"en":"Device info","zh":"设备信息"}
mml_commands.operation_type       = 'LST'
mml_commands.command_name_i18n    = {"en":"Device info(LST DEVICE_INFO)", ...}   (Sprint A 已有，显示侧拼接好)
```

> 树叶子节点 label 渲染 = `command_name_i18n` 直接展示；同 `logical_code` 的多条记录通过 `logical_name_i18n` 在 admin 管理界面分组到"同一逻辑命令"下。

### 4.4 扩展 `mml_param_groups`（i18n 强化 + 管理保护）

```sql
ALTER TABLE mml_param_groups
    -- name_i18n 已在 000090 加；这里只加 source / protected
    ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'admin',
    ADD COLUMN IF NOT EXISTS catalog_protected BOOLEAN NOT NULL DEFAULT false;
```

> `parent_id` 在 000090 已 DROP 替换为 `path` ltree；保持 ltree 实现二级嵌套。

### 4.5 不动的表

- `mml_scripts` / `mml_tasks` — Sprint B 任务体系不变
- `mml_custom_commands` — 自定义命令仅集成，零字段改动
- `mml_param_versions` — 版本号体系保留；STANDARD 版本作为系统内置版本
- `mml_group_param_rel` — group → params 关系保留

### 4.6 新增迁移文件

```
migrations/000095_mml_command_catalog_v2.sql
    - ALTER mml_params: 加 access_type / is_object / supports_add / supports_delete / change_applies
                       / constraint_text_i18n / catalog_protected
    - CREATE mml_command_sub_fields 表
    - ALTER mml_commands: 加 logical_code / logical_name_i18n / source / catalog_protected
    - ALTER mml_param_groups: 加 source / catalog_protected
    - 触发器：command_sub_fields 变更 → 重算 mml_commands.target_paths（保留 target_paths 作为派生缓存）

migrations/seed/000NNN_mml_standard_import.sql
    - 一次性把 standard-model.xml 转成 INSERT (mml_params)
    - source='standard', catalog_protected=true
    - 由 cmd/omcctl mml export-standard-xml 工具生成；不再启动期重读
```

> **版本号规则**：当前最大 `000094`，新 DDL `000095`；seed 文件版本号紧接 seed/ 目录最大值递增。

---

## 5. XML → DB 一次性迁移

### 5.1 工具：`cmd/omcctl mml import-standard-xml`

```bash
omcctl mml import-standard-xml \
    --xml omcgo/data/param-mappings/standard-model.xml \
    --out migrations/seed/000NNN_mml_standard_import.sql
```

行为：

1. 解析 XML（复用 `mmlstandardloader/parser.go`）
2. 对每个 `<param>` / `<object>` 节点：
   - 计算 `value_type` / `access_type` / `is_object` / `change_applies`
   - 推断 `supports_add` / `supports_delete`（is_object && access_type=READ_WRITE）
   - 由 `min`/`max`/`type` 综合渲染 `constraint_text_i18n`
   - param_code 取 path 末段 + path hash 后缀（避免冲突）
3. 输出 SQL 文件，2001 行 INSERT，幂等：

```sql
INSERT INTO mml_params (
    id, param_code, tr069_path, value_type, access_type,
    is_object, supports_add, supports_delete, change_applies,
    name_i18n, explanation_i18n, constraint_text_i18n,
    catalog_protected, source, param_version
) VALUES
(gen_random_uuid(), 'SOFTWAREVERSION_a3f9', 'Device.DeviceInfo.SoftwareVersion', 'STRING', 'READ_ONLY',
 false, false, false, 'Immediate',
 '{"en-US":"Software Version","zh-CN":"软件版本"}',
 '{}',
 '{"en-US":"Read-only string","zh-CN":"只读字符串"}',
 true, 'standard', 'STANDARD'),
...
ON CONFLICT (tr069_path, param_version) DO UPDATE SET
    access_type = EXCLUDED.access_type,
    is_object = EXCLUDED.is_object,
    ...;
```

### 5.2 禁用启动期 `mmlstandardloader`

```diff
- // cmd/app/router/deps.go
- mmlLoader := mmlstandardloader.NewLoader(pool, ...)
- moduleGraph.Add("mml-standard", mmlLoader, ...)
+ // mmlstandardloader 已下线；catalog 通过 migration seed + admin UI 管理
+ // 历史代码保留在 internal/config/parammodel/mmlstandardloader/ 仅作工具用途
```

但**保留** `mmlstandardloader/parser.go`、`grouper.go`、`command_gen.go` 作为：

- `omcctl import-standard-xml` 命令的解析引擎
- 后续 admin UI "一键重新生成基础 catalog" 功能的底层

### 5.3 多语言注入策略

XML 自身只有英文 standardPath，没有中文标签。两种来源融合：

1. **机器翻译 + 人工校对**：从 path 末段提取候选 zh-CN 标签（如 `SoftwareVersion` → "软件版本"），写入 seed
2. **逐 group 人工补齐**：BSC Configuration 9 子组的命令名称由调研期 playwright 截图提供（已收集），手工填 zh-CN
3. **运行时 fallback**：如果 `*_i18n["zh-CN"]` 为空，UI 显 `["en-US"]` 兜底，并在 admin catalog 管理页面用 "🟡 需翻译" badge 标出来

---

## 6. 后端 API（共 11 端点）

### 6.1 Console 运行期（前端 Console 用）

| Method | Path | 用途 |
|--------|------|------|
| GET | `/api/v1/mml/groups/tree?root={code}&lang={zh-CN}` | 命令树（嵌套），lang 决定字典层渲染语言 |
| GET | `/api/v1/mml/commands/:id` | 命令详情（含 logical/operation/i18n） |
| GET | `/api/v1/mml/commands/:id/sub-fields?lang={...}` | sub-field 列表（join `mml_params` 取 access/object/change_applies/constraint） |
| POST | `/api/v1/mml/render` | 由 `{command_id, op, selected_sub_field_ids, values}` 渲染 MML 字符串 |
| POST | `/api/v1/mml/parse` | MML 字符串 → 拆为 statements 数组 |
| POST | `/api/v1/mml/execute` | 批量执行：`{device_sns:[], statements:[{command_id, op, selected_sub_field_ids?, values?}]}` |
| GET | `/api/v1/mml/custom-commands` | 现有；列 PrivateTemplate + PublicTemplate（按 user/scope 过滤） |

### 6.2 Catalog 管理（admin UI 用，权限：`mml.catalog.manage`）

| Method | Path | 用途 |
|--------|------|------|
| POST | `/api/v1/mml/admin/groups` | 创建一级 / 二级 group |
| PATCH | `/api/v1/mml/admin/groups/:id` | 改名 / 改排序 / 改 i18n |
| DELETE | `/api/v1/mml/admin/groups/:id` | 删 group（catalog_protected=true 时拒绝） |
| POST | `/api/v1/mml/admin/commands` | 创建命令（指定 logical_code / group_id / op_type / target_paths） |
| PATCH | `/api/v1/mml/admin/commands/:id` | 改 logical_name_i18n / require_confirm / sub-field 绑定 |
| DELETE | `/api/v1/mml/admin/commands/:id` | 删命令（catalog_protected=true 时拒绝） |
| POST | `/api/v1/mml/admin/commands/:id/sub-fields` | 给命令加 sub-field（指定 param_id + mml_code + label） |
| PATCH | `/api/v1/mml/admin/commands/:cid/sub-fields/:sid` | 改 label / sort / default_selected / is_required |
| DELETE | `/api/v1/mml/admin/commands/:cid/sub-fields/:sid` | 移除 sub-field 绑定 |
| GET | `/api/v1/mml/admin/params` | 浏览 mml_params 字典（搜索 / 分页 / 按 access_type / is_object 过滤） |
| POST | `/api/v1/mml/admin/params` | 新增非标 param（source='admin'） |
| PATCH | `/api/v1/mml/admin/params/:id` | 改 i18n / access_type / 元数据（catalog_protected=true 时部分字段锁定） |
| DELETE | `/api/v1/mml/admin/params/:id` | 删（catalog_protected=true 拒绝；有 sub-field 引用时拒绝） |

### 6.3 关键端点 payload 详细

**POST `/api/v1/mml/execute`**（核心，用户输入入口）：

```json
{
  "device_sns": ["F4F1F7...","0D59FB..."],
  "statements": [
    {
      "command_id": "uuid-1",
      "operation_type": "LST",
      "selected_sub_field_ids": ["sf-1","sf-2","sf-5"]
    },
    {
      "command_id": "uuid-2",
      "operation_type": "MOD",
      "values": { "Mcc": "460", "Encryption": "1" }
    }
  ],
  "execute_type": "immediate",
  "task_name": "BSC LST + MOD Quick"
}
```

行为：

- 多设备 fanout：`fanout.go` 生成 N×M 个 device_task
- 一条 LST statement → 后端拼 `GetParameterValues` RPC 一次发 K 个 path（K = selected sub-fields）
- 一条 MOD statement → 拼 `SetParameterValues` RPC 一次发 V 个 path-value（V = values keys）
- ADD → `AddObject` + 紧接 `SetParameterValues` 设置 instance 字段
- RMV → `DeleteObject(target_object + index)`
- statement 数组顺序敏感 → 默认串行（依赖现有 Sprint B `sequencer.go`），可选并发

---

## 7. 前端 UI

### 7.1 Console 三栏（恢复老布局）

`omcmb/webcode/src/pages/mml/Console/index.tsx`：

```
+---- StepBar 1-2-3-4 -------------------------------------+
+--------+-----------------+------------------------------+
| Step1  | Step2           | Step3                        |
| Device | CommandTree     | RightPanel (Tab)             |
| Tree   |  ├ BSC Config   |  ┌─ Control Panel | ParamPath│
| (多选) |  │  ├ Basic Info |  ├─ MmlEditor (textbox + DO) │
|        |  │  ├ BTS        |  ├─ SubFieldPanel (按 op 切) │
|        |  │  ├ ...        |  │   • LST → Checklist       │
|        |  ├ Customized    |  │   • MOD → InputList       │
|        |  │  ├ Private    |  │   • ADD → InputList +     │
|        |  │  │  ├ admin   |  │           target_object   │
|        |  │  │  └ ...     |  │   • RMV → InstancePicker  │
|        |  │  └ Public     |  └─ Result/Help radio + log  │
+--------+-----------------+------------------------------+
```

### 7.2 组件总览

| 组件 | 路径 | 职责 |
|------|------|------|
| `DeviceTree` | `Console/components/` 现有 | 多选设备（不动） |
| `CommandTree` | 重写 | 拉 `/groups/tree`，渲染嵌套；点击叶子→ `appendStatement(commandId)` |
| `MmlEditor` | 新增 | 顶部 textbox + DO 按钮；防抖 300ms 触发 parse；语法错误高亮 |
| `SubFieldChecklist` | 新增 | LST 用：勾选列表，每行 (label, mml_code, tr069_path, access 图标) |
| `SubFieldInputList` | 新增 | MOD/ADD 用：输入框 + constraintText + required 红 `*` + change_applies=OnReboot 黄 badge |
| `InstancePicker` | 新增 | RMV 用：先列出当前实例（GPV 探测），让用户选要删的 index |
| `TerminalPanel` | 现有 | 保留作为 Step4 Result 显示 |
| `ParamPathExpert` | 重命名 | 原 `ParamPathPanel`，重命名 + 改造为"专家视图 tab"（保留双 tab 切换） |
| `AddTemplateModal` | 现有 | 保存当前 statements 为 PrivateTemplate / PublicTemplate（写 `mml_custom_commands`） |
| `BatchSnModal` | 现有 | 批量输入设备 SN（不动） |
| 删除 | `ParamFormRenderer` | 替换为 SubFieldInputList |

### 7.3 元数据驱动的 UI 差异化（用户澄清 #2 重点）

| param 元数据 | LST 列表呈现 | MOD 列表呈现 |
|------|------|------|
| `access_type=READ_ONLY` | ☑ 勾选行（默认勾选） | **不出现**（MOD 列表过滤掉） |
| `access_type=READ_WRITE` | ☑ 勾选行 | 行内显输入框 |
| `is_object=true && supports_add=true` | LST 显 "查看实例" 图标 | (MOD 不涉及对象) |
| `is_object=true && supports_delete=true` | LST 显示该 object 的实例列表，每行带 🗑 | (RMV 视图用) |
| `change_applies=OnReboot` | 列表行右侧黄色 ⚠ "需重启生效" badge | 输入框旁带同 badge，提交时弹二次确认 |
| `catalog_protected=true` | 无视觉差异 | (admin 管理界面"删除"按钮禁用) |

### 7.4 状态管理

新建 `omcmb/frontend-core/src/store/mmlConsoleStore.ts`：

```ts
type Statement = {
  uid: string;                           // 客户端 nanoid
  commandId: string;
  operationType: 'LST'|'MOD'|'ADD'|'RMV';
  commandCode: string;                   // 例 DEVICE_INFO
  logicalNameI18n: I18n;                 // {"en":"Device info","zh":"设备信息"}
  subFields: SubFieldDef[];              // 命令完整子字段（来自 GET /sub-fields）
  selectedSubFieldIds: string[];         // LST
  values: Record<string, string>;        // MOD/ADD：key = mml_code
  rmvInstanceIndex?: number;             // RMV
};

interface MmlConsoleState {
  selectedDeviceSns: string[];
  statements: Statement[];               // 顺序敏感
  mmlText: string;                       // 与 statements 双向同步
  activeStatementUid: string|null;
  lang: 'zh-CN'|'en-US';
  // ...
  appendStatement(commandId): Promise<void>;
  removeStatement(uid): void;
  setStatementsFromText(text): Promise<void>;   // 失败降级走 POST /parse
  setStatementsFromUI(s: Statement[]): void;    // 反算 mmlText
  execute(): Promise<TaskID>;
}
```

**双向同步流**：

```
[勾选/输入] → setStatementsFromUI → 本地 render → 写 mmlText
[textbox 改] → debounce 300ms → 本地 parse → 成功写 statements
                                            → 失败 → 降级 POST /parse → 写 statements 或保留错误
```

### 7.5 多皮肤影响

业务层 (API / Hook / Store / Types / i18n 语料) **全部进 `omcmb/frontend-core/`**。
UI 壳本期只动 `omcmb/webcode/`（主皮肤）。
变更后须运行：

```bash
cd omcmb/webcode-v2 && npm run typecheck   # 不破坏候选皮肤编译
cd omcmb/webcode-v3 && npm run typecheck
```

### 7.6 admin Catalog 管理 UI（独立路由）

`/mml/admin/catalog`（权限 `mml.catalog.manage`）：

- Tab 1：**Groups** — 树形增删改，拖拽排序，i18n 行内编辑
- Tab 2：**Commands** — 列表 + 详情：基本信息、绑定 group、sub-field 绑定面板
- Tab 3：**Params 字典** — 表格 + 搜索 + 按 access_type / is_object 过滤；选中行可看 "被哪些命令引用"
- Tab 4：**XML 导入** — 上传/选择 XML 文件，预览 diff，确认后生成迁移 seed（不立即写表）

> **关键点**：admin UI 改的内容立即写表（走 admin API），不依赖重启；通过 `parammodel:cache_version` 让其他实例失效缓存。

---

## 8. 兼容性 & 迁移

### 8.1 历史数据处理

| 历史资产 | v2 处置 |
|---------|---------|
| Sprint A Loader 已写入的 mml_commands / mml_param_groups / mml_params | 保留，作为 v2 初始数据 |
| `mml_commands.target_paths` JSONB | 保留，语义降级为派生缓存（由 `mml_command_sub_fields` 触发器维护） |
| `mml_custom_commands.parameters / param_paths` | 保留；Console 加载时通过 adapter 翻为 statements |
| `mml_param_versions` STANDARD 行 | 保留作 import_standard 的 version_code 锚点 |
| Sprint B `sequencer.go` / `fanout.go` / `scheduler.go` / `result_aggregator.go` | 不动；只调整 Execute 入参格式 |
| `mmlstandardloader/` 包 | 启动期注册下线；保留代码作 omcctl 工具引擎 |

### 8.2 回滚策略

迁移 `000095` 的 Down 段：

- DROP `mml_command_sub_fields` 表
- ALTER ... DROP COLUMN（按 Up 顺序逆序）
- seed `mml_standard_import` 数据保留（不删，避免数据丢失）

注：Down 之后 catalog UI 不可用，但 Sprint A/B 老路径功能不受影响。

### 8.3 多语言种子文件结构

`omcmb/frontend-core/src/i18n/zh-CN/mml.ts`：

```ts
export default {
  // Step bar / 按钮 / Tab 等静态文案
  console: { step1: '选择设备', step2: '选择命令', step3: '配置参数', step4: '查看结果', ... },
  // 命令树 / 子字段标签**不在此处**：它们来自 DB 的 *_i18n JSONB，运行时按 lang 取
};
```

DB 字段优先于前端 i18n 语料；前端语料仅承担 UI 框架文案（按钮、标题、错误码）。

---

## 9. 实施路线（5 阶段）

| 阶段 | 范围 | 关键交付 | 工作日 |
|------|------|--------|------|
| **P0 数据层** | 迁移 000095 + 一次性 XML 导入 seed + admin API 骨架 | DB 字段齐全；`omcctl mml import-standard-xml` 可用；admin catalog CRUD 后端通过 unit test | 3 |
| **P1 Console 后端** | tree / sub-fields / render / parse / execute 5 端点 | E2E：BSC > Basic Info > Device info(LST/MOD) 全链路通 | 3 |
| **P2 Console 前端** | 三栏 + MmlEditor + SubFieldChecklist + SubFieldInputList + 双向绑定 | playwright e2e：勾选 / textbox / DO 三方向都同步 | 4 |
| **P3 catalog 管理 UI** | admin 四 Tab + 元数据可视化 + 多语言行内编辑 | admin 能新增 group / 移动命令 / 改 sub-field 顺序 | 4 |
| **P4 收尾** | ADD/RMV 命令体（InstancePicker）+ Customized 集成 + 元数据驱动差异化全量 + DoD | e2e 226→240 断言，覆盖率达 80% | 4 |

**总计 18 工作日**（PoC 节奏，可视并行度压缩到 14）。

---

## 10. 风险与对策

| 风险 | 等级 | 对策 |
|------|------|------|
| 现有 `mml_commands` 由 Loader 机器生成，命名 / 分组与老系统不完全一致 | 高 | P0 阶段 admin UI 上线后由维护者人工调整：改 group 名 / 重排 / 改 i18n；标准 import 后的"原料"自动跑过 P1 测试即可，"老系统视感"靠 P3 admin UI 收敛 |
| `mml_code` 的命名规律不在 XML 中（如 `LTE_GSM_MODEL_NAME` 是老系统约定） | 高 | 默认 mml_code = uppercased path 末段；admin UI 允许覆盖；不阻塞执行（执行只看 selected_sub_field_ids → tr069_path） |
| 多语言 zh-CN 标签如何补齐 2001 条 path | 中 | 机器翻译先行（path 末段→ 中文词典），admin UI "🟡 需翻译" badge 引导维护者增量校对；不阻塞上线 |
| sub-field 触发器维护 `target_paths` 性能 | 低 | 用 statement-level trigger 而非 row-level；批量 admin 操作时单事务一次重算 |
| Customized 自定义命令的 statements 适配 | 中 | 写 adapter：旧 `parameters` JSONB → statements 形式；T-0 期保留旧调用兼容 |
| 启动期 mmlstandardloader 下线导致 dev 环境数据丢失 | 中 | seed/000NNN_mml_standard_import.sql 等价提供同样数据；同 commit 提交 |
| admin UI 改坏 catalog 影响所有用户 | 高 | admin 操作走 audit log；catalog_protected=true 行锁定关键字段；提供 "导入 / 导出 catalog JSON" 用于备份/恢复 |
| Sprint B group execute / sequencer 与新 execute payload 兼容 | 中 | 在 service 层加 statement → 老 Internal Command 适配器；既有调用方零改动 |

---

## 11. 验收（DoD）

### 11.1 数据层

- [ ] `migrations/000095_mml_command_catalog_v2.sql` 通过 goose up/down
- [ ] `mml_params` 含 access_type / is_object / supports_add / supports_delete / change_applies / constraint_text_i18n / catalog_protected
- [ ] `mml_command_sub_fields` 表存在；包含 mml_code / label_i18n / default_selected / is_required / sort_order
- [ ] `mml_commands` 含 logical_code / logical_name_i18n / source / catalog_protected
- [ ] `seed/000NNN_mml_standard_import.sql` 2001 条 INSERT 通过；启动期 loader 已禁用
- [ ] `mml_params.constraint_text_i18n` zh-CN 覆盖率 ≥ 60%

### 11.2 后端 API

- [ ] 11 个端点通过 unit test
- [ ] `/groups/tree?lang=zh-CN` 返回 BSC Configuration → Basic Info → Device info(LST/MOD) 完整树
- [ ] `/commands/:id/sub-fields` 含 14 项 sub-field（与 §1.3 表一致）
- [ ] `/execute` 入参 N 设备 × M statements 创建 N×M device_task
- [ ] catalog 管理 API 通过权限校验（无权返回 403）

### 11.3 前端

- [ ] Console 三栏 + Step Bar 重现
- [ ] 点击 "Device info(LST DEVICE_INFO)" → 右栏勾选 14 行全勾，textbox 实时同步
- [ ] 勾掉 Model Name → textbox `lstId={...}` 实时少 LTE_GSM_MODEL_NAME
- [ ] textbox 输入 `LST DEVICE_INFO:lstId={LTE_GSM_IP};` → 勾选 UI 只剩 IP
- [ ] MOD DEVICE_INFO 输入 Mcc=460 → textbox 自动 `MOD DEVICE_INFO:Mcc=460;`
- [ ] textbox `LST DEVICE_INFO;LST BTS_INFO;` 二条 statement → 2 设备多选 → 4 device_task
- [ ] Control Panel ⇄ ParameterPath Command tab 切换不丢状态
- [ ] change_applies=OnReboot 字段 UI 显黄 badge，MOD 提交弹二次确认
- [ ] READ_ONLY param 在 MOD 列表不出现
- [ ] is_object && supports_delete=true 对象在 LST 显示实例列表（每行带 🗑）
- [ ] Customized > PrivateTemplate / PublicTemplate 子树渲染、双击加载到右栏可用
- [ ] admin `/mml/admin/catalog` 四 Tab 可用，groups 拖拽排序生效

### 11.4 E2E + 兼容

- [ ] `bash omcgo/scripts/e2e_verify.sh` Sprint 0-9 既有 226 断言全通过
- [ ] 新增 14 条 MML 专项断言（catalog/tree/sub-fields/render/parse/execute/admin CRUD 各 2 条）
- [ ] `cd omcmb/webcode-v2 && npm run typecheck` 通过
- [ ] `cd omcmb/webcode-v3 && npm run typecheck` 通过

---

## 12. v2 已审决（全部锁定）

| # | 决策点 | 锁定方案 | 落实位置 |
|---|--------|---------|---------|
| 1 | `mml_code` 命名策略 | **默认 = path 末段大写化**（`Device.DeviceInfo.SoftwareVersion` → `SOFTWAREVERSION`）；admin UI 允许行内覆盖；老系统 `LTE_GSM_*` 前缀由 admin UI 手工补，不程序化推断 | §5.1 import 工具 + §6.2 admin API + §7.6 admin UI |
| 2 | 多语言种子来源 | **机器翻译先行 + admin UI 增量校对**；不在 P0 阻塞上线；前端用 "🟡 需翻译" badge 引导校对 | §5.3 多语言注入策略 |
| 3 | admin RBAC 粒度 | **单一权限点 `mml.catalog.manage`**（默认仅 admin role）；不按 groups/commands/params 三层细分；未来如需收紧再扩展 | §6.2 catalog 管理 API + §7.6 admin UI |

---

## 13. 下一步（dev-pipeline 入口）

按项目流程（参见 `docs/project/dev-pipeline-design-20260420.md`）：

1. **S0 任务登记**：在 `docs/project/backlog.md` 新建 `T-NNNN` 条目，PRD 链接指向本文档
2. **S1-S6 由 `/dev-pipeline pick T-NNNN` 拉起**，分 5 个 sprint（P0-P4）
3. 每个 P 阶段开 PR；P0 是阻塞下游 P1-P4 的"地基"，需先合入

**第一刀建议**：从 P0 数据层切入（迁移 000095 + `omcctl mml import-standard-xml` 工具 + admin API 骨架），不引入 UI 改动；3 工作日内可见首个绿色 PR，再决定 P1-P4 并发还是串行。

---

附：v1 设计中"引入 `mml-command-catalog.xml`"思路已废弃；现统一通过 DB + admin UI + 一次性 XML import seed 实现 catalog 管理。playwright 实测数据见调研期 yml 快照（已清理）。

---

# 设计备忘（T-0123-P0 S2 设计定稿，2026-05-14）

> **生效阶段**：S2 design → S3 implement 入参
> **范围**：仅 T-0123-P0（数据层 + 工具 + admin API 骨架）；P1-P4 设计另起备忘
> **审签**：架构 / 数据 / 安全 / 电信 / Go 工程 五专家并行检视
> **待定点**：3 个（见 §M.9）

## M.1 文件清单（S3 实施输出）

| 路径 | 类型 | 行数估算 |
|------|------|---------|
| `omcgo/migrations/000095_mml_command_catalog_v2.sql` | DDL + trigger | ~200 |
| `omcgo/migrations/seed/000096_mml_standard_params_import.sql` | DML（由工具生成） | ~6000（2001 INSERT × 3 行） |
| `omcgo/cmd/omcctl/cmd/mml_import.go` | cobra 子命令 | ~250 |
| `omcgo/cmd/omcctl/cmd/mml.go` | mml 父命令注册 | ~30 |
| `omcgo/internal/mml/model.go` | 加字段 | +60 |
| `omcgo/internal/mml/sub_field_model.go` | 新 sub-field 类型 | ~80 |
| `omcgo/internal/mml/admin_repository.go` | admin CRUD 接口 + Pg 实现 | ~600 |
| `omcgo/internal/mml/admin_service.go` | admin 服务层（校验 / catalog_protected 守护 / audit log） | ~400 |
| `omcgo/internal/mml/admin_handler.go` | 13 个 admin HTTP handler | ~700 |
| `omcgo/cmd/app/router/router.go` | 注册 13 端点 + RBAC 中间件 | +30 |
| `omcgo/cmd/app/router/deps.go` | 注入 admin service + 注释下线 mmlstandardloader | +20 / -5 |
| `omcgo/internal/mml/*_test.go` | table-driven 单测（admin repo + service + import_test） | ~1500 |

总变更：~10000 行（含 seed 自动生成的 6000 行）；人工撰写约 4000 行。

## M.2 Migration 000095 schema 定稿

### M.2.1 mml_params 扩展（8 列 + `is_writable` 转为 GENERATED STORED 派生列）

> **Q1=B 决议（2026-05-14）**：放弃触发器同步方案，改用 PG 12+ 原生 GENERATED ALWAYS AS STORED — 强保证 `is_writable` 与 `access_type` 永不漂移；老 SQL `WHERE is_writable=true` 透明继续工作；admin/loader 直接写 `is_writable` 会被 PG 自动拒绝（语义即"派生字段"）。

**步骤拆解**（Up 段顺序敏感）：

```sql
-- Step 1: 新增 8 列
ALTER TABLE mml_params
    ADD COLUMN IF NOT EXISTS access_type          VARCHAR(20) NOT NULL DEFAULT 'READ_ONLY'
        CHECK (access_type IN ('READ_ONLY','READ_WRITE','WRITE_ONLY')),
    ADD COLUMN IF NOT EXISTS is_object            BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_add         BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_delete      BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS change_applies       VARCHAR(20) NOT NULL DEFAULT 'Immediate',
    ADD COLUMN IF NOT EXISTS constraint_text_i18n JSONB       NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS catalog_protected    BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS source               VARCHAR(20) NOT NULL DEFAULT 'admin';

-- Step 2: backfill access_type 从既有 is_writable（保护现有数据）
UPDATE mml_params SET access_type = 'READ_WRITE' WHERE is_writable = true;

-- Step 3: DROP 旧 is_writable 普通列（连同任何依赖索引；既有 schema 中无单独 is_writable 索引）
ALTER TABLE mml_params DROP COLUMN is_writable;

-- Step 4: 重建 is_writable 为 GENERATED STORED 派生列
ALTER TABLE mml_params
    ADD COLUMN is_writable BOOLEAN
        GENERATED ALWAYS AS (access_type IN ('READ_WRITE','WRITE_ONLY')) STORED;

-- Step 5: 索引（access_type / is_object / source）
CREATE INDEX IF NOT EXISTS idx_mml_params_access_type ON mml_params(access_type);
CREATE INDEX IF NOT EXISTS idx_mml_params_is_object   ON mml_params(is_object);
CREATE INDEX IF NOT EXISTS idx_mml_params_source      ON mml_params(source);
```

**Down 段（反向，保留 is_writable 数据完整性）**：

```sql
-- Step 1: 临时列捕获 is_writable 当前值
ALTER TABLE mml_params ADD COLUMN _is_writable_temp BOOLEAN NOT NULL DEFAULT false;
UPDATE mml_params SET _is_writable_temp = (access_type IN ('READ_WRITE','WRITE_ONLY'));

-- Step 2: DROP generated is_writable
ALTER TABLE mml_params DROP COLUMN is_writable;

-- Step 3: 重命名 temp 列为 is_writable
ALTER TABLE mml_params RENAME COLUMN _is_writable_temp TO is_writable;

-- Step 4: DROP 其他 7 新列 + 索引
ALTER TABLE mml_params
    DROP COLUMN IF EXISTS access_type,
    DROP COLUMN IF EXISTS is_object,
    DROP COLUMN IF EXISTS supports_add,
    DROP COLUMN IF EXISTS supports_delete,
    DROP COLUMN IF EXISTS change_applies,
    DROP COLUMN IF EXISTS constraint_text_i18n,
    DROP COLUMN IF EXISTS catalog_protected,
    DROP COLUMN IF EXISTS source;
DROP INDEX IF EXISTS idx_mml_params_access_type;
DROP INDEX IF EXISTS idx_mml_params_is_object;
DROP INDEX IF EXISTS idx_mml_params_source;
```

**消费者侧影响**：
- Sprint A `mmlstandardloader/loader.go` 已下线（不再启动期注册）→ 即使它仍写 `is_writable` 也不会被触发
- `omcctl mml import-standard-xml` 工具 INSERT 时**不再包含** `is_writable` 列（auto-derive）
- `internal/mml/pg_repository.go` 的 UPSERT 语句需移除 `is_writable` — 在 S3 同步修改
- admin Service Update 路径不允许写 `is_writable`（不在 PATCH 字段集，本身就该这样）

### M.2.2 mml_commands 扩展（4 列）

```sql
ALTER TABLE mml_commands
    ADD COLUMN IF NOT EXISTS logical_code      VARCHAR(100),
    ADD COLUMN IF NOT EXISTS logical_name_i18n JSONB       NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS source            VARCHAR(20) NOT NULL DEFAULT 'admin',
    ADD COLUMN IF NOT EXISTS catalog_protected BOOLEAN     NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_mml_commands_logical_code ON mml_commands(logical_code);
```

> **`logical_code` 派生规则**：admin 创建或 import 时由代码计算 — `command_code` 若以 `<OP>_` 开头则去前缀（LST_DEVICE_INFO → DEVICE_INFO）；admin UI 允许手工覆盖。

### M.2.3 mml_param_groups 扩展（2 列）

```sql
ALTER TABLE mml_param_groups
    ADD COLUMN IF NOT EXISTS source            VARCHAR(20) NOT NULL DEFAULT 'admin',
    ADD COLUMN IF NOT EXISTS catalog_protected BOOLEAN     NOT NULL DEFAULT false;
```

### M.2.4 新表 mml_command_sub_fields

```sql
CREATE TABLE IF NOT EXISTS mml_command_sub_fields (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id       UUID NOT NULL REFERENCES mml_commands(id) ON DELETE CASCADE,
    param_id         UUID NOT NULL REFERENCES mml_params(id)   ON DELETE RESTRICT,

    mml_code         VARCHAR(100) NOT NULL,
    label_i18n       JSONB NOT NULL DEFAULT '{}',
    default_selected BOOLEAN NOT NULL DEFAULT true,
    is_required      BOOLEAN NOT NULL DEFAULT false,
    sort_order       INT NOT NULL DEFAULT 0,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_command_mml_code UNIQUE (command_id, mml_code),
    CONSTRAINT uq_command_param    UNIQUE (command_id, param_id)
);

CREATE INDEX IF NOT EXISTS idx_mml_command_sub_fields_command ON mml_command_sub_fields(command_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_mml_command_sub_fields_param   ON mml_command_sub_fields(param_id);

-- updated_at 自动维护（复用既有共享函数 update_updated_at_column）
CREATE TRIGGER trg_mml_command_sub_fields_updated_at
    BEFORE UPDATE ON mml_command_sub_fields
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### M.2.5 target_paths 派生触发器

每当 sub_fields 变化（INSERT / UPDATE / DELETE），重算所属 command 的 `target_paths` JSONB 数组：

```sql
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_mml_command_target_paths(p_command_id UUID) RETURNS VOID AS $$
BEGIN
    UPDATE mml_commands c
    SET target_paths = COALESCE((
        SELECT jsonb_agg(p.tr069_path ORDER BY csf.sort_order)
        FROM mml_command_sub_fields csf
        JOIN mml_params p ON p.id = csf.param_id
        WHERE csf.command_id = p_command_id
    ), '[]'::jsonb),
        updated_at = NOW()
    WHERE c.id = p_command_id;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trg_mml_sub_fields_refresh_paths() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_mml_command_target_paths(OLD.command_id);
        RETURN OLD;
    ELSE
        PERFORM refresh_mml_command_target_paths(NEW.command_id);
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_mml_sub_fields_target_paths
    AFTER INSERT OR UPDATE OR DELETE ON mml_command_sub_fields
    FOR EACH ROW EXECUTE FUNCTION trg_mml_sub_fields_refresh_paths();
```

> **设计依据**：sub_fields 是写少读多场景（admin 操作典型 N<100/次），ROW-level trigger 性能可接受。批量 INSERT 时同 command_id 多次 refresh 是幂等的，最终结果一致。

### M.2.6 Down 段（完整反向）

> mml_params 部分见 §M.2.1（is_writable 临时列保护流程）；其余按 Up 顺序逆序执行。

```sql
-- +goose Down

-- 1. 触发器 + 函数
DROP TRIGGER IF EXISTS trg_mml_sub_fields_target_paths ON mml_command_sub_fields;
DROP TRIGGER IF EXISTS trg_mml_command_sub_fields_updated_at ON mml_command_sub_fields;
DROP FUNCTION IF EXISTS trg_mml_sub_fields_refresh_paths();
DROP FUNCTION IF EXISTS refresh_mml_command_target_paths(UUID);

-- 2. mml_command_sub_fields 表
DROP TABLE IF EXISTS mml_command_sub_fields;

-- 3. mml_param_groups 反向
ALTER TABLE mml_param_groups
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS catalog_protected;

-- 4. mml_commands 反向
DROP INDEX IF EXISTS idx_mml_commands_logical_code;
ALTER TABLE mml_commands
    DROP COLUMN IF EXISTS logical_code,
    DROP COLUMN IF EXISTS logical_name_i18n,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS catalog_protected;

-- 5. mml_params 反向（生成列 → 普通列 → DROP 其他 7 列），详 §M.2.1 Down
```

## M.3 omcctl mml import-standard-xml 工具

### M.3.1 命令签名

```bash
omcctl mml import-standard-xml \
    --xml omcgo/data/param-mappings/standard-model.xml \
    --out omcgo/migrations/seed/000096_mml_standard_params_import.sql \
    --version-code STANDARD \
    [--dry-run]
```

### M.3.2 处理流程

1. **解析**：复用 `internal/config/parammodel/mmlstandardloader/parser.go::ParseStandardModelXML`
2. **逐节点映射**：
   - `param_code` = `<UPPER_PATH_SEGMENT_LAST>_<hash8(fullPath)>` （示例：`SOFTWAREVERSION_a3f912cd`）— 保证全局唯一
   - `tr069_path` = standardPath（原样）
   - `value_type` = type（string/INT/U_INT/BOOLEAN/ENUM）映射小写枚举
   - `access_type` = access (READ_ONLY/READ_WRITE) — XML 缺省 READ_ONLY
   - `is_object` = path 以 `.` 结尾 → true
   - `supports_add` = `is_object && access_type='READ_WRITE'`（admin 可单独 PATCH 覆盖）
   - `supports_delete` = 同 `supports_add`
   - `change_applies` = changeApplies (Immediate / OnReboot)
   - `name_i18n["en-US"]` = path 末段 PascalCase（如 SoftwareVersion → "Software Version"）；`name_i18n["zh-CN"]` = 翻译模板渲染（基于内置词典 + 路径上下文，无法翻则留空）
   - `constraint_text_i18n` = `renderConstraintText(type, min, max)` 生成双语提示
   - `catalog_protected` = true
   - `source` = "standard"
   - `param_version` = "STANDARD"
3. **输出 SQL**：每行 INSERT 用既有 unique constraint `uq_param_version_path (param_version, tr069_path)` 做 ON CONFLICT UPSERT；**不包含 is_writable 列**（Q1=B 决议后由 GENERATED 自动派生）：

```sql
INSERT INTO mml_params (
    id, param_version, param_code, tr069_path, value_type,
    access_type, is_object, supports_add, supports_delete, change_applies,
    name_i18n, explanation_i18n, constraint_text_i18n,
    catalog_protected, source, group_id, display_order, created_at, updated_at
) VALUES
  (gen_random_uuid(), 'STANDARD', 'SOFTWAREVERSION_a3f912cd',
   'Device.DeviceInfo.SoftwareVersion', 'STRING',
   'READ_ONLY', false, false, false, 'Immediate',
   '{"en-US":"Software Version","zh-CN":"软件版本"}'::jsonb,
   '{}'::jsonb,
   '{"en-US":"Read-only string","zh-CN":"只读字符串"}'::jsonb,
   true, 'standard', NULL, 0, NOW(), NOW()),
  ...
ON CONFLICT (param_version, tr069_path) DO UPDATE SET
    access_type = EXCLUDED.access_type,
    is_object = EXCLUDED.is_object,
    supports_add = EXCLUDED.supports_add,
    supports_delete = EXCLUDED.supports_delete,
    change_applies = EXCLUDED.change_applies,
    constraint_text_i18n = EXCLUDED.constraint_text_i18n,
    updated_at = NOW()
WHERE mml_params.catalog_protected = true;  -- 仅刷新 protected 行，保护 admin 编辑（Q2=C 决议）
```

**碰撞兜底**（F2 自审）：`hash8(fullPath)` 全空间 4.3B，2001 path 碰撞概率 4.6e-7；工具内部预 check unique，若发现碰撞自动升 12 字符 hash 兜底。

**Down 段**：

```sql
DELETE FROM mml_params WHERE source = 'standard' AND param_version = 'STANDARD';
```

### M.3.3 翻译模板（中文标签兜底）

> **Q4=A 决议（2026-05-14）**：P0 阶段词典 **硬编码在 Go map 中** （位置 `cmd/omcctl/cmd/mml_zh_dictionary.go` 文件级 `var zhDictionary = map[string]string{...}`）；P3 admin UI 上线后再迁出到 `omcgo/data/mml-zh-dictionary.json` 走文件加载，让维护者通过 UI 编辑。

**机器翻译路径末段 → zh-CN 规则**：

| 模式 | 示例 path 末段 | en-US 默认 | zh-CN 输出 |
|------|---------------|-----------|-----------|
| PascalCase 拆词后命中词典 | SoftwareVersion | Software Version | 软件版本（"Software"+" 版本"逐词查询）|
| 已知缩写 | OUI / MAC / IP | OUI / MAC / IP | OUI / MAC / IP（缩写不译） |
| 未命中词 | XCellDimm | X Cell Dimm | （留空 `""` → admin UI "🟡 需翻译" badge 提示） |

**词典覆盖范围**（~200 词）：TR-069 高频词 — Device / Software / Hardware / Service / Cell / Carrier / Frequency / Power / Tx / Rx / Rate / Status / Mode / Enable / Disable / Version / Address / Port / Channel / Bandwidth / Threshold / Timeout / Count / Index / Type / Name / ID / Reset / Reboot / Config / Param / Info / List / Add / Delete / Update 等。

**实现位置**：`omcgo/cmd/omcctl/cmd/mml_zh_dictionary.go` 单文件，仅 omcctl 工具引用，main app 完全不感知。

## M.4 admin CRUD API（13 端点）骨架

### M.4.1 路由表

| 方法 | 路径 | Handler 函数 |
|------|------|-------------|
| POST   | `/api/v1/mml/admin/groups` | `AdminCreateGroup` |
| PATCH  | `/api/v1/mml/admin/groups/:id` | `AdminUpdateGroup` |
| DELETE | `/api/v1/mml/admin/groups/:id` | `AdminDeleteGroup` |
| POST   | `/api/v1/mml/admin/commands` | `AdminCreateCommand` |
| PATCH  | `/api/v1/mml/admin/commands/:id` | `AdminUpdateCommand` |
| DELETE | `/api/v1/mml/admin/commands/:id` | `AdminDeleteCommand` |
| POST   | `/api/v1/mml/admin/commands/:cid/sub-fields` | `AdminCreateSubField` |
| PATCH  | `/api/v1/mml/admin/commands/:cid/sub-fields/:sid` | `AdminUpdateSubField` |
| DELETE | `/api/v1/mml/admin/commands/:cid/sub-fields/:sid` | `AdminDeleteSubField` |
| GET    | `/api/v1/mml/admin/params` | `AdminListParams` |
| POST   | `/api/v1/mml/admin/params` | `AdminCreateParam` |
| PATCH  | `/api/v1/mml/admin/params/:id` | `AdminUpdateParam` |
| DELETE | `/api/v1/mml/admin/params/:id` | `AdminDeleteParam` |

GET (list/tree) 端点复用 Console 既有路径（`/mml/groups/tree` 在 P1 实施）。

### M.4.2 RBAC 端点级集成（W2 audit 后修订）

> **W2 audit 结论（2026-05-14）**：原 `permissions` 表已在 migration 000064 DROP，项目 RBAC 切换到**端点级 Casbin 策略**：`role_api_permissions(role_id, endpoint_id) JOIN api_endpoints(path, method)`。原 §M.4.2 假设的 `permissions.code='mml.catalog.manage'` 模型**不适用**。

**新方案**（与既有 RBAC 模型对齐）：

1. **路由自注册**：13 个 admin 端点在 Gin 路由表注册后，由 `internal/admin/api_endpoint_registry` 启动期自动扫描并 UPSERT 到 `api_endpoints` 表（is_auto=TRUE）。命名约定：
   - `api_group` = `mml_admin`
   - `name` = 中文动作（如 "创建 MML 命令分组"）
   - `description` = 中文长说明
2. **admin 角色批量授权**：seed 文件 `migrations/seed/000097_mml_admin_role_permissions.sql` 在 admin role × `mml_admin` group 端点上批量 INSERT `role_api_permissions`：

```sql
-- +goose Up
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT r.id, e.id
FROM roles r
JOIN api_endpoints e ON e.api_group = 'mml_admin'
WHERE r.code = 'admin'
  AND e.path LIKE '/api/v1/mml/admin/%'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints WHERE api_group = 'mml_admin'
)
AND role_id IN (SELECT id FROM roles WHERE code = 'admin');
```

3. **运行期保障**：Casbin 中间件（`internal/admin/casbin.go`）已基于 `role_api_permissions` LoadPolicy；管理员调用 `/api/v1/mml/admin/...` 时自动 403 if no permission。**no app-level RequirePermission middleware needed**（既有 Casbin 已覆盖）。

> 13 个 admin 端点列表见 §M.4.1；启动期 api_endpoints 自注册逻辑在 `internal/admin/api_endpoint_scanner.go`（既有）会自动捕获新路由，**S3 实施时只需在路由注册时给 group `mml_admin` 命名 + 在 Gin route handler 函数上加注释 doc**。
>
> **dev 环境快速验证**：`SELECT * FROM api_endpoints WHERE api_group='mml_admin';` 应返 13 行；`SELECT COUNT(*) FROM role_api_permissions JOIN api_endpoints USING(id) WHERE api_group='mml_admin'` 应返 13（admin role 全覆盖）。

### M.4.3 catalog_protected 守护逻辑

> **Q2=C 决议（2026-05-14）**：`catalog_protected` 字段**不在任何 PATCH API 的允许字段集**。语义即 `catalog_protected ≡ (source = 'standard')`，仅 INSERT 时由 source 决定，运行期不可切换。admin 想"解锁" standard 行只能改 `source='admin'`（但这同样不在 PATCH 字段集，需通过 super_admin 专门工具 / 数据库直改 — 后续 P3 admin UI 可考虑给 super_admin 暴露但本期不做）。

`admin_service.go` 中每个 Update/Delete 方法首查 `catalog_protected`：

```go
func (s *adminService) UpdateParam(ctx context.Context, id uuid.UUID, req UpdateParamReq) error {
    existing, err := s.repo.GetByID(ctx, id)
    if err != nil { return fmt.Errorf("get param: %w", err) }
    if existing.CatalogProtected {
        // 仅允许更新非锁定字段：name_i18n / explanation_i18n / constraint_text_i18n
        // 锁定字段：tr069_path / param_code / access_type / is_object / source / catalog_protected
        if req.Tr069Path != "" || req.ParamCode != "" || req.AccessType != "" {
            return fmt.Errorf("param %s is catalog_protected, cannot modify locked fields: %w",
                id, ErrCatalogProtected)
        }
    }
    return s.repo.Update(ctx, existing.ApplyUpdate(req))
}

func (s *adminService) DeleteParam(ctx context.Context, id uuid.UUID) error {
    existing, err := s.repo.GetByID(ctx, id)
    if err != nil { return fmt.Errorf("get param: %w", err) }
    if existing.CatalogProtected {
        return ErrCatalogProtected
    }
    // 还要检查 sub_fields 引用
    refs, err := s.subFieldRepo.CountByParam(ctx, id)
    if err != nil { return err }
    if refs > 0 {
        return fmt.Errorf("param referenced by %d sub_fields: %w", refs, ErrParamInUse)
    }
    return s.repo.Delete(ctx, id)
}
```

**handler 层 payload 守护**（PATCH 端点拒收 `catalog_protected` / `source` / `is_writable` 字段）：

```go
type UpdateParamReq struct {
    NameI18n            *map[string]string `json:"name_i18n,omitempty"`
    ExplanationI18n     *map[string]string `json:"explanation_i18n,omitempty"`
    ConstraintTextI18n  *map[string]string `json:"constraint_text_i18n,omitempty"`
    DefaultValue        *string            `json:"default_value,omitempty"`
    JsRegex             *string            `json:"js_regex,omitempty"`
    // catalog_protected / source / is_writable / tr069_path / access_type / is_object — 不暴露
}
```

Sentinel errors：

```go
var (
    ErrCatalogProtected = errors.New("entry is catalog_protected; modification denied")
    ErrParamInUse       = errors.New("param is referenced by sub_fields")
    ErrGroupNotEmpty    = errors.New("group has commands; cannot delete")
)
```

Handler 把 sentinel 翻成 HTTP 403 / 409。

### M.4.4 审计日志

每个写操作（POST/PATCH/DELETE）调 `internal/admin/auditlog.Write(ctx, audit.Entry{...})`：

```go
op := "mml.catalog.command.created"
entry := audit.Entry{
    Actor:    middleware.UsernameFromCtx(ctx),
    Op:       op,
    Resource: fmt.Sprintf("command:%s", cmd.ID),
    Detail:   map[string]any{
        "command_code": cmd.CommandCode,
        "logical_code": cmd.LogicalCode,
        "group_id":     cmd.GroupID,
        "source":       cmd.Source,
    },
}
auditlog.Write(ctx, entry)
```

12 个 audit op：
- `mml.catalog.group.{created,updated,deleted}`
- `mml.catalog.command.{created,updated,deleted}`
- `mml.catalog.sub_field.{created,updated,deleted}`
- `mml.catalog.param.{created,updated,deleted}`

## M.5 Go 接口契约

### M.5.1 model.go 加字段

```go
type MMLCommand struct {
    // ... 现有字段（id / command_name / command_code / category / operation_type / target_paths / ...）
    LogicalCode      string             `json:"logical_code" db:"logical_code"`
    LogicalNameI18n  map[string]string  `json:"logical_name_i18n" db:"logical_name_i18n"`
    Source           string             `json:"source" db:"source"`
    CatalogProtected bool               `json:"catalog_protected" db:"catalog_protected"`
    SubFields        []MMLCommandSubField `json:"sub_fields,omitempty"`
}

// 新类型
type MMLCommandSubField struct {
    ID              uuid.UUID         `json:"id" db:"id"`
    CommandID       uuid.UUID         `json:"command_id" db:"command_id"`
    ParamID         uuid.UUID         `json:"param_id" db:"param_id"`
    MMLCode         string            `json:"mml_code" db:"mml_code"`
    LabelI18n       map[string]string `json:"label_i18n" db:"label_i18n"`
    DefaultSelected bool              `json:"default_selected" db:"default_selected"`
    IsRequired      bool              `json:"is_required" db:"is_required"`
    SortOrder       int               `json:"sort_order" db:"sort_order"`
    CreatedAt       time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}
```

`Param` 加字段（在 `param_model.go`）：

```go
type Param struct {
    // ... 现有字段
    AccessType         string            `json:"access_type" db:"access_type"`
    IsObject           bool              `json:"is_object" db:"is_object"`
    SupportsAdd        bool              `json:"supports_add" db:"supports_add"`
    SupportsDelete     bool              `json:"supports_delete" db:"supports_delete"`
    ChangeApplies      string            `json:"change_applies" db:"change_applies"`
    ConstraintTextI18n map[string]string `json:"constraint_text_i18n" db:"constraint_text_i18n"`
    CatalogProtected   bool              `json:"catalog_protected" db:"catalog_protected"`
    Source             string            `json:"source" db:"source"`
}
```

`ParamGroup` 加字段（同上）。

### M.5.2 Repository 接口（消费者驱动，小接口原则）

`internal/mml/admin_repository.go`：

```go
// SubFieldRepository — 4 method
type SubFieldRepository interface {
    Create(ctx context.Context, sf *MMLCommandSubField) error
    Update(ctx context.Context, sf *MMLCommandSubField) error
    Delete(ctx context.Context, id uuid.UUID) error
    ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubField, error)
    CountByParam(ctx context.Context, paramID uuid.UUID) (int64, error)
}

// AdminGroupRepository — 4 method
type AdminGroupRepository interface {
    Create(ctx context.Context, g *ParamGroup) error
    Update(ctx context.Context, g *ParamGroup) error
    Delete(ctx context.Context, id uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*ParamGroup, error)
}

// AdminCommandRepository — 3 method （Read 复用既有 CommandRepository.GetByID）
type AdminCommandRepository interface {
    Create(ctx context.Context, c *MMLCommand) error
    Update(ctx context.Context, c *MMLCommand) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// AdminParamRepository — 5 method
type AdminParamRepository interface {
    Create(ctx context.Context, p *Param) error
    Update(ctx context.Context, p *Param) error
    Delete(ctx context.Context, id uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*Param, error)
    List(ctx context.Context, f AdminParamFilter) ([]Param, int64, error)
}

type AdminParamFilter struct {
    Search       *string  // tr069_path / name_i18n 全文模糊
    AccessType   *string
    IsObject     *bool
    Source       *string
    PageNum      int
    PageSize     int
}
```

### M.5.3 Service 层

`internal/mml/admin_service.go`（"接受接口、返回结构体"）：

```go
type AdminService struct {
    groupRepo    AdminGroupRepository
    commandRepo  AdminCommandRepository
    subFieldRepo SubFieldRepository
    paramRepo    AdminParamRepository
    auditWriter  audit.Writer    // 来自 internal/admin/auditlog
    logger       *zap.Logger
    metrics      *AdminMetrics
}

func NewAdminService(
    groupRepo AdminGroupRepository,
    commandRepo AdminCommandRepository,
    subFieldRepo SubFieldRepository,
    paramRepo AdminParamRepository,
    auditWriter audit.Writer,
    logger *zap.Logger,
    metrics *AdminMetrics,
) *AdminService {
    return &AdminService{...}
}
```

Service 方法签名（13 个）一律带 `ctx context.Context`，错误用 `fmt.Errorf("op: %w", err)` 包装。

### M.5.4 Handler 层

`internal/mml/admin_handler.go` 13 handler，遵循既有项目模式（Gin + ShouldBindJSON 校验 + service 调用 + errors.Render）。

## M.6 启动期 mmlstandardloader 下线

`cmd/app/router/deps.go` 改动：

```diff
- // 注册 MML standard model loader（每次启动重读 XML，~0.5s）
- mmlLoader := mmlstandardloader.NewLoader(c.PG.Pool, c.DictLoaderCfg.XMLBaseDir, "", "", c.Logger, c.Registerer)
- if err := c.ModuleGraph.Add("mml-standard", mmlLoader, []string{"dictload"}); err != nil {
-     return fmt.Errorf("register mml-standard loader: %w", err)
- }
+ // mmlstandardloader 启动期注册下线（T-0123-P0）：
+ // 改为 omcctl mml import-standard-xml 一次性导出 seed → DB 一次性导入。
+ // parser.go / grouper.go / command_gen.go 整包保留作为 omcctl 工具引擎。
+ // 历史方案见 docs/design/mml-rebuild-plan-20260513.md；下线方案见
+ // docs/design/mml-restore-old-interaction-plan-20260514.md §5
```

acs / worker 两个 cmd 同步处理（若有引用）。

**保留**整个 `internal/config/parammodel/mmlstandardloader/` 包，作为：
1. `omcctl mml import-standard-xml` 的解析引擎
2. 未来 admin UI "一键重生 catalog 基础数据" 功能的底层

## M.7 观测埋点清单

### M.7.1 Prometheus metrics（新增 3 个）

| Metric Name | Type | Labels | 用途 |
|-------------|------|--------|------|
| `omc_mml_admin_request_total` | Counter | `op, resource, result` | 请求计数 op=create/update/delete/list, resource=group/command/sub_field/param, result=success/forbidden/notfound/conflict/error |
| `omc_mml_admin_request_duration_seconds` | Histogram | `op, resource` | 请求时延 P50/P95/P99 |
| `omc_mml_admin_protected_denied_total` | Counter | `resource` | catalog_protected 拒绝次数（安全监控） |

### M.7.2 Zap log 关键字段

每个 admin handler 入口与出口 log，必含字段：

- `op` = "mml_admin_<verb>_<resource>"（如 `mml_admin_create_group`）
- `actor` = username from RBAC middleware
- `request_id` from gin context
- `resource_id` = uuid（按资源类型 group_id / command_id / sub_field_id / param_id）
- `catalog_protected` = true/false（当涉及 protected 资源时）
- `result` = success / failure (with error)

### M.7.3 Audit log subject

12 个 audit subject（§M.4.4 列出）。

## M.8 测试矩阵

| 测试类型 | 文件 | 覆盖 |
|---------|------|------|
| Unit — repo | `admin_repository_test.go` | Pg CRUD 表驱动（success / not-found / unique-violation / fk-violation） |
| Unit — service | `admin_service_test.go` | catalog_protected 守护 / sentinel error 翻译 / audit 调用 |
| Unit — handler | `admin_handler_test.go` | 200/400/403/404/409 全状态码 / payload 校验 |
| Unit — import | `cmd/omcctl/cmd/mml_import_test.go` | 解析样本 XML → 验证 SQL 输出格式 + ON CONFLICT 行为（in-memory pg or testcontainer） |
| Integration — migration | `test/integration/migration_000095_test.go` | up + down 双向演练；触发器行为 — sub_field INSERT/DELETE 后 target_paths 重算 |
| E2E — admin API | `scripts/e2e_verify.sh` 新加 `mml_admin_*` claim | 4 资源 × 3 verb 共 12 claim + 1 forbidden + 1 protected = 14 claim |

覆盖率门槛：
- `internal/mml/admin_*.go` 覆盖率 ≥ 75%
- `cmd/omcctl/cmd/mml_*.go` 覆盖率 ≥ 60%（CLI 工具下限放宽）

## M.9 待定点（Q5 决议：均同意，处置如下）

| # | 待定点 | 处置方案（已 APPROVED） |
|---|--------|---------|
| W1 | 已有 Sprint A 写入的 mml_params 行如何与 import seed 共存 | **同意**：默认 import 仅 UPSERT `catalog_protected=true` 行；admin 改过的 catalog_protected=false 行**永不被 standard re-import 覆盖**（Q2=C 决议同步守护）。**S3 实施时 audit Sprint A loader 写入行的当前 catalog_protected 值**（应当为 false 默认值，因为 Sprint A 早于本任务，未写入新字段）→ standard import 第一次跑会全量 UPSERT 这些 false 行（因 catalog_protected=false 跳过），结果是 Sprint A 行被冷藏 / standard import 仅在它们不存在时 INSERT 新行。**预期最终态**：Sprint A 残余行 (catalog_protected=false) + standard import 新 INSERT 行 (catalog_protected=true) 共存；admin UI 可在 P3 上线后清理 Sprint A 残余 |
| W2 | `permissions` / `role_permissions` 实际 schema | **同意**：S3 起手第一动作 `grep -nE "CREATE TABLE permissions\|role_permissions" omcgo/migrations/`；与本备忘 §M.4.2 假设 schema 对齐。若不一致，调整 seed INSERT 语法。本任务 migration 000095 中**不含权限 seed**，权限 seed 拆到独立文件 `seed/000097_mml_catalog_manage_permission.sql`，方便单独迁移 / 单独回滚 |
| W3 | constraint_text 多语言模板覆盖率（zh-CN 缺失行） | **同意**：P0 阶段先英文全覆盖 + zh-CN ~200 词典（Q4=A：硬编码 Go map）；剩余 admin UI（P3）用 "🟡 需翻译" badge 引导维护者增量补；不阻塞 P0 |

## M.10 S3 实施顺序（推荐）

```
D1 (3-4h):  Migration 000095 + 触发器 + 本地 up/down 演练 + uuid/format/lint 全过
D1 (3-4h):  Go model.go 加字段 + admin_repository 接口 + Pg 实现（含表驱动单测）
D2 (4h):    omcctl mml import-standard-xml 工具 + 解析逻辑 + i18n 模板词典 + 单测
D2 (4h):    生成 seed/000096_*.sql + 本地 migrate-up 验证 2001 行 + ON CONFLICT 路径
D3 (4h):    admin_service.go + admin_handler.go 13 端点 + RBAC seed
D3 (4h):    单测全过 + golangci-lint + go test -race + verify report + S5 准备
```

## M.11 S2 出口门核查

- [✓] 接口契约明确（§M.5 列出 5 个 repo 接口 + 4 个 service / handler 类）
- [✓] 迁移草案（§M.2 全 DDL + 触发器；Down 段完整）
- [✓] Carrier 差异点（§0.5 已锁定"无差异，本任务三家一致"，无新增点）
- [✓] 观测埋点（§M.7 列 3 metric + 6 关键 log 字段 + 12 audit subject）
- [✓] 待定点 < 3（§M.9 列 3 个）

S2 PASS。

---
# 设计备忘（T-0123-P1 S2 设计定稿，2026-05-14）

> **生效阶段**：S2 design → S3 implement 入参
> **范围**：T-0123-P1（Console 后端 5 端点 + Go MML renderer/parser + Execute statements fanout）
> **依赖**：T-0123-P0 ✅（数据层 + admin 13 端点 + 4 admin repo + seed 000096 已落库）
> **关联**：PRD §6.1 / §6.3 / §M.5 / §M.6 Console runtime
> **审签**：架构 / 电信 / Go / 数据 四专家并行检视
> **待定点**：3 个（见 §N.10）

## N.1 文件清单（S3 实施输出）

| 路径 | 类型 | 行数估算 |
|------|------|---------|
| `omcgo/internal/mml/mml_renderer.go` | 新 — render statement → MML string | ~180 |
| `omcgo/internal/mml/mml_parser.go` | 新 — parse MML string → statements | ~250 |
| `omcgo/internal/mml/console_handler.go` | 新 — 5 Console endpoints (tree/sub-fields/render/parse/execute) | ~400 |
| `omcgo/internal/mml/console_service.go` | 新 — service 层：buildGroupTree / executeStatements 编排 | ~350 |
| `omcgo/internal/mml/group_tree_repository.go` | 新 — 树形 SQL + LTREE path 查询 | ~150 |
| `omcgo/internal/mml/service.go` | mod — 加 ExecuteStatements 方法（兼容老 ExecuteCommand） | +120 |
| `omcgo/internal/mml/handler.go` | mod — 删除 `/mml/execute` 旧路由让位新 console_handler；保留兼容 fallback | +30/-15 |
| `omcgo/internal/mml/tr069_payload.go` | mod — 支持 statement→TR-069 payload（LST GPV/MOD SPV/ADD AddObject+SPV/RMV DeleteObject） | +200 |
| `omcgo/internal/mml/repository.go` | mod — CommandRepository.GetByID 增 SubFields 字段加载 | +30 |
| `cmd/app/provider/modules.go` | mod — DI 注入 console_handler + group_tree_repo | +10 |
| `cmd/app/provider/router.go` | mod — 注册 5 Console endpoints（permGroup("devices")） | +6 |
| `omcgo/internal/mml/mml_renderer_test.go` | 新 — 表驱动 render 单测 | ~250 |
| `omcgo/internal/mml/mml_parser_test.go` | 新 — 表驱动 parse 单测（含语法错） | ~300 |
| `omcgo/internal/mml/console_service_test.go` | 新 — buildGroupTree + executeStatements 单测 | ~400 |
| `omcgo/internal/mml/console_handler_test.go` | 新 — 5 endpoint payload + 错误路径 | ~300 |
| `omcgo/scripts/e2e_verify.sh` | mod — +5 console claim (tree/sub-fields/render/parse/execute) | +50 |

总计：~3030 行（人工 ~2400 + 测试 ~1250）；3 工作日预算。

## N.2 5 Console API 端点契约

### N.2.1 `GET /api/v1/mml/groups/tree?root={code}&lang={zh-CN}`

返回嵌套树（一级 group → 二级 group → 命令叶子），前端 `CommandTree` 直接渲染。

**Request**：
- `root` (optional)：根 group_code，缺省返所有顶级（BSC_CONFIGURATION / ENB_CONFIG / 等）
- `lang` (optional, 默认 zh-CN)：i18n 选语种

**Response**（信封 `response.OK(c, data)`）：

```json
{
  "code": "BSC_CONFIGURATION",
  "name": "BSC 配置",
  "name_i18n": {"zh-CN":"BSC 配置","en-US":"BSC Configuration"},
  "children": [
    {
      "code": "BASIC_INFO",
      "name": "基本信息",
      "name_i18n": {...},
      "commands": [
        {
          "id": "uuid-1",
          "command_code": "LST_DEVICE_INFO",
          "logical_code": "DEVICE_INFO",
          "logical_name": "设备信息",
          "logical_name_i18n": {"zh-CN":"设备信息","en-US":"Device info"},
          "operation_type": "LST",
          "display_name": "设备信息(LST DEVICE_INFO)",
          "require_confirm": false,
          "rpc_method": "GetParameterValues"
        }
      ],
      "children": []
    }
  ]
}
```

**实现**：`group_tree_repository.go::BuildTree(rootCode, lang)` 单 SQL JOIN：
```sql
SELECT g.id, g.group_code, g.name_i18n, g.path::text, g.display_order,
       c.id, c.command_code, c.logical_code, c.logical_name_i18n,
       c.operation_type, c.command_name_i18n, c.require_confirm, c.rpc_method
FROM mml_param_groups g
LEFT JOIN mml_commands c ON c.group_id = g.id
WHERE g.path <@ $1::ltree  -- 按 LTREE 子树查
   OR g.path = $2::ltree   -- 根 group 本身
ORDER BY g.path, g.display_order, c.operation_type;
```
然后 Go 侧按 ltree path 长度分层组装 children。

**display_name 派生**：`logical_name + "(" + operation_type + " " + logical_code + ")"`，与老 OMC 实测格式一致。

### N.2.2 `GET /api/v1/mml/commands/:id/sub-fields?lang={...}`

**P0 已 prep**：`admin_repository.go::PgSubFieldRepository.ListEnrichedByCommand` 返 `[]MMLCommandSubFieldEnriched`（含 join `mml_params` 的 tr069_path / access_type / is_object / supports_add / supports_delete / change_applies / constraint_text_i18n / default_value / js_regex / name_i18n）。

**Response**：

```json
{
  "command_id": "uuid-1",
  "operation_type": "LST",
  "sub_fields": [
    {
      "id": "sf-uuid-1",
      "mml_code": "LTE_GSM_MODEL_NAME",
      "label": "Model Name",
      "label_i18n": {...},
      "tr069_path": "Device.DeviceInfo.X_COM_MODULE_TYPE",
      "value_type": "string",
      "access_type": "READ_ONLY",
      "is_object": false,
      "supports_add": false,
      "supports_delete": false,
      "change_applies": "Immediate",
      "constraint_text": "Read-only string",
      "constraint_text_i18n": {...},
      "default_value": null,
      "js_regex": null,
      "default_selected": true,
      "is_required": false,
      "sort_order": 1
    }
  ]
}
```

**实现**：`console_service.go::GetCommandSubFields(commandID, lang) ([]SubFieldDTO, error)` 调 SubFieldRepository.ListEnrichedByCommand → 选 lang 派生 label / constraint_text 顶级字段。

### N.2.3 `POST /api/v1/mml/render`

**Request**：

```json
{
  "command_id": "uuid-1",
  "operation_type": "LST",
  "selected_sub_field_ids": ["sf-uuid-1","sf-uuid-2"],
  "values": {}
}
```

**Response**：

```json
{ "mml_string": "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_SYS_TIME};" }
```

**实现**：`mml_renderer.go::RenderStatement(stmt Statement, subFields []SubField) (string, error)`，纯函数无 DB；用 sort_order 决定字段顺序。

### N.2.4 `POST /api/v1/mml/parse`

**Request**：

```json
{ "mml_string": "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_IP};MOD DEVICE_INFO:Mcc=460;" }
```

**Response**：

```json
{
  "statements": [
    {
      "operation_type": "LST",
      "command_code": "DEVICE_INFO",
      "command_id": "uuid-1",
      "selected_sub_field_ids": ["sf-uuid-1","sf-uuid-3"],
      "values": {}
    },
    {
      "operation_type": "MOD",
      "command_code": "DEVICE_INFO",
      "command_id": "uuid-1",
      "values": { "Mcc": "460" },
      "unknown_codes": []
    }
  ],
  "parse_errors": []
}
```

**实现**：`mml_parser.go::ParseMMLString(s, lookup CommandLookup) ([]Statement, []ParseError)`；lookup 由 service 提供（按 (operation_type, command_code) 查 mml_commands）。未命中 command_code → ParseError 累加但不中断；mml_code 未命中 → `unknown_codes` 累加（前端 toast 提示）。

### N.2.5 `POST /api/v1/mml/execute`

**Request**：

```json
{
  "device_sns": ["F4F1F7...","0D59FB..."],
  "statements": [
    {
      "command_id": "uuid-1",
      "operation_type": "LST",
      "selected_sub_field_ids": ["sf-1","sf-2"]
    },
    {
      "command_id": "uuid-2",
      "operation_type": "MOD",
      "values": {"Mcc": "460"}
    }
  ],
  "execute_type": "immediate",
  "task_name": "BSC LST+MOD"
}
```

**Response**：

```json
{
  "task_id": "task-uuid",
  "device_task_count": 4,
  "statement_count": 2,
  "sse_subscription_url": "/api/v1/mml/tasks/task-uuid/stream"
}
```

**实现**：复用既有 `Service.Execute` 升级为 `ExecuteStatements`（兼容 fallback：单 statement + 老 ExecuteHTTPRequest payload）。Statements 进 fanout → N×M device_task。

## N.3 Go MML Renderer

**核心函数**：

```go
// RenderStatement 把单条 statement 渲染为 MML 字符串片段（不含末尾 ;）。
// subFields 按 sort_order 已排序；selectedIDs 用 set 做 O(1) 查找。
func RenderStatement(stmt Statement, subFields []MMLCommandSubField, commandCode string) (string, error) {
    switch stmt.OperationType {
    case "LST":
        codes := collectSelectedMMLCodes(stmt.SelectedSubFieldIDs, subFields)
        return fmt.Sprintf("LST %s:lstId={%s}", commandCode, strings.Join(codes, ",")), nil
    case "MOD":
        kvs := collectModValues(stmt.Values, subFields)
        if len(kvs) == 0 { return "MOD " + commandCode, nil }  // 无值时仍返裸 op
        return fmt.Sprintf("MOD %s:%s", commandCode, strings.Join(kvs, ",")), nil
    case "ADD":
        kvs := collectModValues(stmt.Values, subFields)
        return fmt.Sprintf("ADD %s:%s", commandCode, strings.Join(kvs, ",")), nil
    case "RMV":
        if stmt.RmvInstanceIndex != nil {
            return fmt.Sprintf("RMV %s:Index=%d", commandCode, *stmt.RmvInstanceIndex), nil
        }
        return "RMV " + commandCode, nil
    }
    return "", fmt.Errorf("unsupported operation: %s", stmt.OperationType)
}

// RenderStatements 多条 statement → MML 字符串（用 ; 分隔，末尾 ;）。
func RenderStatements(stmts []StatementWithMeta) (string, error) {
    parts := make([]string, 0, len(stmts))
    for _, sm := range stmts {
        part, err := RenderStatement(sm.Stmt, sm.SubFields, sm.CommandCode)
        if err != nil { return "", fmt.Errorf("render statement %d: %w", sm.Index, err) }
        parts = append(parts, part)
    }
    return strings.Join(parts, ";") + ";", nil
}
```

老 OMC 实测 MML 语法：
- `LST <CODE>:lstId={CODE1,CODE2,...};`
- `MOD <CODE>:Field1=Value1,Field2=Value2;`（无值时裸 `MOD <CODE>`）
- 多条 `;` 连接，末尾 `;`

## N.4 Go MML Parser

**核心函数**：

```go
// ParseMMLString 解析 MML 字符串为 statements；
// commandLookup 由 service 注入（按 (op, code) 返 *MMLCommand + []MMLCommandSubField）。
// 解析失败累加 ParseError，不中断整个 string；未知 mml_code 累加 stmt.UnknownCodes。
func ParseMMLString(s string, lookup CommandLookup) ([]Statement, []ParseError) {
    // 1. 按 ; 分割（容忍末尾 ; 和连续 ;;）
    // 2. 每段 trim space；空段 skip
    // 3. 按 ' ' 分前缀（op + code）
    // 4. 按 ':' 分 op_code 与 params
    // 5. op==LST：parse "lstId={K1,K2,...}" → SelectedMMLCodes → lookup 转 sub_field_ids
    //    op==MOD/ADD：parse "K1=V1,K2=V2" → Values map
    //    op==RMV：parse "Index=N" → RmvInstanceIndex
    // 6. lookup(op, code) → command_id；未命中累 ParseError
    // 7. mml_code 在 lookup 的 sub_fields 找 → sub_field_id；未命中累 unknown_codes
}

type ParseError struct {
    StatementIndex int      // 第几条 statement (0-based)
    Raw            string   // 原始字符串
    Reason         string   // 中文错误描述
}
```

**语法容错**：
- 大小写：op 忽略大小写（lst / LST 都识别）；code 严格保持
- 空格：`LST  DEVICE_INFO  :  lstId={...}` 容忍多空格
- 顺序：MOD `Mcc=460,Encryption=1` 与 `Encryption=1,Mcc=460` 等效
- 连续 ;;：跳过
- 缺末尾 ;：补一个解析（前端友好）
- 引号：值带空格用 `"v with space"` 双引号包裹（罕见但需支持）

## N.5 ExecuteStatements 服务层 fanout

**核心**：

```go
// ExecuteStatements 是 ExecuteCommand 的多语句版本。
// 行为：
//   1. 校验每条 statement 的 command_id 存在 + operation_type 合法
//   2. 对 LST：从 sub_field_ids 查 tr069_paths（join mml_command_sub_fields + mml_params）
//      → BuildTR069Params(GPV) → device_task.params
//   3. 对 MOD：values map keys 查 sub_field（通过 mml_code）→ tr069_path
//      → BuildTR069Params(SPV) → device_task.params
//   4. 对 ADD：target_object 查 mml_commands.target_object → AddObject RPC + 紧跟 SPV
//   5. 对 RMV：target_object + index → DeleteObject(target_object + "{i}") 通过 path 实例化
//   6. Fanout: N device_sns × M statements → N×M device_tasks
//      - 顺序敏感 → 用既有 sequencer (sequentialMode=true 多 statement 强制串行)
//      - 单 statement + N device → fanouter 默认并发
//
// 返回 mml_task_id + statement_count + device_task_count
func (s *Service) ExecuteStatements(ctx context.Context, req ExecuteStatementsRequest) (*MMLTask, error) {
    // 校验
    statementsMeta, err := s.loadStatementsWithMeta(ctx, req.Statements)
    if err != nil { return nil, fmt.Errorf("load statements meta: %w", err) }
    // 创建 mml_task 记录所有 statements
    task, err := s.taskRepo.Create(ctx, &MMLTask{
        DeviceSNs:    req.DeviceSNs,
        Commands:     statementsToCommandsJSON(statementsMeta),  // 兼容老 commands 字段
        TotalDevices: len(req.DeviceSNs) * len(req.Statements),
        // ...
    })
    if err != nil { return nil, fmt.Errorf("create task: %w", err) }
    // Fanout 走既有 fanouter — 仅 cmd_idx=0 入队，sequencer 链式入队后续
    if err := s.fanouter.Fanout(ctx, task.ID, statementsMeta, sequentialMode); err != nil { ... }
    return task, nil
}
```

**单 statement 兼容**：handler 收到老 `ExecuteHTTPRequest` 形态（含 command_code + parameters）时，转 1 元素 statements 数组走同一路径。

## N.6 TR-069 payload 构造

`tr069_payload.go` 现有 `BuildTR069Params(method, paths)` 已支持 partial path 透明展开（T-0119 Sprint B-3）。P1 新加：

```go
// BuildStatementPayload 把单条 statement 翻为单条 device_task.params (JSONB)。
func BuildStatementPayload(stmt StatementMeta) (json.RawMessage, string, error) {
    switch stmt.OperationType {
    case "LST":
        // GPV partial path 展开复用 expandInstancePaths
        paths := collectTr069Paths(stmt.SelectedSubFieldIDs, stmt.SubFields)
        return BuildTR069Params(RPCGetParameterValues, paths)
    case "MOD":
        // SPV: { "ParameterList": [{"Name": tr069_path, "Value": v, "Type": value_type}, ...] }
        pvs := buildParameterValueList(stmt.Values, stmt.SubFields)
        return BuildTR069SetParams(pvs)
    case "ADD":
        // AddObject(target_object) + 后续 SPV 设字段值；分两个 device_task or 一个聚合 task
        // 选择：合并为 1 device_task，params={"ObjectName": target_object, "SetParams": [...]}，
        //       ACS handler 内部链式发 AddObject → SPV（既有 dispatcher 支持）
        return BuildTR069AddObjectPayload(stmt.TargetObject, stmt.Values, stmt.SubFields)
    case "RMV":
        // DeleteObject(target_object 替换 {i} 为 RmvInstanceIndex)
        path := injectInstanceIndex(stmt.TargetObject, *stmt.RmvInstanceIndex)
        return BuildTR069DeleteObjectPayload(path)
    }
    return nil, "", fmt.Errorf("unsupported op: %s", stmt.OperationType)
}
```

**RPC method 派生**：每 statement → RPC method（LST→GetParameterValues / MOD→SetParameterValues / ADD→AddObject / RMV→DeleteObject）；mml_commands.rpc_method 字段已有值（mmlstandardloader 写入），verify 一致性。

## N.7 数据流图

```
HTTP POST /mml/execute
  ↓
console_handler.Execute(c *gin.Context)
  - Bind ExecuteStatementsRequest
  - validate device_sns / statements 非空
  ↓
console_service.ExecuteStatements(ctx, req)
  - loadStatementsWithMeta: 按 command_id list 查 mml_commands + sub_fields (1 SQL JOIN)
  - 校验每 statement 的 sub_field_ids/values 引用合法
  - taskRepo.Create(mml_task)
  ↓
fanouter.Fanout(ctx, taskID, statementsMeta, sequentialMode)
  - 多 statement → sequentialMode=true → 只 enqueue cmd_idx=0
  - sequencer 链式：device_task 0 完成 → enqueue cmd_idx=1 → ...
  ↓
tr069_payload.BuildStatementPayload(stmt)  ← 每 enqueue 调用
  - LST: GPV path 展开
  - MOD: SPV param-value list
  - ADD: AddObject + 紧跟 SPV
  - RMV: DeleteObject path
  ↓
internal/task.CreateTask(...)
  - device_tasks 表 + Redis Sorted Set
  - source=mml, source_id=mml_task_id, cmd_idx=N, device_idx=M
  ↓
ACS PopTask → CWMP SOAP → 设备执行
  ↓
ACS MarkTaskCompleted → completion event
  ↓
sequencer.OnTaskCompleted → 链式入队下一 cmd
  ↓
result_aggregator.OnTaskCompleted → MML SSE frame 推 frontend
```

## N.8 观测埋点清单

### N.8.1 Prometheus metrics（新增 4 个）

| Metric | Type | Labels | 用途 |
|--------|------|--------|------|
| `omc_mml_console_request_total` | Counter | `endpoint, status` | 5 endpoint 请求计数 |
| `omc_mml_render_duration_seconds` | Histogram | — | render 性能 |
| `omc_mml_parse_duration_seconds` | Histogram | — | parse 性能（user textbox 防抖 300ms） |
| `omc_mml_execute_statements_total` | Counter | `op_type, result` | statement 数 × op_type 分布 |

### N.8.2 Zap log 关键字段

- `op` = "mml_console_<verb>"（tree / sub_fields / render / parse / execute）
- `actor` = username from JWT middleware
- `request_id` from gin context
- `task_id` (execute path)
- `statement_count` / `device_count`（fanout 后）
- `result` = success / failure (with error)

### N.8.3 Event subjects（无新增）

ExecuteStatements 复用既有 fanout / sequencer / result_aggregator 链路，事件主题不变。

## N.9 测试矩阵

| 测试类型 | 文件 | 覆盖 |
|---------|------|------|
| Unit — renderer | `mml_renderer_test.go` | 4 op × (空字段/单字段/多字段) + 12 testcase 含老 OMC 实测样例 `LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_SYS_TIME};` |
| Unit — parser | `mml_parser_test.go` | 大小写容忍 / 空格容忍 / 缺末尾 ; / 未知 code 累加 unknown_codes / ParseError 不中断 / 双引号值 / Round-trip 自洽（parse(render(x)) == x） |
| Unit — service | `console_service_test.go` | buildGroupTree LTREE 层级 / ExecuteStatements 单/多 statement / 校验 sub_field_id 非法 → 400 |
| Unit — handler | `console_handler_test.go` | 5 endpoints × 4 状态码（200/400/404/500） |
| Integration | `test/integration/mml_console_test.go` | 真 PG + tx rollback，validate sub_fields JOIN 性能 |
| E2E | `scripts/e2e_verify.sh` console-1..5 | 5 endpoint curl 全覆盖 + 1 forbidden test = +6 claim |

覆盖率门槛：
- `mml_renderer.go` / `mml_parser.go` ≥ 85%（纯函数易测）
- `console_handler.go` / `console_service.go` ≥ 75%

## N.10 待定点

| # | 待定点 | 处置方案 |
|---|--------|---------|
| W1 | ADD 操作的 AddObject + SPV 是分两个 device_task 还是合并为 1 个？ | **合并为 1 个**：ACS handler 内部链式发，避免 fanout 中间状态错误处理复杂。但 ACS handler 是否支持复合 RPC payload 需 audit `internal/acs/rpc/`。**S3 起手第一动作 audit**；如不支持则拆 2 个 device_task。 |
| W2 | mml_string parse 时多 statement 间相同 command_code 但不同 op（如 `LST DEVICE_INFO; MOD DEVICE_INFO`）如何 lookup command_id？ | (op, command_code) 联合唯一 — `mml_commands` 表既有 UNIQUE(command_code) 约束但 P0 schema 后可能允许同 code 多 op（实际数据：LST_DEVICE_INFO / MOD_DEVICE_INFO 是不同 command_code）。**lookup 用 (op, logical_code) 联合查**：`SELECT * FROM mml_commands WHERE logical_code=$1 AND operation_type=$2`；命中 ≥ 2 行报歧义 ParseError。 |
| W3 | parse 是否要支持子字符串约束（如 `MOD DEVICE_INFO:Mcc=460,Mnc=00,Encryption=1` 全部 fill 但 mml_command_sub_fields 仅定义 Mcc/Mnc 两个字段时如何处理）？ | **严格校验**：未知 mml_code 累加 `unknown_codes`，前端 toast 提示；执行时跳过 unknown_codes。**不在 parse 阶段拒绝**（用户体验：让用户看到 unknown_codes 后手动修正）。 |

## N.11 S3 实施顺序（推荐）

```
D1 (4h):  mml_renderer.go + mml_renderer_test.go（纯函数，先写测试 RED → GREEN）
D1 (4h):  mml_parser.go + mml_parser_test.go（同上）
D2 (4h):  group_tree_repository.go + console_service.go buildGroupTree
D2 (4h):  console_service.go GetCommandSubFields / RenderMML / ParseMML（透传）
D3 (4h):  ExecuteStatements service 层 + tr069_payload.go statement 派发
D3 (4h):  console_handler.go 5 endpoint + provider DI + router 注册 + 单测
```

S3 出口门复用 §B3：build / test 全绿 / 无新 TODO/FIXME / 无新 any（payload struct 严格类型）/ 无 carrier 硬编码。

## N.12 与既有 Sprint B 能力的兼容性

| Sprint B 能力 | P1 兼容方式 |
|--------------|------------|
| `Fanouter.SetSequentialMode(true)` | P1 多 statement 设 sequentialMode=true 强制串行；单 statement 维持并发 |
| `Sequencer.OnTaskCompleted` | 链式入队 statement i+1 行（既有 cmd_idx 语义直接复用） |
| `ResultAggregator` | per-device frame SSE 推送（T-0102-d 已就绪） |
| `script_parser.go::ParseScriptContent` | 旧 script 单行 entry parser 保留；P1 加 ParseMMLString 平行通道，handler 检 content-type 路由 |
| `ExecuteGroup` (`POST /mml/groups/:id/execute`) | 保留作为快捷批量入口（一键执行 group 下所有 LST），不与 P1 statements 入口冲突 |
| `ExecuteCommand` 旧 (`POST /mml/execute`) | 升级为 ExecuteStatements 入口，老 payload (command_code + parameters) 转单元素 statements fallback |

## N.13 S2 出口门核查

- [✓] 接口契约明确（§N.2 5 endpoints 完整 request/response struct + §N.5 service 方法签名）
- [✓] 迁移草案（N/A — P0 migration 000095 已落地，P1 无新 DDL）
- [✓] Carrier 差异点（§0.5 已锁定无差异，本任务三家一致）
- [✓] 观测埋点（§N.8 4 metric + 6 log 字段）
- [✓] 待定点 < 3（§N.10 列 3 个）

S2 PASS。

---

# 设计备忘（T-0123-P2-b S2 设计定稿，2026-05-14）

> P2-b 范围：仅**新增** 5 个 UI 组件（PRD §7.2 表格逐项），**不动** `Console/index.tsx`（那是 P2-c 三栏重构的范围）。
> 业务层（types / store / hooks / api）已由 P2-a `commit 3c760ce9` 全量落地，本期组件直接消费 `@core/*` 引用。

## O.1 文件清单（S3 实施输出）

5 个新组件，全部位于 `omcmb/webcode/src/pages/mml/Console/components/`：

| 文件 | 行数预估 | 主依赖 |
|------|----------|--------|
| `MmlEditor.tsx` | ~120 | antd `Input.TextArea`/`Button`/`Spin` + useMmlConsoleStore + useParseMML + useExecuteStatements |
| `SubFieldChecklist.tsx` | ~110 | antd `Checkbox`/`Tag`/`Tooltip` + useMmlConsoleStore (toggleSubField) |
| `SubFieldInputList.tsx` | ~130 | antd `Input`/`Tag` + useMmlConsoleStore (setValue) + READ_ONLY 过滤逻辑 |
| `InstancePicker.tsx` | ~80 | antd `Select` + useMmlConsoleStore (setRmvIndex) — GPV 探测延 P4 |
| `StepBar.tsx` | ~40 | antd `Steps` — 纯展示无 store 订阅 |

单测同目录 `__tests__/`（Vitest），每组件至少 2-3 case 覆盖正常路径 + 元数据差异化。

## O.2 5 组件 Props 与 store 接通

```ts
// MmlEditor — Step3 顶部 textbox + DO 按钮，与 mmlText 双向
interface MmlEditorProps {
  deviceSns: string[];                          // 上层（index.tsx）传入；DO 时附给 execute
  onExecuted?: (task: MMLTask) => void;         // 用于上层导航到任务详情
}
// 订阅：mmlText / parsePending / parseErrors
// 写入：setMmlTextDebounced(text, useParseMML.mutateAsync)
// 内置：DO onClick → 若 statements 含 changeApplies==='OnReboot' → Modal.confirm 后 execute

// SubFieldChecklist — LST 视图，勾选 sub_field
interface SubFieldChecklistProps {
  statement: Statement;                         // 上层按 activeStatementUid 取
}
// 订阅：statement.subFields / statement.selectedSubFieldIds
// 写入：toggleSubField(statement.uid, subFieldId)
// 默认勾选：subField.defaultSelected===true（PRD §1.3 LST 全勾选语义）

// SubFieldInputList — MOD/ADD 视图，输入框列表
interface SubFieldInputListProps {
  statement: Statement;                         // operationType in ('MOD','ADD')
}
// 订阅：statement.subFields / statement.values
// 写入：setValue(statement.uid, mmlCode, value)
// 过滤：MOD 跳过 accessType==='READ_ONLY' 的 sub_field（PRD §7.3）
// ADD：保留全部输入项（含 target_object 行，由 statement.commandCode 携带）

// InstancePicker — RMV 视图，选要删的实例 index
interface InstancePickerProps {
  statement: Statement;                         // operationType==='RMV'
}
// 订阅：statement.rmvInstanceIndex
// 写入：setRmvIndex(statement.uid, index)
// P2-b 实现：手输 index Number 输入 + min=0 校验（GPV 自动探测延 P4，§O.9 待定点 1）

// StepBar — 顶部 1-2-3-4 进度指示，纯 props 驱动
interface StepBarProps {
  current: 1 | 2 | 3 | 4;
  labels?: { step1: string; step2: string; step3: string; step4: string };
}
// 不订阅 store；index.tsx 按"设备选过 / 命令选过 / 有 statements / 有结果"派生 current
```

## O.3 元数据驱动 UI 差异化落地（PRD §7.3 → 具体规则）

| sub_field 元数据 | SubFieldChecklist（LST） | SubFieldInputList（MOD） | SubFieldInputList（ADD） |
|------------------|--------------------------|---------------------------|---------------------------|
| `accessType==='READ_ONLY'` | 勾选行 + 锁 icon | **过滤掉**（不渲染） | **过滤掉** |
| `accessType==='READ_WRITE'` | 勾选行 + 编辑 icon | 输入框 | 输入框 |
| `isObject && supportsAdd` | 行内 "实例" 提示文案 | — | — |
| `changeApplies==='OnReboot'` | 行末黄色 Tag "需重启生效" | 输入框右侧同 Tag | 同 MOD |
| `isRequired===true` | — | label 前红色 `*` | label 前红色 `*` |
| `constraintText` 非空 | — | 输入框下方灰色 hint | 同 MOD |
| `defaultSelected===true` | 初次进入自动勾上 | — | — |
| `defaultValue` 非空 | — | 输入框 placeholder=`defaultValue` | 同 MOD |

## O.4 i18n key 清单（zh-CN + en-US 同步）

新增 keys（落 `frontend-core/src/i18n/{zh-CN,en-US}/mml.ts`，与 P2-a 11 keys 同文件追加）：

```ts
'mml.console.editor.execute': 'DO' / 'DO'                            // DO 按钮（中英均保留 DO 字样匹配老 OMC）
'mml.console.editor.parsing': '解析中...' / 'Parsing...'
'mml.console.editor.onRebootConfirm': '当前命令包含需重启生效字段，确认执行？' / '...'
'mml.console.checklist.readOnly': '只读' / 'Read-only'
'mml.console.checklist.onReboot': '需重启生效' / 'Requires reboot'
'mml.console.input.required': '必填' / 'Required'
'mml.console.input.constraint': '约束' / 'Constraint'
'mml.console.picker.indexLabel': '实例 index' / 'Instance index'
'mml.console.picker.indexHelp': '从 0 开始；可通过 LST 查询当前实例数量' / 'Starts at 0; use LST to view current instance count'
'mml.console.stepBar.step1': '选择设备' / 'Select device'
'mml.console.stepBar.step2': '选择命令' / 'Select command'
'mml.console.stepBar.step3': '配置参数' / 'Configure'
'mml.console.stepBar.step4': '查看结果' / 'View result'
```

12 个新 key × 2 lang = 24 行 i18n 增量。

## O.5 单测策略（Vitest + Testing Library）

每组件最少 3 case（共 ≥ 15 case）：

| 组件 | case 1 | case 2 | case 3 |
|------|--------|--------|--------|
| MmlEditor | 输入 → 300ms 防抖后 parseFn 被调一次 | parseErrors 非空时显示红条 | DO 按钮含 OnReboot 弹 Modal.confirm |
| SubFieldChecklist | READ_ONLY 项默认勾选不可改 | 点击 toggle 调 store.toggleSubField | OnReboot 字段显示 Tag |
| SubFieldInputList | MOD 模式过滤 READ_ONLY | required 字段缺值时 label `*` 红 | constraintText 显示在 hint 处 |
| InstancePicker | 输入 5 → store.setRmvIndex(uid, 5) | 负数被拒（min=0） | 空 → setRmvIndex(uid, undefined) |
| StepBar | current=3 时第 3 步高亮 | labels prop 覆盖默认 i18n | 不订阅 store（无 re-render 噪声） |

Store / API 走 Vitest mock，不依赖真后端。

## O.6 多皮肤兼容

5 个组件都通过 `@core/*` 引用业务层：
- `@core/types/mmlConsole` — Statement / SubFieldDef / MMLOperationType
- `@core/store/mmlConsoleStore` — useMmlConsoleStore + Statement 类型
- `@core/hooks/api/useMmlConsole` — useParseMML / useExecuteStatements

组件文件本身只在 `webcode/src/`，**不进 `frontend-core/`**（按 §7.5：UI 壳本期只动主皮肤）。
webcode-v2 / webcode-v3 不需要镜像该组件——它们若需要 MML Console 自行实现 UI 即可。

## O.7 OnReboot 二次确认 + DO 按钮责任划分

- **DO 按钮**只挂在 `MmlEditor`（PRD §7.1 ASCII 图锁定位置）
- DO onClick 内部：扫 `statements[].subFields.find(sf => sf.changeApplies==='OnReboot' && (selectedSubFieldIds.includes(sf.id) || values[sf.mmlCode]!==undefined))` → 命中则 Modal.confirm 后再 mutate
- `SubFieldChecklist` / `SubFieldInputList` 只负责显示 OnReboot Tag，不做确认动作

## O.8 P2-b 不做项（边界澄清，避免 scope creep）

1. **不重写 `Console/index.tsx`**（属 P2-c）
2. **不删除 `CommandTree.tsx` / `ParamPathPanel.tsx` / `ParamFormRenderer.tsx`**（属 P2-c）
3. **不接入 GPV 探测实例**（InstancePicker 自动列表，属 P4）
4. **不做 Customized PrivateTemplate adapter**（属 P4）
5. **不做 admin Catalog 管理 UI**（属 P3）
6. **不做 E2E spec / DoD 验收**（属 P2-d）

## O.9 待定点

1. **InstancePicker GPV 自动探测延后**：P2-b 仅手输 index + min=0 校验；P4 接入 `/ops/commands/rpc` GPV 探测，反查实例列表填 antd Select。当前文案明确提示"可通过 LST 查询"。

**待定点合计 = 1**（< 3 ✅）

## O.10 S2 出口门核查

- [✓] 接口契约明确（§O.2 5 组件 Props + store actions 绑定一一锁定）
- [N/A] 迁移草案（前端无 DB 变更）
- [✓] Carrier 差异点列出（§0.5 锁定三家一致，本任务无 carrier 分支代码）
- [✓] 观测埋点（前端组件按需 `console.warn` parse 错误；遥测埋点延 P4 统一接入）
- [✓] 待定点 < 3（§O.9 列 1 个）

S2 PASS。

---
