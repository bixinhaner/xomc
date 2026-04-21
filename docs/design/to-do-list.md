# MML 需求实施清单

## 1. 新增"基本信息"分类及 DEVICE_INFO 命令

### 层级结构

```
分类（category） → 命令（command） → 子命令（sub_command）
基本信息            LST DEVICE_INFO    设备类型、运行时长、IP、MAC...
                    MOD DEVICE_INFO    MCC、MNC、BtsNum、Encryption...
```

每个子命令独立执行并返回结果，命令与子命令为 N:M 关系。

### 表结构分析

MML 模块共 9 张表，核心关联关系：

```
mml_commands (命令定义)
    ├── category VARCHAR(50) — 分类编号，字典翻译
    ├── command_code VARCHAR(100) UNIQUE — 如 "LST CELL"
    └── param_template JSONB — 参数定义 schema

mml_scripts → mml_tasks.script_id FK
mml_tasks → mml_audit_log.task_id FK
mml_templates → mml_commands.command_code
mml_param_versions → mml_param_groups → mml_params (via mml_group_param_rel)
```

### 新增表设计：命令与子命令 N:M

#### 新表 `mml_sub_commands` — 子命令定义

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | UUID | PK DEFAULT gen_random_uuid() | |
| `name` | VARCHAR(200) | NOT NULL | 显示名称，如"设备类型" |
| `code` | VARCHAR(100) | NOT NULL | 标识符，如"LTE_GSM_MODEL_NAME" |
| `tr069_path` | VARCHAR(500) | NOT NULL | TR-069 参数路径 |
| `description` | TEXT | | 描述 |
| `value_type` | VARCHAR(20) | NOT NULL DEFAULT 'string' | string/number/boolean/enum |
| `is_writable` | BOOLEAN | NOT NULL DEFAULT false | 是否可写（MOD 操作） |
| `options` | JSONB | DEFAULT '[]' | enum 类型的选项列表 |
| `unit` | VARCHAR(20) | | 单位 |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

#### 新表 `mml_command_subcommand_rel` — N:M 关联

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `command_id` | UUID | FK → mml_commands(id) ON DELETE CASCADE | |
| `subcommand_id` | UUID | FK → mml_sub_commands(id) ON DELETE CASCADE | |
| `sort_order` | INT | NOT NULL DEFAULT 0 | 显示排序 |
| PRIMARY KEY | | (command_id, subcommand_id) | |

### 完整 ER 关系

```
mml_commands (命令)
    │
    ├── 1:N ── mml_command_subcommand_rel (关联表)
    │               │
    │               └── N:1 ── mml_sub_commands (子命令)
    │
    └── 现有关联不变：mml_tasks、mml_templates、mml_audit_log 等
```

### 新增分类 "基本信息" (category='8')

**Step 1: 新增字典项**

```sql
INSERT INTO sys_dictionary_details (dictionary_id, value, label, sort_order, status)
SELECT id, '8', '基本信息', 80, 1
FROM sys_dictionaries WHERE type = 'mml_command_category';
```

**Step 2: 新增命令**

```sql
-- LST DEVICE_INFO
INSERT INTO mml_commands (id, command_name, command_code, category, description, rpc_method, operation_type, param_template, param_paths, supported_operations, product_types)
VALUES (
  gen_random_uuid(), '设备信息', 'LST DEVICE_INFO', '8',
  '查询设备基本信息', 'GetParameterValues', 'LST', '{}', '[]', '["LST"]', '["eNB", "gNB"]'
);

-- MOD DEVICE_INFO
INSERT INTO mml_commands (id, command_name, command_code, category, description, rpc_method, operation_type, param_template, param_paths, supported_operations, product_types)
VALUES (
  gen_random_uuid(), '设备信息', 'MOD DEVICE_INFO', '8',
  '修改设备基本信息', 'SetParameterValues', 'MOD', '{}', '[]', '["MOD"]', '["eNB", "gNB"]'
);
```

**Step 3: 新增子命令（14 条，7 条可写）**

