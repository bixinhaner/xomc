# MML 脚本 TXT 导入重设计

> 版本：2026-07-10
> 状态：实现完成；Task 13 的本地栈/浏览器验收需在运行完整 Docker 栈后补做
> 适用模块：MML 脚本库、脚本导入校验、MML 任务创建、任务调度与结果追溯
> 关联文档：
> - `docs/design/MMLScript-模板导入校验与执行逻辑总结.md`
> - `docs/design/MMLTemplate.txt`
> - `docs/design/mml-script-task-device-bound-redesign-20260708.md`
> - `docs/design/mml-tasks-vs-device-tasks-analysis.md`

## 1. 决策摘要

MML 脚本库继续作为可复用资产保留，但取消在线编写、命令选择、参数编辑和公共/设备绑定模式选择。TXT 文件成为脚本内容的唯一来源：用户下载模板，在本地填写后上传；服务端完成权威校验，任何错误都阻止保存；保存后的内容只读，修改脚本内容必须重新上传完整 TXT。

脚本保存与任务执行保持分离。用户从脚本库发起执行时配置立即、挂起、定时、周期、离线等待和失败重试。执行端复用现有 `mml_tasks -> device_tasks -> Sequencer`，同一 SN 按 TXT 行序串行，不同 SN 并行。

本次按全量切换实施，不兼容、不迁移现有 MML 脚本和任务数据。升级时清理现有 `mml_scripts`、`mml_tasks` 以及 `source='mml'` 的设备子任务，但不得影响其他来源的 `device_tasks`。

## 2. 背景与现状

当前系统已经具备设备绑定计划、任务调度、设备级 fanout、顺序执行和结果聚合，但脚本库仍暴露在线文本编辑、命令选择器、参数编辑器和执行模式等多层概念。用户需要先理解并保存脚本，再进入第二套界面执行，学习成本高，也容易让浏览器解析结果被误认为服务端权威结果。

参考网管采用“下载模板、填写 TXT、上传校验、创建任务”的低门槛方式，值得借鉴的核心不是页面外观，而是以下边界：

1. 空行和 `#` 注释不执行，一行只表达一条 MML。
2. 标准和自定义命令分别识别，命令、参数、SN 和设备类型分层校验。
3. 错误按原文件行号返回并可导出。
4. 导入校验与实际执行分离，设备在线、TR-069 响应和重试属于运行时条件。
5. 同一设备串行，不同设备并行。

本设计不照搬参考系统的两点缺陷：不允许一行多个 SN，不提供 Continue 执行正确行。xomc 使用服务端权威校验、严格的单 SN 行格式和现有 Sequencer 保证顺序。

## 3. 目标与非目标

### 3.1 目标

1. 新增脚本只需填写基本信息并上传 TXT。
2. 服务端统一完成解析、命令、参数、设备和产品能力校验。
3. 任何错误阻止保存；警告可保存但必须显式展示。
4. 脚本内容只读，重新上传是唯一内容更新方式。
5. 保存结果、执行快照和设备任务均可追溯到文件摘要与原始行号。
6. 复用现有 MML 调度、重试、fanout、Sequencer 和结果聚合。
7. v1、v2、v3 三套皮肤提供相同业务能力。

### 3.2 非目标

1. 不提供在线逐行编辑、复制、拖拽排序或参数批量替换。
2. 不支持 CSV、XLSX 或 `SN | order | command` 作为新导入格式。
3. 不支持一行绑定多个 SN。
4. 不提供 Continue、忽略错误行或保存无效草稿。
5. 不引入跨设备依赖、条件分支、循环和人工审批节点。
6. 不保留现有 MML 脚本和任务历史数据。

## 4. 用户流程

### 4.1 新增脚本

```text
下载模板
  -> 本地填写 TXT
  -> 上传 TXT
  -> 服务端解析与校验
  -> 只读预览和错误定位
  -> 全部错误修正后重新上传
  -> 确认保存到 mml_scripts
```

脚本库操作收敛为：新增、查看、执行、重新导入、下载 TXT、修改基本信息、删除。

### 4.2 执行脚本

```text
选择脚本
  -> 配置执行方式与重试策略
  -> 服务端动态预检
  -> 错误阻断 / 警告确认
  -> 复制脚本计划为 mml_task 快照
  -> 调度或立即 fanout
  -> device_tasks 执行与结果聚合
```

脚本保存不自动创建任务。定时和周期任务在实际触发前再次检查动态条件。

## 5. TXT 格式

### 5.1 推荐格式

