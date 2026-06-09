# 智能提交 (Smart Commit)

检测变更、执行代码审查、生成审查报告，然后创建规范化的 Conventional Commits 提交。审查报告随代码一起提交。

## 参数

- `$ARGUMENTS` — 可选。可指定提交类型: `feat` | `fix` | `refactor` | `docs` | `test` | `chore` | `perf` | `build` | `ci` | `style`。留空则自动推断。

---

## 执行步骤

请按以下顺序严格执行：

### Step 1: 检测变更

1. 运行 `git status` 查看工作区状态
2. 运行 `git diff --staged --name-only` 获取已暂存文件列表
3. 运行 `git diff --staged --stat` 查看暂存变更统计

**判断逻辑**：
- 如果有暂存文件 → 继续
- 如果无暂存文件但有未暂存变更 → **提示用户先执行 `git add`，列出变更文件供用户选择，等待用户确认后再继续**
- 如果无任何变更 → **停止，提示"无变更可提交"**

### Step 2: 确定提交仓库

根据暂存文件路径判断目标仓库：

| 文件路径前缀 | 目标仓库 | git 操作目录 |
|-------------|---------|-------------|
| `omcgo/` | 后端仓库 | `cd omcgo` |
| `omcmb/` | 前端仓库 | `cd omcmb` |
| 根目录文件 | 根仓库 | 根目录 |

**注意**：如果暂存文件跨越多个仓库（如同时有 omcgo/ 和 omcmb/ 的文件），提示用户分开提交——每个子仓库应该独立提交。

### Step 3: 自动推断 type

如果 `$ARGUMENTS` 指定了 type，直接使用。否则按以下规则推断：

| 变更特征 | 推断 Type |
|----------|-----------|
| 新增功能文件（handler/service/repository/page/hook） | `feat` |
| 修改现有文件中的 bug 逻辑（条件判断/错误处理修复） | `fix` |
| 仅 `.md` 文件变更 | `docs` |
| 仅 `_test.go` / `*.test.ts` / `*.spec.ts` 文件 | `test` |
| 仅 `Makefile` / `Dockerfile` / `docker-compose*` / `.github/` | `build` / `ci` |
| 文件重命名/移动/结构调整，功能不变 | `refactor` |
| 仅格式化/空白/注释变更 | `style` |
| 明确的性能优化（索引/缓存/算法优化） | `perf` |
| 依赖更新/工具配置 | `chore` |

如果无法明确推断，**询问用户确认 type**。

### Step 4: 自动推断 scope

使用与 `/review` 相同的路径→scope 映射表：

| 路径前缀 | Scope |
|----------|-------|
| `omcgo/internal/acs/` | acs |
| `omcgo/internal/config/` | config |
| `omcgo/internal/pm/` | pm |
| `omcgo/internal/alarm/` | alarm |
| `omcgo/internal/mr/` | mr |
| `omcgo/internal/device/` | device |
| `omcgo/internal/admin/` | admin |
| `omcgo/internal/topology/` | topology |
| `omcgo/internal/software/` | software |
| `omcgo/internal/backup/` | backup |
| `omcgo/internal/dashboard/` | dashboard |
| `omcgo/internal/ops/` | ops |
| `omcgo/internal/report/` | report |
| `omcgo/internal/mml/` | mml |
| `omcgo/internal/filemanager/` | filemanager |
| `omcgo/internal/syslog/` | syslog |
| `omcgo/internal/license/` | license |
| `omcgo/internal/nedirect/` | nedirect |
| `omcgo/internal/northbound/` | northbound |
| `omcgo/internal/provision/` | provision |
| `omcgo/internal/interop/` | interop |
| `omcgo/internal/carrier/` | carrier |
| `omcgo/internal/components/` | components |
| `omcgo/internal/middleware/` | middleware |
| `omcgo/internal/event/` | event |
| `omcgo/cmd/` / Makefile / Dockerfile / deployments/ | deploy |
| `omcgo/migrations/` | migration |
| `omcmb/webcode/src/services/api/` | api |
| `omcmb/webcode/src/hooks/` | hooks |
| `omcmb/webcode/src/pages/<name>/` | 使用 `<name>` |
| `omcmb/webcode/src/components/` | components |
| `omcmb/webcode/src/store/` | store |
| `docs/project/` / `.claude/commands/` / 根目录流程制品（CLAUDE.md / settings.json） | process |
| 根目录 `*.md` / `docs/`（非流程类） | docs |

**组合规则**：
- 单一 scope → 直接使用
- 2-3 个 scope → 用主要 scope
- 超过 3 个 → 省略 scope 或使用最主要的那个

### Step 4.5: 识别关联 GitHub Issue（流程闭环）

**背景**：commit 应引用对应的 GitHub Issue（活任务源，见 `docs/agents/issue-tracker.md`），让 `/ship` 流程可追溯。

按以下顺序推断 `#NN`，首次命中即止：

1. `$ARGUMENTS` 含 `--issue=NN` → 采用
2. 当前分支名含 `#NN` / `issue-NN`（如 `feat/42-alarm-email`）→ 采用
3. `git log -10 --format=%B` 最近 commit 引用过同一 `#NN` 且本次变更延续其范围 → 采用
4. `gh issue list --label ready-for-agent --state open` 恰好一条 → 采用并提示用户确认
5. 以上都无 → 提示（**不阻塞**，纯流程/工具链提交可无 Issue）：
   ```
   未识别关联 Issue。可任选：
     a) 重试：/commit --issue=NN
     b) 尚无 Issue：/ship P2（to-prd）或 /to-issues 先登记
     c) 临时修复：在 body 说明原因，/ship P9 收尾时补 Issue
   ```

