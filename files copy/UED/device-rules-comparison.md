# 设备归属设备组规则页面分析

> 基于 original-omc 项目 JSP 页面分析
> 源文件: `original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/NewEGW/regist/deviceAttrRule.jsp`

---

## 1. 页面概述

**设备归属设备组规则** (Device Attribute Rule) 是用于自动将设备分配到指定设备组的功能。通过配置匹配规则，系统可以自动将符合条件的设备移动到目标设备组。

### 入口位置
- 从设备注册页面 (eGW_regist.jsp) 点击"配置规则"按钮进入
- 入口函数: `configuraRule()`
- API路径: `/moveDeviceGroupRule/rule/toDeviceAttrRule.action`

---

## 2. 规则列表页面

### 2.1 列表字段

| 字段名 | 属性名 | 宽度 | 说明 |
|--------|--------|------|------|
| 操作 | - | 110px | 包含启动/禁用、编辑、删除按钮 |
| Enable | enable | 80px | 开关，控制规则启用状态 (1=启用, 0=禁用) |
| 操作 | operators | 320px | 显示规则的匹配条件描述 |
| 创建时间 | create_time | 120px | 规则创建时间 |

### 2.2 工具栏按钮

| 按钮 | 图标 | 功能 |
|------|------|------|
| 搜索 | el-icon-common-search | 按操作描述模糊搜索规则 |
| 添加规则 | el-icon-plus | 打开添加规则表单 |
| 关闭 | el-icon-close | 关闭规则页面 |

### 2.3 行操作按钮

| 按钮 | 图标 | 功能 | 显示条件 |
|------|------|------|----------|
| 重新执行 | el-icon-operation-start | 重新对选定设备组执行规则 | enable=1 时显示 |
| 编辑 | el-icon-operation-edit | 编辑规则 | 始终显示 |
| 删除 | el-icon-operation-delete | 删除规则 | 始终显示 |

### 2.4 特殊功能

- **拖拽排序**: 支持通过拖拽表格行调整规则执行顺序
- **开关切换**: 直接在列表中切换规则启用状态

---

## 3. 添加/编辑规则表单

### 3.1 表单字段

| 字段名 | 属性名 | 类型 | 必填 | 说明 |
|--------|--------|------|------|------|
| 目标设备组 | move_to_group_id | Select | 是 | 下拉选择目标设备组 |
| Enable | enable | Switch | 否 | 启用/禁用开关 (默认禁用) |
| 匹配规则 | matching_mode | Radio | 是 | 匹配模式选择 |

### 3.2 匹配规则选项

| 选项值 | 显示名称 | 显示条件 |
|--------|----------|----------|
| deviceName | 设备名称 | 始终显示 |
| lac | LAC | 仅 eNB 且支持 GSM 时显示 |
| tac | TAC | 非 CPE 设备时显示 |

### 3.3 设备名称匹配 (matching_mode = 'deviceName')

当选择"设备名称"匹配时，显示多条件过滤表单：

#### 过滤条件选项

| 值 | 显示名称 |
|------|----------|
| contain | 包含 |
| notContain | 不包含 |
| startWith | 以什么开始 |
| endWith | 以什么结束 |

#### 逻辑组合选项

| 值 | 显示名称 | 说明 |
|------|----------|------|
| and | 与 | 所有条件都必须满足 |
| or | 或 | 任一条件满足即可 |

#### 条件输入

- **第一行**: 只有过滤条件 + 输入框
- **后续行**: And/Or 下拉 + 过滤条件 + 输入框 + 删除按钮
- **最大条件数**: 10 条
- **输入长度限制**: 64 字符

#### 条件组合规则

- Or 下面不能出现 And
- Or 上面相邻的 And 能改成 Or
- 一旦选择了 Or，后续新增条件默认也是 Or

### 3.4 TAC/LAC 匹配 (matching_mode = 'tac' 或 'lac')

当选择 TAC 或 LAC 匹配时：

| 字段 | 属性名 | 格式 | 验证规则 |
|------|--------|------|----------|
| TAC | tac_rag | 逗号分隔或范围 | eNB: 0-65535, gNB: 0-16777215 |
| LAC | tac_rag | 逗号分隔或范围 | 0-65535 |

#### 格式示例
- 单个值: `1,2,3`
- 范围: `1-10,20-30`
- 混合: `1,2,3,10-20`

---

## 4. 重新执行规则 (Active)

### 4.1 功能说明
对选定的设备组重新执行规则，将符合条件的设备移动到目标设备组。

### 4.2 表单字段

| 字段名 | 类型 | 说明 |
|--------|------|------|
| 执行规则的设备组 | Tree | 设备组树形选择器，多选 |

### 4.3 验证规则
- 必须至少选择一个设备组

---

## 5. API 接口

### 5.1 规则列表查询
```
POST /moveDeviceGroupRule/rule/list/get.action
参数:
  - timeZone: 时区
  - searchText: 搜索文本
  - deviceType: 设备类型 (ENB/GNB/CPE)
```

