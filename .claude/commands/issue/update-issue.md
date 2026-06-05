# 更新问题单/需求单状态

逐步推进 BaiBM 项目 **软件问题单**(tracker_id=1) 或 **需求单**(tracker_id=2) 的状态，
并根据类型自动识别对应阶段的必填字段。

## 使用方法
```
/update-issue <问题ID> [目标状态]
```

$ARGUMENTS 说明：
- 参数1（必填）: 问题 ID，如 `109561`
- 参数2（可选）: 目标状态，不填则自动推进到下一个状态
  - `assigned` — ASSIGNED
  - `progress` — IN-PROGRESS
  - `review` — CODE-REVIEW
  - `accepted` — CODE-ACCEPTED
  - `committed` — CODE-COMMITTED
  - `resolved` — RESOLVED
  - `reopen` — REOPENED

示例：
```
/update-issue 109561              (自动推进到下一状态)
/update-issue 109561 progress     (直接设为 IN-PROGRESS)
/update-issue 109561 review       (推进到 CODE-REVIEW)
/update-issue 109561 resolved     (推进到 RESOLVED)
```

---

## 状态流转图

```
OPEN(1) → ASSIGNED(7) → IN-PROGRESS(2) → CODE-REVIEW(14)
    → CODE-ACCEPTED(15) → CODE-COMMITTED(12) → RESOLVED(3)
        → TESTING(32) → VERIFIED(4)/CLOSED(5)

任意状态 → REOPENED(13) → IN-PROGRESS(2) ...
```

## 执行步骤

### 第 1 步：读取配置并获取当前状态

读取 `.claude/commands/issue/issue-tracker.json` 获取 `base_url` 和 `token`。

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/issues/${ISSUE_ID}.json?include=attachments,journals"
```

解析并展示当前状态，**同时识别 tracker 类型**：
```
问题 #${ID}: ${SUBJECT}
类型: ${TRACKER_NAME}（软件问题单/需求单）
当前状态: ${STATUS} → 下一步: ${NEXT_STATUS}
```

**关键**: 从返回的 `issue.tracker.id` 判断类型：
- `tracker_id == 1` → **软件问题单**（CODE-REVIEW 阶段有更多必填项）
- `tracker_id == 2` → **需求单**（CODE-REVIEW 阶段字段较少，但仍需填写自测结果）

### 第 1.5 步：检查关联问题单

获取问题详情时（使用 `?include=relations`），检查是否有关联的问题单。如果有，**必须一起更新到相同状态**。

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/issues/${ISSUE_ID}.json?include=relations"
```

关联类型说明：
- `copied_to` / `copied_from` — 复制关系（最常见，必须同步更新）
- `relates` — 相关联（应同步更新）
- `duplicates` / `duplicated` — 重复单（应同步更新）
- `blocks` / `blocked` — 阻塞关系（视情况而定）

**处理流程**：
1. 从返回的 `relations` 数组中提取所有关联的问题 ID
2. 查询每个关联问题的当前状态和指派人
3. 筛选出指派给自己（`assigned_to_id == 977`）且状态落后于目标状态的关联问题
4. 展示关联问题列表，提示用户将一起更新：
   ```
   检测到 1 个关联问题单：
     #110214 (copied_from) — 当前状态: OPEN — 将同步推进到 CODE-REVIEW
   ```
5. 用户确认后，**先更新主问题，再逐个更新关联问题**（使用相同的字段值）
6. 汇总报告中列出所有更新的问题单

**注意**：
- 只自动同步更新指派给自己的关联问题，其他人的问题只提示不操作
- 关联问题使用与主问题相同的 custom_fields 值（修正结论、���因、方案、备注等）
- 关联问题的解决版本需要根据各自的问题版本独立查找下一版本

### 第 2 步：确定目标状态

如果 $ARGUMENTS 中有目标状态，使用指定的状态。否则根据当前状态自动推进：

