# 系统日志查询 System Log Query

> 模块路径：`/log/query` Tab 4
> 布局模板：同日志查询页通用结构
> 数据来源：广研院 日志查询页 — 系统日志 Tab

---

## 1. 页面概述

系统日志记录网管系统自身的运行日志，包括各模块的运行状态、错误信息、异常堆栈等。主要面向系统管理员和开发人员进行问题排查。共享日志查询页的 4-Tab 结构。

---

## 2. 筛选栏

| 筛选项 | 组件 | 说明 |
|--------|------|------|
| 模块 | Select | 系统模块（ACS-COMMUNICATION、DEVICE-SERVICE、ALARM-SERVICE 等） |
| 日志级别 | Select | DEBUG/INFO/WARN/ERROR/FATAL |
| 消息内容 | Input | 关键词搜索 |

---

## 3. 表格列定义

| 列名 | 字段 | 宽度 | 说明 |
|------|------|------|------|
| 时间 | `timestamp` | 180px | `YYYY-MM-DD HH:mm:ss.SSS` 毫秒级 |
| 模块 | `module` | 180px | 系统模块名，等宽字体 |
| 日志级别 | `level` | 80px | Tag + 颜色 |
| 消息内容 | `message` | 400px | 截断 + Tooltip |
| 详情 | — | 60px | 图标按钮（展开详情） |

### 日志级别颜色

| 级别 | Tag 颜色 | 说明 |
|------|---------|------|
| DEBUG | `default` (灰) | 调试信息 |
| INFO | `blue` | 正常运行信息 |
| WARN | `orange` | 警告 |
| ERROR | `red` | 错误 |
| FATAL | `red` + 加粗 | 致命错误 |

---

## 4. 消息内容弹窗

> 点击详情图标打开
> 宽度 800px

```
┌─────────────────────────────────────────────────┐
│ 日志详情                                          │
├─────────────────────────────────────────────────┤
│                                                 │
│ 时间：2024-01-15 14:30:25.123                    │
│ 模块：ACS-COMMUNICATION                          │
│ 级别：ERROR                                      │
│                                                 │
│ ┌─ 消息内容 ──────────────────────────────────┐ │
│ │ java.lang.NullPointerException:             │ │
│ │   Cannot invoke method on null object       │ │
│ │ at com.omc.acs.service.DeviceService        │ │
│ │   .processHeartbeat(DeviceService.java:156) │ │
│ │ at com.omc.acs.handler.TR069Handler         │ │
│ │   .handleInform(TR069Handler.java:89)       │ │
│ │ at com.omc.acs.handler.TR069Handler         │ │
│ │   .processRequest(TR069Handler.java:45)     │ │
│ │ at sun.reflect.NativeMethodAccessorImpl      │ │
│ │   .invoke0(Native Method)                   │ │
│ │ at org.springframework.web.servlet           │ │
│ │   .FrameworkServlet.service(FS.java:897)    │ │
│ │ ...                                         │ │
│ └─────────────────────────────────────────────┘ │
│                                                 │
│                 [复制]    [关闭]                  │
└─────────────────────────────────────────────────┘
```

### 弹窗规格

| 属性 | 值 |
|------|-----|
| 宽度 | 800px |
| 内容区最大高度 | 600px（超出滚动） |
| 字体 | 等宽字体 `SFMono-Regular` / `Consolas` 12px |
| 行号 | 左侧行号 |
| 高亮 | Java 堆栈追踪语法高亮（类名蓝色、行号绿色） |
| 复制 | 复制完整日志文本 |
| 搜索 | 支持 Ctrl+F 文本内搜索 |

---

## 5. 日志统计视图

> 对应日志统计页 `/log/statistics` Tab 4

### 5.1 筛选

| 筛选项 | 组件 |
|--------|------|
| 统计粒度 | Toggle：按小时 / 按天 |
| 模块 | Select |
| 日志级别 | Select |

### 5.2 图表 — 柱状+折线组合图

| 属性 | 值 |
|------|-----|
| 类型 | 组合图（ECharts `bar` + `line`） |
| 主图（柱状） | ERROR 级别日志数量（红色柱体） |
| 副图（折线） | 全部日志总量（灰色折线） |
| X 轴 | 时间（小时/天） |
| Y 轴 | 日志数量（左轴 ERROR，右轴 总量） |
| 特征 | 可观察到时间规律（如凌晨日志量低，白天高） |

---

## 6. 模块列表

系统日志涉及的主要模块：

| 模块标识 | 说明 |
|---------|------|
| `ACS-COMMUNICATION` | ACS 通信服务 |
| `DEVICE-SERVICE` | 设备管理服务 |
| `ALARM-SERVICE` | 告警处理服务 |
| `CONFIG-SERVICE` | 配置管理服务 |
| `PERF-SERVICE` | 性能采集服务 |
| `FILE-SERVICE` | 文件传输服务 |
| `AUTH-SERVICE` | 认证授权服务 |
| `SCHEDULER` | 任务调度器 |
| `GATEWAY` | API 网关 |
| `MQ-CONSUMER` | 消息队列消费者 |

---

## 7. 权限控制

| 操作 | 所需权限 |
|------|---------|
| 查看系统日志 | `log:system:view` |
| 查看日志详情 | `log:system:detail` |
| 导出 | `log:system:export` |
| 查看统计 | `log:system:statistics` |
