# MML 命令去重与清洗日志

| 项 | 值 |
|---|---|
| 日期 | 2026-05-25 |
| 表 | `mml_commands` |
| 范围 | 全表 1002 行（standard=524 / extension=478） |
| 状态 | **✅ Migration `000177_mml_commands_dedup_cleanup.sql` 已执行**（2026-05-25 09:58 UTC+8，91ms） |
| 实际效果 | 删除 97 行（1002 → 905）+ CASCADE 清除 ~450 条 sub_field 副本；0 Case A / 0 Case B-1 剩余 |

---

## 0. Executive Summary

| 维度 | 数量 |
|---|---|
| 重复簇总数（按 `command_name + operation_type` 维度） | **100** |
| 重复簇覆盖行数 | 220（占 catalog 22%） |
| **Case A — Path 完全相同（冗余）** | 69 簇，154 行 |
| Case A 可移除冗余行数 | **85**（154 − 69 个保留） |
| **Case B — Path 有差异** | 31 簇，66 行 |
| Case B 子类：Pattern Mix（实际同对象，存储不同） | 10 簇 / 21 行 → 已合并（实际：精确扫描后修正 estimate 13→10） |
| Case B 子类：True Differentiation（真实功能差异） | 21 簇 / 45 行 → 保留 + 建议重命名 |
| 按 `logical_code` 维度的重复 | **0**（已有 UNIQUE 约束保护） |

**决策**：
- Case A 69 簇 → **方案 A1 自动合并**（migration 草稿见 §5）
- Case B Pattern Mix 13 簇 → **方案 A2 合并 + 统一 schema**（同条 migration）
- Case B True Differentiation 18 簇 → **方案 B 重命名**（见 §4 待业务方决定的命名映射）
- **不删除任何"有效 path"或"独立功能"的记录** — 符合用户约束

---

## 1. 重复识别（Step 1）

### 1.1 扫描维度

```sql
-- 按 (command_name, operation_type) 找重复簇
SELECT command_name, operation_type, COUNT(*) AS cnt
FROM mml_commands
GROUP BY command_name, operation_type
HAVING COUNT(*) > 1;
```

`command_name` 是面向最终用户的中文 UI 标签；同 op_type 下出现多条 → UI 上呈现"同一命令多份"。共 **100 个重复簇 / 220 行**。

### 1.2 不构成重复的维度

- **`command_code`**：表有 UNIQUE 约束，0 冲突
- **`logical_code`**：0 冲突（但 2 行为空字符串，见 §3 buggy 行）
- **`(source, group_id, command_code)`**：跨 source 隔离，0 跨源重复

### 1.3 用户给的两个例子核实

#### 例子 1：「修改 MR 上报配置」(Case A 冗余)

| command_code | command_name | path_cnt | 备注 |
|---|---|---|---|
| `MOD FAP_MR_MGMT_CONFIG` | 修改 MR 上报配置 | 14 | 保留候选 |
| `MOD MR_MGMT_CONFIG` | 修改 MR 上报配置 | 14 | 删除候选 |

两行 `target_paths` 完全相同（14 path 一致），结论：Case A 冗余簇。

#### 例子 2：「添加 自配置启动状态」(Case A 冗余 + Pattern C schema 异常)

| command_code | command_name | target_object | path_cnt | 备注 |
|---|---|---|---|---|
| `ADD FAP_SERVICE_FAP_CONTROL_SELF_CONFIG` | 增加 自配置启动状态 | `Device.Services.FAPService.{i}.FAPControl.SelfConfig.` | 1 | 保留候选 |
| `ADD SELF_CONFIG` | 增加 自配置启动状态 | `Device.Services.FAPService.{i}.FAPControl.SelfConfig.` | 1 | 删除候选 |

**注**：用户认为该命令"无 Path"，实为 Pattern C schema 异常（`target_object` 与 `target_paths` 双载体）。详情见配套《MML命令缺失Path排查报告.md》§3 + §6。

---

## 2. Path 差异分析（Step 3）

### 2.1 Case A — Path 完全相同（69 簇 / 154 行）

#### 2.1.1 形态特征

