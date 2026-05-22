# 中国移动TD-LTE南向数据模型 v2.3 技术实施方案

## 背景与目标

将 `cmcc-tdlte-southbound-data-model-v2.3.md` 中定义的 **624个标准参数、18个分组、71条命令** 落地为可运行的MML控制台功能，打通从"标准参数定义 → MML命令树 → TR-069 RPC下发"的完整链路。

本方案是对现有 `/mml/console` 功能的调整，非新建模块。

> **数据职责划分**：`standard_params` 表是产品/参数模型管理流程的产出，作为只读引用供
> seed importer 按 path 查找 ID 建立关联；importer 只写入版本/分组/命令/子字段这 4 张表。

---

## 现有基础设施（可复用）

| 模块 | 文件位置 | 状态 |
|------|---------|------|
| MML标准参数加载器 | `/omcgo/internal/config/parammodel/mmlstandardloader/` | 已有XML解析+DB upsert框架 |
| 参数模型翻译器 | `/omcgo/internal/config/parammodel/translator.go` | standard↔private双向完整 |
| TR-069 RPC全12方法 | `/omcgo/internal/acs/rpc/` | GetPV/SetPV/AddObj/DelObj均可用 |
| MML→TR069桥接 | `/omcgo/internal/mml/tr069_payload.go` | LST/MOD/ADD/RMV→RPC转换已有 |
| 命令树API | `/omcgo/internal/mml/console_handler.go` | GET /api/v1/mml/group-tree |
| 前端MML控制台 | `/omcmb/webcode/src/pages/mml/Console/` | 三栏布局完整 |
| 数据库表 | 000058/000090/000113 | standard_params/mml_commands/mml_command_sub_fields |

---

## 核心设计约束

1. **严格一级分组**：18个分组直接包含命令，禁止嵌套子分组
2. **MOD命令必须有RW路径**：如果一条命令关联的所有参数均为只读(R)，则不生成MOD操作叶子
3. **非可创建对象过滤**：17条白名单对象不生成ADD/RMV
4. **分组命名纯中文**：禁止包含英文路径/技术标识符
5. **命令中文名来自权威表**：§R-2.4的71条命令名为唯一真值源

---

## 涉及表结构

### 已有表

| 表名 | 来源迁移 | 用途 |
|------|---------|------|
| `mml_param_versions` | 000022 | 版本记录 cmcc-tdlte-v2.3（写入） |
| `standard_params` | 000058 | **只读引用**：数据由产品/参数模型页面预先维护，importer 仅按 `standard_path` 查找已有记录的 `id` 用于建立关联，不写入新行 |
| `mml_param_groups` | 000022 | 18个一级分组（chapter_code=SA~SR），写入 |
| `mml_commands` | 000090 | 190条派生命令，写入 |
| `mml_command_sub_fields` | 000090/000113 | M:N关联（`standard_path_id` FK 从 `standard_params` 查询所得），写入 |

### 新增表

| 表名 | 来源迁移 | 用途 |
|------|---------|------|
| `standard_commands` | 000151 | 命令元数据缓存 |

### 关系图

```
mml_param_groups (18个一级分组, chapter_code=SA~SR)
        │
        ▼ group_id
mml_commands (190条操作命令: LST/MOD/ADD/RMV)
        │
        ▼ command_id                    standard_params (只读引用, 不写入)
mml_command_sub_fields ──────────────► standard_path_id FK
                              （按 path 查询现有记录的 id 填入）
```

---

## Task 1：标准参数数据源准备 ✅

**产出**：`/omcgo/internal/config/parammodel/mmlstandardloader/seeds/cmcc_tdlte_v23.json`
**提取脚本**：`/tmp/extract_params_to_json.py`

校验结果：分组=18, 命令=71, 参数=624, 非可创建对象=17

---

## Task 2：数据库Schema ✅

**迁移文件**：`/omcgo/migrations/000151_standard_commands.sql`

- `standard_params` 表已存在（000058），无需重建
- 000151 仅创建 `standard_commands` 表（version_code VARCHAR(50) FK）
- 外键使用 `version_code` 而非 BIGINT id（与现有表一致）

---

## Task 3：Seed加载器实现 ✅

