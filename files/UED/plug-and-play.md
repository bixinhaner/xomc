# 即插即用页面

## 页面概述

**页面名称**: 即插即用 (Plug and Play)
**菜单路径**: 自配置 > 即插即用
**功能说明**: 实现设备零接触部署，包括软件升级、License管理、参数自配置等功能的策略管理和任务执行状态监控

---

## 页面布局

### Tab 页签
页面支持多网元类型切换：
- **eNB**: 小基站网元（默认）
- **CPE**: 客户端设备网元
- **GSM**: GSM网元（需配置启用）

### 双面板结构
每个Tab页签包含上下两个面板：
1. **策略列表** - 上半部分，展示已配置的策略
2. **执行状态** - 下半部分，展示任务执行状态
3. **可拖拽分割线** - 两个面板之间可拖拽调整高度

---

## eNB 策略列表

### 策略列表字段

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 操作 | - | 40px | 更多操作菜单按钮 |
| 是否启用 | selfStartEnable | 70px | 开关控件，'1'-启用，'0'-禁用 |
| 产品类型标志 | productType | 240px | 产品型号标识 |
| 策略名称 | policyName | 自适应 | 策略的唯一名称 |
| 执行方式 | executeType | 200px | '0'-eNB自动执行，'1'-eNB手动执行 |
| 软件升级 | upgradeEnable | 自适应 | '0'-禁用，'1'-启用（显示目标版本） |
| License | licenseEnable | 200px | '0'-禁用，'1'-启用 |
| eNB参数自配置 | selfConfigEnable | 200px | '0'-禁用，'1'-启用 |

### 软件升级字段详情

| 值 | 显示 | 说明 |
|----|------|------|
| '0' | 禁用 目标版本=xxx | 禁用软件升级，显示目标版本 |
| '1' | 启用 目标版本=xxx | 启用软件升级，显示目标版本 |

### 策略列表操作

#### 工具栏

| 操作 | 说明 |
|------|------|
| 新增 | 新增策略按钮（右上角） |
| 策略列表(N) | 显示策略总数 |
| 搜索框 | 按策略名称/目标版本搜索 |

#### 行操作菜单（更多按钮）

| 操作 | 图标 | 说明 | 显示条件 |
|------|------|------|----------|
| 信息 | info | 查看策略详情（只读） | 始终显示 |
| 修改 | edit | 编辑策略配置 | 策略开关为关闭状态 |
| 检测 | scan | 下发策略检测 | 始终显示 |
| 删除 | delete | 删除策略 | 策略开关为关闭状态 |

**操作禁用规则**:
- 当策略开关（selfStartEnable）为'1'（启用）时，修改和删除操作禁用

### 策略开关操作

- 点击开关可直接切换策略启用/禁用状态
- 需要权限码：`CODE_PLUG_AND_PLAY`

---

## eNB 执行状态

### 工具栏 - 统计信息

| 显示项 | 说明 |
|--------|------|
| 成功数 | enbSuccessCount，绿色图标 |
| 失败数 | enbFailCount，红色图标 |

### 工具栏 - Tab 切换

| Tab | 值 | 说明 |
|-----|-----|------|
| 所有任务 | '0' | 显示所有任务类型 |
| 软件升级 | '1' | 仅显示软件升级任务 |
| License | '2' | 仅显示License任务 |
| eNB参数自配置 | '3' | 仅显示参数配置任务 |

### 工具栏 - 筛选条件

| 字段 | 类型 | 说明 |
|------|------|------|
| 产品类型标志 | 下拉框 | 按产品类型筛选 |
| 状态 | 下拉框 | 按执行状态筛选 |
| 时间范围 | 日期时间范围 | 按执行时间筛选 |
| 搜索框 | 输入框 | 按策略名称/设备编码搜索 |

### 状态筛选选项

| 值 | 显示文本 |
|----|----------|
| ' ' | 所有 |
| '0' | 成功 |
| '1' | 失败 |
| '2' | 进行中 |
| '3' | 未执行 |
| '4' | 跳过 |

### 执行状态列表字段（所有任务 Tab）

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 选择 | - | 50px | 复选框，仅"所有任务"Tab显示 |
| 操作 | - | 125px | 重新执行、执行、删除、信息 |
| 设备编码 | serial_number | 240px | 设备序列号 |
| 产品类型标志 | product_type | 160px | 产品型号 |
| 策略名称 | policy_name | 200px | 策略名称 |
| 执行方式 | execute_type | 130px | '0'-自动执行，'1'-手动执行 |
| 开始时间 | start_time | 150px | 任务开始时间 |
| 结束时间 | end_time | 150px | 任务结束时间 |
| 步进度 | execute_procedure | 210px | 当前执行步骤 |
| 状态 | status | 110px | 执行状态标签 |
| 失败原因 | failure_reason | 200px | 失败时的原因说明 |

### 执行状态列表字段（软件升级 Tab）

额外显示字段：
| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 初始版本 | original_version | 150px | 升级前版本 |
| 目标版本 | target_version | 150px | 升级目标版本 |

### 执行状态列表字段（License Tab）

额外显示字段：
| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| License文件 | lic_file | 200px | License文件名 |

### 执行状态值说明

| 值 | 显示文本 | 样式 | 说明 |
|----|----------|------|------|
| '0' | 成功 | 绿色背景 #EEFFF3 | 任务执行成功 |
| '1' | 失败 | 红色背景 #FEF2F2 | 任务执行失败 |
| '2' | 进行中 | 蓝色背景 #E4F1FF | 任务正在执行 |
| '3' | 未执行 | 蓝色背景 #E4F1FF | 等待执行 |
| '4' | 跳过 | 黄色背景 #FFFDE3 | 任务被跳过 |

### 执行状态行操作

| 操作 | 图标 | 说明 | 显示条件 |
|------|------|------|----------|
| 重新执行 | restart | 重新执行任务 | 状态为成功/失败/跳过 |
| 执行 | start | 开始执行任务 | 手动模式且状态为未执行 |
| 删除 | delete | 删除任务记录 | 状态为成功/失败/未执行/跳过 |
| 信息 | info | 查看执行详情 | 始终显示 |

### 批量操作

#### 已选设备面板

| 功能 | 说明 |
|------|------|
| 已选数量 | 显示已选设备数量 |
| 查看已选 | 展开已选设备列表 |
| 清空 | 清空所有已选设备 |
| 单个删除 | 从已选列表中移除单个设备 |

#### 批量重新执行

- 弹出确认对话框
- 选项：同时执行成功任务（复选框）
- 执行后刷新列表

### 执行详情面板

点击"信息"按钮后，底部展开执行详情面板：

| 字段 | 字段名 | 说明 |
|------|--------|------|
| 进度 | progress | 1-软件升级，2-License，3-eNB参数自配置，4-Cell active |
| 状态 | status | 0-成功，1-失败，2-进行中，3-未执行，4-跳过 |
| 开始时间 | start_time | 步骤开始时间 |
| 结束时间 | end_time | 步骤结束时间 |
| 失败原因 | failure_reason | 失败时的原因说明 |

