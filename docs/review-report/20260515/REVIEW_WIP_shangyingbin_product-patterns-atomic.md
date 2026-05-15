---
date: 2026-05-15
author: shangyingbin
scope: product (backend) + frontend (webcode + frontend-core)
type: fix
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD)
---

# 产品新增表单"正则模式"tab 解锁 + 后端原子保存 patterns

## 变更范围

| 文件 | 变更 |
|------|------|
| `omcgo/internal/product/handler.go` | `createProductReq` 加 `Patterns []string`，透传到 `CreateProductInput.Patterns` |
| `omcgo/internal/product/pg_handler_repo.go` | `CreateProductInput` 加 `Patterns` 字段；`CreateProduct` 改为事务，事务内先 INSERT products，再 batch INSERT product_class_patterns（sort_order 取全局 max+1 起递增），失败回滚；事务前 pre-clean（trim + dedup + 去空） |
| `omcmb/frontend-core/src/types/product.ts` | `CreateProductInput` 加 `patterns?: string[]` |
| `omcmb/frontend-core/src/services/api/productApi.ts` | `buildCreatePayload` 把非空 patterns 加入 body |
| `omcmb/frontend-core/src/mock/services/productService.ts` | mock `create` 把 input.patterns 同步初始化到 `patterns[newP.id]` |
| `omcmb/webcode/src/pages/product/products/ProductDrawer.tsx` | 新增本地 `pendingPatterns` state；正则模式 tab 移除 `disabled: !isEdit`；新增模式渲染独立"添加 / 列表 / 删除"UI（仅本地数组操作）；保存时把 pendingPatterns 传给 createMut；编辑模式不变 |

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项

### INFO

1. 后端未补单测：`CreateProduct` 现有事务路径无 handler 层单测；product 模块原本 Create 路径也无 handler-level test，覆盖度一致，不形成回退。
2. 前端编辑模式下的 Pattern UI 与新增模式逻辑分离（两条独立渲染分支），有少量重复但语义不同（编辑用 API，新增用本地数组），不强行抽公共组件。

## 验证

- `go build ./...` ✅
- `go test ./internal/product/...` ✅ ok 0.813s
- `npm run typecheck` ✅
- Docker 重建 ✅
- 浏览器联调（Playwright）：
  - 正则模式 tab 在新增模式下不再灰
  - 输入 3 个正则（TEST-Pattern-A/B/C）暂存到列表，序号 1/2/3 正确
  - 切回基本信息填好必填字段（产品名 TEST-PATTERNS-CREATE / LTE / ENB / BLQ / ENB）保存
  - DB 实测：products 1 行 + 3 个 patterns 同事务入库，sort_order 30/31/32 接全局 active max+1 起，is_active=true
  - 删除测试产品 → FK CASCADE 自动清理 patterns（products=0 / patterns=0）

## 结论

PASS — 无阻塞项，可合入。
