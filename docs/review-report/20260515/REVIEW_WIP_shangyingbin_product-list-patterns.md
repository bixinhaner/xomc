---
date: 2026-05-15
author: shangyingbin
scope: product (backend) + frontend (webcode + frontend-core)
type: feat
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD)
---

# 产品列表展示正则规则

## 变更范围

| 文件 | 变更 |
|------|------|
| `omcgo/internal/product/pg_handler_repo.go` | 新增 `ListPatternsAllByProduct() (map[uuid.UUID][]string, error)`，按 sort_order 升序拉全表 active patterns 分组 |
| `omcgo/internal/product/handler.go` | `productView` 加 `Patterns []string`；`toProductView(p, deviceCount, patterns)` 签名加第三参（nil 归一为空数组）；List handler 调 `ListPatternsAllByProduct`（与 CountDevicesByProduct 同模式，失败非致命 Warn）；Get / Create / Update / Match handler 同步更新调用 |
| `omcmb/frontend-core/src/types/product.ts` | `Product` 加 `patterns: string[]` |
| `omcmb/frontend-core/src/services/api/productApi.ts` | `BackendProduct.patterns?` + `mapBackendProduct` 默认 `[]` |
| `omcmb/frontend-core/src/mock/data/product.ts` | 两个 mock 产品加示例 patterns |
| `omcmb/frontend-core/src/mock/services/productService.ts` | `create` mock 把 input.patterns 同步到 product.patterns |
| `omcmb/webcode/src/pages/product/products/index.tsx` | 加"正则规则"列：最多 3 个 Tag（geekblue），超出折叠 `+N`（Tooltip 显示完整列表），空时显示 `—` |

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项

### INFO

1. Update handler 在 update 后再查一次 patterns 以填充返回视图。与 List/Get 路径保持视图一致，避免前端读到不含 patterns 字段的旧数据；小代价，无需优化。
2. 后端单测未补：repo 新方法 + handler 三处 view 调用都直链 DB，与现有 `CountDevicesByProduct` 路径无单测的状态一致。

## 关键校验

- `productView.Patterns` 用 `json:"patterns"`（非 omitempty），nil 在 `toProductView` 归一为空数组，确保前端 `bp.patterns ?? []` 永远拿到数组
- `ListPatternsAllByProduct` 只筛 `is_active = TRUE`，与 Get 视图的 `ListPatternsByProduct` 一致（Get 用 IsActive 字段过滤），语义对齐
- 前端 mock data 同步更新避免 `useMock=true` 模式下类型不全错

## 验证

- `go build ./...` ✅
- `go test ./internal/product/...` ✅ ok 0.781s
- `npm run typecheck` ✅
- Docker 重建 ✅
- 浏览器联调（Playwright）：
  - 列表 15 行"正则规则"列全部正确填充：BAIBLQ 单 pattern 原文 / MLN 系列三个 Tag / QRTB 四个折叠 `+1` / 华为 TCELL 六个折叠 `+3`
  - 新建带 2 个 patterns 的 TEST-LIST-PATTERNS 产品后，列表立即显示 `LIST-PAT-X` + `LIST-PAT-Y` 两个 Tag（React Query invalidation 自动刷新）
  - 删除测试产品后 DB 清理干净

## 结论

PASS — 无阻塞项，可合入。
