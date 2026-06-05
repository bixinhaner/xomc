# Auto Fix Issues - BaiBM 自动问题定位与修复

连接 Redmine (http://192.168.5.41:8080/redmine) 的 BaiBM 项目，自动获取问题单/需求单，
下载附件日志，分析定位问题，并在本地代码库中修改。

> **这是完整的自动修复流程（拉取 → 分析 → 下载日志 → 定位 → 改代码）。**
> 如果只需要执行部分操作，请使用以下分步命令：
>
> | 命令 | 功能 | 是否修改代码 |
> |------|------|:----------:|
> | `/list-bugs` | 查看软件问题单列表 | 否 |
> | `/list-reqs` | 查看需求单列表 | 否 |
> | `/list-all` | 查看所有待处理项（含统计） | 否 |
> | `/view-issue <ID>` | 查看单个问题完整详情 | 否 |
> | `/download-logs <ID>` | 下载并解压问题的日志附件 | 否 |
> | `/analyze-issue <ID>` | 分析日志并定位代码（不改代码） | 否 |
> | `/update-issue <ID>` | 推进问题状态 + 填写必填字段 | **Redmine更新（需确认）** |
> | `/auto-fix-issues` | **完整流程：分析 + 修复代码** | **是（需确认）** |

## 使用方法

```
/auto-fix-issues [过滤条件]
```

$ARGUMENTS 说明（可选，默认拉取指派给自己的开放问题）：
- `all`          — 所有开放问题（不限指派人）
- `my`           — 仅指派给自己的（默认）
- `bug`          — 仅软件问题单（tracker_id=1）
- `req`          — 仅需求单（tracker_id=2）
- `high`         — 仅 P1-High 优先级
- `id:109940`    — 直接处理指定 ID 的问题
- 可组合：`my bug high`

示例：
```
/auto-fix-issues
/auto-fix-issues my bug
/auto-fix-issues id:109561
/auto-fix-issues all high
```

---

## 执行流程

### 第 0 步：读取配置并校验连接

读取配置文件 `.claude/commands/issue/issue-tracker.json`：

```json
{
  "platform": "redmine",
  "base_url": "http://192.168.5.41:8080/redmine",
  "token": "<API Key>",
  "username": "wangyong1246",
  "default_project": "baibm",
  "log_download_dir": "/tmp/issue-logs",
  "max_issues": 20,
  "auto_commit": false
}
```

如果配置文件不存在，**立即停止**并提示用户创建。

使用以下命令验证连接：
```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" "${BASE_URL}/users/current.json"
```
确认返回用户信息后继续。

### 第 1 步：获取问题列表

解析 `$ARGUMENTS`，构建 API 请求 URL：

**基础 URL：**
```
${BASE_URL}/projects/baibm/issues.json?status_id=open&sort=priority:desc,updated_on:desc&limit=${MAX_ISSUES}
```

**根据过滤条件追加参数：**
- `my` / 默认 → `&assigned_to_id=me`
- `all` → 不加 assigned_to_id
- `bug` → `&tracker_id=1`（软件问题单）
- `req` → `&tracker_id=2`（需求单）
- `high` → `&priority_id=3`（P1-High）
- `id:XXXXX` → 直接请求 `${BASE_URL}/issues/XXXXX.json?include=attachments,journals`，跳到第 2 步

**执行请求：**
```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" "${API_URL}" | python3 -m json.tool
```

将返回结果整理为表格展示给用户：

```
 #   ID       类型          优先级        分类       标题
─────────────────────────────────────────────────────────────
 1   109940   需求单        P2-Middle    OAM       2+4 同步源同步失步后处理方案更新
 2   109861   软件问题单    P2-Middle    OAM       【BM_1.0.36】【OXM】11184告警优化
 ...
```

**使用 AskUserQuestion 询问用户**：要处理哪些问题？选项：
- 全部处理
- 选择特定编号（如 1,3,5）
- 仅处理第一个

### 第 2 步：逐个处理问题单

对每个选中的问题，**按顺序**执行以下子步骤。用 TodoWrite 追踪每个问题的处理进度。

#### 2.1 获取问题详情

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/issues/${ISSUE_ID}.json?include=attachments,journals" \
  | python3 -m json.tool
```

提取关键字段：
- `issue.subject` — 标题
- `issue.description` — 完整描述
- `issue.journals` — 评论/历史记录
- `issue.attachments` — 附件列表（含 content_url）
- `issue.category.name` — 所属模块（如 OAM, L3, MAC, BSP, LMT 等）
- `issue.custom_fields` 中的关键字段：
  - `严重性/Severity` (id:6)
  - `可复现性/Reproducibility` (id:16)
  - `问题版本/Issue Version` (id:11)
  - `影响分析/Impact Analysis` (id:124)
  - `缺陷类型/Defect Type` (id:71)

#### 2.2 分析问题描述

仔细阅读问题描述（description）和评论（journals），提取：

1. **问题现象**：从「现象描述」段落提取
2. **测试组网**：从「测试组网」段落提取网络拓扑
3. **参数配置**：从「参数配置」段落提取相关配置
4. **复现步骤**：从「测试步骤」段落提取
5. **预期 vs 实际**：从「预期结果」和「实际结果」段落对比
6. **问题分析**：从「问题分析」段落提取（如果有）
7. **所属模块**：根据 category（OAM/L3/MAC/BSP/LMT/FPGA/BSC/PDCP）定位代码目录

输出格式：
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
问题 #${ISSUE_ID}: ${SUBJECT}
模块: ${CATEGORY} | 严重性: ${SEVERITY} | 版本: ${VERSION}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[现象] ...
[复现] ...
[预期] ...
[实际] ...
[分析] ...
```

#### 2.3 下载附件日志

从 `issue.attachments` 中筛选日志相关文件：
- 匹配扩展名：`.log`, `.txt`, `.zip`, `.tar.gz`, `.gz`, `.rar`, `.7z`, `.pcap`, `.cap`
- 跳过图片文件：`.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`

下载命令：
```bash
mkdir -p "/tmp/issue-logs/${ISSUE_ID}"

# Redmine 附件下载（需要 API Key 认证）
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  -o "/tmp/issue-logs/${ISSUE_ID}/${FILENAME}" \
  "${ATTACHMENT_CONTENT_URL}"
```

**自动解压：**
```bash
cd "/tmp/issue-logs/${ISSUE_ID}"

# ZIP 文件
if [[ "${FILENAME}" == *.zip ]]; then
  unzip -o "${FILENAME}" 2>/dev/null

# tar.gz / tgz 文件
elif [[ "${FILENAME}" == *.tar.gz ]] || [[ "${FILENAME}" == *.tgz ]]; then
  tar -xzf "${FILENAME}"

# gz 单文件
elif [[ "${FILENAME}" == *.gz ]] && [[ "${FILENAME}" != *.tar.gz ]]; then
  gunzip -k "${FILENAME}"

# rar 文件
elif [[ "${FILENAME}" == *.rar ]]; then
  unrar x "${FILENAME}" 2>/dev/null || echo "需要安装 unrar"
fi
```

解压后列出所有文件：
```bash
find "/tmp/issue-logs/${ISSUE_ID}" -type f | head -50
```

#### 2.4 分析日志定位问题

**步骤 A — 搜索错误关键字：**

在所有日志文件中搜索以下关键字（使用 Grep 工具）：
- `ERROR`, `FATAL`, `FAIL`, `CRASH`, `PANIC`, `ASSERT`, `Segfault`, `core dump`
- `Exception`, `Traceback`, `abort`, `signal 11`, `signal 6`
- 问题描述中提到的特定关键字（如告警编号、模块名、错误码）

**步骤 B — 提取关键上下文：**

对每个匹配到的错误，读取前后 20 行上下文，关注：
- 时间戳 — 确定问题发生的时间窗口
- 调用链 / 堆栈 — 追踪问题根因
- 函数名、文件名 — 映射到代码位置
- 线程/进程信息 — 确定问题发生的执行上下文

**步骤 C — 关联到本地代码：**

从日志中提取的文件名、函数名、模块名，使用 Grep 和 Glob 在本地代码库中查找：
```
# 示例：根据日志中的函数名查找
Grep: pattern="函数名" path="."
Glob: pattern="**/*模块名*"
```

输出结构化分析结果：
```
问题 #${ISSUE_ID}: ${SUBJECT}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[现象] ...
[根因] 根据日志分析得出的根本原因
[关键日志]
  ${TIMESTAMP} ${LOG_FILE}:${LINE} - 关键错误信息
  ${TIMESTAMP} ${LOG_FILE}:${LINE} - 相关上下文
[定位文件] path/to/source/file.c:123 — 函数名()
[修复方向] 简述修复思路
```

#### 2.5 定位并修改代码

1. 根据 2.4 的分析，使用 Grep 和 Glob 找到需要修改的源代码文件
2. 使用 Read 阅读相关代码，理解完整上下文（至少读取前后 50 行）
3. **使用 AskUserQuestion 向用户确认修复方案**：
   - 展示问题根因
   - 展示修复方案（伪代码或具体 diff）
   - 列出将修改的文件和行号
   - 评估修改影响范围
4. 用户确认后，使用 Edit 工具修改代码
5. 修改完成后，如果 `auto_commit` 为 true，创建 commit：
   ```
   fix(#${ISSUE_ID}): 简短描述

   问题现象: ...
   根本原因: ...
   修改方案: ...

   Redmine: ${BASE_URL}/issues/${ISSUE_ID}
   ```

#### 2.6 更新 Redmine 问题单（可选）

修复完成后，询问用户是否要更新 Redmine 上的问题单状态和字段：

```bash
# 更新问题状态为 IN-PROGRESS (status_id=2)
curl -s -X PUT -H "X-Redmine-API-Key: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"issue":{"status_id":2,"notes":"已完成代码修改，详见 commit xxx"}}' \
  "${BASE_URL}/issues/${ISSUE_ID}.json"

# 也可以填写自定义字段：
# - 修正结论 (id:1)
# - 问题根因 (id:73)
# - 修改方案 (id:19)
# - SVN/GIT修订号 (id:3)
# - 问题引入人 (id:202) — 默认 674 (liurong)
# - Review工程师 (id:78) — 默认 1096 (zhanglu1378)
# - 开发工程师 (id:23) — 固定 977 (wangyong1246)
```

**重要**：更新 Redmine 操作必须获得用户明确确认。

### 第 3 步：生成汇总报告

所有问题处理完毕后，生成汇总报告：

```
╔════════════════════════════════════════════════════╗
║           BaiBM 问题处理汇总报告                    ║
╠════════════════════════════════════════════════════╣
║ 项目: BaiBM (003软件-BaiBM)                        ║
║ 处理人: 王勇 (wangyong1246)                        ║
║ 时间: YYYY-MM-DD HH:MM                            ║
║ 总计处理: X 个问题                                  ║
║ 成功修复: Y 个                                      ║
║ 需人工介入: Z 个                                    ║
╚════════════════════════════════════════════════════╝

详细列表:
 [OK]  #109561 - TR069 CRASH → 修改 oam/tr069/xxx.c:123
 [OK]  #109612 - 导入160邻区重启 → 修改 oam/lmt/neighbor.c:456
 [!!]  #109940 - 同步源方案更新 → 需求单，需设计评审

日志文件位置: /tmp/issue-logs/
```

---

## 安全注意事项

- **绝对不要**将 `.claude/commands/issue/issue-tracker.json` 提交到 Git（已加入 .gitignore）
- Token 仅用于 API 调用，不会被发送到其他地址
- 下载的日志文件位于 `/tmp/issue-logs/`，处理完后建议清理
- **所有代码修改必须经用户确认**后才执行
- **所有 Redmine 状态更新**必须经用户确认后才执行

## 错误处理

- API 返回 401：Token 失效，提示用户更新 `.claude/commands/issue/issue-tracker.json` 中的 token
- API 返回 403：无权限访问该项目，提示检查项目权限
- API 返回 404：项目或问题 ID 不存在，提示检查输入
- 附件下载失败：跳过并记录，继续处理下一个附件
- 日志文件过大（>50MB）：用 Bash `head`/`tail` 取首尾各 5000 行，再用 Grep 搜索关键错误
- 压缩包解压失败：记录错误，尝试其他解压方式
- 无法定位到代码：标记为「需人工介入」，输出分析结果供参考
- 代码修改不确定时：**不强行修改**，标记为「需人工介入」

## Redmine 平台信息备忘

| 类型 | tracker_id | 名称 |
|------|-----------|------|
| 软件问题单 | 1 | 代码缺陷 |
| 需求单 | 2 | 新功能/优化需求 |
| 硬件问题单 | 12 | 硬件相关 |

| 状态 | status_id | 名称 |
|------|----------|------|
| OPEN | 1 | 新建 |
| ASSIGNED | 7 | 已指派 |
| IN-PROGRESS | 2 | 处理中 |
| CODE-REVIEW | 14 | 代码评审 |
| CODE-ACCEPTED | 15 | 评审通过 |
| CODE-COMMITTED | 12 | 已提交 |
| RESOLVED | 3 | 已解决 |
| TESTING | 32 | 测试中 |
| VERIFIED | 4 | 已验证(关闭) |
| CLOSED | 5 | 已关闭 |
| REOPENED | 13 | 重新打开 |

| 模块分类 | 说明 |
|---------|------|
| OAM | 操作管理维护 |
| L3 | 层3协议 |
| MAC | MAC层 |
| BSP | 板级支持包 |
| LMT | 本地维护终端 |
| BSC | 基站控制器 |
| PDCP | 数据汇聚协议 |
| FPGA | 硬件逻辑 |
