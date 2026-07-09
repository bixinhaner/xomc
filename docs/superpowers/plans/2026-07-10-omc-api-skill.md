# OMC API Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a production OMC Skill that uses the complete live OMC API catalog, bind it to the External Operations Agent Mode, and reduce common-query latency through progressive context disclosure and direct known-API execution.

**Architecture:** OMC remains the authoritative source for API availability and policy. The Skill carries only routing knowledge, common-operation semantics, and execution discipline; it discovers long-tail operations through the live outbound connector catalog. Agent Studio uses its existing managed-skill and Agent Mode package mechanisms, with no OMC-specific runtime branch.

**Tech Stack:** Go, Gin, TypeScript, Codex managed skills, PostgreSQL/Prisma, Docker Compose, SSE action connector.

## Global Constraints

- OMC owns business API knowledge and policy enforcement.
- Agent Studio runtime code remains business-neutral.
- The Skill must support every live `/api/v1` route allowed by the OMC Agent policy.
- Known read operations bypass catalog/describe; unknown and write operations retain discovery and confirmation.
- Existing OMC web UI calls and non-Agent behavior must remain unchanged.
- Production verification must cover correctness, tool count, token usage, and end-to-end latency.

---

### Task 1: Complete live catalog navigation

**Files:**
- Modify: `omcgo/internal/agentruntime/tool_executor.go`
- Modify: `omcgo/internal/agentruntime/api_catalog.go`
- Test: `omcgo/internal/agentruntime/tool_executor_test.go`

**Interfaces:**
- Consumes: `gin.RoutesInfo`, `agentconfig.RuntimePolicy`
- Produces: paged `/api/v1/agent/catalog`, `/api/v1/agent/catalog/categories`, stable `catalogVersion`

- [ ] Write failing tests proving tokenized search, category filtering, pagination, stable versioning, and category coverage of all policy-visible routes.
- [ ] Run `go test ./internal/agentruntime -run 'TestToolExecutorCatalog' -count=1` and confirm the new assertions fail.
- [ ] Add token-AND search, deterministic ranking, `offset`/`limit`, category summaries, and catalog version hashing.
- [ ] Re-run the focused tests and `go test ./internal/agentruntime -count=1`.

### Task 2: Build and validate the canonical OMC Skill

**Files:**
- Create: `omcgo/data/agent-skill/omc-operations/SKILL.md`
- Create: `omcgo/data/agent-skill/omc-operations/agents/openai.yaml`
- Create: `omcgo/data/agent-skill/omc-operations/references/domain-index.md`
- Create: `omcgo/data/agent-skill/omc-operations/references/common-operations.md`

**Interfaces:**
- Consumes: `.agent-studio/action-connector-cli.mjs`, live catalog endpoints
- Produces: `$omc-operations` Skill usable by any Codex runtime with the Action Connector CLI

- [ ] Initialize `omc-operations` with the system `skill-creator` script.
- [ ] Define direct fast paths, full-catalog fallback, same-thread reuse, batching, safety, and response rules.
- [ ] Keep `SKILL.md` below 500 lines and load domain/common-operation references only when needed.
- [ ] Run `quick_validate.py` and verify every referenced file exists.

### Task 3: Bundle and activate the Skill in Agent Studio

**Files:**
- Create: `agent-api/bundled-skills/omc-operations/**`
- Modify: `agent-api/src/integrations/action-connector/default-prompt.ts`
- Test: `agent-api/src/integrations/action-connector/prompt.test.ts`

**Interfaces:**
- Consumes: canonical Skill artifact from OMC
- Produces: managed Skill source and a generic runtime prompt that prefers enabled Skills

- [ ] Vendor the validated canonical Skill without changing its contents.
- [ ] Add failing prompt tests requiring direct request for Skill-known APIs and conditional catalog/describe behavior.
- [ ] Replace unconditional catalog/describe instructions with Skill-first progressive disclosure.
- [ ] Run focused Vitest, Agent Studio build, and Skill validation.

### Task 4: Configure production and local deployments

**Files:**
- Runtime data: Agent Studio managed skill, skill package, Agent Mode binding, runtime prompt, workspace AGENTS.md, run profile

**Interfaces:**
- Consumes: deployed bundled Skill path and existing Agent Mode repositories
- Produces: active `omc-operations` selection in External Operations Codex RunConfig

- [ ] Commit OMC locally and rebuild `docker compose -p omc -f deployments/docker/docker-compose.yml`.
- [ ] Commit and push Agent Studio `main`, then deploy with `scripts/deploy-agent-studio.sh`.
- [ ] Install/update the managed Skill from the deployed bundled path, create/update its package, and bind it to the External Operations Agent Mode.
- [ ] Set `gpt-5.5`, reasoning `medium`, web search `disabled`, network access enabled, and slim duplicate Agent instructions.
- [ ] Verify the materialized Action Connector CODEX_HOME contains `skills/omc-operations` and the thread RunConfig selects only that Skill.

### Task 5: Benchmark and iterate

**Files:**
- No required source changes; revise Skill/prompt/catalog only when evidence identifies a bottleneck.

**Interfaces:**
- Consumes: OMC browser UI, Agent Studio audit/usage records, OMC request logs
- Produces: evidence for correctness and user-perceived latency

- [ ] Test common read scenarios: online devices, active alarms, network anomaly summary, users, and system status.
- [ ] Test long-tail discovery, multi-API dependent reasoning, and a write request that must stop for confirmation/policy.
- [ ] Record total latency, time to first tool, catalog/describe/request counts, input/cached/output tokens, and answer correctness.
- [ ] Iterate until common reads use one direct request where sufficient and achieve a practical target of P50 8–15 seconds, with long-tail queries normally within 15–25 seconds.
- [ ] Re-run smoke, focused tests, repository status checks, production health checks, and browser verification before completion.