获取 `#NN` 后：`gh issue view NN` 确认其存在且未关闭，供 Step 7 footer 使用。

### Step 5: 执行代码审查（核心步骤）

**重要：提交前必须完成代码审查。**

执行与 `/review` 命令完全相同的审查流程：

1. 读取所有暂存文件的 diff 内容
2. 按照 `/review` 中定义的检查项逐项审查：
   - **Go 后端**: 命名规范、错误处理、SQL 安全、运营商硬编码、认证、资源泄漏、测试覆盖
   - **React/TS 前端**: 类型安全、API 模式、Hook 模式、XSS、Token 处理
   - **通用**: 代码重复、硬编码、日志质量
3. 按严重级别分类发现（CRITICAL / WARNING / INFO）
4. 生成审查报告到：
   ```
   docs/review-report/<YYYYMMDD>/REVIEW_<short_hash>_<author>_<scope>.md
   ```
5. 确定审查结论

**审查门禁**：
- `NEEDS_FIX`（存在 CRITICAL）→ **停止提交**，打印所有 CRITICAL 问题，要求用户修复后重新执行
- `PASS` 或 `PASS_WITH_WARNINGS` → 继续提交流程

### Step 6: 暂存审查报告

将审查报告添加到暂存区，与代码一起提交：

```bash
mkdir -p docs/review-report/$(date +%Y%m%d)
# 审查报告已在 Step 5 写入
git add docs/review-report/$(date +%Y%m%d)/REVIEW_*.md
```

**注意**：审查报告根据目标仓库决定存放位置：
- 后端提交 → `omcgo/docs/review-report/YYYYMMDD/`（在 omcgo 内）
- 前端提交 → `omcmb/docs/review-report/YYYYMMDD/`（在 omcmb 内）
- 根仓库提交 → `docs/review-report/YYYYMMDD/`（在根目录）

### Step 7: 构建提交消息

按照 Conventional Commits 格式构建提交消息：

**Subject（第一行）**：
- 格式: `<type>(<scope>): <中文简要描述>`
- 不超过 72 字符
- 中文描述，动词开头：实现 / 修复 / 重构 / 添加 / 更新 / 移除 / 优化 / 调整
- 不以句号结尾

**Body（正文）**：
```
What: [具体变更内容，1-3 句话]
Why: [变更原因/背景]
Impact: [对其他模块或用户的影响，如无则写"无"]
```

**Footer（尾部）— 流程闭环字段**：

> 规则源：`docs/project/dod.md §流程闭环` + `.claude/commands/ship.md`

```
Issue: Closes #NN              或   N/A (<原因，如 流程/工具链提交无关联 Issue>)
Review: <审查报告相对路径>     或   N/A (skipped per /ship 快速通道 <type>)
```

**可选扩展**：

```
[BREAKING CHANGE: <破坏性变更说明>]
[Related: <功能域编号或其它 issue，如 F06, #123>]
[Skip: P1,P2]                  # /ship 快速通道裁剪阶段（见 ship.md §快速通道）
```

**字段来源**（Step 4.5 + Step 5 自动填入）：

| 字段 | 来源 |
|------|------|
| `Issue` | Step 4.5 识别结果（`Closes #NN` 合并时自动关闭）；无关联则 `N/A (<原因>)` |
| `Review` | Step 5 生成报告路径；快速通道跳过则 `N/A (skipped per /ship 快速通道 <type>)` |

### Step 8: 执行提交

在目标仓库目录下执行提交：

```bash
git commit -m "$(cat <<'EOF'
<type>(<scope>): <subject>

What: <what>
Why: <why>
Impact: <impact>

Issue: <Closes #NN 或 N/A (<原因>)>
Review: <docs/review-report/YYYYMMDD/REVIEW_*.md 或 N/A (skipped per /ship 快速通道 <type>)>

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Step 9: 提交后验证

1. `git log --oneline -1` — 确认提交成功
2. `git diff --stat HEAD~1` — 展示变更文件统计
3. 打印提交摘要：
   - 提交哈希
   - 提交消息（第一行）
   - 变更文件数量
   - 审查结论
   - 审查报告路径

---

## 完整示例

```
feat(device): 实现设备列表分页查询与批量操作

What: 新增 device handler 的 List/BatchDelete 接口，支持按 carrier/technology/status 过滤，squirrel 动态 SQL 构建
Why: 设备管理基础功能需求，前端设备列表页面需要对接后端接口
Impact: 新增 GET /api/v1/devices 和 DELETE /api/v1/devices/batch 端点

Issue: Closes #42
Review: docs/review-report/20260312/REVIEW_abc1234_watermelon_device.md
Related: F06

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

---

## 注意事项

- 如果审查发现 CRITICAL 级问题，**绝不自动提交**，必须等用户修复
- 审查报告始终与代码变更在同一个 commit 中
- 每个子仓库独立提交，不要混合 omcgo 和 omcmb 的变更
- 如果是纯文档变更（`docs` type），审查可以简化，仅检查格式和内容
- 使用 HEREDOC 传递提交消息以确保格式正确
- **Footer 闭环**：`Issue:` 字段优先填 `Closes #NN`（合并自动关 Issue）；纯流程/工具链提交无 Issue 时填 `N/A (<原因>)`，不阻塞
- **Review 字段**：审查报告路径或 `N/A (skipped per /ship 快速通道 <type>)`，二选一必填
