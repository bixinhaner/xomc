# 04 — gNodeB (gNodeB 管理)

> **覆盖状态：⚠️ 部分覆盖**

---

## 原系统二级菜单

| # | 二级菜单 | 原系统文件数 | 新系统对应 | 状态 |
|---|---------|------------|----------|------|
| 1 | Monitor (监控) | 11 | Device → DeviceList/OnlineMonitoring | ⚠️ |
| 2 | Monitor SettingPage (监控设置) | 16 | — | ❌ 缺失 |
| 3 | Topology (拓扑) | 1 | Topology → Canvas | ⚠️ |
| 4 | Maintenance (维护) | 16 | MML + Log (部分) | ⚠️ |
| 5 | Backup & Restore (备份恢复) | 2 | Backup → BackupTasks | ⚠️ |
| 6 | Upgrade (升级) | 5 | Software → UpgradePlan | ⚠️ |
| 7 | Data Model (数据模型) | 2 | — | ❌ 缺失 |

---

## 4.1 Monitor — gNodeB 监控

### 原系统 `gnodeb_monitor.jsp`

| 原系统列 | 新系统 | 状态 |
|---------|--------|------|
| 连接状态图标 | connStatus | ✅ |
| `serial_number` | sn | ✅ |
| `host_name` | name | ✅ |
| `product_type` | productType | ✅ |
| `group_name` | — | ❌ |
| 告警(Critical列) | alarmLevel(单列合并) | ⚠️ |
| 告警(Major列) | alarmLevel(单列合并) | ⚠️ |
| 告警(Minor列) | alarmLevel(单列合并) | ⚠️ |
| 告警(Warning列) | alarmLevel(单列合并) | ⚠️ |

**缺失：** 分组筛选, 四级告警分列, 消息采集, 列自定义

---

## 4.2 Maintenance — 维护操作

### 原系统 `gnodeb/maintenance/` (16 文件)

| 页面 | 功能 | 新系统 | 状态 |
|------|------|--------|------|
| `device.jsp` | gNB 设备管理 | Device → DeviceList | ⚠️ |
| `deviceAdd.jsp` | 添加 gNB | Device → Registration | ⚠️ |
| `groupAdd.jsp` | 创建分组 | Device → Grouping | ✅ |
| `reboot.jsp` | gNB 重启 | — | ❌ |
| `rebootTaskAdd.jsp` | 创建重启任务 | — | ❌ |
| `mml.jsp` | MML 命令 | MML → Console | ✅ |
| `mmlScript.jsp` | MML 脚本 | MML → ScriptTask | ✅ |
| `addMMLTask.jsp` | MML 任务 | MML → ScriptTask | ✅ |
| `license.jsp` | 许可证 | License → LicenseList | ⚠️ |
| `selfConfig.jsp` | 自配置 | — | ❌ |
| `logs.jsp` | 日志 | Log → NEMessageLog | ⚠️ |
| `logsAdd.jsp` | 日志采集 | File → LogRetrieval | ⚠️ 缺创建 |
| `addSignalingTrace.jsp` | 信令追踪 | — | ❌ |
| `viewSignalingTrace.jsp` | 追踪结果 | — | ❌ |
| `signalingDetailInfo.jsp` | 追踪详情 | — | ❌ |
| `trace_detail.jsp` | 消息详情 | — | ❌ |

### rebootTaskAdd.jsp 表单字段

| 字段 | 说明 | 新系统 |
|------|------|--------|
| `taskname` | 任务名称(50字符) | ❌ |
| `productValue` | 产品类型 | ❌ |
| PairGrid左列 | connection_status, serial_number, host_name, product_type, group_name | ❌ |
| PairGrid右列 | serial_number, host_name | ❌ |
| `cellCodes` | 选中编码 | ❌ |
| `status` | 执行模式(立即/计划/周期) | ❌ |

### logsAdd.jsp 表单字段

| 字段 | 说明 | 新系统 |
|------|------|--------|
| PairGrid | 最多5台设备 | ❌ |
| `execute_type` | 立即/周期 | ❌ |
| `time` | 时间范围 | ❌ |
| `reportPeriod` | 上报周期(15/30/60分钟) | ❌ |

---

## 4.3 Backup & Restore

| 原系统页面 | 新系统 | 状态 |
|-----------|--------|------|
| `backupRestore.jsp` | Backup → BackupTasks | ⚠️ |
| `taskAdd.jsp` 创建表单 | — | ❌ 缺创建表单 |

---

## 4.4 Upgrade

| 原系统页面 | 新系统 | 状态 |
|-----------|--------|------|
| `gnbUpgradeTask.jsp` | Software → UpgradePlan | ⚠️ |
| 文件导入/编辑 | Software → FirmwareUpload | ⚠️ |

**缺失：** 逐设备进度, 升级类型选择, 回滚

---

## 4.5 Data Model (全部 ❌ 缺失)

`addParam.jsp` — 添加参数, `list.jsp` — 参数列表

---

## 缺失操作流程汇总

| 流程 | 状态 |
|------|------|
| gNodeB 重启全流程 | ❌ |
| 信令追踪全流程 | ❌ |
| gNodeB 自配置 | ❌ |
| 日志采集任务创建 | ❌ |
| 备份/恢复任务创建 | ❌ |
| 监控设置(16 文件) | ❌ |
| 数据模型管理 | ❌ |