```text
# 每行一条命令，井号开头为注释
LST DEVICE_INFO;1202000091177SP0005
MOD DEVICE_INFO:USER_LABEL=Site-A;1202000091177SP0006
LST DEVICE_INFO;1202000091177SP0006
```

### 5.2 格式规则

1. 仅接受 `.txt`，编码为 UTF-8，可带 UTF-8 BOM。
2. 兼容 LF 和 CRLF；保存前统一换行为 LF。
3. 空行和 `trim` 后以 `#` 开头的行忽略，但保留原始行号。
4. 一行只能有一条有效命令。
5. 每条有效行必须使用 `命令;SN`；最后一个顶层分号后的片段解析为 SN。
6. 每行只允许一个 SN；逗号分隔多个 SN 是错误。
7. 同一 SN 的内部顺序由有效行在原文件中的先后自动生成，用户不填写 `order`。
8. 参数中的空格、逗号、分号等特殊字符沿用现有 MML 引号或大括号规则。
9. 单脚本最多 200 个不同 SN、2000 条有效命令。

## 6. 服务端校验

### 6.1 校验层级

| 层级 | 校验内容 | 处理 |
|---|---|---|
| 文件 | 扩展名、编码、空文件、大小、单行长度、有效行数 | 错误阻止 |
| 语法 | 注释、操作类型、命令结构、参数分隔、尾部 SN、单行单命令 | 错误阻止 |
| 命令 | 标准/自定义命令存在性、启用状态、RPC 解析 | 错误阻止 |
| 参数 | 必填、未知、读写属性、类型、枚举、范围、正则 | 错误阻止 |
| 设备 | SN 存在性、制式、产品型号和命令能力匹配 | 确定无效时阻止 |
| 运行条件 | 离线、危险命令、需要重启、路径映射回退 | 警告，可保存 |

命令码、SN、产品和参数定义必须先去重后批量查询，禁止逐行访问数据库。

### 6.2 逐行结果

```json
{
  "line_no": 3,
  "raw_line": "MOD DEVICE_INFO:USER_LABEL=Site-A;SN002",
  "device_sn": "SN002",
  "command_code": "MOD DEVICE_INFO",
  "status": "error",
  "issues": [
    {
      "code": "MML_PARAMETER_UNKNOWN",
      "field": "USER_LABEL",
      "message": "命令 MOD DEVICE_INFO 不包含参数 USER_LABEL"
    }
  ]
}
```

有任意错误时，确认保存按钮禁用，并允许按错误/警告筛选和下载错误报告。只有警告时允许保存，但警告必须在预览区保持可见。

前端解析只用于上传前反馈。服务端解析、校验和规范化结果是唯一权威数据，保存接口不接收前端构造的 `content` 或 `plan_items`。

## 7. 两阶段原子导入

### 7.1 校验接口

```http
POST /api/v1/mml/scripts/import/validate
Content-Type: multipart/form-data
```

请求只包含 `file`。响应包含：

- 一次性 `validation_token` 和过期时间；
- 文件名、大小、编码和 SHA-256；
- 有效行、设备、操作类型、错误和警告统计；
- 规范化计划预览；
- 逐行问题。

服务端把规范化文本、解析结果、摘要、校验版本和用户标识暂存在 Redis，默认有效期 15 分钟。令牌必须高强度随机、只存哈希、绑定当前用户且只能成功消费一次。

### 7.2 保存接口

```http
POST /api/v1/mml/scripts/import
Content-Type: application/json
```

```json
{
  "validation_token": "opaque-token",
  "script_name": "北向站点参数核查",
  "description": "现场维护脚本",
  "tags": ["4G", "巡检"]
}
```

服务端依次检查令牌存在、未过期、未消费、属于当前用户且校验结果无错误；随后在一个数据库事务中保存脚本。事务成功后消费令牌，失败时保留令牌供有效期内重试。

### 7.3 重新导入

```http
POST /api/v1/mml/scripts/{id}/import/validate
PUT  /api/v1/mml/scripts/{id}/import
```

重新导入必须上传完整 TXT，并携带脚本版本或 `updated_at`。若脚本已被其他请求更新，返回 `409 Conflict`。名称、描述和标签可通过普通基本信息接口修改；内容只能通过重新导入替换。

## 8. 数据模型

### 8.1 mml_scripts

清理历史数据后，脚本统一采用 TXT 导入模型。保留现有标识、基本信息、创建审计与最近执行字段，内容相关字段统一为：

