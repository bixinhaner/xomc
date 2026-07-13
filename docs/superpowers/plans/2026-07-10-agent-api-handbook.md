# Agent API Handbook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate and deploy a complete, versioned OMC API handbook Skill with progressive local loading for every live `/api/v1` operation.

**Architecture:** OMC exports its exact live Gin route set and injects its deterministic version into every connector request. A Go generator merges that inventory with OpenAPI and Go handler contracts into per-category and per-operation Skill references; Agent Studio vendors the generated Skill and its generic connector prompt prefers those local references over remote catalog discovery.

**Tech Stack:** Go 1.25, Gin, Go AST/parser, YAML v3, JSON, TypeScript/Vitest, Codex managed skills, Docker Compose, Action Connector SSE.

## Global Constraints

- Final Agent configuration must remain model `gpt-5.5`, reasoning effort `high`, and Web search enabled.
- Every current OMC `/api/v1` method/path pair must have exactly one generated operation document.
- OMC business knowledge must remain inside the OMC Agent module and `omc-operations` Skill.
- Agent Studio connector runtime changes must remain generic and contain no OMC route names.
- Existing Web UI and non-Agent API behavior must remain unchanged.
- Version-matched runs must not use remote catalog/describe for API discovery.

---

### Task 1: Export and identify the complete live route set

**Files:**
- Modify: `omcgo/internal/agentruntime/handler.go`
- Modify: `omcgo/internal/agentruntime/tool_executor.go`
- Create: `omcgo/internal/agentruntime/handbook_routes.go`
- Test: `omcgo/internal/agentruntime/handbook_routes_test.go`

**Interfaces:**
- Produces: `HandbookRouteExport`, `ToolExecutor.HandbookRouteExport()`, authenticated `GET /api/v1/agent/handbook/routes`
- Injects: `context.externalIdentity.metadata.apiHandbook.catalogVersion` and `totalRoutes`

- [ ] Write failing tests for sorted `/api/v1`-only route export, stable version, handler identity, and external context metadata.
- [ ] Run the focused tests and confirm failures are caused by missing handbook export behavior.
- [ ] Implement route export and metadata injection without changing business route execution.
- [ ] Re-run focused and complete `internal/agentruntime` tests.

### Task 2: Generate deterministic progressive handbook files

**Files:**
- Create: `omcgo/internal/agentruntime/handbookgen/model.go`
- Create: `omcgo/internal/agentruntime/handbookgen/openapi.go`
- Create: `omcgo/internal/agentruntime/handbookgen/gosource.go`
- Create: `omcgo/internal/agentruntime/handbookgen/generator.go`
- Create: `omcgo/internal/agentruntime/handbookgen/generator_test.go`
- Create: `omcgo/cmd/agent-handbook/main.go`

**Interfaces:**
- Consumes: live route-export JSON, `api/openapi/openapi.yaml`, OMC Go source root
- Produces: deterministic `manifest.json`, category indexes, and one JSON document per operation

- [ ] Write failing fixture tests for OpenAPI path normalization, request/response merge, handler query/body extraction, risk metadata, deterministic output, duplicate IDs, and complete route coverage.
- [ ] Run focused tests and confirm each expected behavior fails before implementation.
- [ ] Implement the smallest parser and generator that satisfies those contracts.
- [ ] Run focused tests twice and compare output hashes to prove determinism.

### Task 3: Replace the partial Skill with the complete handbook

**Files:**
- Modify: `omcgo/data/agent-skill/omc-operations/SKILL.md`
- Modify: `omcgo/data/agent-skill/omc-operations/agents/openai.yaml`
- Delete: `omcgo/data/agent-skill/omc-operations/references/domain-index.md`
- Generate: `omcgo/data/agent-skill/omc-operations/references/manifest.json`
- Generate: `omcgo/data/agent-skill/omc-operations/references/api-categories/*.json`
- Generate: `omcgo/data/agent-skill/omc-operations/references/api-docs/*.json`

**Interfaces:**
- Produces: `$omc-operations` with local category search and one-document progressive loading

- [ ] Rebuild the local OMC app with the route-export endpoint and export the exact live inventory.
- [ ] Generate the handbook and assert document count equals live route count with no extra or missing method/path pair.
- [ ] Rewrite `SKILL.md` to use manifest check, local category lookup, exact operation document loading, operation reuse, API execution, and explicit version-mismatch handling.
- [ ] Validate the Skill and run generator `--check` against the live export.

### Task 4: Vendor and activate the handbook in Agent Studio

**Files:**
- Replace: `agent-api/bundled-skills/omc-operations/**`
- Modify: `agent-api/src/integrations/action-connector/default-prompt.ts`
- Test: `agent-api/src/integrations/action-connector/prompt.test.ts`
- Create: `agent-api/src/integrations/action-connector/omc-skill-sync.test.ts`

**Interfaces:**
- Consumes: canonical generated OMC Skill tree
- Produces: byte-identical bundled Skill and generic local-handbook-first connector instructions

- [ ] Write failing tests requiring local handbook priority, version validation, no normal catalog/describe discovery, and byte-identical canonical/bundled trees.
- [ ] Run focused Vitest and confirm failures.
- [ ] Vendor the generated tree and update only generic connector instructions.
- [ ] Re-run focused tests, all Agent Studio tests, build, and Skill validation.

### Task 5: Deploy and verify configuration invariants

**Files:**
- Runtime data only: managed Skill version/package binding and existing Agent run profile

**Interfaces:**
- Preserves: `gpt-5.5`, reasoning effort `high`, Web search enabled

- [ ] Record persisted model, reasoning, and Web settings before deployment.
- [ ] Refresh local OMC Docker services and verify route export/manifest version match.
- [ ] Commit/push/deploy Agent Studio only when permitted by repository instructions and user scope; otherwise run the equivalent local runtime.
- [ ] Refresh the managed Skill version and preserve its Agent Mode package binding.
- [ ] Re-read persisted configuration and completed run metadata; fail verification if any required setting changed.

### Task 6: Perform comprehensive end-to-end validation

**Files:**
- Test artifacts only under each repository's `temp/` directory

**Interfaces:**
- Produces: auditable evidence for correctness, progressive loading, safety, and latency

- [ ] Run common direct-read, three cross-domain long-tail reads, multi-API dependent reasoning, unsupported intent, and unconfirmed write scenarios.
- [ ] Inspect OMC request logs and Agent Studio run events for local handbook reads, business API calls, catalog/describe count, Web usage, write safety, thread continuity, and final answer correctness.
- [ ] Confirm version-matched long-tail runs use zero remote catalog/describe calls.
- [ ] Run OMC build/tests, Agent Studio tests/build, both Skill validators, Docker health checks, and repository status checks.
- [ ] Audit every objective requirement against direct evidence before marking the goal complete.
