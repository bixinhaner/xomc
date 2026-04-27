# KPI 指标管理页面分析

> 来源文件：`omcmb/original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/pm/perfmgmt/kpi_management.jsp`

## 页面整体布局

页面分为左右两部分：
- **左侧面板**：指标功能集树形结构（240px 宽度）
- **右侧面板**：指标列表数据表格

---

## 右侧面板 - 数据表格字段

| 序号 | 字段名 | 显示名称 | 宽度 | 说明 |
|------|--------|----------|------|------|
| 0 | ck | 复选框 | - | 用于批量选择指标 |
| 1 | operation | 操作 | 85px | 操作按钮区域 |
| 2 | isEnable | 测量 | 50px | 是否启用测量，值：是/否 |
| 3 | kpiId | Counter/指标ID | 100px | 指标唯一标识 |
| 4 | kpiName | Counter/指标名称 | 140px | 指标名称，有冲突时显示警告图标 |
| 5 | product_type | 产品类型 | 90px | 产品类型分类 |
| 6 | custName | 自定义指标名称 | 140px | 用户自定义的指标名称 |
| 7 | indicatorLevel | 等级 | 40px | Device 或 PLMN |
| 8 | unit | 单位 | 50px | 指标单位 |
| 9 | isCustomize | 指标类型 | 80px | 基础指标 / 自定义指标 |
| 10 | updater | 更新人 | 50px | 最后更新人 |
| 11 | updateTime | 更新时间 | 80px | 最后更新时间 |

### 操作列功能说明

根据指标类型显示不同的操作按钮：

| 场景 | 查看按钮 | 修改按钮 | 删除按钮 |
|------|----------|----------|----------|
| 基础指标 | ✅ | ✅ | ❌ |
| 自定义指标(counter类型) | ✅ | ✅ | ✅ |
| 自定义指标(非counter类型) | ✅ | ✅ | ✅ |

---

## 工具栏功能

### 1. 搜索功能
- **输入框**：支持按「指标名称」或「指标ID」搜索
- **触发方式**：点击搜索图标触发 `kpiArithmeticTreeLoad()`

### 2. 产品类型筛选（仅 eNB 网元）
- **组件**：`el-popfilter` 单选下拉
- **选项**：全部、动态获取的产品类型列表
- **触发**：`productChange()` 刷新树和表格

### 3. 等级筛选（仅 eNB 网元）
- **组件**：`el-popfilter` 单选下拉
- **选项**：
  - 全部
  - Device
  - PLMN
- **触发**：`levelChange()` 刷新树和表格

### 4. 已选功能（仅 eNB 网元）
- **显示**：已选中的指标数量 `( N )`
- **点击展开**：显示已选指标列表弹窗
  - 列表字段：指标ID
  - 支持单个删除、清空全部

### 5. 批量操作按钮（仅 eNB 网元）

| 按钮 | 图标 | 功能 |
|------|------|------|
| 测量 | el-icon-KPI-Meas | 启用选中指标的测量 |
| 取消测量 | el-icon-operation-CancelMeasure | 禁用选中指标的测量 |

---

## 右上角功能按钮

| 按钮 | 功能 |
|------|------|
| 添加 (+) | 打开新建指标公式页面 `goAddKpiArithmeticWin()` |
| 导出 | 导出所有指标信息 `exportKpiInfo()` |

---

## 网元类型适配

页面支持三种网元类型，通过 `currentKpiNetType` 判断：

| 网元类型 | API 前缀 | 产品类型筛选 | 等级筛选 | 已选功能 |
|----------|----------|--------------|----------|----------|
| eNB (4G) | `/pm/` | ✅ 显示 | ✅ 显示 | ✅ 显示 |
| gNB (5G) | `/gnb/pm/` | ❌ 隐藏 | ❌ 隐藏 | ❌ 隐藏 |
| eGW | `/egw/pm/` | ❌ 隐藏 | ❌ 隐藏 | ❌ 隐藏 |

---

## 弹窗组件

### 1. 重名提示弹窗
- **触发**：指标名称冲突时点击警告图标
- **字段**：
  - 指标ID（只读显示）
  - 指标名称（输入框，最多50字符）
