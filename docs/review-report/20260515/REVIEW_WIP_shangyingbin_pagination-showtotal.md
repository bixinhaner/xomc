---
date: 2026-05-15
author: shangyingbin
scope: frontend (product center pages)
type: style
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD 审计)
---

# 产品中心 5 个分页表格统一显示总数

## 背景

接 `ecc7984f` (KPI 库分页修复) 之后审计产品中心其他子页：

| 子页 | 分页配置 | 实测数据 | 之前 UX |
|------|---------|---------|---------|
| 产品管理 | `pagination={{ pageSize: 20, showSizeChanger: true }}` | 15 条 | 不显示总数 |
| 参数模型 / 模型清单 | `pagination={{ pageSize: 20 }}` | 9 条 | 不显示总数 |
| 参数模型 / 默认映射 | `pagination={{ pageSize: 50, showSizeChanger: true }}` | 827 条 | 不显示总数 |
| 参数模型 / 标准参数树 | `pagination={{ pageSize: 50, showSizeChanger: true }}` | 2001 条 | 不显示总数 |
| 告警库 | 已正确接 API total 与 onChange | 442 条 | 不显示总数 |

功能上分页都正常工作（前端 `dataSource.length` 自动当 total），只是缺统一的"共 N 条"提示。

## 变更范围

5 个文件各加一行 `showTotal: (t) => \`共 ${t} 条\`` 到 pagination 配置：
- `omcmb/webcode/src/pages/product/products/index.tsx`
- `omcmb/webcode/src/pages/product/param-model/ModelsTab.tsx`
- `omcmb/webcode/src/pages/product/param-model/MappingsTab.tsx`
- `omcmb/webcode/src/pages/product/param-model/StandardParamsTab.tsx`
- `omcmb/webcode/src/pages/product/alarm-library/index.tsx`

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项
### INFO — 0 项

5 处重复 `showTotal: (t) => \`共 ${t} 条\`` 函数字面量，但 4 字节回调不值得抽公共常量。

## 验证

- `npm run typecheck` ✅
- Docker 重建 ✅
- 浏览器实测 5 个页面：
  - 产品管理 "共 15 条" ✅
  - 参数模型清单 "共 9 条" ✅
  - 标准参数树 "共 2000 条" ✅
  - 默认映射 BaiBNQ "共 833 条" ✅
  - 告警库 "共 442 条" ✅

加上之前已有的 KPI 指标库 + 孤儿设备，**产品中心 7 个分页表格全部统一显示总数**。

## 结论

PASS — 纯 UI 一致性增强。
