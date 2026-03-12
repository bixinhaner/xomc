# 05 — CPE (CPE 管理)

> **覆盖状态：⚠️ 大量缺失**

---

## 原系统二级菜单

| # | 二级菜单 | 原系统文件数 | 新系统对应 | 状态 |
|---|---------|------------|----------|------|
| 1 | Monitor (CPE监控) | 14 | Device → OnlineMonitoring | ⚠️ 大量缺失 |
| 2 | Monitor SettingPage (监控设置) | 7 | — | ❌ 缺失 |
| 3 | Upgrade (CPE升级) | 5 | Software → UpgradePlan | ⚠️ |
| 4 | Maintenance (维护) | 2 | Log → NEMessageLog | ⚠️ |
| 5 | BatchConfig (批量配置) | 3 | Config → BatchParamTemplate | ⚠️ |
| 6 | Certificate (证书) | 1 | — | ❌ 缺失 |
| 7 | DataModel (数据模型) | 1 | — | ❌ 缺失 |
| 8 | FactoryReset (工厂复位) | 2 | — | ❌ 缺失 |
| 9 | Strategy/LockPCI (PCI锁定策略) | 2 | — | ❌ 缺失 |

---

## 5.1 Monitor — CPE 监控

### 原系统 `cpe/monitor/` (14 文件)

| 页面 | 功能 | 新系统 | 状态 |
|------|------|--------|------|
| `cpe_monitor.jsp` | CPE 监控主面板 | Device → OnlineMonitoring | ⚠️ 通用监控 |
| `cpe_monitor_vue.jsp` | Vue 版 CPE 监控 | — | ❌ |
| `cpeTopo_tab.jsp` | CPE 拓扑视图 | Topology → Canvas | ⚠️ |
| `cpe_query.jsp` | CPE 设备查询 | Device → DeviceList | ⚠️ |
| `cpe_scan.jsp` | 网络扫描发现 CPE | — | ❌ |
| `cpe_info_chart.jsp` | CPE 信息图表 | — | ❌ |
| `cpe_diagnosetics.jsp` | CPE 诊断工具 | — | ❌ |
| `export_config.jsp` | 导出 CPE 配置 | Device → ImportExport | ⚠️ |
| `setting_config.jsp` | CPE 配置设置 | — | ❌ |
| `setting_apn.jsp` | CPE APN 设置 | — | ❌ |
| `lock_frequency.jsp` | 频率锁定 | — | ❌ |

**缺失的关键功能：**
- ❌ CPE 网络扫描发现
- ❌ CPE 专用诊断工具
- ❌ CPE APN 设置
- ❌ CPE 频率锁定
- ❌ CPE 信号信息图表

---

## 5.2 Upgrade — CPE 升级

### 原系统 `cpe/upgrade/` (5 文件)

| 页面 | 功能 | 新系统 | 状态 |
|------|------|--------|------|
| `cpeUpgradePage.jsp` | CPE 升级管理 | Software → UpgradePlan | ⚠️ 通用 |
| `cpeUpgradeAddTask.jsp` | 创建升级任务 | — | ❌ 缺CPE专用 |
| `imageImportFile.jsp` | 导入固件 | Software → FirmwareUpload | ⚠️ |
| `moduleImportFile.jsp` | 导入模块 | — | ❌ |
| `middleImportFile.jsp` | 导入中间件 | — | ❌ |

---

## 5.3 Maintenance — 维护

| 页面 | 功能 | 新系统 | 状态 |
|------|------|--------|------|
| `cpeLogAdd.jsp` | CPE 日志采集 | File → LogRetrieval | ⚠️ |
| `accessLog.jsp` | 接入日志 | Log → NEMessageLog | ⚠️ |

---

## 5.4 BatchConfig — 批量配置

| 页面 | 功能 | 新系统 | 状态 |
|------|------|--------|------|
| 批量配置管理(3文件) | CPE 批量参数下发 | Config → BatchParamTemplate | ⚠️ 通用 |

---

## 5.5~5.9 其他子模块（全部 ❌ 缺失）

| 二级菜单 | 功能 |
|---------|------|
| Certificate | CPE SSL/TLS 证书管理 |
| DataModel | CPE 参数定义 |
| FactoryReset | CPE 工厂复位(创建任务/任务列表) |
| Strategy/LockPCI | PCI 频率锁定策略配置 |

---

## 缺失操作流程汇总

| 流程 | 状态 |
|------|------|
| CPE 网络扫描发现 | ❌ |
| CPE 专用诊断 | ❌ |
| CPE APN 配置 | ❌ |
| CPE 频率锁定 | ❌ |
| CPE 工厂复位 | ❌ |
| CPE 证书管理 | ❌ |
| CPE 模块/中间件升级 | ❌ |
| CPE PCI 锁定策略 | ❌ |