| 当前状态 | status_id | → 下一状态 | next_status_id |
|---------|-----------|-----------|---------------|
| OPEN | 1 | ASSIGNED | 7 |
| ASSIGNED | 7 | IN-PROGRESS | 2 |
| IN-PROGRESS | 2 | CODE-REVIEW | 14 |
| CODE-REVIEW | 14 | CODE-ACCEPTED | 15 |
| CODE-ACCEPTED | 15 | CODE-COMMITTED | 12 |
| CODE-COMMITTED | 12 | RESOLVED | 3 |
| RESOLVED | 3 | TESTING | 32 |
| TESTING | 32 | VERIFIED | 4 |
| REOPENED | 13 | IN-PROGRESS | 2 |

### 第 3 步：收集必填字段

根据目标状态，使用 AskUserQuestion 收集必填字段。以下是每个状态转换需要的字段：

---

#### → ASSIGNED (status_id=7)

需要确认/设置指派人。如果已指派给自己则直接推进。

**注意**：此状态转换还需要同时提交以下字段（否则返回 422 错误）：
- **漏测标识/Missed Test Indicator** (custom_field id:262) — 默认 `N`
- **非漏测缘由/Non-missed causes** (custom_field id:263) — 默认 `NULL`
- **开发工程师/Development Engineer** (custom_field id:23) — 固定填 `977`（部分问题单在此步也要求填写）

更新数据：
```json
{
  "issue": {
    "status_id": 7,
    "assigned_to_id": 977,
    "custom_fields": [
      {"id": 262, "value": "N"},
      {"id": 263, "value": "NULL"}
    ]
  }
}
```

---

#### → IN-PROGRESS (status_id=2)

需要填写：
- **开发工程师** (custom_field id:23) — 固定填 `977` (wangyong1246)，无需询问
- **测试建议/Testing Suggestions** (custom_field id:79) — 默认 `--`（否则返回 422 错误）

更新数据：
```json
{
  "issue": {
    "status_id": 2,
    "custom_fields": [
      {"id": 23, "value": "977"},
      {"id": 79, "value": "--"}
    ]
  }
}
```

---

#### → CODE-REVIEW (status_id=14)

这是关键步骤，**根据 tracker 类型填写不同字段**。

##### 软件问题单 (tracker_id=1) 的必填字段：

| 字段 | custom_field id | 必填 | 说明 |
|------|----------------|:----:|------|
| 修正结论/Correction Conclusion | 1 | 是 | 选项: FIXED / NOT-BUG / DUPLICATE / WONT-FIX / BY-DESIGN |
| 问题根因/Root Cause | 73 | 是 | 文字描述根本原因 |
| 修改方案/Modification Solution | 19 | 是 | 文字描述修改方案 |
| SVN/GIT修订号/Revision Number | 3 | 是 | commit hash，非代码修改（如 OAM 平台配置）填 `NA` |
| 问题类型/Issue Type | 253 | 是 | 选项: 新增需求bug / 历史遗留bug / 回归bug / 其他 |
| 问题引入人/Issue Introducer | 202 | 是 | 默认 `674` (liurong)，无需询问 |
| Review工程师/Review Engineer | 78 | 是 | 默认 `1096` (zhanglu1378)，无需询问 |
| 自测结果/Self-test Results | 21 | 是 | 默认 `OK`（否则返回 422 错误） |
| 解决版本/Resolved version | 13 | 是 | **必须填问题版本的下一个版本**（见下方版本规则），值为数组格式 |

##### 需求单 (tracker_id=2) 的必填字段：

| 字段 | custom_field id / notes | 必填 | 说明 |
|------|------------------------|:----:|------|
| SVN/GIT修订号/Revision Number | 3 | 是 | commit hash |
| Review工程师/Review Engineer | 78 | 是 | 默认 `1096` (zhanglu1378)，无需询问 |
| 自测结果/Self-test Results | 21 | 是 | 默认 `OK`（否则返回 422 错误） |
| 实现方案说明 | notes | 是 | 作为评论写入，描述实现方案 |