---

## CPE 策略列表

### 策略列表字段

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 操作 | - | 40px | 更多操作菜单按钮 |
| 是否启用 | policySwitch | 70px | 开关控件 |
| 产品型号 | productType | 自适应 | 产品型号 |
| 策略名称 | policyName | 自适应 | 策略名称 |
| 软件升级 | upgradeEnable | 自适应 | false-禁用，true-启用（显示目标版本） |
| eNB参数自配置 | configEnable | 自适应 | false-禁用，true-启用 |

### CPE 策略操作菜单

| 操作 | 图标 | 说明 | 显示条件 |
|------|------|------|----------|
| 信息 | info | 查看策略详情 | 始终显示 |
| 修改 | edit | 编辑策略配置 | 策略开关为关闭状态 |
| 删除 | delete | 删除策略 | 策略开关为关闭状态 |

---

## CPE 执行状态

### 工具栏 - Tab 切换

| Tab | 值 | 说明 |
|-----|-----|------|
| 所有任务 | '0' | 显示所有任务类型 |
| 软件升级 | '1' | 仅显示软件升级任务 |
| eNB参数自配置 | '3' | 仅显示参数配置任务 |

### 状态筛选选项

| 值 | 显示文本 |
|----|----------|
| ' ' | 所有 |
| success | 成功 |
| fail | 失败 |
| running | 进行中 |
| waitting | 等待 |
| skipped | 跳过 |

### 执行状态列表字段（所有任务 Tab）

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 选择 | - | 50px | 复选框，仅"所有任务"Tab显示 |
| 操作 | - | 90px | 重新执行、删除、信息 |
| 设备编码 | serialNumber | 240px | 设备序列号 |
| 产品型号 | productType | 160px | 产品型号 |
| 策略名称 | policyName | 200px | 策略名称 |
| 开始时间 | startTime | 150px | 任务开始时间 |
| 结束时间 | endTime | 150px | 任务结束时间 |
| 步进度 | step | 210px | 当前执行步骤 |
| 状态 | status | 110px | 执行状态标签 |
| 失败原因 | failureReason | 200px | 失败时的原因说明 |

### 执行状态列表字段（软件升级 Tab）

额外显示字段：
| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 初始版本 | original_version | 150px | 升级前版本 |
| 目标版本 | target_version | 150px | 升级目标版本 |

### CPE 执行状态值说明

| 值 | 显示文本 | 样式 |
|----|----------|------|
| success | 成功 | 绿色背景 #EEFFF3 |
| fail | 失败 | 红色背景 #FEF2F2 |
| running | 进行中 | 蓝色背景 #E4F1FF |
| waitting | 等待 | 蓝色背景 #E4F1FF |
| skipped | 跳过 | 黄色背景 #FFFDE3 |

### CPE 执行详情面板

| 字段 | 字段名 | 说明 |
|------|--------|------|
| 进度 | taskType | upgrade-软件升级，config-参数自配置 |
| 状态 | status | success/fail/running/waitting/skipped |
| 开始时间 | startTime | 步骤开始时间 |
| 结束时间 | endTime | 步骤结束时间 |
| 失败原因 | failureReason | 失败时的原因说明 |

---

## 策略检测功能（仅 eNB）

### 检测弹窗

点击"检测"操作后弹出设备选择弹窗。

#### 弹窗属性

| 属性 | 值 |
|------|------|
| 宽度 | 1100px |
| 组件 | el-pairgrid（双列表穿梭框） |
| 列表高度 | 400px |
| 显示行号 | 是 |
| 行标识字段 | serialNumber |

#### 弹窗标题

| 元素 | 内容 |
|------|------|
| 主标题 | 基站设备列表 |
| 副标题 | (设备检测提示) |

#### 左侧待选设备列表

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 选择 | - | - | 复选框列 |
| 连接状态 | connection_status | 45px | 设备在线状态图标（绿色-在线，灰色-离线） |
| 设备编码 | serialNumber | 120px | 设备序列号 |
| 名称(HostName) | cellName | 100px | 主机名 |
| 版本号 | softwareVersion | 100px | 软件版本 |
| 产品类型标志 | product | 80px | 产品类型标志 |
| 设备组 | groupName | 120px | 所属设备组名称 |

#### 右侧已选设备列表

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 设备编码 | serialNumber | 120px | 设备序列号 |
| 名称(HostName) | cellName | 100px | 主机名 |

#### 工具栏功能

| 功能 | 位置 | 说明 |
|------|------|------|
| 搜索框 | 左侧 | 按设备编码搜索，回车触发 |
| 批量输入按钮 | 右侧 | 点击打开批量输入弹窗 |

#### 底部操作按钮

| 按钮 | 说明 |
|------|------|
| 确定 | 下发策略检测任务，关闭弹窗 |
| 取消 | 关闭弹窗 |

### 批量输入弹窗

#### 弹窗属性

| 属性 | 值 |
|------|------|
| 标题 | 添加 |
| 宽度 | 630px |

#### 表单字段

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 设备编码 | serialNumber | 文本域 | 4行高度，支持换行、分号、空格分隔 |

#### 提示信息

显示在输入框下方：支持输入多个设备编码，使用换行或分号分隔

#### 验证规则

| 规则 | 说明 |
|------|------|
| 必填 | 不能为空 |
| 格式 | 数字、字母、中划线(-)、空格 |
| 长度 | 每个设备编码最多30字符 |

#### 正则表达式

```javascript
/^(\d|[a-zA-Z]|-|\s){1,30}$/
```

#### 底部操作按钮

| 按钮 | 说明 |
|------|------|
| 确定 | 验证通过后将设备添加到已选列表 |
| 取消 | 关闭弹窗 |

### 检测流程

```
1. 点击策略列表中的"检测"操作
   ↓
2. 弹出"基站设备列表"弹窗
   ↓
3. 从左侧待选列表选择设备（支持搜索、批量输入）
   ↓
4. 选中的设备显示在右侧已选列表
   ↓
5. 点击"确定"按钮
   ↓
6. 调用接口: /SON/SelfConfiguration/exeSelfConfigDetectPolicy.action
   ↓
7. 下发检测任务成功，关闭弹窗
```

---

## 新增/编辑策略

### 侧滑页面

通过侧滑页面展示策略配置表单，页面标题：新增/修改 Policy

### 功能区域

1. **基本信息** - 策略名称、产品类型等
2. **软件升级配置** - 目标版本、升级包选择
3. **License配置** - License文件选择
4. **参数自配置** - 参数模板选择

### 基本信息字段

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 设置开关 | selfStartEnable | 开关 | '1'-启用，'0'-禁用 |
| 策略名称 | policyName | 输入框 | 必填，策略唯一名称 |
| 产品类型标志 | productType | 下拉框 | 必填，从产品列表选择 |
| 执行方式 | executeType | 单选框 | '0'-eNB自动执行，'1'-eNB手动执行 |