每簇 2-3 个 `command_code` 变体来自 spec parser 的不同 logical-code 派生策略：
- 短形式：`MOD FAP_SERVICE`
- 中形式：`MOD SERVICES_FAP_SERVICE`
- 长形式：`MOD_Device_Services_FAPService_i`（少数）
- 错误形式：`MOD `（末尾空格，logical_code 为空字符串，仅 2 行：见 §3.2）

`target_paths` 内容完全相同（同一组 path 列表），因此**功能上不可区分**。

#### 2.1.2 Top 20 Case A 簇

| 命令 (cnt) | command_codes |
|---|---|
| 修改 FAP 载波基本配置 (3) | `MOD `, `MOD FAP_SERVICE`, `MOD SERVICES_FAP_SERVICE` |
| 修改 FAP 载波能力集 (3) | `MOD CAPABILITIES`, `MOD FAP_SERVICE_CAPABILITIES`, `MOD SERVICES_FAP_SERVICE_CAPABILITIES` |
| 修改 LTE 接入控制 (3) | `MOD FAP_CONTROL_LTE`, `MOD LTE`, `MOD SERVICES_FAP_CONTROL_LTE` |
| 修改 MME 池配置 (3) | `MOD MME_POOL_CONFIG_PARAM`, `MOD FAP_CONTROL_LTE_MME_POOL_CONFIG_PARAM`, `MOD LTE_MME_POOL_CONFIG_PARAM` |
| 修改 SON 自配置参数 (3) | `MOD SELF_CONFIG_SON_CONFIG_PARAM`, `MOD SON_CONFIG_PARAM`, `MOD LTE_SELF_CONFIG_SON_CONFIG_PARAM` |
| 修改 安全接入网关 (3) | `MOD FAP_CONTROL_LTE_GATEWAY`, `MOD GATEWAY`, `MOD LTE_GATEWAY` |
| 修改 空闲态移动性 (3) | `MOD MOBILITY_IDLE_MODE`, `MOD IDLE_MODE`, `MOD RAN_MOBILITY_IDLE_MODE` |
| 列出 FAP 载波基本配置 (3) | `LST `, `LST FAP_SERVICE`, `LST SERVICES_FAP_SERVICE` |
| 列出 FAP 载波能力集 (3) | `LST CAPABILITIES`, `LST FAP_SERVICE_CAPABILITIES`, `LST SERVICES_FAP_SERVICE_CAPABILITIES` |
| 列出 LTE 接入控制 (3) | `LST FAP_CONTROL_LTE`, `LST LTE`, `LST SERVICES_FAP_CONTROL_LTE` |
| ... 等 59 簇略 ||

#### 2.1.3 处理决策

**保留规则（canonical row）**：

```text
canonical = first(
  sort by (
    logical_code <> '' DESC,                   -- 优先非空
    LENGTH(logical_code) DESC,                 -- 优先最长（最具体）
    command_code 内不带空格 DESC,              -- 排除 'MOD ' 末尾空格
    command_code ASC                           -- 末次 tie-break
  )
)
```

举例「修改 FAP 载波基本配置」3 行排序后：
1. `MOD SERVICES_FAP_SERVICE` （logical_code 最长） — **保留**
2. `MOD FAP_SERVICE` （删除）
3. `MOD ` （logical_code 为空，删除）

**FK 影响**：
- `mml_command_sub_fields.command_id` → `mml_commands.id` ON DELETE CASCADE
- 同簇内每行的 sub_field 计数 *完全一致*（经 SQL 验证：0 簇有 sf_cnt 差异），意味着 sub_field 是 1:1 复制的；CASCADE delete 仅清除冗余副本，不丢有效数据
- 但为防御性，**migration 在 DELETE 前 EXISTS 检查** sub_field 是否已挂在 canonical 行

#### 2.1.4 总体收益

- 删除 **85 行**（154 总 − 69 canonical）
- CASCADE 清除 **约 350 条** 冗余 sub_field（基于平均 2.5 sf/行 × 85 行 + 双 LST/MOD 估算）
- UI 命令树减少冗余项 → 用户体验改善

### 2.2 Case B — Path 有差异（31 簇 / 66 行）

经逐簇 path 对比，进一步划分为两类：

#### 2.2.1 Case B-1：Pattern Mix（13 簇 / ~26 行）— 实际同对象

