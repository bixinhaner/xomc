# 系统配置、存储、PM 与日志模块功能测试报告

> 测试日期：2026-07-20（Asia/Shanghai）
> 源码基线：`cccbb2b3bca9b8c4267e140856d4a2a65881d356`
> 分支状态：`main` 与 `origin/main` 一致
> 测试方式：Go / Vitest / ESLint / Playwright、真实后端页面、只读 API、PostgreSQL / TimescaleDB、MinIO、Docker 与主机磁盘交叉验证
> 安全边界：未保存页面、未修改密码/保留期/Agent 配置、未重启或重建服务

## 1. 总体结论

系统配置 9 个页签都能在真实后端页面中打开，现网 PM 保留策略、MinIO 生命周期、数据库日志清理和文件日志轮转也确实在运行；但整个模块不能判定为“功能全部通过”。

本轮结果：

| 结果 | 功能 |
|---|---|
| PASS | Basic 时区链路；当前 PM / TimescaleDB 策略运行；当前 MinIO ILM；当前文件日志轮转；匿名访问受保护接口返回 401 |
| PARTIAL | Basic OMC 名称；Storage；ACS Transfer；Agent；资源保留/背压；数据库日志保留 |
| FAIL | 安全配置整体契约；设备离线开关与默认值；系统仪表盘真实指标；服务端管理权限边界 |
| NOT TESTABLE | 现有非管理员账号的真实 403；未配置外部 Agent 的连通测试；会改变现网数据的页面保存/策略重建 |

最需要先处理的不是表单样式，而是：

1. 管理配置接口只有登录校验，没有真正 RBAC；
2. 通用配置接口可能返回原始 secret，并允许修改 `is_public`；
3. 安全页默认值与后端有效默认值漂移，首次保存会改变现网策略；
4. “保存成功”不代表 TimescaleDB / MinIO / 多进程运行态应用成功；
5. 设备离线开关没有消费者，基站超时仍被 seed 和页面固定为 100 秒；
6. 仪表盘显示的是硬编码演示数据，不是系统真实资源状态。

## 2. 使用的测试 Skills

本仓已有可用于功能测试的 skills，不需要额外安装插件：

| Skill | 本轮用途 |
|---|---|
| 项目 `$tdd` | 按可观察行为、边界和回归风险建立测试矩阵 |
| `superpowers:systematic-debugging` | 对每个失败测试定位到产品、测试、环境或并发稳定性层 |
| `browser:control-in-app-browser` | 登录真实后端页面并逐一验收 9 个页签的实际 DOM |
| `superpowers:writing-plans` | 把 UI → API → 存储 → 运行态拆成纵向测试计划 |
| `superpowers:verification-before-completion` | 交付前重新核对命令、退出码、失败数和报告内容 |

当前没有一个“一键覆盖所有系统配置”的专用 skill。现有 skills 负责方法和流程，业务契约仍需要本项目自己的测试用例。建议后续把本文矩阵固化为项目级 `system-config-functional-test` skill 或 CI 套件。

## 3. 自动化测试结果

| 检查 | 结果 | 说明 |
|---|---|---|
| `cd omcgo && go test ./...` | PASS | 全部 Go package、integration、e2e 测试通过 |
| `cd omcmb && npm run typecheck` | PASS | TypeScript 类型检查通过 |
| `cd omcmb && npm run lint` | FAIL | 1658 项：3 errors、1655 warnings |
| SystemConfig 定向 ESLint | PASS | SystemConfig、admin API、system hooks 无 error |
| 全量前端 Vitest | FAIL | 最终复跑 182 files 通过、6 files 失败；1274 tests 通过、7 tests 失败 |
| SystemConfig serializer | PASS | 6/6 通过 |
| Security Playwright mock | PASS | 5/5 通过 |
| 真实后端 system smoke | PARTIAL | 11/12 通过；操作日志导出断言已过期 |

### 3.1 前端全量测试的确定性失败

