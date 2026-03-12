# DHCP (DHCP 管理) 对比

## 覆盖状态：❌ 完全缺失

---

## 原系统

**目录：** `dhcp/` — 4 个文件

### 二级菜单

| 页面 | 功能 |
|------|------|
| `dhcp_config.jsp` | DHCP 服务器配置（地址池范围、租约时间、默认网关、DNS） |
| `dhcp_client_list.jsp` | 当前 DHCP 客户端租约列表（IP、MAC、租约时间、主机名） |
| `dhcp_restart.jsp` | 重启 DHCP 服务 |
| `dhcp_log.jsp` | DHCP 服务运行日志 |

---

## 新系统

**无对应菜单和页面。**

---

## 缺失严重度：🔴 高

使用 DHCP 自动分配 IP 的组网场景（如 OMC 作为 DHCP 服务器为基站分配地址）无法管理。