需求单**不需要**填写修正结论、问题根因、问题类型等字段，但**必须在 notes 中写明实现方案**，并同时填写自测结果 `OK`。

##### 需求单实现方案说明 (notes) 的编写规则：

使用 AskUserQuestion 让用户提供实现方案信息，然后按以下模板组织 notes 内容：

**重要：根据问题单所属项目区分 notes 内容格式**（见下方「项目与 CLI 命令对应规则」）。

```
【实现方案】
<实现方案的文字描述，包括修改了哪些模块、修改思路等>

【涉及MIB参数】（如涉及，必须给出以下信息）
参数名: <参数名称>
PARAM_ID: PARAM_ID_<参数名>
MIB路径: <完整MIB路径，如 FAP.0.TCP_AGENT_PARAMS.0.TCP_AGENT_ENABLE>
trpath: <TR-069 路径>
类型: <数据类型> | 权限: <读写权限> | 取值: <min>~<max> | 默认: <默认值>

--- 以下内容按项目区分 ---

[002软件-BaiBN 项目] 需要补充 CLI 命令：
GET: mibcli get <MIB路径:参数名>
SET: mibcli set <MIB路径:参数名>:=<值>

[003软件-BaiBM 项目] 不需要 CLI 命令，只备注参数层级即可

（如涉及多个MIB参数，逐个列出）
```

#### 项目与 CLI 命令对应规则

根据问题单的目标分支所属项目，决定 notes 中 MIB 参数部分的格式：

| 项目 | CLI 命令格式 | notes 中的 MIB 参数内容 |
|------|-------------|----------------------|
| **002软件-BaiBN** | `mibcli` | 参数层级 + trpath + **CLI 命令** |
| **003软件-BaiBM** | 不涉及 | **仅参数层级 + trpath**（不写 CLI 命令） |

**判断方法**：通过问题单的 `目标分支/Branch` (cf_257) 字段关联的版本，查询其所属 `project.name`：
- `002软件-BaiBN` → 备注完整 MIB 参数信息 + mibcli 命令
- `003软件-BaiBM` → 仅备注参数层级（MIB路径、PARAM_ID、trpath、类型、取值范围），不写 CLI 命令

**mibcli 命令格式**（仅 002软件-BaiBN 项目）：
```bash
# GET 命令
mibcli get ${MIB_DN}:${PARAM_NAME}
# 示例: mibcli get FAP.0.TCP_AGENT_PARAMS.0:TCP_AGENT_ENABLE

# SET 命令
mibcli set ${MIB_DN}:${PARAM_NAME}:=${VALUE}
# 示例: mibcli set FAP.0.TCP_AGENT_PARAMS.0:TCP_AGENT_ENABLE:=1
```

**示例 — 002软件-BaiBN 项目（带 CLI 命令）：**
```
【实现方案】
新增COLI模糊查询功能，在oam_coli模块中添加模糊匹配逻辑，
支持用户输入部分参数名进行搜索，返回匹配的参数列表。

【涉及MIB参数】
MIB Object: COLI_FILTER (parent: FAP)
trpath Object: Device.Services.FAPService.1.CellConfig.1.LTE.RAN.Common.
mibdn: FAP.0.COLI_FILTER.0
涉及分支: 5G/master

1. MIB: FAP.0.COLI_FILTER.0.COLI_FILTER_ENABLE
   PARAM_ID: PARAM_ID_COLI_FILTER_ENABLE
   trpath: Device.Services.FAPService.1.CellConfig.1.LTE.RAN.Common.ColiFilterEnable
   类型: U32 | 权限: READ_WRITE | 取值: 0~1 | 默认: 0
   GET: mibcli get FAP.0.COLI_FILTER.0:COLI_FILTER_ENABLE
   SET: mibcli set FAP.0.COLI_FILTER.0:COLI_FILTER_ENABLE:=1
```