以下失败多数与系统配置无直接关系，但说明当前前端主线不是全绿：

1. `frontend-core/src/utils/ufteCategory.test.ts:37` 仍断言 `device_upgrade=false`，而实现已明确支持自定义模板直接保存该值，属于旧测试未随需求更新；
2. `frontend-core/src/i18n/en-US/index.ts:6356` 的 `.<instance>.` 被 ICU 当作未闭合标签，`messageFormat.test.ts` 正确检出格式错误；
3. `UserManagement/passwordMasking.test.ts` 和 `UserDropdown.test.ts` 通过 `new URL(..., import.meta.url)` 读取源码，在当前 Vitest 运行器中报 `The URL must be of scheme file`，属于测试实现/运行器兼容问题；
4. `AlarmRuleDrawer.test.tsx` 有 2 个断言仍查找旧文案“批量勾选当前页（3 台）”“已选设备（2 台）”，实际可访问名称已经变化，属于用例与 UI 契约漂移；
5. `KPIQuery/index.test.tsx` 有 1 个 5 秒超时，需要治理时序和测试隔离。

初跑曾出现 11 files / 17 tests 失败，最终完整复跑为 6 files / 7 tests 失败；失败数量受并行负载影响，本身也是测试稳定性问题。确定性的 UFTE 旧断言和 ICU 文案错误在两次运行中都复现。

### 3.2 Lint 的 3 个 error

- `frontend-core/src/agentkit/runtimeClient.test.ts:3`：未使用 `AgentRuntimeError`；
- `frontend-core/src/store/mmlConsoleStore.ts:406`：未使用异常变量 `e`；
- `webcode/src/pages/device/DeviceDetail/kpiSeries.ts:24`：注释中包含不可见异常空白。

SystemConfig 相关文件定向 lint 为 0 error，但仓库级质量门仍失败。

### 3.3 冒烟测试本身过期

`webcode/e2e/smoke/system.spec.ts:61-75` 要求操作日志页存在“导出”按钮；产品代码在 `webcode/src/pages/log/OperationLog/index.tsx:297-298` 明确隐藏按钮，因为当前导出只返回假的 task ID。因而：

- 产品问题：真实日志导出功能尚未实现；
- 测试问题：冒烟测试未随按钮隐藏同步更新；
- 不能通过恢复一个假按钮来让测试变绿。

## 4. 九个配置页签逐项测试

### 4.1 Basic

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| 页面加载、保存入口 | PASS | 真实页面可打开，控件与保存按钮正常渲染 |
| OMC 名称 | PARTIAL | 页面为空，与数据库 `mrOMCName=""` 一致；空名称是否允许缺少业务约束 |
| 系统时区 | PASS | 页面和数据库均为 `UTC`；Worker 定时任务使用该配置解释 cron |

建议增加 OMC 名称非空/长度/字符集服务端校验，以及时区变更后的下一次调度时间展示。

### 4.2 Security

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| 浏览器是否记住密码 | PASS | Playwright 分别验证 true / false 行为 |
| 空闲锁屏 | PASS | 1 分钟锁屏、解锁恢复、0 禁用、用户活动重置，5/5 通过 |
| 密码、锁定、验证码、并发会话表单 | FAIL | 当前 DB 只有 2 个 key，其余全部使用前端私有默认值 |
| 默认密码保护 | FAIL | 通用接口原样返回配置 `value`；专用脱敏模型可被绕过 |
| 保存校验 | FAIL | 只校验旧状态下的单个默认密码，缺少整批跨字段校验 |

实际页面显示的前端默认包括密码长度 10–23、有效期 70 天、提示 6 天、账号锁定 8 次/2 分钟等；后端 effective defaults 是密码长度 8–32、有效期 90 天、提示 7 天、账号锁定 10 次/30 分钟。用户只打开页面并保存就会生成缺失 key 并改变有效策略。