| 字段 | 类型 | 说明 |
|---|---|---|
| `content` | text | 规范化 TXT，换行统一为 LF |
| `original_filename` | text | 去除路径和控制字符后的文件名 |
| `content_sha256` | text | 规范化内容摘要 |
| `validation_version` | text | 解析和校验规则版本 |
| `validated_at` | timestamptz | 最近内容校验时间 |
| `plan_items` | jsonb | 服务端生成的规范化设备绑定计划 |
| `validation_summary` | jsonb | 行、设备、操作类型和警告统计 |

`plan_items` 是脚本执行源，单项至少包含：`line_no`、`device_sn`、`order`、`raw_line` 和规范化 `command`。

首期不新建脚本行表。2000 行上限下 JSONB 与现有 `mml_tasks.plan_items` 一致，减少额外关联和迁移复杂度。

### 8.2 mml_tasks

执行时复制脚本计划为不可变任务快照：

```text
mml_scripts.plan_items
  -> 执行前动态预检
  -> mml_tasks.plan_items
  -> device_tasks
```

`mml_tasks` 继续保留 `script_id`、`execute_mode=device_bound` 和 `plan_items`，新增：

| 字段 | 说明 |
|---|---|
| `script_content_sha256` | 本次任务使用的脚本内容版本 |
| `script_validation_version` | 本次任务对应的校验规则版本 |

脚本重新导入不会改变已创建任务的计划快照。

## 9. 执行 API 与语义

### 9.1 创建执行任务

```http
POST /api/v1/mml/scripts/{id}/executions
```

请求只包含任务名、执行方式、时间和重试策略，不接收 `commands`、`device_sns` 或 `plan_items`。服务端加载脚本并重新校验：

1. 文件摘要和计划完整性；
2. 校验规则是否升级；
3. 命令是否仍存在并启用；
4. SN 是否仍存在；
5. 产品能力和参数映射是否仍适配；
6. 设备是否离线；
7. 是否包含危险或需要重启的命令。

有错误时返回逐行问题且不创建任务。只有警告时首次返回警告；用户确认后以 `confirm_warnings=true` 重试。全通过后，在事务中保存 `mml_task` 快照并进入现有调度或 fanout 链路。

### 9.2 执行顺序与失败

1. 所有脚本统一使用设备绑定模式。
2. 同一 SN 严格按 TXT 行序串行，不同 SN 并行。
3. 单行失败后继续执行该 SN 的后续命令，逐行保留成功和失败结果。
4. 设备离线按任务配置等待；超时后相关行失败。
5. TR-069 Fault、超时和异常沿用失败重试配置。
6. 定时和周期任务在每次实际触发前重新检查动态条件。
7. 每个结果必须可追溯到脚本 SHA-256、原始行号、SN、命令、设备任务和响应。

## 10. 前端设计

### 10.1 脚本库

保留脚本列表、详情和最近执行信息。操作只包括新增、查看、执行、重新导入、下载 TXT、修改基本信息和删除。

删除在线文本编辑、命令选择器、参数编辑器以及公共/设备绑定模式选择器。

### 10.2 新增和重新导入

页面分为：

1. 基本信息：脚本名称、描述、标签；
2. 导入文件：选择 TXT、下载模板、文件摘要；
3. 校验摘要：有效行、设备、操作类型、错误和警告；
4. 只读预览：行号、SN、顺序、命令、参数摘要和校验结果；
5. 错误工具：仅看错误、仅看警告、下载错误报告；
6. 操作区：取消、确认保存。

上传和校验期间禁用保存；存在错误时持续禁用。重新导入使用相同页面，但成功后整体替换脚本内容。

### 10.3 三皮肤边界

以下能力放在 `frontend-core`：

- 导入 API、执行 API 和类型；
- React Query Hooks；
- 错误码与国际化 key；
- 校验摘要、计划预览和分页数据模型；
- 文件限制和快速客户端检查。

`webcode`、`webcode-v2`、`webcode-v3` 分别实现符合自身组件体系的页面和弹窗，但字段、校验状态、操作能力、分页及提交语义必须一致。

## 11. 后端组件边界

| 组件 | 职责 |
|---|---|
| `ScriptImportParser` | 编码、BOM、注释、行格式、SN 和参数解析 |
| `ScriptImportValidator` | 命令、参数、设备、产品能力和规模校验 |
| `ImportSessionStore` | Redis 校验会话、用户绑定、过期和一次性消费 |
| `ScriptImportService` | 编排校验、摘要、预览和事务保存 |
| `ScriptExecutionPreflight` | 创建任务和调度触发前的动态预检 |
| `ScriptRepository` | 保存文本、计划和校验元数据 |
| `Fanouter` / `Sequencer` | 复用现有设备派发和设备内顺序执行 |