**示例 — 003软件-BaiBM 项目（不带 CLI 命令）：**
```
【实现方案】
新增TCP Agent参数配置，在TCP_AGENT_PARAMS对象下新增MIB参数和trpath映射。

【涉及MIB参数】
MIB Object: TCP_AGENT_PARAMS (parent: FAP)
trpath Object: Device.FAP.TcpAgent.
mibdn: FAP.0.TCP_AGENT_PARAMS.0
涉及分支: 5G/release_BaiBNQ_3.0_TR5

1. MIB: FAP.0.TCP_AGENT_PARAMS.0.TCP_AGENT_ENABLE
   PARAM_ID: PARAM_ID_TCP_AGENT_ENABLE
   trpath: Device.FAP.TcpAgent.Enable
   类型: U32 | 权限: READ_WRITE | 取值: 0~1 | 默认: 0
```

**示例 — 不涉及 MIB 参数的需求单 notes：**
```
【实现方案】
优化同步源失步后的处理逻辑，当检测到同步失步时，
自动切换到备用同步源，并上报告警。修改涉及 oam_sync 模块。
```

**编写要点：**
- 使用 AskUserQuestion 先询问用户：1) 实现方案概述  2) 是否涉及 MIB 参数
- 如果涉及 MIB 参数，**优先从代码中自动查找**（见下方自动查找流程），再让用户确认/补充
- 如果用户已完成代码修改，从 `git diff` 中自动提取涉及的 MIB 参数信息

##### 从 OAM 代码自动查找 MIB 参数和 trpath 的流程：

用户提供关键字（参数名称、功能模块名等）后，按以下步骤在代码中自动定位：

**步骤 A — 查找 PARAM_ID（MIB 属性 ID）**

在 MIB 属性 ID 定义文件中搜索：
```
Grep: pattern="PARAM_ID_.*关键字" path="oaim/oam-h/parameter/MibAttributeId.h"
```
该文件使用 `ENUM_ELEMENT(PARAM_ID_xxx)` 宏定义所有参数 ID（约 3000+ 个）。

**步骤 B — 查找 trpath（TR-069 路径）**

在 TR-069 数据模型映射文件中，用 PARAM_ID 反查对应的 TR-069 路径：
```
Grep: pattern="PARAM_ID_XXX" path="oaim/oam-h/datamodel/"
```

数据模型文件列表（按优先级搜索）：
1. `oaim/oam-h/datamodel/RadisysExtDataModelRadisys.h` — Radisys 扩展参数（390K行，优先搜索）
2. `oaim/oam-h/datamodel/Tr196DataModel2Radisys.h` — TR-196 参数��122K行）
3. `oaim/oam-h/datamodel/Tr181DataModelRadisys.h` — TR-181 参数（42K行）
4. `oaim/oam-h/datamodel/Tr262DataModelRadisys.h` — TR-262 参数（7K行）
5. `oaim/oam-h/datamodel/Tr157DataModelRadisys.h` — TR-157 参数（12K行）

每条记录的格式为：
```c
{"Device.Services.FAPService.1.CellConfig.1.LTE.RAN.xxx",  // ← 这就是 trpath
 READ_WRITE,       // 访问权限
 STRING,           // 数据类型
 "0", "256",       // 值范围
 0,
 NO_NOTIFICATION,
 false,
 "",
 GROUP_ID_INVALID,
 "FAP.0",          // MIB DN
 PARAM_ID_XXX,     // ← 对应的 PARAM_ID
 Immediate,
 false},
```

**从搜索结果中提取**：第一个字段即为完整的 trpath，同时可提取访问权限（READ_ONLY/READ_WRITE）和数据类型。

**步骤 C — 查找 MIB 属性描述（类型、读写性）**

