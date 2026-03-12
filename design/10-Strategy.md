# Strategy (策略管理) 对比

## 覆盖状态：⚠️ 大量缺失

---

## 原系统

**目录：** `strategy/` — 5 个文件

### 二级菜单

| 页面 | 功能 | 新系统 |
|------|------|--------|
| `configuration_plan.jsp` | 创建/管理配置计划 | ⚠️ Config→Baseline |
| `configuration_oper.jsp` | 执行策略操作 | ❌ |
| `configuration_modify.jsp` | 修改策略配置 | ❌ |
| `configuration_import.jsp` | 导入策略文件 | ❌ |
| `configuration_GSM.jsp` | GSM 专用策略配置 | ❌ |

---

## 新系统部分对应

Config 菜单下有部分相关功能：
| 新系统页面 | 功能 | 说明 |
|-----------|------|------|
| Baseline (基线管理) | 配置基线 | 部分对应 configuration_plan |
| CommonConfig (公共配置) | 公共参数配置 | 部分对应 |
| BatchTemplate (批量模板) | 批量参数模板 | 新增 |

---

## 缺失汇总

| 缺失功能 | 严重度 |
|---------|--------|
| 策略执行操作 | 🔴 高 |
| 策略导入 | 🟡 中 |
| 策略修改 | 🟡 中 |
| GSM 策略 | 🟡 中 |
| 策略计划全流程 | 🔴 高 |
