# 中国移动TD-LTE皮站/飞站基站设备南向接口数据配置模型规范 V2.3

> 基于规范文件自动解析生成，按目录索引分组，命令智能拆分
>
> - 命令拆分规则：按路径公共前缀最小化划分，{i}代表数组实例
> - 父级命令不含子级路径（如 DeviceInfo.* 不含 SwUpgrade.* 子路径）
> - 每个路径唯一归属一个最精确命令，无重复映射
> - 数字实例编号（如 A1MeasureCtrl.1/2/3）已归一化为 {i} 模板并去重
> - 📖=只读(R)  📝=可读可写(RW)

**参数总数**: 625（去重后唯一模板参数）
**分组数**: 18
**命令总数**: 73

---

## 原始需求（MML 控制台命令树重构 · 2026-05-19）

> 本节是 MML 控制台（前端路径 `/mml/console`）"标准命令树"使用本规范的实施合约。
> SA…SR 范围内的命令分组、命令、路径、权限，以**本文档**为唯一权威源。
> 任何对下方 `## 目录索引` / `#### 命令:` / 路径权限的修改，都必须等价反映到 MML 控制台。
> 本节描述实施需求，不修改下方协议解析内容。

### R-1 命令树结构（**两层：object 组 → operation 叶子**）

- 命令树**只有一级分组**（object 组），不再有 SA…SR 章节级分组。
- 一级分组节点 = 本文档每个 `#### 命令: <路径模板>` 标题对应的"对象"；
  - 显示名 = **对象的中文别名**（如 `设备信息` / `设备版本升级` / `当前告警实例` / ...）；
  - 路径模板（如 `Device.DeviceInfo.*` / `Device.DeviceInfo.SwUpgrade.*`）作为副标题或 tooltip。
- 一级分组**互为平级**，与原"目录索引"章节归属无关（例：原属 SA 的"设备信息"与"设备版本升级"现在是同级，原属 SD 的多个 FaultMgmt 子对象也是同级）。
- 一级分组排序遵循本文档 SA → SR 出现顺序（保持运营商规范的阅读 locality，但不渲染章节名）。
- **章节信息 (SA…SR) 仅作为元数据**保留在导入产物里（用于排序 + 管理后台 / 报表），**不进入用户可见命令树**。
- 一级分组的**稳定标识 (group_code)** = TR-181 路径模板归一化后的字符串（例 `Device.DeviceInfo.SwUpgrade.*`），用于幂等导入。
- 一级分组**不可被用户修改/删除**（`catalog_protected = true`），仅由本规范文档的导入流程更新。

### R-2 命令叶子（**二级，operation 维度展开**）

- 每个一级分组下，按 **TR-069 标准 + 权限矩阵** 展开为 1-4 个**操作叶子**，每个叶子是一条独立的可执行命令：
  - **LST `<Object>`**（恒有，只要路径集非空）
  - **MOD `<Object>`**（仅当路径集中存在 ≥ 1 条 RW 路径 📝 时存在）
  - **ADD `<Object>`**（仅当路径模板含 `{i}` 占位符 **且** 路径集中存在 ≥ 1 条 RW 路径 时存在；read-only 事件表如 CurrentAlarm/ExpeditedEvent **不展开 ADD**）
  - **RMV `<Object>`**（与 ADD 同条件）
- 每个叶子的**显示名**：`<OP> <对象中文别名>` 或 `<OP> <对象英文路径末段>`（如 `LST 设备信息` / `MOD ManagementServer`）。
- 每个叶子是 DB 中一条独立的 `mml_commands` 行，**operation_type 固化**（不可在运行时切换）：
  - `(source='standard', group_id, operation_type)` 三元组唯一；
  - `target_paths`（该操作下要执行的 path 集合）**按操作维度独立**（详见 R-3）。
- 命令叶子拆分原则（与生成器口径一致）：
  - 按路径公共前缀**最小化**划分（父级 group 不含子级 group 路径，例 `Device.DeviceInfo.*` group 的叶子集合**不含** `Device.DeviceInfo.SwUpgrade.*`）；
  - 数字实例编号（如 `A1MeasureCtrl.1/2/3`）归一化为 `{i}` 模板并去重；
  - 多层实例占位符 `{iα}` / `{iβ}` / `{iγ}` / `{iδ}` / `{iε}` 一律折叠回 `{i}`，实例号在运行时由用户填写区分。

### R-3 各操作的 path 集与 RPC 映射

**操作类型在 catalog 加载期就被固化**，每条命令的 `target_paths` 内容**仅与该操作类型对应的 TR-069 标准语义有关**，不依赖运行时选择：

| 操作 | TR-069 RPC | `target_paths` 内容（针对同一 object） | 何时生成命令行 |
|------|-----------|---------------------------------------|---------------|
| **LST** | `GetParameterValues` | object 路径集**全集**（含 📖 RO + 📝 RW） | 路径集非空 → 恒生成 |
| **MOD** | `SetParameterValues` | object 路径集中**仅 📝 RW 子集** | 至少 1 条 📝 → 生成 |
| **ADD** | `AddObject` | 仅**父级对象路径**（如 `Device.X.{i}.`）；子参数初值取 RW 字段 | 路径含 `{i}` **且** 路径集存在 RW → 生成 |
| **RMV** | `DeleteObject` | 仅**实例路径**（`Device.X.{i}.`）；运行时用 InstancePicker 选实例号 | 同 ADD |

> 这意味着同一 object（如 `Device.DeviceInfo.*` 📖📝）在 DB 中存在**两条** `mml_commands` 行（一条 LST 全 17 path，一条 MOD 只含 2 条 📝 path）；
> `Device.FaultMgmt.CurrentAlarm.{i}.* 📖`（全 RO 事件表）只产生**一条 LST**；
> `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{iβ}.* 📝`（writable 多实例）产生**四条**（LST / MOD / ADD / RMV）。

控制台只展示**已实际生成的命令行**；不存在的操作类型在树上不出现（无需置灰）。

### R-4 操作面板（Control Panel）行为（按命令的固定 operation_type 单一渲染）

> **不再有"操作切换条"**。用户点选哪条命令叶子，右侧操作面板就只渲染那个 operation_type 的视图。
> 想换操作（LST → MOD），用户回到左侧命令树点另一个叶子即可。

#### R-4.1 LST

- 默认勾选命令路径集中的**全部 path**；
- 用户可在 Control Panel **取消勾选**部分 path；
- 提交时对**勾选后的路径**调用 `GetParameterValues`；
- 若命令含 `{i}`，Control Panel 顶部提供"实例索引输入区"，支持：
  - 留空 = 查询整个 partial path（CPE 返回所有实例，对应 TR-069 "partial path" 语义）；
  - 单值（如 `1`）；
  - 范围（如 `1,2,5`）—— 前端拆解为多条独立路径。
- 多层 `{i}`：按从外到内顺序逐级填写。

#### R-4.2 MOD

- Control Panel 仅列出命令路径集中**RW (📝) 路径**；只读路径不出现；
- 默认勾选全部 RW 路径；用户可取消勾选；
- 每条勾选路径需要在右侧填**新值**输入框；
- 提交时只对**勾选 + 已填值**的路径调用 `SetParameterValues`；
- 若命令含 `{i}`，必须先指定具体实例号（不允许留空）；
- 字段值的类型 / 取值范围按本文档表格 "类型" 列约束（如 `unsignedInt[0:65535]` / `boolean` / `string(64)`），前端做软校验，后端走 ParamModel `MappingValidator` 硬校验。

#### R-4.3 ADD

- 仅当命令路径含 `{i}` 时可用；
- Control Panel 列出该对象下的**RW (📝) 子参数**作为新实例的初始值字段，勾选规则同 MOD；
- 提交时调用 `AddObject(parentPath)` 创建新实例，再用 `SetParameterValues` 写入勾选字段的初始值（同一会话内）。

#### R-4.4 RMV

- 仅当命令路径含 `{i}` 时可用；
- Control Panel 提供"实例选择器"：列出设备当前已存在的实例索引（由前置 LST 或缓存提供）；用户勾选要删除的实例（支持多选）；
- 提交时对每个选中实例调用 `DeleteObject(instancePath)`。

### R-5 Customized 分组（公私模板权限模型）

- 控制台命令树固定保留**一个名为 "Customized" 的一级分组**，**不在本文档 SA…SR 范围内**；与标准 object 组同级，**固定排在所有标准 object 组之后**（CommandTree 末尾）；
- Customized 下有**两个二级容器**：`Public Template`（公有模板）与 `Private Template`（私有模板）；
- Customized 命令的增删改由前端 "+" 入口维护（commit `2849f65d`），与本文档导入流程**互不影响**（`source='admin'`）。

#### R-5.1 公有模板（Public Template）

- 任意**已登录用户**均可见、均可使用；
- 创建后由所有用户共享，删除需要 admin 或创建者权限（见 D26）；
- **结构约束**：`Public Template` 下**所有命令完全平级**（一级展开即所有命令叶子），**不允许子文件夹 / 命令分类等任何嵌套**；新增/未来扩展同样禁止深层嵌套。
- DB 标识：`source='admin', visibility='public'`，`owner_user_id` 仍记录创建人但不参与可见性过滤。

#### R-5.2 私有模板（Private Template）

- **结构始终包含"用户名目录"这一层**（无论登录身份是否 admin）：`Private Template > <username> > <command-1, command-2, ...>`；
- **普通用户**视角：Private Template 下**只显示自己一个用户名目录**（owner = 当前登录用户）；展开即看到自己的命令清单；
- **admin 角色**视角：Private Template 下**显示所有用户名目录**（含 admin 自己），可逐个展开查看任意用户的私有命令；
- **每个用户名目录下命令完全平级**（一级展开即叶子），同 R-5.1 不允许深层嵌套；
- DB 标识：`source='admin', visibility='private', owner_user_id=<creator user_id>`；
- 后端 BuildTree API 强制按当前登录用户 + 角色做可见性过滤，并按 owner_user_id 聚合返回。

#### R-5.3 树结构（用户视角对比）

```
普通用户 alice 视角：                    admin 用户视角：
─────────────────────────              ──────────────────────────────────
... (所有标准 object 组在上方)          ... (所有标准 object 组在上方)
设备信息                                设备信息
  LST 设备信息                            LST 设备信息
  MOD 设备信息                            MOD 设备信息
设备版本升级                            设备版本升级
  LST 设备版本升级                        LST 设备版本升级
...                                     ...
（标准 object 组结束后）                （标准 object 组结束后）

Customized （末尾固定）                  Customized （末尾固定）
├─ Public Template                      ├─ Public Template
│   ├─ <cmd-by-anyone-1>                │   ├─ <cmd-by-anyone-1>
│   ├─ <cmd-by-anyone-2>                │   ├─ <cmd-by-anyone-2>
│   └─ <cmd-by-anyone-3>                │   └─ <cmd-by-anyone-3>
│       (完全平级)                       │       (完全平级)
└─ Private Template                     └─ Private Template
    └─ alice                                ├─ alice
        ├─ <my-cmd-1>                       │   ├─ <cmd-1>
        ├─ <my-cmd-2>                       │   └─ <cmd-2>
        └─ <my-cmd-3>                       ├─ bob
            (alice 自己的命令，平级)         │   └─ <cmd-1>
                                            ├─ charlie
                                            │   └─ <cmd-1>
                                            └─ Admin
                                                ├─ <cmd-1>
                                                ├─ <cmd-2>
                                                └─ <cmd-3>
                                                (admin 用户自己的私有命令)
```

> **关键点（与早期版本的区别）**：
> 1. **Customized 在 CommandTree 末尾**（不再是置顶）
> 2. **Public Template 命令完全平级**，永不出现下级文件夹
> 3. **Private Template 始终有用户名目录层**：普通用户只显示自己一个目录；admin 显示所有人的目录（admin 自己也作为一个用户出现）

### R-10 UI 文案与组件命名约定

| 位置 | 旧名 / 历史命名 | 新名（中文） | 新名（英文 i18n） |
|------|----------------|-------------|------------------|
| RightPanel 主 Tab | Control Panel | **操作面板** | **Control Panel** |
| RightPanel 副 Tab | Param Path Expert | **参数路径指定** | **ParameterPath Command** |

- 副 Tab 重命名同时考虑前端组件文件 `ParamPathExpert.tsx` 是否物理 rename → `ParameterPathCommand.tsx`（见 D28）；
- i18n key 需补：`mml.tabs.controlPanel`、`mml.tabs.parameterPathCommand`（zh-CN / en-US）；
- 所有文档、注释、PR 描述统一使用新名。

### R-6 命令树更新与权威源

- 本文档是 SA…SR 范围内命令分组 / 命令路径 / 参数 / 权限的**唯一权威源**；
- 实施侧须提供一个**幂等导入流程**，与现有 `param_models / products / indicators / alarms` 4 个 dictloader 对齐（参考整合方案 §5）：
  - **生产运行时不依赖任何脚本语言**（Go binary + 数据文件 + DB 即可启动 / 重载）；
  - "MD → 结构化数据文件 (JSON / XML)" 的转换由**离线开发期工具**完成（Python 或其他实现，**不进生产镜像**）；结构化文件随代码提交、随 Go binary 部署；
  - 启动期由 Go `mml-catalog` Loader 幂等 UPSERT 入库；admin API（如 `POST /api/v1/mml/catalog/reload`）可触发热重载，无需新增 SQL 迁移；
- 该流程必须保证：
  1. 本文档新增/删除一行 `####` 命令 → 离线工具重新生成结构化文件 → 重启 / 热重载后命令树同步；
  2. 路径权限（📖 / 📝 / 📖📝）变更 → 该 group 名下的 per-op `mml_commands` 行集**按权限矩阵重算**（自动增删 LST/MOD/ADD/RMV 行）；
  3. **不破坏** Customized 分组及用户已创建的自定义命令（按 `source='admin'` 隔离）；
  4. **不重置** `mml_command_sub_fields` 的"默认勾选"等已被 admin 调整过的用户视角配置（通过 override 表保护）；
  5. 删除（regression）操作仅在文档明确移除时生效，且仅删除 `source='standard'` 的行。

### R-7 非目标

- **不涉及** 电信 / 联通规范导入（CTCC / CUCC 命令树由后续独立立项处理，保留现有数据不动）；
- **不修改** Customized 分组的交互逻辑；
- **不修改** ParamModel 字典 standardPath ↔ privatePath 翻译机制（仍由 `internal/config/parammodel/` Translator 处理）。

### R-8 设备列表的产品类型约束（强约束）

> 背景：不同产品类型对应不同的 Product 装配件（含 ParamModel / IndicatorPlatform / AlarmNeType），命令路径集与权限矩阵可能完全不同。若允许跨产品类型批量执行，会出现部分设备路径不存在 / 权限不匹配 / 翻译失败的发散后果。
> 该约束适用于 MML 控制台的"设备选择列表"（前端 `Console/components/DeviceTree.tsx` 上方的产品类型筛选器）。
>
> **命名约定（CRITICAL）**：
> - "**产品类型**" 是用户可见的中文术语；
> - 数据库 / 后端 / TR-069 标准命名为 **`product_class`**（取 TR-069 标准参数 `Device.DeviceInfo.ProductClass`，即 CPE 上报的厂商产品标识，如 `BLQ-Q436` / `Nova430E`），devices 表对应列 = `product_class`；
> - 前端历史代码中的 `productType` 字段（如 `ConsoleDevice.productType`、`useDictionary('product_type')`）实质就是 `product_class` 的视图层别名，命名不一致是历史遗留，详见调整方案文档 D20；
> - 本节文字使用"产品类型"指代用户可见概念；指代具体字段时统一用 **`product_class`**（DB / API）/ `productClass`（JSON 驼峰）。

- **R-8.1 默认值**：页面首次进入时，自动取产品类型字典（`useDictionary('product_type')`，字典 code 历史遗留，**字典 value 实际是 `product_class`**）的**第一项**作为 `productClassFilter` 初值，不再保持空值。
- **R-8.2 必选**：产品类型下拉框**取消 `allowClear`**，不允许置空；若产品类型字典为空，禁用整个 DeviceTree 并提示"请先在系统管理 → 字典维护中配置产品类型"。
- **R-8.3 单一性**：同一次命令执行只能在**同一 `product_class`** 下进行。切换产品类型时：
  1. 已勾选的设备清空；
  2. 命令树右侧已配置的 statements 不清空（用户可继续配置）；
  3. 弹 toast 提示"产品类型已切换，已清空选中设备"。
- **R-8.4 后端兜底**：`POST /mml/console/execute-statements` 入参 `device_sns`，在后端**按 SN 查 `devices.product_class` 列分组校验**，若**包含 ≥ 2 种 `product_class`** 直接返回 400 错误码（新增 `ErrDevicesMixedProductClass`），不下发任务。该校验独立于前端约束，作为最后一道防线。
- **R-8.5 与命令兼容性提示（可选 P2）**：当用户已选中某 `product_class` + 命令，但该 `product_class` 经 `ProductRegistry` 解析到的 `product → param_model` 不包含该命令路径时，命令节点旁显示警告图标 + Tooltip"该产品类型不支持本命令"（不阻塞，仅提示；执行时由后端 Translator 缺失映射时报错）。

### R-9 命令执行 API 重构（直传 path + 服务端翻译）

> 背景：当前 MML 控制台采用"前端渲染 MML 文本 → 后端解析 MML → 抽取 path"的双向 round-trip 模式。该模式存在两个问题：
> ① Control Panel 同时展示 MML 文本与勾选面板，两处状态可能不一致；
> ② 客户端对 standardPath 解析与服务端不一致（如实例号格式、空格转义）会导致执行偏差。
> 本期把"执行什么"完全交给 Control Panel 下的勾选 / 填值面板，API 入参改为结构化路径数组，由服务端按设备产品类型做 standardPath ↔ privatePath 翻译。

#### R-9.1 移除 Control Panel 的 MML 文本预览

- **删除** `RightPanel` 内 `MmlEditor` 组件（命令文本框 + "生成 MML" + "执行" 按钮）；
- Control Panel 不再展示"将执行哪些 path"的预览文本；**以 SubFieldChecklist / SubFieldInputList / InstancePicker 当前勾选与填值状态作为唯一真相来源**；
- 执行按钮迁移至 Control Panel 底部一行（建议布局：`[ 校验 ] [ 全选 ] [ 全不选 ] [ 执行 <OP> ]`）；
- **TerminalPanel 保留**，但其内容从"用户编辑的 MML 文本"切换为"服务端实际下发后的 RPC 摘要 + 响应"（执行后才有内容）；
- 本条**仅作用于 `/mml/console`**（控制台主流程）；MML 脚本任务页 (`/mml/script-task` 与 `Console/components` 之外的 `mml/components/CommandCodeTextarea.tsx`、`ScriptTaskDrawer.tsx`) **不在本期改动范围**，脚本任务仍可使用 MML 文本通道。

#### R-9.2 API 入参重构

`POST /mml/console/execute-statements` 入参由"MML 文本 statements"重构为**结构化路径数组**：

```json
{
  "device_sns": ["SN001", "SN002"],
  "statements": [
    {
      "command_id": "<uuid>",
      "command_code": "Device.DeviceInfo.*",
      "operation_type": "LST" | "MOD" | "ADD" | "RMV",
      "paths": [
        "Device.DeviceInfo.UserLabel",
        "Device.DeviceInfo.SoftwareVersion"
      ],
      "values": {
        "Device.DeviceInfo.UserLabel": "myDevice"
      },
      "instance_selectors": { "iα": "1", "iβ": "2" },
      "instance_indices": [3, 5, 7]
    }
  ]
}
```

- `paths`：**全部使用 standardPath**（与本规范"TR-181 路径"列一致）；
- `values`：MOD/ADD 时填，key 必须为 `paths` 子集；
- `instance_selectors`：`{i}` 占位符填值（含多层 `{iα}/{iβ}/{iγ}` 等）；LST 模式允许部分留空 = partial path；
- `instance_indices`：RMV 多选场景（与 `instance_selectors` 二选一）；
- **旧的 `POST /mml/console/render` / `POST /mml/console/parse`** 接口在 Console 主流程中**不再调用**，过渡期保留 1 个 release 供 Customized / 脚本场景使用，之后从 Console 链路移除。

#### R-9.3 服务端翻译 standardPath → privatePath（依设备产品类型）

入队 device_task 前，对**每个目标设备**调用 ParamModel Translator（契约见《参数-KPI-告警-整合设计方案》§1.6）：

```
TranslateToPrivate(ctx, productId, softwareVersion, standardPath) → TranslationResult
```

查找顺序：
1. **discovered**：`discovered_param_mappings WHERE product_id=? AND software_version=?`（精确）
2. **default**：`param_mappings WHERE param_model_id=?`（默认映射，通过 product.param_model_id 关联）
3. **passthrough**：均未命中 → **透传 standardPath**，不阻塞执行；ACS RPC 后续若返回"参数不存在"由协议层处理

