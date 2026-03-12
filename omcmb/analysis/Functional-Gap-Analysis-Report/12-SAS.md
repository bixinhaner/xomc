# SAS (频谱接入系统) 对比

## 覆盖状态：❌ 完全缺失

---

## 原系统

**目录：** `sas/` — 9 个文件

### 二级菜单

| 页面 | 功能 |
|------|------|
| `sas_cpi.jsp` | CPI (Common Public Interface) 配置 |
| `sas_properties.jsp` | SAS 属性配置 |
| `sas_log.jsp` | SAS 运行日志 |
| `sas_monitor.jsp` | SAS 状态监控 |
| `sasImportInstallParam.jsp` | 导入安装参数 |
| 其他 4 个文件 | SAS 相关配置和操作 |

---

## 新系统

**无对应菜单和页面。**

---

## 影响

CBRS 频段（美国市场 3.5GHz 共享频谱）设备必须通过 SAS 进行频谱授权。缺失此模块意味着：
- 无法进行设备 CPI 注册
- 无法监控频谱授权状态
- 无法管理安装参数

**缺失严重度：🔴 高** — 如涉及 CBRS 部署，此模块为合规必需