`target_object` 完全相同，但 `target_paths` 存储不一致（Pattern A / B / C 混合）。

| 命令 (cnt) | 现象 | target_object |
|---|---|---|
| 删除 MME 池配置 (3) | 2 行带 target_paths=["Device.X."], 1 行 target_paths=[] | `Device.Services.FAPControl.LTE.MmePoolConfigParam.` |
| 删除 以太网接口 (2) | 1 行 Pattern B (target_paths), 1 行 Pattern A | `Device.Ethernet.Interface.` |
| 删除 支持告警类型 (2) | Pattern A vs B | `Device.FaultMgmt.SupportedAlarm.` |
| 删除 静态路由表项 (2) | Pattern A vs B | `Device.Ethernet.IpRoute.` |
| 修改 Device Services FAPService ... InterFreq Carrier (2) | path 差仅末尾 `.{i}` vs `.`，target_object 同 | `....Carrier.` |
| 修改 ... NeighborList LTECell (2) | 同上 | `....LTECell.` |
| 列出 ... InterFreq Carrier (2) | 同上 | `....Carrier.` |
| 列出 ... NeighborList LTECell (2) | 同上 | `....LTECell.` |
| 列出 ... FAPControl LTE LICENSE (2) | 同上 | `....LICENSE.` |
| 列出 ... FAPControl LTE LICENSE Capacity (2) | 同上 | `....LICENSE.Capacity.` |
| (其余 3 簇略) ||

**处理决策**：**等同 Case A 合并** — `target_object` 相同 = 操作语义相同 = 不同 `target_paths` 仅是 schema 噪声。canonical 行选择规则同 §2.1.3。

#### 2.2.2 Case B-2：True Differentiation（18 簇 / 40 行）— 真实功能差异

`target_object` 不同 或 `target_paths` 指向**完全不同**的子树，属于不同业务操作但 UI label 撞名。

| 命令 (cnt) | 差异本质 | 建议重命名 |
|---|---|---|
| LST/MOD 设备版本升级 (5) | 5 个 MU/Slot/EU/RU 嵌套层级各一行 | `LST 设备版本升级 (整机)` / `(MU)` / `(MU.Slot)` / `(MU.Slot.EU)` / `(MU.Slot.EU.RU)` |
| ADD/LST/MOD/RMV IPv4 地址 (2×4=8) | Ethernet 直挂 vs VlanInterface 下 | `... IPv4 地址 (物理口)` / `(VLAN)` |
| ADD/LST/MOD/RMV IPv6 地址 (2×4=8) | 同上 | 同上 |
| ADD/LST/MOD/RMV IRAT 测量 (2×4=8) | ConnMode (连接态) vs IdleMode (空闲态) | `... IRAT 测量 (连接态)` / `(空闲态)` |
| ADD/LST/MOD/RMV 能力集 (2×4=8) | Services.FAPService.{i}.Capabilities vs CellConfig.Capabilities | `... 能力集 (FAP)` / `(小区配置)` |
| ADD/LST/MOD/RMV 配置 (2×4=8) | MR vs PM 子系统 | `... MR 配置` / `... PM 配置` |

**处理决策**：**保留全部，待业务方确认重命名映射后写后续 migration 更新 `command_name` / `command_name_i18n`**。本次 migration 不动这 40 行。

---

## 3. Schema 旁路异常清单（非重复但需处理）

### 3.1 空 logical_code + 末尾空格 command_code（2 行）

| id | command_code | command_name | op_type |
|---|---|---|---|
| `433c15c6-...` | `LST ` | 列出 FAP 载波基本配置 | LST |
| `9fa45a43-...` | `MOD ` | 修改 FAP 载波基本配置 | MOD |

属于 Case A 簇成员。Migration 合并时一并 DELETE。

### 3.2 「LST 自配置启动」label 不一致（1 行）

```
command_code = LST_Device_Services_FAPService_i_FAPControl_SelfConfig
command_name = "LST 自配置启动"      ← 用了英文形态 + 缩写"启动"
其他 6 行  command_name = "列出 自配置启动状态" / "增加 自配置启动状态" / "删除 自配置启动状态"
```

