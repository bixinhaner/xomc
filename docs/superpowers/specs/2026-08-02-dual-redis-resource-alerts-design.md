# Dual Redis Resource Alert Design

## Problem

Production now exposes cAdvisor services `redis-core` and `redis-pm`, while the resource-plan alert selectors still match only the removed `redis` service name. The declared schema-v3 resource series therefore remain unmatched and emit false critical missing-series alerts even though quota, period, and memory-limit metrics exist.

## Design

Use the explicit bounded dependency selector `postgres|postgres-tsdb|redis-core|redis-pm|nats|minio|web` wherever resource-plan alerts compare cAdvisor limits with `resources.env`. Explicit names avoid accidentally admitting helper or migration containers through a broad `redis.*` selector.

Add an executable monitoring contract test that extracts every resource-plan alert block and requires both physical Redis services while rejecting the legacy standalone `redis` alternative. The same test validates CPU quota, CPU period, memory limit, and drift expressions.

## Acceptance

- The regression test fails against `5ea2b31b3` because both physical Redis names are absent.
- The monitoring validation suite and Prometheus rule validation pass after the minimal selector change.
- After deployment, Prometheus sees quota/period/memory series for both Redis services and no `OMCResourcePlan*SeriesAbsent` or `OMCResourcePlanDrift` alert remains active.