### 功能模块选择

使用 el-radio-group (radio-button 类型) 展示三个功能模块，水平排列。

**提示文字**: 您可以选择以下模块进行配置

#### 模块选项

| 模块 | 值 | 图标 | 显示条件 | 说明 |
|------|-----|------|----------|------|
| 软件升级 | '0' | el-icon-status-upgrading | 始终显示 | 软件版本升级功能 |
| License | '1' | el-icon-operation-edit | 产品类型 != 'DXDF' | License文件管理 |
| eNB参数自配置 | '2' | el-icon-menu-system | 始终显示 | 参数自动配置 |

#### 模块切换交互

- 点击模块按钮切换到对应的配置面板
- 每个模块有独立的功能开关
- 切换模块时保留已配置的数据
- 模块按钮选中状态使用主色高亮

#### 模块配置面板

| 模块 | 显示条件 | 配置内容 |
|------|----------|----------|
| 软件升级 | functionModulesSelect == '0' | 初始版本配置、目标版本、保留配置 |
| License | functionModulesSelect == '1' | License文件列表、导入/删除操作 |
| eNB参数自配置 | functionModulesSelect == '2' | 配置列表（支持导入/导出/查看/修改/删除） |

### 软件升级配置

#### 功能开关

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| 软件升级开关 | upgradeEnable | Switch | '1'-启用，'0'-禁用 | 控制是否启用软件升级功能 |

**开关位置**: 在"软件升级"分组标题右侧

#### 初始版本配置

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| 初始版本 | - | 复选框+表格 | - | 支持指定版本或所有版本 |
| 初始版本类型 | specifyVersionType | Checkbox | '1'-指定版本，'0'-所有版本 | 勾选后显示"所有"文字，不勾选可添加具体版本 |

**初始版本类型说明**:
- `specifyVersionType = '0'`: 可添加具体的初始版本号
- `specifyVersionType = '1'`: 使用所有版本，此时添加按钮禁用

#### 添加初始版本面板

点击"+"按钮后，右侧弹出"添加版本"面板：

**面板标题**: 添加版本

**面板布局**: 上下两部分

##### 手动输入版本

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 初始版本 | versionStr | 输入框+添加按钮 | 手动输入版本号，点击"+"添加到列表 |
| 已添加版本列表 | versionList | 标签列表 | 显示已手动添加的版本，可单独删除 |

**验证规则**:
- 版本号不能为空
- 版本号不能重复

##### 初始版本列表（从OMC选择）

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 选择 | - | 50px | 复选框，支持多选 |
| 初始版本 | originalVersion | 自适应 | 已在OMC上线或上传的版本号 |

**列表特性**:
- 支持多选（checkbox）
- 根据已选产品类型过滤版本列表
- 与手动添加的版本合并去重

##### 添加版本操作

| 按钮 | 说明 |
|------|------|
| 确定 | 将手动添加和列表选中的版本合并添加到结果列表 |
| 取消 | 关闭面板，不保存更改 |

#### 已选初始版本结果列表

当添加版本后，显示已选版本表格：

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 初始版本 | originalVersion | 自适应 | 软件版本号 |
| 操作 | - | 50px | 删除按钮（hover显示） |

**工具栏操作**:

| 操作 | 图标 | 说明 | 显示条件 |
|------|------|------|----------|
| 搜索 | - | 按初始版本搜索过滤 | 始终显示 |
| 清空 | el-icon-operation-clear | 清空所有已选版本 | 非只读模式且specifyVersionType='0' |
| 添加 | el-icon-plus | 继续添加版本 | 非只读模式且specifyVersionType='0' |

#### 目标版本配置

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 目标版本 | targetVersion | 下拉框 | 选择升级目标版本，从可用版本列表中选择 |

**版本来源**: 根据产品类型获取可用的目标版本列表

#### 保留配置

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| eNB保留配置 | preserveSetting | Checkbox | '1'-保留，'0'-不保留 | 升级时是否保留eNB的配置信息 |

#### 软件升级配置数据结构

```javascript
{
  upgradeEnable: '0' | '1',           // 软件升级开关
  specifyVersionType: '0' | '1',      // 初始版本类型
  originalVersion: 'V1.0,V1.1,V1.2',  // 已选初始版本（逗号分隔）
  targetVersion: 'V2.0',              // 目标版本
  preserveSetting: '0' | '1'          // 保留配置
}
```

### License 配置

#### 功能开关

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| License开关 | licenseEnable | Switch | '1'-启用，'0'-禁用 | 控制是否启用License功能 |

**开关位置**: 在"License"分组标题右侧

**显示条件**: 仅当产品类型不是"DXDF"时显示此模块

#### License 文件列表

**列表标题**: 导入License文件

**工具栏操作**:

| 操作 | 图标 | 说明 | 显示条件 |
|------|------|------|----------|
| 搜索 | el-icon-common-search | 按小站编码搜索过滤 | 始终显示 |
| 导入 | el-icon-operation-import | 导入License文件 | 非只读模式 |

#### License 文件表格字段

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 操作 | - | 40px | 删除按钮 |
| 小站编码 | serial_number | 自适应 | 设备序列号 |
| License文件 | file_name | 自适应 | License文件名（.lic格式） |
| 上传时间 | upload_time | 150px | 文件上传时间 |
| 状态 | execute_status | 100px | 执行状态 |

#### License 状态值映射

| 值 | 显示文本 |
|----|----------|
| 0 | 未执行 |
| 1 | 执行成功 |
| 2 | 执行失败 |
| 3 | 执行中 |

#### 导入 License 面板

点击"导入"按钮后，右侧弹出"导入 License"面板：

**面板标题**: 导入 License

**面板属性**:
- 宽度: 320px
- 位置: 右侧滑出

##### 导入表单字段

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 文件名 | fileName | 文件上传 | 上传License文件（.lic格式） |

**文件上传要求**:
- 文件格式: .lic
- 支持多文件上传
- 上传前校验文件格式

##### 导入面板操作

| 按钮 | 说明 |
|------|------|
| 确定 | 上传文件并导入License |
| 取消 | 关闭面板 |

#### License 删除操作

点击删除按钮后弹出确认对话框：

**对话框内容**:
- 标题: 确认
- 消息: 确认删除？
- 按钮: 确定 / 取消

**删除接口**: `/cell/license/doClearLicenseFile.action`

**参数**: `{ fileNames: row.file_name }`

#### License 配置数据结构

```javascript
{
  licenseEnable: '0' | '1',  // License开关
  licenseList: [             // License文件列表（从后端获取）
    {
      serial_number: 'SN001',      // 小站编码
      file_name: 'license_001.lic', // License文件名
      upload_time: '2026-04-07 10:00:00', // 上传时间
      execute_status: '0'          // 状态
    }
  ]
}
```

