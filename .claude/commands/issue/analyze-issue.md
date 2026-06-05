# 分析问题

对指定问题单进行完整分析：读取描述 → 下载日志 → 分析日志 → 定位代码位置。
**只分析不修改代码**，适合先了解问题再决定是否修改。

## 使用方法
```
/analyze-issue <问题ID>
```

$ARGUMENTS 为问题 ID（必填），例如：
```
/analyze-issue 109561
/analyze-issue 109940
```

---

## 执行步骤

### 第 1 步：获取问题详情

读取 `.claude/commands/issue/issue-tracker.json` 获取配置。

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/issues/${ISSUE_ID}.json?include=attachments,journals" \
  | python3 -m json.tool
```

输出问题摘要：
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
分析问题 #${ISSUE_ID}: ${SUBJECT}
模块: ${CATEGORY} | 严重性: ${SEVERITY} | 版本: ${VERSION}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 第 2 步：解读问题描述

仔细阅读 description 和 journals，按结构提取：

1. **问题现象**：从「现象描述」中提取核心异常表现
2. **测试组网**：从「测试组网」中提取网络拓扑和硬件配置
3. **参数配置**：提取涉及的关键配置参数
4. **复现步骤**：提取明确的操作序列
5. **预期 vs 实际**：对比预期和��际行为的差异
6. **已有分析**：提取已有的「问题分析」（如果有）
7. **关键线索**：提取评论中其他人的分析意见

输出：
```
[现象] 简述问题表现
[组网] 拓扑描述
[配置] 关键参数
[复现] 1. 步骤一  2. 步骤二  ...
[预期] 期望的正确行为
[实际] 实际的错误行为
[线索] 已知的分析方向或他人意见
```

### 第 3 步：下载并解压日志

从 `issue.attachments` 中筛选日志相关文件（`.log`, `.txt`, `.zip`, `.tar.gz`, `.gz`, `.pcap` 等），下载到 `/tmp/issue-logs/${ISSUE_ID}/`。

压缩包自动解压。具体逻辑同 `/download-logs` 命令。

如果该问题没有附件或没有日志类附件，则跳过此步，仅基于问题描述进行分析。

### 第 4 步：分析日志

**A. 搜索错误关键字**

在下载的所有日志文件中，使用 Grep 搜索：
- 通用错误：`ERROR`, `FATAL`, `FAIL`, `CRASH`, `PANIC`, `ASSERT`, `abort`
- C/C++ 相关：`Segfault`, `signal 11`, `signal 6`, `core dump`, `stack trace`, `backtrace`
- 嵌入式相关：`watchdog`, `reboot`, `hardfault`, `assert failed`
- 问题描述中提到的**特定关键字**（如告警编号、模块名、错误码、小区 ID 等）

**B. 提取上下文**

对每个错误匹配，读取**前后 30 行**上下文，重点关注：
- 时间戳 — 问题发生的时间点和持续时间
- 调用链/堆栈 — 追踪 crash 或异常的调用路径
- 函数名/文件名 — 代码中的确切位置
- 线程/进程/模块 — 问题发生的执行上下文
- 参数值 — 触发问题的关键参数

**C. 时间线还原**

如果日志有时间戳，按时间排序关键事件，构建问题发生的时间线：
```
[HH:MM:SS.mmm] 事件1 — 正常操作
[HH:MM:SS.mmm] 事件2 — 触发条件
[HH:MM:SS.mmm] 事件3 — 异常发生 ← 问题点
[HH:MM:SS.mmm] 事件4 — 后续影响
```

### 第 5 步：关联本地代码

根据日志中提取的文件名、函数名、模块名，在本地代码库中查找：

1. 使用 Grep 搜索函数名、错误码
2. 使用 Glob 按模块目录结构查找相关文件
3. 使用 Read 读取定位到的代码，理解逻辑

根据模块���类（category）确定搜索范围：
- OAM → 搜索 `oaim/` 目录
- L3 → 搜索 l3/rrc 相关目录
- MAC → 搜索 mac 相关目录
- BSP → 搜索 bsp/driver 相关目录
- LMT → 搜索 lmt/web 相关目录

### 第 5.1 步：OAM 模块 — 自动查找涉及的 MIB 参数和 trpath

如果问题涉及 OAM 模块（参���配置、告警、TR-069 等），自动执行以下查找：

**A. 从日志/描述中提取关键字，查找 PARAM_ID：**
```
Grep: pattern="PARAM_ID_.*关键字" path="oaim/oam-h/parameter/MibAttributeId.h"
```

**B. 用 PARAM_ID 反�� TR-069 路径 (trpath)：**
```
Grep: pattern="PARAM_ID_XXX" path="oaim/oam-h/datamodel/"
```
数据模型文件（按搜索优先级）：
1. `oaim/oam-h/datamodel/RadisysExtDataModelRadisys.h` — 扩展参数（390K行）
2. `oaim/oam-h/datamodel/Tr196DataModel2Radisys.h` — TR-196（122K行）
3. `oaim/oam-h/datamodel/Tr181DataModelRadisys.h` — TR-181（42K行）
4. `oaim/oam-h/datamodel/Tr262DataModelRadisys.h` — TR-262（7K行）
5. `oaim/oam-h/datamodel/Tr157DataModelRadisys.h` — TR-157（12K行）

每条记录格式：`{"Device.xxx.yyy", READ_WRITE, STRING, ..., "FAP.0", PARAM_ID_XXX, ...}`
第一个字段即为 trpath。

**C. 查找 MIB 属性类型和读写性：**
```
Grep: pattern="PARAM_ID_XXX" path="oaim/oam-h/parameter/MibAttributes.h"
```
格式：`{PARAM_ID_XXX, MIB_ATTRIBUTE_TYPE_U32, "名称", true, false, 1}`

**D. 生成 CLI 命令（使用 MIB 参数路径，非 trpath）：**

MIB 路径 = 数据模型中的 MIB DN（如 `FAP.0`）+ PARAM_ID 去前缀（如 `MTU_SIZE`）

```bash
oam_cli_tool get FAP.0.XXX          # GET
oam_cli_tool set FAP.0.XXX value    # SET
oam_cli_tool memget FAP.0.XXX       # 共享内存直读
```

将查找到的 MIB 参数信息加入分析报告的 `[涉及MIB参数]` 部分。

### 第 6 步：输出分析报告

```
╔══════════════════════════════════════════════════════════╗
║  问题分析报告 #${ISSUE_ID}
╠══════════════════════════════════════════════════════════╣

