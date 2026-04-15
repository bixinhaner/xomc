# Code Review: 修复设备列表 pageSize 超出后端校验限制

**日期**: 2026-04-15
**审查范围**: omcmb — useDeviceSelection hook
**审查结论**: PASS

---

## 审查文件

| # | 文件 | 变更类型 |
|---|------|---------|
| 1 | `omcmb/webcode/src/pages/mml/Console/hooks/useDeviceSelection.ts` | 修复 — pageSize 1000→100 |

## 发现

### CRITICAL — 无

### WARNING — 无

### INFO

**I1. pageSize 对齐后端校验** — 后端 `ListRequest.PageSize` 限制 `max=100`，前端从 1000 改为 100，消除 400 Bad Request 错误

---

## 审查清单

- [x] TypeScript 编译通过
- [x] 修复符合后端约束
