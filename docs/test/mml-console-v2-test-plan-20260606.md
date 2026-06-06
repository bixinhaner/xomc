# MML 控制台 V2（/mml/console-v2）测试方案

> 版本：2026-06-06 · 维护者：QA · 适用页面：`omcmb/webcode/src/pages/mml/ConsoleV2`
> 配套结果文档：[mml-console-v2-test-result-20260606.md](./mml-console-v2-test-result-20260606.md)
> 本文档为活文档，后续补充用例直接在 §5 追加 TC 编号。

---

## 1. 测试对象与环境

| 项 | 值 |
|---|---|
| 被测页面 | `/mml/console-v2`（MML 控制台 V2 三步流程：选择设备 → 选择命令 → 配置参数 → 执行 → 结果） |
| 运行环境 | docker compose 栈（`deployments/docker/docker-compose.yml`），web(nginx) `:8080`/`:8081`，app/acs/worker 容器 |
| 重启生效 | `docker compose -f deployments/docker/docker-compose.yml down && … up -d --build`（或定向 `up -d --build app worker web`） |
| 登录 | `admin / admin123` |
| 真机设备 | **SN=1202000240194DP0026**（cmcc / lte / product_class `FAP/mBS31001/SC`，**真实在线**，连 ACS，可响应 TR-069 RPC） |
| 自动化 | 浏览器自动化（Playwright MCP），辅以后端 DB / API 校验 |

> ⚠️ 数据备注：该 SN 在 `devices` 表存在 2 行（不同 `product_id`），属数据异常，测试时关注是否影响设备筛选/执行（见 TC-DEV-09）。

---

## 2. 页面功能分析（被测能力清单）

### 2.1 第一步「选择设备」(DeviceSelectModal)
- 列表固定 `is_online=true`，**只显示在线设备**；筛选条件 = 草稿态，**点蓝色「搜索」按钮**才触发查询（输入 SN / 选产品 / 选产品类型不实时调 API）。
- 列：`SN | 状态 | 产品 | 产品类型`（状态前置、无设备分组）。
- 「批量输入」：输入多 SN → 设为 `sn_list` 过滤 + `is_online`，**只显示匹配且在线的**，并清空已选。
- 分页：每页 10/20/50 可切，行超高度竖向滚动。
- 表头全选 = 选中筛选命中全部（跨页，上限 200）。

### 2.2 第二步「选择命令」(CommandSelectModal)
- 左侧命令树（分组 → 命令），搜索框「命令分组 / 名称」。
- 标题处「指定参数 ›」直接跳「配置参数 / 指定参数」裸路径模式。
- 右侧详情：操作类型缩写(Tag) + 命令名称 + 参数 PATH 列表；点击命令名有选中高亮。

### 2.3 第三步「配置参数」(ConfigParamsModal)
- 双标签：**命令参数（结构化）** / **指定参数（裸路径）**；标题「选择命令 ›」与命令选择「指定参数 ›」双向切换。
- 命令参数布局统一（LST/MOD/ADD/RMV 同款头部：全选(仅读类) + 操作类型 + 命令名称 + 操作提示）。
- **实例 `{i}`**：path(LST/MOD)/targetObject(ADD/RMV) 的父级 `.{i}.` 渲染实例号输入，**默认 1**。
- **值默认**：MOD/ADD 标量参数值默认取 `standard_params.min_value`。
- 写类按操作类型提醒（ADD/RMV 单行锁定提示；MOD 确认提醒）。
- **指定参数（裸路径）**：ADD/RMV 按 TR-069 校验（拒绝 `{i}`；ADD 须 `.` 结尾且末级非实例号；RMV 须 `.<实例号>.` 结尾），不合规禁用执行。

### 2.4 执行 + 结果表格 (ResultTable)
- 结构化 → `POST …/execute-statements-structured`；裸路径 → `POST /mml/execute`。
- SSE（`mml_device_frame`/`mml_task_completed`）实时回填；**结果表格列按命令/执行 path 动态生成**；读类（GPV）按 path 回填读回值（exact→leaf 匹配）。
- 状态四态 + 故障码；行「查看」详情（原始报文）。

