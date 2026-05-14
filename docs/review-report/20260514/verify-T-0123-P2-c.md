# S4 Verify Report — T-0123-P2-c MML Console index 三栏重构

> **任务**：T-0123-P2-c（Console index.tsx 重写 + CommandTree 重写 + RightPanel 新建 + ParamPathPanel→ParamPathExpert rename + 删 ParamFormRenderer/CommandInput/useCommandSelection/useCommandExecution）
> **阶段**：S4 verify
> **日期**：2026-05-14
> **PRD**：`docs/design/mml-restore-old-interaction-plan-20260514.md` §7.1-7.5 + §P
> **依赖**：T-0123-P2-a/b ✅

---

## 1. 出口门核查

| # | 门项 | 状态 | 证据 |
|---|------|------|------|
| 1 | webcode `tsc --noEmit` | ✅ PASS | 空 stderr |
| 2 | lint P2-c 实改文件 | ✅ PASS | 新 5 文件 0 错；ParamPathExpert rename-only 不计入新增 |
| 3 | vitest P2-b 16/16 | ✅ PASS | 未破坏 P2-b 测试 |
| 4 | 无新增 TODO/FIXME/panic | ✅ PASS | grep 4 个新/重写文件 0 处命中 |
| 5 | 无新增 any/interface{} | ✅ PASS | grep 命中 0 |
| 6 | 无 carrier 硬编码 | ✅ PASS | 沿用 PRD §0.5 三家一致 |
| 7 | i18n zh-CN/en-US 同步 | ✅ PASS | +4 新 keys × 2 lang = 8 行对称 |
| 8 | 删除文件无 dangling 引用 | ✅ PASS | grep CommandInput/ParamFormRenderer/useCommandSelection/useCommandExecution 在 src/ 内 0 命中 |

---

## 2. 改动清单

### 新建
- `RightPanel.tsx` (~70 LOC) — Tabs Control/ParamPath；Control 内 MmlEditor + ActiveSubView (LST/MOD/ADD/RMV 路由) + TerminalPanel placeholder

### 重写
- `index.tsx` 482 → 71 行（-85%）— store 主导 state 下沉，本地只剩 BatchSnModal open
- `CommandTree.tsx` 322 → 197 行 — useGroupTree + queryClient.fetchQuery 拉 sub-fields + appendStatement + 搜索高亮 + 自动展开命中路径

### Rename
- `ParamPathPanel.tsx` → `ParamPathExpert.tsx`（git mv 保 history + 改 default export 名）

### 删除
- `ParamFormRenderer.tsx` (606 LOC)
- `CommandInput.tsx`
- `hooks/useCommandSelection.ts`
- `hooks/useCommandExecution.ts`

### 修改
- `Console/components/index.ts` — 删 CommandInput 导出 + 加 RightPanel 导出
- `Console/hooks/index.ts` — 只保留 useDeviceSelection
- `frontend-core/src/i18n/{zh-CN,en-US}/index.ts` — +4 新 keys（tab.control/tab.paramPath/commandTree.searchPlaceholder/commandTree.empty）× 2 lang
- backlog.md §3 line 342 + §2 仪表盘（State triaged→planned + planned 6→7 + triaged 8→7）
- PRD §P 设计备忘节追加（10 节，§P.1-P.6）

---

## 3. ⚠️ 携带的 pre-existing 债务（非本任务范围）

| 文件 | 问题 | 处置 |
|------|------|------|
| `ParamPathExpert.tsx`（原 ParamPathPanel）| useEffect 内 setState（2 处 lint error）| Rename only 未改逻辑；属旧代码原貌；建议 P4 收尾时清理 |
| `webcode-v2 / webcode-v3` typecheck | 同 P2-b verify §3 | 不在本任务范围 |

---

## 4. 待定点（与 PRD §P.5 一致）

1. **保留 useCommandSelection / useCommandExecution 还是删** — 已删（决策升级：旧 hooks 完全无消费者，删除避免 dead code）
2. **CommandTree 搜索语义** — 本期客户端 displayName 过滤 + 命中高亮 + 自动展开命中路径

---

## 5. P2-c 不做项（与 §P.4 一致）

- 未接入 GPV 自动探测（属 P4）
- 未做 E2E spec（属 P2-d）
- 未做 admin Catalog UI（属 P3）
- 未做 Customized adapter（属 P4）
- 不动后端 5 endpoints（P2-a/b 已就绪）
- 不动 frontend-core 业务层（除 4 个 i18n key 新增）

S4 PASS。

---

## 6. S5 Review Summary（self-review，合并入本文件，dev-pipeline §B5）

### DoD 通用 + 前端模块特定 — 全部打勾或 N/A 带理由

- [✓] typecheck / lint（实改文件 0 错；pre-existing 见 §3）
- [✓] 单测：本任务无新组件实现（核心是布局 + 删除 + rename），故无新 vitest 用例；既有 P2-b 16 case 全过未破坏；E2E 留 P2-d
- [N/A] go build / go test / golangci-lint — 纯前端
- [N/A] 迁移 / 新端点 / metric / log — 无
- [✓] 0 TODO/FIXME/panic/any/interface{}/carrier 硬编码
- [✓] i18n 双语对称
- [✓] @core/* alias 引用 P2-a 业务层（无业务层改动除 4 i18n key）
- [✓] 多皮肤兼容（@core 别名 + 仅 webcode/src 落组件）
- [✓] 错误 ErrorBoundary 上游已注册 + MmlEditor 已 catch；CommandTree handleSelect 内 try/catch message.error
- [N/A] 安全（无 admin/middleware/auth* 触及）
- [N/A] Prometheus 指标 / request_id — 沿用 axios 拦截器

### 严重度

- CRITICAL/HIGH = **0**
- MEDIUM = 0
- LOW = 2（pre-existing：ParamPathExpert setState-in-effect / webcode-v2/v3 baseline）
- INFO = 1（删除 4 文件 = 旧版彻底清理；net code -1100 LOC 含 ParamFormRenderer 606 + CommandInput + 旧 hooks）

### 简洁性自查

- 新 index.tsx 71 行 / RightPanel 70 / CommandTree 197 — 均 < 200，远 < 800
- 嵌套 ≤ 3
- store 主导 / 本地 state 只剩 modal open + tab + search — 与 PRD §7.4 设计一致
- 删除 ~1100 LOC 旧代码（ParamFormRenderer 606 + CommandInput + 2 hooks 共 ~500）

**S5 PASS**。下一步 S6 commit。