Parser 和 Validator 不依赖 Gin。Handler 只负责请求体限制、参数绑定、鉴权和 HTTP 响应。

## 12. 错误契约

| 错误码 | 含义 |
|---|---|
| `MML_FILE_TYPE_INVALID` | 不是 TXT |
| `MML_FILE_ENCODING_INVALID` | 编码不可识别 |
| `MML_FILE_EMPTY` | 没有有效命令 |
| `MML_FILE_TOO_LARGE` | 文件或行数超限 |
| `MML_LINE_FORMAT_INVALID` | 行格式错误 |
| `MML_DEVICE_SN_REQUIRED` | 缺少 SN |
| `MML_DEVICE_SN_MULTIPLE` | 一行包含多个 SN |
| `MML_DEVICE_NOT_FOUND` | SN 不存在 |
| `MML_COMMAND_NOT_FOUND` | 命令不存在 |
| `MML_COMMAND_DISABLED` | 命令已禁用 |
| `MML_PARAMETER_UNKNOWN` | 参数不存在 |
| `MML_PARAMETER_REQUIRED` | 缺少必填参数 |
| `MML_PARAMETER_READ_ONLY` | 修改只读参数 |
| `MML_PARAMETER_VALUE_INVALID` | 参数值类型、范围或格式错误 |
| `MML_DEVICE_COMMAND_INCOMPATIBLE` | 设备与命令不兼容 |
| `MML_IMPORT_TOKEN_EXPIRED` | 校验会话过期 |
| `MML_IMPORT_TOKEN_CONSUMED` | 校验会话已使用 |
| `MML_SCRIPT_VERSION_CONFLICT` | 重新导入并发冲突 |
| `MML_EXECUTION_PREFLIGHT_FAILED` | 执行前动态校验失败 |

HTTP 状态：`400` 表示文件或字段错误，`404` 表示脚本不存在，`409` 表示令牌或版本冲突，`413` 表示上传超限，`422` 表示业务校验不通过，`503` 表示命令字典、设备能力或 Redis 等必要依赖不可用。

## 13. 安全、审计与可观测性

1. 不信任浏览器 MIME；服务端同时检查扩展名和内容。
2. 文件名去除路径、控制字符和不可见字符。
3. 限制请求体、文件、单行、有效行和单行参数数量。
4. 日志只记录 SHA-256、文件大小、行数和错误码，不记录完整脚本和敏感参数值。
5. 审计记录上传人、保存人、脚本 ID、文件摘要、校验版本和警告确认。
6. 危险命令保存时标记，执行时再次确认。
7. 指标至少覆盖校验耗时、校验成功/失败数、各错误码数量、令牌过期/重放、执行前预检失败和计划行规模。

## 14. 数据清理与上线

### 14.1 数据库迁移

迁移严格限定 MML 数据：

1. 删除 `device_tasks WHERE source='mml'`；
2. 清空 `mml_tasks`；
3. 清空 `mml_scripts`；
4. 增加并约束 TXT、摘要、校验和计划字段；
5. 增加 `content_sha256` 和 JSONB 查询索引；
6. 校验不存在孤立 MML 子任务。

不得无范围级联删除其他来源的设备任务。

### 14.2 部署步骤

1. 停止 MML 新建、调度和消费；
2. 记录待清理 MML 脚本、父任务和子任务数量；
3. 仅清除 Redis 中 MML 队列和调度状态，禁止 `FLUSHDB`；
4. 执行数据库迁移；
5. 启动服务并验证健康状态；
6. 导入标准模板样例；
7. 验证任务快照、设备派发和三皮肤页面。

提交迁移前必须获取最新目标分支，检查 Goose 编号和非破坏性合并冲突，并运行：

```bash
bash omcgo/scripts/check-migrations.sh --strict
```

## 15. 测试策略

### 15.1 后端单元与集成测试

- UTF-8、BOM、CRLF、空行和注释；
- 标准和自定义 LST/MOD/ADD/RMV；
- 引号、大括号、逗号和分号参数；
- 缺少 SN、多 SN、未知 SN；
- 未知/禁用命令，未知/只读/缺失/越界参数；
- 同一 SN 顺序自动派生；
- 200 台、2000 行边界和超限；
- 批量查询及 N+1 防护；
- 校验令牌过期、越权、重放和事务失败；
- 重新导入版本冲突；
- 数据清理只影响 MML 来源。

### 15.2 执行测试