### 2.5 命令记录 (useConsoleHistory)
- localStorage 存命令 ID 数组，本会话 recordStore 存完整结果；切换记录切换结果显示；清空。

### 2.6 结果下载
- 后端 CSV 导出：`POST /mml/tasks/:id/export`（全设备汇总）、`POST /mml/tasks/:id/devices/:sn/export`（单设备）→ presigned URL 下载。

---

## 3. 测试范围与方法

1. **功能完整性 + 结果正确性**：三步流程闭环可走通；真机执行后读回值正确显示。
2. **真机执行**：以 SN=1202000240194DP0026 执行「命令参数」与「指定参数」两类，读结果并分析。
3. **动态列**：验证结果列随命令/path 动态增减，每个 path 的读回结果清晰对应。
4. **命令记录**：多次执行后切换记录，结果正确切换。
5. **下载**：批量 + 单设备 CSV 下载成功、格式完整易读。
6. **TR-069 合规**：path 执行严格遵守 GetParameterValues / SetParameterValues / AddObject / DeleteObject 语义（见 §6）。

方法：Playwright 驱动真实浏览器走 UI；关键点用后端 DB（`mml_tasks` / `device_tasks`）与 API 交叉校验；发现 bug → 修复 → `up -d --build` → 复测。

---

## 4. TR-069 协议合规要点（执行硬约束）

| RPC | 触发操作 | path/对象约束 |
|---|---|---|
| GetParameterValues | LST | path 所有 `.{i}.` 须具体实例号；返回 name/value 列表 |
| SetParameterValues | MOD | 同上 + 每 path 带值；值类型/范围合规 |
| AddObject | ADD | ObjectName=对象表路径，`.` 结尾、**末级不带实例号**；CPE 回 InstanceNumber |
| DeleteObject | RMV | ObjectName 以 `.<实例号>.` 结尾（具体实例） |

裸路径模式：不经字典翻译，**拒绝 `{i}` 占位符**。结构化模式：父级 `.{i}.` 由实例选择器替换、ADD 复合 `.{NEW}.` 运行时回填、RMV 末级 `RmvInstanceIndex` 拼接。

---

## 5. 测试用例

> 状态列在结果文档回填。P=优先级（P0 必过 / P1 重要 / P2 一般）。

### 5.1 选择设备
| 编号 | P | 用例 | 预期 |
|---|---|---|---|
| TC-DEV-01 | P0 | 打开「选择设备」 | 列表只显示在线设备；列序 SN/状态/产品/产品类型，无设备分组 |
| TC-DEV-02 | P0 | 输入 SN 草稿后**不点搜索** | 列表不变（不实时查询） |
| TC-DEV-03 | P0 | 输入 SN + 点蓝色「搜索」 | 按 SN 过滤；命中目标 SN |
| TC-DEV-04 | P1 | 选产品/产品类型后点搜索 | 按 product_id / product_class 过滤生效 |
| TC-DEV-05 | P0 | 批量输入多个 SN（含目标 SN + 1 个离线/不存在 SN） | 只显示匹配且在线的；已选被清空；离线/不存在不显示 |
| TC-DEV-06 | P1 | 分页切 20/50 条 | 每页条数生效，超高度竖向滚动 |
| TC-DEV-07 | P1 | 表头全选 | 选中筛选命中全部（≤200） |
| TC-DEV-08 | P0 | 勾选目标 SN → 确定 | 顶部条显示已选 1 台 |
| TC-DEV-09 | P2 | 目标 SN 重复行（数据异常）影响 | 设备列表/执行不因重复行报错或重复扇出 |

### 5.2 选择命令
| 编号 | P | 用例 | 预期 |
|---|---|---|---|
| TC-CMD-01 | P0 | 命令树渲染 + 搜索「命令分组/名称」 | 分组→命令两级；搜索过滤正确 |
| TC-CMD-02 | P0 | 选一个 LST 命令 | 右侧详情显示操作类型 + 命令名 + 参数 PATH；点击高亮 |
| TC-CMD-03 | P1 | 标题「指定参数 ›」 | 跳「配置参数 / 指定参数」裸路径模式 |

