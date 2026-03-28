# 代码审查报告：系统配置默认值更新

**审查日期**: 2026-03-28
**审查者**: Claude Code
**变更范围**: SystemConfig
**审查结论**: PASS_WITH_WARNINGS

---

## 1. 变更概览

| 文件 | 变更行数 | 变更类型 |
|------|---------|---------|
| BasicSettings.tsx | 2 | 默认值调整 |
| DeviceSettings.tsx | 22 | 默认值调整 |
| NotificationSettings.tsx | 2 | 默认值调整 |
| OmcSettings.tsx | 24 | 默认值调整 + 选项扩展 |
| SecuritySettings.tsx | 203 | 文案更新 + 默认值调整 + 布局重组 |
| StorageSettings.tsx | 30 | 默认值调整 |

**总计**: 6 文件，~142 行新增，~143 行删除

---

## 2. 变更详情

### 2.1 BasicSettings.tsx
- `mrOMCName`: `'OMC'` → `''`
- `timezoneCode`: `'Asia/Shanghai'` → `''`

### 2.2 DeviceSettings.tsx
- `enbInformPeriod`: `300` → `60`
- `enbTimeout`: `900` → `''`
- `cpeInformPeriod`: `300` → `60`
- `cpeTimeout`: `900` → `''`
- RSRP 相关字段: 具体值 → `''`
- `uploadSelected`: `'3'` → `''`
- `deviceOfflineSaveDay`: `30` → `''`
- `latitudeToleranceRange`: `100` → `''`

### 2.3 NotificationSettings.tsx
- `mailPort`: `25` → `''`

### 2.4 OmcSettings.tsx
- `diskSpaceOptions`: 新增 10%-40% 选项，value 类型从 number 改为 string
- 磁盘阈值字段: `80` → `'10%'`
- `rsysLogPort`: `514` → `''`

### 2.5 SecuritySettings.tsx（主要变更）
**默认值调整**:
- `defaultPasswd`: `''` → `'OMC@123456'`
- `pwdMinLength`: `8` → `10`
- `pwdMaxLength`: `32` → `23`
- `validPeriod`: `90` → `70`
- `promptBeforeDays`: `7` → `6`
- `attemptTimes`: `3` → `5`
- `sumTimes`: `5` → `8`
- `unlockMinu`: `30` → `2`
- `limitMinus`: `5` → `1`
- `limitCount`: `10` → `30`
- `limitTimes`: `30` → `120`
- `userSessionExpirationMin`: `30` → `0`

**文案更新**（与旧版 JSP 对齐）:
- "首次登录修改密码" → "用户需要在首次登录时修改默认密码"
- "密码必须两种类型" → "密码需要包含数字、小写字母、大写字母和特殊字符(.!@#$%^&*?)"
- "用户修改密码频率" → "密码有效期为"
- "验证码验证时的用户名或密码提示" → "用户登录需要验证码验证，如果用户名或密码输入错误"
- "开启浏览器记录密码" → "禁止浏览器自动记录密码"
- "自动锁定超过" → "连续未登录omc超过"
- "同一用户只允许一个会话登录" → "允许用户同时在多个设备登陆"

**布局重组**:
- 移除未使用的 `sectionTitleStyle` 样式
- Card 分组调整：默认密码、密码强度、用户、密码有效期、登录锁定、IP限流、屏幕锁定、禁止浏览器自动记录密码、账户锁定、最大会话限制、登录提示

### 2.6 StorageSettings.tsx
- `sysOperateLogDataSaveDays`: `365` → `90`
- `kpiStorge60DataDays`: `60` → `30`
- `signalingTraceSaveDays`: `30` → `7`
- 其他存储周期字段: 具体值 → `''`

---

## 3. 审查检查项

### 3.1 前端规范 ✅
- [x] 无 `any` 类型使用
- [x] API 服务模式符合规范
- [x] 类型映射一致（diskSpaceOptions value 类型变更与 Select 组件兼容）
- [x] 无 XSS 风险

### 3.2 代码质量 ✅
- [x] 移除未使用代码（sectionTitleStyle）
- [x] 命名规范一致
- [x] 代码格式正确

### 3.3 业务逻辑 ⚠️
- [ ] 大量字段默认值改为空字符串，需确认后端能正确处理
- [ ] 磁盘阈值从 80% 改为 10%，需确认是否符合运维需求

---

## 4. 发现问题

### WARNING-1: 默认值空字符串处理
**级别**: WARNING
**位置**: 多个 Settings 组件
**描述**: 大量字段默认值从具体值改为空字符串，后端 API 需要能正确处理空字符串或 null 值
**建议**: 确认后端 API 对空字符串的处理逻辑，或考虑使用 `undefined` 代替

### INFO-1: 磁盘阈值默认值变更
**级别**: INFO
**位置**: OmcSettings.tsx
**描述**: 磁盘告警阈值从 80% 改为 10%，这是一个显著变更
**建议**: 确认这是否符合实际运维需求，10% 可能会导致频繁告警

---

## 5. 审查结论

**PASS_WITH_WARNINGS**

代码变更质量良好，文案更新与旧版 JSP 保持一致，布局清晰。建议关注默认值变更对后端 API 的影响。

---

## 6. 相关信息

- **功能域**: F06 OMC-R 核心
- **变更原因**: 与旧版 OMC 设置页面保持一致
- **影响范围**: 系统配置页面初始值显示
