# MML 控制台改版设计方案（v1，待审核）

> 日期：2026-06-03　状态：**待审核**　范围：`/mml/console`（前端 `pages/mml/Console/` + 后端 `internal/mml/`）

---

## 1. 背景与目标

现有 MML 控制台以「设备树 + 命令树 + 右侧操作/终端」三栏铺开，执行结果以**终端文本流**呈现。本次改版围绕四个诉求：

1. **结果表格化** —— 命令执行结果以表格形式显示。
2. **批量设备 + 结果汇聚下载** —— 批量选设备执行同一命令；每个设备结果汇总、单设备结果文件下载、全设备结果一次性下载。
3. **重排版面占比** —— 选设备/选命令收缩占比，主舞台让给「操作面板 + 参数路径指定 + 执行结果表格」。
4. **重做布局** —— 先出文档方案供审核，再实施。

---

## 2. 现状分析

### 2.1 前端布局现状

```
┌──────────────────────────────────────────────────────────────┐
│ StepBar（选设备 → 选命令 → 配参 → 执行）                        │
├──────────────┬───────────────┬─────────────────────────────────┤
│ DeviceTree   │ CommandTree   │ RightPanel（约 12/12 宽，最高）   │
│ 5–6 / 12     │ 7–8 / 12      │  Tabs:[控制面板][参数路径指定]    │
│ 设备多选树    │ 命令树(2 层)   │  · 操作面板(SubFieldChecklist/   │
│ 搜索/产品过滤 │ 搜索/模板      │    InputList/InstancePicker)320px│
│ 全选/已选标签 │ 操作类型彩 Tag │  · ConsoleActionBar(执行行)      │
│              │               │  · TerminalPanel(SSE 文本流 flex) │
└──────────────┴───────────────┴─────────────────────────────────┘
```

- 设备树 + 命令树合计吃掉左/中约 **12–14/12** 横向空间。
- 结果区是 `TerminalPanel`：深色终端、彩色行（stdout/stderr/info/success）、SSE 实时、可复制/清空。

### 2.2 后端能力现状（关键）

| 能力 | 现状 | 端点/实现 |
|---|---|---|
| 单/批量执行 | ✅ 已具备 | `POST /mml/execute-statements`、`POST /mml/console/execute-statements-structured`；`DeviceSNs[]` 驱动；`fanout.go` 逐 (设备×命令) 建 device_task |
| 批量扇出 | ✅ 已具备 | `Fanouter`（设备间并行、单设备内串行 Sequential）；`ResultAggregator` 回写 success/failed 计数 |
| 结果查询 | ⚠️ 部分 | `GET /mml/tasks/:id`（聚合统计）、`GET /mml/tasks/:id/results?page=`（per-device JSONB 行，含 `param_values`/`param_faults`） |
| 结果 schema/列定义 | ❌ 无 | 前端无法据此动态渲染表头 |
| 结果下载/导出 | ❌ 无 | 无 export/download 端点、无 CSV/XLSX、无 MinIO 落地 |
| 实时推送 | ✅ | SSE `mml_device_frame` / `mml_task_status` / `mml_task_completed` |

### 2.3 三需求现状评估

| 需求 | 现状 | 结论 |
|---|---|---|
| ① 批量多设备执行 | DeviceTree 多选 + BatchSnModal + 后端 `DeviceSNs[]` + Fanouter | **✅ 已具备**（仅前端体验需重排） |
| ② 结果表格化 | 仅终端文本流；后端有 per-device JSONB 但无 schema/转置 | **⚠️ 部分**（前端要表格化，后端要补 schema） |
| ③ 结果下载（单/全） | 仅「复制到剪贴板」；下载按钮未接 handler；后端无 export 端点 | **❌ 缺失**（前后端都要新增） |

---

## 3. 改版设计

### 3.1 设计原则

- **选择即收纳**：设备/命令是「输入条件」，选完即收起为摘要条，点击才展开（抽屉），不长期占版面。
- **结果即主角**：执行结果表格是页面主舞台（≥ 60% 面积）。
- **不破坏后端契约**：批量/执行链路全部复用，新增能力以「增量端点」交付，旧 SSE 文本流保留为「原始报文」备查。
- **表格随操作类型自适应**：LST=参数值矩阵；MOD/ADD/RMV=状态+故障矩阵。

### 3.2 新版页面布局（主推方案）

```
┌────────────────────────────────────────────────────────────────────────┐
│ ① 顶部选择条（1 行）                                                      │
│  [① 选择目标设备 ▸ 已选 12 台]   [② 选择 MML 命令 ▸ LST 查询设备基本信息] │
│   点击 → 弹出「设备选择弹框 / 命令选择弹框」(Modal)，确定后关闭并回填摘要   │
├──────────────────────────────────┬───────────────────────────────────────┤
│ ② 操作 & 参数区（左，~38%）        │ ③ 执行结果表格（右，~62%，主舞台）     │
│  ┌ 操作类型: ● LST ○ MOD ○ ADD…  │  ┌ 汇总条: 共 12 台 ✓10 ✗2 · 用时 8s   │
│  ├ 参数路径指定:                  │  │        [下载全部 ▾ CSV/XLSX/JSON]    │
│  │  · SubField 勾选 / 填值        │  ├ [设备过滤▾] [状态过滤: 全部/失败]    │
│  │  · 实例号选择({i})             │  ├─表格────────────────────────────┐  │
│  │  · 路径详情 Popover            │  │序│设备SN │状态│参数A│参数B│故障│⬇│  │
│  ├ 执行模式: ● 整体 ○ 单PATH      │  │1 │dev001│✓成功│v1  │v2  │ - │⤓│  │
│  └ [▶ 执行]  [清空]               │  │2 │dev002│✗失败│ -  │ - │9005│⤓│  │
│                                  │  │…按设备一行,失败行可展开看原始报文   │  │
│                                  │  └────────────────────────────────┘  │
│                                  │  ▸ 原始报文(SSE 文本流) 折叠面板/Tab    │
└──────────────────────────────────┴───────────────────────────────────────┘
```

要点：
- **顶部选择条**取代常驻的设备树+命令树两大栏，只显示「已选摘要」，点击弹出 **Modal 弹框**选择（设备弹框复用 DeviceTree+BatchSnModal；命令弹框复用 CommandTree），确定后关闭并回填（见 §3.3 两步弹框）。
- **左操作区**保留现有 SubFieldChecklist / SubFieldInputList / InstancePicker / ConsoleActionBar（几乎零改造，仅容器宽度变化）；顶部加「标准参数 / 参数路径指定」模式切换（见 §3.9）。
- **右结果区**为新增的**结果表格**，是本次重点；下方折叠保留原 TerminalPanel 作「原始报文」。

**备选布局（供对比）**：
- 备选 B「上下分区」：上半=操作&参数（横向铺开），下半=结果表格全宽。适合参数项极多、希望表格更宽的场景。
- 备选 C「选择条不折叠」：设备/命令选择常驻为左侧**窄抽屉栏**（可 pin/unpin），不用 Drawer。

### 3.3 选择交互：两步弹框（Modal）

**流程**：顶部条两个按钮「① 选择目标设备」「② 选择 MML 命令」→ 各自弹出 Modal → 确定后关闭并回填摘要，主舞台（操作/参数/结果表格）不被长期遮挡。第①步选完设备 → 第②步选命令 → 进入左操作区配参 → 执行。

#### 3.3.1 设备选择弹框

支持按 **设备编码(SN) / 产品 / 产品类型** 三维筛选（可组合）+ 表格多选 + 批量粘贴 SN，确定即关闭。

