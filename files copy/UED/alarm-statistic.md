# 告警统计页面分析

> 来源文件：`OMCWebServer/src/main/webapp/WEB-INF/content/cell/fault/alarm_statistic_total.jsp`

---

## 1. 页面概述

告警统计页面通过点击告警视图页面右上角的"统计图表"按钮打开，以侧边栏(Slide)形式展示。该页面提供告警数据的可视化统计功能，包含两个堆叠柱状图，分别展示告警变化趋势和告警存量分布。

---

## 2. 页面结构

### 2.1 顶部控制区 (Header)

| 组件 | 功能 | 说明 |
|------|------|------|
| 日期选择器 | 日期/月份切换 | 支持"天"和"月"两种统计维度 |
| 左箭头按钮 | 时间递减 | 查看前一天/前一个月数据 |
| 右箭头按钮 | 时间递增 | 查看后一天/后一个月数据（当天/当月禁用） |
| 天/月切换按钮 | 维度切换 | `day` = 按小时统计，`month` = 按天统计 |
| 关闭按钮 | 关闭面板 | 关闭统计图表侧边栏 |

---

## 3. 统计图表区域

### 3.1 上部图表 - 告警变化趋势 (topBarChart)

**位置**: `#topBarChart`

**功能**: 统计告警的新增/清除趋势

**标签切换**:
| 选项 | 统计类型 | 说明 |
|------|----------|------|
| 新增 | `ActiveIncr` | 统计时间段内新增的告警数量 |
| 清除 | `ClearIncr` | 统计时间段内清除的告警数量 |

**图表类型**: 堆叠柱状图 (Stacked Bar Chart)

**数据维度**:
| 维度 | 天视图 | 月视图 |
|------|--------|--------|
| X轴 | 小时 (0-23) | 日期 (1-31) |
| Y轴 | 告警数量 | 告警数量 |

### 3.2 下部图表 - 告警存量分布 (footBarChart)

**位置**: `#footBarChart`

**功能**: 统计当前告警的存量分布

**标签切换**:
| 选项 | 统计类型 | 说明 |
|------|----------|------|
| 活动 | `ActiveTotal` | 统计当前活动告警数量（未清除） |
| 所有 | `Total` | 统计所有告警数量（包含已清除） |

**图表类型**: 堆叠柱状图 (Stacked Bar Chart)

**数据维度**:
| 维度 | 天视图 | 月视图 |
|------|--------|--------|
| X轴 | 小时 (1-24) | 日期 (1-31) |
| Y轴 | 告警数量 | 告警数量 |

---

## 4. 告警级别颜色

所有图表统一使用以下颜色标识告警级别：

| 级别 | 颜色代码 | 颜色说明 | 图标类 |
|------|----------|----------|--------|
| Critical | `#FC5959` | 红色 - 紧急告警 | `alarmCritical` |
| Major | `#FF973E` | 橙色 - 主要告警 | `alarmMajor` |
| Minor | `#FFDA41` | 黄色 - 次要告警 | `alarmMinor` |
| Warning | `#67DFF8` | 青色 - 警告告警 | `alarmWarning` |

---

## 5. API 接口

### 5.1 统一接口

**URL**: `/cell/fault/queryAlarmStatisticResult.action`

**Method**: POST

### 5.2 上部图表参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| templateId | string | 空字符串 |
| timeZone | string | 时区 |
| statisticObject | string | 固定值 `Serverity` (按级别统计) |
| statisticType | string | `ActiveIncr` 或 `ClearIncr` |
| statisticTime | string | `Day` 或 `Month` |
| queryStartTime | string | 查询开始时间 |
| queryEndTime | string | 查询结束时间 |

### 5.3 下部图表参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| templateId | string | 空字符串 |
| timeZone | string | 时区 |
| statisticObject | string | 固定值 `Serverity` (按级别统计) |
| statisticType | string | `ActiveTotal` 或 `Total` |
| statisticTime | string | `Day` 或 `Month` |
| queryStartTime | string | 查询开始时间 |
| queryEndTime | string | 查询结束时间 |

---

## 6. 数据结构

### 6.1 上部图表响应数据 (新增/清除)