```
Grep: pattern="PARAM_ID_XXX" path="oaim/oam-h/parameter/MibAttributes.h"
```

格式：`{PARAM_ID_XXX, MIB_ATTRIBUTE_TYPE_U32, "参数名", true, false, 1}`
可提取数据类型和读写标志。

**步骤 D — 生成 CLI 命令（仅 002软件-BaiBN 项目需要）**

根据项目决定是否生成 CLI 命令：
- **002软件-BaiBN** → 生成 mibcli 命令
- **003软件-BaiBM** → 跳过此步骤

MIB 参数路径格式：`{对象类}.{实例ID}.{属性名}`
- 对象类 = `MIB_OBJECT_CLASS_XXX` 去掉前缀 → `XXX`
- 属性名 = `PARAM_ID_XXX` 去掉前缀 → `XXX`
- 示例：`FAP.0.MTU_SIZE`、`FAP.0.FAP_LTE.0.CARRIER.0.DL_BANDWIDTH`

从步骤 B 的数据模型记录中提取 MIB DN 字段（如 `"FAP.0"`）和 PARAM_ID（去掉 `PARAM_ID_` 前缀），拼接成 MIB 路径。

**mibcli 命令格式**（002软件-BaiBN 项目使用）：
```bash
# GET 命令（读取参数值）
mibcli get ${MIB_DN}:${PARAM_NAME}

# SET 命令（设置参数值）
mibcli set ${MIB_DN}:${PARAM_NAME}:=${VALUE}
```

示例：
```bash
mibcli get FAP.0.TCP_AGENT_PARAMS.0:TCP_AGENT_ENABLE
mibcli set FAP.0.TCP_AGENT_PARAMS.0:TCP_AGENT_ENABLE:=1
mibcli get FAP.0.FAP_LTE.0.CARRIER.0:DL_BANDWIDTH
```

**步骤 E — 从 git diff 自动提取（如果已有代码修改）**

如果用户已经修改了代码，从 diff 中提取新增/修改的 PARAM_ID：
```bash
git diff --unified=0 | grep -E "^\+.*PARAM_ID_"
```
然后对每个提取到的 PARAM_ID 执行步骤 B~D。

##### 完整自动查找示例：

用户说"新增一个 MTU 配置参数"（002软件-BaiBN 项目），执行流程：

1. 搜索 `PARAM_ID_.*MTU` → 找到 `PARAM_ID_MTU_SIZE`
2. 搜索 `PARAM_ID_MTU_SIZE` in datamodel → 找到 trpath `"Device.IP.Interface.1.MaxMTUSize"`，MIB DN `"FAP.0"`
3. 搜索 `PARAM_ID_MTU_SIZE` in MibAttributes → 找到类型 `MIB_ATTRIBUTE_TYPE_U32`
4. 拼接 MIB 路径：`FAP.0` + `MTU_SIZE`（PARAM_ID 去前缀）→ `FAP.0.MTU_SIZE`
5. 生成命令（仅 002软件-BaiBN）：
   ```
   GET: mibcli get FAP.0:MTU_SIZE
   SET: mibcli set FAP.0:MTU_SIZE:=1500
   ```
6. 组装 notes（002软件-BaiBN 项目）：
   ```
   【涉及MIB参数】
   参数名: MTU_SIZE
   PARAM_ID: PARAM_ID_MTU_SIZE
   MIB路径: FAP.0.MTU_SIZE
   trpath: Device.IP.Interface.1.MaxMTUSize
   类型: U32 | 权限: READ_WRITE
   GET: mibcli get FAP.0:MTU_SIZE
   SET: mibcli set FAP.0:MTU_SIZE:=1500
   取值范围: 0 ~ 65535
   ```

   组装 notes（003软件-BaiBM 项目，不带 CLI 命令）：
   ```
   【涉及MIB参数】
   参数名: MTU_SIZE
   PARAM_ID: PARAM_ID_MTU_SIZE
   MIB路径: FAP.0.MTU_SIZE
   trpath: Device.IP.Interface.1.MaxMTUSize
   类型: U32 | 权限: READ_WRITE | 取值: 0 ~ 65535 | 默认: 0
   ```