### 5.2 添加规则
```
POST /moveDeviceGroupRule/rule/add.action
Content-Type: application/json

参数:
{
  "device_type": "ENB/GNB/CPE",
  "order": 规则顺序,
  "move_to_group_id": "目标设备组ID",
  "enable": "0/1",
  "matching_mode": "deviceName/tac/lac",
  "name_contains": "设备名称匹配字符串",
  "tac_rag": "TAC/LAC范围",
  "nameRuleList": [
    {
      "condition": "contain/notContain/startWith/endWith",
      "value": "匹配值",
      "andOr": "and/or"
    }
  ]
}
```

### 5.3 编辑规则
```
POST /moveDeviceGroupRule/rule/update.action
Content-Type: application/json

参数: 同添加规则，额外包含 id 和 order
```

### 5.4 删除规则
```
POST /moveDeviceGroupRule/rule/del.action
参数:
  - id: 规则ID
```

### 5.5 启用/禁用规则
```
POST /moveDeviceGroupRule/rule/enable/update.action
参数:
  - id: 规则ID
  - enable: "0" 或 "1"
```

### 5.6 执行规则
```
POST /moveDeviceGroupRule/rule/execute.action
参数:
  - id: 规则ID
  - groupIds: 设备组ID列表 (逗号分隔)
```

### 5.7 更新规则顺序
```
POST /moveDeviceGroupRule/rule/order/update.action
Content-Type: application/json

参数:
[
  { "id": "规则ID", "order": 顺序 },
  ...
]
```

### 5.8 设备组下拉列表
```
POST /system/deviceGroup/getSimpleDeviceGroupList.action
参数:
  - isAll: "0"
```

### 5.9 设备组树结构
```
POST /system/deviceGroup/getFullDeviceGroupList.action
参数:
  - search_text: 搜索文本
  - type: 设备类型 (0=ENB, 1=GNB, 2=CPE)
```

---

## 6. 数据结构

### 6.1 规则表单数据 (addRuleForm)
```javascript
{
  move_to_group_id: '',    // 目标设备组ID
  enable: '0',             // 启用状态 (0/1)
  matching_mode: 'deviceName', // 匹配模式
  name_contains: '',       // 设备名称匹配 (已废弃，改用nameRuleList)
  tac_rag: ''              // TAC/LAC范围
}
```

### 6.2 名称规则列表 (nameContainsContentList)
```javascript
[
  {
    condition: 'contain',  // 过滤条件
    value: '',             // 匹配值
    andOr: 'and',          // 逻辑组合 (仅第2行及以后)
    hasError: false        // 错误标记
  }
]
```

### 6.3 规则列表数据 (ruleListTableData)
```javascript
[
  {
    id: '',                // 规则ID
    enable: '0/1',         // 启用状态
    operators: '',         // 操作描述
    create_time: '',       // 创建时间
    order: 1,              // 排序
    move_to_group_id: '',  // 目标设备组
    matching_mode: '',     // 匹配模式
    name_contains: '',     // 名称匹配
    tac_rag: '',           // TAC/LAC
    nameRuleList: []       // 名称规则列表
  }
]
```

---

## 7. 验证规则

### 7.1 TAC/LAC 验证
- 必填项 (当 matching_mode 为 tac 或 lac 时)
- 格式: `数字,数字,数字-数字`
- eNB TAC 范围: 0-65535
- gNB TAC 范围: 0-16777215
- LAC 范围: 0-65535

### 7.2 设备名称验证
- 当 enable=1 且 matching_mode=deviceName 时
- 每个条件输入框不能为空
- 最大长度: 64 字符
- 最大条件数: 10 条

---

## 8. 设备类型差异

| 功能 | eNB | gNB | CPE |
|------|-----|-----|-----|
| 设备名称匹配 | ✓ | ✓ | ✓ |
| LAC 匹配 | ✓ (支持GSM时) | - | - |
| TAC 匹配 | ✓ | ✓ | - |

---

## 9. 相关页面

| 页面 | 文件路径 | 说明 |
|------|----------|------|
| 设备注册 | eGW_regist.jsp | 规则配置入口页面 |
| 设备回收站 | deviceRecycleBin.jsp | 设备回收站管理 |
| 规则配置 | deviceAttrRule.jsp | 本页面 |

---

## 10. 功能清单汇总

| 序号 | 功能 | 操作类型 | 说明 |
|------|------|----------|------|
| 1 | 规则列表查询 | 查询 | 支持模糊搜索 |
| 2 | 添加规则 | 新增 | 配置匹配条件和目标设备组 |
| 3 | 编辑规则 | 修改 | 修改已有规则配置 |
| 4 | 删除规则 | 删除 | 确认后删除规则 |
| 5 | 启用/禁用规则 | 切换 | 控制规则是否生效 |
| 6 | 重新执行规则 | 执行 | 对选定设备组重新应用规则 |
| 7 | 拖拽排序 | 排序 | 调整规则执行优先级 |
| 8 | 多条件组合 | 配置 | And/Or 逻辑组合设备名称匹配 |
| 9 | TAC/LAC范围验证 | 验证 | 输入格式和范围验证 |