```json
{
  "rows": [
    {
      "startTime": "2026-03-20 10:00:00",
      "critical": 5,
      "major": 10,
      "minor": 8,
      "warning": 3
    }
  ]
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| startTime | string | 统计时间段开始时间 |
| critical | number | 紧急告警数量 |
| major | number | 主要告警数量 |
| minor | number | 次要告警数量 |
| warning | number | 警告告警数量 |

### 6.2 下部图表响应数据 (活动/所有)

```json
{
  "rows": [
    {
      "endTime": "2026-03-20 10:00:00",
      "critical": 15,
      "major": 25,
      "minor": 18,
      "warning": 7
    }
  ]
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| endTime | string | 统计时间段结束时间 |
| critical | number | 紧急告警数量 |
| major | number | 主要告警数量 |
| minor | number | 次要告警数量 |
| warning | number | 警告告警数量 |

---

## 7. 图表配置

### 7.1 通用配置

```javascript
{
  tooltip: {
    trigger: 'axis',
    triggerOn: 'none',
    enterable: true,
    axisPointer: { type: 'line' }
  },
  legend: {
    data: ['Critical', 'Major', 'Minor', 'Warning'],
    bottom: 10,
    selectedMode: false,
    itemHeight: 8,
    itemWidth: 8,
    itemGap: 50
  },
  grid: {
    top: '12%',
    left: '3%',
    right: '6%',
    bottom: '10%',
    containLabel: true
  },
  yAxis: {
    name: 'Alarm Count',
    type: 'value'
  }
}
```

### 7.2 系列配置

```javascript
series: [
  { name: 'Critical', type: 'bar', stack: '程度', barWidth: 20, color: '#FC5959' },
  { name: 'Major', type: 'bar', stack: '程度', barWidth: 20, color: '#FF973E' },
  { name: 'Minor', type: 'bar', stack: '程度', barWidth: 20, color: '#FFDA41' },
  { name: 'Warning', type: 'bar', stack: '程度', barWidth: 20, color: '#67DFF8' }
]
```

---

## 8. 时间处理逻辑

### 8.1 天视图 (Day)

- 查询时间范围: `yyyy-MM-dd 00:00:00` ~ `yyyy-MM-dd+1 00:00:00`
- X轴显示: 24 个时间点 (0-23 或 1-24)
- 上部图表: 使用 `startTime` 定位数据
- 下部图表: 使用 `endTime` 定位数据

### 8.2 月视图 (Month)

- 查询时间范围: `yyyy-MM-01 00:00:00` ~ `yyyy-MM+1-01 00:00:00`
- X轴显示: 当月天数 (1-31)
- 日期解析: 从日期字符串中提取日期部分

### 8.3 日期限制

- 禁止选择未来日期
- 当前日期/月份时，右侧箭头按钮禁用

---

## 9. 交互功能

### 9.1 标签切换

| 图表 | 切换方法 | 触发事件 |
|------|----------|----------|
| 上部图表 | `ChartTopTitleClick(type)` | `topDataType` 变化，重新加载图表 |
| 下部图表 | `ChartFootTitleClick(type)` | `footDataType` 变化，重新加载图表 |

### 9.2 时间导航

| 操作 | 方法 | 说明 |
|------|------|------|
| 时间递减 | `timeReduce()` | 天视图减1天，月视图减1月 |
| 时间递增 | `timeAdd()` | 天视图加1天，月视图加1月 |
| 维度切换 | `dayAndMonthClick(type)` | 切换天/月视图并重置时间 |

### 9.3 图表交互

- 鼠标悬停显示 tooltip（总数 + 各级别数量）
- 图表自适应窗口大小
- 关闭面板: `alarmViewVue.$refs.sharingSlide.hide()`

---

## 10. 统计类型汇总

| 统计类型 | 英文标识 | 图表位置 | 说明 |
|----------|----------|----------|------|
| 新增告警 | `ActiveIncr` | 上部图表 | 统计时间段内新产生的告警 |
| 清除告警 | `ClearIncr` | 上部图表 | 统计时间段内被清除的告警 |
| 活动告警 | `ActiveTotal` | 下部图表 | 当前未清除的告警存量 |
| 所有告警 | `Total` | 下部图表 | 包含已清除的全部告警 |

---

## 11. 关联文件

| 文件 | 说明 |
|------|------|
| `alarm_statistic_total.jsp` | 告警统计页面（总量统计） |
| `alarm_statistic.jsp` | 告警统计页面（模板统计） |
| `view.jsp` | 告警视图主页面（调用方） |
| `FaultAction.java` | 后端 Action 控制器 |

---

## 12. 页面差异对比

| 特性 | alarm_statistic_total.jsp | alarm_statistic.jsp |
|------|---------------------------|---------------------|
| 上部图表标签 | 新增/清除 | 活动告警 |
| 下部图表标签 | 活动/所有 | 告警列表 |
| Top 10 排行 | 无 | 有 |
| 告警列表 | 无 | 有 |
| 导出功能 | 无 | 有 |
| 模板关联 | 无 | 需要模板ID |
| 设备类型筛选 | 无 | 有 |