```
┌── 选择目标设备 ─────────────────────────────────────────────── [×] ──┐
│ 筛选:                                                                 │
│ [设备编码: 输入 SN 模糊搜索…] [产品: 全部 ▾] [产品类型: 全部 ▾] [查询][重置] │
├──────────────────────────────────────────────────────────────────────┤
│ ☐ 全选     已选 3 台                                   [批量粘贴 SN]   │
│ ┌──────────────────────────────────────────────────────────────────┐ │
│ │ ☑ │ 设备编码(SN)     │ 产品       │ 产品类型 │ 状态 │ 设备分组      │ │
│ │ ☑ │ 1202000…0026    │ Comba X1  │ CPE     │ 在线 │ 默认设备组    │ │
│ │ ☐ │ 1202000…0027    │ Baicells  │ ENB     │ 离线 │ 移动设备域    │ │
│ │ … （服务端分页，每页 50；列可排序）                                │ │
│ └──────────────────────────────────────────────────────────────────┘ │
│ 已选: [1202…0026 ×] [1202…0030 ×] [1202…0031 ×]                      │
├──────────────────────────────────────────────────────────────────────┤
│                                          [取消]   [确定（已选 3 台）]  │
└──────────────────────────────────────────────────────────────────────┘
```

- **筛选维度**：① 设备编码 = SN `ILIKE` 模糊；② 产品 = product 装配件下拉（`products` 表 / ProductRegistry）；③ 产品类型 = productClass 下拉（字典 `product_class`）。三者 AND 组合，服务端过滤。
- **复用**：DeviceTree 的多选/分页/已选标签 + BatchSnModal 的批量粘贴 SN（验证存在性/去重）。
- **回填**：确定 → 关闭弹框 → 顶部条「① 选择目标设备」显示「已选 N 台」，再触发第②步。

#### 3.3.2 命令选择弹框

```
┌── 选择 MML 命令 ───────────────────────────────────────────── [×] ──┐
│ [搜索: 命令码 / 名称 / 路径 / 描述               ]  [产品类型: 全部 ▾]  │
├──────────────────────────────┬───────────────────────────────────────┤
│ 命令树（左 ~40%）             │ 命令预览（右 ~60%，选中后）            │
│ ▸ 设备信息参数管理            │  LST 查询设备基本信息                  │
│   · LST 查询设备基本信息 ◀选中│  操作类型: LST（查询）                 │
│   · MOD 修改设备基本信息      │  说明: …                               │
│ ▸ 软件版本参数管理            │  涉及 PATH: Device.DeviceInfo.*（N 项）│
│ ▸ 告警参数管理                │  ┌ [标准命令] [自定义模板] 两 Tab      │
│ … （两层树 + 搜索高亮）       │                                        │
├──────────────────────────────┴───────────────────────────────────────┤
│                                          [取消]   [确定选择]           │
└──────────────────────────────────────────────────────────────────────┘
```

- **复用**：CommandTree 的两层树 / 搜索（命令码·名称·路径·描述 ILIKE）/ 自定义模板，useGroupTree。
- **回填**：确定 → 关闭 → 顶部条「② 选择 MML 命令」显示命令名 + 操作类型彩 Tag → 进入左操作区配参。

> 据此，§6 决策点①**敲定为「弹框（Modal）」**，设备/命令各一弹框，不再用 Drawer。

### 3.4 执行结果表格设计（核心）

**行 = 设备**，**列 = 操作相关字段**，随操作类型自适应：

- **LST（查询）**：`序号 | 设备SN | 状态 | <参数路径1> | <参数路径2> | … | 单设备下载`
  - 参数列由「本次查询的 path 集合」动态生成（来自后端 schema）。
- **MOD/ADD/RMV（写）**：`序号 | 设备SN | 状态 | 每参数结果(✓/✗) | 故障码 | 单设备下载`
  - 失败行**可展开**显示 fault 详情 + 原始报文。

通用能力：
- 顶部**汇总条**：共 N 台 / ✓成功 / ✗失败 / 用时；状态过滤（全部/仅失败）、设备过滤、列设置。
- **实时填充**：SSE `mml_device_frame` 到达即把对应设备行从「执行中→成功/失败」就地更新（不再只追加文本）。
- **单设备下载**（行尾 ⤓）+ **下载全部**（汇总条，CSV/XLSX/JSON）。
- 保留 `TerminalPanel` 为「原始报文」折叠区（审计/排错）。

### 3.5 批量设备执行（复用为主）

- 后端 `DeviceSNs[]` + Fanouter 已支持，无需改执行链路。
- 前端增强：设备 Drawer 内增「按设备分组批量选」（选 group → 展开 SN）。
- 结果按 device_task 天然 per-device，正好映射表格行。

### 3.6 结果下载（新增）

| 下载 | 入口 | 内容 |
|---|---|---|
| 单设备 | 结果表格行尾 ⤓ | 该设备的参数值/故障（CSV 或 JSON） |
| 全设备 | 汇总条「下载全部 ▾」 | 整张结果表（CSV/XLSX）；可选打包每设备单文件的 zip |

### 3.7 后端改动（复用 + 新增）

**复用（不动）**：execute-statements(-structured)、Fanouter、Sequencer、ResultAggregator、SSE 事件。

**新增/增强**：
1. **结果 schema 端点**　`GET /mml/tasks/:id/results-schema`
   → `{ operationType, columns:[{key,label,path,type}], deviceCount }`，前端据此渲染表头。
2. **结果行端点升级**（沿用 `GET /mml/tasks/:id/results`）
   → 每行 `{ deviceSn, status, cells:{path:value|fault}, raw }`，配合 schema 即可成表。
3. **导出端点**　`POST /mml/tasks/:id/export`
   → body `{ format: csv|xlsx|json, scope: all|device, deviceSn?, columns? }`
   → 小数据集同步返回文件流；大数据集走 worker 异步 + MinIO presigned URL（`GET …/download/:fileId`）。
4. （可选）导出鉴权：download URL 绑 session/API key，防越权。

> 说明：表格化「最小可用」其实只需 ①+②（schema + 行），下载 ③ 可作为第二阶段；①② 基于现有 `GetTaskResults` 的 JSONB 增量即可，不动执行链路。

### 3.8 数据流（改版后）

```
选设备(Drawer) → 选命令(Drawer) → 左操作区配参 → [执行]
  → POST /mml/(console/)execute-statements(-structured) → task.id
  → 前端 GET /results-schema → 渲染表头(设备行占位"执行中")
  → SSE mml_device_frame → 就地更新对应设备行(成功/失败/值/故障)
  → mml_task_completed → 汇总条统计
  → [下载全部/单设备] → POST /export → (同步流 | MinIO URL)
```

### 3.9 「参数路径指定」裸路径专家模式（V2 补齐）

> 2026-06-04 用户决策：V2 左操作区缺了 V1 的「参数路径指定」功能，需补齐。本节定义其在 V2 的形态，**待 UI 布局确认后实施**。

#### 3.9.1 V1 现状（分析结论）

V1 `/mml/console` 右栏是 `Tabs:[控制面板][参数路径指定]` 两个并列页签：

| 页签 | 模式 | 数据来源 | 后端通道 |
|---|---|---|---|
| 控制面板 | **结构化** | 所选命令的 sub_field 字典（勾选/填值/实例号） | `execute-statements-structured`（带 command_id，经字典校验） |
| 参数路径指定 | **裸路径专家** | 用户手输任意 TR-069 path（`command=null`，不绑命令） | `POST /mml/execute`（command_code 空 + param_paths 非空 → 后端 `service.go` 合成 RAW LST/MOD/ADD/RMV，**不经 sub_field 校验**） |

「参数路径指定」(`ParameterPathCommand.tsx`) 的构成：
- **操作类型下拉**：LST / MOD / ADD / RMV（各映射 TR-069 RPC：GetParameterValues / SetParameterValues / AddObject / DeleteObject）。
- **路径行列表**：每行一个 path 输入（AutoComplete，命令绑定时给 paramPaths 建议）；可 `[+]/[-]` 增删行。
- **MOD 追加值列**：MOD 时每行右侧出现「参数值」输入框，path 与 value 下标平行对齐。
- **单行约束**：ADD / RMV 协议单次仅作用一个对象 → 锁单行；LST / MOD 支持多行。
- **用途**：sub_field 未维护 / catalog 外路径的临时探测下发。

