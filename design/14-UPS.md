# UPS (电源管理) 对比

## 覆盖状态：❌ 完全缺失

---

## 原系统

**目录：** `UPS/` — 14 个文件

### 二级菜单

#### 1. UPS 信息管理 — 8 文件
| 页面 | 功能 |
|------|------|
| `ups_information.jsp` | UPS 状态信息（电压、电量、温度、运行时间） |
| `ups_register.jsp` | 注册新 UPS |
| `ups_monitor.jsp` | UPS 实时监控（电池健康、输入/输出电压、负载） |
| `add_ups.jsp` | 添加 UPS 设备 |
| `import_ups.jsp` | 批量导入 UPS |
| `move_ups.jsp` | 迁移 UPS 到其他基站 |
| `fileMgrforUPS.jsp` | UPS 固件文件管理 |
| `versionMgrForUPS.jsp` | UPS 版本管理 |

#### 2. UPS 升级 — 6 文件
| 页面 | 功能 |
|------|------|
| `upgrade/upsUpgradePage.jsp` | UPS 升级管理主页 |
| `upgrade/upsUpgradeAddTask.jsp` | 创建 UPS 升级任务 |
| `upgrade/upsUpgradeImportFile.jsp` | 导入 UPS 固件文件 |
| `upgrade/upsUpgradeFileEdit.jsp` | 编辑固件文件信息 |
| `upgrade/upsUpgradeTaskList.jsp` | 升级任务列表 |
| `upgrade/upsUpgradeTaskResult.jsp` | 升级结果查看 |

---

## 新系统

**无对应菜单和页面。**

---

## 缺失严重度：🟡 中

如站点部署了 UPS 电源管理模块，则需要此功能进行电源状态监控和固件升级。