### eNB 参数自配置（已废弃）

> **⚠️ 废弃说明**: 此设计已废弃，请参考下方"参数自配置模块（新设计）"章节。

<details>
<summary>点击展开废弃的参数数据池设计</summary>

#### 功能开关

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| 参数自配置开关 | selfConfigEnable | Switch | '1'-启用，'0'-禁用 | 控制是否启用参数自配置功能 |

**开关位置**: 在"eNB参数自配置"分组标题右侧

#### 参数数据池

**Tab页签结构**: 使用 el-tabs (border-card 类型) 展示参数配置

##### 参数设置开关

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| 设置开关 | switchEnable | Switch | '1'-启用，'0'-禁用 | 控制是否应用参数配置 |

**显示位置**: 参数数据池 Tab 内顶部

#### 参数数据池 - eNB 基础配置

使用 el-collapse 折叠面板展示配置项，默认展开。

##### 模块类型列表（仅 QAFA 产品）

**显示条件**: `curProductName == 'QAFA'`

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| 模块类型列表开关 | module_enable | Checkbox | '1'-启用，'0'-禁用 | 控制是否启用模块类型列表 |
| 添加按钮 | - | 按钮 | - | 添加模块类型（仅非只读模式显示） |

**模块类型列表表格字段**:

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 操作 | - | 70px | 编辑、删除按钮 |
| 设备型号名 | module_type | 自适应 | 模块类型名称 |
| 带宽 | band_width | 自适应 | 带宽配置值 |
| eNB频率 | frequency | 自适应 | 频率值 |

**模块类型添加/编辑弹窗**:

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 设备型号名 | module_type | 输入框 | 模块类型名称 |
| 带宽 | band_width | 下拉框 | 5MHz/10MHz/15MHz/20MHz |
| eNB频率 | frequency | 输入框 | 整形，范围: 1~65535 |

##### eNB 基础配置字段（通用）

| 字段 | 字段名 | 类型 | 验证规则 | 显示条件 | 说明 |
|------|--------|------|----------|----------|------|
| 支持频段 | bands_support | 输入框 | 整形，范围：1~62 | 所有产品 | 支持的频段 |
| 支持频段 | bands_support | 输入框 | 整形，范围：1~85 | DXDF产品 | 支持的频段 |
| 带宽 | band_width | 下拉框 | - | DXDF产品 | 6/15/25/50/75/100 |
| 带宽 | band_width | 下拉框 | - | 非NBIOT非DXDF产品 | 5MHz/10MHz/15MHz/20MHz |
| 带宽 | band_width | - | - | NBIOT产品 | 不显示 |
| eNB频率 | frequency | 输入框 | 整形，范围：0-262143 | DXDF产品 | 频率值 |
| eNB频率 | frequency | 输入框 | 整形，范围：1~65535 | 非NBIOT非DXDF产品 | 频率值 |
| DL Frequency | frequency | 输入框 | 整形，范围：1~65535 | 仅NBIOT产品 | 下行频率 |
| 子帧配比 | subframe_assignment | 下拉框 | - | 非NBIOT/QAFA/QAFB/BAIBLQ/DXDF | 0(DL:UL=1:3)/1(DL:UL=2:2)/2(DL:UL=3:1)/6(DL:UL=3:5) |
| 特殊子帧配比 | special_subframe_patterns | 下拉框 | - | 非NBIOT/QAFA/QAFB/BAIBLQ/DXDF | 5/7 |
| TAC | tac | 输入框 | 整形，范围：0~65535 | 所有产品 | 跟踪区域码 |
| ECI | cell_identity | 输入框 | 整形，范围：0~268435455 | 所有产品 | 小区标识 |
| PCI | phycellid | 输入框 | 整形，范围：0~503 | 所有产品 | 物理小区标识 |
| PLMN ID | plmn_id | 输入框 | 整形，范围：00000-999999 | DXDF产品 | PLMN标识 |
| CPE TxPower | txPower | 多选下拉框 | - | DXDF产品 | 发射功率 |
| 根序列索引 | root_sequence_index | 输入框 | 整形，范围：0~837 | 非NBIOT非DXDF产品 | RSI |
| 载波类型 | carrier_mode | 下拉框 | - | RTD产品 | 单载波/双载波 |

#### 参数数据池 - 核心网配置

**显示条件**: `curProductName != 'DXDF'`

使用 el-collapse 折叠面板展示配置项。

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| HaloB开关 | halob_enable | 下拉框 | '1'-开启，'0'-关闭 | 控制HaloB功能 |
| PLMN | plmn_id | 输入框 | 整形，范围：00000~999999 | PLMN标识 |
| MME | mme | 输入框+标签列表 | - | MME地址列表 |

**MME 配置说明**:
- 输入框右侧有"+"按钮添加MME地址
- 已添加的MME显示为标签列表
- 每个标签右侧有删除按钮
- 支持IP地址验证

**MME 列表字段**:

| 字段 | 说明 |
|------|------|
| domain | MME域名/IP地址 |
| 删除按钮 | 从列表中移除 |

#### 参数数据池 - 高级设置

**显示条件**: `groupShow.wan || groupShow.ipsec || groupShow.power || groupShow.dns || groupShow.ntp`

使用 el-collapse 折叠面板展示配置项。

##### 高级设置总开关

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| 高级设置开关 | advanceEnable | Switch | '1'-启用，'0'-禁用 | 控制是否应用高级设置 |

##### 参数配置选择器

点击"+"按钮打开参数选择树，可选参数组：
- WAN Config
- IPSec Settings
- Power Control Parameters
- DNS Config
- NTP Config

##### WAN Config

**显示条件**: `advanceTreeSelected.includes('wanConfig') && groupShow.wan`

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| MTU | mtu | 输入框 | 范围：700-1600 | 最大传输单元 |

##### IPSec Settings

**显示条件**: `advanceTreeSelected.includes('ipsecSettings') && groupShow.ipsec`

**IPSec 总开关**:

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| IPSEC Enable | ipsec_enable | Switch | '1'-启用，'0'-禁用 | 控制IPSec功能 |

**Tunnel 配置（支持多组）**:

| 产品类型 | 最大Tunnel数量 |
|----------|---------------|
| QAFA/QATA/QAFB | 1组 |
| RTS/QRTB/BAIBLQ/BLX/MLQ/CR-B4860/RTD/MLN | 2组 |

**Tunnel 字段**:

