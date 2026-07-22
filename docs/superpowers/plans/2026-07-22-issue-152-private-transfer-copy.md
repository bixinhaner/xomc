# Issue 152 Private Transfer Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct the ACS transfer help text so it states that production allows operator-network-reachable private addresses.

**Architecture:** Keep the backend validator and transfer consumers unchanged. Lock the user-visible rule in the existing `TransferSettings` component test, then update the shared zh-CN and en-US message resources.

**Tech Stack:** React 19, React Intl, Vitest, TypeScript.

## Global Constraints

- Production allows private addresses reachable from base-station networks.
- Production still rejects localhost, loopback, link-local, and unspecified literal addresses.
- Do not change backend validation or transfer behavior.
- Do not include unrelated dirty-worktree files.

---

### Task 1: Correct and lock the reachability help text

**Files:**
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/TransferSettings.test.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Consumes: `TransferSettings` and the shared `system.transfer.deviceReachabilityHelp` message key.
- Produces: Correct zh-CN/en-US help text rendered by the existing configuration page.

- [ ] **Step 1: Write the failing test**

Add a test that renders `TransferSettings` and asserts the Chinese help text contains `生产环境允许运营商网络中基站可达的私网地址` and does not contain `生产环境不能使用 localhost、回环、链路本地或 RFC1918 私网地址`.

- [ ] **Step 2: Run the test and verify RED**

Run: `cd omcmb && npm run test --workspace webcode -- src/pages/system/SystemConfig/TransferSettings.test.tsx`

Expected: FAIL because the current resource still says production rejects RFC1918 private addresses.

- [ ] **Step 3: Update the two message resources**

Set the Chinese value to:

```text
填写的地址必须能从设备所在网络访问；生产环境允许运营商网络中基站可达的私网地址，但不能使用 localhost、回环、链路本地或未指定地址。
```

Set the English value to:

```text
The address must be reachable from device networks. Production allows private addresses reachable within the operator network, but rejects localhost, loopback, link-local, and unspecified addresses.
```

- [ ] **Step 4: Run the test and verify GREEN**

Run: `cd omcmb && npm run test --workspace webcode -- src/pages/system/SystemConfig/TransferSettings.test.tsx`

Expected: 1 test file passes and all tests in it pass.

- [ ] **Step 5: Run frontend type checking**

Run: `cd omcmb && npm run typecheck`

Expected: exit code 0.

- [ ] **Step 6: Review the scoped diff**

Run: `git diff --check` and inspect only the three files listed above plus this plan.

Expected: no whitespace errors and no backend behavior changes.