`TranslationResult` 携带 `source: discovered | default | passthrough`，逐 path 写入 device_task 执行日志（可观测性 / 排查必备）。

#### R-9.4 多设备同 statement 的独立翻译

- R-8.4 已保证同次执行的 device_sns 同 `product_class`（同 `devices.product_class`），但 `devices.software_version` 可能差异 → 不同设备的 discovered 映射可能不同；
- 因此每个 device_task 在入队时**独立翻译**一次，不做"一份 RPC payload 多设备共用"的批量优化；
- ACS 侧 GPV/SPV 响应回流时同样由 Translator 反向翻译回 standardPath 落库（与整合方案 §1.11 Path B 一致）。
- 解析链路：`devices.product_class` → `ProductRegistry.MatchProductClass()` → `products.id` (productID) → `Translator.TranslateToPrivate(productID, softwareVersion, standardPath)`。

#### R-9.5 Customized 命令的执行通道

- 默认 Customized 私有 / 公共命令**也走结构化 API**（前提：Customized 命令在 admin 编辑时已绑定 commandCode + sub_fields，commit `2849f65d` 的 AddTemplateModal 已具备）；
- 若现网存在 Customized "裸 MML 高级用法"（用户在文本中手写 SetParameterValues 等），由 D15 决策（§12）确认是否保留单独的 MML 文本通道。

---

| 索引 | 参数管理类别 | 说明 |
|------|-------------|------|
| SA | DeviceInfo | 设备信息参数管理 |
| SB | SoftwareCtrl | 软件版本参数管理 |
| SC | ManagementServer | 基站网管参数管理 |
| SD | FaultMgmt | 告警参数管理 |
| SE | DeviceLogMgmt | 日志参数管理 |
| SF | Services.FAPService | 小区服务参数管理（总体） |
| SG | Services.FAPService.{i}.SCTP.Transport | SCTP参数管理 |
| SH | Services.FAPService.{i}.CellConfig.LTE.RAN | RAN协议栈参数 |
| SI | Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList | 邻区参数管理 |
| SJ | Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility | 移动性参数管理 |
| SK | Services.FAPService.{i}.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam | SON参数管理 |
| SL | WANDevice | WAN口配置参数管理 |
| SM | Ipsec | IPsec参数管理 |
| SN | Time | 时间服务器参数管理 |
| SO | FAP.GPS | GPS信息参数管理 |
| SP | FAP.MRMgmt | MR参数管理 |
| SQ | FAP.PerfMgmt | 性能参数管理 |
| SR | ENanocell | 扩展型一体化皮基站参数 |

---

## SA - DeviceInfo

**设备信息参数管理**  
参数数量: 20

#### 命令: Device.DeviceInfo.* 📖📝

只读(📖): 15 | 可写(📝): 2

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.UserLabel` | `Device.DeviceInfo.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
| 2 | `InternetGatewayDevice.DeviceInfo.DnPrefix` | `Device.DeviceInfo.DnPrefix` | DnPrefix | DN前缀 | 📝 RW | string |
| 3 | `InternetGatewayDevice.DeviceInfo.ManufacturerOUI` | `Device.DeviceInfo.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | 📖 R | string(6) |
| 4 | `InternetGatewayDevice.DeviceInfo.Manufacturer` | `Device.DeviceInfo.Manufacturer` | Manufacturer | 制造商 | 📖 R | string(64) |
| 5 | `InternetGatewayDevice.DeviceInfo.ModelName` | `Device.DeviceInfo.ModelName` | ModelName | 设备型号 | 📖 R | string(64) |
| 6 | `InternetGatewayDevice.DeviceInfo.SerialNumber` | `Device.DeviceInfo.SerialNumber` | SerialNumber | 序列号 | 📖 R | string(64) |
| 7 | `InternetGatewayDevice.DeviceInfo.HardwareVersion` | `Device.DeviceInfo.HardwareVersion` | HardwareVersion | 硬件版本 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.DeviceInfo.SoftwareVersion` | `Device.DeviceInfo.SoftwareVersion` | SoftwareVersion | 软件版本 | 📖 R | string(64) |
| 9 | `InternetGatewayDevice.DeviceInfo.HardwarePlatform` | `Device.DeviceInfo.HardwarePlatform` | HardwarePlatform | 硬件平台 | 📖 R | string(64) |
| 10 | `InternetGatewayDevice.DeviceInfo.AdditionalHardwareVersion` | `Device.DeviceInfo.AdditionalHardwareVersion` | AdditionalHardwareVersion | 附加硬件版本 | 📖 R | string(64) |
| 11 | `InternetGatewayDevice.DeviceInfo.AdditionalSoftwareVersion` | `Device.DeviceInfo.AdditionalSoftwareVersion` | AdditionalSoftwareVersion | 附加软件版本 | 📖 R | string(64) |
| 12 | `InternetGatewayDevice.DeviceInfo.ProvisioningCode` | `Device.DeviceInfo.ProvisioningCode` | ProvisioningCode | 供应商代号 | 📖 R | string(64) |
| 13 | `InternetGatewayDevice.DeviceInfo.ProductClass` | `Device.DeviceInfo.ProductClass` | ProductClass | 产品分类 | 📖 R | string(64) |
| 14 | `InternetGatewayDevice.DeviceInfo.UpTime` | `Device.DeviceInfo.UpTime` | UpTime | 运行时间 | 📖 R | unsignedInt |
| 15 | `InternetGatewayDevice.DeviceInfo.3GPPSpecVersion` | `Device.DeviceInfo.3GPPSpecVersion` | 3GPPSpecVersion | 3GPP协议版本 | 📖 R | string |
| 16 | `InternetGatewayDevice.DeviceInfo.FirstUseDate` | `Device.DeviceInfo.FirstUseDate` | FirstUseDate | 首次使用日期 | 📖 R | dateTime |
| 17 | `InternetGatewayDevice.DeviceInfo.DataModelSpecVersion` | `Device.DeviceInfo.DataModelSpecVersion` | DataModelSpecVersion | 数据模型版本 | 📖 R | string |

#### 命令: Device.DeviceInfo.SwUpgrade.* 📖

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.SwUpgrade.Stage` | `Device.DeviceInfo.SwUpgrade.Stage` | Stage | 升级阶段 | 📖 R | unsignedInt[1:5] |
| 2 | `InternetGatewayDevice.DeviceInfo.SwUpgrade.FailureCause` | `Device.DeviceInfo.SwUpgrade.FailureCause` | FailureCause | 升级失败原因 | 📖 R | string(512) |
| 3 | `InternetGatewayDevice.DeviceInfo.SwUpgrade.Status` | `Device.DeviceInfo.SwUpgrade.Status` | Status | 状态 | 📖 R | unsignedInt[1:3] |

---

## SB - SoftwareCtrl

**软件版本参数管理**  
参数数量: 5

#### 命令: Device.SoftwareCtrl.* 📖📝

只读(📖): 2 | 可写(📝): 3

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.SoftwareCtrl.AutoActivateEnable` | `Device.SoftwareCtrl.AutoActivateEnable` | AutoActivateEnable | 立即激活目标升级版本使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.SoftwareCtrl.ActivateTime` | `Device.SoftwareCtrl.ActivateTime` | ActivateTime | 软件激活时间 | 📝 RW | dateTime |
| 3 | `InternetGatewayDevice.SoftwareCtrl.ActivateEnable` | `Device.SoftwareCtrl.ActivateEnable` | ActivateEnable | 激活备份版本使能开关 | 📝 RW | boolean |
| 4 | `InternetGatewayDevice.SoftwareCtrl.SystemCurrentVersion` | `Device.SoftwareCtrl.SystemCurrentVersion` | SystemCurrentVersion | 系统当前版本 | 📖 R | string(64) |
| 5 | `InternetGatewayDevice.SoftwareCtrl.SystemBackupVersion` | `Device.SoftwareCtrl.SystemBackupVersion` | SystemBackupVersion | 系统备份版本 | 📖 R | string(64) |

---

## SC - ManagementServer

**基站网管参数管理**  
参数数量: 19

#### 命令: Device.ManagementServer.* 📖📝

只读(📖): 4 | 可写(📝): 15

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.ManagementServer.URL` | `Device.ManagementServer.URL` | URL | 网管服务器URL | 📝 RW | string(256) |
| 2 | `InternetGatewayDevice.ManagementServer.Username` | `Device.ManagementServer.Username` | Username | 连接用户名 | 📝 RW | string(256) |
| 3 | `InternetGatewayDevice.ManagementServer.Password` | `Device.ManagementServer.Password` | Password | 连接密码 | 📝 RW | string(256) |
| 4 | `InternetGatewayDevice.ManagementServer.PeriodicInformEnable` | `Device.ManagementServer.PeriodicInformEnable` | PeriodicInformEnable | 连接使能开关 | 📝 RW | boolean |
| 5 | `InternetGatewayDevice.ManagementServer.PeriodicInformTime` | `Device.ManagementServer.PeriodicInformTime` | PeriodicInformTime | 上报周期 | 📝 RW | dateTime |
| 6 | `InternetGatewayDevice.ManagementServer.PeriodicInformInterval` | `Device.ManagementServer.PeriodicInformInterval` | PeriodicInformInterval | 周期上报时间间隔 | 📝 RW | unsignedInt[1:] |
| 7 | `InternetGatewayDevice.ManagementServer.ParameterKey` | `Device.ManagementServer.ParameterKey` | ParameterKey | 键值 | 📖 R | string(32) |
| 8 | `InternetGatewayDevice.ManagementServer.ConnectionRequestURL` | `Device.ManagementServer.ConnectionRequestURL` | ConnectionRequestURL | 连接请求URL | 📖 R | string(256) |
| 9 | `InternetGatewayDevice.ManagementServer.ConnectionRequestUsername` | `Device.ManagementServer.ConnectionRequestUsername` | ConnectionRequestUsername | 连接请求用户名 | 📝 RW | string(256) |
| 10 | `InternetGatewayDevice.ManagementServer.ConnectionRequestPassword` | `Device.ManagementServer.ConnectionRequestPassword` | ConnectionRequestPassword | 连接请求密码 | 📝 RW | string(256) |
| 11 | `InternetGatewayDevice.ManagementServer.UDPConnectionRequestAddress` | `Device.ManagementServer.UDPConnectionRequestAddress` | UDPConnectionRequestAddress | UDP连接请求地址 | 📖 R | string(256) |
| 12 | `InternetGatewayDevice.ManagementServer.STUNEnable` | `Device.ManagementServer.STUNEnable` | STUNEnable | STUN使能开关 | 📝 RW | boolean |
| 13 | `InternetGatewayDevice.ManagementServer.STUNServerAddress` | `Device.ManagementServer.STUNServerAddress` | STUNServerAddress | STUN服务器地址 | 📝 RW | string(256) |
| 14 | `InternetGatewayDevice.ManagementServer.STUNServerPort` | `Device.ManagementServer.STUNServerPort` | STUNServerPort | STUN服务器端口 | 📝 RW | unsignedInt[0:65535] |
| 15 | `InternetGatewayDevice.ManagementServer.STUNUsername` | `Device.ManagementServer.STUNUsername` | STUNUsername | STUN用户名 | 📝 RW | string(256) |
| 16 | `InternetGatewayDevice.ManagementServer.STUNPassword` | `Device.ManagementServer.STUNPassword` | STUNPassword | STUN密码 | 📝 RW | string(256) |
| 17 | `InternetGatewayDevice.ManagementServer.STUNMaximumKeepAlivePeriod` | `Device.ManagementServer.STUNMaximumKeepAlivePeriod` | STUNMaximumKeepAlivePeriod | STUN最大生存周期 | 📝 RW | int[-1:65535] |
| 18 | `InternetGatewayDevice.ManagementServer.STUNMinimumKeepAlivePeriod` | `Device.ManagementServer.STUNMinimumKeepAlivePeriod` | STUNMinimumKeepAlivePeriod | STUN最小生存周期 | 📝 RW | unsignedInt[0:65535] |
| 19 | `InternetGatewayDevice.ManagementServer.NATDetected` | `Device.ManagementServer.NATDetected` | NATDetected | NAT转换地址 | 📖 R | boolean |

---

## SD - FaultMgmt

**告警参数管理**  
参数数量: 55

#### 命令: Device.FaultMgmt.* 📖

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FaultMgmt.SupportedAlarmNumberOfEntries` | `Device.FaultMgmt.SupportedAlarmNumberOfEntries` | SupportedAlarmNumberOfEntries | 最大告警实例数量 | 📖 R | unsignedInt |
| 2 | `InternetGatewayDevice.FaultMgmt.MaxCurrentAlarmEntries` | `Device.FaultMgmt.MaxCurrentAlarmEntries` | MaxCurrentAlarmEntries | 最大当前告警实例数量 | 📖 R | unsignedInt |
| 3 | `InternetGatewayDevice.FaultMgmt.CurrentAlarmNumberOfEntries` | `Device.FaultMgmt.CurrentAlarmNumberOfEntries` | CurrentAlarmNumberOfEntries | 当前告警实例数量 | 📖 R | unsignedInt |
| 4 | `InternetGatewayDevice.FaultMgmt.HistoryEventNumberOfEntries` | `Device.FaultMgmt.HistoryEventNumberOfEntries` | HistoryEventNumberOfEntries | 历史事件实例数量 | 📖 R | unsignedInt |
| 5 | `InternetGatewayDevice.FaultMgmt.ExpeditedEventNumberOfEntries` | `Device.FaultMgmt.ExpeditedEventNumberOfEntries` | ExpeditedEventNumberOfEntries | 紧急事件实例数量 | 📖 R | unsignedInt |
| 6 | `InternetGatewayDevice.FaultMgmt.QueuedEventNumberOfEntries` | `Device.FaultMgmt.QueuedEventNumberOfEntries` | QueuedEventNumberOfEntries | 队列事件实例数量 | 📖 R | unsignedInt |

#### 命令: Device.FaultMgmt.CurrentAlarm.{i}.* 📖

- **{i}** 取值范围: `0~N` — 当前告警实例编号，N由MaxCurrentAlarmEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.AlarmIdentifier` | `Device.FaultMgmt.CurrentAlarm.{i}.AlarmIdentifier` | AlarmIdentifier | 该告警的唯一序列号 | 📖 R | string(20) |
| 2 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.AlarmRaisedTime` | `Device.FaultMgmt.CurrentAlarm.{i}.AlarmRaisedTime` | AlarmRaisedTime | 告警发生时间 | 📖 R | dateTime |
| 3 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.AlarmChangedTime` | `Device.FaultMgmt.CurrentAlarm.{i}.AlarmChangedTime` | AlarmChangedTime | 告警改变时间 | 📖 R | dateTime |
| 4 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.FaultLocation` | `Device.FaultMgmt.CurrentAlarm.{i}.FaultLocation` | FaultLocation | 告警源定位信息 | 📖 R | string(512) |
| 5 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.ManagedObjectInstance` | `Device.FaultMgmt.CurrentAlarm.{i}.ManagedObjectInstance` | ManagedObjectInstance | 管理对象实例 | 📖 R | string(512) |
| 6 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.EventType` | `Device.FaultMgmt.CurrentAlarm.{i}.EventType` | EventType | 告警类型 | 📖 R | string(64) |
| 7 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.ProbableCause` | `Device.FaultMgmt.CurrentAlarm.{i}.ProbableCause` | ProbableCause | 告警原因 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.SpecificProblem` | `Device.FaultMgmt.CurrentAlarm.{i}.SpecificProblem` | SpecificProblem | 告警描述 | 📖 R | string(128) |
| 9 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.PerceivedSeverity` | `Device.FaultMgmt.CurrentAlarm.{i}.PerceivedSeverity` | PerceivedSeverity | 当前告警级别 | 📖 R | string |
| 10 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.AdditionalText` | `Device.FaultMgmt.CurrentAlarm.{i}.AdditionalText` | AdditionalText | 告警附加文本 | 📖 R | string(256) |
| 11 | `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation` | `Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation` | AdditionalInformation | 告警附加信息 | 📖 R | string(256) |

#### 命令: Device.FaultMgmt.ExpeditedEvent.{i}.* 📖

- **{i}** 取值范围: `0~N` — 实时告警实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.EventTime` | `Device.FaultMgmt.ExpeditedEvent.{i}.EventTime` | EventTime | 告警发生、改变、清除的时间 | 📖 R | dateTime |
| 2 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.AlarmIdentifier` | `Device.FaultMgmt.ExpeditedEvent.{i}.AlarmIdentifier` | AlarmIdentifier | 该告警的唯一序列号 | 📖 R | string(20) |
| 3 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.NotificationType` | `Device.FaultMgmt.ExpeditedEvent.{i}.NotificationType` | NotificationType | 通知类型 | 📖 R | string |
| 4 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.FaultLocation` | `Device.FaultMgmt.ExpeditedEvent.{i}.FaultLocation` | FaultLocation | 告警源定位信息 | 📖 R | string(512) |
| 5 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.ManagedObjectInstance` | `Device.FaultMgmt.ExpeditedEvent.{i}.ManagedObjectInstance` | ManagedObjectInstance | 管理对象实例 | 📖 R | string(512) |
| 6 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.EventType` | `Device.FaultMgmt.ExpeditedEvent.{i}.EventType` | EventType | 告警类型 | 📖 R | string(64) |
| 7 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.ProbableCause` | `Device.FaultMgmt.ExpeditedEvent.{i}.ProbableCause` | ProbableCause | 告警原因 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.SpecificProblem` | `Device.FaultMgmt.ExpeditedEvent.{i}.SpecificProblem` | SpecificProblem | 告警描述 | 📖 R | string(128) |
| 9 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.PerceivedSeverity` | `Device.FaultMgmt.ExpeditedEvent.{i}.PerceivedSeverity` | PerceivedSeverity | 告警级别 | 📖 R | string |
| 10 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.AdditionalText` | `Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalText` | AdditionalText | 告警附加文本 | 📖 R | string(256) |
| 11 | `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.AdditionalInformation` | `Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalInformation` | AdditionalInformation | 告警附加信息 | 📖 R | string(256) |

#### 命令: Device.FaultMgmt.HistoryEvent.{i}.* 📖

- **{i}** 取值范围: `0~N` — 历史告警实例编号，N由HistoryEventNumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.EventTime` | `Device.FaultMgmt.HistoryEvent.{i}.EventTime` | EventTime | 告警发生、改变、清除的时间 | 📖 R | dateTime |
| 2 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.AlarmIdentifier` | `Device.FaultMgmt.HistoryEvent.{i}.AlarmIdentifier` | AlarmIdentifier | 该告警的唯一序列号 | 📖 R | string(20) |
| 3 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.NotificationType` | `Device.FaultMgmt.HistoryEvent.{i}.NotificationType` | NotificationType | 通知类型 | 📖 R | string |
| 4 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.FaultLocation` | `Device.FaultMgmt.HistoryEvent.{i}.FaultLocation` | FaultLocation | 告警源定位信息 | 📖 R | string(512) |
| 5 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.ManagedObjectInstance` | `Device.FaultMgmt.HistoryEvent.{i}.ManagedObjectInstance` | ManagedObjectInstance | 管理对象实例 | 📖 R | string(512) |
| 6 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.EventType` | `Device.FaultMgmt.HistoryEvent.{i}.EventType` | EventType | 告警类型 | 📖 R | string(64) |
| 7 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.ProbableCause` | `Device.FaultMgmt.HistoryEvent.{i}.ProbableCause` | ProbableCause | 告警原因 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.SpecificProblem` | `Device.FaultMgmt.HistoryEvent.{i}.SpecificProblem` | SpecificProblem | 告警描述 | 📖 R | string(128) |
| 9 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.PerceivedSeverity` | `Device.FaultMgmt.HistoryEvent.{i}.PerceivedSeverity` | PerceivedSeverity | 告警级别 | 📖 R | string |
| 10 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.AdditionalText` | `Device.FaultMgmt.HistoryEvent.{i}.AdditionalText` | AdditionalText | 告警附加文本 | 📖 R | string(256) |
| 11 | `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.AdditionalInformation` | `Device.FaultMgmt.HistoryEvent.{i}.AdditionalInformation` | AdditionalInformation | 告警附加信息 | 📖 R | string(256) |

#### 命令: Device.FaultMgmt.QueuedEvent.{i}.* 📖

- **{i}** 取值范围: `0~N` — 队列告警实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.EventTime` | `Device.FaultMgmt.QueuedEvent.{i}.EventTime` | EventTime | 告警发生、改变、清除的时间 | 📖 R | dateTime |
| 2 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.AlarmIdentifier` | `Device.FaultMgmt.QueuedEvent.{i}.AlarmIdentifier` | AlarmIdentifier | 该告警的唯一序列号 | 📖 R | string(20) |
| 3 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.NotificationType` | `Device.FaultMgmt.QueuedEvent.{i}.NotificationType` | NotificationType | 通知类型 | 📖 R | string |
| 4 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.FaultLocation` | `Device.FaultMgmt.QueuedEvent.{i}.FaultLocation` | FaultLocation | 告警源定位信息 | 📖 R | string(512) |
| 5 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.ManagedObjectInstance` | `Device.FaultMgmt.QueuedEvent.{i}.ManagedObjectInstance` | ManagedObjectInstance | 管理对象实例 | 📖 R | string(512) |
| 6 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.EventType` | `Device.FaultMgmt.QueuedEvent.{i}.EventType` | EventType | 告警类型 | 📖 R | string(64) |
| 7 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.ProbableCause` | `Device.FaultMgmt.QueuedEvent.{i}.ProbableCause` | ProbableCause | 告警原因 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.SpecificProblem` | `Device.FaultMgmt.QueuedEvent.{i}.SpecificProblem` | SpecificProblem | 告警描述 | 📖 R | string(128) |
| 9 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.PerceivedSeverity` | `Device.FaultMgmt.QueuedEvent.{i}.PerceivedSeverity` | PerceivedSeverity | 告警级别 | 📖 R | string |
| 10 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.AdditionalText` | `Device.FaultMgmt.QueuedEvent.{i}.AdditionalText` | AdditionalText | 告警附加文本 | 📖 R | string(256) |
| 11 | `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.AdditionalInformation` | `Device.FaultMgmt.QueuedEvent.{i}.AdditionalInformation` | AdditionalInformation | 告警附加信息 | 📖 R | string(256) |

#### 命令: Device.FaultMgmt.SupportedAlarm.{i}.* 📖📝

- **{i}** 取值范围: `0~N` — 告警实例编号，N由SupportedAlarmNumberOfEntries决定

只读(📖): 4 | 可写(📝): 1

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FaultMgmt.SupportedAlarm.{i}.EventType` | `Device.FaultMgmt.SupportedAlarm.{i}.EventType` | EventType | 告警类型 | 📖 R | string(64) |
| 2 | `InternetGatewayDevice.FaultMgmt.SupportedAlarm.{i}.ProbableCause` | `Device.FaultMgmt.SupportedAlarm.{i}.ProbableCause` | ProbableCause | 告警原因 | 📖 R | string(64) |
| 3 | `InternetGatewayDevice.FaultMgmt.SupportedAlarm.{i}.SpecificProblem` | `Device.FaultMgmt.SupportedAlarm.{i}.SpecificProblem` | SpecificProblem | 告警描述 | 📖 R | string(128) |
| 4 | `InternetGatewayDevice.FaultMgmt.SupportedAlarm.{i}.PerceivedSeverity` | `Device.FaultMgmt.SupportedAlarm.{i}.PerceivedSeverity` | PerceivedSeverity | 支持告警级别 | 📖 R | string |
| 5 | `InternetGatewayDevice.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism` | `Device.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism` | ReportingMechanism | 告警上报机制 | 📝 RW | string |

