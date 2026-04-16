# 用户管理功能设计文档

## 1. 页面概述

用户管理页面用于管理系统中的用户信息，包括用户的创建、编辑、删除、锁定、密码重置等操作。

## 2. 列表字段

| 字段名 | 字段标识 | 宽度 | 说明 |
|--------|----------|------|------|
| 操作 | - | 100px | 操作按钮列（放在最前面） |
| 在线状态 | online_status | 100px | 显示"在线"或"不在线" |
| 锁定状态 | lock_status | 100px | 解锁/有效期锁定 |
| 用户名称 | user_name | 130px | 登录用户名 |
| 邮箱 | email | 自适应 | - |
| 用户组 | group_name | 150px | 支持多个用户组，悬浮显示全部 |
| 上次登录时间 | last_login_time | 160px | - |
| 来源 | source | 80px | 本地/LDAP |

## 3. 搜索条件

| 字段名 | 字段标识 | 类型 | 说明 |
|--------|----------|------|------|
| 用户名称 | userName | 输入框 | 模糊搜索，支持回车查询 |

## 4. 工具栏按钮

| 按钮名称 | 图标 | 权限控制 | 说明 |
|----------|------|----------|------|
| 导入 | ImportOutlined | CODE_SYSTEM_USERS_USER | 导入用户（Excel文件） |
| 导出 | ExportOutlined | - | 导出用户列表 |
| 添加 | PlusOutlined | CODE_SYSTEM_USERS_USER | 新增用户 |

## 5. 列表操作

每行数据有一个操作菜单（更多按钮），包含以下操作项：

| 操作名称 | 图标 | 说明 | 权限控制 | 条件 |
|----------|------|------|----------|------|
| 查看 | EyeOutlined | 查看用户详细信息 | - | 所有用户可查看 |
| 修改 | EditOutlined | 修改用户信息 | CODE_SYSTEM_USERS_USER | 非内置用户或admin |
| 复制 | CopyOutlined | 复制用户信息创建新用户 | CODE_SYSTEM_USERS_USER | - |
| - | - | 分割线 | - | - |
| 锁定/解锁 | LockOutlined/UnlockOutlined | 锁定或解锁用户 | CODE_SYSTEM_USERS_USER | admin可操作 |
| 强制退出登录 | LogoutOutlined | 强制用户下线 | CODE_SYSTEM_USERS_USER | admin可操作且用户在线 |
| 重置密码 | KeyOutlined | 重置用户密码 | CODE_SYSTEM_USERS_USER | admin可操作，LDAP用户不可重置 |
| - | - | 分割线 | - | - |
| 删除 | DeleteOutlined | 删除用户 | CODE_SYSTEM_USERS_USER | 非内置用户可删除（危险操作） |

## 6. 批量操作

勾选多条记录后，底部显示批量操作栏：

| 操作名称 | 图标 | 说明 | 条件 | 确认弹窗 |
|----------|------|------|------|----------|
| 强制退出登录 | LogoutOutlined | 批量强制用户下线 | 选中用户在线 | 是 |
| 锁定 | LockOutlined | 批量锁定用户 | - | 是 |
| 解锁 | UnlockOutlined | 批量解锁用户 | - | 是 |
| 重置密码 | KeyOutlined | 批量重置选中用户密码 | 非LDAP用户 | 是 |
| 移动用户组 | - | 将用户移动到指定用户组 | - | 否（弹选择框） |
| 删除 | DeleteOutlined | 批量删除选中用户 | 非内置用户 | 是（危险操作） |

## 7. 添加/编辑用户

通过抽屉（Drawer）滑出面板进行用户信息的添加和编辑。抽屉宽度520px。

### 表单字段

| 字段名 | 字段标识 | 必填 | 类型 | 校验规则 | 说明 |
|--------|----------|------|------|----------|------|
| 用户名称 | userName | 是 | Input | 3-32字符，仅字母数字下划线 | 修改时不可编辑 |
| 密码 | password | 是（新增） | Input.Password | 最少8位，需包含两种类型 | - |
| 确认密码 | confirmPassword | 是（新增） | Input.Password | 需与密码一致 | - |
| 邮箱 | email | 否 | Input | 邮箱格式，最大50字符 | - |
| 手机号 | phone | 是 | Input | 手机号格式(11位)或固话格式 | 短信验证模式下必填 |
| 用户组 | groupNames | 是 | Select[mode=multiple] | 至少选择一个 | 支持多选 |
| 锁定状态 | lockStatus | 否 | Radio | 0=解锁，1=有效期锁定 | 默认解锁 |
| 到期时间 | expireTime | 条件 | DatePicker | 需大于当前时间 | 不勾选"不限制时间"时必填 |
| 不限制时间 | limitTime | 否 | Checkbox | - | 默认勾选 |
| 描述 | description | 否 | TextArea | 最大500字符 | - |