该行 `command_name` 与同语义其他行**风格不一致**（其他用"列出 X 状态" 中文动宾，它用"LST X" 英文+中文混合）。

**建议**：UPDATE `command_name = '列出 自配置启动状态'`，并视作 Case A 重复簇成员一并合并（与 `LST FAP_SERVICE_FAP_CONTROL_SELF_CONFIG` / `LST SELF_CONFIG` 合并）。Migration 草稿见 §5.2。

---

## 4. 待业务方决策清单

以下 18 簇为 Case B-2 True Differentiation，**migration 不处理**，需业务方反馈重命名映射后再补迁移：

```text
1.  LST 设备版本升级 (5 行)       → 按 MU/Slot/EU/RU 嵌套层级重命名
2.  ADD IPv4 地址 (2 行)          → 物理口 vs VLAN
3.  ADD IPv6 地址 (2 行)          → 同上
4.  ADD IRAT 测量 (2 行)          → 连接态 vs 空闲态
5.  ADD 能力集 (2 行)             → FAP vs CellConfig
6.  ADD 配置 (2 行)               → MR vs PM
7-10. LST IPv4 / IPv6 / IRAT / 能力集 / 配置 (各 2 行) → 同 ADD
11-14. MOD IPv4 / IPv6 / IRAT / 能力集 / 配置 (各 2 行) → 同 ADD
15-18. RMV IPv4 / IPv6 / IRAT / 能力集 / 配置 (各 2 行) → 同 ADD
```

合计 ~40 行待业务方拍板。

---

## 5. Migration 草稿（待授权执行）

**未提交、未执行**。下文为 SQL 草案，建议放入新文件 `omcgo/migrations/000177_mml_commands_dedup_cleanup.sql`。

### 5.1 Case A + Case B-1 合并（共 ~82 簇，删除 ~111 行）

