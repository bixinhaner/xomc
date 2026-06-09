# 系统设置功能清单

> 整理自旧版 OMC 系统设置页面（settings_tabs.jsp + UICustom.jsp）

---

## 一、设置页签（sysSettings）

### 1. 基本设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| 运营商名称 | mrVendor | 输入运营商名称，最大50字符 | 空 |
| OMC名称 | mrOMCName | 输入网管系统名称，最大200字符 | 空 |
| 时区设置 | timezoneCode | 选择系统时区 | 空 |
| 语言设置 | languageCode | 选择系统语言（中文/英文）| 空 |

### 2. 安全设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| **默认密码** | | | |
| 首次登录修改密码 | modifyPWD | 勾选启用首次登录强制修改密码 | ☐ 关闭 |
| 默认密码值 | defaultPasswd | 设置密码重置后的默认密码 | 空 |
| **密码强度** | | | |
| 密码必须两种类型 | passwordContent | 勾选启用密码复杂度要求 | ☐ 关闭 |
| 密码最小长度 | pwdMinLength | 设置密码最小长度 | 6 位 |
| 密码最大长度 | pwdMaxLength | 设置密码最大长度 | 20 位 |
| **用户名称** | | | |
| 用户名必含字符提示 | checkUserCodeEnable | 用户名命名规则提示 | ☐ 关闭 |
| **密码有效期** | | | |
| 用户修改密码频率 | expires | 勾选启用密码有效期 | ☐ 关闭 |
| 有效期天数 | validPeriod | 设置密码有效期（天）| 空 |
| 提示到期天数 | promptBeforeDays | 密码到期前N天提示 | 空 |
| **登录错误限制** | | | |
| 验证码验证 | verifyEnable | 登录错误N次后需要验证码 | ☐ 关闭 |
| 错误次数阈值 | attemptTimes | 触发验证码的错误次数 | 空 |
| 登录失败锁定次数 | sumTimes | 登录失败N次后锁定用户 | 空 |
| 锁定时间 | unlockMinu | 用户锁定时间（分钟）| 空 |
| **IP限流** | | | |
| 限流时间窗口 | limitMinus | X分钟内 | 空 |
| 连续错误次数 | limitCount | 连续错误N次 | 空 |
| 自动释放时间 | limitTimes | Y分钟后自动释放IP | 空 |
| **锁屏时间** | userSessionExpirationMin | 用户无操作锁屏时间（分钟）| 空 |
| **浏览器记录密码** | | | |
| 开启浏览器记录密码 | isBrowserAutoRecordPass | 允许浏览器保存密码 | ☐ 关闭 |
| **自动锁定** | | | |
| 自动锁定开关 | autoLockUserDayEnable | 启用自动锁定长期未登录用户 | ☐ 关闭 |
| 锁定天数阈值 | autoLockUserDay | 超过N天未登录自动锁定用户 | 90 天 |
| **最大会话限制** | | | |
| 限制单一会话 | isOnlyOneUserLoginEnable | 同一用户只允许一个会话登录 | ☐ 关闭 |
| **登录提示** | | | |
| 启用登录后提示 | enabledFlag | 勾选启用登录后消息通知 | ☐ 关闭 |
| 提示消息内容 | msg | 登录后显示的通知消息文本 | 空 |