V2 当前左操作区 `OperationPanel` 只实现了「控制面板（结构化）」一半，缺「参数路径指定」。

#### 3.9.2 V2 布局（主推方案 A：左操作区顶部模式切换）

在左「操作 & 参数」面板顶部加一个 **Segmented 模式切换**，对标 V1 双页签，把两种模式收进同一栏，主舞台（右结果表格）不变：

```
┌ 操作 & 参数 ─────────────────────────────────┐
│ [ 标准参数 | 参数路径指定 ]   ← Segmented 切换  │
│ ─────────────────────────────────────────────│
│ 〈参数路径指定〉选中时：                        │
│  操作类型: [ LST - GetParameterValues  ▾ ]     │
│  参数路径:                                      │
│   1. [Device.DeviceInfo.* ............] [＋][－]│
│   2. [Device.Services.FAPService.{i}..] [＋][－]│
│      (MOD 时每行右侧追加 [参数值] 输入框)       │
│      (ADD/RMV 锁单行，[＋] 置灰)               │
│  ⓘ 裸路径直发，不经 sub_field 字典校验          │
│  执行模式: ● 整体下发  ○ 逐 PATH               │
│  [▶ 执行（N 台）]   [清空结果]                  │
└───────────────────────────────────────────────┘
```

行为约定：
- **标准参数**模式：维持现状，需先在顶部选「② MML 命令」；无命令 → 空态提示。
- **参数路径指定**模式：**不需要选命令**，只需「① 目标设备」+ ≥1 条 path 即可执行（与 V1 一致）；若已选命令，则把该命令的 paramPaths 作为 path 输入的 AutoComplete 建议（V1 未做的小增强，可选）。
- 切换模式不互相清空对方已填内容（各自独立 state）。
- **执行结果表格复用 §3.4**：列由本次 path 集合动态生成（LST=参数值矩阵；MOD/ADD/RMV=状态+故障矩阵），原始报文折叠区保留。
- 顶部选择条「② 选择 MML 命令」在「参数路径指定」模式下文案弱化为「(可选)」，不再是执行前置条件。

#### 3.9.3 备选方案（供对比）

- **方案 B「折叠高级区」**：标准参数列表下方挂一个可折叠「参数路径指定（高级）」区，执行时把结构化路径 + 裸路径合并下发。优点是一屏全见；缺点是两种通道语义混在一次执行里，与 V1「独立两通道」契约不一致，故障归因更难。
- **方案 C「独立第三入口」**：顶部选择条加「③ 参数路径指定」开关，整页切到裸路径模式。占用顶部空间，且与「② 命令」语义重叠，体感割裂。

> 推荐 **方案 A**：最贴合 V1 心智（双模式并列）、改动内聚（仅左面板加 Segmented + 一个 RawPathPanel 子组件）、不污染右结果表格与执行链路。

### 3.10 V2 二轮完善（2026-06-04，待实现）

> 本节是在 §3.2~§3.9 已落地（mock）基础上的 6 项增强。**会调整主版面**：操作配参从「左侧常驻面板」改为「弹框」（§3.10.3），主舞台让位给「命令记录 ｜ 执行结果」左右分区（§3.10.4~3.10.6）。下文 §3.2 主推布局中"左操作区"部分以本节为准。

调整后的顶部流程与主版面：

```
┌────────────────────────────────────────────────────────────────────────┐
│ 顶部条: [① 选择目标设备 ▸已选N台] [② 选择 MML 命令 ▸LST…] [③ 配置参数 ▸…] [▶ 执行] │
├──────────────┬───────────────────────────────────────────────────────────┤
│ 命令记录(可收缩)│ 执行结果(主舞台)                                          │
│ 展开=24%       │  ┌ 汇总条: 共N台 ✓ ✗ 用时   [下载全部▾]                   │
│ 默认收缩=窄条   │  ├ [状态过滤][SN过滤]                                      │
│ ┌───────────┐ │  ├─结果表格(行=设备, 列=参数/状态, 行尾[查看][下载])─────┐ │
│ │14:02 LST设备│ │  │ …                                                    │ │
│ │  基本信息 12台│ │  └──────────────────────────────────────────────────┘ │
│ │13:55 MOD小区 │ │                                                          │
│ │  功率 1台    │ │  (默认显示最近一次执行结果；点击左侧某条记录→切到该次结果)│
│ └───────────┘ │                                                          │
└──────────────┴───────────────────────────────────────────────────────────┘
```

#### 3.10.1 设备弹框「表头全选 = 筛选命中的全部数据」（需求①，2026-06-04 修订）

> **修订（2026-06-04）**：去掉原先在「已选 N 台」旁单独加的「全选（满足筛选条件的全部设备）」复选框；改为复用 **表格表头的全选复选框**，但语义扩展为「选中筛选命中的**全部数据**（不限于当前页）」。

- **语义**：点击表头全选复选框 → 跨分页选中**当前筛选条件（SN/产品/产品类型）命中的全部设备**（不只当前页），上限见下；再次点击取消 → 清空。
- **不默认全选**：弹框打开恢复上次选择（无既有选择则为空），不再自动全选。
- **表头勾选态**：全部命中已选 = 勾选；部分已选 = 半选（indeterminate）。
- **实现要点（前端）**：AntD `Table.rowSelection` 默认表头全选只覆盖当前页；用 `columnTitle` 自定义表头复选框接管，`onChange` 选中 = 筛选命中前 N 台 SN（跨页），配合 `preserveSelectedRowKeys` 让各页行勾选态一致。
- **实现要点（接后端）**：复用 `GET /api/v1/devices`（服务端 SN/产品/productClass 过滤）取回命中 SN（`page_size=<上限>` 或先取 `total`）。

**最大设备数量限制（接口分析结论）**：

| 环节 | 现状 | 约束 |
|---|---|---|
| 设备列表接口 `GET /devices` | `page_size` 默认 1000，**硬上限 10000** | 一次可取回 ≤10000 SN |
| MML 执行 `POST /mml/console/execute-statements(-structured)` | 仅 `device_sns: min=1`，**无上限校验** | 无显式限制 |
| 扇出 `Fanouter.Fanout` | 单次 `BatchCreateTasks` 批量插入 (设备×命令) 条 device_task | PostgreSQL 单语句 65535 bind 参数；device_task ~12 列/行 → 批量插入硬顶 ~5000 行 |
| 下游 | 每设备一个 Connection Request，受 ACS 准入/限流（容量基线 ~5000 在线 CPE） | 一次性洪泛大量 CR 压 ACS |

→ **结论（已定 2026-06-04）**：表头全选选中**上限 200 台**（常量 / 可配置 `MAX_CONSOLE_SELECT_ALL=200`）。理由：
- 远低于批量插入 ~5000 行的硬顶，JSON body（200 个 SN）无压力；
- ACS 一次承接 200 个 CR 在容量基线内，留足并发余量；
- MML 控制台是交互式即时下发，>200 台的规模化下发应走「脚本任务」批量入口（§非目标）。

**超限处理**：当筛选命中 `total > 200` 时——表头全选仅选中前 200 台，并在「已选 N 台」旁以橙字提示「筛选命中 {total} 台，已超单次上限，仅选中前 200 台」。

#### 3.10.2 命令弹框「参数 PATH」列表改造（需求②）

§3.3.2 命令预览区「涉及 PATH」一节改造：

- 标题 **「涉及参数路径」→「参数 PATH」**。
- 列表**不再显示「只读/可写」标签**（去掉 §3.3.2 / 现 mock 里的 `只读/可写` Tag）。
- 每条显示两段：**PATH 名称**（短标签，如「厂商」）+ **PATH**（完整 TR-069 路径）。
- **PATH 过长截断**（单行 ellipsis），**鼠标移上 Tooltip 显示完整 PATH**。

```
参数 PATH（4 项）
  厂商        Device.DeviceInfo.Manufacturer
  软件版本    Device.Services.FAPService.1.…(截断) ⟵hover 显示完整
```

