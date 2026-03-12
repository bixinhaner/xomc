# System (���统管理) 对比

## 覆盖状态：⚠️ 部分覆盖

---

## 二级菜单对比

### 1. User Management (用户管理)

#### 原系统 user.jsp

**用户标签页 — 表格列：**
| 列 | 字段名 | 新系统 UserManagement |
|----|-------|---------------------|
| 在线状态 | online_status | ❌ **缺失** |
| 锁定状态 | lock_status | ✅ status (active/locked/inactive) |
| 用户名称 | user_name | ✅ username |
| 邮箱 | email | ✅ email |
| 用户组 | group_name | ❌ **缺失** — 新系统无用户组概念 |
| 上次登录时间 | last_login_time | ✅ lastLoginTime |
| 来源 | source | ❌ **缺失** |
| — | — | ✅ displayName **新增** |
| — | — | ✅ phone **新增** |
| — | — | ✅ role **新增** |

**角色标签页 — 表格列：**
| 列 | 原系统 | 新系统 RolePermission |
|----|--------|---------------------|
| 角色名称 | role_name | ✅ 角色列表 |
| 支持批量 | batch_operation | ❌ |
| 操作人 | user | ❌ |
| 更新时间 | update_time | ❌ |
| 描述 | desc | ✅ description |
| 用户数 | — | ✅ userCount **新增** |

**用户组标签页 — 表格列：**
| 列 | 原系统 |
|----|--------|
| 组名 | group_name |
| 用户数 | user_count |
| 角色数 | role_count |
| 操作人 | upd_user |
| 更新时间 | upd_time |

**新系统：** ❌ 用户组功能完全缺失

#### 原系统 user_add.jsp 表单字段
| 字段 | 原系统 | 新系统 CreateUser |
|------|--------|------------------|
| 用户名 | ✅ userName | ✅ username |
| 邮箱 | ✅ email | ✅ email |
| 手机号 | ✅ phone | ✅ phone |
| 密码 | ✅ password | ✅ password |
| 确认密码 | ✅ confirmPwd | ✅ confirmPassword |
| 用户组 | ✅ group (admin/default/customize) | ❌ **缺失** |
| 用户组 ID | ✅ groupIds | ❌ **缺失** |
| 到期时间 | ✅ expireTime | ❌ **缺失** |
| 不限时间 | ✅ limitTime | ❌ **缺失** |
| 描述 | ✅ desc | ❌ **缺失** |
| 状态 | ✅ status | ❌ (创建时无) |
| — | — | ✅ displayName **新增** |
| — | — | ✅ role **新增** |

**操作按钮对比：**
| 操作 | 原系统 | 新系统 |
|------|--------|--------|
| 添加 | ✅ | ✅ |
| 导出 | ✅ | ❌ |
| 导入 | ✅ | ❌ |
| 强制退出登录 | ✅ | ❌ **缺失** |
| 有效期锁定 | ✅ | ❌ — 无有效期概念 |
| 解锁 | ✅ | ✅ Unlock |
| 密码重置 | ✅ | ✅ Reset Password |
| 移动用户组 | ✅ | ❌ |
| 删除 | ✅ | ✅ |
| 锁定 | — | ✅ Lock **新增** |

---

### 2. Operator Management (运营商管理)

#### 原系统 operatorList.jsp
**表格列：**
| 列 | 字段名 |
|----|-------|
| 运营商名称 | operator_name |
| CLOUDKEY | cloud_key |
| eNB 数量 | eNodeBCount |
| CPE 数量 | CPECount |
| Beta 状态 | is_beta |

**表单字段：**
| 字段 | 说明 |
|------|------|
| operatorCode | 运营商名称 |
| cloudKey | Cloud Key |
| adminUserCode | 默认管理员 |

**操作：** 添加、查看信息、修改、删除、移到运营商

**新系统：** ❌ 完全缺失 — 运营商管理功能在新系统中没有��应

---

### 3. Role Management (角色管理)

**原系统文件：** `sys/role/` — 4 个文件

**新系统对应：** System → RolePermission
| 功能 | 原系统 | 新系统 |
|------|--------|--------|
| 角色列表 | ✅ | ✅ |
| 权限分配 | ✅ | ✅ 权限树 (8 模块) |
| 角色 CRUD | ✅ | ⚠️ (仅权限编辑，无创建/删除) |

**新系统权限模块：** device, alarm, performance, software, file, log, system, report, ops

---

### 4. Resource Monitoring (资源监控)

#### 原系统 monitor_resource.jsp
**CPU 表格列：** cpuUserUsage, cpuSystemUsage, cpuTotalUsagePt, cpuFreePercent
**磁盘表格列：** devName, sysTotalSize, sysFreeSize, sysUsedSize
**内存表格列：** memoryTotal, memoryFree, memoryFreePt, memoryUsed, memoryUsedPt
**数据库当前：** threads_connected, threads_running, max_connections, data_size, index_size, free_size
**数据库历史：** time, data_size, index_size, free_size, increase_size