### 4.3 Device

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| eNB Inform | PASS | 当前启用、60 秒；后端 consumer 有对应读取 |
| eNB 离线判断开关 | FAIL | 页面有 `enbTimeoutEnable`，离线扫描器不读取该 key |
| eNB 离线阈值 | FAIL | 页面、seed、当前 DB 均为 100 秒；后端修正后的 fallback 为 600 秒被覆盖 |
| CPE 参数 | PARTIAL | 后端支持 CPE Inform/timeout，当前页面未完整暴露 |
| 名称同步 | PARTIAL | prompt 模式可展示；注释“四选一”与实际三种策略漂移 |
| 自动回收 | PASS/PARTIAL | 配置链和 Go 测试存在；当前关闭、90 天，未做破坏性运行验证 |
| 周期参数同步 | PASS/PARTIAL | 配置链和 Go 测试存在；当前关闭，未触发真实批任务 |

### 4.4 Storage

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| MinIO 公网端点 | PARTIAL | 页面可编辑；当前 category 不存在，页面为空 |
| 告警保留 | PASS/PARTIAL | 配置缺失时后端 fallback 365 天，运行日志确认策略已应用 |
| 保存后应用结果 | FAIL | API 成功只证明 DB 提交；策略 hook 的失败不会返回页面 |

当前环境没有 `storage` category。打开后直接保存可能把前端默认写入库，因此未执行现网保存。

### 4.5 ACS Transfer

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| Host、端口、路径、最大文件、并发 | PARTIAL | 页面完整渲染，相关 Go policy 测试通过；当前 DB 无 category |
| 地址规范化 | PARTIAL | 页面显示 `127.0.0.1`、HTTP 8080、1 GiB、并发 100；保存会把用户输入重写为固定结构，缺少无损 round-trip 契约 |
| 跨进程生效 | PARTIAL | 有通知/轮询机制，但 UI 不显示 applied version |

### 4.6 Agent

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| 配置读取、Token 脱敏 | PASS/PARTIAL | 专用 API 仅返回 configured 状态；当前未配置 |
| 测试连接 | NOT TESTABLE | 没有外部 Agent URL/Token，不能安全构造真实连通测试 |
| 保存与同步 | NOT TESTABLE | 会改变外部连接和运行态，本轮未执行 |
| 阻断路径 | FAIL | 强制 denylist 也是可编辑字段，清空后运行时不再补回 |
| 通用 secret 保护 | FAIL | 通用 sysConfig 接口可绕过 Agent 专用 DTO 读取 raw value |

### 4.7 PM Retention

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| 页面和 DB 值 | PASS | raw 30 天、hourly 180 天、adhoc 730 天、文件 730 天、MR 1825 天 |
| TimescaleDB 策略 | PASS | 所有 12 个 job 最近状态均为 Success；当前 desired 与 applied 一致 |
| PM 清理任务 | PASS | 当日 11:00 执行成功，删除 0 |
| 保存失败可见性 | FAIL | 策略采用 remove 后 add；add 失败时页面仍可能显示保存成功 |

当前表大小：`pm_metrics` 约 3.0 MiB、`pm_metrics_hourly` 约 1.1 MiB、`pm_group_hourly` 48 KiB、`alarms` 80 KiB，未发现当前容量压力。

### 4.8 Retention / Backpressure

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| ACS 磁盘背压 | PASS/PARTIAL | watchdog 已启动，当前阈值 85%/75%、30 秒；主机磁盘 43%，未触发高水位 |
| IO PSI 与 max inflight | FAIL | 后端 seed/模型已有新 key，当前 DB 与页面均未完整覆盖 |
| 原始报文保留 | PASS/PARTIAL | 当前 60 天，压缩开启；未人为制造过期对象 |
| 站点日志保留 | PASS | 当前 60 天、每次 20、延迟 5 秒；当日任务成功 |
| MinIO ILM | PASS | `pm-files`、`mr-files` 均存在启用的 60 天规则 |