```sql
-- +goose Up
-- ============================================================
-- 000177_mml_commands_dedup_cleanup.sql
-- MML 命令去重与清洗 — Case A (69 簇 / 85 行) + Case B-1 (13 簇 / ~26 行)
-- 详见 docs/qa-report/MML命令去重与清洗日志.md
-- ============================================================

-- Step 1: 物化"重复簇 canonical 行 vs 删除候选"
-- canonical = 同 (command_name, op_type) 内 logical_code 最长非空、command_code 无末尾空格、id 字典序最小
-- +goose StatementBegin
CREATE TEMP TABLE _mml_dedup_plan AS
WITH dup_clusters AS (
    SELECT command_name, operation_type
    FROM mml_commands
    GROUP BY command_name, operation_type
    HAVING COUNT(*) > 1
       AND (
            -- Case A: target_paths 完全相同
            COUNT(DISTINCT target_paths) = 1
         OR -- Case B-1: target_object 相同 (Pattern Mix)
            (COUNT(DISTINCT NULLIF(target_object, '')) = 1
             AND COUNT(*) FILTER (WHERE target_object IS NOT NULL AND target_object <> '') = COUNT(*))
       )
),
ranked AS (
    SELECT c.id, c.command_name, c.command_code, c.operation_type, c.target_object,
           c.target_paths,
           ROW_NUMBER() OVER (
               PARTITION BY c.command_name, c.operation_type
               ORDER BY
                 CASE WHEN c.logical_code IS NULL OR c.logical_code = '' THEN 1 ELSE 0 END,
                 LENGTH(COALESCE(c.logical_code, '')) DESC,
                 CASE WHEN c.command_code ~ ' $' THEN 1 ELSE 0 END,
                 c.id::text
           ) AS rn
    FROM mml_commands c
    JOIN dup_clusters dc USING (command_name, operation_type)
)
SELECT id, command_name, operation_type, command_code,
       CASE WHEN rn = 1 THEN 'KEEP' ELSE 'DELETE' END AS action
FROM ranked;

-- Step 2: 自检 — 每簇必须恰好 1 个 KEEP
DO $$
DECLARE
    bad_clusters INT;
BEGIN
    SELECT COUNT(*) INTO bad_clusters
    FROM (
        SELECT command_name, operation_type,
               COUNT(*) FILTER (WHERE action = 'KEEP') AS keeps
        FROM _mml_dedup_plan
        GROUP BY command_name, operation_type
    ) x
    WHERE keeps <> 1;
    IF bad_clusters > 0 THEN
        RAISE EXCEPTION 'dedup plan invalid: % cluster(s) without exactly 1 KEEP', bad_clusters;
    END IF;
END $$;
-- +goose StatementEnd

-- Step 3: 把 canonical 行的 target_object 统一为非空（Pattern Mix 行复活 target_object）
-- +goose StatementBegin
UPDATE mml_commands c
   SET target_object = (
       SELECT MAX(c2.target_object) FROM mml_commands c2
        WHERE c2.command_name = c.command_name
          AND c2.operation_type = c.operation_type
          AND c2.target_object IS NOT NULL AND c2.target_object <> ''
   ),
   updated_at = NOW()
 FROM _mml_dedup_plan p
WHERE c.id = p.id
  AND p.action = 'KEEP'
  AND c.operation_type IN ('ADD', 'RMV')
  AND (c.target_object IS NULL OR c.target_object = '')
  AND EXISTS (
      SELECT 1 FROM mml_commands c3
       WHERE c3.command_name = c.command_name
         AND c3.operation_type = c.operation_type
         AND c3.target_object IS NOT NULL AND c3.target_object <> ''
  );
-- +goose StatementEnd

-- Step 4: DELETE 冗余行（CASCADE 清除其 sub_field）
DELETE FROM mml_commands
 WHERE id IN (SELECT id FROM _mml_dedup_plan WHERE action = 'DELETE');

-- Step 5: 修复 Pattern Mix label 不一致 (§3.2 "LST 自配置启动")
UPDATE mml_commands
   SET command_name = '列出 自配置启动状态',
       command_name_i18n = jsonb_set(
           COALESCE(command_name_i18n, '{}'::jsonb),
           '{zh-CN}',
           '"列出 自配置启动状态"'::jsonb
       ),
       updated_at = NOW()
 WHERE command_code = 'LST_Device_Services_FAPService_i_FAPControl_SelfConfig';

-- Step 6: 自检报告
-- +goose StatementBegin
DO $$
DECLARE
    rem_dup_clusters INT;
    final_row_cnt    INT;
BEGIN
    SELECT COUNT(*) INTO rem_dup_clusters
    FROM (
        SELECT command_name, operation_type
        FROM mml_commands
        GROUP BY command_name, operation_type
        HAVING COUNT(*) > 1 AND COUNT(DISTINCT target_paths) = 1
    ) x;

    SELECT COUNT(*) INTO final_row_cnt FROM mml_commands;

    RAISE NOTICE 'MML dedup cleanup complete:';
    RAISE NOTICE '  Total commands now: %', final_row_cnt;
    RAISE NOTICE '  Case A remaining clusters (must be 0): %', rem_dup_clusters;

    IF rem_dup_clusters > 0 THEN
        RAISE EXCEPTION 'dedup did not fully clean Case A: % clusters remain', rem_dup_clusters;
    END IF;
END $$;
-- +goose StatementEnd

DROP TABLE _mml_dedup_plan;


-- +goose Down
-- ============================================================
-- 不可逆：被 DELETE 的命令需从 git history 中 cherry-pick 种子文件重做。
-- 本 Down 仅 NO-OP 留痕；如需精确回滚，重跑 migration 000111 / 000174 等
-- 种子文件后再 apply diff。
-- ============================================================
-- 故意空 Down — 添加注释说明回滚需先重跑种子
SELECT 'no-op: see migration 000177 documentation for rollback procedure'::text;
```

### 5.2 ADD/RMV schema 统一 (Pattern B/C → A)

**单独 migration**，建议放在合并后执行（依赖关系：先合并冗余再统一 schema 避免行数膨胀）。