### 3. 设备设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| **设备Inform周期** | | | |
| ENB心跳周期检测 | enbInformPeriodAdjustEnable | 勾选启用ENB心跳周期自动调整 | ☐ 关闭 |
| ENB Inform周期 | enbInformPeriod | ENB心跳周期（秒）| 60 秒 |
| ENB超时时间 | enbTimeout | 规定时间无响应设备关机（秒）| 空 |
| CPE心跳周期检测 | cpeInformPeriodAdjustEnable | 勾选启用CPE心跳周期自动调整 | ☐ 关闭 |
| CPE Inform周期 | cpeInformPeriod | CPE心跳周期（秒）| 60 秒 |
| CPE超时时间 | cpeTimeout | 规定时间无响应设备关机（秒）| 空 |
| **设备名称同步** | | | |
| 检查相同设备名称 | nameSettingEnable | 检查和设置相同设备名称的LMT | ☐ 关闭 |
| 通知手动同步 | prompt | 通知是否手动同步 | ☐ 关闭 |
| **设备访问控制** | | | |
| 访问控制开关 | accessContralEnable | 只允许符合规则的设备访问系统 | ☐ 关闭 |
| **设备信号强度显示** | | | |
| 信号弱阈值 | rsrpVal0 | 信号弱 < 阈值 dBm | 空 |
| 信号正常阈值 | rsrpVal1 | 信号正常 < 阈值 dBm | 空 |
| 信号强 | - | >= rsrpVal1 显示为强 | - |
| **UE设备信号强度** | | | |
| UE信号弱阈值 | uersrpVal0 | UE信号弱 < 阈值 | 空 |
| UE信号正常阈值 | uersrpVal1 | UE信号正常 < 阈值 | 空 |
| **基站文件上传协议** | | | |
| 上传协议选择 | uploadSelected | HTTP / HTTPS / 保持基站不变 | 空 |
| **回收站** | | | |
| 离线设备移入回收站 | deviceOfflineEnable | 启用离线设备自动移入回收站 | ☐ 关闭 |
| 保存天数 | deviceOfflineSaveDay | 回收站设备保存天数 | 空 |
| **基站位置移动检测（GPS检测）** | | | |
| 位置检测开关 | locationDetection | 启用设备经纬度变化检测 | ☐ 关闭 |
| 经纬度容差范围 | latitudeToleranceRange | 经纬度变化超过此范围触发告警（米）| 空 |

### 4. 通知设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| **通知服务器** | | | |
| 启用邮件通知 | emEnabel | 勾选启用通知邮件服务器 | ☐ 关闭 |
| 邮箱 | mailUsername | 发件邮箱地址 | 空 |
| 密码 | mailPassword | 邮箱密码 | 空 |
| SMTP服务器 | mailHost | SMTP服务器地址 | 空 |
| 端口 | mailPort | SMTP端口 | 空 |
| 测试按钮 | - | 发送测试邮件验证配置 | - |

### 5. 存储设置

#### 5.1 日志设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| 设备原始文件存储 | logDataSaveDays | 日志存储时长（1/3/6月）| 空 |
| 异常日志存储 | rebootLogDataSaveDays | 异常日志存储时长（1/7/30/60/90天）| 空 |
| 保留异常日志次数 | rebootLogSaveCount | 最多保留异常日志次数（1/2次）| 空 |
| 操作日志存储时长 | sysOperateLogDataSaveDays | 操作日志存储时长（3/6/12/24/36月）| 90 天 |
| **远程存储** | | | |
| 转发到远程地址 | logFtpEnable | 勾选启用日志远程转发 | ☐ 关闭 |
| FTP协议 | logFtpType | SFTP / FTP | 空 |
| 上传路径 | logFtpSavePath | 远程服务器存储路径 | / |
| IP地址 | logFtpIpAddr | 远程服务器IP | 空 |
| 端口 | logFtpPort | 远程服务器端口 | 空 |
| 用户名 | logFtpUser | 登录用户名 | 空 |
| 密码 | logFtpPassword | 登录密码 | 空 |

#### 5.2 告警设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| 历史告警存储天数 | alarmHisMaxHoldTime | 数据库数据存储天数 | 空 |

#### 5.3 指标设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| KPI文件存储天数 | kpiFilesSaveDays | 设备报告存储天数 | 空 |
| KPI报表文件存储天数 | kpiReportDataSaveDays | KPI报表存储天数 | 空 |
| KPI原始数据存储 | kpiStorge15DataDays | 15分钟粒度数据存储天数 | 空 |
| KPI小时数据存储 | kpiStorge60DataDays | 60分钟粒度数据存储天数（30/60/90天）| 30 天 |
| KPI天数据存储 | kpiStorge1440DataDays | 1440分钟粒度数据存储天数 | 空 |
| KPI周月查询粒度 | kpiWeekAndMonthSwitch | 支持周月查询粒度开关 | ☐ 关闭 |