**新增文件**（`/omcgo/internal/config/parammodel/mmlstandardloader/`）：
- `json_parser.go`：JSON seed解析
- `command_derivator.go`：操作叶子派生 + logical_code自动消歧
- `seed_importer.go`：单事务幂等UPSERT 4张表（不写入 `standard_params` / `mml_params`）
- `seed_importer_test.go`：数量验证+派生规则测试

**standard_params 不写入，仅关联**：importer 不向 `standard_params` 表写入任何行，
该表数据由产品/参数模型管理页面预先维护。importer 在起步阶段按 `standard_path`
批量 SELECT 已有记录的 `id`，用于填入 `mml_command_sub_fields.standard_path_id`
建立 M:N 关联。对于在 `standard_params` 中未找到的 path，打印 warning 并跳过对
应 sub_field（不中断导入）。

**事务流程**：`mml_param_versions → (查询 standard_params 获得 path→id 映射) → mml_param_groups → mml_commands → mml_command_sub_fields`（仅用 `command_id` + `standard_path_id`）

**派生结果**：190条命令（71 LST + 57 MOD + 31 ADD + 31 RMV）

**启动期调用**：
```go
mmlstandardloader.ImportSeedFile(ctx, pool, seedPath, logger)
```

**命令派生规则**：
- LST：有参数即生成，关联全部参数
- MOD：必须有≥1个RW参数，仅关联RW参数
- ADD/RMV：路径含{i} + 有RW + 不在非可创建白名单

---

## Task 4：API格式调整 ✅

**端点**：`GET /api/v1/mml/group-tree?format=flat`

**实现文件**：`/omcgo/internal/mml/flat_group_tree.go`

**响应格式**：
```json
{
  "groups": [
    {
      "code": "SA",
      "name": "设备信息参数管理",
      "commands": [
        {
          "id": "uuid-xxx",
          "name": "LST 设备基本信息",
          "object_path": ["Device.DeviceInfo.UserLabel", "Device.DeviceInfo.SerialNumber"]
        },
        {
          "id": "uuid-yyy",
          "name": "MOD 设备基本信息",
          "object_path": [
            {"path": "Device.DeviceInfo.UserLabel", "type": "string", "max_length": 64},
            {"path": "Device.DeviceInfo.PeriodicInformInterval", "type": "unsignedInt", "min": 1}
          ]
        },
        {
          "id": "uuid-zzz",
          "name": "ADD FAP载波",
          "object_path": "Device.Services.FAPService.{i}."
        }
      ]
    }
  ]
}
```

**格式规则**：
- LST → object_path 为 string[]（参数路径列表）
- MOD → object_path 为 object[]（含path + type + 约束）
- ADD/RMV → object_path 为 string（目标对象路径）
- 命令name格式：`<OP> <中文名>`

**兼容性**：原有 `?format=tree` 行为零变化

---

## Task 5：翻译器集成 ✅

- Translator 自动工作：TR-181设备 standardPath=privatePath 透传
- 无需额外 param_mappings 配置
- 若 `products.param_model_id IS NULL`，Registry 返回 ErrNoParamModel，调用方按标准路径透传

---

## Task 6：端到端验证 ✅

| 检查项 | 结果 |
|--------|------|
| go build ./... | ✅ 通过 |
| go vet（目标包） | ✅ 零警告 |
| MML单元测试 | ✅ 全部通过 |
| mmlstandardloader测试 | ✅ LST=71, MOD=57, ADD=31, RMV=31 |
| JSON seed数据 | ✅ 18分组/71命令/624参数 |
| 迁移文件无冲突 | ✅ 000151仅含standard_commands |
| API格式正确 | ✅ LST→string[], MOD→object[], ADD/RMV→string |

---

## 后续演进（不在v2.3范围内）

1. **动态范围校验**：LST设备NumberOfEntries → Redis缓存 → 前端实例号输入校验
2. **参数同步性能优化**：从逐参数RPC升级为部分路径RPC
3. **多运营商扩展**：联通/电信南向模型接入同一框架（version_code + carrier区分）
4. **参数变更追踪**：TimescaleDB存储参数历史值，支持趋势分析
5. **前端适配**：CommandTree组件适配 `?format=flat` 新响应格式