##### 通用说明：

使用 AskUserQuestion 询问用户字段值。对于问题单的修正结论和问题类型提供选项。

**问题根因** 和 **修改方案**（仅问题单）：
- 如果之前用 `/analyze-issue` 分���过，可以根据分析结果**预填建议值**给用户确认
- 如果是 `/auto-fix-issues` 流程中已经修改了代码，自动从 commit 信息中提取

**SVN/GIT修订号**（问题单和需求单都需要）：
- 如果本地有新的 commit，自动获取最新 commit hash：`git log -1 --format=%H`
- 否则让用户输入

**trpath/MIB 映射关系备注**（重要）：
- 如果问题修复涉及 **OAM 平台 trpath 配置变更**（而非代码修改），**必须在 notes 中添加完整的 MIB 参数与 trpath 对应关系**
- 详细格式见下方「涉及 trpath/MIB 变更时的备注规则」章节
- 判断依据：修改方案中包含"trpath"、"OAM平台"、"MIB映射"、"tr069"等关键词

##### 问题单更新数据：
```json
{
  "issue": {
    "status_id": 14,
    "custom_fields": [
      {"id": 1, "value": "FIXED"},
      {"id": 73, "value": "问题根因描述"},
      {"id": 19, "value": "修改方案描述"},
      {"id": 3, "value": "commit_hash"},
      {"id": 253, "value": "新增需求bug"},
      {"id": 202, "value": "674"},
      {"id": 78, "value": "1096"},
      {"id": 21, "value": "OK"},
      {"id": 13, "value": ["下一版本ID"]}
    ]
  }
}
```

##### 需求单更新数据：
```json
{
  "issue": {
    "status_id": 14,
    "notes": "【实现方案】\n实现方案描述...\n\n【涉及MIB参数】\nMIB Object: XXX (parent: FAP)\ntrpath Object: Device.xxx.yyy.\nmibdn: FAP.0.XXX.0\n涉及分支: 5G/xxx\n\n1. MIB: FAP.0.XXX.0.PARAM_NAME\n   PARAM_ID: PARAM_ID_XXX\n   trpath: Device.xxx.yyy.ParamName\n   类型: U32 | 权限: READ_WRITE | 取值: 0~1 | 默认: 0\n   GET: mibcli get FAP.0.XXX.0:PARAM_NAME    (仅002软件-BaiBN)\n   SET: mibcli set FAP.0.XXX.0:PARAM_NAME:=0  (仅002软件-BaiBN)",
    "custom_fields": [
      {"id": 3, "value": "commit_hash"},
      {"id": 78, "value": "1096"},
      {"id": 21, "value": "OK"}
    ]
  }
}
```

如果不涉及 MIB 参数，notes 中省略「涉及MIB参数」部分，只保留「实现方案」。

---

#### → CODE-ACCEPTED (status_id=15)

需要填写：
- **Review意见/comments** (custom_field id:20) — 默认 `OK`

更新数据：
```json
{
  "issue": {
    "status_id": 15,
    "custom_fields": [
      {"id": 20, "value": "OK"}
    ]
  }
}
```

---

#### → CODE-COMMITTED (status_id=12)

通常无额外必填字段，直接推进状态。

更新数据：
```json
{
  "issue": {
    "status_id": 12
  }
}
```

---

#### → RESOLVED (status_id=3)

**根据 tracker 类型填写不同字段。**

##### 软件问题单 (tracker_id=1)：
- **自测结果/Self-test Results** (custom_field id:21) — 默认 `OK`
- **测试建议/Testing Suggestions** (custom_field id:79) — 文字描述
- **漏测标识/Missed Test Indicator** (custom_field id:262) — 选项: Y / N
- **非漏测缘由/Non-missed causes** (custom_field id:263) — 如漏测标识为 N 则填写原因

