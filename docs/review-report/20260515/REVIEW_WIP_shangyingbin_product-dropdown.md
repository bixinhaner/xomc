---
date: 2026-05-15
author: shangyingbin
scope: product (backend) + frontend (webcode)
type: fix
verdict: PASS_WITH_INFO
backlog: HOTFIX (源自外部 TODO.MD)
---

# 产品管理添加表单：指标平台名 / 告警网元类型 改为下拉框

## 变更范围

| 文件 | 变更 |
|------|------|
| `omcgo/internal/product/handler.go` | 新增 `ListIndicatorPlatforms` / `ListAlarmNeTypes` 端点；Create/Update 接口中 `indicator_platform` 改为"仅 ENB 必填"条件校验 |
| `omcgo/internal/product/pg_repository.go` | 新增 `ListIndicatorPlatforms(deviceType)` / `ListAlarmNeTypes()` repo 方法，返回 distinct + ORDER BY 的 `[]string` |
| `omcmb/frontend-core/src/services/api/productApi.ts` | 新增 `listIndicatorPlatforms` / `listAlarmNeTypes` |
| `omcmb/frontend-core/src/mock/services/productService.ts` | mock 同步形状 |
| `omcmb/frontend-core/src/hooks/api/useProducts.ts` | 新增 `useIndicatorPlatforms(deviceType)` / `useAlarmNeTypes()`，5 min staleTime |
| `omcmb/webcode/src/pages/product/products/ProductDrawer.tsx` | 指标平台名条件渲染 + 改 Select；告警网元类型改 Select |

## 审查发现

### CRITICAL — 0 项

### WARNING — 0 项

### INFO

1. **预先存在的大小写不一致**：`DEVICE_TYPE_OPTIONS` 用大写 `ENB/GNB/GSM`，XML loader 写库时小写 `enb/gnb/gsm`。本次不修复（已登记到外部 `TODO.MD`）。后端枚举接口已通过 `strings.ToLower` 做归一化兼容。
2. 新端点未补 `*_test.go`：与 product 模块现有 handler 测试覆盖度一致（既有 List/Create 也无 handler 层测试），不形成回退。

## 验证

- 后端 `go build ./...` ✅
- 后端 `go test ./internal/product/...` ✅ ok 0.941s
- 前端 `npm run typecheck` ✅
- Docker 重建后浏览器联调全部通过（9 个验证点）：
  - 初始无 deviceType → 指标平台名隐藏
  - 选 ENB → 字段出现 + 8 个真实值（ALL/BLQ/BLX/BM/ENB_DEFAULT_098/ENB_DEFAULT_181/MLN/MLQ）
  - 切 GNB / GSM → 字段消失
  - 切回 ENB → 字段重新出现
  - 告警网元类型下拉填充 7 个真实值（CPE/EGW/ENB/EPC/GNB/OMC/UPS）
  - 完整 ENB 产品创建+保存成功
  - 测试数据已清理

## 结论

PASS_WITH_INFO — 无阻塞项，可直接合入。