| 字段 | 字段名 | 类型 | 选项 | 说明 |
|------|--------|------|------|------|
| Enable | TUNNEL_ENABLE | Switch | '1'/'0' | 隧道开关 |
| AuthBy | authBy | 下拉框 | psk/cert/aka_psk/aka_cert | 认证方式 |
| leftAuth | LEFT_AUTH | 下拉框 | psk/pubkey/eap-aka | 左侧认证 |
| rightAuth | RIGHT_AUTH | 下拉框 | psk/pubkey/eap-aka | 右侧认证 |
| Gateway | TUNNEL_GATEWAY | 输入框 | 0-64字符，不含中文 | 网关地址 |
| leftId | LEFT_IDENTIFIER | 输入框 | 0-64字符，不含中文 | 左侧标识 |
| rightId | RIGHT_IDENTIFIER | 输入框 | 0-64字符，不含中文 | 右侧标识 |
| leftCert | LEFT_CERT | 输入框 | 0-64字符，不含中文 | 左侧证书 |
| secretKey | SECRET_KEY | 输入框 | 0-64字符，不含中文 | 密钥 |
| rightSecretKey | RIGHT_SECRET_KEY | 输入框 | 0-64字符，不含中文 | 右侧密钥 |
| leftSourceIp | LEFTSOURCEIP | 输入框 | IP地址或%config | 左侧源IP |
| leftSubnet | LEFT_SUBNET | 输入框 | 0-64字符，不含中文 | 左侧子网 |
| rightSubnet | RIGHT_SUBNET | 输入框 | 0-64字符，不含中文 | 右侧子网 |
| IKE Encryption | IKE_ENCRYPTION | 下拉框 | aes128/aes256/3des/des | IKE加密算法 |
| IKE DH Group | IKE_DH_GROUP | 下拉框 | modp768/modp1024/modp1536/modp2048/modp4096 | IKE DH组 |
| IKE Authentication | IKE_AUTHENTICATION | 下拉框 | sha1/sha1_160/sha256_96/sha256 | IKE认证算法 |
| ESP Encryption | ESP_ENCRYPTION | 下拉框 | aes128/aes256/3des/des | ESP加密算法 |
| ESP DH Group | ESP_DH_GROUP | 下拉框 | modp768/modp1024/modp1536/modp2048/modp4096 | ESP DH组 |
| ESP Authentication | ESP_AUTHENTICATION | 下拉框 | sha1/sha1_160/sha256_96/sha256 | ESP认证算法 |
| IKELifeTime | IKELIFETIME | 输入框 | 数字+s/m/h/d格式 | IKE生命周期 |
| KeyLife | KEYLIFE | 输入框 | 数字+s/m/h/d格式 | 密钥生命周期 |
| RekeyMargin | REKEYMARGIN | 输入框 | 数字+s/m/h/d格式 | 重密钥边界 |
| Dpdaction | DPDACTION | 下拉框 | none/clear/hold/restart | DPD动作 |
| Dpddelay | DPDDELAY | 输入框 | 数字+s/m/h/d格式 | DPD延迟 |

##### Power Control Parameters

**显示条件**: `advanceTreeSelected.includes('powerControl') && groupShow.power`

| 字段 | 字段名 | 类型 | 选项 | 显示条件 | 说明 |
|------|--------|------|------|----------|------|
| Total Tx Power | totalTxPower | 可搜索下拉框 | - | QAFA/QATA产品 | 总发射功率 |
| Power Ramping | powerRamping | 下拉框 | 0/2/4/6 | - | 功率爬升 |
| Preamble Init Target Power | preambleInitTargetPower | 下拉框 | -120~-90 | - | 前导初始目标功率 |
| Po_nominal_pusch | poNominalPusch | 输入框 | 范围：-126~24 | - | PUSCH标称功率 |
| Po_nominal_pucch | poNominalPucch | 输入框 | 范围：-126~24 | - | PUCCH标称功率 |

#### eNB 参数自配置数据结构