### 5.3 配置参数（命令参数 / 结构化）
| 编号 | P | 用例 | 预期 |
|---|---|---|---|
| TC-CFG-01 | P0 | 选 LST 命令进入 | 头部=全选+操作类型+命令名+「勾选要查询的参数」；PATH 勾选列表 |
| TC-CFG-02 | P0 | 含 `.{i}.` 的命令 | 渲染实例号输入，默认 1；下发 instance_selectors |
| TC-CFG-03 | P0 | 选 MOD 命令 | 头部一致；标量参数值默认 = min_value；可改值 |
| TC-CFG-04 | P1 | 选 RMV 命令 | 实例号输入；提醒「单次仅作用一个对象」 |
| TC-CFG-05 | P1 | 选择命令/配置参数双向切换 | 标题链接互跳，已选命令保留 |

### 5.4 配置参数（指定参数 / 裸路径）
| 编号 | P | 用例 | 预期 |
|---|---|---|---|
| TC-RAW-01 | P0 | 裸路径 LST 具体实例 path | 校验通过，可执行 |
| TC-RAW-02 | P0 | 路径含 `{i}` | 行内报错 + 禁用执行 |
| TC-RAW-03 | P0 | ADD 路径末级带实例号 / 不以 `.` 结尾 | 报错 + 禁用 |
| TC-RAW-04 | P0 | RMV 路径不以 `.<实例号>.` 结尾 | 报错 + 禁用 |

### 5.5 执行 + 结果（真机 SN=1202000240194DP0026）
| 编号 | P | 用例 | 预期 |
|---|---|---|---|
| TC-EXE-01 | P0 | 命令参数 LST 执行 | 下发成功；SSE 回填；结果行 success；**每个勾选 path 一列**，列显示读回值 |
| TC-EXE-02 | P0 | 结果列动态性 | 改变勾选 path → 结果列随之增减，path↔值清晰对应 |
| TC-EXE-03 | P0 | 指定参数 LST 执行 | 裸路径执行成功；结果列按 path 生成、回填读回值 |
| TC-EXE-04 | P1 | 行「查看」详情 | 显示原始 CWMP 报文 + 逐 path 读回 |
| TC-EXE-05 | P1 | 执行中/失败态 | 状态正确（执行中→success/failed）；故障码显示 |

### 5.6 命令记录
| 编号 | P | 用例 | 预期 |
|---|---|---|---|
| TC-HIS-01 | P0 | 连续执行 ≥2 个命令 | 命令记录列表新增，命令名/时间正确 |
| TC-HIS-02 | P0 | 点不同命令记录 | 结果表格切换为对应记录的结果（列 + 行 + 值都对） |
| TC-HIS-03 | P2 | 清空记录 | 列表清空 |

### 5.7 下载
| 编号 | P | 用例 | 预期 |
|---|---|---|---|
| TC-DL-01 | P0 | 全设备汇总 CSV 下载 | 生成 + 下载成功；CSV 列=参数 path、行=设备，值=读回，UTF-8 中文正常 |
| TC-DL-02 | P0 | 单设备 CSV 下载 | 生成 + 下载成功；纵向(path,值) + 摘要，清晰易读 |
| TC-DL-03 | P1 | CSV 内容与页面一致 | 下载值与结果表格一致 |

### 5.8 多设备批量 + 结果详情 + 命名（2026-06-06 第二轮，可复用）
> 环境约束：当前**仅 `1202000240194DP0026` 真实可响应**；DB 中其余 `is_online=true` 设备为 stale（ACS 发 Connection Request 后设备永不应答，device_task 长期 `pending`，仅任务到期才转 `expired/failed`，约 1h）。故"其余设备失败"在测试窗口内表现为 **执行中(pending)**，超时后才转失败——非阻断，属环境数据问题。真正"失败行"用**无效 path 触发 9005** 复现。