[问题概述]
  ${一句话概括}

[现象描述]
  ${详细的问题表现}

[根因分析]
  ${根据日志和代码分析得出的根本原因}
  置信度: 高/中/低

[关键日志证据]
  ${TIMESTAMP} ${FILE}:${LINE}
  > 关键错误日志行
  > 相关上下文日志

[代码定位]
  文件: ${FILE_PATH}:${LINE_NUMBER}
  函数: ${FUNCTION_NAME}()
  问题代码段:
  │ ${代码片段，标注问题行}

[涉及MIB参数]（如涉及OAM参数配置）
  参数名: XXX
  PARAM_ID: PARAM_ID_XXX
  MIB路径: FAP.0.XXX
  trpath: Device.Services.FAPService.1.xxx
  类型: U32 | 权限: READ_WRITE
  GET: oam_cli_tool get FAP.0.XXX
  SET: oam_cli_tool set FAP.0.XXX 值

[修复建议]
  方向: ${简述修复思路}
  影响: ${修改可能影响的其他功能}
  风险: 高/中/低

╚══════════════════════════════════════════════════════════╝

后续操作:
  /auto-fix-issues id:${ISSUE_ID} — 自动修复此问题
  手动修改后可更新 Redmine 问题状态
```

**重要**：
- 这是**只读分析**命令，**不修改任何代码**
- 如果无法确定根因，明确标注「置信度: 低」并说明还需要哪些信息
- 如果没有日志附件，基于问题描述和代码结构给出推测性分析，并明确标注为推测
- 分析结论要有日志证据支撑，避免无根据的猜测