```sql
INSERT INTO mml_sub_commands (id, name, code, tr069_path, description, value_type, is_writable, options) VALUES
-- 只读子命令
('00000001-0000-0000-0000-000000000001', '设备类型',   'LTE_GSM_MODEL_NAME',   'Device.DeviceInfo.X_COM_MODULE_TYPE',           '设备型号/模块类型',      'string',  false, '[]'),
('00000001-0000-0000-0000-000000000002', '运行时长',   'LTE_GSM_SYS_TIME',     'Device.DeviceInfo.X_COM_STATION_RUN_Time',       '设备运行时长',           'string',  false, '[]'),
('00000001-0000-0000-0000-000000000003', 'IP地址',     'LTE_GSM_IP',           'Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress', '设备IP地址',         'string',  false, '[]'),
('00000001-0000-0000-0000-000000000004', 'MAC地址',    'LTE_GSM_MAC',          'Device.DeviceInfo.X_COM_MACAddress',             '设备MAC地址',            'string',  false, '[]'),
('00000001-0000-0000-0000-000000000005', '软件版本',   'LTE_GSM_SOFTWARE',     'Device.DeviceInfo.SoftwareVersion',              '软件版本号',             'string',  false, '[]'),
('00000001-0000-0000-0000-000000000006', '硬件版本',   'LTE_GSM_HARDWARE',     'Device.DeviceInfo.HardwareVersion',              '硬件版本号',             'string',  false, '[]'),
('00000001-0000-0000-0000-000000000007', 'MME状态',    'LTE_GSM_MME_STATUS',   'Device.DeviceInfo.X_COM_MME_Status',             'MME连接状态',            'string',  false, '[]'),
-- 可写子命令（同时属于 LST 和 MOD）
('00000001-0000-0000-0000-000000000008', 'MCC',        'DEVICEGSM_MCC',        'DeviceGSM.Mcc',                                  '移动国家代码',           'string',  true,  '[]'),
('00000001-0000-0000-0000-000000000009', 'MNC',        'DEVICEGSM_MNC',        'DeviceGSM.Mnc',                                  '移动网络代码',           'string',  true,  '[]'),
('00000001-0000-0000-0000-000000000010', 'BtsNum',     'LTE_BTSNUM',           'DeviceGSM.BtsNum',                               '基站数量',               'number',  true,  '[]'),
('00000001-0000-0000-0000-000000000011', 'Encryption',  'BSC_ENCRYPTION',      'DeviceGSM.Encryption',                           '加密方式',               'enum',    true,  '[{"label":"不加密","value":0},{"label":"A5/1","value":1},{"label":"A5/3","value":3}]'),
('00000001-0000-0000-0000-000000000012', 'TimerNetT3212','DEVICEGSM_TIMERNETT3212','DeviceGSM.TimerNetT3212',                      'T3212定时器(秒)',        'number',  true,  '[]'),
('00000001-0000-0000-0000-000000000013', 'NriBitLen',  'DEVICEGSM_NRIBITLEN',  'DeviceGSM.NriBitLen',                            'NRI比特长度',            'number',  true,  '[]'),
('00000001-0000-0000-0000-000000000014', 'NriNullAdd', 'DEVICEGSM_NRINULLADD', 'DeviceGSM.NriNullAdd',                           'NRI空地址',              'number',  true,  '[]');
```

**Step 4: 建立命令-子命令 N:M 关联**

```sql
-- LST DEVICE_INFO 关联全部 14 个子命令
INSERT INTO mml_command_subcommand_rel (command_id, subcommand_id, sort_order)
SELECT c.id, s.id, s.code
FROM mml_commands c, mml_sub_commands s
WHERE c.command_code = 'LST DEVICE_INFO'
ORDER BY s.code;

-- MOD DEVICE_INFO 关联 7 个可写子命令
INSERT INTO mml_command_subcommand_rel (command_id, subcommand_id, sort_order)
SELECT c.id, s.id, s.code
FROM mml_commands c, mml_sub_commands s
WHERE c.command_code = 'MOD DEVICE_INFO'
  AND s.is_writable = true
ORDER BY s.code;
```

### 前端命令树展示结构

