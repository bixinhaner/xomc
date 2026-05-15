---
date: 2026-05-15
author: shangyingbin
scope: pm (indicator)
type: fix
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD)
---

# KPI 指标库列表分组列显示组名而非 UUID

## 变更范围

| 文件 | 变更 |
|------|------|
| `omcgo/internal/pm/indicator/pg_indicator_repository.go` | `List` 与 `ListAll` 查询都加 `LEFT JOIN <group_table> g ON g.id = i.group_id`（表名来自 `dt.GroupTable()` 白名单生成）+ selectCols 加 `COALESCE(g.cn_name, g.en_name, '') AS group_name` + `scanIndicatorListItem` 两个分支扫描该列 |
| `omcgo/internal/pm/indicator/model.go` | `IndicatorListItem` 加 `GroupName string \`json:"group_name,omitempty"\`` |

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项

### INFO

1. 未补单测：indicator List 路径既有测试覆盖度较薄，本次新加列只影响序列化输出（不改业务逻辑），与既有现状一致。
2. 前端 mapping (`groupName: b.group_name`) 与列渲染 (`v || row.groupId || '—'`) 早就就位，本次后端补字段后自然 ENABLE，无需前端改动。

## 关键校验

- LEFT JOIN 表名来自 `dt.GroupTable()` — 通过 `ParseDeviceType` 白名单验证（ENB/GSM/GNB），无 SQL 注入风险
- COALESCE 顺序：`cn_name` → `en_name` → 空串，与运营商习惯一致（中文标签优先）
- scan 顺序：HasProductTypes (ENB/GSM) 分支与 GNB 分支都正确追加 `&item.GroupName`，与 select 列顺序一一对应

## 验证

- `go build ./...` ✅
- `go test ./internal/pm/indicator/...` ✅ ok 0.814s
- Docker 重建 ✅
- 浏览器联调（Playwright）：
  - ENB Tab "分组"列显示真实组名（DRB / ERAB / MR / IRATHO / HO / ENDC MN）—— 不再是 UUID
  - GNB / GSM Tab 同样显示组名
  - 表头/其他列无回退

## 旁支发现（不在本次范围，已登记 TODO.MD）

- GNB/GSM Tab 显示的指标数据与 ENB Tab 几乎一致（同一批 indicator IDs）——疑似 seed 数据三表内容重复或 tab 切换未按 deviceType 过滤。需独立排查。

## 结论

PASS — 无阻塞项，可合入。