#### 5.4 MR设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| MR存储天数 | mrFileSaveDays | 设备报告原始文件存储天数 | 空 |

#### 5.5 心令追踪设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| 心令追踪文件存储 | signalingTraceSaveDays | 设备报告存储天数 | 7 天 |

### 6. 网管设置（OMC）

#### 6.1 协议方式（Syslog）

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| 协议方式开关 | rsysLogEnable | 启用/禁用Syslog | ☐ 关闭 |
| IP地址 | rsysLogIp | Syslog服务器IP | 空 |
| 端口 | rsysLogPort | Syslog端口 | 空 |
| 测试按钮 | - | 测试连接状态 | - |

#### 6.2 磁盘告警

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| 日志目录阈值 | varDiskAlarmThresHold | /var 目录磁盘告警阈值 | 10% |
| 数据目录阈值 | homeDiskAlarmThresHold | /home 目录磁盘告警阈值 | 10% |
| 应用目录阈值 | usrDiskAlarmThresHold | /usr 目录磁盘告警阈值 | 10% |
| 根目录阈值 | rootDiskAlarmThresHold | / 目录磁盘告警阈值 | 10% |

### 7. 北向接口设置

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| IP地址 | northboundIp | 北向接口IP（只读）| 空 |
| 端口 | northboundPort | 北向接口端口（只读）| 空 |
| 服务状态 | northboundServiceStatus | 服务运行状态显示 | 0 (停止) |
| **用户列表** | | | |
| 是否启用 | userEnable | 用户启用/禁用开关 | - |
| 用户名称 | userName | 北向接口用户名 | - |
| 密码 | userPwd | 北向接口密码 | - |
| 创建时间 | responseTime | 用户创建时间 | - |
| 操作 | - | 添加/编辑/删除用户 | - |

### 8. SAS设置

#### 8.1 SAS心跳日志

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| SAS心跳日志 | mainLogHbEnable | 启用/禁用SAS心跳日志 | ☐ 关闭 |

#### 8.2 SAS Provider列表

表格列：
| 列名 | 字段名 | 说明 |
|------|--------|------|
| 操作 | - | 修改/删除操作按钮 |
| Provider名称 | providerName | SAS提供者名称 |
| 服务器URL | url | SAS服务器地址 |
| TLS证书 | certName, validTimeStr, uploadSuccess | 证书名称(有效期)，状态图标 |
| 更新时间 | updateTimeStr | 最后更新时间 |

#### 8.3 添加/编辑 SAS Provider 弹窗

| 字段 | 字段名 | 类型 | 说明 | 必填 |
|------|--------|------|------|------|
| SAS Provider | provider | 文本输入 | SAS提供者名称，最大50字符，编辑时不可修改 | 是 |
| SAS Server URL | serverUrl | 文本输入 | SAS服务器URL，最大200字符 | 是 |
| **TLS证书** | | | | |
| 证书文件类型 | type | 单选 | .PEM / .P12 | 是 |
| 证书文件 | certFile | 文件上传 | PEM格式(.pem/.crt)或P12格式(.p12) | 添加时必填 |
| 私钥文件 | privateKeyFile | 文件上传 | 仅PEM模式显示，PEM格式 | 添加时必填 |
| 密码 | password | 密码输入 | 证书存在密码时输入 | 否 |

**PEM模式**：需上传证书文件(.pem/.crt)和私钥文件(.pem)
**P12模式**：仅需上传证书文件(.p12)，如有密码则输入

按钮：确定 / 取消

### 9. LDAP协议

| 功能项 | 字段名 | 说明 | 默认值 |
|--------|--------|------|--------|
| LDAP启用 | ldapEnable | 启用/禁用LDAP认证 | ☐ 关闭 |
| LDAP IP | ldapIp | LDAP服务器IP | 空 |
| LDAP端口 | ldapPort | LDAP服务器端口 | 空 |
| SSL/TLS | ldapSSL | 启用SSL/TLS加密 | ☐ 关闭 |
| LDAP Base | ldapBase | LDAP基础DN | 空 |
| LDAP用户 | ldapUser | LDAP绑定用户 | 空 |
| LDAP密码 | ldapPwd | LDAP绑定密码 | 空 |
| 测试按钮 | - | 测试LDAP连接 | - |