### 用户组选择说明

用户组分为三种预设类型：
- **Super Admin**：超级管理员组（内置组，ID=1）
- **Default Group**：默认组（内置组，ID=100000）
- **Customize**：自定义用户组（多选）

### 表单校验规则详情

#### 用户名校验
- 必填
- 不能包含中文
- 只能包含字母(a-z, A-Z)、数字(0-9)、下划线(_)、减号(-)、破折号(—)
- 长度：3-32字符
- 不能与已存在用户名重复

#### 密码校验
- 必填
- 长度：8-20字符
- 允许字符：字母、数字、特殊字符(_!@#$%^&*?)
- 必须包含至少两种类型（大写字母、小写字母、数字、特殊字符）

#### 邮箱校验
- 格式：标准邮箱格式
- 最大长度：50字符

#### 手机号校验
- 短信验证模式：必填，11位手机号(1开头)
- 非短信模式：可选，固话格式(数字、-、+、()、空格)

### 新增用户API

```
POST /api/v1/users
```

请求体：
```json
{
  "userName": "string",
  "password": "string",
  "email": "string",
  "phone": "string",
  "groupNames": ["string"],
  "lockStatus": 0,
  "expireTime": "2026-12-31 23:59:59",
  "description": "string"
}
```

### 编辑用户API

```
PUT /api/v1/users/{id}
```

请求体：
```json
{
  "email": "string",
  "phone": "string",
  "groupNames": ["string"],
  "lockStatus": 0,
  "expireTime": "2026-12-31 23:59:59",
  "description": "string"
}
```

## 8. 查看用户

通过抽屉（Drawer）滑出面板查看用户详细信息，所有字段只读。

### 查看字段

| 字段名 | 说明 |
|--------|------|
| 在线状态 | 在线/不在线 |
| 锁定状态 | 解锁/有效期锁定 |
| 用户名称 | - |
| 邮箱 | - |
| 用户组 | 多个用逗号分隔 |
| 来源 | 本地/LDAP |
| 上次登录时间 | - |
| 描述 | - |

## 9. 导入用户

### 导入入口
- 点击工具栏"导入"按钮
- 右侧滑出导入面板（Drawer，宽度350px）

### 导入面板内容

| 元素 | 说明 |
|------|------|
| 标题 | 导入 |
| 文件选择 | 支持拖拽或点击选择 |
| 文件格式 | .xlsx, .xls |
| 下载模板链接 | 点击下载导入模板 |
| 确定按钮 | 开始导入 |
| 取消按钮 | 关闭面板 |

### 导入流程
1. 点击"导入"按钮，右侧滑出导入面板
2. 点击选择Excel文件（仅支持.xlsx/.xls格式）
3. 选择文件后，文件名显示在输入框中
4. 点击"确定"按钮开始导入
5. 导入成功后自动关闭面板并刷新列表
6. 导入失败显示错误信息

### 导入API

```
POST /api/v1/users/import
Content-Type: multipart/form-data
```

请求参数：
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | File | 是 | Excel文件(.xlsx/.xls) |

响应：
```json
{
  "success": true,
  "message": "导入成功",
  "data": {
    "total": 100,
    "success": 98,
    "failed": 2,
    "errors": [
      {"row": 5, "reason": "用户名已存在"},
      {"row": 12, "reason": "邮箱格式错误"}
    ]
  }
}
```

### 下载模板API

```
GET /api/v1/users/import/template
```

返回：Excel文件下载

### 导入模板格式

| 列 | 字段名 | 必填 | 说明 |
|----|--------|------|------|
| A | 用户名称 | 是 | 3-32字符 |
| B | 密码 | 是 | 8-20字符 |
| C | 邮箱 | 否 | 邮箱格式 |
| D | 手机号 | 是 | 11位手机号 |
| E | 用户组 | 否 | 多个用逗号分隔 |
| F | 描述 | 否 | 最大500字符 |

## 10. 导出用户

### 导出入口
- 点击工具栏"导出"按钮
- 导出当前搜索条件下的所有用户

### 导出API

```
GET /api/v1/users/export
```

请求参数：
| 参数名 | 类型 | 说明 |
|--------|------|------|
| userName | string | 用户名称（模糊搜索） |

返回：Excel文件下载

## 11. 移动用户组

### 弹窗内容
- 弹窗标题：移动用户组
- 弹窗宽度：420px
- 显示目标用户组下拉选择框

### 弹窗字段

| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| 目标用户组 | Select | 是 | 单选，显示所有用户组 |

### 移动用户组API

```
POST /api/v1/users/move-group
```

请求体：
```json
{
  "userIds": ["string"],
  "groupId": "string"
}
```

## 12. 重置密码

### 单个重置
- 点击操作菜单"重置密码"
- 弹出重置密码弹窗（Modal，宽度420px）

### 弹窗字段

| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| 新密码 | Input.Password | 是 | 需符合密码规则 |
| 确认密码 | Input.Password | 是 | 需与新密码一致 |

### 批量重置
- 选中用户后点击"重置密码"
- 弹出确认弹窗
- 确认后批量重置为默认密码

### 重置密码API

```
POST /api/v1/users/{id}/reset-password
```

请求体：
```json
{
  "newPassword": "string"
}
```

### 批量重置密码API

```
POST /api/v1/users/batch-reset-password
```

请求体：
```json
{
  "userIds": ["string"]
}
```

## 13. 锁定/解锁用户

### 锁定API

```
POST /api/v1/users/{id}/lock
```

### 解锁API

```
POST /api/v1/users/{id}/unlock
```

### 批量锁定API

```
POST /api/v1/users/batch-lock
```

请求体：
```json
{
  "userIds": ["string"]
}
```

### 批量解锁API

```
POST /api/v1/users/batch-unlock
```

请求体：
```json
{
  "userIds": ["string"]
}
```

## 14. 强制退出登录

### 单个强制退出
- 点击操作菜单"强制退出登录"
- 弹出确认弹窗

### 批量强制退出
- 选中在线用户后点击"强制退出登录"
- 弹出确认弹窗

### 强制退出API

```
POST /api/v1/users/{id}/force-logout
```

### 批量强制退出API

```
POST /api/v1/users/batch-force-logout
```

请求体：
```json
{
  "userIds": ["string"]
}
```

## 15. 删除用户

### 单个删除
- 点击操作菜单"删除"
- 弹出确认删除弹窗
- 内置用户不可删除

### 批量删除
- 选中用户后点击"删除"
- 弹出确认删除弹窗
- 内置用户自动跳过

### 删除API

```
DELETE /api/v1/users/{id}
```

### 批量删除API

```
DELETE /api/v1/users/batch
```

请求体：
```json
{
  "userIds": ["string"]
}
```

## 16. 复制用户

### 复制流程
- 点击操作菜单"复制"
- 系统自动复制用户信息
- 跳转到新增页面，预填被复制用户的信息
- 用户名需要修改（不能重复）

### 复制用户API

```
POST /api/v1/users/{id}/copy
```

## 17. 状态说明

### 在线状态
| 状态值 | 显示文本 | 标签颜色 |
|--------|----------|----------|
| online | 在线 | green |
| offline | 不在线 | default |

### 锁定状态
| 状态值 | 显示文本 | 标签颜色 |
|--------|----------|----------|
| 0 | 解锁 | default |
| 1 | 有效期锁定 | warning |
| 2 | 有效期锁定 | warning |

### 来源
| 值 | 显示文本 |
|----|----------|
| 本地 | 本地 |
| LDAP | LDAP |

## 18. 权限控制

| 权限码 | 说明 |
|--------|------|
| CODE_SYSTEM_USERS_USER | 用户管理权限（增删改查） |
| CODE_SYSTEM_USERS_USER_GROUP | 用户组管理权限 |
| CODE_SYSTEM_USERS_ROLE | 角色管理权限 |

## 19. 特殊规则

### 内置用户
- `admin` 用户为超级管理员，不可删除
- `build_in = 1` 的用户为内置用户，部分操作受限
- 运营商管理员 `build_in = 9`，有特殊权限限制

### LDAP用户
- 来源为 LDAP 的用户，密码相关操作不可用
- 密码由LDAP服务器管理，OMC不可修改

### 短信验证模式
- 系统配置项，影响手机号校验规则
- 开启时手机号必填且需为11位手机号
- 关闭时手机号可选且支持固话格式

### 密码规则配置
- 最小长度：系统配置 `min_password_length`
- 最大长度：系统配置 `max_password_length`
- 密码强度：是否要求包含多种字符类型

### 用户名规则配置
- 最小长度：系统配置 `min_username_length`
- 是否检查特殊字符：系统配置 `check_username_chars`
