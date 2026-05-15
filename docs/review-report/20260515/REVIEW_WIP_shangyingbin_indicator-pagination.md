---
date: 2026-05-15
author: shangyingbin
scope: pm (indicator) + frontend (kpi-library)
type: fix
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD)
---

# KPI 指标库列表分页失效修复（ENB 实际 1409 条但只显示 20 条）

## 变更范围

| 文件 | 变更 |
|------|------|
| `omcgo/internal/pm/indicator/rest_handler.go` | `ListIndicators` handler 从 query 读 `page` / `page_size`（兼容 axios snake_case 与 camelCase 兜底）覆盖 `DefaultListRequest()` 默认 PageSize=20；加 `strconv` import；使用 `strconv.Atoi + n>0` 防御负数和非数字 |
| `omcmb/webcode/src/pages/product/kpi-library/IndicatorTab.tsx` | 加 `page` / `pageSize` state 传给 `useIndicatorList`；`<Table pagination>` 接 `current` / `pageSize` / `total` / `onChange` / `showTotal` / `showSizeChanger`；keyword 改变时 page 重置 1 |

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项

### INFO

1. 后端只改了 `ListIndicators` 一处。其他 list 端点（`ListGroups` / `ListUnits` / 启用指标等）若也有大数据量，仍硬用 `DefaultListRequest()` 默认 PageSize=20，需要时再补。
2. 前端 IndicatorTab 是用户最常访问的页面，其他 tab（EnabledIndicatorsTab / IndicatorUnitsTab）若也有 1k+ 数据，需同样改造为服务端分页。

## 关键校验

- 后端 page/page_size 解析：`strconv.Atoi + n>0` — 负数和非数字静默回退默认值，不抛 400（与 keyword 等可选参数处理风格一致）
- snake_case 优先（axios 自动转换后的格式）+ camelCase 兜底（直接调用 API 的兼容性）
- 前端 keyword 改变时 page 重置 1 — 防止搜索后 page 超出新结果集页数

## 验证

- `go build ./...` ✅
- `go test ./internal/pm/indicator/...` ✅ ok 0.682s
- `npm run typecheck` ✅
- Docker 重建 ✅
- 浏览器联调（Playwright）：
  - ENB Tab：50 条/页、分页器显示 1-5 页、`共 1409 条`
  - 翻第 3 页：换了一批 IDs（C000060176/C000190041/C000170065）
  - 切 GSM Tab：50 行（73 条只够一页）、`共 73 条`、IDs CGSM0040007/CGSM0050003/CGSM0050002（GSM 真实数据）

## 结论

PASS — 无阻塞项，可合入。