```json
{
  "issue": {
    "status_id": 3,
    "custom_fields": [
      {"id": 21, "value": "OK"},
      {"id": 79, "value": "--"},
      {"id": 262, "value": "N"},
      {"id": 263, "value": "NULL"}
    ]
  }
}
```

##### 需求单 (tracker_id=2)：
- **自测结果/Self-test Results** (custom_field id:21) — 默认 `OK`

需求单**不需要**填写测试建议、漏测标识等字段。

```json
{
  "issue": {
    "status_id": 3,
    "custom_fields": [
      {"id": 21, "value": "OK"}
    ]
  }
}
```

---

#### → REOPENED (status_id=13)

需要添加备注说明 reopen 原因。

更新数据：
```json
{
  "issue": {
    "status_id": 13,
    "notes": "reopen 原因说明"
  }
}
```

---

### 第 4 步：确认并提交

在提交前，展示完整的更新内容让用户确认：

```
即将更新问题 #${ISSUE_ID}:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  状态: ${OLD_STATUS} → ${NEW_STATUS}
  修正结论: FIXED
  问题根因: xxx
  修改方案: xxx
  GIT修订号: abc123...
  问题类型: 新增需求bug
  Review工��师: zhangsan
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
确认提交？
```

**必须使用 AskUserQuestion 获得用户确认后才能提交。**

用户确认后，执行更新：
```bash
curl -s -X PUT \
  -H "X-Redmine-API-Key: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '${JSON_PAYLOAD}' \
  "${BASE_URL}/issues/${ISSUE_ID}.json"
```

### 第 5 步：验证更新结果

提交后重新获取问题状态，确认更新成功：

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/issues/${ISSUE_ID}.json"
```

输出结果：
```
问题 #${ISSUE_ID} 更新成功
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  状态: ${OLD_STATUS} → ${NEW_STATUS} [OK]
  链接: ${BASE_URL}/issues/${ISSUE_ID}

下一步操作:
  /update-issue ${ISSUE_ID}    — 继续推进到下一状态
  /view-issue ${ISSUE_ID}      — 查看完整详情
```

---

## 批量快速推进

如果用户想一次性从 OPEN 推进到 RESOLVED（比如代码已经改好，只需要在 Redmine 上走完流程），
可以在 $ARGUMENTS 中直接指定最终目标状态，系统会逐步提交每个中间状态：

```
/update-issue 109561 resolved
```

这会依次执行：OPEN → ASSIGNED → IN-PROGRESS → CODE-REVIEW → CODE-ACCEPTED → CODE-COMMITTED → RESOLVED，
每一步自动填写默认值，仅在需要用户输入时暂停：

- **软件问题单**：在 CODE-REVIEW 阶段暂停（需填写修正结论/根因/方案/commit/问题类型 5 个字段，Review工程师和问题引入人使用默认值）
- **需求单**：在 CODE-REVIEW 阶段暂停（需填写 commit 号和实现方案，并补充自测结果 `OK`；Review工程师使用默认值）

---

## 解决版本规则

**解决版本 (cf 13) 必须填问题版本的下一个版本，而非当前问题版本。**

查询方法：从问题的 `问题版本 (cf 11)` 获取版本名（如 `BaiBNQ_3.9.6`），然后查询该项目的版本列表，找到下一个版本号：

```bash
# 1. 获取问题版本 ID（如 6037）
ISSUE_VERSION_ID=$(从 issue 的 custom_fields 中 id=11 的 value 获取)

# 2. 查询版本名称
curl -s -H "X-Redmine-API-Key: ${TOKEN}" "${BASE_URL}/versions/${ISSUE_VERSION_ID}.json"
# → 返回 name: "BaiBNQ_3.9.6", project: "002软件-BaiBN"

