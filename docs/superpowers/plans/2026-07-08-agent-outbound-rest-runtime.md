# Agent Outbound REST Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the old remote action connector execution with outbound OMC REST tool execution.

**Architecture:** Agent Studio gets a generic bridge that turns Codex CLI calls into `tool_request` SSE events and waits for `tool_result`. OMC reads those events, executes local `/api/v1` under configurable policy, and posts results back to Agent Studio.

**Tech Stack:** Go/Gin OMC backend, React/Ant Design OMC frontend, TypeScript/Express Agent Studio backend, SSE, existing Agent panel protocol.

## Global Constraints

- Agent Studio must not contain OMC business actions.
- OMC System Config -> Agent owns safety settings.
- First slice defaults to read-only.
- Frontend must match `docs/design/agent-outbound-runtime/*.png`.
- Old `/api/v1/agent-actions` runtime dependency should be removed instead of retained as fallback.
- No brand-specific names in Agent Studio or OMC UI copy.

---

### Task 1: Agent Studio Generic Bridge

**Files:**
- Modify: `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/runtime.ts`
- Modify: `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/routes.ts`
- Modify: `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/index.ts`

**Interfaces:**
- Produces `tool_request` stream events with `runId`, `toolCallId`, `tool`, and `input`.
- Consumes `POST /api/action-connectors/:connectorId/tool-results` with matching `Authorization`.

- [x] Add generic bridge event types and result waiting.
- [x] Change materialized CLI to submit `rest.request` to Agent Studio bridge, not to external system APIs.
- [x] Change prompt to describe `rest.request` and API catalog search as generic behavior.
- [x] Add tests for pending request, matched result, timeout, and auth mismatch.

### Task 2: OMC Runtime Proxy and Tool Executor

**Files:**
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentruntime/handler.go`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentruntime/tool_executor.go`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentruntime/policy.go`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentruntime/handler_test.go`

**Interfaces:**
- Consumes Agent Studio `tool_request` events.
- Produces `tool_result` callbacks to Agent Studio.

- [x] Parse upstream SSE instead of byte-copying it blindly.
- [x] Forward normal events to browser unchanged.
- [x] On `tool_request`, execute local `/api/v1` under policy.
- [x] Post `tool_result` to Agent Studio using the same delegation bearer.
- [x] Add tests for forwarding, tool execution, denied method, and blocked path.

### Task 3: OMC Agent Configuration Policy

**Files:**
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentconfig/model.go`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentconfig/service.go`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentconfig/service_test.go`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/types/agentConfig.ts`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/services/api/agentApi.ts`

**Interfaces:**
- Produces `AgentRuntimePolicy` with allowed methods, blocked prefixes, timeout, and response limit.

- [x] Add persisted settings for allowed methods, blocked paths, timeout seconds, and response limit.
- [x] Include policy in admin and runtime config responses.
- [x] Send policy to Agent Studio during sync as generic connector config.
- [x] Add tests for defaults, patch semantics, and persistence.

### Task 4: OMC Frontend

**Files:**
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/pages/system/SystemConfig/AgentSettings.tsx`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/components/AgentPanel/AgentPanel.tsx`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/components/AgentPanel/AgentPanel.module.css`
- Modify equivalent v2/v3 Agent panels as needed.
- Modify i18n in `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/i18n/*/index.ts`

**Interfaces:**
- Consumes new config policy fields.
- Consumes `tool_request` and `tool_result` events already normalized into Agent panel activities.

- [x] Add Security Policy card matching the reference image.
- [x] Add REST method/path presentation in Agent activity cards.
- [x] Add read-only/write-disabled policy indicator.
- [x] Keep three skin route/menu parity intact.

### Task 5: Validation and Commits

**Files:**
- Validate both repositories.

- [x] Run Agent Studio focused tests and typecheck.
- [x] Run `cd omcgo && go test ./...`.
- [x] Run `cd omcmb && npm run skin-parity`.
- [x] Run `cd omcmb && npm run typecheck`.
- [ ] Commit Agent Studio and OMC separately.