#### 3.10.3 配置参数改为弹框「③ 配置参数」（需求③）

把 §3.9 的「左操作区 命令参数/指定参数 Tabs」**整体搬进一个 Modal**，参考「选择 MML 命令」弹框的实现：

- 顶部条新增第三个按钮 **「③ 配置参数」**（在「② 选择 MML 命令」之后；未选命令时禁用）。
- 点击弹出 **配置参数弹框**：内含「命令参数 ｜ 指定参数」两 Tab（即现 `OperationPanel` 的主体：操作类型、参数全选/勾选/填值/实例号；指定参数裸路径行）。
- 弹框底部「确定」回填顶部条摘要（如「③ 配置参数 ▸ 已选 3 路径 / LST」）；「执行」按钮移到顶部条（或弹框内「确定并执行」）。
- 收益：左侧版面腾出给「命令记录 ｜ 执行结果」（§3.10.4~6）。

#### 3.10.4 执行命令记录列表（需求④）

主舞台左侧新增 **「执行命令记录」列表**：

- 每条记录含：**时间**（HH:mm:ss）、**执行的命令名称**（含 op 彩 Tag）、**设备数**（N 台）。
- 数据来源：每次「执行」成功后在前端追加一条（mock 阶段为本地 state；后续可对接 `GET /mml/tasks` 任务列表，按 console 来源过滤）。
- **可靠左收缩**：标题栏带折叠按钮；**默认收缩**为左侧窄条（仅图标/竖排标题），点击展开。

#### 3.10.5 结果跟随记录联动（需求⑤）

- 执行结果区**默认显示最近一次执行**的结果（最新记录）。
- **点击命令记录某条** → 右侧执行结果切换为该记录对应的结果内容（列/行/汇总/「查看」详情均随之）。
- 当前选中的记录在列表中高亮。
- mock：每条记录保存其 `{execMeta, columns, rows}` 快照，点击即切换；真实接入时按 `task.id` 拉 `/results-schema` + `/results`。

#### 3.10.6 命令记录 ｜ 执行结果 左右分区（需求⑥）

- 二者**左右排列**：左=命令记录，右=执行结果。
- 命令记录**可靠左收缩**；**展开时占比 24%**（执行结果 76%；2026-06-04 由 30% 再缩小 20% 至 24%），收缩时仅留窄条（执行结果近 100%）。
- 用 AntD `Splitter` 或固定 30/70 `Row`+折叠态切换实现；收缩态用图标条 + 展开按钮。

### 3.11 V2 三轮完善（2026-06-05，结果表格强化 + 任务层级 + 记录持久化）

> 本节 6 项需求聚焦「执行结果」表格的列定稿、**写类命令读后核实**、**任务 ID 层级体系**、**命令记录跨刷新持久化**。其中 §3.11.2（读后核实）与 §3.11.3（逐 PATH 父任务）涉及后端模型，是本轮**重点风险区**，已在 §3.11.6 / §7 单列提醒。需求编号：①列固定 ②动态 path 列 ③尾部时间列 ④读后核实 ⑤任务层级 ID ⑥记录持久化+清空。

#### 3.11.1 结果表格列结构定稿（需求①②③）

最终列序（左→右）：

```
[固定左] 序号 | 设备SN | 状态 ‖ <path1> | <path2> | … | <pathN> ‖ [固定右] 下发时间 | 响应时间 | 操作
         └──── 需求① ────┘   └──────── 需求②(每 path 一列) ───────┘   └──── 需求③ ────┘  └查看/下载┘
```

- **需求①**：`序号 / 设备SN / 状态` 三列 `fixed:'left'`，恒显最前（现状已实现，保持）。
- **需求②**：中间动态列 = 本次命令所选 path 集合，**每 path 一列**，单元格显示该 path 的执行结果值（现状已实现）。写类命令此前显示 `✓`，本轮按 §3.11.2 改为显示**读回的实际值**。
- **需求③**：尾部新增两列并 `fixed:'right'`：
  - **下发时间**（RPC 任务下发时间）← `device_tasks.sent_at`
  - **响应时间**（RPC 任务执行响应时间）← `device_tasks.completed_at`
  - 未下发/未响应（pending/sent）时对应单元格留空（显示 `-`）。
- **「操作」列**（查看/下载）保持 `fixed:'right'` 最右；故已固定右侧 = `下发时间 | 响应时间 | 操作`（约 150+150+110 ≈ 410px）。
- **「故障码」独立列取消**：原 §3.4 的「故障码」列移除，故障信息收进「状态」列 Tooltip + 「查看」详情，给尾部时间列腾宽。失败行的故障码仍在 `ResultDetailModal` 完整呈现。

> 列宽权衡：左 3 固定 + 右 3 固定 + N 动态列，横向较挤；命令记录展开（24%）时结果区更窄。建议动态列 `width:140` + 横向滚动；path 数 > 8 时提示用户收敛勾选。

**汇总统计与标题同行（2026-06-05 修订）**：原「执行结果」标题与「设备总数 / 成功 / 未核实 / 未生效 / 失败 / 最长用时」汇总条分两行显示，改为**并入 Card 标题行**——`title` 内左侧「执行结果」+ 右随紧凑文本统计（彩色：成功绿、未核实黄、未生效/失败红、用时灰），写类才显示「未核实/未生效」；下载全部保持在 `extra`（最右）。腾出一行垂直空间给结果表格。空态（未执行）只显示「执行结果」标题。

#### 3.11.2 写类命令「读后核实」read-after-write（需求④，**重点**）

**目标**：MOD/ADD/RMV 等写类命令，基站可能「响应 success 但实际未生效」。本轮要求**写完自动追加一次对应 path 的查询（LST/GetParameterValues），以查询结果为准**回填结果列，并据此判定真实成败。

**核实语义（按 op 分治，差异大）**：

| 写 op | 写 RPC | 核实查询 | 判定「真正成功」 | 单元格显示 |
|---|---|---|---|---|
| MOD | SetParameterValues | GetParameterValues(同 path) | 读回值 == 预期下发值 | 读回的实际值（不匹配标红+「未生效」） |
| ADD | AddObject(返回 InstanceNumber) | GetParameterValues(新实例 path) | 新实例存在/可读 | 实例号 + 关键字段读回值 |
| RMV | DeleteObject | GetParameterValues(原实例 path) | 实例不存在（查询应 9005/空） | 「已删除」（仍存在→标红「未删除」） |

**状态机扩展**：单设备状态从 `success/failed` 细化为：
- `success`（写 OK 且核实通过：读回值 == 预期值）→ 绿色「成功·已核实」
- `unverified`（写 OK 但**无法核实**，**不判失败**，但列表须明确提示原因）→ 黄色，含两种子类（详情区分）：
  - **只写不可读**：path write-only / 查询返回不支持 → 提示「只写参数·无法核实」
  - **需重启生效**：参数标记为「重启后生效」→ 提示「需重启生效·暂不核实」（避免写后立即读读到旧值的误判）
  - 兜底（查询超时等）→ 提示「核实查询失败」
- `mismatch`（写 OK 但核实**不一致**：响应成功实际未更新）→ 红色「未生效」，正是本需求要暴露的场景
- `failed`（写 RPC 本身失败）→ 红色「下发失败」

> **判失败边界（2026-06-05 决策）**：**只写参数、重启生效参数一律不计入失败**（落 `unverified`），但**必须在结果列与详情中明确给出原因提示**，让运维知道"未核实 ≠ 失败、也 ≠ 已确认生效"。仅 `mismatch`（可读且读回不符）与 `failed`（RPC 失败）计为问题行。

