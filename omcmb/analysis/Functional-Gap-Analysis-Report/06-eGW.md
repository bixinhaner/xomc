# 06 — eGW (eGW 网关管理)

> **覆盖状态：❌ 完全缺失**

---

## 原系统二级菜单

| # | 二级菜单 | 原系统文件数 | 新系统对应 | 状态 |
|---|---------|------------|----------|------|
| 1 | Maintenance (维护) | 3 | — | ❌ |
| 2 | Monitor (监控) | 3 | — | ❌ |
| 3 | Registration (注册) | 3 | — | ❌ |
| 4 | Upgrade (升级) | 7 | — | ❌ |

---

## 6.1 Maintenance — 维护

| 页面 | 功能 |
|------|------|
| `eGW_config.jsp` | eGW 配置管理 |
| `eGW_reboot.jsp` | eGW 重启 |
| `eGW_configEdit.jsp` | 编辑 eGW 配置 |

---

## 6.2 Monitor — 监控

| 页面 | 功能 |
|------|------|
| `eGW_monitor.jsp` | eGW 性能监控面板 |
| `eGW_kpi.jsp` | eGW KPI 展示 |
| `eGW_status.jsp` | eGW 运行状态 |

---

## 6.3 Registration — 注册

| 页面 | 功能 |
|------|------|
| `eGW_regist.jsp` | 注册新 eGW |
| `eGW_regist_list.jsp` | 已注册 eGW 列表 |
| `eGW_regist_detail.jsp` | 注册详情 |

---

## 6.4 Upgrade — 升级

| 页面 | 功能 |
|------|------|
| `eGW_upgrade.jsp` | eGW 升级管理 |
| `eGW_upgradeAddTask.jsp` | 创建升级任务 |
| `eGW_upgradeImportFile.jsp` | 导入升级文件 |
| `eGW_upgradeFileEdit.jsp` | 编辑升级文件 |
| `eGW_upgradeTaskList.jsp` | 升级任务列表 |
| `eGW_upgradeTaskResult.jsp` | 升级结果 |
| `eGW_upgradeTaskProgress.jsp` | 升级进度 |

---

## 说明

新系统的 Device 模块支持 eGW 作为设备类型之一（productType 包含 eGW），但仅限于设备注册、列表查看和基本监控。原系统 eGW 的专用维护、KPI 监控、注册管理和升级管理全部缺失。
