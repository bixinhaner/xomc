# Agent API Handbook Design

## Goal

Turn `omc-operations` from a small fast-path guide into a complete, versioned API handbook that covers every current OMC `/api/v1` route and lets the Agent load only the relevant category and operation document for each request.

## Constraints

- OMC remains the source of truth for routes, request execution, user identity, permissions, and Agent policy.
- Agent Studio remains business-neutral; OMC knowledge lives in the `omc-operations` Skill package.
- The final Agent run profile is `gpt-5.5`, reasoning effort `high`, and Web search enabled. This optimization must not lower reasoning effort or disable Web search.
- Existing Web UI API calls and non-Agent behavior remain unchanged.
- Runtime catalog/describe is not the normal discovery path after this change. A handbook version mismatch is surfaced explicitly instead of silently falling back to repeated remote discovery.

## Architecture

### Runtime route authority

OMC exposes an authenticated Agent-module route export built from the live Gin route table. It includes every current `/api/v1` method/path pair, stable operation ID, handler identity, category, risk, title, description, path parameters, and a deterministic catalog version.

The same version and route count are injected into the external request context. This lets the Skill compare the connected OMC route set with its local `manifest.json` without another network call.

### Build-time handbook generation

A deterministic Go generator combines:

1. Live route export, which guarantees complete route coverage.
2. Existing OpenAPI operations, which provide structured parameters, bodies, responses, examples, and schemas where available.
3. Go handler source analysis, which adds handler comments, literal query/form parameter names, defaults, binding requirements, and request struct fields for routes missing or exceeding the static OpenAPI document.
4. Explicit Skill reference overrides for endpoint-specific empty-result and safety semantics.

The generator fails on duplicate operation IDs, malformed route input, missing generated documents, category count mismatches, or non-deterministic output.

### Progressive Skill layout

```text
omc-operations/
├── SKILL.md
├── agents/openai.yaml
├── references/
│   ├── manifest.json
│   ├── common-operations.md
│   ├── api-categories/<category>.json
│   └── api-docs/<operationId>.json
```

`SKILL.md` contains only the execution workflow, safety rules, version check, and local lookup commands. Category files contain compact intent-oriented entries for every operation. Each operation file contains the complete available request and response contract, examples, risk, confirmation, idempotency, and source confidence.

The Agent first reuses an operation already proven in the thread. Otherwise it searches one local category file, reads one operation document, and calls the business API. Independent business reads may run concurrently. It does not call remote catalog/describe when the local handbook version matches.

### Agent Studio integration

Agent Studio vendors the generated Skill unchanged. The generic Action Connector prompt tells any enabled API-handbook Skill to prefer local progressive references and reuse successful operations. It contains no OMC route names or OMC-specific branches.

The production managed Skill version and Agent Mode binding are refreshed after deployment. The existing run profile is checked before and after deployment to prove `gpt-5.5/high/Web enabled` remains true.

## Verification

- Unit tests: route export, version injection, OpenAPI merge, handler-source extraction, deterministic generation, duplicate detection, and manifest/category/document consistency.
- Coverage gate: generated operation count and method/path set exactly equal the live OMC route export.
- Skill validation: system `quick_validate.py` on both canonical and Agent Studio copies; byte-for-byte tree comparison.
- Repository verification: OMC build/tests, Agent Studio tests/build, and existing frontend parity checks when unaffected code still participates in the normal gate.
- End-to-end: common direct read, at least three long-tail domains, multi-API reasoning, unknown/nonexistent capability, and an unconfirmed write request. Audit must show local handbook reads, no catalog/describe for version-matched requests, correct API calls, and no unintended write.
- Configuration gate: audit the actual persisted run configuration and completed run metadata for model `gpt-5.5`, reasoning `high`, and Web search enabled.
