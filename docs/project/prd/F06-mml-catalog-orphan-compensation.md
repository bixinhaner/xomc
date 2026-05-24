# PRD — MML catalog 孤儿 path 审计 + 补偿（T-0171）

| 项 | 值 |
|---|---|
| Backlog | T-0171 |
| Type / Prio | feat+refactor / P2 |
| Sprint / Owner | sprint-12 / Claude |
| Est | M (~1d，已完成 — 补流程) |
| Deps | T-0170 ✅（sub_field paramModel 过滤）+ T-0098 ✅（standard_params/param_mappings schema） |

---

## 1. 业务背景

T-0170 修复 sub_field/fanout 后，DB 现状 standard_params 共 2168 行，但仅 624 行被 mml_command_sub_fields 引用。剩余 **1544 行"孤儿 path"** 既不属于 cmcc-tdlte-v2.3 spec §R-2.4 的 71 个 group，也未被任何命令操作。这些 path 主要是：

- BAICELLS BLQ/MLN 私有扩展（如 `Device.DeviceInfo.AntennaInfo.*` 18 path / `Device.DeviceInfo.1588_*` / `Device.DeviceInfo.AmbrLimitSwitch`）
- 5G NR 扩展子树（`Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.*` 100+ paths）
- GSM 设备子树（`DeviceGSM.Bts.*` / `DeviceGSM.Msc.*` / `DeviceGSM.Mgw.*`）
- SAS / IPsec / 板卡扩展子树

**后果**：用户在 MML 控制台看不到这些 path 对应的命令叶子；前端 quicksettings 通过 param_mappings 仍可用，但 MML 命令树覆盖率不全。

## 2. 用户故事

### 2.1 网管运维 — MML 命令树全覆盖
> 作为运维，**我希望 MML 控制台覆盖 BAICELLS 私有扩展 path**（如 AntennaInfo.Azimuth），能在命令树中找到对应的 LST/MOD 命令操作这些参数，而不是只能用 quicksettings 间接操作。

### 2.2 DBA — 审计可追溯
> 作为 DBA，**我希望孤儿 path 有独立审计表记录**（不污染业务表），便于后续业务方按 disposition 字段决策处置策略。

### 2.3 业务方 — 扩展 catalog 隔离
> 作为业务方，**我希望扩展派生的命令与 spec v2.3 'standard' catalog 完全隔离**（source='extension'），便于 admin UI 后续清理或独立维护。

## 3. 验收标准（Given-When-Then）

### GWT-1（孤儿 path 审计完整）
- **Given** standard_params 中存在 1544 条孤儿 path
- **When** goose 应用 migration 000173
- **Then** `mml_catalog_orphan_paths_audit_t0171` 表写入 1544 行，含 standardPath + object_prefix + access/data_type 快照

### GWT-2（孤儿 path 100% 补偿）
- **Given** 审计表已建
- **When** goose 应用 migration 000174
- **Then**:
  - 新增 ≥38 个 chapter 分组（source='extension'，按 path 顶层 1-2 段聚合）
  - 派生 LST 命令 ≥270 个（每个 object_prefix 1 个 LST）
  - 派生 MOD 命令 ≥200 个（仅 RW path 子集）
  - sub_field 关联 ≥2700 个
  - **1544 条 path 全部覆盖**（漏覆盖数 = 0）

### GWT-3（父子命令 path 无重叠 — 用户硬约束）
- **Given** migration 000174 应用完
- **When** 跑自检 SQL
- **Then**:
  - LST 命令两两间 target_paths 交集 = 0
  - 例：`LST EXT_DEVICEINFO` 的 target_paths **不含** `Device.DeviceInfo.EU.*` / `Device.DeviceInfo.AntennaInfo.*` 等子树 path
  - Step 5 自检 `RAISE EXCEPTION` 强约束

### GWT-4（隔离不污染 standard / admin）
- **Given** 应用前 standard catalog 18 chapter + 524 command
- **When** 应用 + 重跑 reset 脚本
- **Then** standard 18 chapter + 524 command **完全不动**；仅 source='extension' 行受影响

### GWT-5（独立 reset 脚本）
- **Given** DBA 想快速清理 extension 数据不走 goose
- **When** 跑 `omcgo/scripts/mml_catalog_extension_reset.sql`
- **Then** 删 478 命令 + 38 chapter + 2795 sub_field，standard / admin 全保留

## 4. 运营商差异矩阵

**无运营商差异**。孤儿 path 多数是 BAICELLS 私有，但补偿规则按 path 形态派生，三家通用。

## 5. 非目标

- 不动 standard_params schema（用户拍板：全局字典不加 is_supported）
- 不自动派生 ADD/RMV 命令（保守 — 含 {i} 但是否允许 AddObject 是业务决策）
- 不补 BLQ.xml 映射（孤儿 path 多是私有扩展，跨 product 通用性不明）
- 不修 product_class_patterns（T-0142 独立）

## 6. 依赖

- T-0098 ✅ standard_params + param_mappings + mml_command_groups schema
- T-0123 ✅ mml_command_sub_fields(standard_path_id) schema
- T-0169 ✅ T-0169 完成的 spec parser 工具 — 本任务的"扩展派生"是它的补集（spec parser 派生 standard catalog，本任务派生 extension catalog）
- T-0170 ✅ sub_field 按 paramModel 过滤已上线，本任务派生的 extension 命令会与 T-0170 协同工作