---

## SE - DeviceLogMgmt

**日志参数管理**  
参数数量: 5

#### 命令: Device.LogMgmt.* 📝

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.LogMgmt.PeriodicUploadEnable` | `Device.LogMgmt.PeriodicUploadEnable` | PeriodicUploadEnable | 日志周期上传使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.LogMgmt.URL` | `Device.LogMgmt.URL` | URL | 日志上传URL | 📝 RW | string(256) |
| 3 | `InternetGatewayDevice.LogMgmt.Username` | `Device.LogMgmt.Username` | Username | 日志管理用户名 | 📝 RW | string(256) |
| 4 | `InternetGatewayDevice.LogMgmt.Password` | `Device.LogMgmt.Password` | Password | 日志管理密码 | 📝 RW | string(256) |
| 5 | `InternetGatewayDevice.LogMgmt.PeriodicUploadInterval` | `Device.LogMgmt.PeriodicUploadInterval` | PeriodicUploadInterval | 日志周期上传时间间隔 | 📝 RW | unsignedInt[1:] |

---

## SF - Services.FAPService

**小区服务参数管理（总体）**  
参数数量: 58

#### 命令: Device.Services.FAPControl.LTE.* 📖📝

只读(📖): 2 | 可写(📝): 1

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPControl.LTE.AdminState` | `Device.Services.FAPControl.LTE.AdminState` | AdminState | 基站管理状态 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPControl.LTE.OpState` | `Device.Services.FAPControl.LTE.OpState` | OpState | 基站运行状态 | 📖 R | boolean |
| 3 | `InternetGatewayDevice.Services.FAPControl.LTE.RFTxStatus` | `Device.Services.FAPControl.LTE.RFTxStatus` | RFTxStatus | 基站射频状态 | 📖 R | boolean |

#### 命令: Device.Services.FAPControl.LTE.Gateway.* 📝

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.SecGWServer1` | `Device.Services.FAPControl.LTE.Gateway.SecGWServer1` | SecGWServer1 | 安全网关1 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.SecGWServer2` | `Device.Services.FAPControl.LTE.Gateway.SecGWServer2` | SecGWServer2 | 安全网关2 | 📝 RW | string(64) |
| 3 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.SecGWServer3` | `Device.Services.FAPControl.LTE.Gateway.SecGWServer3` | SecGWServer3 | 安全网关3 | 📝 RW | string(64) |
| 4 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.AGServerEnable` | `Device.Services.FAPControl.LTE.Gateway.AGServerEnable` | AGServerEnable | HeGW方式接入 | 📝 RW | boolean |
| 5 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.AGServerIp1` | `Device.Services.FAPControl.LTE.Gateway.AGServerIp1` | AGServerIp1 | 接入网关1 | 📝 RW | string(64) |
| 6 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.AGServerIp2` | `Device.Services.FAPControl.LTE.Gateway.AGServerIp2` | AGServerIp2 | 接入网关2 | 📝 RW | string(64) |
| 7 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.AGServerIp3` | `Device.Services.FAPControl.LTE.Gateway.AGServerIp3` | AGServerIp3 | 接入网关3 | 📝 RW | string(64) |
| 8 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.AGPort1` | `Device.Services.FAPControl.LTE.Gateway.AGPort1` | AGPort1 | 接入网关端口1 | 📝 RW | string(64) |
| 9 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.AGPort2` | `Device.Services.FAPControl.LTE.Gateway.AGPort2` | AGPort2 | 接入网关端口2 | 📝 RW | string(64) |
| 10 | `InternetGatewayDevice.Services.FAPControl.LTE.Gateway.AGPort3` | `Device.Services.FAPControl.LTE.Gateway.AGPort3` | AGPort3 | 接入网关端口3 | 📝 RW | string(64) |

#### 命令: Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.* 📖📝

- **{i}** 取值范围: `0~N` — MME池配置实例编号，N由对应NumberOfEntries决定

只读(📖): 3 | 可写(📝): 2

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPControl.LTE.MmePoolConfigParam.{i}.PLMNIDList` | `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.PLMNID` | PLMNIDList | MMEPLMN标识 | 📖 R | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEGroupID` | `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEGroupID` | MMEGroupID | MMEGroup标识 | 📖 R | unsignedInt[0:65535] |
| 3 | `InternetGatewayDevice.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMECode` | `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMECode` | MMECode | MME代码 | 📖 R | unsignedInt[0:255] |
| 4 | `InternetGatewayDevice.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp1` | `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp1` | MMEIp1 | MMEIP1 | 📝 RW | string(64) |
| 5 | `InternetGatewayDevice.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp2` | `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp2` | MMEIp2 | MMEIP2 | 📝 RW | string(64) |

#### 命令: Device.Services.FAPControl.LTE.S1U.{i}.* 📖

- **{i}** 取值范围: `0~N` — S1U实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPControl.LTE.S1U.{i}.LocIpAddrList` | `Device.Services.FAPControl.LTE.S1U.{i}.LocIpAddrList` | LocIpAddrList | 本地IP地址列表 | 📖 R | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPControl.LTE.S1U.{i}.FarIpSubnetworkList` | `Device.Services.FAPControl.LTE.S1U.{i}.FarIpSubnetworkList` | FarIpSubnetworkList | 远端IP地址列表 | 📖 R | string(64) |

#### 命令: Device.Services.FAPControl.X2IpAddrMapInfo.{i}.* 📝

- **{i}** 取值范围: `0~N` — X2接口IP映射实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPControl.X2IpAddrMapInfo.{i}.PLMNIDList` | `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.PLMNID` | PLMNIDList | X2PLMN标识 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbType` | `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbType` | EnbType | 基站类型 | 📝 RW | unsignedInt |
| 3 | `InternetGatewayDevice.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbId` | `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbId` | EnbId | eNBID | 📝 RW | unsignedInt |
| 4 | `InternetGatewayDevice.Services.FAPControl.X2IpAddrMapInfo.{i}.WanIpAddress` | `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.WanIpAddress` | WanIpAddress | WAN口IP | 📝 RW | string(64) |
| 5 | `InternetGatewayDevice.Services.FAPControl.X2IpAddrMapInfo.{i}.SubnetMask` | `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.SubnetMask` | SubnetMask | 子网掩码 | 📝 RW | string(64) |

#### 命令: Device.Services.FAPService.{i}.* 📖📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

只读(📖): 2 | 可写(📝): 22

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred` | CellBarred | 小区闭塞 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState` | AdminState | 小区管理状态 | 📝 RW | boolean |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.OpState` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.OpState` | OpState | 小区运行状态 | 📖 R | boolean |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.MaxUEsServed` | `Device.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.MaxUEsServed` | SupportRRCNumbers | 同时支持RRC连接最大用户数 | 📖 R | int[-1:] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1` | `Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1` | MultiBandInfoListSIB1 | SIB1中多频段指示参数 | 📝 RW | unsignedInt[1:64] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5` | `Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5` | MultiBandInfoListSIB5 | SIB5中多频段指示参数 | 📝 RW | unsignedInt[1:64] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList` | RouteIndexList | 路由指示列表 | 📝 RW | string(512) |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList` | RuList | RU列表 | 📝 RW | string(512) |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL` | EARFCNDL | 下行EARFCN | 📝 RW | string(128) |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID` | PhyCellID | PCI | 📝 RW | string(512) |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth` | DLBandwidth | 下行带宽 | 📝 RW | string(32) |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth` | ULBandwidth | 上行带宽 | 📝 RW | string(32) |
| 14 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset` | PSCHPowerOffset | PSCH功率偏置 | 📝 RW | string(512) |
| 15 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset` | SSCHPowerOffset | SSCH功率偏置 | 📝 RW | string(512) |
| 16 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset` | PBCHPowerOffset | PBCH功率偏置 | 📝 RW | string(512) |
| 17 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL` | EARFCNUL | 上行EARFCN | 📝 RW | string(128) |
| 18 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator` | FreqBandIndicator | 频带指示 | 📝 RW | int[1:40] |
| 19 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower` | ReferenceSignalPower | 参考信号功率 | 📝 RW | string(512) |
| 20 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity` | CellIdentity | 小区标识 | 📝 RW | unsignedInt[0:268435455] |
| 21 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType` | EnbType | 基站类型 | 📝 RW | unsignedInt[0:1] |
| 22 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul` | `Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul` | SPSSwitchQCI1Ul | QCI1上行SPS功能开关 | 📝 RW | string(64) |
| 23 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl` | `Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl` | CASwitchUl | 上行载波聚合功能开关 | 📝 RW | string(64) |
| 24 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl` | `Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl` | CASwitchDl | 下行载波聚合功能开关 | 📝 RW | string(64) |

#### 命令: Device.Services.FAPService.{i}.Capabilities.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported` | `Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported` | NNSFSupported | 负载均衡参数 | 📝 RW | boolean |

#### 命令: Device.Services.FAPService.{i}.CellConfig.Capabilities.* 📖📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

只读(📖): 2 | 可写(📝): 1

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.UeInactiveTimer` | `Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.UeInactiveTimer` | UeInactiveTimer | 连接态UE不活动定时器 | 📝 RW | unsignedInt[0:65535] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.SupportActiveRRCNumbers` | `Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.SupportActiveRRCNumbers` | SupportActiveRRCNumbers | 同时支持RRC激活最大用户数 | 📖 R | unsignedInt[0:65535] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.MaxTxPower` | `Device.Services.FAPService.{i}.CellConfig.Capabilities.MaxTxPower` | MaxTxPower | 最大发射功率 | 📖 R | unsignedInt |

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.EPC.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID` | `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID` | EAID | EPCEAID | 📝 RW | unsignedInt[0:16777216] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC` | `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC` | TAC | TAC | 📝 RW | unsignedInt[0:65535] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.EPC.PLMNList.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — PLMN列表实例编号，N由PLMNListNumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID` | `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID` | PLMNID | 小区PLMNID | 📝 RW | string(6) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse` | `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse` | CellReservedForOperatorUse | 小区预留标识 | 📝 RW | boolean |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.VoLTE.PdcpInitParam.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — PDCP初始参数实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.{i}.RohcEn` | `Device.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.{i}.RohcEn` | RohcEn | 分QCI头压缩功能开关 | 📝 RW | string(64) |

---

## SG - Services.FAPService.{i}.SCTP.Transport

**SCTP参数管理**  
参数数量: 13

#### 命令: Device.Services.FAPControl.Transport.SCTP.* 📝

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.Enable` | `Device.Services.FAPControl.Transport.SCTP.Enable` | Enable | 使能开关 | 📝 RW | Boolean |
| 2 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.RTOInitial` | `Device.Services.FAPControl.Transport.SCTP.RTOInitial` | RTOInitial | 重传超时的初始值 | 📝 RW | unsignedInt |
| 3 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.RTOMin` | `Device.Services.FAPControl.Transport.SCTP.RTOMin` | RTOMin | 重传超时的最小值 | 📝 RW | unsignedInt |
| 4 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.RTOMax` | `Device.Services.FAPControl.Transport.SCTP.RTOMax` | RTOMax | 重传超时的最大值 | 📝 RW | unsignedInt |
| 5 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.MaxInitRetransmits` | `Device.Services.FAPControl.Transport.SCTP.MaxInitRetransmits` | MaxInitRetransmits | 最大初始重传数 | 📝 RW | unsignedInt |
| 6 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.HBInterval` | `Device.Services.FAPControl.Transport.SCTP.HBInterval` | HBInterval | 心跳时间间隔 | 📝 RW | unsignedInt[1:] |
| 7 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.MaxPathRetransmits` | `Device.Services.FAPControl.Transport.SCTP.MaxPathRetransmits` | MaxPathRetransmits | 最大重传次数 | 📝 RW | unsignedInt |
| 8 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.MaxAssociationRetransmits` | `Device.Services.FAPControl.Transport.SCTP.MaxAssociationRetransmits` | MaxAssociationRetransmits | 本端的最大连续重传数目 | 📝 RW | unsignedInt |
| 9 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.ValCookieLife` | `Device.Services.FAPControl.Transport.SCTP.ValCookieLife` | ValCookieLife | 有效cookie的生命周期 | 📝 RW | unsignedInt |

#### 命令: Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.* 📖

- **{i}** 取值范围: `0~N` — SCTP关联实例编号，N由AssocNumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.Assoc.{i}.SCTPAssocLocalAddr` | `Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.SCTPAssocLocalAddr` | SCTPAssocLocalAddr | 本端IP地址 | 📖 R | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.Assoc.{i}.LocalPort` | `Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.LocalPort` | LocalPort | 本端端口 | 📖 R | unsignedInt |
| 3 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.Assoc.{i}.PrimaryPeerAddress` | `Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.PrimaryPeerAddress` | PrimaryPeerAddress | 远端IP地址 | 📖 R | string(64) |
| 4 | `InternetGatewayDevice.Services.FAPControl.Transport.SCTP.Assoc.{i}.RemotePort` | `Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.RemotePort` | RemotePort | 远端端口 | 📖 R | unsignedInt |

---

## SH - Services.FAPService.{i}.CellConfig.LTE.RAN

**RAN协议栈参数**  
参数数量: 73