- 每个 SN 只执行绑定命令；
- 同一 SN 严格串行，不同 SN 并行；
- 前一行失败后继续下一行；
- 离线等待、失败重试、定时和周期触发前预检；
- 脚本重新导入不改变已创建任务快照；
- 结果可回溯到文件摘要和原始行号。

### 15.3 前端与端到端测试

- TXT 选择、上传、校验中、失败和重新上传；
- 错误阻止保存，警告允许保存；
- 只读预览、错误过滤和报告下载；
- 执行前警告确认；
- v1/v2/v3 能力一致；
- `npm run skin-parity` 和三皮肤 typecheck；
- 以 `docs/design/MMLTemplate.txt` 为参考，再准备全合法样例和覆盖各错误码的非法样例。

## 16. 分阶段实施

1. 实现后端 Parser、Validator、错误码及测试。
2. 实现 Redis 校验会话、导入 API、数据模型和事务保存。
3. 实现脚本执行 API、任务快照和动态预检。
4. 改造 v1 脚本库、新增、详情、重新导入和执行页面。
5. 补齐 v2/v3，并把共享能力收敛到 `frontend-core`。
6. 实现并验证限定范围的数据清理迁移。
7. 完成后端、前端、迁移和浏览器端到端验收。

## 17. 验收标准

1. 用户不能在线编写或修改脚本内容，只能上传 TXT。
2. TXT 中任一错误都会阻止脚本保存，系统不会静默丢弃错误行。
3. 脚本保存内容与服务端预览、计划和 SHA-256 一致。
4. 执行请求不能由前端替换脚本命令或 SN。
5. 同一 SN 按文件顺序执行，不同 SN 并行。
6. 任务结果可追溯到脚本版本、原始行号、SN 和设备任务。
7. 立即、挂起、定时、周期、离线等待和失败重试可用。
8. v1、v2、v3 具备相同业务能力。
9. 升级后旧 MML 脚本和任务数据被清除，其他任务来源不受影响。

## 18. Task 13 实现与验证证据（2026-07-10）

### 18.1 已落地契约

- 迁移文件：`omcgo/migrations/000016_redesign_mml_script_txt_import.sql`。
- 导入 API：`POST /api/v1/mml/scripts/import/validate`、
  `POST /api/v1/mml/scripts/import`，重新导入对应 `/:id/import/validate` 和 `PUT /:id/import`。
- 执行 API：`POST /api/v1/mml/scripts/:id/executions`；请求只含任务名、调度和重试策略，
  不接受浏览器提交的 `commands`、`device_sns` 或 `plan_items`。
- 稳定错误码：`MML_SCRIPT_VALIDATION_FAILED`（422）、`MML_IMPORT_TOKEN_CONSUMED` /
  `MML_IMPORT_TOKEN_EXPIRED`（409）、`MML_SCRIPT_VERSION_CONFLICT`（409）、
  `MML_FILE_TOO_LARGE`（413）、`MML_EXECUTION_VALIDATION_FAILED`（422）。
- 结果追溯字段：任务快照及设备结果保留 `plan_line_no`、`plan_device_sn`、`plan_order`、
  `script_content_sha256`。

### 18.2 可重复 E2E 与环境限制

`omcgo/scripts/e2e_mml_script_import.sh` 覆盖非法 TXT 422/无 token、合法 TXT 校验和保存、
同 token 重放幂等返回原脚本、服务端快照执行及结果追溯字段；样例为
`omcgo/internal/mml/testdata/import-valid.txt` 与 `import-invalid.txt`。本轮运行命令：

```text
bash omcgo/scripts/e2e_mml_script_import.sh
```

本轮先在旧进程上观察到导入路径 404；重建当前分支 app 镜像并直连
`http://localhost:18081` 后，使用管理员登录令牌复跑，15 项断言全部通过：非法/合法 TXT、
保存、同 token 幂等重放、任务快照哈希、结果计划行追溯。三皮肤真实浏览器验收仍需带 web
镜像和浏览器会话的环境补齐，不能用本轮 API 结果替代截图证据。

### 18.3 切换限制

发布顺序仍为停止 MML 新建/调度、执行 `omcctl mml reset-script-data --dry-run`，复核数量后
使用 `--apply --confirm DELETE-MML-RUNTIME`，应用 `000015` 迁移，启动服务，再运行 E2E 和
健康检查。`--apply` 是唯一 Redis 写入口且禁止 `FLUSHDB`；迁移删除旧 MML 数据，不提供
数据恢复 Down，回滚只能依赖发布前数据库快照。非 MML `device_tasks` 必须在切换前后单独
统计并保持不变。