**后端实现路径 ——（2026-06-05 决策）方案 A·命令链**：
- 控制台在写命令后**自动追加一条 LST 命令**（同 path 集合，且过滤掉「只写/重启生效」path），组成 2-command MML 任务，走现有 `Sequencer` 顺序执行（写 → 读）。
- `ResultAggregator` 合并：以**写 device_task** 定 `下发/响应时间` + RPC 成败；以**读 device_task** 的值回填单元格并与预期比对 → 派生 `success/mismatch`；被过滤掉读取的 path（只写/重启生效）直接落 `unverified` + 原因码。
- **零执行链路改动**，仅控制台编排（追加 LST）+ 聚合器加比对/原因码。
- 「只写 / 重启生效」标记来源：参数字典（sub_field / param_model 的可读性 + 生效方式属性）；字典缺失时默认尝试读取，读不到→`unverified`。

> **本轮先做前端 mock**：`buildResultRows` 对写类命令生成「读回值 + 个别 mismatch + 个别 unverified(只写/重启)」样例，跑通四态展示与详情；后端方案 A 待 §6 决策落地后单列 backlog 任务实施。

#### 3.11.3 任务层级与 ID 体系（需求⑤，**重点**）

要求显式呈现三级 ID 层级，并落到任务记录：

```
命令ID (commandId)                       ← 一次「执行」= 一个命令对 N 设备的下发
 ├─ 设备1 设备任务ID (deviceTaskId)
 │    └─[逐PATH] path1 子任务 / path2 子任务 … 共享「设备1 父任务ID」
 ├─ 设备2 设备任务ID
 └─ …
```

**与现有后端模型的映射（关键结论）**：

| 概念 | 现有承载 | 状态 |
|---|---|---|
| **命令 ID** | `mml_tasks.id`（一个 MML 任务 = **一次批量执行**，聚合 N 设备） | ✅ **已存在**，直接复用；UUID `gen_random_uuid()` **全局唯一**（满足「每次批量执行一个全局唯一命令 ID」要求）；UI 展示并可深链 `/mml/tasks/:id` |
| **设备任务 ID** | `device_tasks.id`（per 设备×命令，`source_id = mml_tasks.id`） | ✅ **已存在**（整体下发模式天然 1 设备 = 1 device_task） |
| **逐 PATH 父任务 ID** | —— `device_tasks` **无 `parent_id` 列**；现扇出粒度是 per-(设备,命令)，**非** per-(设备,path) | ⚠️ **缺口**，见下 |

**逐 PATH 模式的缺口与方案（需决策）**：
- 现状「逐 PATH」只是前端把多 path 展开为多次 RPC 的设想，后端 `Fanouter` 仍是「一命令一 device_task」。要实现「一设备多 path 子任务 + 共享父任务 ID」，需：
  - **方案 A·加列**：`device_tasks` 增 `parent_id uuid`（nullable）；逐 PATH 时每 path 一条子 device_task 指向「该设备父任务」（父可为一条虚拟聚合行或写命令行）。需 1 个迁移。
  - **方案 B·复用 source_id + 索引**：父级用 `source_id(=命令ID)` + `device_index` 标识，path 子任务用新增 `path_index` 区分；不加 `parent_id`，靠 (source_id, device_index) 聚合。改动小但「父任务」无独立实体。
  - **方案 C·先不落库**：逐 PATH 父子层级**仅前端建模**（ExecRecord/ResultRow 内存结构），后端整体下发不变，待真有逐 PATH 下发需求再补迁移。
- **（2026-06-05 决策）采方案 C**：前端先把层级建模 + mock，整体下发对接 §3.11.2 方案 A；把 `device_tasks.parent_id` 迁移列为「逐 PATH 真实落地」的后置任务，不阻塞本轮交付。
- **命令 ID 全局唯一性（决策）**：命令 ID = `mml_tasks.id`（UUID `gen_random_uuid()`），**每次批量执行生成一个、全局唯一**；mock 阶段前端用全局唯一生成器（如 `crypto.randomUUID()`）模拟，对接后端后由后端返回真实 `mml_tasks.id` 覆盖，前端不自造命令 ID 入库。

**结果表格落点**：表格行仍 = 设备（= 设备任务）。逐 PATH 下，`下发/响应时间`列取该设备所有 path 子任务的**聚合**（min(sent_at) / max(completed_at)），逐 path 明细在「查看」详情里按子任务列出（含各自子任务 ID 与时间）。汇总条/记录面板展示**命令 ID**。

#### 3.11.4 命令记录前端持久化 + 清空（需求⑥）

落定 §6.8 遗留项：命令记录**跨刷新持久化（前端缓存）**，并提供**清空**操作。

> **（2026-06-05 决策）localStorage 只持久化「命令 ID 列表」，凭命令 ID 回拉全部数据**——不缓存命令名/设备数等摘要，更不缓存结果行快照。

- **持久化内容**：`useConsoleHistory` 仅向 `localStorage`（key `mml-console-v2-history`）写入**命令 ID 数组**（有序，最近在前），例如 `["<uuid1>","<uuid2>", …]`。
- **挂载恢复**：进入页面时读取该命令 ID 列表 → 对每个命令 ID（接后端阶段）`GET /mml/tasks/:id` 拉记录摘要（时间/命令名/op/设备数）、`select` 时再惰性 `GET …/results-schema` + `…/results` 拉结果。**所有展示数据由命令 ID 现拉，localStorage 不存任何派生内容**。
- **关联任务记录**：命令 ID 即 `mml_tasks.id`，记录项天然可深链 `/mml/tasks/:id`，与「任务记录」页贯通。
- **清空操作**：命令记录面板标题栏加「清空」按钮（二次确认），清空内存列表 + localStorage 的命令 ID 数组。
- **约束**：
  - 命令 ID 列表设上限（如最近 50 个），超出 FIFO 淘汰（仅丢 ID，不丢服务端任务）。
  - 某命令 ID 对应任务已被服务端清理 → 回拉 404 时该记录标「已过期」或自动剔除。
- **mock 阶段适配**：当前无后端，按"凭命令 ID 取数"的同一契约，在内存 `Map<commandId, ExecRecord>` 里存执行产物，localStorage 仍只存命令 ID 列表；刷新后 mock 内存 Map 丢失属预期（仅演示"列表恢复+按 ID 取数"的链路），真实数据以后端为准。
- **方案一致性**：以上仍在 `useConsoleHistory` 内部完成，组件层 `records/activeId/select/append/clear` 契约不变。

#### 3.11.5 类型/数据结构变更汇总

前端（mock 即可落，后端字段一一对应）：
- `ExecRecord` 增 `commandId: string`（= mml_task.id，全局唯一）。
- `ResultRow` 增 `deviceTaskId: string`、`dispatchedAt?: string`、`respondedAt?: string`；`status` 扩为 `'success'|'unverified'|'mismatch'|'failed'|'pending'|'running'`；增 `unverifiedReason?: 'write-only'|'reboot-required'|'query-failed'`（unverified 时的提示原因）；写类可读 `cells` 改存读回值；增 `verify?: { path; expected; actual; matched }[]` 供详情比对展示。
- 逐 PATH：`ResultRow` 增 `pathTasks?: { pathIndex; path; subTaskId; status; dispatchedAt; respondedAt; value }[]`（详情页用，父任务 ID = deviceTaskId）。
- `useConsoleHistory`：增 `clear()`；**localStorage 只持久化命令 ID 数组**（不存摘要/快照）；mock 阶段内存 `Map<commandId, ExecRecord>` 承载执行产物。
- `STATUS_META` 增 `unverified`（黄，按 reason 显示「只写参数·无法核实 / 需重启生效 / 核实查询失败」）、`mismatch`（红，「未生效」）。
- `ResultTable` 尾部加「下发时间/响应时间」两固定列、移除故障码独立列；`ResultDetailModal` 增「核实对比（预期 vs 读回）」「逐 PATH 子任务」分区，并展示命令 ID / 设备任务 ID。

#### 3.11.6 风险与权衡（重点提醒）