**新系统对应：** System → SystemDashboard
| 功能 | 原系统 | 新系统 | 差异 |
|------|--------|--------|------|
| CPU 监控 | ✅ 4 列 | ⚠️ | 需验证 |
| 磁盘监控 | ✅ 4 列 | ⚠️ | 需验证 |
| 内存监控 | ✅ 5 列 | ⚠️ | 需验证 |
| 数据库监控 | ✅ 11 列 | ❌ | **缺失** |

---

### 5. Certificate Management (证书管理)

**原系统文件：** `sys/certificate/` — 8 个文件

| 功能 | 新系统 |
|------|--------|
| SSL/TLS 证书上传 | ❌ |
| 证书查看/删除 | ❌ |
| CA 根证书管理 | ❌ |
| 证书有效期监控 | ❌ |

**新系统：** ❌ 完全缺失

---

### 6. Log Management (日志管理)

**原系统文件：** `sys/logmanage/` — 5 个文件
| 子目录 | 功能 |
|--------|------|
| `omc/` (2 文件) | OMC 系统日志 |
| `device/` (3 文件) | 设备操作日志 |

**新系统对应：** Log 菜单 (6 个页面)
| 功能 | 原系统 | 新系统 | 差异 |
|------|--------|--------|------|
| 系统日志 | ✅ omc/ | ✅ SystemLog | — |
| 设备操作日志 | ✅ device/ | ✅ OperationLog | — |
| 网元消息日志 | ❌ | ✅ NEMessageLog | **新增** |
| 心跳日志 | ❌ | ✅ HeartbeatLog | **新增** |
| 告警日志 | ❌ | ✅ AlarmLog | **新增** |
| 日志配置 | ❌ | ✅ LogConfig | **新增** |

**结论：日志管理新系统功能超越原系统。**

---

### 7. License Management (许可管理)

**原系统文件：** `sys/license/` — 1 个文件 + `sys/device/license/` — 6 个文件

| 功能 | 原系统 | 新系统 License 菜单 |
|------|--------|-------------------|
| 许可证列表 | ✅ license_task.jsp | ✅ LicenseList |
| 许可任务列表 | ✅ taskList.jsp | ⚠️ LicenseOperations |
| 创建许可任务 | ✅ addTask.jsp | ❌ **缺失** |
| 任务进度 | ✅ taskProgress.jsp | ❌ **缺失** |
| 文件信息查看 | ✅ viewInfo.jsp | ✅ 详情面板 |
| 文件信息修改 | ✅ modifyInfo.jsp | ❌ **缺失** |
| 许可文件导入 | ✅ | ❌ **缺失** |
| 许可证撤销 | ❌ | ✅ Revoke **新增** |
| 许可证日志 | ❌ | ✅ LicenseLogs **新增** |

---

### 8. Backup & Recovery (系统备份恢复)

**���系统文件：** `sys/backupAndRecover/` — 1 个文件

**新系统对应：** ❌ 系统级备份缺失（Backup 菜单是设备配置备份）

---

### 9. Block Management (设备黑名单)

**原系统文件：** `sys/blockManage/` — 1 个文件

**新系统：** ❌ 完全缺失

---

### 10. Device Migration (设备迁移)

**原系统文件：** `sys/deviceMigration/` — 1 个文件

**新系统：** ❌ 完全缺失

---

### 11. Settings (系统设置)

**原系统文件：** `sys/settings/` — 2 个文件

**新系统对应：** System → SystemConfig + DataDict + NotificationSettings
| 功能 | 原系统 | 新系统 | 差异 |
|------|--------|--------|------|
| 系统配置 | ✅ | ✅ SystemConfig | — |
| 数据字典 | ❌ | ✅ DataDict | **新增** |
| 通知设置 | ❌ | ✅ NotificationSettings | **新增** |

---

### 12. Provider (服务商)

**原系统文件：** `sys/provider/` — 1 个文件

**新系统：** ❌ 完全缺失

---

## 缺失汇总

| 缺失功能 | 严重度 | 说明 |
|---------|--------|------|
| 运营商管理 (CRUD + 设备分配) | 🔴 高 | 多运营商场景必需 |
| 用户组管理 | 🟡 中 | 批量用户管理 |
| 用户有效期/过期锁定 | 🟡 中 | 安全管理 |
| 强制退出登录 | 🟡 中 | 安全控制 |
| 证书管理 (8 页面) | 🔴 高 | HTTPS/IPSec 安全 |
| 数据库监控 | 🟡 中 | 系统运维 |
| 设备黑名单管理 | 🟡 中 | 安全控制 |
| 设备迁移 | 🟡 中 | 跨 OMC 迁移 |
| 服务商管理 | 🟡 中 | 多服务商场景 |
| 系统级备份恢复 | 🔴 高 | OMC 系统自身备份 |
| 许可任务创建和进度 | 🔴 高 | 许可下发管理 |
| 用户导入/导出 | 🟡 中 | 批量用户管理 |