MinIO 当前 `pm-files` 约 116 KiB / 9 objects，`mr-files` 为空。功能正在工作，但缺少 applied status、积压量和最近清理统计的页面展示。

### 4.9 Logs

| 功能 | 结果 | 证据与问题 |
|---|---|---|
| DB 日志保留配置 | PASS/PARTIAL | 各类 30–180 天；当日 13:00 清理成功 |
| 文件轮转 | PASS | 50 MiB、1440 分钟、30 天、10 个归档；现场存在约 52.4 MB 归档，watcher 已启动 |
| 总容量保护 | FAIL | 只限制单文件/归档数，没有整个日志目录硬上限 |
| 清理积压可见性 | PARTIAL | 每轮有删除上限，但页面没有 backlog、最老记录或预计清空时间 |
| 日志导出 | FAIL | 真实后端导出未实现，UI 已隐藏假按钮 |

`run/logs` 当前约 256 MiB，其中 app 约 241 MiB、worker 12 MiB、nginx 2.7 MiB、ACS 120 KiB。当前轮转正常，但在多进程、多日志类型扩展后仍需要目录级容量预算。

## 5. API、权限与敏感信息测试

| 场景 | 结果 | 说明 |
|---|---|---|
| 匿名访问 `/api/v1/admin/sysConfig` | PASS | 返回 401，提示缺少 Authorization |
| 匿名访问 public security | PASS/PARTIAL | 当前仅返回 `isBrowserAutoRecordPass=false`，未返回默认密码 |
| 已登录管理员访问配置 | PASS | 九页签均成功加载当前 category |
| 已登录非管理员访问 | NOT TESTABLE / 代码判定 FAIL | 当前只有 `admin`、`system` 用户，无可用 operator；路由中间件仅认证不鉴权 |
| 通用接口 secret 脱敏 | FAIL | response DTO 直接序列化 `Value` |
| 任意设置 public | FAIL | 通用 create/update 请求暴露 `is_public` |

公共端点当前数据未发生 secret 泄露，但“当前库里没有被标 public 的 secret”不等于接口设计安全。

## 6. 真实运行态与容量

### 6.1 数据保留与清理

- PM 清理、站点日志清理、数据库日志清理当天均执行成功，删除数均为 0；
- TimescaleDB 的 alarms、MR、PM raw/hourly/adhoc/trace 等 12 个策略最近执行均成功；
- MinIO 两个桶的 `omc-raw-expire-60d` 规则为 Enabled；
- 应用、Worker、ACS 的日志轮转 watcher 都已启动。

### 6.2 磁盘

| 项目 | 现场值 |
|---|---:|
| 主机 Data 卷 | 460 GiB 总量，182 GiB 已用，248 GiB 可用，43% |
| Docker images | 24.77 GB |
| Docker build cache | 21.31 GB，其中 20.95 GB 可回收 |
| Docker volumes | 2.619 GB，其中 1.365 GB 可回收 |
| `run/logs` | 约 256 MiB |

Docker build cache 是当前最明显的可回收空间，但清理属于环境变更，本轮只记录、不执行。

### 6.3 仪表盘真实性

真实页面显示 CPU 42%、内存 67%、磁盘 58%、设备 189/215、会话 8，并展示固定服务 uptime 和样例操作记录。`frontend-core/src/services/api/systemApi.ts:3-20` 的后端 DTO 并不包含这些指标；`webcode/src/pages/system/SystemDashboard/index.tsx:44-49` 使用固定 fallback。

因此仪表盘属于 FAIL。资源指标未采集时应显示“未采集”，不能显示看起来可信的确定值。

## 7. 关键问题与解决优先级

### P0：立即处理

1. **补服务端 RBAC**：所有 `/admin` 与系统配置接口必须按 read/write/secret-read 分权；增加 operator 返回 403 的集成测试。
2. **封闭 secret/public 组合链**：通用 DTO 不回显 secret；从请求中移除任意 `is_public`；公开项改为代码白名单；敏感读取和写入必须审计。