1. **读后核实（§3.11.2，已定方案 A）= 最高风险**：
   - **不可读 / 重启生效参数**：**不判失败**（落 `unverified`），但**列表与详情必须明确提示原因**（只写参数 / 需重启生效 / 核实查询失败），让运维分清「未核实 ≠ 失败 ≠ 已生效」。
   - **最终一致/时序**：写后立即读可能读到旧值；「重启生效」类参数直接跳过读取标 `unverified`，不参与比对，避免误报 `mismatch`。
   - **RPC 翻倍**：每个写 → 追加一次读，200 台 MOD = 200 Set + 200 Get = 400 次 CR，压 ACS；需纳入单次上限（§3.10.1 的 200 台）评估。
   - **比对基线**：判定「成功但未更新」需拿**预期下发值**与读回值比对；ADD（实例号）、RMV（不存在性）语义各异，逐 PATH 更复杂——一期先做 MOD 值比对，ADD/RMV 存在性核实二期。
   - **依赖**：「只写 / 重启生效」标记需参数字典（param_model / sub_field）维护可读性与生效方式属性；字典缺失时默认尝试读取，读不到→`unverified`。
2. **逐 PATH 父任务（§3.11.3，已定方案 C）**：`device_tasks` 缺 `parent_id`，现扇出非 per-path → 真实落地需迁移 + Fanouter 改造。本轮**仅前端建模**，迁移列为后置任务，不阻塞本轮。
3. **命令 ID（§3.11.3，已定）低风险**：`mml_tasks.id`（UUID）即命令 ID，每次批量执行全局唯一；前端不自造入库，mock 用 `crypto.randomUUID()` 临时模拟。
4. **localStorage 持久化（§3.11.4，已定只存命令 ID）**：仅存命令 ID 数组、凭 ID 现拉，天然规避容量问题；ID 上限 FIFO；任务被服务端清理→回拉 404 标「已过期」；共享浏览器多用户可按用户隔离 key。
5. **列宽/版面**：固定列增多 + 记录面板占 24%，结果区横向更挤；建议移除故障码独立列、限制动态列数。
6. **范围控制**：本轮**前端 mock 全量可交付**（列、三态、层级、持久化、清空）；后端「读后核实」「逐 PATH 落库」拆为独立 backlog 任务，按 §3.11.2/3.11.3 决策推进。

#### 3.11.7 收尾微调（2026-06-05，4 项展示优化）

四项纯展示层调整，不动数据契约：

1. **「执行结果」标题带命令名**（§3.11.1 标题行补充）：`ResultTable` 的 Card `title` 在「执行结果」之后追加**当前回看记录的命令名称**（标准模式取 `execMeta.commandName`，裸路径模式回退 `execMeta.label`），形如「执行结果 · 查询设备基本信息」。空态（未执行）仍只显示「执行结果」。
2. **汇总统计改为「设备总数 / 执行中 / 成功 /（未核实 /未生效）/ 失败」**（§3.11.1 汇总条修订）：**删除「最长用时」**统计项；在「设备总数」之后**新增「执行中」**计数（= `status ∈ {pending, running}` 的设备行数，蓝色 `processing`）。写类仍保留「未核实/未生效」。
3. **命令记录去掉数量提醒**（§3.10.4 / §3.11.4 面板修订）：移除收缩态历史图标上的**红色数量 Badge**，并移除展开态标题里的 `(N)` 计数文本——命令记录面板不再显示条数。
4. **配置参数弹框：隐藏「操作类型」文字标签、显示命令名**（§3.10.3 弹框修订）：标准模式头部原「操作类型」标签文字去除，改为顶部显示**命令名称**（`command.commandName`），其下保留操作类型彩 Tag（`LST · 查询`）+ 命令码，信息更聚焦。

#### 3.11.8 命令选择弹框：搜索框缩短 + 「指定参数」快捷入口（2026-06-05）

面向「已知裸路径、无需挑命令」的专家用户，在第二步「选择命令」弹框（§3.3.2）顶部增加一个直达裸路径配置的捷径，**跳过命令选择**：

1. **搜索框缩短**：原整行宽 `Input.Search` 改为 `flex:1, maxWidth:420`，与右侧入口排在同一 flex 行。
2. **「指定参数」链接入口**：搜索框右侧加 `Button type="link"`（`EditOutlined` + `指定参数` + `RightOutlined`，带 Tooltip「跳过命令选择，直接用「指定参数」（裸路径）方式配置并执行」）。
3. **跳转语义**：点击 → 关闭「选择命令」弹框 → 打开第三步「配置参数」弹框（§3.10.3）并**自动停在「指定参数」标签**（裸路径专家模式）；正常选命令进入配置弹框时仍默认停在「命令参数」标签。
   - 实现：`CommandSelectModal` 新增 `onGotoRawParams` 回调；`ConfigParamsModal` 新增 `initialMode?: OperationMode`（每次打开按它切换激活 Tab）；`index.tsx` 加 `configMode` 状态——点「指定参数」置 `'raw'`、选命令置 `'standard'`，作为 `initialMode` 透传。
   - 前置：裸路径执行仍需先在第一步选好设备；下发走 legacy `/mml/execute`（§3.12.2 决策 3）。

#### 3.11.9 设备选择：恢复「产品」筛选（product_id）+ 「产品类型」后置（2026-06-05）

§3.3.1 要求设备弹框按 **SN / 产品 / 产品类型** 三维筛选，但 P1 落地时因「后端无 product 维度过滤」临时**只留了 SN + 产品类型两维**。本次补齐第三维「产品」，恢复设计：

- **「产品」筛选（新增）**：下拉选项来自 `useProductList()`（`GET /products` 全量，返回 `{id,name}[]`）；过滤值 = `devices.product_id`（T-0098 ProductRegistry 路由 productClass 后写入的产品 UUID 软引用）。
- **「产品类型」筛选（沿用）**：字典 `product_class` 下拉 → `devices.product_class`，**位置后置**到「产品」之后。
- **顺序**：SN 搜索 → 产品（product_id） → 产品类型（product_class） → 批量输入。三者 AND 组合，全部服务端过滤。
- **后端改动（device 模块）**：`DeviceFilter` 加 `ProductID *uuid.UUID`；`device_handler.go` List 读 `c.Query("product_id")` 解析 UUID；`device_repository.go` List() 对 `d.product_id` 加 WHERE（builder + countBuilder 同步）。
- **前端业务层**：`DeviceFilter` 加 `productId?: string`；`deviceApi.getList` 映射 `productId → query.product_id`。
- **前端 UI（V2 DeviceSelectModal）**：新增「产品」`Select`（选项 = 产品列表），`productFilter` 状态并入 `filterParams`，打开时重置。

### 3.12 后端 API 对接方案（2026-06-05，落地真实数据）

> 在 §3.2~§3.11 前端 mock 全量落地基础上，把 `ConsoleV2/` 的 mock 层替换为真实后端端点。**复用老 console（`mml/Console/`）已验证的结构化执行通道**，不新增后端端点。

#### 3.12.1 端点映射（全部已存在，base `/api/v1`）

| 环节 | 复用 hook / api | 端点 |
|------|----------------|------|
| 设备列表 | `deviceApi.getList`（服务端分页/筛选） | `GET /devices?page&page_size&search&product_id&product_class` |
| 产品下拉 | `useProductList()`（全量，无分页） | `GET /products` → `{id,name}[]`（产品筛选下拉选项） |
| 命令树 | `useGroupTreeFlat(lang)` | `GET /mml/group-tree?format=flat&lang` |
| 命令参数 | `useCommandSubFields(commandId, lang, deviceKey)` | `GET /mml/commands/:id/sub-fields` → `SubFieldDef[]` |
| 命令搜索 | `useSearchCommands(q)` | `GET /mml/commands/search` |
| 标准执行 | `useExecuteStatementsStructured` | `POST /mml/console/execute-statements-structured` → `MMLTask` |
| 裸路径执行 | `useExecuteMMLCommand`（legacy） | `POST /mml/execute`（`command_code` + `param_paths`） |
| 结果（实时） | `useMmlTaskStream(taskIds)`（SSE） | `EventSource /api/v1/events/stream`，事件 `mml_device_frame`/`mml_task_status`/`mml_task_completed` |
| 结果（落库） | `getTaskById` + `getTaskResults` | `GET /mml/tasks/:id` + `GET /mml/tasks/:id/results` |

