# 系统 UI 架构 (System UI Architecture)

## 应用外壳 (App Shell)

```
┌──────────────────────────────────────────────────────────────┐
│  Header (48px)                                                │
│  [Logo] [系统名称]              [紧急:12][主要:45][次要:89]    │
│                                 [警告:234] [🔔] [UTC] [admin▾]│
├─────────┬────────────────────────────────────────────────────┤
│ Sidebar │  Secondary Bar (40px, 可选)                         │
│ (240px) │  面包屑 / 页面标签栏                                 │
│         ├────────────────────────────────────────────────────┤
│ 深色背景 │                                                    │
│ #001529 │  Content Area (流式宽度)                             │
│         │  padding: 24px 水平, 16px 垂直                      │
│ [设备类型]│  背景: #FAFAFA                                     │
│ [模块菜单]│                                                    │
│  ▶ 子项  │                                                    │
│         ├────────────────────────────────────────────────────┤
│ [v1.0]  │  Task Panel (40/240px, 可选, 全局持久)              │
└─────────┴────────────────────────────────────────────────────┘
```

## 导航层级

### 一级导航（侧边栏模块组）

```
📊 Dashboard（首页）
📱 设备管理
🔔 告警管理
🗺️ 基站监控与拓扑
⚙️ 配置管理
📈 性能管理
📡 MR 管理
📋 报表管理
💻 MML 管理
💾 备份恢复
🔑 License 管理
📦 软件版本管理
📁 文件管理
📝 日志管理
🔧 运维工具
🛠️ 系统管理
```

### 二级导航（侧边栏子项展开）

每个模块展开后显示其子页面，例如：
- 告警管理 → 当前告警 / 历史告警 / ���警统计 / 告警规则 / 告警支持库 / 告警同步

### 三级导航（页面内 Tab 切换）

部分页面内部使用 Tab 切换，例如：
- 性能文件 → 参数配置 Tab / 文件列表 Tab

## 全局状态管理

| 状态 | 类型 | 来源 | 影响范围 |
|------|------|------|---------|
| 当前设备类型 | eNB/gNB/CPE/eGW | 侧边栏顶部选择器 | 全局数据过滤 |
| 告警等级计数 | 4 个等级各自计数 | WebSocket 实时���送 | Header 告警徽标 |
| 活跃任务数 | 数字 | WebSocket 实时推送 | 底部任务面板徽标 |
| 用户会话 | 用户名/角色/Token | 登录接口 | 权限控制 |
| 当前时区 | UTC/本地 | Header 时区选择器 | 所有时间显示 |
| 侧边栏状态 | 展开/收起 | 用户操作，本地存储 | 布局宽度 |

## 路由设计

| 模块 | 路由前缀 | 子路由示例 |
|------|---------|-----------|
| Dashboard | `/dashboard` | — |
| 设备管理 | `/device` | `/device/list`, `/device/group`, `/device/ne`, `/device/monitor` |
| 告警管理 | `/alarm` | `/alarm/current`, `/alarm/history`, `/alarm/statistics`, `/alarm/rules` |
| 基站监控 | `/topology` | `/topology/map`, `/topology/canvas`, `/topology/domain` |
| 配置管理 | `/config` | `/config/param-sync`, `/config/cell`, `/config/baseline`, `/config/common` |
| 性能管理 | `/performance` | `/performance/report`, `/performance/extraction`, `/performance/threshold` |
| MR 管理 | `/mr` | `/mr/indicators`, `/mr/reports`, `/mr/tasks`, `/mr/files` |
| 报表管理 | `/report` | `/report/standard`, `/report/station`, `/report/poll` |
| MML 管理 | `/mml` | `/mml/console`, `/mml/scripts` |
| 备份恢复 | `/backup` | `/backup/overview`, `/backup/create` |
| License | `/license` | `/license/list` |
| 软件版本 | `/software` | `/software/version`, `/software/upgrade`, `/software/firmware` |
| 文件管理 | `/file` | `/file/config`, `/file/log`, `/file/script`, `/file/user` |
| 日志管理 | `/log` | `/log/message`, `/log/heartbeat`, `/log/operation`, `/log/system` |
| 运维工具 | `/ops` | `/ops/template`, `/ops/instruction`, `/ops/task` |
| 系统管理 | `/system` | `/system/classification`, `/system/user` |

## 实时通信

| 通道 | 协议 | 数据 | 刷新频率 |
|------|------|------|---------|
| 告警推送 | WebSocket | 新增/变更告警 | 实时 |
| 任务状态 | WebSocket | 任务进度/完成/失败 | 实时 |
| Dashboard KPI | WebSocket | 设备/告警计数 | 5s |
| 设备状态 | Polling | 在线/离线状态 | 30s |
| 其他 CRUD | REST API | 按需请求 | — |
