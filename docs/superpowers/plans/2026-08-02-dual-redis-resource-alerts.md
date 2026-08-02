# Dual Redis Resource Alerts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate false resource-plan critical alerts for the physically separated Core and PM Redis containers.

**Architecture:** Keep Prometheus selectors bounded and explicit. Extend the existing executable cAdvisor contract test, then update only the four affected resource-plan expressions.

**Tech Stack:** Prometheus rules, Bash contract tests, promtool.

## Global Constraints

- Match `redis-core` and `redis-pm` explicitly; do not use a broad `redis.*` matcher.
- Cover CPU quota, CPU period, memory limit, and CPU/memory drift expressions.
- Do not change alert duration, severity, labels, or unrelated dashboards.

---

### Task 1: Protect dual Redis resource-plan selectors

**Files:**
- Modify: `deployments/monitoring/tests/validate-cadvisor-service-labels.sh`
- Modify: `deployments/monitoring/alerts/host-container-alerts.yml`

**Interfaces:**
- Consumes: cAdvisor `container_label_com_docker_compose_service` values.
- Produces: bounded PromQL selectors matching both physical Redis services.

- [ ] **Step 1: Write the failing test**

Extract the `OMCResourcePlanCPUQuotaSeriesAbsent` through `OMCResourcePlanDrift` rules and assert that their selectors contain `redis-core|redis-pm` and do not contain the legacy `|redis|` alternative.

- [ ] **Step 2: Run test to verify it fails**

Run: `bash deployments/monitoring/tests/validate-cadvisor-service-labels.sh`

Expected: FAIL identifying that the resource-plan rules omit the dual Redis services.

- [ ] **Step 3: Write minimal implementation**

Replace only the four affected bounded selectors in `host-container-alerts.yml`, changing `redis` to `redis-core|redis-pm`.

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