### P1：下一迭代优先

1. **建立唯一 effective config 契约**：默认值只由后端返回，前端不再自带业务默认；加载失败时禁止保存。
2. **整批 typed validation**：校验最终候选状态、未知 key、范围与跨字段关系。
3. **统一写入口与 applied status**：单条 CRUD 和 batch 使用同一 domain command；返回 `desired/applied/last_error`，hook panic 必须记录。
4. **修复设备配置**：让 `enbTimeoutEnable` 真正控制扫描；迁移 100 秒历史值；统一 seed、UI、fallback 和注释。
5. **固化 Agent 强制 denylist**：代码内不可变，页面只能追加；补编码、大小写、路径规范化测试。
6. **移除仪表盘假数据**：接真实指标前展示 unavailable，并标注采集时间和数据源。
7. **补全背压配置**：页面、DB migration、consumer 同时覆盖 IO PSI high/low 和 max inflight。

### P2：随后治理

1. 实现真实日志导出并修正 smoke test；
2. 增加日志目录总容量、磁盘剩余空间和 Docker cache 告警；
3. 展示 cleanup backlog、最老数据、最近成功/失败、预计清空时间；
4. 修复前端 3 个 lint error、ICU 文案错误、过期 UFTE 测试和并发超时测试；
5. 为 transfer 表单增加无损 round-trip 测试；
6. 给 OMC 名称、CPE 配置和名称同步补完整业务契约。

## 8. 必须补充的测试

当前 SystemConfig 只有 6 个序列化测试和 5 个安全行为 mock E2E，缺少完整页面/后端契约。优先补：

1. 空库加载后不保存；保存时 effective policy 不改变；
2. GET 失败、部分 category 缺失时保存按钮禁用；
3. viewer/operator 读取和写入配置均为 403；
4. default password、Agent token 永不出现在通用响应；
5. 非白名单 key 不能设为 public；
6. 安全策略 `min <= max`、提示期小于有效期、默认密码满足新策略；
7. `enbTimeoutEnable=false` 时扫描不下线设备；
8. 保存 PM/MinIO 配置后读取 applied status，模拟外部策略应用失败；
9. 九页签真实后端加载、修改、保存、重载、运行态一致性；
10. 仪表盘数值与指标 API 一致，API 缺失时显示“未采集”。

安全 mock E2E 还应阻断所有未 mock 请求；本轮运行时出现默认后端 `ECONNREFUSED`，但相关请求不在现有断言中，测试仍可通过，容易掩盖集成断链。

## 9. 测试限制

- 没有现成非管理员用户，因此没有通过真实账号执行 403；代码链已能确定当前中间件只验证登录；
- Agent 未配置，未调用外部服务；
- 没有点击任何保存按钮，避免把前端默认写入现网或重建保留策略；
- 没有制造过期 PM、日志和 MinIO 对象，只验证策略、任务状态和当前执行记录；
- 运行容器未重建，运行态镜像不能证明与当前源码 HEAD 字节级一致；源码审查、测试与运行态证据分别记录。

## 10. 复现命令

```bash
cd omcgo
go test ./...

cd ../omcmb
npm run typecheck
npm run lint
npm run test --workspace webcode

npx eslint \
  webcode/src/pages/system/SystemConfig \
  frontend-core/src/services/api/adminApi.ts \
  frontend-core/src/hooks/api/useSystem.ts \
  --quiet

cd webcode
npx playwright test e2e/security-policy.spec.ts --workers=1
SMOKE_API_TARGET=http://localhost:18081 \
  npx playwright test --config playwright.smoke.config.ts e2e/smoke/system.spec.ts
```

本报告应与同目录的完整代码审查 `REVIEW_cccbb2b3_system-config-code-review.md` 配合阅读：本文回答“实际测到了什么”，代码审查回答“为什么会这样以及调用链风险在哪里”。
