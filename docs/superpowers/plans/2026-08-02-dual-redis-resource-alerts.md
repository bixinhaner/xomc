# Dual Redis Resource Alerts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate false resource-plan critical alerts for the physically separated Core and PM Redis containers.

**Architecture:** Keep Prometheus selectors bounded and explicit. Extend the existing executable cAdvisor contract test, then update only the four affected resource-plan expressions.

**Tech Stack:** Prometheus rules, Bash contract tests, promtool.

## Global Constraints

- Match `redis-core` and `redis-pm` explicitly; do not use a broad `redis.*` matcher.
- Cover CPU quota, CPU period, memory limit, and CPU/memory drift expressions.
- Render drift summaries from the normalized `compose_service` label.
- Do not change alert duration, severity, labels, or unrelated dashboards.

---

### Task 1: Protect dual Redis resource-plan selectors

**Files:**
- Modify: `deployments/monitoring/alerts/resource-plan-alerts.test`
- Modify: `deployments/monitoring/alerts/host-container-alerts.yml`

**Interfaces:**
- Consumes: cAdvisor `container_label_com_docker_compose_service` values.
- Produces: bounded PromQL selectors matching both physical Redis services.

- [ ] **Step 1: Write the failing test**

Add live `redis-core` and `redis-pm` quota, period, memory-limit, and `container_last_seen` input series. Declare matching schema-v3 resource-plan values and require that neither service produces a missing-series alert. Add the required freshness input to the existing Worker fixture so the test exercises the real `and on (id)` boundary.

Add a second fixture whose Redis Core actual CPU quota is 3 cores while the plan declares 2 cores. Require `OMCResourcePlanDrift` with summary naming `redis-core`.

- [ ] **Step 2: Run test to verify it fails**

Run: `docker run --rm -v "$PWD/deployments/monitoring/alerts:/rules:ro" --entrypoint promtool prom/prometheus:v2.51.0 test rules /rules/resource-plan-alerts.test`

Expected: FAIL with unexpected `redis-core` and `redis-pm` missing-series alerts for CPU quota, CPU period, and memory limit.

- [ ] **Step 3: Write minimal implementation**

Replace only the four affected bounded selectors in `host-container-alerts.yml`, changing `redis` to `redis-core|redis-pm`, and render the drift summary from `$labels.compose_service`.

- [ ] **Step 4: Run focused and release verification**

Run:

```bash
bash deployments/monitoring/tests/validate-cadvisor-service-labels.sh
bash deployments/release/bundle/deploy/storage-compose_test.sh
```

Expected: both commands pass with zero failures.

- [ ] **Step 5: Validate Prometheus rules**

Run promtool against every monitoring rule file and require success.

- [ ] **Step 6: Commit**

Commit the test and rule change with a Conventional Commit message.
