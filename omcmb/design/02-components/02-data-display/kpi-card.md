# KPICard KPI 指标卡

## 概述
Dashboard 上显示关键指标的卡片，包含指标名、数值、趋势。

## 结构
```
┌──────────────────────┐
│ 📊 设备总数            │
│                       │
│     1,284    ↑ +12    │
│              较昨日     │
│ [今日/本周] 趋势图      │
└──────────────────────┘
```

## 元素
| 元素 | 字号 | 字重 | 颜色 |
|------|------|------|------|
| 图标 | icon-lg (24px) | — | `primary-600` |
| 指标名 | font-base (14px) | medium | `neutral-600` |
| 数值 | font-2xl~3xl (24-32px) | bold | `neutral-800` |
| 趋势箭头 | font-sm (13px) | medium | 上升 `status-success` / 下降 `status-error` |
| 对比文字 | font-xs (12px) | regular | `neutral-500` |
| 趋势迷你图 | 高 32px | — | `primary-600` 填充 |

## OMC 使用场景
- Dashboard 第一行: 设备总数 / 小区总数 / EU 总数 / RU 总数
