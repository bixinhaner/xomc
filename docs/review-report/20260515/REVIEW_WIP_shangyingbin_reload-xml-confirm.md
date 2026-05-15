---
date: 2026-05-15
author: shangyingbin
scope: frontend (webcode product pages)
type: fix
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD)
---

# 产品中心 4 页"重载 XML"按钮加 Popconfirm 确认 + 修长时间转圈

## 变更范围

| 文件 | 按钮文案 |
|------|---------|
| `omcmb/webcode/src/pages/product/products/index.tsx` | "重载 XML" |
| `omcmb/webcode/src/pages/product/param-model/index.tsx` | "XML 导入 / 重载" |
| `omcmb/webcode/src/pages/product/kpi-library/index.tsx` | "XML 导入 / 重载" |
| `omcmb/webcode/src/pages/product/alarm-library/index.tsx` | "重载 XML" |

每个页面统一改造：
- 移除外层 Tooltip
- 外层包 `<Popconfirm>` — 标题"确认重载 XML？"、各自专属影响描述（指标平台/告警网元类型/参数模型映射/告警定义等）、`okText="确认重载"`、`okButtonProps={{ danger: true }}`、`placement="bottomRight"`
- `onConfirm` 用**语句体**而非表达式（不返回 Promise）— 让弹层立即关闭，loading 反馈仅交给触发按钮
- 触发按钮加 `danger` 红色样式
- "刷新缓存"按钮**全部保持原样**（无副作用，加确认反而稀释警惕度 — cry-wolf 反模式）

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项

### INFO

1. 4 个 Popconfirm 块结构高度重复（标题、okText、placement、onConfirm 模式都相同），可考虑抽 `<ReloadXmlButton>` 公共组件。**暂不抽**：CLAUDE.md "三处相似才考虑提取"，且每个页面的 description 文案需要专属（覆盖字段不同），抽象后参数化反而拗口。后续若加第 5 处再考虑。
2. 与上一轮 commit `94369e28` / `63c74ad0` 同步：`message.success(...)` 在 antd v5 静态 API 下未实际渲染到 DOM（既有问题，与本次改造无关；属于 antd v5 `<App>` 包装层缺失，独立修复）。

## 关键校验

- onConfirm `() => { ... }` 是语句体不返回 Promise — Antd Popconfirm 因此立即关闭，不再卡在"确认按钮加载状态"
- "刷新缓存"无副作用按钮全部保留默认样式 + 直接调用模式
- 4 个文件 import 同步更新（Tooltip 改 Popconfirm；只有 products/index.tsx 因正则列还用 Tooltip 故保留）

## 验证

- `npm run typecheck` ✅
- Docker 重建 ✅
- Playwright 浏览器联调 — **4 页全部通过完整三路径测试**：

| 页面 | 弹层结构 | 取消关闭 | 确认 spinner / 后端实测 |
|------|---------|---------|----------------------|
| 产品管理 | ✅ | ✅ 无请求 | ~100ms / 100ms |
| 参数模型 | ✅ | ✅ 无请求 | 1228ms / 1184ms |
| KPI 指标库 | ✅ | ✅ 无请求 | 922ms / 865ms |
| 告警库 | ✅ | ✅ 无请求 | 204ms / 170ms |

前端 loading 时长与后端 API 真实耗时完全吻合（差值即网络 + React 渲染），无前端额外卡顿，弹层关闭与按钮 spinner 行为符合预期。

## 结论

PASS — 无阻塞项，可合入。
