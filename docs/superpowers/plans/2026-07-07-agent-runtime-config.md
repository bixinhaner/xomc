# Agent Runtime Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a production Agent configuration page in goomc that provisions a generic Agent Studio Action Connector and feeds the Agent panel from runtime configuration.

**Architecture:** goomc is the source of truth for Agent integration settings and stores them in `sys_configs.agent`. agent-studio exposes a generic service-token-protected `action_connector` provision endpoint. The browser reads runtime config from goomc after login and no longer requires `VITE_AGENT_*` for production.

**Tech Stack:** Go/Gin/PostgreSQL `sys_configs`; React/TypeScript/TanStack Query/Ant Design/shadcn/Tailwind; Express/TypeScript/Prisma-style repositories in agent-studio.

## Global Constraints

- New Agent Studio code must not contain goomc or brand-specific names.
- goomc UI must add Agent settings to all three skins.
- The Agent Studio service token is server-side only and must be masked in frontend reads.
- Connector policy remains read-only: read actions allowed, low-risk and high-risk disabled.
- Keep existing Docker dirty files untouched.
- Do not use MCP.

---

### Task 1: Agent Studio Generic Provision API

**Files:**
- Create: `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/provision-router.ts`
- Modify: `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/index.ts`
- Test: `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/provision-router.test.ts`

**Interfaces:**
- Consumes: existing `IntegrationCenterService.saveInstance`, `findInstanceBySlug`, and `validateInstance`.
- Produces: `POST /api/integrations/action-connectors/provision` with response `{ connectorId, slug, runtimeStreamPath, runtimeStreamUrl, status }`.

- [ ] **Step 1: Write router tests**

Create tests that cover service token rejection through existing middleware, create-by-slug, update-by-slug, and remote validation failure shape.

- [ ] **Step 2: Implement `createActionConnectorProvisionRouter`**

Use zod input:

```ts
{
  slug: string;
  name: string;
  description?: string | null;
  status?: "draft" | "active" | "disabled" | "error";
  config: {
    displayName: string;
    baseUrl: string;
    healthPath?: string;
    actionListPath?: string;
    actionSearchPath?: string;
    actionDescribePath?: string;
    actionPreviewPath?: string;
    actionExecutePath?: string;
    delegationHeader?: string;
    policy?: {
      allowReadActions?: boolean;
      allowLowRiskActions?: boolean;
      allowHighRiskActions?: boolean;
    };
  };
}
```

- [ ] **Step 3: Mount the router**

Mount at `/api/integrations/action-connectors` behind `requireServiceToken`, separate from `/api/action-connectors/:connectorId/chat/stream`.

- [ ] **Step 4: Run tests and commit**

Run targeted tests, then commit `feat(integrations): 添加 action connector provision api`.

### Task 2: goomc Backend Agent Config Service

**Files:**
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentconfig/model.go`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentconfig/service.go`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentconfig/handler.go`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentconfig/service_test.go`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/cmd/app/provider/router.go`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcgo/cmd/app/provider/admin.go`

**Interfaces:**
- Consumes: `admin.SysConfigRepository` and `admin.SysConfigService.BatchUpsert`.
- Produces:
  - `GET /api/v1/admin/agent-config`
  - `POST /api/v1/admin/agent-config`
  - `POST /api/v1/admin/agent-config/test`
  - `POST /api/v1/admin/agent-config/sync`
  - `GET /api/v1/agent/config`

- [ ] **Step 1: Write service tests**

Cover token masking, keep-existing-token updates, URL normalization, stream URL derivation, provision success, and provision failure status persistence.

- [ ] **Step 2: Implement service**

Use `sys_configs` category `agent`; save secrets only through backend calls; call Agent Studio provision endpoint with `Authorization: Bearer <service token>`.

- [ ] **Step 3: Implement handler and routes**

Admin endpoints use existing admin permission group; runtime endpoint uses authenticated v1 group and returns only safe fields.

- [ ] **Step 4: Run Go tests and commit**

Run `cd omcgo && go test ./internal/agentconfig ./internal/agentaction ./internal/admin`, then commit `feat(agent): 添加运行期配置服务`.

### Task 3: frontend-core Runtime Config API

**Files:**
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/services/api/agentApi.ts`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/hooks/useAgentRuntimeClient.ts`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/hooks/api/useAgentConfig.ts`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/types/agentConfig.ts`
- Modify tests under `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/services/api/__tests__` and `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/config`.

**Interfaces:**
- Consumes: `GET /agent/config`.
- Produces: `useAgentRuntimeClient` that prefers backend runtime config and falls back to `VITE_AGENT_*`.

- [ ] **Step 1: Add typed Agent config API methods**

Add `getRuntimeConfig`, `getAdminConfig`, `saveAdminConfig`, `testAdminConfig`, and `syncAdminConfig`.

- [ ] **Step 2: Add hooks**

Add `useAgentRuntimeConfig`, `useAdminAgentConfig`, `useSaveAdminAgentConfig`, `useTestAdminAgentConfig`, and `useSyncAdminAgentConfig`.

- [ ] **Step 3: Update runtime client hook**

Use backend config when loaded; keep env fallback while loading or when backend returns disabled.

- [ ] **Step 4: Run frontend tests/typecheck and commit**

Run targeted frontend tests and `npm run typecheck`, then commit `feat(agent): 支持前端运行期配置`.

### Task 4: goomc Three-Skin Agent Settings UI

**Files:**
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/pages/system/SystemConfig/index.tsx`
- Create: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/pages/system/SystemConfig/AgentSettings.tsx`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v2/src/pages/system/SystemConfig.tsx`
- Modify: `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v3/src/pages/system/SystemConfig.tsx`
- Modify i18n files under all three skins or shared core where existing Agent strings live.

**Interfaces:**
- Consumes: frontend-core Agent config hooks.
- Produces: one Agent configuration tab in each skin with matching field semantics.

- [ ] **Step 1: Implement v1 Ant Design form**

Match the generated classic admin mockup: structured form, status badge, test/save/sync actions, masked token behavior.

- [ ] **Step 2: Implement v2 Tailwind/shadcn editor**

Match the generated light compact mockup with existing component primitives and no nested card layout.

- [ ] **Step 3: Implement v3 neon HUD editor**

Match the generated v3 mockup and add save/test controls; v3 must become editable for the Agent category even if other categories remain readonly.

- [ ] **Step 4: Run parity/typecheck and commit**

Run `cd omcmb && npm run skin-parity && npm run typecheck`, then commit `feat(agent): 添加系统配置 agent 页签`.

### Task 5: Final Verification

**Files:**
- No new files unless test snapshots require updates.

**Interfaces:**
- Consumes: all previous tasks.
- Produces: verified local implementation and final status report.

- [ ] **Step 1: Run agent-studio tests**

Run targeted API tests and typecheck/build available in package scripts.

- [ ] **Step 2: Run goomc backend tests**

Run `cd omcgo && go test ./internal/agentconfig ./internal/agentaction ./internal/admin`.

- [ ] **Step 3: Run goomc frontend checks**

Run `cd omcmb && npm run skin-parity && npm run typecheck`.

- [ ] **Step 4: Inspect git status**

Confirm only intended files are staged/committed and existing Docker dirty files remain untouched.