```javascript
{
  selfConfigEnable: '0' | '1',      // 参数自配置总开关
  switchEnable: '0' | '1',          // 参数设置开关
  // eNB基础配置
  module_enable: '0' | '1',         // 模块类型列表开关（仅QAFA）
  moduleTypeList: [],               // 模块类型列表
  bands_support: '',                // 支持频段
  band_width: '',                   // 带宽
  frequency: '',                    // 频率
  subframe_assignment: '',          // 子帧配比
  special_subframe_patterns: '',    // 特殊子帧配比
  tac: '',                          // TAC
  cell_identity: '',                // ECI
  phycellid: '',                    // PCI
  plmn_id: '',                      // PLMN ID
  root_sequence_index: '',          // 根序列索引
  carrier_mode: '',                 // 载波类型（仅RTD）
  // 核心网配置
  halob_enable: '0' | '1',         // HaloB开关
  mme: '',                          // MME输入框
  mmeGroup: [],                     // MME列表
  // 高级设置
  advanceEnable: '0' | '1',        // 高级设置开关
  advanceTreeSelected: [],          // 已选参数组
  // ... 其他高级设置字段
}

</details>

### 表单验证规则

| 字段 | 验证规则 | 错误提示 |
|------|----------|----------|
| 策略名称 | 必填 | 请输入策略名称 |
| 产品类型标志 | 必填 | 请选择产品类型标志 |
| 执行方式 | 必填 | 请选择执行方式 |
| 初始版本 | 当specifyVersionType='0'且升级开关开启时必填 | 请添加至少一个初始版本 |
| 目标版本 | 当升级开关开启时必填 | 请选择目标版本 |
| 支持频段 | 整形，范围：1~62（DXDF: 1~85） | 整形，范围：1~62 |
| eNB频率 | 整形，范围：0-262143（DXDF）/ 1~65535（其他） | 整形，范围：0-262143 |
| TAC | 整形，范围：0~65535 | 整形，范围：0~65535 |
| ECI | 整形，范围：0~268435455 | 整形，范围：0~268435455 |
| PCI | 整形，范围：0~503 | 整形，范围：0~503 |
| PLMN ID | 整形，范围：00000~999999 | 整形，范围：00000~999999 |
| 根序列索引 | 整形，范围：0~837 | 整形，范围：0~837 |
| MTU | 范围：700-1600 | Range: 700 - 1600 |
| IPSec Gateway | 0-64字符，不含中文 | 0-64字符，不含中文 |
| IPSec生命周期 | 数字+s/m/h/d格式 | 请输入数字加s/m/h/d格式 |

### 底部操作按钮

| 按钮 | 说明 |
|------|------|
| 保存 | 验证通过后保存策略配置 |
| 取消 | 关闭侧滑页面 |

---

## 参数自配置模块（新设计）

> **设计变更说明**: 原有的"参数数据池"Tab页签设计已废弃，改为直接显示配置列表。

### 功能开关

| 字段 | 字段名 | 类型 | 值 | 说明 |
|------|--------|------|-----|------|
| 参数自配置开关 | selfConfigEnable | Switch | '1'-启用，'0'-禁用 | 控制是否启用参数自配置功能 |

**开关位置**: 在"eNB参数自配置"分组标题右侧

### 配置列表

#### 工具栏

| 功能 | 说明 |
|------|------|
| 导入 | 点击弹出导入配置弹窗，支持Excel/CSV文件导入 |
| 导出 | 导出当前列表数据为Excel文件 |
| 搜索框 | 按基站编码搜索，支持模糊查询 |

#### 列表字段

| 字段 | 字段名 | 宽度 | 说明 |
|------|--------|------|------|
| 操作 | - | 120px | 查看、修改、删除按钮 |
| 基站编码 | serialNumber | 180px | 设备序列号 |
| 基站名称 | cellName | 150px | 基站名称/主机名 |
| 支持频段 | bandsSupport | 100px | 支持的频段 |
| 带宽 | bandWidth | 100px | 带宽配置值（5MHz/10MHz/15MHz/20MHz） |
| 频点 | frequency | 120px | 频率/频点值 |
| 子帧配比 | subframeAssignment | 120px | 子帧配比（0/1/2/6） |
| 更新人 | updatedBy | 100px | 最后更新用户 |
| 更新时间 | updatedAt | 160px | 最后更新时间 |

#### 行操作

| 操作 | 图标 | 说明 | 显示条件 |
|------|------|------|----------|
| 查看 | eye | 查看配置详情（只读） | 始终显示 |
| 修改 | edit | 修改配置 | 参数自配置开关为关闭状态 |
| 删除 | delete | 删除配置 | 参数自配置开关为关闭状态 |

**操作禁用规则**:
- 当参数自配置开关（selfConfigEnable）为'1'（启用）时，修改和删除操作禁用

### 导入配置弹窗

#### 弹窗属性

| 属性 | 值 |
|------|------|
| 标题 | 导入参数配置 |
| 宽度 | 500px |

#### 导入表单

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 选择文件 | file | 文件上传 | 支持 .xlsx/.xls/.csv 格式 |
| 模板下载 | - | 链接 | 下载导入模板 |

#### 导入模板字段

| 字段 | 字段名 | 必填 | 说明 |
|------|--------|------|------|
| 基站编码 | serialNumber | 是 | 设备序列号 |
| 基站名称 | cellName | 否 | 基站名称 |
| 支持频段 | bandsSupport | 是 | 支持的频段 |
| 带宽 | bandWidth | 是 | 5/10/15/20 |
| 频点 | frequency | 是 | 频率值 |
| 子帧配比 | subframeAssignment | 是 | 0/1/2/6 |

#### 底部操作按钮

| 按钮 | 说明 |
|------|------|
| 确定 | 上传文件并导入数据 |
| 取消 | 关闭弹窗 |

### 配置详情/修改弹窗

#### 弹窗属性

| 属性 | 值 |
|------|------|
| 标题 | 查看配置 / 修改配置 |
| 宽度 | 600px |

#### 配置表单字段

| 字段 | 字段名 | 类型 | 验证规则 | 只读 | 说明 |
|------|--------|------|----------|------|------|
| 基站编码 | serialNumber | 输入框 | 必填 | 是 | 设备序列号 |
| 基站名称 | cellName | 输入框 | - | 否 | 基站名称 |
| 支持频段 | bandsSupport | 输入框 | 整形，范围：1~62 | 否 | 支持的频段 |
| 带宽 | bandWidth | 下拉框 | 必填 | 否 | 5MHz/10MHz/15MHz/20MHz |
| 频点 | frequency | 输入框 | 整形，范围：1~65535 | 否 | 频率值 |
| 子帧配比 | subframeAssignment | 下拉框 | 必填 | 否 | 0(DL:UL=1:3)/1(DL:UL=2:2)/2(DL:UL=3:1)/6(DL:UL=3:5) |

#### 底部操作按钮

| 按钮 | 说明 |
|------|------|
| 保存 | 保存配置（仅修改模式显示） |
| 取消 | 关闭弹窗 |

### 删除确认弹窗

点击删除按钮后弹出确认对话框：

**对话框内容**:
- 标题: 确认删除
- 消息: 确定要删除该配置吗？
- 按钮: 确定 / 取消

### 数据结构

```javascript
// 配置列表项
interface ParamConfig {
  id: string;                    // 配置ID
  serialNumber: string;          // 基站编码
  cellName: string;              // 基站名称
  bandsSupport: number;          // 支持频段
  bandWidth: string;             // 带宽 (5MHz/10MHz/15MHz/20MHz)
  frequency: number;             // 频点
  subframeAssignment: number;    // 子帧配比 (0/1/2/6)
  updatedBy: string;             // 更新人
  updatedAt: string;             // 更新时间 (ISO 8601)
}

// 列表查询参数
interface ParamConfigListParams {
  serialNumber?: string;         // 基站编码（模糊查询）
  page: number;                  // 页码
  pageSize: number;              // 每页条数
}
```

### 交互流程

```
1. 用户点击"eNB参数自配置"模块
   ↓
2. 显示配置列表（无需Tab页签切换）
   ↓
3. 支持操作：
   - 搜索：输入基站编码进行模糊查询
   - 导入：上传Excel/CSV文件批量导入配置
   - 导出：导出当前列表为Excel文件
   - 查看：弹出只读详情弹窗
   - 修改：弹出编辑弹窗（需关闭开关）
   - 删除：确认后删除配置（需关闭开关）
```

---

## API 接口

### 策略管理接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/SON/SelfConfiguration/querySelfConfigPolicyPageList.action` | POST | 查询策略列表 |
| `/SON/SelfConfiguration/getSelfConfigurationInfo.action` | POST | 获取策略详情（用于编辑/查看） |
| `/SON/SelfConfiguration/addSelfConfigurationInfo.action` | POST | 新增策略 |
| `/SON/SelfConfiguration/updateSelfConfigurationInfo.action` | POST | 更新策略 |
| `/SON/SelfConfiguration/delSelfConfigPolicy.action` | POST | 删除策略 |
| `/SON/SelfConfiguration/getProductTypeSelect.action` | POST | 获取产品类型列表 |

### 软件升级接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/plugandplay/policy/getOriginalVersionPageList.action` | POST | 获取初始版本列表 |
| 参数 | 类型 | 说明 |
| product | string | 产品类型 |
| searchText | string | 搜索关键字 |
| page | number | 页码 |
| rows | number | 每页条数 |

### License 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/cell/license/queryLicenseFilePageList.action` | POST | 查询License文件列表 |
| `/cell/license/doClearLicenseFile.action` | POST | 删除License文件 |
| `/cell/license/uploadLicenseFile.action` | POST | 上传License文件 |

#### License 上传参数

| 参数 | 类型 | 说明 |
|------|------|------|
| uploadFile | file | License文件（.lic格式） |
| FileName | string | 文件名 |

### 参数配置接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/SON/SelfConfiguration/getParamConfig.action` | POST | 获取参数配置详情 |
| `/SON/SelfConfiguration/saveParamConfig.action` | POST | 保存参数配置 |