#### 命令: Device.Services.FAPService.{i}.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300` | T300 | T300定时器 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301` | T301 | T301定时器 | 📝 RW | string(64) |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302` | T302 | T302定时器 | 📝 RW | unsignedInt[1:16] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA` | T304EUTRA | T304(EUTRA)定时器 | 📝 RW | string(64) |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT` | T304IRAT | T304(异系统)定时器 | 📝 RW | string(64) |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310` | T310 | T310定时器 | 📝 RW | unsignedInt[0,50,100,200,500,1000,2000] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311` | T311 | T311定时器 | 📝 RW | string(64) |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320` | T320 | T320定时器 | 📝 RW | unsignedInt[5, 10, 20, 30, 60, 120, 180] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310` | N310 | T310定时器 | 📝 RW | unsignedInt[1:4, 6, 8, 10, 20] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311` | N311 | T311定时器 | 📝 RW | unsignedInt[1:6, 8, 10] |

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles` | NumberOfRaPreambles | 竞争随机接入前导码数 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA` | SizeOfRaGroupA | 随机接入前导码组A大小 | 📝 RW | string(64) |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA` | MessageSizeGroupA | 组A信息大小 | 📝 RW | string(32) |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB` | MessagePowerOffsetGroupB | 组B功率补偿信息 | 📝 RW | string(32) |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PowerRampingStep` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PowerRampingStep` | PowerRampingStep | 功率增加补偿 | 📝 RW | string(16) |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleInitialReceivedTargetPower` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleInitialReceivedTargetPower` | PreambleInitialReceivedTargetPower | 前导码初始接收目标功率 | 📝 RW | string(128) |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax` | PreambleTransMax | 最大随机接入次数 | 📝 RW | string(64) |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize` | ResponseWindowSize | 随机接入响应窗口大小 | 📝 RW | string(32) |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer` | ContentionReslutionTimer | 冲突解决定时器 | 📝 RW | string(32) |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx` | MaxHARQMsg3Tx | Msg3HARQ最大传输次数 | 📝 RW | string(32) |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled` | DRXEnabled | DRX功能开关 | 📝 RW | Boolean |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx` | MaxHARQTx | 上行HARQ最大传输次数 | 📝 RW | unsignedInt[1:8, 10, 12, 16, 20, 24, 28] |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer` | PeriodicBSRTimer | 周期性BSR定时器 | 📝 RW | unsignedInt[0, 5, 10, 16, 20, 32, 40, 64, 80, 128, 160, 320, 640, 1280, 2560] |
| 14 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer` | RetxBSRTimer | 重传BSR定时器 | 📝 RW | unsignedInt[320, 640, 1280, 2560, 5120, 10240] |
| 15 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling` | TTIBundling | TTI绑定功能开关 | 📝 RW | Boolean |
| 16 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxUePerUlSf` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxUePerUlSf` | MaxUePerUlSf | 上行单帧最大调度用户数 | 📝 RW | unsignedInt[0:65535] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — DRX初始参数实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.DRXShortCycleTimer` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.DRXShortCycleTimer` | DRXShortCycleTimer | DRX短周期定时器 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.ONDurationTimer` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.ONDurationTimer` | ONDurationTimer | DRX持续时间定时器 | 📝 RW | string(64) |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.DRXInactivityTimer` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.DRXInactivityTimer` | DRXInactivityTimer | DRX非激活定时器 | 📝 RW | string(64) |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.DRXRetransmissionTimer` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.DRXRetransmissionTimer` | DRXRetransmissionTimer | DRX等待重传定时器 | 📝 RW | string(32) |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.LongDRXCycle` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.LongDRXCycle` | LongDRXCycle | DRX长周期 | 📝 RW | string(128) |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.ShortDRXCycle` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.ShortDRXCycle` | ShortDRXCycle | DRX短周期 | 📝 RW | string(64) |

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.* 📖📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

只读(📖): 2 | 可写(📝): 33

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ConfigurationIndex` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ConfigurationIndex` | ConfigurationIndex | PRACH配置索引 | 📝 RW | string(256) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.FreqOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.FreqOffset` | FreqOffset | PRACH频率偏移 | 📝 RW | string(256) |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag` | HighSpeedFlag | 高速状态标识 | 📝 RW | boolean |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex` | RootSequenceIndex | 逻辑根序列索引 | 📝 RW | string(512) |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig` | ZeroCorrelationZoneConfig | 零相关配置 | 📝 RW | string(64) |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled` | SRSEnabled | SRS使能开关 | 📝 RW | Boolean |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig` | SRSBandwidthConfig | SRS带宽配置 | 📝 RW | string(32) |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS` | SRSMaxUpPTS | SRSUpPTS带宽重配指示 | 📝 RW | Boolean |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission` | AckNackSRSSimultaneousTransmission | SRS/ACK/NACK同时传输标识 | 📝 RW | Boolean |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift` | DeltaPUCCHShift | PUCCH循环移位间隔 | 📝 RW | string |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI` | NRBCQI | PUCCH2/2a/2b占用RB数 | 📝 RW | string |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN` | NCSAN | PUCCH1/1a/1b预留资源数 | 📝 RW | unsignedInt[0:7] |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN` | N1PUCCHAN | SPSACK/NACK及SR预留资源数 | 📝 RW | string(512) |
| 14 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex` | CQIPUCCHResourceIndex | CQI报告使用PUCCH资源索引 | 📝 RW | string(512) |
| 15 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K` | K | 子带CQI报告带宽 | 📝 RW | unsignedInt[1:4] |
| 16 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUSCHPowerCtrlSwitch` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUSCHPowerCtrlSwitch` | PUSCHPowerCtrlSwitch | 上行PUSCH闭环功控开关 | 📝 RW | unsignedInt[0:1] |
| 17 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUCCHPowerCtrlSwitch` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUCCHPowerCtrlSwitch` | PUCCHPowerCtrlSwitch | 上行PUCCH闭环功控开关 | 📝 RW | unsignedInt[0:1] |
| 18 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM` | Enable64QAM | PUSCH64QAM使能开关 | 📝 RW | Boolean |
| 19 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode` | HoppingMode | PUSCH跳频模式 | 📝 RW | string |
| 20 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset` | HoppingOffset | PUSCH跳频偏置 | 📝 RW | string(256) |
| 21 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB` | NSB | 跳频子带个数 | 📝 RW | unsignedInt[1:4] |
| 22 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumPRSResourceBlocks` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumPRSResourceBlocks` | NumPRSResourceBlocks | PRSRB数目 | 📝 RW | unsignedInt |
| 23 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.PRSConfigurationIndex` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.PRSConfigurationIndex` | PRSConfigurationIndex | PRS配置索引 | 📝 RW | unsignedInt[0:4095] |
| 24 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumConsecutivePRSSubfames` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumConsecutivePRSSubfames` | NumConsecutivePRSSubfames | PRS连续子帧数 | 📝 RW | unsignedInt[1:2,4,6] |
| 25 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns` | SpecialSubframePatterns | 特殊子帧配置 | 📝 RW | unsignedInt[0:8] |
| 26 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment` | SubFrameAssignment | 上下行子帧配置 | 📝 RW | unsignedInt[0:6] |
| 27 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pb` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pb` | Pb | 天线端口信号功率比 | 📝 RW | string(32) |
| 28 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pa` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pa` | Pa | 小区PDSCH采用固定功率分配时的PA取值 | 📝 RW | string(64) |
| 29 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCHPersistent` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCHPersistent` | P0NominalPUSCHPersistent | 持续调度期望接收功率 | 📝 RW | Int[-126:24] |
| 30 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH` | P0NominalPUSCH | 非持续调度期望功率 | 📝 RW | string(512) |
| 31 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.Alpha` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.Alpha` | Alpha | 部分路损补偿系数 | 📝 RW | string(64) |
| 32 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH` | P0NominalPUCCH | PUCCH期望功率 | 📝 RW | string(512) |
| 33 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.DeltaMCSEnabled` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.DeltaMCSEnabled` | DeltaMCSEnabled | MCS补偿值 | 📝 RW | unsignedInt[0:1] |
| 34 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfTxChannel` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfTxAntenna` | NumOfTxAntenna | 发射通道数 | 📖 R | unsignedInt[1:7] |
| 35 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfRxChannel` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfRxAntenna` | NumOfRxAntenna | 接收通道数 | 📖 R | unsignedInt[1:7] |

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.NeighCellConfig` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.NeighCellConfig` | NeighCellConfig | 邻区配置 | 📝 RW | unsignedInt[0:3] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — 子帧配置实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.RadioframeAllocationOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.RadioframeAllocationOffset` | RadioframeAllocationOffset | 无线帧分配偏置 | 📝 RW | unsignedInt[0:6] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.RadioFrameAllocationPeriod` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.RadioFrameAllocationPeriod` | RadioFrameAllocationPeriod | 无线帧分配周期 | 📝 RW | unsignedInt[0:8] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.RadioFrameAllocationSize` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.RadioFrameAllocationSize` | RadioFrameAllocationSize | MBSFN连续无线帧数 | 📝 RW | unsignedInt[1,4] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.SubFrameAllocations` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.SubFrameAllocations` | SubFrameAllocations | MBSFN子帧分配样式 | 📝 RW | string(64) |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.SyncStratumID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.SyncStratumID` | SyncStratumID | 空口同步等级 | 📝 RW | unsignedInt[1:8] |

---

## SI - Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList

**邻区参数管理**  
参数数量: 39

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — GSM邻区实例编号，N由对应NumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.PLMNID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.PLMNID` | PLMNID | PLMN ID | 📝 RW | string(6) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.LAC` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.LAC` | LAC | LAC | 📝 RW | unsignedInt[0:65535] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.BSIC` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.BSIC` | BSIC | BSIC | 📝 RW | unsignedInt[0:255] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.CI` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.CI` | CI | CI | 📝 RW | unsignedInt[0:65535] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.BandIndicator` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.BandIndicator` | BandIndicator | 频段指示 | 📝 RW | string |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.BCCHARFCN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.BCCHARFCN` | BCCHARFCN | BCCH ARFCN | 📝 RW | unsignedInt[0:1023] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.RAC` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.RAC` | RAC | RAC | 📝 RW | unsignedInt[0:255] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — NR实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PLMNID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PLMNID` | PLMNID | 邻区PLMN ID | 📝 RW | string(6) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.CID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.CID` | CID | 邻区CID | 📝 RW | unsignedLong[1:68719476735] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.GnbIdLen` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.GnbIdLen` | GnbIdLen | 邻区GnbId长度 | 📝 RW | unsignedInt[22:32] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbFrequency` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbFrequency` | SsbFrequency | SSB频点号 | 📝 RW | unsignedInt[0:3279165] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbPeriodicity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbPeriodicity` | SsbPeriodicity_r15 | SSB周期 | 📝 RW | unsignedInt[5,10,20,40,80,160] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.SsbOffset` | SsbOffset_r15 | SSB偏移 | 📝 RW | unsignedInt[0..159] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Ssb_Duration` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Ssb_Duration` | Ssb_Duration_r15 | ssb_Duration_r15 | 📝 RW | unsignedInt[1,2,3,4,5] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PhyCellID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.PhyCellID` | PhyCellID | 邻区物理小区ID | 📝 RW | unsignedInt[0:1007] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.TAC` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.TAC` | TAC | 邻区TAC | 📝 RW | unsignedInt[0:16777215] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Qoffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.Qoffset` | QOffset | 小区特定偏移量 | 📝 RW | int[-24:-8, -6:6, 8:24] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NRband` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NRband` | NRband_r15 | Nr频段 | 📝 RW | unsignedInt[1:1024] |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NeighType` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{i}.NeighType` | NeighType | 邻区类型 | 📝 RW | unsignedInt[0，1，2] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — UMTS邻区实例编号，N由对应NumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.PLMNID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.PLMNID` | PLMNID | PLMN ID | 📝 RW | string(6) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.RNCID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.RNCID` | RNCID | RNC ID | 📝 RW | unsignedInt[0:65535] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.CID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.CID` | CID | CID | 📝 RW | unsignedInt[0:65535] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.LAC` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.LAC` | LAC | LAC | 📝 RW | unsignedInt[0:65535] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.RAC` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.RAC` | RAC | RAC | 📝 RW | unsignedInt[0:255] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.URA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.URA` | URA | URA | 📝 RW | unsignedInt[0:65535] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.UARFCNUL` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.UARFCNUL` | UARFCNUL | 上行UARFCN | 📝 RW | unsignedInt[0:16383] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.UARFCNDL` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.UARFCNDL` | UARFCNDL | 下行UARFCN | 📝 RW | unsignedInt[0:16383] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.PCPICHScramblingCode` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.PCPICHScramblingCode` | PCPICHScramblingCode | Primary CPICH扰码 | 📝 RW | unsignedInt[0:511] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.PCPICHTxPower` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.PCPICHTxPower` | PCPICHTxPower | Primary CPICH发射功率 | 📝 RW | int[-100:500] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.LTECell.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — LTE邻区实例编号，N由LTECellNumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PLMNID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PLMNID` | PLMNID | 邻区PLMN ID | 📝 RW | string(6) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID` | CID | 邻区CID | 📝 RW | unsignedInt[1:268435455] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.EUTRACarrierARFCN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.EUTRACarrierARFCN` | EUTRACarrierARFCN | 邻区EARFCN | 📝 RW | unsignedInt[0:65535] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PhyCellID` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PhyCellID` | PhyCellID | 邻区PCI | 📝 RW | unsignedInt[0:503] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.QOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.QOffset` | QOffset | 小区特定偏移量 | 📝 RW | int[-24:-8, -6:6, 8:24] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CIO` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CIO` | CIO | 小区独立偏移量 | 📝 RW | int[-24:-8, -6:6, 8:24] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.RSTxPower` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.RSTxPower` | RSTxPower | 参考信号发射功率 | 📝 RW | int[-60:50] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.Blacklisted` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.Blacklisted` | Blacklisted | 禁止切换标识 | 📝 RW | boolean |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.TAC` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.TAC` | TAC | 邻区TAC | 📝 RW | unsignedInt[0:65535] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.EnbType` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.EnbType` | EnbType | 邻区基站类型 | 📝 RW | unsignedInt |

---

## SJ - Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility

**移动性参数管理**  
参数数量: 145

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.MeasureCtrl.Smeasure` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.MeasureCtrl.Smeasure` | Smeasure | 测量启动门限 | 📝 RW | unsignedInt[0:97] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — A1测量控制实例编号

只读(📖): 1 | 可写(📝): 10

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.Enable` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.Enable` | Enable | 使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.A1ThresholdRSRP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.A1ThresholdRSRP` | A1ThresholdRSRP | A1事件RSRP门限 | 📝 RW | unsignedInt[0:97] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.A1ThresholdRSRQ` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.A1ThresholdRSRQ` | A1ThresholdRSRQ | A1事件RSRQ门限 | 📝 RW | unsignedInt[0:34] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.Hysteresis` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.Hysteresis` | Hysteresis | 迟滞值 | 📝 RW | unsignedInt[0:30] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.MaxReportCells` | MaxReportCells | 最大上报小区数 | 📝 RW | unsignedInt[1:8] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📖 R | unsignedInt[1:100] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.ReportAmount` | ReportAmount | 上报次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.ReportInterval` | ReportInterval | 上报间隔 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240, 60000, 360000, 720000, 1800000, 3600000] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.ReportQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.ReportQuantity` | ReportQuantity | 上报量 | 📝 RW | string |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.TimeToTrigger` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.TimeToTrigger` | TimeToTrigger | 触发时间 | 📝 RW | unsignedInt[0,40, 64, 80,100,128, 160,256,320, 480, 512, 640, 1024,1280,2560,5120] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.TriggerQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.TriggerQuantity` | TriggerQuantity | 触发量 | 📝 RW | string |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — A2测量控制实例编号

只读(📖): 1 | 可写(📝): 10

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.Enable` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.Enable` | Enable | 使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.A2ThresholdRSRP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.A2ThresholdRSRP` | A2ThresholdRSRP | A2事件RSRP门限 | 📝 RW | unsignedInt[0:97] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.A2ThresholdRSRQ` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.A2ThresholdRSRQ` | A2ThresholdRSRQ | A2事件RSRQ门限 | 📝 RW | unsignedInt[0:34] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.Hysteresis` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.Hysteresis` | Hysteresis | 迟滞值 | 📝 RW | unsignedInt[0:30] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.MaxReportCells` | MaxReportCells | 最大上报小区数 | 📝 RW | unsignedInt[1:8] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📖 R | unsignedInt[1:100] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.ReportAmount` | ReportAmount | 上报次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.ReportInterval` | ReportInterval | 上报间隔 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240, 60000, 360000, 720000, 1800000, 3600000] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.ReportQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.ReportQuantity` | ReportQuantity | 上报量 | 📝 RW | string |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.TimeToTrigger` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.TimeToTrigger` | TimeToTrigger | 触发时间 | 📝 RW | unsignedInt[0,40, 64, 80,100,128, 160,256,320, 480, 512, 640, 1024,1280,2560,5120] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.TriggerQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.TriggerQuantity` | TriggerQuantity | 触发量 | 📝 RW | string |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — A3测量控制实例编号

只读(📖): 1 | 可写(📝): 10

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.Enable` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.Enable` | Enable | 使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.A3Offset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.A3Offset` | A3Offset | A3事件门限 | 📝 RW | int[-30:30] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.Hysteresis` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.Hysteresis` | Hysteresis | 迟滞值 | 📝 RW | unsignedInt[0:30] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.MaxReportCells` | MaxReportCells | 最大上报小区数 | 📝 RW | unsignedInt[1:8] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📖 R | unsignedInt[1:100] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportAmount` | ReportAmount | 上报次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportOnLeave` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportOnLeave` | ReportOnLeave | A3事件离开上报指示 | 📝 RW | boolean |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportInterval` | ReportInterval | 上报间隔 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240, 60000, 360000, 720000, 1800000, 3600000] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.ReportQuantity` | ReportQuantity | 上报量 | 📝 RW | string |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.TimeToTrigger` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.TimeToTrigger` | TimeToTrigger | 触发时间 | 📝 RW | unsignedInt[0,40, 64, 80,100,128, 160,256,320, 480, 512, 640, 1024,1280,2560,5120] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.TriggerQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.TriggerQuantity` | TriggerQuantity | 触发量 | 📝 RW | string |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — A4测量控制实例编号

只读(📖): 1 | 可写(📝): 10

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.Enable` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.Enable` | Enable | 使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.A4ThresholdRSRP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.A4ThresholdRSRP` | A4ThresholdRSRP | A4事件RSRP门限 | 📝 RW | unsignedInt[0:97] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.A4ThresholdRSRQ` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.A4ThresholdRSRQ` | A4ThresholdRSRQ | A4事件RSRQ门限 | 📝 RW | unsignedInt[0:34] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.Hysteresis` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.Hysteresis` | Hysteresis | 迟滞值 | 📝 RW | unsignedInt[0:34] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.MaxReportCells` | MaxReportCells | 最大上报小区数 | 📝 RW | unsignedInt[1:8] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📖 R | unsignedInt[1:100] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.ReportAmount` | ReportAmount | 上报次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.ReportInterval` | ReportInterval | 上报间隔 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240, 60000, 360000, 720000, 1800000, 3600000] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.ReportQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.ReportQuantity` | ReportQuantity | 上报量 | 📝 RW | string |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.TimeToTrigger` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.TimeToTrigger` | TimeToTrigger | 触发时间 | 📝 RW | unsignedInt[0,40, 64, 80,100,128, 160,256,320, 480, 512, 640, 1024,1280,2560,5120] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.TriggerQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.TriggerQuantity` | TriggerQuantity | 触发量 | 📝 RW | string |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — A5测量控制实例编号

只读(📖): 1 | 可写(📝): 12

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.Enable` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.Enable` | Enable | 使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold1RSRP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold1RSRP` | A5Threshold1RSRP | A5事件RSRP门限1 | 📝 RW | unsignedInt[0:97] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold1RSRQ` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold1RSRQ` | A5Threshold1RSRQ | A5事件RSRQ门限1 | 📝 RW | unsignedInt[0:34] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold2RSRP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold2RSRP` | A5Threshold2RSRP | A5事件RSRP门限2 | 📝 RW | unsignedInt[0:97] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold2RSRQ` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.A5Threshold2RSRQ` | A5Threshold2RSRQ | A5事件RSRQ门限2 | 📝 RW | unsignedInt[0:34] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.Hysteresis` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.Hysteresis` | Hysteresis | 迟滞值 | 📝 RW | unsignedInt[0:30] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.MaxReportCells` | MaxReportCells | 最大上报小区数 | 📝 RW | unsignedInt[1:8] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📖 R | unsignedInt[1:100] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.ReportAmount` | ReportAmount | 上报次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.ReportInterval` | ReportInterval | 上报间隔 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240, 60000, 360000, 720000, 1800000, 3600000] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.ReportQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.ReportQuantity` | ReportQuantity | 上报量 | 📝 RW | string |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.TimeToTrigger` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.TimeToTrigger` | TimeToTrigger | 触发时间 | 📝 RW | unsignedInt[0,40, 64, 80,100,128, 160,256,320, 480, 512, 640, 1024,1280,2560,5120] |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.TriggerQuantity` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.TriggerQuantity` | TriggerQuantity | 触发量 | 📝 RW | string |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — 周期性测量控制实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📝 RW | unsignedInt[1:7] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.MaxReportCells` | MaxReportCells | 小区的最大数目 | 📝 RW | unsignedInt[1:65535] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.ReportInterval` | ReportInterval | 周期性测量报告 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240,60000,360000, 720000,1800000,3600000] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.ReportAmount` | ReportAmount | 测量报告次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN` | QoffsetGERAN | GERAN频点偏移量 | 📝 RW | string(128) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD` | MeasQuantityUTRAFDD | UTRA测量量 | 📝 RW | string |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityGERAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityGERAN` | MeasQuantityGERAN | GERAN测量量 | 📝 RW | string |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetUTRA` | QoffsetUTRA | UTRA频点偏移量 | 📝 RW | string(128) |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — B1测量控制实例编号