- **操作**：确定 / 取消

### 2. 确认取消测量弹窗
- **触发**：批量取消测量时
- **内容**：
  - 确认提示文字
  - 被模板关联的指标列表（如有）
- **操作**：确定 / 取消

---

## 数据表格配置

```javascript
{
  idField: 'kpiId',           // 唯一标识字段
  singleSelect: false,         // 支持多选
  rownumbers: true,           // 显示行号
  pagination: true,           // 启用分页
  fitColumns: true,           // 自动适应列宽
  striped: true,              // 斑马纹样式
  checkOnSelect: false,       // 点击行不自动勾选
  selectOnCheck: true         // 勾选自动选中行
}
```

---

## API 接口

### 获取指标列表

| 网元 | 接口 |
|------|------|
| eNB | `GET /pm/indicatormg/getIndicatorListByPage.action` |
| gNB | `GET /gnb/pm/indicatormg/getIndicatorListPageData.action` |
| eGW | `GET /egw/pm/indicatormg/getIndicatorListPageData.action` |

### 请求参数

| 参数 | 类型 | 说明 |
|------|------|------|
| catagoryId | string | 功能集ID |
| searchText | string | 搜索关键词 |
| timeZone | string | 时区 |
| product_type | string | 产品类型（仅eNB） |
| indicatorLevel | string | 等级（仅eNB） |

### 批量操作接口

| 功能 | 接口 |
|------|------|
| 启用测量 | `POST /cell/perfmgmt/kpimanage/enableIndicator.action` |
| 禁用测量 | `POST /cell/perfmgmt/kpimanage/disableIndicator.action` |
| 检查模板关联 | `GET /cell/perfmgmt/kpimanage/isIndicatorInTemplate.action` |

---

## 权限控制

- 添加按钮：`CODE_PERFORMANCE_MANAGEMENT`
- 修改按钮：`CODE_PERFORMANCE_MANAGEMENT`
- 删除按钮：`CODE_PERFORMANCE_MANAGEMENT`

无权限时按钮通过 `hidden` class 隐藏。

---

## 新建指标页面分析

> 来源文件：`omcmb/original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/pm/perfmgmt/kpi_addArithmetic.jsp`

### 页面结构

页面分为两个主要区块：
1. **基本信息** - 指标基本属性配置
2. **计算公式** - 仅 KPI 类型显示，用于构建指标计算表达式

---

### 基本信息字段

| 字段名 | 显示名称 | 类型 | 长度限制 | 必填 | 显示条件 | 说明 |
|--------|----------|------|----------|------|----------|------|
| indicatorType | 类型 | 单选 | - | 否 | 有权限时显示 | KPI / Counter |
| indicatorLevel | 等级 | 单选 | - | 否 | 仅eNB显示 | Device / PLMN |
| kpiName | 指标名称 | 文本 | 50 | ✅ | 始终显示 | 不能为空 |
| custName | 自定义名称 | 文本 | 50 | 否 | 仅Counter显示 | Custom Name |
| catagoryId | 所属功能集 | 下拉 | - | ✅ | 始终显示 | eNB用树形，其他用普通下拉 |
| unit | 单位 | 下拉 | - | ✅ | 始终显示 | 预设单位选项 |
| statisType | 统计类型 | 下拉 | - | ✅ | 始终显示 | 影响公式校验规则 |
| isEnable | 测量 | 下拉 | - | ✅ | 仅eNB显示 | 是/否 |
| definition | 说明 | 多行文本 | 2000 | 否 | 始终显示 | 支持换行 |

---

### 类型选择与页面变化

#### 1. 类型切换 (indicatorType)

| 类型值 | 显示内容 | 隐藏内容 |
|--------|----------|----------|
| **kpi** (默认) | 计算公式区域 | Custom Name 字段 |
| **counter** | Custom Name 字段 | 计算公式区域 |

```javascript
// 类型切换逻辑
if(indicatorType != 'counter') {
    $('#kpi_param_part_add').show();    // 显示计算公式区域
    $('#kpi_cus_name_div').hide();      // 隐藏自定义名称
} else {
    $('#kpi_param_part_add').hide();    // 隐藏计算公式区域
    $('#kpi_cus_name_div').show();      // 显示自定义名称
}
```