---

## 二、UI定制化页签（UICustom）

| 功能项 | 字段名 | 说明 |
|--------|--------|------|
| OMC名称 | ui_omc_name | 系统名称，显示在浏览器标题 |
| 主题色 | ui_color | 系统主题颜色（颜色选择器），支持预览 |
| 登录背景 | ui_login_background | 登录页面背景图片（最大1MB，支持JPG/PNG）|
| Logo小图 | ui_menu_logo_up | 菜单收起时显示的Logo（最大400KB）|
| Logo大图 | ui_menu_logo_down | 菜单展开时显示的Logo（最大400KB）|
| 预览按钮 | - | 预览主题色效果 |
| 确定按钮 | - | 保存当前设置 |
| 恢复按钮 | - | 恢复默认UI设置 |

### UI定制化默认值

| 项目 | 默认值 |
|------|--------|
| 主题色 | #FF4614 |
| 登录背景 | ./images/login/login_bg.png |
| Logo小图 | ./images/login/nav_logo_collapse.png |
| Logo大图 | ./images/login/logo_big.png |
| OMC名称 | BaiOMC |

---

## 三、权限说明

| 页签/模块 | 权限要求 |
|-----------|----------|
| 设置页签 | 根据具体模块有不同权限要求 |
| 基本设置 | 部分功能仅超级用户可见 |
| 安全设置 | 仅管理员（isAdmin）可见 |
| 设备设置 | 部分功能仅超级用户可见 |
| 通知设置 | 仅管理员（isAdmin）可见 |
| 存储设置 | 仅管理员（isAdmin）可见 |
| 北向接口设置 | 管理员/子管理员/北向角色可见 |
| SAS设置 | 仅管理员且isSAS存在时可见 |
| LDAP协议 | 仅Local版本可见 |
| UI定制化 | 仅管理员且Local版本可见 |

---

## 四、API接口

| 接口 | 方法 | 说明 |
|------|------|------|
| /system/settings/query.action | GET | 查询系统设置 |
| /system/settings/update.action | POST | 更新系统设置 |
| /ui/customization/updateCustomizationInfo.action | POST | 更新UI定制化设置 |
| /system/northuser/list.action | GET | 查询北向接口用户列表 |
| /cell/SAS/provider/list.action | GET | 查询SAS Provider列表 |

---

## 五、默认勾选状态说明

> ☐ 表示默认未勾选，☑ 表示默认勾选

### 安全设置默认状态

| 功能项 | 默认勾选 |
|--------|----------|
| 首次登录修改密码 | ☐ |
| 密码必须两种类型 | ☐ |
| 用户名必含字符提示 | ☐ |
| 用户修改密码频率 | ☐ |
| 验证码验证 | ☐ |
| 开启浏览器记录密码 | ☐ |
| 自动锁定 | ☐ |
| 限制单一会话 | ☐ |
| 启用登录后提示 | ☐ |

### 设备设置默认状态

| 功能项 | 默认勾选 |
|--------|----------|
| ENB心跳周期检测 | ☐ |
| CPE心跳周期检测 | ☐ |
| 检查相同设备名称 | ☐ |
| 通知手动同步 | ☐ |
| 访问控制开关 | ☐ |
| 离线设备移入回收站 | ☐ |
| 位置检测开关 | ☐ |

### 存储设置默认状态

| 功能项 | 默认勾选 |
|--------|----------|
| 转发到远程地址 | ☐ |
| KPI周月查询粒度 | ☐ |

### 其他设置默认状态

| 功能项 | 默认勾选 |
|--------|----------|
| 启用邮件通知 | ☐ |
| 协议方式开关 | ☐ |
| SAS心跳日志 | ☐ |
| LDAP启用 | ☐ |
| SSL/TLS | ☐ |