只读(📖): 1 | 可写(📝): 10

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.Enable` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.Enable` | Enable | 使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdCDMA2000` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdCDMA2000` | B1ThresholdCDMA2000 | B1事件门限(CDMA2000) | 📝 RW | int[-5:91] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdGERAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdGERAN` | B1ThresholdGERAN | B1事件门限(GERAN) | 📝 RW | unsignedInt[0:63] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdUTRAEcN0` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdUTRAEcN0` | B1ThresholdUTRAEcN0 | B1事件EcN0门限(UTRA) | 📝 RW | unsignedInt[0:49] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdUTRARSCP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.B1ThresholdUTRARSCP` | B1ThresholdUTRARSCP | B1事件RSCP门限(UTRA) | 📝 RW | int[-5:91] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.Hysteresis` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.Hysteresis` | Hysteresis | 迟滞值 | 📝 RW | unsignedInt[0:30] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.MaxReportCells` | MaxReportCells | 最大上报小区数 | 📝 RW | unsignedInt[1:8] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📖 R | unsignedInt[1:100] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.ReportAmount` | ReportAmount | 上报次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.ReportInterval` | ReportInterval | 上报间隔 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240, 60000, 360000, 720000, 1800000, 3600000] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.TimeToTrigger` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.TimeToTrigger` | TimeToTrigger | 触发时间 | 📝 RW | unsignedInt[0,40, 64, 80,100,128, 160,256,320, 480, 512, 640, 1024,1280,2560,5120] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — B2测量控制实例编号

只读(📖): 1 | 可写(📝): 12

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.Enable` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.Enable` | Enable | 使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold1EutraRSRP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold1EutraRSRP` | B2Threshold1EutraRSRP | B2事件RSRP门限(EUTRA) | 📝 RW | unsignedInt[0:97] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold1EutraRSRQ` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold1EutraRSRQ` | B2Threshold1EutraRSRQ | B2事件RSRQ门限(EUTRA) | 📝 RW | unsignedInt[0:34] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2CDMA2000` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2CDMA2000` | B2Threshold2CDMA2000 | B2事件门限(CDMA2000) | 📝 RW | unsignedInt[0:63] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2GERAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2GERAN` | B2Threshold2GERAN | B2事件门限(GERAN) | 📝 RW | unsignedInt[0:63] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2UTRAEcN0` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2UTRAEcN0` | B2Threshold2UTRAEcN0 | B2事件EcN0门限(UTRA) | 📝 RW | unsignedInt[0:49] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2UTRARSCP` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.B2Threshold2UTRARSCP` | B2Threshold2UTRARSCP | B2触发RSCP门限(UTRA) | 📝 RW | int[-5:91] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.Hysteresis` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.Hysteresis` | Hysteresis | 迟滞值 | 📝 RW | unsignedInt[0:30] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.MaxReportCells` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.MaxReportCells` | MaxReportCells | 最大上报小区数 | 📝 RW | unsignedInt[1:8] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.MeasurePurpose` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.MeasurePurpose` | MeasurePurpose | 测量目的 | 📖 R | unsignedInt[1:100] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.ReportAmount` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.ReportAmount` | ReportAmount | 上报次数 | 📝 RW | unsignedInt[0:2, 4, 8, 16, 32, 64] |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.ReportInterval` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.ReportInterval` | ReportInterval | 上报间隔 | 📝 RW | unsignedInt[120, 240, 480, 640, 1024, 2048, 5120, 10240, 60000, 360000, 720000, 1800000, 3600000] |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.TimeToTrigger` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.TimeToTrigger` | TimeToTrigger | 触发时间 | 📝 RW | unsignedInt[0,40, 64, 80,100,128, 160,256,320, 480, 512, 640, 1024,1280,2560,5120] |

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst` | Qhyst | 服务小区重选迟滞值 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.IntraFreqReselection` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.IntraFreqReselection` | IntraFreqReselection | 同频重选指示 | 📝 RW | unsignedInt |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFMedium` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFMedium` | QHystSFMedium | Qhyst比例因子(中速) | 📝 RW | int[-6, -4, -2, 0] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFHigh` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFHigh` | QHystSFHigh | Qhyst比例因子(高速) | 📝 RW | int[-6, -4, -2, 0] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.TEvaluation` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.TEvaluation` | TEvaluation | 允许小区重选数目的间隔时间 | 📝 RW | unsignedInt[30, 60, 120, 180, 240] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.THystNormal` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.THystNormal` | THystNormal | 正常状态附加判决时长 | 📝 RW | unsignedInt[30, 60, 120, 180, 240] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeMedium` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeMedium` | NCellChangeMedium | 进入中速重选次数门限 | 📝 RW | unsignedInt[1:16] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeHigh` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeHigh` | NCellChangeHigh | 进入高速重选次数门限 | 📝 RW | unsignedInt[1:16] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1` | QRxLevMinSIB1 | 服务小区最小接收电平 | 📝 RW | int[-70:-22] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3` | QRxLevMinSIB3 | 同频邻区最小接收电平 | 📝 RW | string(256) |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinOffset` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinOffset` | QRxLevMinOffset | 最小接收电平偏移量 | 📝 RW | unsignedInt[1:8] |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearch` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearch` | SIntraSearch | 同频测量启动门限 | 📝 RW | string(128) |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRA` | TReselectionEUTRA | 同频小区重选时间 | 📝 RW | string(32) |
| 14 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearch` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearch` | SNonIntraSearch | 非同频测量启动门限 | 📝 RW | string(128) |
| 15 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchPR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchPR9` | SNonIntraSearchPR9 | 异频/异系统RSRP测量启动门限 | 📝 RW | unsignedInt[0:31] |
| 16 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchQR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchQR9` | SNonIntraSearchQR9 | 异频/异系统RSRQ测量启动门限 | 📝 RW | unsignedInt[0:31] |
| 17 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.CellReselectionPriority` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.CellReselectionPriority` | CellReselectionPriority | 同频小区重选优先级 | 📝 RW | unsignedInt[0:7] |
| 18 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.PMax` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.PMax` | PMax | UE允许最大发射功率 | 📝 RW | int[-30:33] |
| 19 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLow` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLow` | ThreshServingLow | 低优先级重选门限 | 📝 RW | unsignedInt[0:31] |
| 20 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLowQR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLowQR9` | ThreshServingLowQR9 | 服务频点低优先级RSRQ重选门限 | 📝 RW | unsignedInt[0:31] |
| 21 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium` | TReselectionEUTRASFMedium | TReselectionUTRA比例因子(中速) | 📝 RW | unsignedInt[25, 50, 75, 100] |
| 22 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh` | TReselectionEUTRASFHigh | TReselectionUTRA比例因子(高速) | 📝 RW | unsignedInt[25, 50, 75, 100] |
| 23 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchPR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchPR9` | SIntraSearchPR9 | 同频RSRP测量启动门限 | 📝 RW | unsignedInt[0:31] |
| 24 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchQR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchQR9` | SIntraSearchQR9 | 同频RSRQ测量启动门限 | 📝 RW | unsignedInt[0:31] |
| 25 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Reselection` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Reselection` | QQualMinR9Reselection | 小区重选最低接入信号质量 | 📝 RW | int[-34:-3] |
| 26 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Selection` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Selection` | QQualMinR9Selection | 小区选择最低接入信号质量 | 📝 RW | int[-34:-3] |
| 27 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinOffsetR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinOffsetR9` | QQualMinOffsetR9 | 小区选择最低接入信号质量偏置 | 📝 RW | int[1:8] |
| 28 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.AllowedMeasBandwidth` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.AllowedMeasBandwidth` | AllowedMeasBandwidth | 测量带宽 | 📝 RW | unsignedInt(6,15,25,50,75,100) |

#### 命令: Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.* 📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA` | TReselectionUTRA | UTRAN小区重选时间 | 📝 RW | string(32) |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN` | TReselectionGERAN | GERAN小区重选时间 | 📝 RW | string(32) |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — GERANFreqGroup实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.BCCHARFCN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.BCCHARFCN` | BCCHARFCN | GERAN小区频点 | 📝 RW | unsignedInt[0:1023] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.CellReselectionPriority` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.CellReselectionPriority` | CellReselectionPriority | 异系统GERAN重选优先级配置 | 📝 RW | unsignedInt[0:7] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.QRxLevMin` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.QRxLevMin` | QRxLevMin | GERAN最低接收电平 | 📝 RW | unsignedInt[0:45] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.ThreshXHigh` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.ThreshXHigh` | ThreshXHigh | GERAN高优先级重选门限值 | 📝 RW | unsignedInt[0:31] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.ThreshXLow` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.ThreshXLow` | ThreshXLow | GERAN低优先级重选门限值 | 📝 RW | unsignedInt[0:31] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.PMaxGERAN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{i}.PMaxGERAN` | PMaxGERAN | UE最大许可传输功率 | 📝 RW | unsignedInt[0:39] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — UTRA频点实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.UTRACarrierARFCN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.UTRACarrierARFCN` | UTRACarrierARFCN | UTRAN小区频点 | 📝 RW | unsignedInt[0:16383] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.CellReselectionPriority` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.CellReselectionPriority` | CellReselectionPriority | UTRAN频点重选优先级 | 📝 RW | unsignedInt[0:7] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.ThreshXHigh` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.ThreshXHigh` | ThreshXHigh | UTRAN频点高优先级重选门限 | 📝 RW | unsignedInt[0:31] |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.ThreshXLow` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.ThreshXLow` | ThreshXLow | UTRAN频点低优先级重选门限 | 📝 RW | unsignedInt[0:31] |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.QRxLevMin` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.QRxLevMin` | QRxLevMin | UTRAN最低接收电平 | 📝 RW | int[-60:-13] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.PMaxUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.PMaxUTRA` | PMaxUTRA | UTRAN最大允许发射功率 | 📝 RW | int[-50:33] |

#### 命令: Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{iβ}.* 📝

- **{iα}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）
- **{iβ}** 取值范围: `0~N` — 载波实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.EUTRACarrierARFCN` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.EUTRACarrierARFCN` | EUTRACarrierARFCN | EARFCN | 📝 RW | unsignedInt[0:65535] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.QRxLevMinSIB5` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.QRxLevMinSIB5` | QRxLevMinSIB5 | 异频邻区最小接收电平 | 📝 RW | string(256) |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.QOffsetFreq` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.QOffsetFreq` | QOffsetFreq | 小区间频率偏移量 | 📝 RW | string(128) |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRA` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRA` | TReselectionEUTRA | 异频小区重选时间 | 📝 RW | string(32) |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.QQualMinR9Reselection` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.QQualMinR9Reselection` | QQualMinR9Reselection | 小区重选最低接入信号质量 | 📝 RW | int[-34:-3] |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.CellReselectionPriority` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.CellReselectionPriority` | CellReselectionPriority | 异频小区重选优先级 | 📝 RW | unsignedInt[0:7] |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXHigh` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXHigh` | ThreshXHigh | 异频高优先级重选门限 | 📝 RW | unsignedInt[0:31] |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXLow` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXLow` | ThreshXLow | 异频低优先级重选门限 | 📝 RW | unsignedInt[0:31] |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.PMax` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.PMax` | PMax | UE最大允许发射功率 | 📝 RW | int[-30:33] |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFMedium` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFMedium` | TReselectionEUTRASFMedium | TReselectionEUTRA比例因子(中速) | 📝 RW | unsignedInt[25, 50, 75, 100] |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFHigh` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFHigh` | TReselectionEUTRASFHigh | TReselectionEUTRA比例因子(高速) | 📝 RW | unsignedInt[25, 50, 75, 100] |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXHighQR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXHighQR9` | ThreshXHighQR9 | 异频频点RSRQ高优先级重选门限 | 📝 RW | unsignedInt[0:31] |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXLowQR9` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.ThreshXLowQR9` | ThreshXLowQR9 | 异频频点RSRQ低优先级重选门限 | 📝 RW | unsignedInt[0:31] |

---

## SK - Services.FAPService.{i}.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam

**SON参数管理**  
参数数量: 28

#### 命令: Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.* 📖📝

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

只读(📖): 1 | 可写(📝): 24

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONSysMode` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONSysMode` | SONSysMode | SON系统模式 | 📝 RW | string |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONWorkMode` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONWorkMode` | SONWorkMode | SON模式设置 | 📝 RW | string |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIOptEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIOptEnable` | PCIOptEnable | PCI自优化算法开关 | 📝 RW | boolean |
| 4 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIReconfigWaitTime` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIReconfigWaitTime` | PCIReconfigWaitTime | PCI重配等待定时器 | 📝 RW | unsignedInt |
| 5 | `InternetGatewayDevice.Services.FAPService.{i}.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidateARFCNList` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidateARFCNList` | CandidateARFCNList | 候选频点列表 | 📝 RW | string(64) |
| 6 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidatePCIList` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidatePCIList` | CandidatePCIList | 候选PCI列表 | 📝 RW | string(64) |
| 7 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANREnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANREnable` | ANREnable | ANR算法总开关 | 📝 RW | boolean |
| 8 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRInterFeqEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRInterFeqEnable` | ANRInterFeqEnable | E-UTRAN异频ANR算法开关 | 📝 RW | boolean |
| 9 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRGERANEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRGERANEnable` | ANRGERANEnable | GERAN异系统ANR算法开关 | 📝 RW | boolean |
| 10 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRUTRANEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRUTRANEnable` | ANRUTRANEnable | UTRAN异系统ANR算法开关 | 📝 RW | boolean |
| 11 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ARFCNEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ARFCNEnable` | ARFCNEnable | 频点自配置算法开关 | 📝 RW | boolean |
| 12 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxLTENeighbourCellNum` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxLTENeighbourCellNum` | MaxLTENeighbourCellNum | 最大LTE邻区数 | 📝 RW | unsignedInt |
| 13 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxUTRANNeighbourCellNum` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxUTRANNeighbourCellNum` | MaxUTRANNeighbourCellNum | 最大UTRAN邻区数 | 📝 RW | unsignedInt |
| 14 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxGRANNeighbourCellNum` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxGRANNeighbourCellNum` | MaxGERANNeighbourCellNum | 最大GERAN邻区数 | 📝 RW | unsignedInt |
| 15 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ReSynCellEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ReSynCellEnable` | ReSynCellEnable | 重新同步小区使能开关 | 📝 RW | boolean |
| 16 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PowerEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PowerEnable` | PowerEnable | 功率自配置算法开关 | 📝 RW | boolean |
| 17 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferFreqBandList` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferFreqBandList` | LTESnifferFreqBandList | LTE侦听频段列表 | 📝 RW | string(64) |
| 18 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferChannelList` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferChannelList` | LTESnifferChannelList | LTE侦听频点列表 | 📝 RW | string(64) |
| 19 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferEnable` | GERANSnifferEnable | GERAN侦听使能 | 📝 RW | boolean |
| 20 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferChannelList` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferChannelList` | GERANSnifferChannelList | GERAN侦听频道号列表 | 📝 RW | string(256) |
| 21 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferEnable` | UTRANSnifferEnable | UTRAN侦听使能 | 📝 RW | boolean |
| 22 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferChannelList` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferChannelList` | UTRANSnifferChannelList | UTRAN侦听频道号列表 | 📝 RW | string(256) |
| 23 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MROEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MROEnable` | MROEnable | 邻区鲁棒性优化功能开关 | 📝 RW | boolean |
| 24 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SHEnable` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SHEnable` | SHEnable | 自治愈功能开关 | 📝 RW | boolean |
| 25 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SyncMode` | `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SyncMode` | SyncMode | 时钟同步模式 | 📖 R | string(64) |

#### 命令: Device.Services.FAPService.{i}.FAPControl.SelfConfig.* 📖

- **{i}** 取值范围: `1~3` — 载波实例（i=1:载波1, i=2:载波2, i=3:载波3）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Stage` | `Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Stage` | Stage | 开站阶段 | 📖 R | unsignedInt[1:3] |
| 2 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Status` | `Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Status` | Status | 开站状态 | 📖 R | unsignedInt[1:2] |
| 3 | `InternetGatewayDevice.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.FailureCause` | `Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.FailureCause` | FailureCause | 开站失败原因 | 📖 R | string(512) |

---

## SL - WANDevice

**WAN口配置参数管理**  
参数数量: 37

#### 命令: Device.Ethernet.Interface.{i}.* 📖📝

- **{i}** 取值范围: `0~N` — 以太网接口实例编号，N由InterfaceNumberOfEntries决定

只读(📖): 5 | 可写(📝): 4

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Ethernet.Interface.{i}.Enable` | `Device.Ethernet.Interface.{i}.Enable` | Enable | 启用或禁用该接口 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Ethernet.Interface.{i}.UserLabel` | `Device.Ethernet.Interface.{i}.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
| 3 | `InternetGatewayDevice.Ethernet.Interface.{i}.Name` | `Device.Ethernet.Interface.{i}.Name` | Name | 端口名称 | 📖 R | string |
| 4 | `InternetGatewayDevice.Ethernet.Interface.{i}.Status` | `Device.Ethernet.Interface.{i}.Status` | Status | 表明接口的状态 | 📖 R | string |
| 5 | `InternetGatewayDevice.Ethernet.Interface.{i}.MACAddress` | `Device.Ethernet.Interface.{i}.MACAddress` | MACAddress | 接口的物理地址 | 📖 R | string(17) |
| 6 | `InternetGatewayDevice.Ethernet.Interface.{i}.MaxBitRate` | `Device.Ethernet.Interface.{i}.MaxBitRate` | MaxBitRate | 该连接可用的最大速率模式 | 📝 RW | string |
| 7 | `InternetGatewayDevice.Ethernet.Interface.{i}.SignTransMedia` | `Device.Ethernet.Interface.{i}.SignTransMedia` | SignTransMedia | 信号传送介质类型 | 📖 R | string |
| 8 | `InternetGatewayDevice.Ethernet.Interface.{i}.DuplexMode` | `Device.Ethernet.Interface.{i}.DuplexMode` | DuplexMode | 该连接使用的双工模式 | 📝 RW | string |
| 9 | `InternetGatewayDevice.Ethernet.Interface.{i}.PortLocation` | `Device.Ethernet.Interface.{i}.PortLocation` | PortLocation | 端口位置 | 📖 R | string |

#### 命令: Device.Ethernet.Interface.{iα}.IPv4Address.{iβ}.* 📝

- **{iα}** 取值范围: `0~N` — 以太网接口实例编号，N由InterfaceNumberOfEntries决定
- **{iβ}** 取值范围: `0~N` — IPv4Address实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv4Address.{i}.IPAddress` | `Device.Ethernet.Interface.{i}.IPv4Address.{i}.IPAddress` | IPAddress | IP 地址 | 📝 RW | string(15) |
| 2 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv4Address.{i}.DefaultGateway` | `Device.Ethernet.Interface.{i}.IPv4Address.{i}.DefaultGateway` | DefaultGateway | WAN 接口的默认网关 | 📝 RW | string(15) |
| 3 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv4Address.{i}.SubnetMask` | `Device.Ethernet.Interface.{i}.IPv4Address.{i}.SubnetMask` | SubnetMask | WAN 接口的子网掩码 | 📝 RW | string |
| 4 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv4Address.{i}.AddressingType` | `Device.Ethernet.Interface.{i}.IPv4Address.{i}.AddressingType` | AddressingType | 外部IP 地址方法 | 📝 RW | string |
| 5 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv4Address.{i}.PortType` | `Device.Ethernet.Interface.{i}.IPv4Address.{i}.PortType` | PortType | 端口类型 | 📝 RW | string |

#### 命令: Device.Ethernet.Interface.{iα}.IPv6Address.{iβ}.* 📝

- **{iα}** 取值范围: `0~N` — 以太网接口实例编号，N由InterfaceNumberOfEntries决定
- **{iβ}** 取值范围: `0~N` — IPv6Address实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv6Address.{i}.IPAddress` | `Device.Ethernet.Interface.{i}.IPv6Address.{i}.IPAddress` | IPAddress | IPv6 地址 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv6Address.{i}.PrefixLength` | `Device.Ethernet.Interface.{i}.IPv6Address.{i}.PrefixLength` | PrefixLength | 前缀长度 | 📝 RW | unsignedInt[1:128] |
| 3 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv6Address.{i}.Origin` | `Device.Ethernet.Interface.{i}.IPv6Address.{i}.Origin` | Origin | 外部IP 来源 | 📝 RW | string |
| 4 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv6Address.{i}.PortType` | `Device.Ethernet.Interface.{i}.IPv6Address.{i}.PortType` | PortType | 端口类型 | 📝 RW | string |
| 5 | `InternetGatewayDevice.Ethernet.Interface.{i}.IPv6Address.{i}.DefaultGateway` | `Device.Ethernet.Interface.{i}.IPv6Address.{i}.DefaultGateway` | DefaultGateway | WAN 接口的默认网关 | 📝 RW | string(64) |

#### 命令: Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.* 📝

- **{iα}** 取值范围: `0~N` — 以太网接口实例编号，N由InterfaceNumberOfEntries决定
- **{iβ}** 取值范围: `0~N` — VlanInterface实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.Name` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.Name` | Name | vlan子接口名称 | 📝 RW | string |
| 2 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.Id` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.Id` | Id | Vlan Id | 📝 RW | interger |
| 3 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.Enable` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.Enable` | Enable | 启用或禁用该接口 | 📝 RW | boolean |

#### 命令: Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv4Address.{iγ}.* 📝

- **{iα}** 取值范围: `0~N` — 以太网接口实例编号，N由InterfaceNumberOfEntries决定
- **{iβ}** 取值范围: `0~N` — VlanInterface实例编号，N由对应NumberOfEntries参数决定
- **{iγ}** 取值范围: `0~N` — IPv4Address实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.IPAddress` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.IPAddress` | IPAddress | IP 地址 | 📝 RW | string(15) |
| 2 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.SubnetMask` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.SubnetMask` | SubnetMask | vlan子接口的子网掩码 | 📝 RW | string |
| 3 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.AddressingType` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.AddressingType` | AddressingType | 外部IP 地址方法 | 📝 RW | string |
| 4 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.DefaultGateway` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.DefaultGateway` | DefaultGateway | vlan子接口的默认网关 | 📝 RW | string(15) |
| 5 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.PortType` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv4Address.{i}.PortType` | PortType | 端口类型 | 📝 RW | string |

#### 命令: Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv6Address.{iγ}.* 📝

- **{iα}** 取值范围: `0~N` — 以太网接口实例编号，N由InterfaceNumberOfEntries决定
- **{iβ}** 取值范围: `0~N` — VlanInterface实例编号，N由对应NumberOfEntries参数决定
- **{iγ}** 取值范围: `0~N` — IPv6Address实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.IPAddress` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.IPAddress` | IPAddress | IPv6 地址 | 📝 RW | string(64) |
| 2 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.PrefixLength` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.PrefixLength` | PrefixLength | 前缀长度 | 📝 RW | unsignedInt[1:128] |
| 3 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.Origin` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.Origin` | Origin | 外部IP 来源 | 📝 RW | string |
| 4 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.DefaultGateway` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.DefaultGateway` | DefaultGateway | vlan子接口的默认网关 | 📝 RW | string(64) |
| 5 | `InternetGatewayDevice.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.PortType` | `Device.Ethernet.Interface.{i}.VlanInterface.{i}.IPv6Address.{i}.PortType` | PortType | 端口类型 | 📝 RW | string |

#### 命令: Device.Ethernet.IpRoute.{i}.* 📝

- **{i}** 取值范围: `0~N` — IpRoute实例编号，N由对应NumberOfEntries参数决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Ethernet.IpRoute.{i}.IpVer` | `Device.Ethernet.IpRoute.{i}.IpVer` | IpVer | IP地址版本 | 📝 RW | unsignedInt[1:2] |
| 2 | `InternetGatewayDevice.Ethernet.IpRoute.{i}.DstIpNetwork` | `Device.Ethernet.IpRoute.{i}.DstIpNetwork` | DstIpNetwork | 目的网段 | 📝 RW | string(64) |
| 3 | `InternetGatewayDevice.Ethernet.IpRoute.{i}.PrefixLength` | `Device.Ethernet.IpRoute.{i}.PrefixLength` | PrefixLength | 前缀长度 | 📝 RW | unsignedInt[0:128] |
| 4 | `InternetGatewayDevice.Ethernet.IpRoute.{i}.GatewayIpAddress` | `Device.Ethernet.IpRoute.{i}.GatewayIpAddress` | GatewayIpAddress | 网关地址 | 📝 RW | string(64) |
| 5 | `InternetGatewayDevice.Ethernet.IpRoute.{i}.InterfaceName` | `Device.Ethernet.IpRoute.{i}.InterfaceName` | InterfaceName | 端口名称 | 📝 RW | string |