**无 results-schema 端点**：结果表格的**列**由前端从「用户所选 path 集合」派生（沿用 `buildColumns`），**单元格值**从 `DeviceTaskResultItem.result.parsedData[path]`（或 SSE `mml_device_frame`）取——与老 console 客户端解析 GPV 一致。

#### 3.12.2 关键决策（2026-06-05 用户确认）

1. **读后核实四态先降级两态**：后端当前不做 read-after-write，真实结果只有 `success`/`failed`。P1–P3 只用真实两态回填，`ExecStatus` 的 `unverified`/`mismatch` 结构**保留**但暂不产生；`§3.11.2` 读后核实拆为后端 backlog（P4）。
2. **实时结果走 SSE**：复用 `useMmlTaskStream`，`mml_device_frame` 逐设备帧**就地更新** active record 的 `ResultRow`；最终态用 `getTaskResults` 兜底校正。
3. **裸路径模式接 legacy `/mml/execute`**：`指定参数` Tab 无 `command_id`，走老的 `command_code` + `param_paths` 通道（`useExecuteMMLCommand`），与结构化通道并存。
4. **分阶段交付**：P1 只读对接 → P2 执行+结果 → P3 命令记录 → P4（后端）读后核实。

#### 3.12.3 类型 / 适配

- `CommandItem.id` 改为**真实 `mml_commands.id`（UUID）**，即结构化执行所需 `commandId`；`paramPaths` 由 `SubFieldDef` 适配（`path=tr069Path`、`label`、`writable=accessType==='READ_WRITE'`）。
- 新增 `ConsoleV2/adapters.ts`：`mapDeviceToItem`、`mapFlatCommand`、`subFieldsToParamPaths`、`mapResultItemToRow`（真实类型 → v2 view 类型，隔离对接面）。
- `useConsoleHistory`：`recordStore` 由「内存 Map」改为凭 localStorage 命令 ID 数组拉 `getTaskById` + `getTaskResults`（§3.11.4 契约不变）。
- `mock.ts` 保留纯函数（`buildColumns`/`buildColumnsFromRawPaths`），删除 `MOCK_DEVICES`/`MOCK_COMMANDS`/`MOCK_HISTORY`/`buildResultRows`。

#### 3.12.4 分阶段文件改动与落地状态

- **P1 只读 ✅ 已落地**：`DeviceSelectModal`（→`useDeviceList` 服务端分页 + `useDictionaryBatch('product_class')`）、`CommandSelectModal`（→`useGroupTree` 拍平二级 + 选中拉 `useCommandSubFields`）、新增 `adapters.ts`（`mapDeviceToItem`/`subFieldsToParamPaths`/`flattenGroupTree`/`mapCommandItem`）；`CommandItem.id` = 真实 `mml_commands.id`。
- **P2 执行+结果 ✅ 已落地**：`ExecRequest` 扩 `values`/`instance`；`ConfigParamsModal.buildRequest` 带写入值/实例号；`index.tsx.runExecute` → 标准 `executeStatementsStructured`（`buildStructuredStatement`）/ 裸路径 `executeMMLCommand`（`buildRawExecutePayload`）；新增 `useExecStream`（SSE 订阅 `mml_device_frame`/`mml_task_completed`）+ `applyFrameToRow`（先降级 success/failed 两态，GPV 按列回填 cells，exact→leaf 匹配）；完成后落入命令记录（commandId = 真实 task id）。列派生 `buildColumns`/`buildColumnsFromRawPaths` 移入 `adapters.ts`，删除 `mock.ts`。
- **P3 命令记录跨刷新恢复 ✅ 已落地**：`useConsoleHistory` 去 mock 种子；`recordStore` 承载本会话完整记录；挂载时对 localStorage 中、不在 `recordStore` 的命令 ID 用 `useQueries(getTaskById)` 重建（`mapTaskToRecord`：命令元信息取首条 `commandsDetail` 的 op_type/param_paths，结果行取任务内嵌 `results` 的 success/rawOutput/parsedData，`retry:false`，404 自动剔除）。`getTaskById` 已内嵌 `results`，无需 select 时再拉 `getTaskResults`。
- **P4 读后核实四态 ⏳ 待后端**：`unverified`/`mismatch` 待后端 read-after-write 支持（§3.11.6 最高风险项）。

> **P2 待运行时核对项**：① GPV 回值键为 privatePath，结果列键为 standardPath，`applyFrameToRow` 用 exact→leaf 两级匹配兜底，translation 较深的厂商私有路径需实测对齐度；② SSE `/events/stream` 依赖 JWT，断线重连沿用 EventSource 默认；③ 裸路径 `/mml/execute` 的 `param_values` 下标须与 `param_paths` 对齐（已保证）。

### 3.13 结果 CSV 导出 → MinIO（2026-06-05，落地需求③下载，✅ 已实现）

补齐 §3.6/§3.7 的「结果下载」缺口。用户决策（2026-06-05）：**只做 CSV；完整参数矩阵；两种文件（单设备 + 全设备汇总）；存 MinIO 独立目录；地址记入 mml_tasks**。

**后端**（`internal/mml/export.go` + 迁移 000026）：
- **迁移 000026**：`mml_tasks` 加 `export_object TEXT`（汇总 CSV object key）、`device_export_objects JSONB`（{device_sn: object_key} 映射）、`export_generated_at TIMESTAMPTZ`。
- **存储**：复用 `reports` bucket + **独立目录** `mml-results/{YYYY}/{MM}/{DD}/{taskID}/`；汇总 `aggregate-{id8}.csv`、单设备 `device-{sn}-{id8}.csv`。DB 存 object key，下载时 `PresignedGetObject` 现签 1h URL（项目惯例）。
- **完整参数矩阵**：列 = 命令查询的 standardPath（取自 `extractPathTranslations`，同时拿到 privatePath）；值 = 解析每台设备 `device_tasks.result.raw_response` 的 GPV（`pkg/soap.DecodeGetParameterValuesResponse`）→ 按 **privatePath → standardPath → 叶子名** 三级对齐回填（比前端更精确，因后端有 privatePath 映射）。汇总=横向矩阵（设备行×参数列）；单设备=纵向（参数路径,读回值）+ 顶部摘要。UTF-8 BOM 让 Excel 正确识别中文。
- **端点**：`POST /mml/tasks/:id/export`（汇总）、`POST /mml/tasks/:id/devices/:sn/export`（单设备）→ `{object, download_url}`；MinIO 未配置→503。
- **DI**：`provider/modules.go` 注入内部 client（PutObject）+ PresignClient（签 URL）+ reports bucket。

**前端**：`mmlApi.exportTaskCSV`/`exportTaskDeviceCSV` + `useExportTaskCSV`/`useExportTaskDeviceCSV` hook；`ResultTable`「下载全部 ▸ CSV」与行尾下载改调后端（生成→拿 presigned URL→`window.open` 下载），XLSX/JSON 保留客户端导出。

> **仍未做**（不在本次范围）：真实 XLSX 流、每设备单文件打包 zip、大批量异步走 worker（当前同步生成，单批 ≤200 台 CSV 体量小，足够）。

---

### 3.14 实例 {i} 与 ADD/RMV 对象增删（2026-06-06，TR-069 语义对齐）

补齐「配置参数」两个标签页对 TR-069 多实例对象的处理，依据 AddObject / DeleteObject 语义：

| 操作 | 末级对象自身 `{i}` | 父级 `{i}` | 说明 |
|---|---|---|---|
| ADD = AddObject | **不填**（CPE 分配，回 InstanceNumber） | 必须具体数字 | ObjectName 是对象表路径，`.` 结尾、末级不带实例号 |
| RMV = DeleteObject | **必须填具体实例号** | 必须具体数字 | ObjectName 以 `.<实例号>.` 结尾 |
| LST/MOD | 操作已存在实例，须具体 | 必须具体数字 | path 中所有 `.{i}.` 都须替换 |

