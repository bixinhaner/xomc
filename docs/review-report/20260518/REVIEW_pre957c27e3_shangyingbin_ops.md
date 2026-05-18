# Code Review — TR069 报文跟踪抽屉表格滚动修复

| 字段 | 值 |
|------|----|
| Reviewer | Claude (auto) |
| Author | shangyingbin |
| Scope | ops (frontend) |
| Type | fix (HOTFIX 快速通道) |
| Pre-commit diff hash | 957c27e3 |
| Date | 2026-05-18 |
| Backlog | HOTFIX (来自用户 TODO.MD 反馈，S7 阶段补登记) |

## 变更范围

仅一处一行修改：

```
omcmb/webcode/src/pages/ops/MessageTrace/index.tsx (+1 / -0)
```

`MessageList` 组件（Drawer "查看报文" 内）给 `<DataTable>` 添加 `scroll` 属性：

```tsx
scroll={{ x: 'max-content', y: 'calc(100vh - 260px)' }}
```

## 缺陷与现象

**用户反馈**（TODO.MD 第 7 行）：
> TR069报文跟踪，查看报文页面，没有纵向滚动条，滚动鼠标时页面也不动，最底部的条目看不到

**根因**：
- `DataTable` 的 `.dataTableWrapper { height: 100% }` 在 `<Drawer>` 内被锁死成抽屉 body 高度
- `.tableContainer { flex: 1; overflow: hidden }` 期望 antd Table 自己产生纵向滚动（即调用方传 `scroll.y`）
- 但 `MessageList` 原代码没传 `scroll`，antd Table 按内容自然高度渲染，超出部分被 `overflow: hidden` 直接裁掉 → 无滚动条、滚轮无响应、底部行不可见

## 审查清单

### 通用
- [x] 改动最小化：单行新增，无副作用
- [x] 命名一致：复用已有 `DataTableProps.scroll` 接口（`{ x?: number | string; y?: number | string }`）
- [x] 无硬编码运营商逻辑
- [x] 无 SQL / XSS / 注入风险（纯 UI 属性）
- [x] 无敏感信息泄露

### 前端专项
- [x] TypeScript 类型安全：`scroll` 已在 `DataTableProps` 定义，字面量结构匹配，typecheck 通过
- [x] 无 `any` / `interface{}`
- [x] 不影响 API / Hook / Store 形态，对 `webcode-v2/v3` 多皮肤包零影响（改的是 `webcode/` 内页面）
- [x] 不引入新依赖

### 影响面
- 受益：抽屉内表格出现纵向滚动条 + 表头悬浮 + 滚轮响应，底部行可见
- 风险：`calc(100vh - 260px)` 在极端窗口高度（<400px）下可视区会过小，但 antd Table 仍可滚动，最坏情况只是 UI 紧凑，无功能损失
- 兼容性：未触及外层 TraceTask 列表表格，不破坏现有行为

## 浏览器验证证据

部署链路：`bash /Users/shangyingbin/project/omc-docker/docker-run.sh`（标准 Docker 部署，4 容器全部健康）

Playwright 自动化验证（http://localhost:8081 → 运维管理 → TR069 报文跟踪 → "查看报文"，106 条报文任务）：

```json
{
  "scrollHeight": 820,
  "clientHeight": 651,
  "scrollable": true,
  "overflowY": "scroll",
  "maxHeight": "651px"
}
```

滚到底部后：

```json
{
  "rowCount": 20,
  "scrolledTo": 169,
  "maxScroll": 169,
  "lastRowVisible": true
}
```

最后一行已进入可视区，符合预期。

## 审查结论

**PASS** — 无 CRITICAL、无 WARNING，仅 1 项 INFO：

- **INFO**：`260px` 是基于"抽屉头 56 + body padding 48 + 工具栏 52 + 分页栏 52 + 安全余量"经验值。已在对话中说明，不在代码中加注释（避免噪声）；若后续布局尺寸变更需同步调整该数值。

允许提交。

## 快速通道说明（HOTFIX）

依据 `.claude/commands/dev-pipeline.md §C` 快速通道：

- 变更性质：UI bug 修复，单行改动，影响面已通过 Playwright 实测验证
- 来源：用户 TODO.MD 第 7 行手记反馈，未先登记 Backlog
- S7 补办：本次 commit 后建议在 `docs/project/backlog.md` 追加 hotfix Task 条目（type=bugfix, prio=P3, scope=frontend/ops），State 直接 done，引用本 commit 哈希