---

## SM - Ipsec

**IPsec参数管理**  
参数数量: 9

#### 命令: Device.IPsec.* 📖📝

只读(📖): 7 | 可写(📝): 2

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.IPsec.Enable` | `Device.IPsec.Enable` | Enable | 开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.IPsec.MyKeyMode` | `Device.IPsec.MyKeyMode` | MyKeyMode | 鉴权方式 | 📝 RW | string |
| 3 | `InternetGatewayDevice.IPsec.Status` | `Device.IPsec.Status` | Status | 状态 | 📖 R | string |
| 4 | `InternetGatewayDevice.IPsec.AHSupported` | `Device.IPsec.AHSupported` | AHSupported | AH是否支持 | 📖 R | boolean |
| 5 | `InternetGatewayDevice.IPsec.IKEv2SupportedEncryptionAlgorithms` | `Device.IPsec.IKEv2SupportedEncryptionAlgorithms` | IKEv2SupportedEncryptionAlgorithms | IKE2加密算法 | 📖 R | string |
| 6 | `InternetGatewayDevice.IPsec.ESPSupportedEncryptionAlgorithms` | `Device.IPsec.ESPSupportedEncryptionAlgorithms` | ESPSupportedEncryptionAlgorithms | ESP加密算法 | 📖 R | string |
| 7 | `InternetGatewayDevice.IPsec.IKEv2SupportedPseudoRandomFunctions` | `Device.IPsec.IKEv2SupportedPseudoRandomFunctions` | IKEv2SupportedPseudoRandomFunctions | 随机函数 | 📖 R | string |
| 8 | `InternetGatewayDevice.IPsec.SupportedIntegrityAlgorithms` | `Device.IPsec.SupportedIntegrityAlgorithms` | SupportedIntegrityAlgorithms | 完整性算法 | 📖 R | string |
| 9 | `InternetGatewayDevice.IPsec.SupportedDiffieHellmanGroupTransforms` | `Device.IPsec.SupportedDiffieHellmanGroupTransforms` | SupportedDiffieHellmanGroupTransforms | Diffie-Hellman交换 | 📖 R | string |

---

## SN - Time

**时间服务器参数管理**  
参数数量: 8

#### 命令: Device.Time.* 📖📝

只读(📖): 1 | 可写(📝): 7

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.Time.Enable` | `Device.Time.Enable` | Enable | NTP使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.Time.NTPServer1` | `Device.Time.NTPServer1` | NTPServer1 | NTP服务器1 | 📝 RW | string(64) |
| 3 | `InternetGatewayDevice.Time.NTPServer2` | `Device.Time.NTPServer2` | NTPServer2 | NTP服务器2 | 📝 RW | string(64) |
| 4 | `InternetGatewayDevice.Time.NTPServer3` | `Device.Time.NTPServer3` | NTPServer3 | NTP服务器3 | 📝 RW | string(64) |
| 5 | `InternetGatewayDevice.Time.NTPServer4` | `Device.Time.NTPServer4` | NTPServer4 | NTP服务器4 | 📝 RW | string(64) |
| 6 | `InternetGatewayDevice.Time.NTPServer5` | `Device.Time.NTPServer5` | NTPServer5 | NTP服务器5 | 📝 RW | string(64) |
| 7 | `InternetGatewayDevice.Time.CurrentLocalTime` | `Device.Time.CurrentLocalTime` | CurrentLocalTime | 本地时间 | 📖 R | dateTime |
| 8 | `InternetGatewayDevice.Time.LocalTimeZone` | `Device.Time.LocalTimeZone` | LocalTimeZone | 本地时区 | 📝 RW | string(64) |

---

## SO - FAP.GPS

**GPS信息参数管理**  
参数数量: 3

#### 命令: Device.FAP.GPS.* 📖

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FAP.GPS.LockedLatitude` | `Device.FAP.GPS.LockedLatitude` | LockedLatitude | 纬度 | 📖 R | int[-90000000:90000000] |
| 2 | `InternetGatewayDevice.FAP.GPS.LockedLongitude` | `Device.FAP.GPS.LockedLongitude` | LockedLongitude | 经度 | 📖 R | int[-180000000:180000000] |
| 3 | `InternetGatewayDevice.FAP.GPS.NumberOfSatellites` | `Device.FAP.GPS.NumberOfSatellites` | NumberOfSatellites | 星个数 | 📖 R | unsignedInt |

---

## SP - FAP.MRMgmt

**MR参数管理**  
参数数量: 14

#### 命令: Device.FAP.MRMgmt.Config.{i}.* 📝

- **{i}** 取值范围: `0~N` — 配置实例编号，N由对应NumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.MrEnable` | `Device.FAP.MRMgmt.Config.{i}.MrEnable` | MrEnable | MR开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.MrUrl` | `Device.FAP.MRMgmt.Config.{i}.MrUrl` | MrUrl | Url地址 | 📝 RW | string(256) |
| 3 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.MrUsername` | `Device.FAP.MRMgmt.Config.{i}.MrUsername` | MrUsername | 用户名 | 📝 RW | string(256) |
| 4 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.MrPassword` | `Device.FAP.MRMgmt.Config.{i}.MrPassword` | MrPassword | 密码 | 📝 RW | string(256) |
| 5 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.MeasureType` | `Device.FAP.MRMgmt.Config.{i}.MeasureType` | MeasureType | MR文件类型 | 📝 RW | string |
| 6 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.OmcName` | `Device.FAP.MRMgmt.Config.{i}.OmcName` | OmcName | OMC-R名称 | 📝 RW | string |
| 7 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.SamplePeriod` | `Device.FAP.MRMgmt.Config.{i}.SamplePeriod` | SamplePeriod | MR采样周期 | 📝 RW | unsignedInt |
| 8 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.UploadPeriod` | `Device.FAP.MRMgmt.Config.{i}.UploadPeriod` | UploadPeriod | MR采集周期 | 📝 RW | unsignedInt |
| 9 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.SampleBeginTime` | `Device.FAP.MRMgmt.Config.{i}.SampleBeginTime` | SampleBeginTime | 绝对时间参考 | 📝 RW | dateTime |
| 10 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.SampleEndTime` | `Device.FAP.MRMgmt.Config.{i}.SampleEndTime` | SampleEndTime | 绝对时间参考 | 📝 RW | dateTime |
| 11 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.PrbNum` | `Device.FAP.MRMgmt.Config.{i}.PrbNum` | PrbNum | 子帧的PRB | 📝 RW | string |
| 12 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.SubFrameNum` | `Device.FAP.MRMgmt.Config.{i}.SubFrameNum` | SubFrameNum | 子帧数 | 📝 RW | string |
| 13 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.MRECGIList` | `Device.FAP.MRMgmt.Config.{i}.MRECGIList` | MRECGIList | MR小区列表 | 📝 RW | string |
| 14 | `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.MeasureItems` | `Device.FAP.MRMgmt.Config.{i}.MeasureItems` | MeasureItems | 测量项 | 📝 RW | string |

---

## SQ - FAP.PerfMgmt

**性能参数管理**  
参数数量: 11

#### 命令: .*

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `1. 根据《中国移动一体化皮基站/扩展型一体化皮基站网络管理南向接口技术规范》7.5.3.1节要求：“基站设备将性能数据以文件的形式定期上传到HeMS上（通过HTTP/HTTPS），当相应的性能数据文件准备好后，HeMS可随时进行解析和计算。”，“性能文件的采集粒度遵循《NanoCell eNB网元统计数据需求规范-PM(V2.9.0)》的要求。”
2. 建议网管南向接口具体性能测量参数的数据模型与《NanoCell eNB网元统计数据需求规范-PM(V2.9.0)》中规定的北向接口性能测量参数数据模型保持一致；
3. 建议南向接口性能测量参数优先级和北向接口保持一致；
4. 网管南向接口性能文件格式（xml）遵循3GPP TS32.435，由网管设备完成南向接口和北向接口性能文件格式的转换。` | `` |  |  |   |  |

#### 命令: Device.FAP.PerfMgmt.Config.{i}.* 📝

- **{i}** 取值范围: `0~N` — 配置实例编号，N由对应NumberOfEntries决定

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.Enable` | `Device.FAP.PerfMgmt.Config.{i}.Enable` | Enable | 文件周期上传使能开关 | 📝 RW | boolean |
| 2 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.Alias` | `Device.FAP.PerfMgmt.Config.{i}.Alias` | Alias | 别名 | 📝 RW | string(64) |
| 3 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.URL` | `Device.FAP.PerfMgmt.Config.{i}.URL` | URL | 文件管理URL | 📝 RW | string(256) |
| 4 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.Username` | `Device.FAP.PerfMgmt.Config.{i}.Username` | Username | 文件管理用户名 | 📝 RW | string(256) |
| 5 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.Password` | `Device.FAP.PerfMgmt.Config.{i}.Password` | Password | 文件管理密码 | 📝 RW | string(256) |
| 6 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.PeriodicUploadInterval` | `Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadInterval` | PeriodicUploadInterval | 文件周期上传时间间隔 | 📝 RW | unsignedInt[1:65535] |
| 7 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime` | `Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime` | PeriodicUploadTime | 文件上传时间 | 📝 RW | dateTime |
| 8 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.ReplenishEnable` | `Device.FAP.PerfMgmt.Config.{i}.ReplenishEnable` | ReplenishEnable | 补采开关 | 📝 RW | boolean |
| 9 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.ReplenishStartTime` | `Device.FAP.PerfMgmt.Config.{i}.ReplenishStartTime` | ReplenishStartTime | 补采开始时间 | 📝 RW | dateTime |
| 10 | `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.ReplenishEndTime` | `Device.FAP.PerfMgmt.Config.{i}.ReplenishEndTime` | ReplenishEndTime | 补采结束时间 | 📝 RW | dateTime |

---

## SR - ENanocell

**扩展型一体化皮基站参数**  
参数数量: 83

#### 命令: Device.DeviceInfo.MU.{i}.* 📖📝

- **{i}** 取值范围: `1~N` — 主机单元(MU)实例编号

只读(📖): 21 | 可写(📝): 3

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.UserLabel` | `Device.DeviceInfo.MU.{i}.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.DnPrefix` | `Device.DeviceInfo.MU.{i}.DnPrefix` | DnPrefix | DN前缀 | 📝 RW | string |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.ManufacturerOUI` | `Device.DeviceInfo.MU.{i}.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | 📖 R | string(6) |
| 4 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Manufacturer` | `Device.DeviceInfo.MU.{i}.Manufacturer` | Manufacturer | 制造商 | 📖 R | string(64) |
| 5 | `InternetGatewayDevice.DeviceInfo.MU.{i}.ModelName` | `Device.DeviceInfo.MU.{i}.ModelName` | ModelName | 设备型号 | 📖 R | string(64) |
| 6 | `InternetGatewayDevice.DeviceInfo.MU.{i}.VendorUnitFamilyType` | `Device.DeviceInfo.MU.{i}.VendorUnitFamilyType` | VendorUnitFamilyType | 归属类型 | 📖 R | string |
| 7 | `InternetGatewayDevice.DeviceInfo.MU.{i}.VendorUnitTypeNumber` | `Device.DeviceInfo.MU.{i}.VendorUnitTypeNumber` | VendorUnitTypeNumber | 资产单元类型版本号 | 📖 R | string |
| 8 | `InternetGatewayDevice.DeviceInfo.MU.{i}.SerialNumber` | `Device.DeviceInfo.MU.{i}.SerialNumber` | SerialNumber | 序列号 | 📖 R | string(64) |
| 9 | `InternetGatewayDevice.DeviceInfo.MU.{i}.HardwareVersion` | `Device.DeviceInfo.MU.{i}.HardwareVersion` | HardwareVersion | 硬件版本 | 📖 R | string(64) |
| 10 | `InternetGatewayDevice.DeviceInfo.MU.{i}.SoftwareVersion` | `Device.DeviceInfo.MU.{i}.SoftwareVersion` | SoftwareVersion | 软件版本 | 📖 R | string(64) |
| 11 | `InternetGatewayDevice.DeviceInfo.MU.{i}.HardwarePlatform` | `Device.DeviceInfo.MU.{i}.HardwarePlatform` | HardwarePlatform | 硬件平台 | 📖 R | string(64) |
| 12 | `InternetGatewayDevice.DeviceInfo.MU.{i}.AdditionalHardwareVersion` | `Device.DeviceInfo.MU.{i}.AdditionalHardwareVersion` | AdditionalHardwareVersion | 附加硬件版本 | 📖 R | string(64) |
| 13 | `InternetGatewayDevice.DeviceInfo.MU.{i}.AdditionalSoftwareVersion` | `Device.DeviceInfo.MU.{i}.AdditionalSoftwareVersion` | AdditionalSoftwareVersion | 附加软件版本 | 📖 R | string(64) |
| 14 | `InternetGatewayDevice.DeviceInfo.MU.{i}.ProvisioningCode` | `Device.DeviceInfo.MU.{i}.ProvisioningCode` | ProvisioningCode | 供应商代号 | 📖 R | string(64) |
| 15 | `InternetGatewayDevice.DeviceInfo.MU.{i}.ProductClass` | `Device.DeviceInfo.MU.{i}.ProductClass` | ProductClass | 产品分类 | 📖 R | string(64) |
| 16 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Status` | `Device.DeviceInfo.MU.{i}.Status` | Status | 机框/服务器状态 | 📖 R | unsignedInt[1:3] |
| 17 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Reboot` | `Device.DeviceInfo.MU.{i}.Reboot` | Reboot | 重启开关 | 📝 RW | boolean |
| 18 | `InternetGatewayDevice.DeviceInfo.MU.{i}.UpTime` | `Device.DeviceInfo.MU.{i}.UpTime` | UpTime | 运行时间 | 📖 R | unsignedInt |
| 19 | `InternetGatewayDevice.DeviceInfo.MU.{i}.FirstUseDate` | `Device.DeviceInfo.MU.{i}.FirstUseDate` | FirstUseDate | 首次使用日期 | 📖 R | dateTime |
| 20 | `InternetGatewayDevice.DeviceInfo.MU.{i}.ClockSource` | `Device.DeviceInfo.MU.{i}.ClockSource` | ClockSource | 时钟源类型 | 📖 R | unsignedInt[1:11} |
| 21 | `InternetGatewayDevice.DeviceInfo.MU.{i}.DateOfLastService` | `Device.DeviceInfo.MU.{i}.DateOfLastService` | DateOfLastService | 最近服务的日期（最近一次恢复工作正常状态的时间） | 📖 R | dateTime |
| 22 | `InternetGatewayDevice.DeviceInfo.MU.{i}.DateOfManufacture` | `Device.DeviceInfo.MU.{i}.DateOfManufacture` | DateOfManufacture | 生产日期 | 📖 R | dateTime |
| 23 | `InternetGatewayDevice.DeviceInfo.MU.{i}.ManufacturerData` | `Device.DeviceInfo.MU.{i}.ManufacturerData` | ManufacturerData | 特殊信息 | 📖 R | string |
| 24 | `InternetGatewayDevice.DeviceInfo.MU.{i}.SlotsInformation` | `Device.DeviceInfo.MU.{i}.SlotsInformation` | SlotsInformation | 插槽信息 | 📖 R | string |

#### 命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.* 📖📝

- **{iα}** 取值范围: `1~N` — 主机单元(MU)实例编号
- **{iβ}** 取值范围: `0~N` — 板卡槽位实例编号

只读(📖): 15 | 可写(📝): 1

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.PackPosition` | `Device.DeviceInfo.MU.{i}.Slot.{i}.PackPosition` | PackPosition | 板卡位置 | 📖 R | string(64) |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.SlotsOccupied` | `Device.DeviceInfo.MU.{i}.Slot.{i}.SlotsOccupied` | SlotsOccupied | 占用槽位 | 📖 R | string(64) |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.ManufacturerOUI` | `Device.DeviceInfo.MU.{i}.Slot.{i}.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | 📖 R | string(6) |
| 4 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.Manufacturer` | `Device.DeviceInfo.MU.{i}.Slot.{i}.Manufacturer` | Manufacturer | 制造商 | 📖 R | string(64) |
| 5 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.ModelName` | `Device.DeviceInfo.MU.{i}.Slot.{i}.ModelName` | ModelName | 设备型号 | 📖 R | string(64) |
| 6 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.SerialNumber` | `Device.DeviceInfo.MU.{i}.Slot.{i}.SerialNumber` | SerialNumber | 序列号 | 📖 R | string(64) |
| 7 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.HardwareVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.HardwareVersion` | HardwareVersion | 硬件版本 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.SoftwareVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.SoftwareVersion` | SoftwareVersion | 软件版本 | 📖 R | string(64) |
| 9 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.ProvisioningCode` | `Device.DeviceInfo.MU.{i}.Slot.{i}.ProvisioningCode` | ProvisioningCode | 供应商代号 | 📖 R | string(64) |
| 10 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.VendorUnitFamilyType` | `Device.DeviceInfo.MU.{i}.Slot.{i}.VendorUnitFamilyType` | VendorUnitFamilyType | 板卡类型 | 📖 R | string(64) |
| 11 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.UpTime` | `Device.DeviceInfo.MU.{i}.Slot.{i}.UpTime` | UpTime | 运行时间 | 📖 R | unsignedInt |
| 12 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.DataModelSpecVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.DataModelSpecVersion` | DataModelSpecVersion | 数据模型版本 | 📖 R | string |
| 13 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.3GPPSpecVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.3GPPSpecVersion` | 3GPPSpecVersion | 3GPP协议版本 | 📖 R | string |
| 14 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.FirstUseDate` | `Device.DeviceInfo.MU.{i}.Slot.{i}.FirstUseDate` | FirstUseDate | 首次使用日期 | 📖 R | dateTime |
| 15 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.Status` | `Device.DeviceInfo.MU.{i}.Slot.{i}.Status` | Status | 板卡状态 | 📖 R | unsignedInt[1:3] |
| 16 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.Reboot` | `Device.DeviceInfo.MU.{i}.Slot.{i}.Reboot` | Reboot | 重启开关 | 📝 RW | boolean |

#### 命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.* 📖📝

- **{iα}** 取值范围: `1~N` — 主机单元(MU)实例编号
- **{iβ}** 取值范围: `0~N` — 板卡槽位实例编号
- **{iγ}** 取值范围: `0~65535` — 扩展单元(EU)实例编号（i=0仅用于远端单元直连主机单元）

只读(📖): 11 | 可写(📝): 2

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.UserLabel` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RouteIndex` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RouteIndex` | RouteIndex | 扩展单元路由指示 | 📖 R | string(64) |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ManufacturerOUI` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | 📖 R | string(6) |
| 4 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Manufacturer` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Manufacturer` | Manufacturer | 制造商 | 📖 R | string(64) |
| 5 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ModelName` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ModelName` | ModelName | 设备型号 | 📖 R | string(64) |
| 6 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SerialNumber` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SerialNumber` | SerialNumber | 序列号 | 📖 R | string(64) |
| 7 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.HardwareVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.HardwareVersion` | HardwareVersion | 硬件版本 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SoftwareVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SoftwareVersion` | SoftwareVersion | 软件版本 | 📖 R | string(64) |
| 9 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ProvisioningCode` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ProvisioningCode` | ProvisioningCode | 供应商代号 | 📖 R | string(64) |
| 10 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Status` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Status` | Status | 状态 | 📖 R | unsignedInt[1:3] |
| 11 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.DLCRCSum` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.DLCRCSum` | DLCRCSum | 扩展单元下行误码率 | 📖 R | string(64) |
| 12 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ULCRCSum` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.ULCRCSum` | ULCRCSum | 扩展单元上行误码率 | 📖 R | string(64) |
| 13 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Reboot` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.Reboot` | Reboot | 重启开关 | 📝 RW | boolean |

#### 命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.* 📖📝

