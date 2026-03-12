# Advance (高级功能) 对比

## 覆盖状态：⚠️ 大量缺失

---

## 原系统

**目录：** `advance/` — 9 个文件

### 二级菜单

#### 1. ANR (自动邻区关系) — 3 文件

| 页面 | 功能 | 新系统 |
|------|------|--------|
| `anr/anr.jsp` | ANR 管理主页 | ❌ |
| `anr/setting.jsp` | ANR 算法参数设置 | ❌ |
| `anr/information.jsp` | ANR 运行信息查看 | ❌ |

---

#### 2. MR (测量报告) — 3 文件

| 页面 | 功能 | 新系统 |
|------|------|--------|
| `mr/mr.jsp` | MR 管理主页 | ✅ MR→MRFiles |
| `mr/addMrTask.jsp` | 创建 MR 采集任务 | ✅ MR→MRTasks |
| `mr/downloadMrFile.jsp` | 下载 MR 文件 | ⚠️ MR→MRFiles (需验证下载) |

---

#### 3. PCI Confused/Conflict (PCI 冲突检测) — 3 文件

| 页面 | 功能 | 新系统 |
|------|------|--------|
| `pci_confused_conflict/pci_confused_conflict.jsp` | PCI 冲突检测主页 | ❌ |
| `pci_confused_conflict/pci_confused_conflict_details.jsp` | 冲突详情分析 | ❌ |
| `pci_confused_conflict/pci_confused_conflict_setting.jsp` | 检测参数设置 | ❌ |

---

## 缺失汇总

| 缺失功能 | 严重度 | 说明 |
|---------|--------|------|
| ANR 全部功能 | 🔴 高 | 自动邻区管理是网络优化核心 |
| PCI 冲突检测全部功能 | 🔴 高 | PCI 干扰影响网络质量 |
| MR 文件下载 | 🟡 中 | 需验证 |