### eNB 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/SON/SelfConfiguration/querySelfConfigPolicyPageList.action` | POST | 查询策略列表 |
| `/SON/SelfConfiguration/getSelfConfigurationInfo.action` | POST | 获取策略详情 |
| `/SON/SelfConfiguration/updateSelfConfigurationInfo.action` | POST | 更新策略启用状态 |
| `/SON/SelfConfiguration/delSelfConfigPolicy.action` | POST | 删除策略 |
| `/SON/SelfConfiguration/getProductTypeSelect.action` | POST | 获取产品类型列表 |
| `/SON/SelfConfigurationTask/getSelfConfigTaskPageList.action` | POST | 查询执行状态列表 |
| `/SON/SelfConfigurationTask/getSelfConfigTaskRecordPageList.action` | POST | 查询执行详情 |
| `/SON/SelfConfigurationTask/reExeSelfConfigTask.action` | POST | 重新执行任务 |
| `/SON/SelfConfigurationTask/exeNextSelfConfigTask.action` | POST | 执行任务（手动模式） |
| `/SON/SelfConfigurationTask/delSelfConfigTask.action` | POST | 删除任务记录 |
| `/SON/SelfConfiguration/queryPolicyCellInfoPageList.action` | POST | 查询策略关联设备 |
| `/SON/SelfConfiguration/exeSelfConfigDetectPolicy.action` | POST | 下发策略检测 |
| `/SON/SelfConfiguration/querySelectedPolicyCellInfos.action` | POST | 批量查询设备信息 |

### CPE 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/plugandplay/policy/queryAutoPolicyInfoCPEPageList.action` | POST | 查询CPE策略列表 |
| `/plugandplay/policy/updatePolicySwitch.action` | POST | 更新策略开关 |
| `/plugandplay/policy/delAutoPolicy.action` | POST | 删除策略 |
| `/plugandplay/policy/getProductModelList.action` | POST | 获取产品型号列表 |
| `/plugandplay/task/queryAutoPolicyInfoCPEPageList.action` | POST | 查询CPE执行状态 |
| `/plugandplay/task/queryAutoPolicyTaskRecordPageList.action` | POST | 查询CPE执行详情 |
| `/plugandplay/task/reExecuteTask.action` | POST | 重新执行任务 |
| `/plugandplay/task/delAutoPolicyTask.action` | POST | 删除任务记录 |

---

## 权限控制

| 权限码 | 说明 |
|--------|------|
| CODE_PLUG_AND_PLAY | 即插即用功能权限（新增、修改、删除、执行） |

**权限影响**:
- 无权限时：新增按钮隐藏，操作菜单中的修改/删除/检测隐藏，执行状态操作禁用
- 有权限时：显示所有操作功能

---

## 前端组件建议

```
src/pages/provision/PlugAndPlay/
├── index.tsx                    # 主页面
├── components/
│   ├── PolicyList.tsx           # 策略列表组件
│   ├── ExecuteStatusList.tsx    # 执行状态列表组件
│   ├── PolicySlide.tsx          # 策略配置侧滑
│   ├── ExecuteDetailSlide.tsx   # 执行详情面板
│   ├── DetectDialog.tsx         # 检测设备选择弹窗
│   ├── BatchInputDialog.tsx     # 批量输入弹窗
│   ├── BatchRetryDialog.tsx     # 批量重试确认弹窗
│   └── SelectedDevicesPanel.tsx # 已选设备面板
├── hooks/
│   ├── useEnbPolicy.ts          # eNB策略相关hooks
│   └── useCpePolicy.ts          # CPE策略相关hooks
└── types.ts                     # 类型定义
```

---

## 状态样式定义

```css
/* 状态标签样式 */
.runningStatus {
  width: 90px;
  background: #E4F1FF;
  color: #4D84FF;
}

.successStatus {
  width: 76px;
  background: #EEFFF3;
  color: #4ED76E;
}

.skipStatus {
  width: 90px;
  background: #FFFDE3;
  color: #FFAA00;
}

.failStatus {
  width: 50px;
  background: #FEF2F2;
  color: #FF6D59;
}

.commonStatus {
  height: 24px;
  line-height: 24px;
  border-radius: 100px;
  text-align: center;
}
```

---

## 注意事项

1. **策略开关控制**: 策略启用时不可修改和删除
2. **执行状态刷新**: 执行状态列表每6秒自动刷新
3. **批量操作限制**: 仅"所有任务"Tab支持批量选择和操作
4. **设备选择限制**: 进行中和等待状态的任务不可选择
5. **面板高度调整**: 上下两个面板可通过拖拽分割线调整高度
6. **网元类型切换**: 切换Tab时清空已选数据和执行详情面板
7. **批量重试选项**: 批量重新执行时可选是否同时执行成功任务
8. **CPE差异**: CPE无License功能模块，状态值使用字符串而非数字

---

## 参数配置 - 指定设备计划（批量导入修改）

在参数配置模块中，支持"指定设备计划"功能，即为特定设备批量导入参数配置。以下是三种设备类型的批量导入修改页面字段。

### eNB 批量导入修改页面

#### 基础配置字段

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| 支持频段 | bands_support | 输入框 | 整形，范围：1~62 | 支持的频段 |
| 带宽 | band_width | 下拉框 | 5MHz/10MHz/15MHz/20MHz | 带宽配置 |
| eNB频率 | frequency | 输入框 | 整形，范围：1~65535 | 频率值 |
| 子帧配比 | subframe_assignment | 下拉框 | 0/1/2/6 | 0(DL:UL=1:3)/1(DL:UL=2:2)/2(DL:UL=3:1)/6(DL:UL=3:5) |
| 特殊子帧配比 | special_subframe_patterns | 下拉框 | 5/7 | 特殊子帧模式 |
| PLMN ID | plmn_id | 输入框 | 整形，范围：00000~999999 | PLMN标识 |
| TAC | tac | 输入框 | 整形，范围：0~65535 | 跟踪区域码 |
| ECI | cell_identity | 输入框 | 整形，范围：0~268435455 | 小区标识 |
| PCI | phycellid | 输入框 | 整形，范围：0~503 | 物理小区标识 |
| 根序列索引 | root_sequence_index | 输入框 | 整形，范围：0~837 | RSI |
| 载波类型 | carrier_mode | 下拉框 | 单载波/双载波 | 仅RTD产品 |
| 时区 | timeZoneUtc | 下拉框 | - | UTC时区设置 |

#### DXDF 特殊字段

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| CPE TxPower | dxdfTxPower | 下拉框 | - | 发射功率（DXDF产品专用） |

#### 核心网配置字段

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| HaloB开关 | halob_enable | 下拉框 | '1'-开启，'0'-关闭 | 控制HaloB功能 |
| MME | mme | 输入框 | - | MME地址（多个用逗号分隔） |
| MME列表 | mmeGroup | 标签列表 | - | MME地址列表 |