## 7. 度量

- 1544 孤儿 path 覆盖率 100%
- 父子命令 path 重叠数 = 0
- migration up: 605ms / down: 169ms（实测）
- standard catalog 完整性: 18 chapter + 524 command 完全保留

## 8. 风险

| ID | 风险 | 缓解 |
|---|---|---|
| R-NEW-T0171-1 | extension 命令在 UI 大量出现（272 LST + 206 MOD），用户感知"杂乱" | source='extension' + catalog_protected=false，admin UI 可批量清理；business 方可后续 reset 后重派生精简版 |
| R-NEW-T0171-2 | 未来 standard_params 新增 path 不被 extension 命令覆盖 | Step 5 自检规则会在 goose up 时检测 + RAISE EXCEPTION；重跑 down/up 可重新派生 |
| R-NEW-T0171-3 | logical_code VARCHAR(100) 截断对超长 5G NR path 引起 sub_field JOIN 失败 | 修复版同步 LEFT(_, 90) 截断（命令派生 + sub_field 反查一致）；自检规则验证 0 漏覆盖 |

## 9. 实施细节（S2 设计备忘）

### 9.1 chapter 切分规则
按 path 顶层 1-2 段聚合：
```
Device.DeviceInfo.*        → chapter:SX_DEVICE_DEVICEINFO_EXT
Device.Services.*          → chapter:SX_DEVICE_SERVICES_EXT
Device.FAP.*               → chapter:SX_DEVICE_FAP_EXT
DeviceGSM.Bts.*            → chapter:SX_DEVICEGSM_BTS_EXT
boardconf.HALOD.*          → chapter:SX_BOARDCONF_HALOD_EXT
... 共 38 chapter
```

### 9.2 command 切分规则（用户硬约束）
按 `object_prefix`（path 去最末段 + `.*`）。
**关键性质**：每条 path 仅归属一个最近的 object_prefix，父子命令的 target_paths **自然不相交**。

```
Device.DeviceInfo.AntennaInfo.Azimuth → object_prefix = Device.DeviceInfo.AntennaInfo.*
Device.DeviceInfo.UserLabel            → object_prefix = Device.DeviceInfo.*
Device.DeviceInfo.EU.{i}.Status        → object_prefix = Device.DeviceInfo.EU.{i}.*
```

→ `LST EXT_DEVICEINFO`（303 paths）不含 `Device.DeviceInfo.EU.*`（0 条）✓

### 9.3 op 派生规则（同 spec §R-3）
- **LST**: 恒生（path 集非空）
- **MOD**: 至少 1 条 RW path（access ∈ {READ_WRITE, WRITE_ONLY}）
- **ADD/RMV**: 暂不派生（业务方拍板后单独补）

### 9.4 logical_code 派生
```sql
LEFT(UPPER(REGEXP_REPLACE(REPLACE(REPLACE(
      REGEXP_REPLACE(object_prefix, '^Device\.', ''),
      '.*', ''),
      '.{i}', '_I'
  ), '\.', '_', 'g')), 90)
```

VARCHAR(100) 字段 LEFT(_, 90) 截断 + EXT_ 4 字符前缀 = 安全。命令派生与 sub_field 反查**两处必须用同样截断**（否则超长 5G NR path JOIN 失败导致漏覆盖；已修复）。

### 9.5 隔离设计
| 来源 | source 值 | 含义 |
|---|---|---|
| spec v2.3 71 group | `standard` | 不可改（catalog_protected=true） |
| 用户自定义 | `admin` | admin UI 可改 |
| 孤儿补偿 | `extension` | catalog_protected=false 允许 admin 后续清理 |

### 9.6 Step 5 自检规则（防退化）
迁移末尾 DO 块强约束 3 条规则，违反即 `RAISE EXCEPTION`：
1. LST 命令两两间 target_paths 无交集（父子不重叠）
2. 每条孤儿 path 至少出现于一个 LST 命令的 target_paths
3. 每条孤儿 path 至少有一个 sub_field 关联

## 10. 实施清单

| # | 制品 | 状态 |
|---|---|---|
| 1 | `omcgo/migrations/000173_mml_catalog_orphan_paths_audit_t0171.sql` | ✅ |
| 2 | `omcgo/migrations/000174_mml_catalog_orphan_paths_compensate_t0171.sql` | ✅ |
| 3 | `omcgo/scripts/mml_catalog_extension_reset.sql` | ✅ |
| 4 | Step 5 自检（防退化） | ✅ |
| 5 | 实测 down/up 演练 | ✅（173: 66ms up / 15ms down；174: 608ms up / 169ms down） |
| 6 | 完整覆盖验证 | ✅（0 漏 / 0 重叠 / 1544 path 100% 覆盖） |

## 11. 实测产物

```
38 个 chapter（chapter:SX_*_EXT）
272 个 LST 命令
206 个 MOD 命令
2795 条 sub_field 关联（LST 1544 + MOD 1251）
父子 0 重叠 / 0 漏覆盖
standard 18 chapter + 524 command 完全保留
```

---

**版本历史**：2026-05-24 v1.0 起草（S0 + S2 备忘合并；本任务功能已实施完毕，补走流程）
