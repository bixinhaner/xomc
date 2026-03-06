# 导航结构设计 (Navigation Structure)

## 页面用途

定义 OMC 统一网管系统的全局导航架构，采用两级导航体系，配合设备类型全局上下文选择器，为用户提供清晰、一致的模块访问路径。

## 导航层级架构

### 整体架构概览

```
+----------------------------------------------------------+
|  Header (48px)                                           |
+--------+-------------------------------------------------+
|        |                                                 |
| Sidebar|            主内容区域                              |
| (240px)|         (Main Content Area)                     |
|        |                                                 |
| [设备类型]|  +-------------------------------------------+  |
| 选择器  |  | Tab Bar (多标签工作区)                         |  |
|        |  +-------------------------------------------+  |
| 一级导航 |  |                                           |  |
| + 二级  |  |         页面内容                             |  |
|   子项  |  |                                           |  |
|        |  |                                           |  |
|        |  |                                           |  |
+--------+-------------------------------------------------+
```

### 第一级导航 (Tier 1) - 模块分组

暗色侧边栏中的模块组列表，每个模块组配有对应图标。

| 序号 | 模块名称     | 英文标识               | 图标建议              | 路由前缀           |
|------|------------|----------------------|---------------------|-------------------|
| 1    | Dashboard  | Dashboard            | `DashboardOutlined` | `/dashboard`      |
| 2    | 设备管理    | Device Management    | `ClusterOutlined`   | `/device`         |
| 3    | 告警管理    | Alarm Management     | `AlertOutlined`     | `/alarm`          |
| 4    | 基站监控    | Topology Monitoring  | `GlobalOutlined`    | `/topology`       |
| 5    | 配置管理    | Configuration Mgmt   | `SettingOutlined`   | `/config`         |
| 6    | 性能管理    | Performance Mgmt     | `LineChartOutlined` | `/performance`    |
| 7    | MR 管理    | MR Management        | `SignalFilled`      | `/mr`             |
| 8    | 报表管理    | Report Management    | `FileTextOutlined`  | `/report`         |
| 9    | MML 管理   | MML Management       | `CodeOutlined`      | `/mml`            |
| 10   | 备份恢复    | Backup & Restore     | `CloudServerOutlined`| `/backup`        |
| 11   | License 管理| License Management  | `SafetyOutlined`    | `/license`        |
| 12   | 软件版本    | Software Version     | `BuildOutlined`     | `/software`       |
| 13   | 文件管理    | File Management      | `FolderOutlined`    | `/file`           |
| 14   | 日志管理    | Log Management       | `AuditOutlined`     | `/log`            |
| 15   | 运维工具    | Ops Tools            | `ToolOutlined`      | `/ops`            |
| 16   | 系统管理    | System Management    | `ControlOutlined`   | `/system`         |

### 第二级导航 (Tier 2) - 子页面

每个一级模块展开后显示其子页面列表，作为侧边栏的子项。点击一级模块自动展开/折叠该模块的二级菜单。

**示例：告警管理展开后的二级导航**

```
  ▼ 告警管理
      当前告警
      历史告警
      告警统计
      告警规则
      告警支持库
      告警同步
```

**示例：设备管理展开后的二级导航**

```
  ▼ 设备管理
      设备列表
      设备注册
      设备分组
      网元管理
      在线监控
      开站管理
      交维管理
      资源统计
      导入导出
      回收站
```

### 设备类型全局上下文选择器 (Device Type Context Selector)

> **重要：设备类型选择器不是导航层级的一部分，而是全局数据过滤上下文。**

位于侧边栏顶部 Logo 区域下方，作为全局数据筛选器。切换设备类型后，所有模块的数据、统计、列表都会根据选定的设备类型进行过滤。

详见 `05-device-type-switching.md`。

## 导航交互规则

### 展开/折叠行为

| 交互          | 行为                                                     |
|--------------|----------------------------------------------------------|
| 点击一级模块   | 展开该模块的二级菜单，折叠其他已展开模块（手风琴模式）           |
| 点击二级子页面  | 高亮当前子页面，在主内容区打开对应页面，同时创建页面 Tab         |
| 侧边栏折叠状态  | 仅显示一级模块图标，hover 时弹出 Popover 显示二级菜单         |
| 键盘导航       | 支持上下键切换、Enter 展开/选中、Escape 折叠                 |

### 路由同步

- URL 路由与导航状态保持双向同步
- 直接访问 URL 时，自动展开对应的一级模块并高亮二级子页面
- 浏览器前进/后退按钮正确更新导航高亮状态

### 权限控制

- 导航项根据用户角色权限动态显示/隐藏
- 无权限的模块在侧边栏中不渲染
- 直接通过 URL 访问无权限页面时，跳转到 403 页面

## 导航状态管理

```typescript
interface NavigationState {
  activeModule: string;        // 当前一级模块 key
  activeSubPage: string;       // 当前二级子页面 key
  expandedModules: string[];   // 已展开的一级模块列表
  sidebarCollapsed: boolean;   // 侧边栏是否折叠
  deviceType: DeviceType;      // 全局设备类型上下文
}

type DeviceType = 'eNB' | 'gNB' | 'CPE' | 'eGW';
```

## 跨模块导航

| 来源             | 目标                | 触发方式                     |
|-----------------|--------------------|-----------------------------|
| Dashboard KPI 卡片 | 设备管理 > 设备列表  | 点击设备总数卡片               |
| Dashboard 告警摘要 | 告警管理 > 当前告警  | 点击告警等级区域               |
| Header 告警角标   | 告警管理 > 当前告警  | 点击角标，携带 severity 参数    |
| 设备详情 > 告警 Tab | 告警管理 > 当前告警 | 点击 "查看全部告警"            |
| 任意列表 > 设备名称 | 设备管理 > 设备详情  | 点击设备名称链接               |