#### 2. 等级切换 (indicatorLevel) - 仅 eNB

| 等级值 | 影响 |
|--------|------|
| device | 设备级指标 |
| plmn | PLMN级指标 |

等级切换时会：
- 清空计算公式
- 重新加载指标功能集树

#### 3. 网元类型适配

| 网元类型 | 类型选择 | 等级选择 | 测量字段 | 功能集组件 |
|----------|----------|----------|----------|------------|
| eNB (4G) | ✅ 显示 | ✅ 显示 | ✅ 显示 | combotree (树形) |
| gNB (5G) | ❌ 隐藏 | ❌ 隐藏 | ❌ 隐藏 | combobox (普通) |
| eGW | ❌ 隐藏 | ❌ 隐藏 | ❌ 隐藏 | combobox (普通) |

---

### 计算公式区域

#### 运算符按钮

| 按钮 | 功能 |
|------|------|
| + | 加法 |
| - | 减法 |
| * | 乘法 |
| / | 除法 |
| ( | 左括号 |
| ) | 右括号 |
| 0-9 | 数字输入 |
| Duration | 持续时间变量 |
| 清除 | 清空整个公式 |

#### 指标选择区域

- **左侧**：指标功能集树形结构 (`kpiSetTree`)
- **右侧**：性能指标列表 (`kpiListDatagrid`)
- **搜索框**：支持按「指标名称 / 指标ID」搜索
- **产品类型筛选**：仅 eNB，选择后过滤可选指标

---

### 字段校验规则

| 字段 | 校验规则 | 错误提示 |
|------|----------|----------|
| 指标名称 | 不能为空 | 指标名称为空 |
| 指标名称 | 最大50字符 | - |
| 所属功能集 | 必选 | - |
| 单位 | 必选 | - |
| 统计类型 | 必选 | - |
| 测量 | 必选(仅eNB) | - |
| 说明 | 最大2000字符 | - |
| 计算公式 | 必填(KPI类型) | 计算公式为空 |
| 计算公式 | 百分比类型必须包含除法 | 公式必须包含除法 |
| 产品类型交集 | 已选指标产品类型必须一致 | 已选指标的产品类型不一致 |

---

### 提交参数

#### KPI 类型参数

```javascript
{
    "kpiName": "指标名称",
    "catagoryId": "功能集ID",
    "unit": "单位",
    "statisType": "统计类型",
    "definition": "说明",
    "arithmetic": "计算公式",
    "product_type": "产品类型(仅eNB)",
    "isEnable": "是否启用(仅eNB)",
    "indicatorLevel": "等级(仅eNB)"
}
```

#### Counter 类型参数

```javascript
{
    "kpiName": "指标名称",
    "kpiCustomName": "自定义名称",
    "catagoryId": "功能集ID",
    "unit": "单位",
    "statisType": "统计类型",
    "definition": "说明",
    "indicatorType": "counter",
    "custName": "自定义名称"
}
```

---

### 提交接口

| 网元类型 | 接口地址 |
|----------|----------|
| eNB | `POST /pm/indicatormg/addOrModifyIndicator.action` |
| gNB | `POST /gnb/pm/indicatormg/addOrModifyIndicator.action` |
| eGW | `POST /egw/pm/indicatormg/addOrModifyIndicator.action` |

---

### 交互流程

1. **打开页面**
   - 加载功能集下拉选项
   - 加载单位下拉选项
   - 加载统计类型下拉选项
   - (eNB) 加载测量下拉选项
   - (eNB) 加载产品类型列表

2. **选择类型**
   - KPI: 显示计算公式区域
   - Counter: 显示自定义名称字段

3. **选择功能集**
   - 清空计算公式
   - 重新加载可选指标树

4. **构建公式** (KPI类型)
   - 点击运算符按钮添加符号
   - 从指标树/列表选择指标添加到公式

5. **提交保存**
   - 校验所有必填字段
   - 校验公式有效性
   - 发送请求保存
