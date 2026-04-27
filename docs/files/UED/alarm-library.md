# 告警库页面分析

> 基于 JSP 文件: `original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/cell/fault/view.jsp`
> 分析日期: 2026-03-20

## 页面概述

告警库页面是告警管理模块的一个 Tab 页签，用于展示告警定义信息。该页面允许用户查看告警定义的详细信息。

## 列表字段

| 序号 | 字段名称 | 字段属性 | 宽度 | 说明 |
|------|----------|----------|------|------|
| 1 | 信告警源 | `DEVICE_TYPE_NAME` | 110px | 告警来源设备类型 |
| 2 | 告警唯一标识 | `ALARM_IDENTIFIER` | 150px | 告警的唯一标识符 |
| 3 | 可能原因 | `ALARM_NAME` | 300px | 告警的可能原因描述 |
| 4 | 严重程度 | `SERVERITY_TYPE` | 140px | 告警级别 |
| 5 | 事件类型 | `EVENT_TYPE` | 160px | 告警事件分类 |
| 6 | 告警解释 | `EXPLANATION` | 700px | 告警详细解释说明 |

## 严重程度类型

| 级别代码 | 显示名称 | 颜色 |
|----------|----------|------|
| 31001 | 紧急告警 (Critical) | #E53935 (Material Red 600) |
| 31002 | 主要告警 (Major) | #FB8C00 (Material Orange 600) |
| 31003 | 次要告警 (Minor) | #FDD835 (Material Yellow 600) |
| 31004 | 警告告警 (Warning) | #42A5F5 (Material Blue 400) |

## 事件类型

| 代码 | 显示名称 |
|------|----------|
| 30000 | 通信告警 |
| 30001 | 服务质量告警 |
| 30002 | 处理失败告警 |
| 30003 | 设备告警 |
| 30004 | 环境告警 |
| 30006 | 性能溢出告警 |

## 页面功能

### 1. 查询功能
- **搜索框**: 支持模糊查询
- **查询字段**: 告警唯一标识 / 可能原因 / 信告警源

### 2. 导出功能
- 点击导出按钮可导出告警库数据

## 权限控制

- `alarmLibShow`: 控制告警库 Tab 是否显示

## API 接口

| 功能 | 接口地址 | 参数 |
|------|----------|------|
| 列表查询 | `/cell/fault/library/queryLibraryPageList.action` | libraryParams |
| 导出 | `/cell/fault/library/exportAlarmLib.action` | - |

## 前端实现建议

### TypeScript 类型定义

```typescript
// 告警库项
interface AlarmLibrary {
  deviceTypeName: string;    // 信告警源
  alarmIdentifier: string;   // 告警唯一标识
  alarmName: string;         // 可能原因
  serverityType: 'Critical' | 'Major' | 'Minor' | 'Warning';  // 严重程度
  eventType: EventType;      // 事件类型
  explanation: string;       // 告警解释
}

// 事件类型
type EventType = '30000' | '30001' | '30002' | '30003' | '30004' | '30006';

// 查询参数
interface AlarmLibraryParams {
  searchText?: string;       // 搜索文本
  page: number;
  pageSize: number;
}
```

### 严重程度颜色配置

```typescript
const SEVERITY_COLORS = {
  critical: '#E53935', // 紧急 - Material Red 600
  major: '#FB8C00',    // 重要 - Material Orange 600
  minor: '#FDD835',    // 次要 - Material Yellow 600
  warning: '#42A5F5',  // 警告 - Material Blue 400
};

const SEVERITY_CONFIG: Record<string, { color: string; bgColor: string; label: string }> = {
  Critical: { color: '#E53935', bgColor: '#FFEBEE', label: '紧急告警' },
  Major: { color: '#FB8C00', bgColor: '#FFF3E0', label: '主要告警' },
  Minor: { color: '#FDD835', bgColor: '#FFFDE7', label: '次要告警' },
  Warning: { color: '#42A5F5', bgColor: '#E3F2FD', label: '警告告警' },
};
```

### 列定义示例

```typescript
const columns: ColumnsType<AlarmLibrary> = [
  {
    title: t('alarm.deviceTypeName'),
    dataIndex: 'deviceTypeName',
    width: 110,
  },
  {
    title: t('alarm.alarmIdentifier'),
    dataIndex: 'alarmIdentifier',
    width: 150,
  },
  {
    title: t('alarm.alarmName'),
    dataIndex: 'alarmName',
    width: 300,
    ellipsis: true,
  },
  {
    title: t('alarm.severity'),
    dataIndex: 'serverityType',
    width: 140,
    render: (value) => {
      const config = SEVERITY_CONFIG[value];
      return (
        <Tag style={{ color: config?.color, backgroundColor: config?.bgColor, border: 'none' }}>
          {config?.label || value}
        </Tag>
      );
    },
  },
  {
    title: t('alarm.eventType'),
    dataIndex: 'eventType',
    width: 160,
    render: (value) => t(`alarm.eventType.${value}`),
  },
  {
    title: t('alarm.explanation'),
    dataIndex: 'explanation',
    width: 700,
    ellipsis: true,
  },
];
```
