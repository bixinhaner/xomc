# MML Execution Mode Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace ambiguous MML script mode names with action-oriented labels and show always-visible explanations where users choose or review the mode.

**Architecture:** Keep backend values `common` and `device_bound` unchanged. Put localized labels and descriptions in `frontend-core`, render a reusable Ant Design selector in v1, and align the read-only mode summaries in v2/v3 with their native visual systems.

**Tech Stack:** React 19, TypeScript, Ant Design 6, Tailwind-based v2/v3 skins, react-intl, Vitest.

## Global Constraints

- User-facing Chinese labels are `统一脚本批量执行` and `按设备编排执行`.
- Explanations stay visible beside the choices; they are not hidden in tooltips.
- `common` and `device_bound` API values and execution behavior do not change.
- Existing unrelated workspace files remain untouched.

---

### Task 1: Shared copy and v1 selectable cards

**Files:**
- Create: `omcmb/webcode/src/pages/mml/ScriptTask/ExecutionModeSelector.tsx`
- Create: `omcmb/webcode/src/pages/mml/ScriptTask/__tests__/ExecutionModeSelector.test.tsx`
- Modify: `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Consumes: `MMLTaskExecuteMode`, the existing `TranslateFn`, selected mode, and automatic device-bound detection.
- Produces: `ExecutionModeSelector` with `value`, `deviceBoundDetected`, and `onChange` props.

- [x] Write a component test asserting both action-oriented labels, both visible descriptions, selection, and disabled fallback behavior.
- [x] Run `npm test --workspace webcode -- ExecutionModeSelector.test.tsx` and confirm it fails because the component does not exist.
- [x] Implement the selector and localized copy, then replace the v1 button-only radio group.
- [x] Re-run the focused test and confirm it passes.

### Task 2: Align remaining MML surfaces and documentation

**Files:**
- Modify: `omcmb/webcode/src/pages/mml/components/ScriptTaskDrawer.tsx`
- Modify: `omcmb/webcode-v2/src/pages/mml/ScriptTask.tsx`
- Modify: `omcmb/webcode-v3/src/pages/mml/script/index.tsx`
- Modify: `docs/design/mml-script-task-device-bound-redesign-20260708.md`

**Interfaces:**
- Consumes: parsed `common | device_bound` execution mode.
- Produces: clear mode title plus one-sentence explanation on each skin.

- [x] Replace stale `普通模式` / `按设备计划行` wording and make the explanatory copy visible in execution/preview surfaces.
- [x] Update the design document naming table and UI guidance.
- [x] Run `npm run skin-parity` and `npm run typecheck` from `omcmb`.
- [x] Run the focused test again and inspect `git diff --check` plus the scoped diff.