| 编号 | P | 用例 | 步骤 | 预期 |
|---|---|---|---|---|
| TC-MULTI-01 | P0 | 多设备批量（含目标） | 选择设备里勾目标 SN + 另 2 台在线设备 → 选命令/指定参数 → 执行 | 顶部「已选 3 台」；下发成功；目标设备结果=成功+读回值；汇总「设备总数 3 / 成功 1 / 执行中 2」 |
| TC-MULTI-02 | P1 | 多设备结果区分 | 同上，观察结果表 + 汇总 | 目标行成功有值；stale 设备行执行中（超时后失败）；逐设备状态独立 |
| TC-FAIL-01 | P0 | 真实失败行（9005） | 单设备执行含设备不支持 path 的命令（如「设备基本信息」17 path，整体下发） | 该设备结果=失败；故障码 9005；读回值列空；详情含故障信息 |
| TC-DETAIL-01 | P0 | 「查看」详情命令 ID | 结果行点「查看」 | 命令 ID = `mml_tasks.id` 正确显示且可复制；设备任务 ID、设备 SN、状态、下发/响应时间齐全 |
| TC-NAME-01 | P0 | 命令记录 ↔ 任务记录 名称对应（req4） | 执行后看「命令记录」名 与 `/mml/task-records` 任务名 | 两者一致（裸路径=「查询 软件版本」；结构化=命令名）；后端 `task_name` = 命令记录名 |
| TC-HIS-DEEPLINK | P1 | 命令记录无「打开任务详情」（req8） | 看命令记录条目 | 不再有跳「任务详情」深链（一条命令对应多条分设备任务记录，单 task 详情无法对应）；逐设备详情走结果行「查看」 |
| TC-DL-MULTI | P1 | 多设备 CSV 成败区分（req7） | 多设备执行后「下载全部 ▸ CSV」 | 汇总 CSV 每行含「状态(成功/失败/执行中)」+「故障码」+「故障信息」逐设备区分；失败设备状态=失败、有故障信息 |
| TC-PATHMODE-1 | P0 | 逐 PATH（指定参数/裸路径） | 指定参数填 2 path（1 有效+1 无效）→「逐 PATH」→ 执行 | 后端拆 2 条 command（2 device_task，顺序）；结果合并一行：有效 path 填值、无效 path 标「✗ 失败」、行状态=失败；「查看」逐 path 列成功/失败 |
| TC-PATHMODE-2 | P0 | 逐 PATH（命令参数/结构化） | 选 LST 命令，命令参数仅勾「软件版本(有效)+附加硬件版本(无效)」→「逐 PATH」→ 执行 | 同上；结果列用友好名（软件版本/附加硬件版本）；命令记录名=命令名 |
| TC-PATHMODE-3 | P1 | 整体下发对照 | 同命令选「整体下发」+ 含无效 path | 单条 GPV 全或无 → 整行 9005 失败（对照逐 PATH 的混合呈现） |
> 注：逐 PATH 顺序执行，N path 需等 ~N×7s；自动化输入 AntD AutoComplete/Checkbox 用真实键鼠、勿按 Escape。

---

## 5.9 复用说明（如何再次验证）
- **环境**：docker compose 栈起后访问 `http://localhost:8081/mml/console-v2`（admin/admin123）。改前端后重建 **必须** `docker compose -f deployments/docker/docker-compose.yml build --no-cache web && … up -d web`（`up --build` 会命中层缓存）。
- **真机设备**：`1202000240194DP0026`（唯一真实在线）。多设备测试再补 2 台任意在线设备即可（它们会停在执行中）。
- **数据交叉校验**：`docker compose exec postgres psql -U omcgo -d omcgo`：`mml_tasks`(task_name/status)、`device_tasks`(device_sn/status/error_code)；结果端 `GET /api/v1/mml/tasks/:id/results`。
- **失败行复现**：用「设备基本信息」LST（含设备不支持 path）整体下发 → 9005 失败行。
- **下载校验**：`POST /api/v1/mml/tasks/:id/export`(汇总) / `…/devices/:sn/export`(单设备) → presigned URL → 取 CSV 比对。
- 逐条用例按 §5 表「步骤/预期」复跑；P0 必过。

---

## 6. 通过标准

- P0 用例全部通过；P1 无阻断性问题；P2 记录但不阻断。
- 真机 LST 读回值与设备实际一致（抽样核对）。
- 结果列动态、path↔值对应清晰无错位。
- TR-069 语义无违背（尤其 ADD/RMV/{i}）。
- CSV 格式完整、中文可读、与页面一致。
- 所有修复在 `up -d --build` 后复测通过。
