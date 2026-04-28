// Package snmp implements the SNMP Trap northbound channel for pushing OMC
// alarms to upstream operator OSS systems (CMCC / CTCC / CUCC).
//
// Status: SKELETON — T-0013 (PRD: docs/project/prd/F08-oss-protocol.md).
//
// This package is intentionally isolated. As of T-0013 it is NOT wired into
// production:
//
//   - Not registered in cmd/app/provider/*
//   - Not exposed through cmd/app/router/*
//   - Not subscribed to internal/alarm.Engine output
//   - Not subscribed to NATS subject "oss.alarm.forward"
//   - No PostgreSQL backing — TargetRegistry is in-memory only
//   - No persistent outbox / retries / circuit breakers (deferred to T-0020)
//   - Carrier interface is NOT modified — a default mapper inside this package
//     emulates CMCC field ordering as a placeholder
//
// The skeleton compiles, tests, and exercises the core wiring (Sender,
// TargetRegistry, Engine) so that T-0017 (OSS interconnect) can plug it in
// without further structural change.
//
// Subsequent stages:
//
//   - T-0017: Wire Engine to alarm.Engine + Carrier.MapAlarmToTrapPDU + DI +
//     admin REST API + migration creating snmp_trap_targets.
//   - T-0020: Reliability hardening — retries, persistent outbox shared with
//     northbound/push, circuit breaker, alerting rules.
//
// Dependencies (intentional minimum):
//
//   - stdlib (context, errors, fmt, sync, time, ...)
//   - github.com/gosnmp/gosnmp (Go SNMP standard library)
//   - go.uber.org/zap (structured logging)
//   - github.com/google/uuid (id generation)
//
// FORBIDDEN dependencies inside this package (skeleton isolation contract):
//
//   - internal/alarm/*
//   - internal/carrier/*
//   - internal/core/event/*
//   - internal/core/appconfig/*
//   - internal/northbound/push/*
//   - cmd/*
package snmp