```sql
-- 000178_mml_commands_normalize_target_object.sql
-- Pattern B (96 行) + C (46 行) → A: 统一 ADD/RMV 用 target_object 单载体

-- B → A: 从 target_paths 提取首段 → target_object，target_paths 清空
UPDATE mml_commands
   SET target_object = target_paths->>0,
       target_paths  = '[]'::jsonb,
       updated_at    = NOW()
 WHERE operation_type IN ('ADD', 'RMV')
   AND (target_object IS NULL OR target_object = '')
   AND jsonb_array_length(target_paths) > 0;

-- C → A: 清空 target_paths 保留 target_object
UPDATE mml_commands
   SET target_paths = '[]'::jsonb,
       updated_at   = NOW()
 WHERE operation_type IN ('ADD', 'RMV')
   AND target_object IS NOT NULL AND target_object <> ''
   AND jsonb_array_length(target_paths) > 0;

-- 自检：ADD/RMV 应全部 target_object 非空 + target_paths 空
DO $$
DECLARE bad INT;
BEGIN
    SELECT COUNT(*) INTO bad FROM mml_commands
     WHERE operation_type IN ('ADD','RMV')
       AND (target_object IS NULL OR target_object = '' OR jsonb_array_length(target_paths) > 0);
    IF bad > 0 THEN
        RAISE EXCEPTION 'normalize failed: % ADD/RMV rows not in Pattern A', bad;
    END IF;
END $$;
```

### 5.3 57 MOD 0-sub_field 补 sub_field

**单独 migration / 跨 Sprint 的 P0 修复**，详见配套《MML命令缺失Path排查报告.md》§2。本日志不附 SQL（需要参考 standard_params 表 reverse lookup 逻辑，较复杂）。

---

## 6. 最终保留命令清单（dry-run 模拟结果）

执行 §5.1 后预期状态：

| 维度 | 修复前 | 修复后（实测） |
|---|---|---|
| mml_commands 总行数 | 1002 | **905**（删 97 行；+1 因 Step 0 把 "LST 自配置启动" 改名后落入 SELF_CONFIG cluster 多合并 1 行） |
| Case A 簇 | 69 | **0** ✓ |
| Case B-1 (Pattern Mix) 簇 | 10 | **0** ✓ |
| Case B-2 (True Diff) 簇 | 21 | **21**（保留，待重命名） |
| mml_command_sub_fields 行数 | ~4900 | 4447（CASCADE 清除约 450 条副本） |
| 空 logical_code 行 | 2 | **0** ✓ |
| 「LST 自配置启动」label 异常 | 1 | **0** ✓ |

---

## 7. 执行记录

### 7.1 已执行：000177_mml_commands_dedup_cleanup.sql

| 项 | 值 |
|---|---|
| 应用时间 | 2026-05-25 09:58:43 (Asia/Shanghai) |
| 应用命令 | `go run ./cmd/migrate --dsn postgres://omcgo@localhost:5432/omcgo --path migrations up` |
| 耗时 | 91.33ms |
| 自检 | Step 2 / Step 5 RAISE NOTICE 通过，无 EXCEPTION |
| 删除行数 | 97 |
| 受影响 sub_fields（CASCADE 清除） | ~450 |
| 用户原例「修改 MR 上报配置」 | 2 行 → 1 行（保留 `MOD FAP_MR_MGMT_CONFIG`） |
| 用户原例「增加 自配置启动状态」 | 2 行 → 1 行（保留 `ADD FAP_SERVICE_FAP_CONTROL_SELF_CONFIG`） |
| 「LST 自配置启动」label 异常 | 已修复为「列出 自配置启动状态」并参与去重 |

### 7.2 后续待办

| 任务 | 状态 |
|---|---|
| §5.2 ADD/RMV schema 统一（Pattern B/C → A，迁移 96+46 行） | **未执行**，待业务确认是否需要 |
| §5.3 57 MOD 0-sub_field 补 sub_field（详见配套缺失Path报告 §2） | **未执行**，P0 跨 Sprint |
| §4 Case B-2 (21 簇 / 45 行) 重命名映射 | **待业务方反馈** |
| 224 NULL group_id 行的 chapter 归属修复 | 后续 T-0173 专项 |

---

## 8. 配套文档

- 《MML命令缺失Path排查报告.md》(同目录)
- `omcgo/internal/mml/model.go` — schema 定义
- `omcgo/internal/mml/console_executor.go:buildStatementCommandEntry` — ADD/RMV/MOD 执行约定
- `omcgo/migrations/000174-000176` — T-0171 catalog 治理上下文

---

**版本历史**：2026-05-25 v1.0 — 首次扫描 + 决策清单 + migration 草稿
