# S4 Verify Report — T-0123-P2-b MML Console 5 个新 UI 组件

> **任务**：T-0123-P2-b（MML Console 前端 5 个新组件：MmlEditor / SubFieldChecklist / SubFieldInputList / InstancePicker / StepBar）
> **阶段**：S4 verify（dev-pipeline §B4）
> **日期**：2026-05-14
> **PRD**：`docs/design/mml-restore-old-interaction-plan-20260514.md` v2 APPROVED（§7.1-7.5 + §O 设计备忘）
> **关联 Risk**：R-206 Mitigating（关闭仍需 P2-c/d + P3 + P4）
> **依赖**：T-0123-P2-a 已闭环（commit `3c760ce9` frontend-core 业务层）

---

## 1. 出口门核查

| # | 门项 | 状态 | 证据 |
|---|------|------|------|
| 1 | webcode `npm run typecheck` (tsc --noEmit) | ✅ PASS | 空 stderr；5 新 .tsx + 5 新 .test.tsx 全过 |
| 2 | webcode `npm run lint` (ESLint on changed files) | ✅ PASS | 空输出 |
| 3 | 单测：5 测试文件 16 case | ✅ PASS | `vitest run` 16/16 pass（StepBar 3 + SubFieldChecklist 3 + SubFieldInputList 4 + InstancePicker 3 + MmlEditor 3）|
| 4 | 全量 vitest 不引入 regression | ✅ PASS | 总 98/98 tests pass；4 pre-existing 文件 `Failed to resolve import "zustand"` 已 stash 复核 = baseline 同样失败 |
| 5 | 多皮肤 webcode-v2 typecheck | ⚠️ pre-existing FAIL | `src/pages/topology/index.tsx(101,3): Cannot find name 'useEffect'` — stash i18n 复核 = baseline 同样失败，与 P2-b 改动无关，遗留债务 |
| 6 | 多皮肤 webcode-v3 typecheck | ⚠️ pre-existing FAIL | `src/pages/alarms/index.tsx(130,27): Property 'alarmCode' does not exist on type 'Alarm'` — 同上，pre-existing |
| 7 | 迁移 self-check | N/A | 纯前端任务，无 DB 变更 |
| 8 | 新端点 E/R | N/A | 无新端点（API 由 P2-a 已 ship；本期为消费侧）|
| 9 | metric/log 命名 grep 命中 | N/A | 组件层暂无新增遥测；按 §O.10 延 P4 统一接入 |
| 10 | 累计型依赖阈值 | N/A | 无 `@累计≥N` 依赖 |
| 11 | 无新增 TODO/FIXME/panic | ✅ PASS | grep 5 个新 .tsx 文件 0 处命中 |
| 12 | 无新增 `any` / `interface{}` | ✅ PASS | grep 5 个新 .tsx + 5 个 .test.tsx 全部命中 0；测试 mock 用泛型 `<T,>(selector: ...) => T` 严格签名 |
| 13 | 无 `if carrier == "cmcc/ctcc/cucc"` 硬编码 | ✅ PASS | grep 5 个新文件 0 处命中（PRD §0.5 三家一致，无 carrier 分支） |
| 14 | i18n 新增 key zh-CN / en-US 同步 | ✅ PASS | 9 新 keys × 2 lang = 18 行；diff 行数完全对称 |
| 15 | 多皮肤共享业务层无破坏 | ✅ PASS | 5 个新组件全部用 `@core/*` alias 引用 store/hooks/types；新增物只在 `webcode/src/pages/mml/Console/components/` |

**结论**：S4 PASS（多皮肤 typecheck 因 pre-existing 状态降级为 ⚠️，详 §3 处置建议）

---

## 2. 改动清单

### 新增文件（10 个）
- `omcmb/webcode/src/pages/mml/Console/components/MmlEditor.tsx` (~120 LOC)
- `omcmb/webcode/src/pages/mml/Console/components/SubFieldChecklist.tsx` (~80 LOC)
- `omcmb/webcode/src/pages/mml/Console/components/SubFieldInputList.tsx` (~75 LOC)
- `omcmb/webcode/src/pages/mml/Console/components/InstancePicker.tsx` (~45 LOC)
- `omcmb/webcode/src/pages/mml/Console/components/StepBar.tsx` (~25 LOC)
- 同目录 `__tests__/` 5 个 `.test.tsx` 文件，共 16 case

### 修改文件（3 个）
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`：+9 i18n keys（editor.execute / parsing / onRebootConfirm / input.constraint / picker.indexLabel / picker.indexHelp / stepBar.step1..4）
- `omcmb/frontend-core/src/i18n/en-US/index.ts`：+9 i18n keys（对称翻译）
- `omcmb/webcode/src/test/setup.ts`：+12 行 `matchMedia` polyfill（antd 在 jsdom 环境必需；既有 `useResponsive.test.tsx` 自带 Object.defineProperty 覆盖，不冲突）

### PRD 追加
- `docs/design/mml-restore-old-interaction-plan-20260514.md`：追加 §O T-0123-P2-b S2 设计备忘节（O.1-O.10，含 5 组件 Props 契约 + 元数据差异化落地表 + 单测策略 + 多皮肤兼容 + 待定点）

### Backlog 回写
- §3 Active 表 line 342：State `triaged → planned`、Owner `— → Claude`、Sprint `sprint-11 候选 → sprint-11`
- §2 仪表盘：planned 6→7 / triaged 9→8

---

## 3. ⚠️ 多皮肤 typecheck pre-existing 失败处置建议

**事实**：
- P2-a backlog evidence 写 "webcode/v2/v3 三皮肤 typecheck ✓"，但当前实测 v2/v3 baseline 已坏
- 我用 `git stash` 移除本任务全部前端改动后实测 v2/v3 typecheck 仍同样失败 → 与 P2-b 无关
- 失败位置（v2: `topology/index.tsx` 缺 useEffect 导入；v3: `alarms/index.tsx` Alarm 类型缺 alarmCode 字段）均与 MML / i18n / Console 无任何路径关联

**处置建议**（不在 P2-b 内修，避免 scope creep）：
1. 在 backlog 新登记 `T-NNNN frontend 多皮肤 v2/v3 typecheck 回归修复`（P2 级）
2. 或并入 P2-d "验证 + DoD"任务（如该任务定位为 P2 系列收尾，则负责清理三皮肤 baseline）
3. 不阻塞本任务 S5 review / S6 commit / S7 handoff —— P2-b 自身 5 文件 typecheck + lint + 16 单测全过

---

## 4. 待定点（与 PRD §O.9 一致）

1. **InstancePicker GPV 自动探测延 P4**：本期手输 index + min=0 校验作为 fallback；P4 接 `/ops/commands/rpc` GPV 探测实例列表。

---

## 5. 后续阶段衔接

- **S5 review**：调 `/review` 主入口 + `/simplify` 末端复查；DoD 逐项勾选
- **S6 commit**：footer 五元组（PRD/Sprint/Risk/Backlog/Review）+ 不绕过 pre-commit hook
- **S7 handoff**：State `in_review → done`；归档到 `backlog/done/2026Q2.md`；§6 速览刷新；changelog 顶部加一行

S4 PASS。
