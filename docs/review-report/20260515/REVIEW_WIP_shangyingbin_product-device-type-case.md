---
date: 2026-05-15
author: shangyingbin
scope: product (backend) + migration + frontend (webcode)
type: fix
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD)
---

# 修复 `products.indicator_device_type` 大小写不一致

## 变更范围

| 文件 | 变更 |
|------|------|
| `omcgo/internal/product/handler.go` | Create / Update 入库前对 `IndicatorDeviceType` 做 `ToLower+TrimSpace` 归一化；ENB 必填校验改为直接 `== "enb"`（归一化后等价于原 `EqualFold`） |
| `omcgo/migrations/000105_products_indicator_device_type_check.sql` | 新增 CHECK 约束 `indicator_device_type IN ('enb','gsm','gnb')` |
| `omcmb/webcode/src/pages/product/products/ProductDrawer.tsx` | `DEVICE_TYPE_OPTIONS` value `ENB/GNB/GSM` → `enb/gnb/gsm`（label 保留大写） |
| `omcmb/webcode/src/pages/product/products/index.tsx` | 列表"指标设备类型"Tag 渲染加 `?.toUpperCase()` 显示统一大写 |

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项
### INFO — 0 项

## 关键校验

- migration `StatementBegin/End` 正确包裹 DO 块（合 CLAUDE.md §5.5.1）
- migration 幂等（先查 `pg_constraint`）
- migration Down 配对完整
- migration 版本号 000105 是 000104 之后下一个（无跳号、无重复）
- 后端归一化在 Create handler 的 binding 之后、`in := CreateProductInput{...}` 构造之前，确保后续所有逻辑路径都看到小写值
- 前端 ProductDrawer 现有 `isENB`/`onChange` 已用 `.toUpperCase()` 比较，本次无需改

## 验证

- `go build ./...` ✅
- `go test ./internal/product/...` ✅ ok 0.822s
- `npm run typecheck` ✅
- Docker 重建 + migration 000105 应用 ✅（`docker exec ... psql` 验证 `chk_products_indicator_device_type` 存在）
- 浏览器联调：
  - 列表 15 行"指标设备类型"列统一大写 ENB/GSM/GNB
  - 编辑既有产品 BAIBLQ：抽屉预选 "ENB (LTE)" / "BLQ" / "ENB"（之前是空白）
  - UI 新建 ENB 产品入库后 DB 实测为小写 `enb`
  - 测试数据已清理
  - 用户硬刷新浏览器后确认列表显示一致

## 结论

PASS — 无阻塞项，可合入。
