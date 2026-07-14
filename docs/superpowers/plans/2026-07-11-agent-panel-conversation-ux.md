# Agent Panel Conversation UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make all three OMC Agent panels follow streamed content intelligently, use resized width effectively, and support practical multi-line prompts.

**Architecture:** Put scroll-state decisions and resize observation in one `frontend-core` hook. Each skin keeps its existing visual implementation and connects the shared refs/actions to its message viewport, responsive bubbles, jump control, and textarea composer.

**Tech Stack:** React 19, TypeScript, Vitest, CSS Modules, Tailwind CSS, existing Ant Design and Lucide icon libraries.

## Global Constraints

- Do not change Agent runtime requests, persistence, protocol, or backend behavior.
- Pause automatic following when the user scrolls upward; resume within 72 px of the bottom.
- Assistant bubbles grow to 640 px; user bubbles remain capped at 78% or 520 px.
- Textarea grows from one to four rows; `Enter` sends and `Shift+Enter` inserts a line break.
- Keep v1, v2, and v3 behavior aligned while preserving each skin's style.

---

### Task 1: Shared Smart Scroll Hook

**Files:**
- Create: `omcmb/frontend-core/src/hooks/useAgentAutoScroll.ts`
- Create: `omcmb/frontend-core/src/hooks/useAgentAutoScroll.test.ts`

**Interfaces:**
- Produces: `isAgentViewportAtBottom(metrics, threshold?)` and `useAgentAutoScroll({ active })`.
- Produces refs `scrollContainerRef`, `scrollContentRef`; state `showJumpToLatest`; callbacks `handleScroll`, `scrollToLatest`.

- [ ] **Step 1: Write failing bottom-threshold tests**

```ts
expect(isAgentViewportAtBottom({ scrollTop: 528, scrollHeight: 1000, clientHeight: 400 })).toBe(true);
expect(isAgentViewportAtBottom({ scrollTop: 400, scrollHeight: 1000, clientHeight: 400 })).toBe(false);
```

- [ ] **Step 2: Run the focused test and verify failure**

Run: `cd omcmb && npm run test --workspace webcode -- ../frontend-core/src/hooks/useAgentAutoScroll.test.ts`

Expected: FAIL because the hook module does not exist.

- [ ] **Step 3: Implement the helper and hook**

```ts
export function isAgentViewportAtBottom(metrics: AgentScrollMetrics, threshold = 72) {
  return metrics.scrollHeight - metrics.clientHeight - metrics.scrollTop <= threshold;
}

export function useAgentAutoScroll({ active }: { active: boolean }) {
  // Track whether the user is pinned, observe content height, and scroll only while pinned.
}
```

- [ ] **Step 4: Run the focused test and verify pass**

Run: `cd omcmb && npm run test --workspace webcode -- ../frontend-core/src/hooks/useAgentAutoScroll.test.ts`

Expected: PASS.

### Task 2: Main Skin Conversation Surface

**Files:**
- Modify: `omcmb/webcode/src/components/AgentPanel/AgentPanel.tsx`
- Modify: `omcmb/webcode/src/components/AgentPanel/AgentPanel.module.css`
- Modify: `omcmb/frontend-core/src/i18n/locales/zh-CN.ts`
- Modify: `omcmb/frontend-core/src/i18n/locales/en-US.ts`

**Interfaces:**
- Consumes: `useAgentAutoScroll({ active: open })`.
- Adds localized `agent.jumpToLatest` accessible label.

- [ ] **Step 1: Connect the scroll viewport and inner content refs**

```tsx
<div ref={scrollContainerRef} className={styles.body} onScroll={handleScroll}>
  <div ref={scrollContentRef} className={styles.messageList}>...</div>
</div>
```

- [ ] **Step 2: Add the jump-to-latest icon control**

```tsx
{showJumpToLatest && (
  <button type="button" className={styles.jumpToLatest} onClick={() => scrollToLatest('smooth')}>
    <ArrowDownOutlined />
  </button>
)}
```

- [ ] **Step 3: Make message widths responsive**

```css
.assistantMessage .bubble { width: min(100%, 640px); max-width: calc(100% - 40px); }
.userMessage .bubble { max-width: min(78%, 520px); }
```

- [ ] **Step 4: Replace the one-line input with the approved textarea behavior**

```tsx
<textarea rows={1} onKeyDown={handleComposerKeyDown} disabled={!controller.enabled} />
```

- [ ] **Step 5: Run main-skin typecheck**

Run: `cd omcmb && npm run typecheck --workspace webcode`

Expected: PASS.

### Task 3: v2 and v3 Behavior Parity

**Files:**
- Modify: `omcmb/webcode-v2/src/components/agent/AgentPanel.tsx`
- Modify: `omcmb/webcode-v3/src/components/agent/AgentConsole.tsx`

**Interfaces:**
- Consumes the Task 1 hook and Task 2 locale key.
- Preserves existing Tailwind and HUD presentation.

- [ ] **Step 1: Add the shared scroll refs, inner content container, and jump control to v2**

```tsx
const autoScroll = useAgentAutoScroll({ active: open });
```

- [ ] **Step 2: Apply role-specific responsive widths and textarea behavior to v2**

- [ ] **Step 3: Add the same behavior to v3 without changing its console styling**

- [ ] **Step 4: Run all skin typechecks and parity**

Run: `cd omcmb && npm run typecheck`

Expected: skin parity and all three TypeScript checks PASS.

### Task 4: Browser Regression

**Files:**
- Verify only; no production source file required.

**Interfaces:**
- Validates the completed behavior through `http://localhost:8081` after rebuilding the local Docker web service.

- [ ] **Step 1: Run focused tests and lint changed files**

Run: `cd omcmb && npm run test --workspace webcode -- ../frontend-core/src/hooks/useAgentAutoScroll.test.ts`

Run: `cd omcmb && npm run lint`

Expected: PASS with no new lint errors.

- [ ] **Step 2: Rebuild the local Docker frontend**

Use the existing local compose topology that owns `omc-web-1`; do not modify tracked deployment files.

- [ ] **Step 3: Verify real browser behavior**

Check normal and expanded widths, initial restore position, send-to-bottom, stream growth, upward user scroll, jump-to-latest, `Enter`, `Shift+Enter`, four-row cap, long Markdown overflow, and narrow viewport layout.

- [ ] **Step 4: Inspect the final diff and report verification evidence**

Run: `git diff --check`

Expected: no whitespace errors.