# 3. 查询项目下所有版本，找到下一个
curl -s -H "X-Redmine-API-Key: ${TOKEN}" "${BASE_URL}/projects/${PROJECT_ID}/versions.json?limit=100"
# 按版本号排序，找到比��前版本号大的下一个版本
```

示例：问题版本 `BaiBNQ_3.9.6` (id:6037) → 解决版本应填 `BaiBNQ_3.9.7` (id:6112)

**注意**：解决版本字段值为**数组格式** `["6112"]`，不是字符串。

---

## 涉及 trpath/MIB 变更时的备注规则

当问题修复涉及 OAM 平台 trpath ��置变更（而非代码修改）时，**必须在问题单添加备注（notes）列出完整的 MIB 参数与 trpath 对应关系**，格式如下：

```
【trpath 与 MIB 参数对应关系】

MIB Object: <MIB 对象名>
trpath Object: <完整 trpath OBJECT 路径>
mibdn: <完整 mibdn 路径>
涉及分支: <分支列��>

1. MIB: <mibdn>.<MIB参数名>
   trpath: <完整 trpath 路径>
   类型: <数据类型> | 权限: <读写权限> | 取值: <min>~<max> | 默认: <默认值>

2. MIB: ...
   trpath: ...
   ...
```

**要点**：
- MIB 路径和 trpath 路径都必须写**完整全路径**（不能缩写）
- 每个参数都要列出类型、权限、取值范围、默认值
- 如果涉及多个分支（如 release 和 master），都要注明
- 此备注通过单独的 PUT 请求以 `notes` 字段添加（不影响状态字段的更新）

**项目区分**：
- **002软件-BaiBN** 项目：额外列出 mibcli GET/SET 命令
  ```
  1. MIB: <mibdn>.<MIB参数名>
     PARAM_ID: PARAM_ID_<参数名>
     trpath: <完整 trpath 路径>
     类型: <数据类型> | 权限: <读写权限> | 取值: <min>~<max> | 默认: <默认值>
     GET: mibcli get <mibdn>:<MIB参数名>
     SET: mibcli set <mibdn>:<MIB参数名>:=<值>
  ```
- **003软件-BaiBM** 项目：只备注参数层级，不写 CLI 命令
  ```
  1. MIB: <mibdn>.<MIB参数名>
     PARAM_ID: PARAM_ID_<参数名>
     trpath: <完整 trpath 路径>
     类型: <数据类型> | 权限: <读写权限> | 取值: <min>~<max> | 默认: <默认值>
  ```

---

## 安全注意事项

- **所有状态更新操作都必须经过用户确认**
- 提交前展示完整的更新内容
- 更新后验证结果
- 如果 API 返回错误（如必填字段缺失），展示错误信息并让用户补充

## 默认人员配置

以下人员信息为默认值，直接使用无需每次询问：

| 角色 | 姓名 | 登录名 | 用户 ID |
|------|------|--------|---------|
| 开发工程师 | 王勇 | wangyong1246 | 977 |
| Review工程师 | 张璐 | zhanglu1378 | 1096 |
| 问题引入人 | 刘荣 | liurong | 674 |

**使用规则**：
- **开发工程师** (cf_23): 固定填当前登录用户 `977` (wangyong1246)
- **Review工程师** (cf_78): 默认填 `1096` (zhanglu1378)，用户指定其他人时再通过 API 查询
- **问题引入人** (cf_202): 默认填 `674` (liurong)，用户指定其他人时再通过 API 查询

如需查询其他用户 ID，可通过 API 查询（支持模糊匹配，注意分页 limit=100，total=201）：
```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/projects/baibm/memberships.json?limit=100&offset=0" \
  | python3 -c "import json,sys; [print(f'{m[\"user\"][\"id\"]:>5} {m[\"user\"][\"name\"]}') for m in json.load(sys.stdin)['memberships'] if 'user' in m]"
```