```
基本信息 (category='8')
├── 设备信息 (LST DEVICE_INFO)
│   ├── 设备类型    LTE_GSM_MODEL_NAME    Device.DeviceInfo.X_COM_MODULE_TYPE
│   ├── 运行时长    LTE_GSM_SYS_TIME       Device.DeviceInfo.X_COM_STATION_RUN_Time
│   ├── IP地址      LTE_GSM_IP             Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress
│   ├── MAC地址     LTE_GSM_MAC            Device.DeviceInfo.X_COM_MACAddress
│   ├── 软件版本    LTE_GSM_SOFTWARE       Device.DeviceInfo.SoftwareVersion
│   ├── 硬件版本    LTE_GSM_HARDWARE       Device.DeviceInfo.HardwareVersion
│   ├── MME状态     LTE_GSM_MME_STATUS     Device.DeviceInfo.X_COM_MME_Status
│   ├── MCC         DEVICEGSM_MCC          DeviceGSM.Mcc
│   ├── MNC         DEVICEGSM_MNC          DeviceGSM.Mnc
│   ├── BtsNum      LTE_BTSNUM             DeviceGSM.BtsNum
│   ├── Encryption  BSC_ENCRYPTION         DeviceGSM.Encryption
│   ├── TimerNetT3212 DEVICEGSM_TIMERNETT3212 DeviceGSM.TimerNetT3212
│   ├── NriBitLen   DEVICEGSM_NRIBITLEN    DeviceGSM.NriBitLen
│   └── NriNullAdd  DEVICEGSM_NRINULLADD   DeviceGSM.NriNullAdd
└── 设备信息 (MOD DEVICE_INFO)
    ├── MCC         DEVICEGSM_MCC          DeviceGSM.Mcc
    ├── MNC         DEVICEGSM_MNC          DeviceGSM.Mnc
    ├── BtsNum      LTE_BTSNUM             DeviceGSM.BtsNum
    ├── Encryption  BSC_ENCRYPTION         DeviceGSM.Encryption
    ├── TimerNetT3212 DEVICEGSM_TIMERNETT3212 DeviceGSM.TimerNetT3212
    ├── NriBitLen   DEVICEGSM_NRIBITLEN    DeviceGSM.NriBitLen
    └── NriNullAdd  DEVICEGSM_NRINULLADD   DeviceGSM.NriNullAdd
```

### 实施步骤

1. 创建迁移文件新增 `mml_sub_commands` 和 `mml_command_subcommand_rel` 两张表
2. 创建 seed 迁移插入字典项、命令、子命令及关联数据
3. 后端新增 `SubCommandRepository` 接口和 `PgSubCommandRepository` 实现
4. 后端 `ListCommands` 接口扩展返回子命令列表（JOIN 查询）
5. 前端命令树组件支持三级展示（分类 > 命令 > 子命令）
6. 子命令执行时独立生成 TR-069 GetParameterValues/SetParameterValues 请求

---

## 2. MML 控制台多语言修复

已完成。修复的文件和内容：

| 文件 | 修复内容 |
|------|---------|
| `CommandInput.tsx` | `'Control Panel'` → `t('mml.console.controlPanel')`, `'ParameterPath Command'` → `t('mml.console.parameterPathCommand')` |
| `ParamPathPanel.tsx` | `'Operation Type'` → `t('mml.console.operationType')`, `'Parameter Path'` → `t('mml.console.parameterPath')` |
| `TerminalPanel.tsx` | 6 处硬编码中文替换为 i18n key |
| `DeviceTree.tsx` | 4 处硬编码中文替换为 i18n key |
| `ParamFormRenderer.tsx` | `'该命令无需配置参数'` → `t('mml.console.noParamsNeeded')` |
| `BatchSnModal.tsx` | `'批量输入设备SN'` → `t('common.batchInputDeviceSN')` |

新增 11 个 i18n key（zh-CN + en-US 各一套）。

---

## 3. mml_templates 创建 500 错误修复

已完成。

**根因**: `pg_repository.go` 中 `templateColumns` 切片缺少 `"category_group"` 列（12 项），而 `scanTemplate` 扫描 13 个字段（含 `CategoryGroup`），导致 `RETURNING` 子句返回 12 列但 Go scanner 期望 13 个 destination。

**修复**: 在 `templateColumns` 中补充 `"category_group"` 字段。