- **{iα}** 取值范围: `1~N` — 主机单元(MU)实例编号
- **{iβ}** 取值范围: `0~N` — 板卡槽位实例编号
- **{iγ}** 取值范围: `0~65535` — 扩展单元(EU)实例编号（i=0仅用于远端单元直连主机单元）
- **{iδ}** 取值范围: `0~65535` — 远端单元(RU)实例编号

只读(📖): 12 | 可写(📝): 4

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.UserLabel` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.UserLabel` | UserLabel | 用户友好名 | 📝 RW | string |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.VendorUnitFamilyType` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.VendorUnitFamilyType` | VendorUnitFamilyType | 归属类型 | 📖 R | string |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.VendorUnitTypeNumber` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.VendorUnitTypeNumber` | VendorUnitTypeNumber | 资产单元类型版本号 | 📖 R | string |
| 4 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RouteIndex` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RouteIndex` | RouteIndex | 射频单元路由指示 | 📖 R | string(64) |
| 5 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Status` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Status` | Status | 状态 | 📖 R | unsignedInt[1:3] |
| 6 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ManufacturerOUI` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ManufacturerOUI` | ManufacturerOUI | 制造商OUI | 📖 R | string(6) |
| 7 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Manufacturer` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Manufacturer` | Manufacturer | 制造商 | 📖 R | string(64) |
| 8 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ModelName` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ModelName` | ModelName | 设备型号 | 📖 R | string(64) |
| 9 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SerialNumber` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SerialNumber` | SerialNumber | 序列号 | 📖 R | string(64) |
| 10 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.HardwareVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.HardwareVersion` | HardwareVersion | 硬件版本 | 📖 R | string(64) |
| 11 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SoftwareVersion` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SoftwareVersion` | SoftwareVersion | 软件版本 | 📖 R | string(64) |
| 12 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ProvisioningCode` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.ProvisioningCode` | ProvisioningCode | 供应商代号 | 📖 R | string(64) |
| 13 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Reboot` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.Reboot` | Reboot | 重启开关 | 📝 RW | boolean |
| 14 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.FrequencyBand` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.FrequencyBand` | FrequencyBand | 支持频段 | 📝 RW | string |
| 15 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFTxStatus` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFTxStatus` | RFTxStatus | 射频状态 | 📝 RW | boolean |
| 16 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.DateOfManufacture` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.DateOfManufacture` | DateOfManufacture | 生产日期 | 📖 R | dateTime |

#### 命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.RFChannel.{iε}.* 📖📝

- **{iα}** 取值范围: `1~N` — 主机单元(MU)实例编号
- **{iβ}** 取值范围: `0~N` — 板卡槽位实例编号
- **{iγ}** 取值范围: `0~65535` — 扩展单元(EU)实例编号（i=0仅用于远端单元直连主机单元）
- **{iδ}** 取值范围: `0~65535` — 远端单元(RU)实例编号
- **{iε}** 取值范围: `0~N` — 射频通道实例编号

只读(📖): 1 | 可写(📝): 1

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.TxGain` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.TxGain` | TxGain | 下行发射功率增益 | 📝 RW | string |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.NoisePwdBm` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.RFChannel.{i}.NoisePwdBm` | NoisePwdBm | 底噪 | 📖 R | string |

#### 命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.SwUpgrade.* 📖

- **{iα}** 取值范围: `1~N` — 主机单元(MU)实例编号
- **{iβ}** 取值范围: `0~N` — 板卡槽位实例编号
- **{iγ}** 取值范围: `0~65535` — 扩展单元(EU)实例编号（i=0仅用于远端单元直连主机单元）
- **{iδ}** 取值范围: `0~65535` — 远端单元(RU)实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.Stage` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.Stage` | Stage | 升级阶段 | 📖 R | unsignedInt[1:5] |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.Status` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.Status` | Status | 状态 | 📖 R | unsignedInt[1:3] |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.FailureCause` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.RU.{i}.SwUpgrade.FailureCause` | FailureCause | 升级失败原因 | 📖 R | string(512) |

#### 命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.SwUpgrade.* 📖

- **{iα}** 取值范围: `1~N` — 主机单元(MU)实例编号
- **{iβ}** 取值范围: `0~N` — 板卡槽位实例编号
- **{iγ}** 取值范围: `0~65535` — 扩展单元(EU)实例编号（i=0仅用于远端单元直连主机单元）

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.Stage` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.Stage` | Stage | 升级阶段 | 📖 R | unsignedInt[1:5] |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.Status` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.Status` | Status | 状态 | 📖 R | unsignedInt[1:3] |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.FailureCause` | `Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.SwUpgrade.FailureCause` | FailureCause | 升级失败原因 | 📖 R | string(512) |

#### 命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.SwUpgrade.* 📖

- **{iα}** 取值范围: `1~N` — 主机单元(MU)实例编号
- **{iβ}** 取值范围: `0~N` — 板卡槽位实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.Stage` | `Device.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.Stage` | Stage | 升级阶段 | 📖 R | unsignedInt[1:5] |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.FailureCause` | `Device.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.FailureCause` | FailureCause | 升级失败原因 | 📖 R | string(512) |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.Status` | `Device.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.Status` | Status | 状态 | 📖 R | unsignedInt[1:3] |

#### 命令: Device.DeviceInfo.MU.{i}.SwUpgrade.* 📖

- **{i}** 取值范围: `1~N` — 主机单元(MU)实例编号

| # | TR-098 路径 | TR-181 路径 | 参数名 | 中文名 | 权限 | 类型 |
|---|------------|------------|--------|--------|------|------|
| 1 | `InternetGatewayDevice.DeviceInfo.MU.{i}.SwUpgrade.Stage` | `Device.DeviceInfo.MU.{i}.SwUpgrade.Stage` | Stage | 升级阶段 | 📖 R | unsignedInt[1:5] |
| 2 | `InternetGatewayDevice.DeviceInfo.MU.{i}.SwUpgrade.FailureCause` | `Device.DeviceInfo.MU.{i}.SwUpgrade.FailureCause` | FailureCause | 升级失败原因 | 📖 R | string(512) |
| 3 | `InternetGatewayDevice.DeviceInfo.MU.{i}.SwUpgrade.Status` | `Device.DeviceInfo.MU.{i}.SwUpgrade.Status` | Status | 状态 | 📖 R | unsignedInt[1:3] |

---

## 附录A: 多实例参数参考表

| TR-098 对象路径 | TR-181 对象路径 | 中文名称 | 读写属性 |
|----------------|----------------|---------|---------|
| `InternetGatewayDevice.Services.FAPService.{i}.{i}` | `Device.Services.FAPService.{i}.{i}` | FAPService实例节点 | R |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.{i}` | LTE邻区实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{i}.{i}` | UMTS邻区实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.{i}` | GSM邻区实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{i}.{i}` | A1测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{i}.{i}` | A2测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{i}.{i}` | A3测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{i}.{i}` | A4测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{i}.{i}` | A5测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.B1MeasureCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{i}.{i}` | B1测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.B2MeasureCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{i}.{i}` | B2测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.{i}` | 异频重选实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{i}.{i}` | 异系统重选实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{i}.{i}` | 周期性测量实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{i}.{i}` | SFConfigList实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{i}.{i}` | DrxInitialParam实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}.{i}` | MmePoolConfigParam实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.FAPService.{i}.FAPControl.X2IpAddrMapInfo.{i}.{i}` | `Device.Services.FAPService.{i}.FAPService.{i}.FAPControl.X2IpAddrMapInfo.{i}.{i}` | X2实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.{i}` | PLMNID实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.S1U.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.S1U.{i}.{i}` | S1U实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.{i}.{i}` | `Device.Services.FAPService.{i}.CellConfig.LTE.VoLTE.PdcpInitParam.{i}.{i}` | VOLTE 实例节点 | RW |
| `InternetGatewayDevice.Services.FAPService.{i}.Transport.SCTP.Assoc.{i}.{i}` | `InternetGatewayDevice.Services.FAPService.{i}.Transport.SCTP.Assoc.{i}.{i}` | SCTP实例节点 | RW |
| `InternetGatewayDevice.WANDevice.{i}.{i}` | `Device.WANDevice.{i}.{i}` | WAN实例 | R |
| `InternetGatewayDevice.WANDevice.{i}.WANConnectionDevice.{i}.WANIPConnection.{i}.{i}` | `Device.WANDevice.{i}.WANConnectionDevice.{i}.WANIPConnection.{i}.{i}` | WANIP实例节点 | R |
| `InternetGatewayDevice.FaultMgmt.SupportedAlarm.{i}.{i}` | `Device.FaultMgmt.SupportedAlarm.{i}.{i}` | 支持告警实例节点 | R |
| `InternetGatewayDevice.FaultMgmt.CurrentAlarm.{i}.{i}` | `Device.FaultMgmt.CurrentAlarm.{i}.{i}` | 当前告警实例节点 | R |
| `InternetGatewayDevice.FaultMgmt.HistoryEvent.{i}.{i}` | `Device.FaultMgmt.HistoryEvent.{i}.{i}` | 历史告警实例节点 | R |
| `InternetGatewayDevice.FaultMgmt.ExpeditedEvent.{i}.{i}` | `Device.FaultMgmt.ExpeditedEvent.{i}.{i}` | 实时告警实例节点 | R |
| `InternetGatewayDevice.FaultMgmt.QueuedEvent.{i}.{i}` | `Device.FaultMgmt.QueuedEvent.{i}.{i}` | 队列告警实例节点 | R |
| `InternetGatewayDevice.FAP.MRMgmt.Config.{i}.{i}` | `Device.FAP.MRMgmt.Config.{i}.{i}` | MR实例节点 | R |
| `InternetGatewayDevice.FAP.PerfMgmt.Config.{i}.{i}` | `Device.FAP.PerfMgmt.Config.{i}.{i}` | PM实例节点 | R |
| `InternetGatewayDevice.DeviceInfo.EU.{i}.{i}` | `Device.DeviceInfo.EU.{i}.{i}` | EU实例节点 | RW |
| `InternetGatewayDevice.DeviceInfo.EU.{i}.RU.{i}.{i}` | `Device.DeviceInfo.EU.{i}.RU.{i}.{i}` | RU实例节点 | RW |
| `InternetGatewayDevice.DeviceInfo.EU.{i}.RU.{i}.LTECell.{i}.{i}` | `Device.DeviceInfo.EU.{i}.RU.{i}.LTECell.{i}.{i}` | RU邻区实例节点 | RW |

---

## 参数管理类别分组清单（派生）

> 本节由 spec doc 中所有 `## SX - ...` 大节标题及其下的 `#### 命令: ...`
> 标题机械汇总而成；以 §（第 295 行）"参数管理类别" 主表为索引：
> **一个 SA-SR 类别 = 一个分组**。共 **18 个分组**、**72 个命令**（已剔除 1 个伪命令 / 规范引用注释段）。
>
> 与 MML 控制台的对应关系：
> - 本节的"分组"对应控制台命令树的一级 chapter 节点（SA-SR）
> - 分组下的"命令"对应控制台二级 group 节点（每个 group 按 op_type 展开
>   为 LST / MOD / ADD / RMV 子命令）
> - family 维度的进一步合并（72 → ~40 顶层项）见
>   `docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md` §15.4
>
> **权限图例**：📖 只读（GPV-only）/ 📝 可写（GPV + SPV）/ 📖📝 混合

### 速览（18 分组 / 72 spec 命令 / 228 派生 op）

| 索引 | 对象路径根 | 分组中文名 | spec 命令数 | 派生 op 数 |
|------|-----------|-----------|-----:|-----:|
| SA | `DeviceInfo` | 设备信息参数管理 | 2 | 3 |
| SB | `SoftwareCtrl` | 软件版本参数管理 | 1 | 2 |
| SC | `ManagementServer` | 基站网管参数管理 | 1 | 2 |
| SD | `FaultMgmt` | 告警参数管理 | 6 | 9 |
| SE | `DeviceLogMgmt` | 日志参数管理 | 1 | 2 |
| SF | `Services.FAPService` | 小区服务参数管理（总体） | 11 | 37 |
| SG | `Services.FAPService.{i}.SCTP.Transport` | SCTP 参数管理 | 2 | 3 |
| SH | `Services.FAPService.{i}.CellConfig.LTE.RAN` | RAN 协议栈参数 | 6 | 24 |
| SI | `Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList` | 邻区参数管理 | 4 | 16 |
| SJ | `Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility` | 移动性参数管理 | 15 | 60 |
| SK | `Services.FAPService.{i}.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam` | SON 参数管理 | 2 | 5 |
| SL | `WANDevice` | WAN 口配置参数管理 | 7 | 28 |
| SM | `Ipsec` | IPsec 参数管理 | 1 | 2 |
| SN | `Time` | 时间服务器参数管理 | 1 | 2 |
| SO | `FAP.GPS` | GPS 信息参数管理 | 1 | 1 |
| SP | `FAP.MRMgmt` | MR 参数管理 | 1 | 4 |
| SQ | `FAP.PerfMgmt` | 性能参数管理 | 1 | 4 |
| SR | `ENanocell` | 扩展型一体化皮基站参数 | 9 | 24 |

### 分组明细（按 R-3 op-split 派生）

> **派生规则（R-3，详见调整方案 §4）**：每个 spec `#### 命令: <path> <perm>` 按下表展开成 1-4 个子命令。
>
> | 条件 | 生成 |
> |------|------|
> | 总是 | **LST** `<对象名>` — GetParameterValues，覆盖 R + RW 全部 path |
> | RW > 0 | **MOD** `<对象名>` — SetParameterValues，仅 RW 子集 path |
> | arity ≥ 1 且 RW > 0 | **ADD** + **RMV** `<对象名>` — AddObject / DeleteObject |
> | RW = 0 | ❌ 不生成 MOD（路径全只读 → 没有可改 path → 命令不应存在） |
> | arity = 0 或 RW = 0 | ❌ 不生成 ADD/RMV |
>
> **⚠ 标记**：路径含 `{i}` 自动派生 ADD/RMV，但 TR-069 业务语义上不是用户可增删的容器（如固定枚举的 FAPService 载波 i=1~3、硬件描述的 MU/Slot/EU/RU 等）；后续 admin overlay 可隐藏。
>
> **命名约定**：派生命令的中文名 = `<OP> <对象中文名>`，例如 `LST 设备信息`、`MOD 软件控制`、`ADD MME 池配置`、`RMV LTE 邻区` 等。

#### SA · DeviceInfo — 设备信息参数管理

**G-01 · `Device.DeviceInfo.*` 📖📝**（R=15 / RW=2 / arity=0）

- **LST 设备信息** — 17 paths（GetParameterValues 全集）
- **MOD 设备信息** — 2 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

**G-02 · `Device.DeviceInfo.SwUpgrade.*` 📖**（R=3 / RW=0 / arity=0）

- **LST 设备版本升级** — 3 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（arity=0，非多实例对象）

#### SB · SoftwareCtrl — 软件版本参数管理

**G-03 · `Device.SoftwareCtrl.*` 📖📝**（R=2 / RW=3 / arity=0）

- **LST 软件控制** — 5 paths（GetParameterValues 全集）
- **MOD 软件控制** — 3 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

#### SC · ManagementServer — 基站网管参数管理

**G-04 · `Device.ManagementServer.*` 📖📝**（R=4 / RW=15 / arity=0）

- **LST 网管参数** — 19 paths（GetParameterValues 全集）
- **MOD 网管参数** — 15 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

#### SD · FaultMgmt — 告警参数管理

**G-05 · `Device.FaultMgmt.*` 📖**（R=6 / RW=0 / arity=0）

- **LST 故障管理** — 6 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（arity=0，非多实例对象）

**G-06 · `Device.FaultMgmt.CurrentAlarm.{i}.*` 📖**（R=11 / RW=0 / arity=1）

- **LST 当前告警实例** — 11 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-07 · `Device.FaultMgmt.ExpeditedEvent.{i}.*` 📖**（R=11 / RW=0 / arity=1）

- **LST 实时告警实例** — 11 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-08 · `Device.FaultMgmt.HistoryEvent.{i}.*` 📖**（R=11 / RW=0 / arity=1）

- **LST 历史告警实例** — 11 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-09 · `Device.FaultMgmt.QueuedEvent.{i}.*` 📖**（R=11 / RW=0 / arity=1）

- **LST 队列告警实例** — 11 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-10 · `Device.FaultMgmt.SupportedAlarm.{i}.*` 📖📝**（R=4 / RW=1 / arity=1）

- **LST 支持告警实例** — 5 paths（GetParameterValues 全集）
- **MOD 支持告警实例** — 1 paths（SetParameterValues，RW 子集）
- **ADD 支持告警实例** — AddObject，新增 `{i}` 实例
- **RMV 支持告警实例** — DeleteObject，删除 `{i}` 实例

#### SE · DeviceLogMgmt — 日志参数管理

**G-11 · `Device.LogMgmt.*` 📝**（R=0 / RW=5 / arity=0）

- **LST 日志管理** — 5 paths（GetParameterValues 全集）
- **MOD 日志管理** — 5 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

#### SF · Services.FAPService — 小区服务参数管理（总体）

**G-12 · `Device.Services.FAPControl.LTE.*` 📖📝**（R=2 / RW=1 / arity=0）

- **LST FAPControl LTE** — 3 paths（GetParameterValues 全集）
- **MOD FAPControl LTE** — 1 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

**G-13 · `Device.Services.FAPControl.LTE.Gateway.*` 📝**（R=0 / RW=10 / arity=0）

- **LST 安全/接入网关** — 10 paths（GetParameterValues 全集）
- **MOD 安全/接入网关** — 10 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

**G-14 · `Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.*` 📖📝**（R=3 / RW=2 / arity=1）

- **LST MME 池配置** — 5 paths（GetParameterValues 全集）
- **MOD MME 池配置** — 2 paths（SetParameterValues，RW 子集）
- **ADD MME 池配置** — AddObject，新增 `{i}` 实例
- **RMV MME 池配置** — DeleteObject，删除 `{i}` 实例

**G-15 · `Device.Services.FAPControl.LTE.S1U.{i}.*` 📖**（R=2 / RW=0 / arity=1）

- **LST S1U** — 2 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-16 · `Device.Services.FAPControl.X2IpAddrMapInfo.{i}.*` 📝**（R=0 / RW=5 / arity=1）

- **LST X2 IP 映射** — 5 paths（GetParameterValues 全集）
- **MOD X2 IP 映射** — 5 paths（SetParameterValues，RW 子集）
- **ADD X2 IP 映射** — AddObject，新增 `{i}` 实例
- **RMV X2 IP 映射** — DeleteObject，删除 `{i}` 实例

**G-17 · `Device.Services.FAPService.{i}.*` 📖📝**（R=2 / RW=22 / arity=1）

- **LST FAPService 载波** — 24 paths（GetParameterValues 全集）
- **MOD FAPService 载波** — 22 paths（SetParameterValues，RW 子集）
- **ADD FAPService 载波** — AddObject，新增 `{i}` 实例 ⚠
- **RMV FAPService 载波** — DeleteObject，删除 `{i}` 实例 ⚠

**G-18 · `Device.Services.FAPService.{i}.Capabilities.*` 📝**（R=0 / RW=1 / arity=1）

- **LST FAPService Capabilities** — 1 paths（GetParameterValues 全集）
- **MOD FAPService Capabilities** — 1 paths（SetParameterValues，RW 子集）
- **ADD FAPService Capabilities** — AddObject，新增 `{i}` 实例 ⚠
- **RMV FAPService Capabilities** — DeleteObject，删除 `{i}` 实例 ⚠

**G-19 · `Device.Services.FAPService.{i}.CellConfig.Capabilities.*` 📖📝**（R=2 / RW=1 / arity=1）

- **LST CellConfig Capabilities** — 3 paths（GetParameterValues 全集）
- **MOD CellConfig Capabilities** — 1 paths（SetParameterValues，RW 子集）
- **ADD CellConfig Capabilities** — AddObject，新增 `{i}` 实例 ⚠
- **RMV CellConfig Capabilities** — DeleteObject，删除 `{i}` 实例 ⚠

**G-20 · `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.*` 📝**（R=0 / RW=2 / arity=1）

- **LST EPC** — 2 paths（GetParameterValues 全集）
- **MOD EPC** — 2 paths（SetParameterValues，RW 子集）
- **ADD EPC** — AddObject，新增 `{i}` 实例 ⚠
- **RMV EPC** — DeleteObject，删除 `{i}` 实例 ⚠

**G-21 · `Device.Services.FAPService.{iα}.CellConfig.LTE.EPC.PLMNList.{iβ}.*` 📝**（R=0 / RW=2 / arity=2）

- **LST PLMN 列表** — 2 paths（GetParameterValues 全集）
- **MOD PLMN 列表** — 2 paths（SetParameterValues，RW 子集）
- **ADD PLMN 列表** — AddObject，新增 `{i}` 实例
- **RMV PLMN 列表** — DeleteObject，删除 `{i}` 实例

