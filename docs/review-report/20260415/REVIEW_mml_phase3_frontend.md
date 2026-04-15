# Code Review Report

| Field | Value |
|-------|-------|
| Date | 2026-04-15 |
| Scope | mml (前端 Phase 3 对接) |
| Files | 5 |
| Lines | +337 / -7 |
| Commit | (pending) |

## Summary

前端 MML 模块 Phase 3 对接：模板 CRUD API/Hook/Mock、危险命令检测、任务结果分页、API 参数命名修正。

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `types/mml.ts` | 新增 MMLTemplate 接口 | +15 |
| `services/api/mmlApi.ts` | BackendMMLTemplate + mapBackendTemplate + 模板 API + 危险检测 + 任务结果 + pageSize→page_size 修正 | +151/-2 |
| `hooks/api/useMML.ts` | 5 个模板 Hook + useDangerousCheck | +66 |
| `mock/services/mmlService.ts` | 模板 Mock + 危险检测 Mock + getTaskResults | +105 |
| `services/api/adminApi.ts` | DictDetailListParams 新增 page/pageSize + 默认分页参数 | +7/-5 |

## Findings

### WARNING (1)

| # | File | Line(s) | Issue |
|---|------|---------|-------|
| W1 | `mmlApi.ts` | mapBackendTemplate | `parameters` 类型断言为 `Record<string, string \| number \| boolean>`，后端实际返回 `unknown` 值（如嵌套对象）时可能不安全 |

### INFO (1)

| # | File | Line(s) | Issue |
|---|------|---------|-------|
| I1 | `adminApi.ts` | getDictionaryDetailList | 分页默认值 page_size=100 可能对大字典数据量过多，但符合现有使用场景 |

### Positive Observations

- 完整的 mock/real API 双实现，与 createApiSwitch 模式一致
- Backend 类型 → mapBackendXxx 转换 → 前端类型，类型安全链路完整
- pageSize → page_size 修正解决前后端字段映射不一致问题
- Hook 遵循 invalidateQueries 缓存失效模式
- useDangerousCheck 带 staleTime 避免重复请求

## Conclusion

**PASS_WITH_WARNINGS** — 无 CRITICAL 问题，类型断言 WARNING 不影响运行时。

## Checklist

- [x] TypeScript tsc --noEmit passes
- [x] No `any` types used
- [x] Backend→Frontend type mapping complete
- [x] Mock/Real API interfaces match
- [x] Query key hierarchy follows conventions
- [x] No hardcoded secrets or credentials
