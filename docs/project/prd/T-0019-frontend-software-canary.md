# PRD: 前端 Software 业务逻辑 — Canary API 消费（T-0019）

> **关联**: Backlog T-0019 / Sprint-04 / Domain=frontend / Type=feat
> **作者**: Claude（代 Owner=前端专家）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施（A 方案：主会话全程深度协作）

---

## 1. 业务背景

T-0018 后端 Software 灰度升级落地后（commit `fd71e4f3`），新增 4 个 Canary Admin endpoints 与 7 个 canary 字段（strategy / canary_stages / current_stage / stage_status / stage_history / auto_advance / auto_advance_minutes）。前端 frontend-core 现有 17 method + 20 hook 完全未消费 canary 能力。R-102 风险描述：「Software 52% 完成度，业务逻辑缺失（任务创建/升级进度/回滚）」。

本任务接 T-0018 链：frontend-core 加 canary 类型 + 4 hook，webcode UpgradePlan 页面显示 canary 进度 + 4 操作按钮。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望在 UpgradePlan 列表看到 canary task 当前 stage（如 "stage 2/4 (10%)"）、stage_status（running/paused），以及 4 操作按钮（推进/暂停/恢复/终止）|
| 部署运维 | 我希望失败率超阈值时（status=paused），按钮高亮提示我介入决策 |
| 开发 / 测试 | 我希望 frontend-core 类型严格（避免 any），mock 模式下 canary 字段默认 strategy='full' 不破坏既有流程 |

---

## 3. 验收标准（Given-When-Then）

### V1 — 类型完整
- **Given** UpgradeTaskInfo 接口
- **When** TypeScript 严格模式编译
- **Then** 含 strategy / canaryStages / currentStage / stageStatus / stageHistory / autoAdvance / autoAdvanceMinutes 7 字段

### V2 — API 映射
- **Given** 后端返 `{strategy: "canary", canary_stages: [...]}`
- **When** mapUpgradeTask 转换
- **Then** 输出 `{strategy: "canary", canaryStages: [...]}` (snake_case→camelCase)

### V3 — 4 API method
- **Given** softwareApi 实例
- **When** 调用 advanceCanary/pauseCanary/resumeCanary/abortCanary
- **Then** 各对应 `POST /upgrade-tasks/:id/{advance|pause-canary|resume-canary|abort-canary}`

### V4 — 4 hook
- **Given** useSoftware 模块
- **When** import { useAdvanceCanary, usePauseCanary, useResumeCanary, useAbortCanary }
- **Then** 4 hook 全部 useMutation + invalidate `['software','tasks']` 和 detail

### V5 — UpgradePlan UI 显示
- **Given** UpgradePlan 列表渲染 canary task
- **When** task.strategy === 'canary'
- **Then** 显示 stage 进度（"stage X/Y - Z%"）+ stageStatus tag + 4 操作按钮（受 status disable 控制）

### V6 — typecheck/lint/build
- `cd omcmb/webcode && npm run typecheck` 0 error
- 仅 pre-existing lint warning（不引入新错误）
- vitest 不退化（T-0055 基线 66.66%/54.7%）

### V7 — Mock 模式向后兼容
- `useMock=true` 时既有 hook 不报错；新 4 hook 调 mock service 默认无 canary task

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| Canary stage 显示 | 一致 | 一致 | 一致 |
| 4 操作按钮 | 一致 | 一致 | 一致 |
| 实际差异 | **无**（前端展示通用层）| **无** | **无** |

---

## 5. 非目标

- ❌ 不改 createUpgradeTask 表单（不加 strategy 选择，本任务仅消费已有 canary task；创建表单后续 PR）
- ❌ 不实现 stage_history timeline 详细 UI（仅最近 stage_status）
- ❌ 不接 webcode-v2/v3（多皮肤同步后续 PR；frontend-core 共享 hook 自动获益）
- ❌ 不补完整 vitest 测试（保持 T-0055 基线，新 hook 不强补）
- ❌ 不实现 stage 编辑器（创建 canary task 用 mock JSON 输入；UI editor 后续 PR）

---

## 6. 依赖

| 依赖 | 状态 |
|------|------|
| T-0018 后端 canary 端点 | ✅ done (fd71e4f3) |
| T-0022 模式参考（前端 page 接 hook 模式）| ✅ done |
| frontend-core useSoftware 既有 20 hook | ✅ 模板 |

---

## 7. 设计备忘

### 7.1 Type 扩展（mock/data/software.ts）

```typescript
// CanaryStage / CanaryStrategy / StageStatus types
export type CanaryStageStatus = 'pending' | 'running' | 'paused' | 'aborted' | 'completed';
export interface CanaryStage {
  percent: number;            // 1-100 cumulative
  failureThreshold: number;   // 1-100 percentage
}
export interface StageHistoryEntry {
  stage: number;
  percent: number;
  devicesInStage: number;
  successCount: number;
  failCount: number;
  failureRate: number;        // 0.0-1.0
  action: string;
  at: string;
  reason?: string;
}

// UpgradeTaskInfo 加 7 字段
export interface UpgradeTaskInfo {
  // ... existing 17 fields ...
  strategy?: 'full' | 'canary';      // default 'full'
  canaryStages?: CanaryStage[];
  currentStage?: number;             // 1-indexed
  stageStatus?: CanaryStageStatus;
  stageHistory?: StageHistoryEntry[];
  autoAdvance?: boolean;
  autoAdvanceMinutes?: number;
}
```

### 7.2 API 4 新 method

```typescript
// softwareApi.ts
async advanceCanary(taskId: string): Promise<void>
async pauseCanary(taskId: string): Promise<void>
async resumeCanary(taskId: string): Promise<void>
async abortCanary(taskId: string): Promise<void>
```

### 7.3 Hook 4 新

```typescript
// useSoftware.ts
export function useAdvanceCanary()
export function usePauseCanary()
export function useResumeCanary()
export function useAbortCanary()
```

模式照抄既有 useSuspendTask（useMutation + invalidates `['software','tasks']` 和 detail key）。

### 7.4 UpgradePlan UI 接入

加 column "Canary Stage"（仅 canary task 显示进度）+ 4 按钮在操作列（受 stage_status disable 控制）。

---

## 8. DoD

- [ ] PRD 七要素全
- [ ] frontend-core type 7 字段加
- [ ] API mapUpgradeTask 7 字段映射 + 4 method
- [ ] useSoftware 4 hook 加
- [ ] UpgradePlan UI 显示 canary stage + 4 按钮
- [ ] `npm run typecheck` 0 error
- [ ] vitest 不退化
- [ ] backlog T-0019 → done

---

*PRD by /dev-pipeline pick T-0019 ULTRATHINK A 方案。*
