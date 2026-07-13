# Agent Motion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add state-driven motion to the embedded Agent panel across all three OMC frontend skins.

**Architecture:** Keep the shared agent protocol unchanged. Derive UI motion from existing panel controller state and message/activity status. Implement per-skin presentation in the existing Agent panel components so each skin keeps its visual language.

**Tech Stack:** React, TypeScript, CSS Modules for `webcode`, Tailwind utility classes for `webcode-v2` and `webcode-v3`, existing `@core/agentkit` state types.

## Global Constraints

- Do not add runtime dependencies.
- Do not change backend or Agent Studio protocol.
- Do not use brand names in UI copy or CSS identifiers.
- Respect `prefers-reduced-motion: reduce`.
- Keep all visible copy internationalized through existing keys.

---

### Task 1: v1 Motion Surface

**Files:**
- Modify: `omcmb/webcode/src/components/AgentPanel/AgentPanel.tsx`
- Modify: `omcmb/webcode/src/components/AgentPanel/AgentPanel.module.css`

**Interfaces:**
- Consumes: `controller.isStreaming`, `message.status`, `activity.status`, `thoughts.status`.
- Produces: state class names that style live connection, active avatar, thought pulse, process activity, activity card running state, and streaming caret.

- [ ] Add active-state class names to the v1 Agent panel.
- [ ] Add CSS keyframes and reduced-motion fallback.
- [ ] Verify the drawer width and message bubble dimensions do not shift.

### Task 2: v2 Motion Surface

**Files:**
- Modify: `omcmb/webcode-v2/src/components/agent/AgentPanel.tsx`

**Interfaces:**
- Consumes: same state as Task 1.
- Produces: Tailwind-only active-state styling matching the light skin.

- [ ] Add live status dot, active avatar ring, thought activity line, process row status dot, running activity style, and streaming caret.
- [ ] Use Tailwind arbitrary animation classes only where existing build supports them.
- [ ] Include `motion-reduce:` static fallbacks.

### Task 3: v3 Motion Surface

**Files:**
- Modify: `omcmb/webcode-v3/src/components/agent/AgentConsole.tsx`

**Interfaces:**
- Consumes: same state as Task 1.
- Produces: Tailwind-only cyber operations motion matching existing v3 scanline style.

- [ ] Add live status glyph, active avatar glow, thought signal bead, process row pulse, running activity edge, and streaming caret.
- [ ] Keep scanline overlay unchanged.
- [ ] Include `motion-reduce:` static fallbacks.

### Task 4: Verification

**Files:**
- Test existing files under `omcmb/frontend-core/src/agentkit`.

- [ ] Run `cd omcmb && npm run test --workspace webcode -- ../frontend-core/src/agentkit/protocol.test.ts ../frontend-core/src/agentkit/runtimeClient.test.ts ../frontend-core/src/agentkit/panel.test.ts`.
- [ ] Run `cd omcmb && npm run typecheck`.
- [ ] Run `cd omcmb && npm run skin-parity`.
- [ ] Inspect git diff for unrelated changes.