**G-22 · `Device.Services.FAPService.{iα}.CellConfig.LTE.VoLTE.PdcpInitParam.{iβ}.*` 📝**（R=0 / RW=1 / arity=2）

- **LST VoLTE PDCP 初始** — 1 paths（GetParameterValues 全集）
- **MOD VoLTE PDCP 初始** — 1 paths（SetParameterValues，RW 子集）
- **ADD VoLTE PDCP 初始** — AddObject，新增 `{i}` 实例
- **RMV VoLTE PDCP 初始** — DeleteObject，删除 `{i}` 实例

#### SG · Services.FAPService.{i}.SCTP.Transport — SCTP 参数管理

**G-23 · `Device.Services.FAPControl.Transport.SCTP.*` 📝**（R=0 / RW=9 / arity=0）

- **LST SCTP** — 9 paths（GetParameterValues 全集）
- **MOD SCTP** — 9 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

**G-24 · `Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.*` 📖**（R=4 / RW=0 / arity=1）

- **LST SCTP Assoc** — 4 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

#### SH · Services.FAPService.{i}.CellConfig.LTE.RAN — RAN 协议栈参数

**G-25 · `Device.Services.FAPService.{i}.*` 📝**（R=0 / RW=10 / arity=1）

- **LST RRC Timers** — 10 paths（GetParameterValues 全集）
- **MOD RRC Timers** — 10 paths（SetParameterValues，RW 子集）
- **ADD RRC Timers** — AddObject，新增 `{i}` 实例 ⚠
- **RMV RRC Timers** — DeleteObject，删除 `{i}` 实例 ⚠

**G-26 · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.*` 📝**（R=0 / RW=16 / arity=1）

- **LST MAC** — 16 paths（GetParameterValues 全集）
- **MOD MAC** — 16 paths（SetParameterValues，RW 子集）
- **ADD MAC** — AddObject，新增 `{i}` 实例 ⚠
- **RMV MAC** — DeleteObject，删除 `{i}` 实例 ⚠

**G-27 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.MAC.DrxInitialParam.{iβ}.*` 📝**（R=0 / RW=6 / arity=2）

- **LST DRX 初始** — 6 paths（GetParameterValues 全集）
- **MOD DRX 初始** — 6 paths（SetParameterValues，RW 子集）
- **ADD DRX 初始** — AddObject，新增 `{i}` 实例
- **RMV DRX 初始** — DeleteObject，删除 `{i}` 实例

**G-28 · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.*` 📖📝**（R=2 / RW=33 / arity=1）

- **LST PHY** — 35 paths（GetParameterValues 全集）
- **MOD PHY** — 33 paths（SetParameterValues，RW 子集）
- **ADD PHY** — AddObject，新增 `{i}` 实例 ⚠
- **RMV PHY** — DeleteObject，删除 `{i}` 实例 ⚠

**G-29 · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.*` 📝**（R=0 / RW=1 / arity=1）

- **LST PHY MBSFN** — 1 paths（GetParameterValues 全集）
- **MOD PHY MBSFN** — 1 paths（SetParameterValues，RW 子集）
- **ADD PHY MBSFN** — AddObject，新增 `{i}` 实例 ⚠
- **RMV PHY MBSFN** — DeleteObject，删除 `{i}` 实例 ⚠

**G-30 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.PHY.MBSFN.SFConfigList.{iβ}.*` 📝**（R=0 / RW=5 / arity=2）

- **LST MBSFN SFConfigList** — 5 paths（GetParameterValues 全集）
- **MOD MBSFN SFConfigList** — 5 paths（SetParameterValues，RW 子集）
- **ADD MBSFN SFConfigList** — AddObject，新增 `{i}` 实例
- **RMV MBSFN SFConfigList** — DeleteObject，删除 `{i}` 实例

#### SI · Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList — 邻区参数管理

**G-31 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{iβ}.*` 📝**（R=0 / RW=7 / arity=2）

- **LST GSM 邻区** — 7 paths（GetParameterValues 全集）
- **MOD GSM 邻区** — 7 paths（SetParameterValues，RW 子集）
- **ADD GSM 邻区** — AddObject，新增 `{i}` 实例
- **RMV GSM 邻区** — DeleteObject，删除 `{i}` 实例

**G-32 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.NR.{iβ}.*` 📝**（R=0 / RW=12 / arity=2）

- **LST NR 邻区** — 12 paths（GetParameterValues 全集）
- **MOD NR 邻区** — 12 paths（SetParameterValues，RW 子集）
- **ADD NR 邻区** — AddObject，新增 `{i}` 实例
- **RMV NR 邻区** — DeleteObject，删除 `{i}` 实例

**G-33 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.InterRATCell.UMTS.{iβ}.*` 📝**（R=0 / RW=10 / arity=2）

- **LST UMTS 邻区** — 10 paths（GetParameterValues 全集）
- **MOD UMTS 邻区** — 10 paths（SetParameterValues，RW 子集）
- **ADD UMTS 邻区** — AddObject，新增 `{i}` 实例
- **RMV UMTS 邻区** — DeleteObject，删除 `{i}` 实例

**G-34 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.NeighborList.LTECell.{iβ}.*` 📝**（R=0 / RW=10 / arity=2）

- **LST LTE 邻区** — 10 paths（GetParameterValues 全集）
- **MOD LTE 邻区** — 10 paths（SetParameterValues，RW 子集）
- **ADD LTE 邻区** — AddObject，新增 `{i}` 实例
- **RMV LTE 邻区** — DeleteObject，删除 `{i}` 实例

#### SJ · Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility — 移动性参数管理

**G-35 · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.*` 📝**（R=0 / RW=1 / arity=1）

- **LST ConnMode EUTRA** — 1 paths（GetParameterValues 全集）
- **MOD ConnMode EUTRA** — 1 paths（SetParameterValues，RW 子集）
- **ADD ConnMode EUTRA** — AddObject，新增 `{i}` 实例 ⚠
- **RMV ConnMode EUTRA** — DeleteObject，删除 `{i}` 实例 ⚠

**G-36 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A1MeasureCtrl.{iβ}.*` 📖📝**（R=1 / RW=10 / arity=2）

- **LST A1 测量控制** — 11 paths（GetParameterValues 全集）
- **MOD A1 测量控制** — 10 paths（SetParameterValues，RW 子集）
- **ADD A1 测量控制** — AddObject，新增 `{i}` 实例
- **RMV A1 测量控制** — DeleteObject，删除 `{i}` 实例

**G-37 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A2MeasureCtrl.{iβ}.*` 📖📝**（R=1 / RW=10 / arity=2）

- **LST A2 测量控制** — 11 paths（GetParameterValues 全集）
- **MOD A2 测量控制** — 10 paths（SetParameterValues，RW 子集）
- **ADD A2 测量控制** — AddObject，新增 `{i}` 实例
- **RMV A2 测量控制** — DeleteObject，删除 `{i}` 实例

**G-38 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3MeasureCtrl.{iβ}.*` 📖📝**（R=1 / RW=10 / arity=2）

- **LST A3 测量控制** — 11 paths（GetParameterValues 全集）
- **MOD A3 测量控制** — 10 paths（SetParameterValues，RW 子集）
- **ADD A3 测量控制** — AddObject，新增 `{i}` 实例
- **RMV A3 测量控制** — DeleteObject，删除 `{i}` 实例

**G-39 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A4MeasureCtrl.{iβ}.*` 📖📝**（R=1 / RW=10 / arity=2）

- **LST A4 测量控制** — 11 paths（GetParameterValues 全集）
- **MOD A4 测量控制** — 10 paths（SetParameterValues，RW 子集）
- **ADD A4 测量控制** — AddObject，新增 `{i}` 实例
- **RMV A4 测量控制** — DeleteObject，删除 `{i}` 实例

**G-40 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A5MeasureCtrl.{iβ}.*` 📖📝**（R=1 / RW=12 / arity=2）

- **LST A5 测量控制** — 13 paths（GetParameterValues 全集）
- **MOD A5 测量控制** — 12 paths（SetParameterValues，RW 子集）
- **ADD A5 测量控制** — AddObject，新增 `{i}` 实例
- **RMV A5 测量控制** — DeleteObject，删除 `{i}` 实例

**G-41 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.PeriodMeasCtrl.{iβ}.*` 📝**（R=0 / RW=4 / arity=2）

- **LST 周期测量控制** — 4 paths（GetParameterValues 全集）
- **MOD 周期测量控制** — 4 paths（SetParameterValues，RW 子集）
- **ADD 周期测量控制** — AddObject，新增 `{i}` 实例
- **RMV 周期测量控制** — DeleteObject，删除 `{i}` 实例

**G-42 · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.*` 📝**（R=0 / RW=4 / arity=1）

- **LST ConnMode IRAT** — 4 paths（GetParameterValues 全集）
- **MOD ConnMode IRAT** — 4 paths（SetParameterValues，RW 子集）
- **ADD ConnMode IRAT** — AddObject，新增 `{i}` 实例 ⚠
- **RMV ConnMode IRAT** — DeleteObject，删除 `{i}` 实例 ⚠

**G-43 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B1MeasureCtrl.{iβ}.*` 📖📝**（R=1 / RW=10 / arity=2）

- **LST B1 测量控制** — 11 paths（GetParameterValues 全集）
- **MOD B1 测量控制** — 10 paths（SetParameterValues，RW 子集）
- **ADD B1 测量控制** — AddObject，新增 `{i}` 实例
- **RMV B1 测量控制** — DeleteObject，删除 `{i}` 实例

**G-44 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.B2MeasureCtrl.{iβ}.*` 📖📝**（R=1 / RW=12 / arity=2）

- **LST B2 测量控制** — 13 paths（GetParameterValues 全集）
- **MOD B2 测量控制** — 12 paths（SetParameterValues，RW 子集）
- **ADD B2 测量控制** — AddObject，新增 `{i}` 实例
- **RMV B2 测量控制** — DeleteObject，删除 `{i}` 实例

**G-45 · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.*` 📝**（R=0 / RW=28 / arity=1）

- **LST IdleMode** — 28 paths（GetParameterValues 全集）
- **MOD IdleMode** — 28 paths（SetParameterValues，RW 子集）
- **ADD IdleMode** — AddObject，新增 `{i}` 实例 ⚠
- **RMV IdleMode** — DeleteObject，删除 `{i}` 实例 ⚠

**G-46 · `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.*` 📝**（R=0 / RW=2 / arity=1）

- **LST IdleMode IRAT** — 2 paths（GetParameterValues 全集）
- **MOD IdleMode IRAT** — 2 paths（SetParameterValues，RW 子集）
- **ADD IdleMode IRAT** — AddObject，新增 `{i}` 实例 ⚠
- **RMV IdleMode IRAT** — DeleteObject，删除 `{i}` 实例 ⚠

**G-47 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{iβ}.*` 📝**（R=0 / RW=6 / arity=2）

- **LST GERAN 频组** — 6 paths（GetParameterValues 全集）
- **MOD GERAN 频组** — 6 paths（SetParameterValues，RW 子集）
- **ADD GERAN 频组** — AddObject，新增 `{i}` 实例
- **RMV GERAN 频组** — DeleteObject，删除 `{i}` 实例

**G-48 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.UTRANFDDFreq.{iβ}.*` 📝**（R=0 / RW=6 / arity=2）

- **LST UTRA FDD 频点** — 6 paths（GetParameterValues 全集）
- **MOD UTRA FDD 频点** — 6 paths（SetParameterValues，RW 子集）
- **ADD UTRA FDD 频点** — AddObject，新增 `{i}` 实例
- **RMV UTRA FDD 频点** — DeleteObject，删除 `{i}` 实例

**G-49 · `Device.Services.FAPService.{iα}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{iβ}.*` 📝**（R=0 / RW=13 / arity=2）

- **LST 异频载波** — 13 paths（GetParameterValues 全集）
- **MOD 异频载波** — 13 paths（SetParameterValues，RW 子集）
- **ADD 异频载波** — AddObject，新增 `{i}` 实例
- **RMV 异频载波** — DeleteObject，删除 `{i}` 实例

#### SK · Services.FAPService.{i}.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam — SON 参数管理

**G-50 · `Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.*` 📖📝**（R=1 / RW=24 / arity=1）

- **LST SON 配置** — 25 paths（GetParameterValues 全集）
- **MOD SON 配置** — 24 paths（SetParameterValues，RW 子集）
- **ADD SON 配置** — AddObject，新增 `{i}` 实例 ⚠
- **RMV SON 配置** — DeleteObject，删除 `{i}` 实例 ⚠

**G-51 · `Device.Services.FAPService.{i}.FAPControl.SelfConfig.*` 📖**（R=3 / RW=0 / arity=1）

- **LST 自配置启动** — 3 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

#### SL · WANDevice — WAN 口配置参数管理

**G-52 · `Device.Ethernet.Interface.{i}.*` 📖📝**（R=5 / RW=4 / arity=1）

- **LST 以太网接口** — 9 paths（GetParameterValues 全集）
- **MOD 以太网接口** — 4 paths（SetParameterValues，RW 子集）
- **ADD 以太网接口** — AddObject，新增 `{i}` 实例
- **RMV 以太网接口** — DeleteObject，删除 `{i}` 实例

**G-53 · `Device.Ethernet.Interface.{iα}.IPv4Address.{iβ}.*` 📝**（R=0 / RW=5 / arity=2）

- **LST IPv4 地址** — 5 paths（GetParameterValues 全集）
- **MOD IPv4 地址** — 5 paths（SetParameterValues，RW 子集）
- **ADD IPv4 地址** — AddObject，新增 `{i}` 实例
- **RMV IPv4 地址** — DeleteObject，删除 `{i}` 实例

**G-54 · `Device.Ethernet.Interface.{iα}.IPv6Address.{iβ}.*` 📝**（R=0 / RW=5 / arity=2）

- **LST IPv6 地址** — 5 paths（GetParameterValues 全集）
- **MOD IPv6 地址** — 5 paths（SetParameterValues，RW 子集）
- **ADD IPv6 地址** — AddObject，新增 `{i}` 实例
- **RMV IPv6 地址** — DeleteObject，删除 `{i}` 实例

**G-55 · `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.*` 📝**（R=0 / RW=3 / arity=2）

- **LST VLAN 接口** — 3 paths（GetParameterValues 全集）
- **MOD VLAN 接口** — 3 paths（SetParameterValues，RW 子集）
- **ADD VLAN 接口** — AddObject，新增 `{i}` 实例
- **RMV VLAN 接口** — DeleteObject，删除 `{i}` 实例

**G-56 · `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv4Address.{iγ}.*` 📝**（R=0 / RW=5 / arity=3）

- **LST VLAN IPv4** — 5 paths（GetParameterValues 全集）
- **MOD VLAN IPv4** — 5 paths（SetParameterValues，RW 子集）
- **ADD VLAN IPv4** — AddObject，新增 `{i}` 实例
- **RMV VLAN IPv4** — DeleteObject，删除 `{i}` 实例

**G-57 · `Device.Ethernet.Interface.{iα}.VlanInterface.{iβ}.IPv6Address.{iγ}.*` 📝**（R=0 / RW=5 / arity=3）

- **LST VLAN IPv6** — 5 paths（GetParameterValues 全集）
- **MOD VLAN IPv6** — 5 paths（SetParameterValues，RW 子集）
- **ADD VLAN IPv6** — AddObject，新增 `{i}` 实例
- **RMV VLAN IPv6** — DeleteObject，删除 `{i}` 实例

**G-58 · `Device.Ethernet.IpRoute.{i}.*` 📝**（R=0 / RW=5 / arity=1）

- **LST IP 路由** — 5 paths（GetParameterValues 全集）
- **MOD IP 路由** — 5 paths（SetParameterValues，RW 子集）
- **ADD IP 路由** — AddObject，新增 `{i}` 实例
- **RMV IP 路由** — DeleteObject，删除 `{i}` 实例

#### SM · Ipsec — IPsec 参数管理

**G-59 · `Device.IPsec.*` 📖📝**（R=7 / RW=2 / arity=0）

- **LST IPsec** — 9 paths（GetParameterValues 全集）
- **MOD IPsec** — 2 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

#### SN · Time — 时间服务器参数管理

**G-60 · `Device.Time.*` 📖📝**（R=1 / RW=7 / arity=0）

- **LST 时间服务器** — 8 paths（GetParameterValues 全集）
- **MOD 时间服务器** — 7 paths（SetParameterValues，RW 子集）
- ❌ 不生成：ADD/RMV（arity=0，非多实例对象）

#### SO · FAP.GPS — GPS 信息参数管理

**G-61 · `Device.FAP.GPS.*` 📖**（R=3 / RW=0 / arity=0）

- **LST GPS** — 3 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（arity=0，非多实例对象）

#### SP · FAP.MRMgmt — MR 参数管理

**G-62 · `Device.FAP.MRMgmt.Config.{i}.*` 📝**（R=0 / RW=14 / arity=1）

- **LST MR 配置** — 14 paths（GetParameterValues 全集）
- **MOD MR 配置** — 14 paths（SetParameterValues，RW 子集）
- **ADD MR 配置** — AddObject，新增 `{i}` 实例
- **RMV MR 配置** — DeleteObject，删除 `{i}` 实例

#### SQ · FAP.PerfMgmt — 性能参数管理

**G-63 · `Device.FAP.PerfMgmt.Config.{i}.*` 📝**（R=0 / RW=10 / arity=1）

- **LST PM 配置** — 10 paths（GetParameterValues 全集）
- **MOD PM 配置** — 10 paths（SetParameterValues，RW 子集）
- **ADD PM 配置** — AddObject，新增 `{i}` 实例
- **RMV PM 配置** — DeleteObject，删除 `{i}` 实例

#### SR · ENanocell — 扩展型一体化皮基站参数

**G-64 · `Device.DeviceInfo.MU.{i}.*` 📖📝**（R=21 / RW=3 / arity=1）

- **LST MU 主机单元** — 24 paths（GetParameterValues 全集）
- **MOD MU 主机单元** — 3 paths（SetParameterValues，RW 子集）
- **ADD MU 主机单元** — AddObject，新增 `{i}` 实例 ⚠
- **RMV MU 主机单元** — DeleteObject，删除 `{i}` 实例 ⚠

**G-65 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.*` 📖📝**（R=15 / RW=1 / arity=2）

- **LST Slot 板卡** — 16 paths（GetParameterValues 全集）
- **MOD Slot 板卡** — 1 paths（SetParameterValues，RW 子集）
- **ADD Slot 板卡** — AddObject，新增 `{i}` 实例 ⚠
- **RMV Slot 板卡** — DeleteObject，删除 `{i}` 实例 ⚠

**G-66 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.*` 📖📝**（R=11 / RW=2 / arity=3）

- **LST EU 扩展单元** — 13 paths（GetParameterValues 全集）
- **MOD EU 扩展单元** — 2 paths（SetParameterValues，RW 子集）
- **ADD EU 扩展单元** — AddObject，新增 `{i}` 实例 ⚠
- **RMV EU 扩展单元** — DeleteObject，删除 `{i}` 实例 ⚠

**G-67 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.*` 📖📝**（R=12 / RW=4 / arity=4）

- **LST RU 远端单元** — 16 paths（GetParameterValues 全集）
- **MOD RU 远端单元** — 4 paths（SetParameterValues，RW 子集）
- **ADD RU 远端单元** — AddObject，新增 `{i}` 实例 ⚠
- **RMV RU 远端单元** — DeleteObject，删除 `{i}` 实例 ⚠

**G-68 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.RFChannel.{iε}.*` 📖📝**（R=1 / RW=1 / arity=5）

- **LST RFChannel 射频通道** — 2 paths（GetParameterValues 全集）
- **MOD RFChannel 射频通道** — 1 paths（SetParameterValues，RW 子集）
- **ADD RFChannel 射频通道** — AddObject，新增 `{i}` 实例 ⚠
- **RMV RFChannel 射频通道** — DeleteObject，删除 `{i}` 实例 ⚠

**G-69 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.SwUpgrade.*` 📖**（R=3 / RW=0 / arity=4）

- **LST RU 升级** — 3 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-70 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.SwUpgrade.*` 📖**（R=3 / RW=0 / arity=3）

- **LST EU 升级** — 3 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-71 · `Device.DeviceInfo.MU.{iα}.Slot.{iβ}.SwUpgrade.*` 📖**（R=3 / RW=0 / arity=2）

- **LST Slot 升级** — 3 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

**G-72 · `Device.DeviceInfo.MU.{i}.SwUpgrade.*` 📖**（R=3 / RW=0 / arity=1）

- **LST MU 升级** — 3 paths（GetParameterValues 全集）
- ❌ 不生成：MOD（路径全只读，无可写 path） · ADD/RMV（无 RW path，新增/删除实例无意义）