#### 小区参数2（仅 CR-B4860/MLN 产品）

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| PLMN ID | cellPlmnId | 输入框 | 5~6位数字 | 小区PLMN标识 |
| 带宽 | cellBandwidth | 下拉框 | 5MHz/10MHz/15MHz/20MHz | 小区带宽 |
| 频率 | cellFrequency | 输入框 | 整形 | 小区频率 |
| ECI | cellEci | 输入框 | 整形 | 小区ECI |
| PCI | cellPhycellid | 输入框 | 整形，范围：0~503 | 小区PCI |
| 根序列索引 | cellRootIndex | 输入框 | 整形，范围：0~837 | 小区RSI |
| MME | cellMme | 输入框 | - | 小区MME地址 |

#### Settings (IPsec) 配置

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| IPsec开关 | ipsec_switch | 开关 | IPsec总开关 |
| IPsec启用 | ipsec_enable | 下拉框 | '1'-启用，'0'-禁用 |
| Right IKE Port | ipsec_rightikeport | 输入框 | 右侧IKE端口 |
| Left Interface | left_interface | 输入框 | 左侧接口 |

**IPsec Tunnel 列表 (ipsecList)**:

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| Enable | TUNNEL_ENABLE | 开关 | 隧道开关 |
| AuthBy | authBy | 下拉框 | psk/cert/aka_psk/aka_cert |
| leftAuth | LEFT_AUTH | 下拉框 | psk/pubkey/eap-aka |
| rightAuth | RIGHT_AUTH | 下拉框 | psk/pubkey/eap-aka |
| Gateway | TUNNEL_GATEWAY | 输入框 | 网关地址 |
| leftId | LEFT_IDENTIFIER | 输入框 | 左侧标识 |
| rightId | RIGHT_IDENTIFIER | 输入框 | 右侧标识 |
| leftCert | LEFT_CERT | 输入框 | 左侧证书 |
| secretKey | SECRET_KEY | 输入框 | 密钥 |
| rightSecretKey | RIGHT_SECRET_KEY | 输入框 | 右侧密钥 |
| leftSourceIp | LEFTSOURCEIP | 输入框 | 左侧源IP |
| leftSubnet | LEFT_SUBNET | 输入框 | 左侧子网 |
| rightSubnet | RIGHT_SUBNET | 输入框 | 右侧子网 |
| IKE Encryption | IKE_ENCRYPTION | 下拉框 | aes128/aes256/3des/des |
| IKE DH Group | IKE_DH_GROUP | 下拉框 | modp768/modp1024/modp1536/modp2048/modp4096 |
| IKE Authentication | IKE_AUTHENTICATION | 下拉框 | sha1/sha1_160/sha256_96/sha256 |
| ESP Encryption | ESP_ENCRYPTION | 下拉框 | aes128/aes256/3des/des |
| ESP DH Group | ESP_DH_GROUP | 下拉框 | modp768/modp1024/modp1536/modp2048/modp4096 |
| ESP Authentication | ESP_AUTHENTICATION | 下拉框 | sha1/sha1_160/sha256_96/sha256 |
| IKELifeTime | IKELIFETIME | 输入框 | 数字+s/m/h/d格式 |
| KeyLife | KEYLIFE | 输入框 | 数字+s/m/h/d格式 |
| RekeyMargin | REKEYMARGIN | 输入框 | 数字+s/m/h/d格式 |
| Dpdaction | DPDACTION | 下拉框 | none/clear/hold/restart |
| Dpddelay | DPDDELAY | 输入框 | 数字+s/m/h/d格式 |

#### WAN Config（仅 CR-B4860/BAIBLQ 产品）

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| WAN发送使能 | wanSendEnable | 开关 | WAN发送功能开关 |

**WAN Binding 列表 (wanBindingList)**:

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| IP地址 | ipAddress | 输入框 | WAN IP地址 |
| 子网掩码 | netmask | 输入框 | 子网掩码 |
| 网关 | gateway | 输入框 | 网关地址 |
| VLAN ID | vlanId | 输入框 | VLAN标识 |
| WAN Binding | wanBinding | 下拉框 | WAN绑定配置 |

---

### gNB 批量导入修改页面

#### PLMN 配置

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| NCI | nci | 输入框 | 整形，范围：0~68719476735 | NR小区标识 |
| TAC | tac | 输入框 | 整形，范围：0~16777215 | 跟踪区域码 |
| RANAC | ranac | 输入框 | 整形，范围：0~255 | RAN区域码 |

#### PLMN Config 列表 (plmnConfigList)

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| PLMN ID | plmnId | 输入框 | 5~6位数字 | PLMN标识 |
| Primary | primary | 下拉框 | 0/1 | 是否为主PLMN |

#### Slice Config 列表 (sliceConfigList)

| 字段 | 字段名 | 类型 | 验证规则 | 说明 |
|------|--------|------|----------|------|
| SD | sd | 下拉框 | 0-空/1-非空 | SD标识 |
| SD Value | sd_value | 输入框 | SNSSAI格式 | SD值 |

#### AMF 配置

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| AMF IP | amfIp | 输入框 | AMF IP地址 |
| PLMN ID | plmnId | 输入框 | PLMN标识 |
| Default | default | 下拉框 | 是否为默认AMF |

#### WAN 配置

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 地址类型 | addressType | 下拉框 | IPv4/IPv6 |
| 承载类型 | bearType | 下拉框 | 承载类型 |
| IP地址 | ipAddress | 输入框 | WAN IP地址 |
| 子网掩码 | subnetMask | 输入框 | 子网掩码（IPv4） |
| 前缀长度 | prefixLength | 输入框 | 前缀长度（IPv6） |
| 网关 | gateway | 输入框 | 网关地址 |
| VLAN ID | vlanId | 输入框 | VLAN标识 |
| VLAN名称 | vlanName | 输入框 | VLAN名称 |

#### LAN 配置

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| LAN IP | lanIp | 输入框 | LAN IP地址 |
| 子网掩码 | lanSubnetMask | 输入框 | LAN子网掩码 |

---

### GSM 批量导入修改页面

#### 基础参数表格

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 序列号 | serialNumber | 输入框 | 设备序列号 |
| IPA Unit ID | ipaUnitid | 输入框 | IPA单元标识 |
| OML Remote IP | omlRemoteIp | 输入框 | OML远程IP地址 |
| OML Remote IP 备份 | omlRemoteIpBak | 输入框 | OML远程IP备份地址 |
| RF Power | rfPower | 输入框 | 射频功率 |

#### Route Config 路由配置

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 开机启动 | onboot | 下拉框 | yes/no |
| 网关 | gateway | 输入框 | 默认网关 |
| 网络地址 | netAddr | 输入框 | 网络地址 |
| 子网掩码 | netMask | 输入框 | 子网掩码 |

#### WAN Config 配置

| 字段 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| 使能 | enable | 开关 | WAN功能开关 |
| IP模式 | ipMode | 下拉框 | static/dhcp |
| IP地址 | ipAddr | 输入框 | WAN IP地址 |
| 子网掩码 | netMask | 输入框 | 子网掩码 |
| 网关 | gateway | 输入框 | 网关地址 |
| VLAN ID | vlanId | 输入框 | VLAN标识 |