**plan A — 指定参数（裸路径）标签**（`rawPathValidate.ts` + `RawPathPanel`/`ConfigParamsModal`）：裸路径专家直发，**不重造 `{i}` 选择器**，改为按操作类型校验 + 提示：
- 一律拒绝 `{i}` 占位符（裸路径不经字典翻译，须填具体实例号）。
- ADD：path 须以 `.` 结尾、末级不能是纯数字（实例号）；提示「末级不带实例号，设备自动分配」。
- RMV：path 须以 `.<实例号>.` 结尾；提示「指定要删除的实例」。
- 校验失败行内红字提示 + 输入框 error 态，且禁用「确定并执行」。

**命令参数（结构化）标签 `{i}` 默认值**（用户决策 2026-06-06）：
- **实例选择器**：path（LST/MOD）/ targetObject（ADD/RMV）中的父级 `.{i}.` 渲染为实例号输入，**默认每个 1**；前端 `computeInstanceSlots` 按 `.{i}.` 个数 + 前一段对象名生成槽位（key=`i01`/`i02`…，与后端 `substituteInstanceSelectors` 字典序左→右映射对齐），随 `instanceSelectors` 下发。
- **标量参数值**：MOD/ADD 的「值」默认取 `standard_params.min_value`。
- 之前 ConsoleV2 结构化通道**不收集** `instance_selectors`，含 `.{i}.` 的命令会在后端 `instance_selectors count mismatch` 失败；本次补齐。

**后端**：运行时 sub-fields 端点（`GET /mml/commands/:id/sub-fields`）的 `SubFieldDTO` / `MMLCommandSubFieldEnriched` 增 `min_value`（取自 `standard_params.min_value`，`ListEnrichedByCommand` SELECT/scan 同步）。`target_object` 复用 GroupTreeCommand 既有字段，前端 `CommandItem.targetObject` 透传。结构化 ADD/RMV/{i} 替换、ADD 复合 `.{NEW}.`、RMV `RmvInstanceIndex` 拼接逻辑（`console_executor.go`）保持不变。

> 单测：`ConsoleV2/__tests__/instanceAndRaw.test.ts` 覆盖 `validateRawPath` 与 `computeInstanceSlots`。

---

## 4. 需求映射

| 用户需求 | 对应设计 | 后端 |
|---|---|---|
| ① 结果表格 | §3.4 结果表格（设备行×参数列，状态/故障，实时填充） | 新增 results-schema + 行升级 |
| ② 批量设备 + 汇总 + 下载 | §3.5 复用 Fanouter；§3.4 汇总条；§3.6 单/全下载 | 复用执行链路 + 新增 export |
| ③ 选设备/命令缩占比，侧重操作/参数/结果 | §3.2 顶部选择条+Drawer；左操作区+右结果表格 | 无 |
| ④ 文档先行审核 | 本文档 | — |

---

## 5. 实施分阶段

| 阶段 | 内容 | 依赖 | 估时 |
|---|---|---|---|
| P0 布局重排 | 顶部选择条 + Drawer 收纳设备/命令；左操作区 + 右结果占位 | 纯前端，复用现组件 | 2–3d |
| P1 结果表格（读） | results-schema 端点 + 行升级；前端表格 + SSE 就地更新 + 汇总条 | P0 | 4–5d |
| P2 下载 | export 端点（CSV/XLSX，单/全）+ 前端下载入口；大集合走 worker+MinIO | P1 | 4–6d |
| P3 收尾 | 原始报文折叠、状态/设备过滤、列设置、e2e | P1/P2 | 2–3d |

---

## 6. 待确认决策点（请审核时圈选）

1. ~~选择收纳形态~~ **已定（本轮）：顶部条 + Modal 弹框**——设备/命令各一弹框；设备弹框支持「设备编码 / 产品 / 产品类型」三维筛选（见 §3.3）。
2. **主体分区**：左右「操作｜结果」（推荐）／ 上下「操作 / 结果」。
3. **结果表格列**：LST 用「参数路径动态列」（推荐，直观对比多设备同参数）／ 统一「设备行+可展开详情」（列少、详情藏行内）。
4. **下载范围与格式**：CSV / XLSX / JSON 选哪些？全设备是否要「每设备单文件 zip」？
5. **导出落地**：小数据同步流即可；是否需要 MinIO + worker 异步（大批量设备/参数时）？
6. **原始报文**：保留为折叠面板（推荐）／ 独立 Tab／ 完全移除（只留表格）。
7. ~~范围边界~~ **已定（2026-06-04）：覆盖「参数路径指定」裸 path 模式，采用方案 A（左面板顶部 Segmented 双模式切换）**——用户已确认布局，进入实施。形态见 §3.9。
8. **二轮完善（2026-06-04，§3.10）**：①设备弹框**表头全选 = 筛选命中的全部数据**（跨页，去掉单独「全选」复选框、不默认全选）+ **单次上限 200 台**（已定；接口分析见 §3.10.1，可配置 `MAX_CONSOLE_SELECT_ALL=200`）；②命令弹框「参数 PATH」去只读、名称+PATH、截断 hover；③配参改为「③ 配置参数」弹框；④⑤⑥ 主舞台改为「命令记录(可收缩,展开 24%) ｜ 执行结果(默认最近一次,点击记录联动)」左右分区。**已实现（mock）**。
   - ~~仍待确认：命令记录是否需要跨刷新持久化~~ **已定（2026-06-05，§3.11.4）：前端 localStorage 持久化 + 清空，记录存命令 ID 深链 `/mml/tasks/:id`**。
9. **三轮完善（2026-06-05，§3.11）—— 已全部定案**：
   - ①②③ 结果表格列定稿（左固定序号/SN/状态、中间每 path 一列、右固定下发/响应时间、移除独立故障码列）—**已定，前端可落**。
   - ④ **写类命令读后核实**：**采方案 A·命令链**（写后自动追加 LST 比对，派生 success/mismatch）；**只写参数、重启生效参数不判失败**，落 `unverified` 并**在列表明确提示原因**（只写/需重启/查询失败）。
   - ⑤ **任务层级 ID**：**采方案 C**（前端先建模，`device_tasks.parent_id` 迁移后置）；**命令 ID = `mml_tasks.id`（每次批量执行全局唯一 UUID）**；设备任务 ID = `device_tasks.id`。
   - ⑥ **命令记录持久化+清空**：**localStorage 只存命令 ID 数组，凭命令 ID 现拉全部数据**；提供清空。
   - 后端两项（④读后核实落地、⑤逐 PATH 落库）拆为独立 backlog 任务；本轮前端 mock 全量交付。重点风险见 §3.11.6。

---

## 7. 风险与兼容

- **执行链路零改动**，新增均为「读侧/导出侧」增量端点，回滚只影响结果展示，不影响下发。
- **大批量**（如几百台 × 几十参数）：表格虚拟滚动 + 导出走异步，避免前端卡顿/超大响应。
- **SSE 与表格一致性**：以 device_task 终态事件为准就地更新；断线重连后用 `GET /results` 全量回填兜底。
- **读后核实使 RPC 翻倍（§3.11.2）**：每个写命令追加一次查询，200 台写 = 400 次 CR，须在单次上限（200 台）与 ACS 准入容量内评估；只写参数须降级 `unverified` 不误判。
- **逐 PATH 落库需迁移（§3.11.3）**：`device_tasks` 缺 `parent_id`、现扇出非 per-path；建议先前端建模，迁移列为后置任务，避免本轮阻塞。
- **命令记录 localStorage（§3.11.4）**：只存摘要不存结果快照（容量），缓存任务被服务端清理需优雅提示，共享浏览器建议按用户隔离 key。
- **权限**：导出/下载端点复用 `devices` 权限组 + 下载 URL 鉴权。
