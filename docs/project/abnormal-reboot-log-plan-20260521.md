# 异常重启记录功能开发计划

> 日期：2026-05-21
> 作者：Claude（与用户讨论确定）
> 范围：F06 设备管理 + F01 ACS 协议层 + F04 告警链路（已有）联动
> 关联规范：`docs/files/Back-end/设备异常重启日志功能流程规范文档.md`（v1.2）

---

## 1. 背景

设备发生异常重启（死机崩溃、看门狗复位、断电恢复等）后，会在重新上线时通过 TR-069 Inform 上报「1 BOOT」事件，并在参数列表中携带：

| 标准路径 | 含义 |
|---------|------|
| `Device.HaltReason.MainReason` | 故障主原因（不为空 = 异常重启） |
| `Device.HaltReason.DetailReason` | 故障详细原因 |

现状：
- 前端 `webcode/src/pages/log/ExceptionLog/` 页面已存在，但使用 Mock 数据；菜单挂在「日志管理」下。
- 后端事件总线已定义 `device.reboot.abnormal` 主题，`alarm/reboot_monitor.go` 订阅它做「频繁异常重启告警」。
- 后端 `station_fault_logs` 表已建（migration 000126），但**仅在 CPE 上传完文件后才写入记录**——不满足「1 BOOT 携带 HaltReason 即落库」的需求。
- 异常重启的判断当前在 `device/device_service.go:1045`，规则是 `"1 BOOT" && !"M Reboot"`（事件码组合），**与用户期望的 HaltReason 不为空规则不一致**——可能漏判（CPE 没下发 M Reboot 但实际是受控重启）或误判。

本计划要解决：识别规则切换 + 识别即落库 + UI 接入真实接口 + 菜单归位。

---

## 2. 目标与范围

### 2.1 本期范围（MVP）

| # | 目标 | 验收 |
|---|------|------|
| G1 | 设备 1 BOOT 且 `Device.HaltReason.MainReason` 不为空时，**识别即落库**一条异常重启记录（不依赖文件上传） | 后端 E2E：CPE 模拟器发 1 BOOT + HaltReason 参数 → `station_fault_logs` 新增一行 |
| G2 | 前端「设备异常日志」菜单移到「设备管理」下，重命名为「异常重启记录」 | 侧边栏 / 路由 / 权限码全部更新 |
| G3 | 前端列表从 Mock 切到真实后端 API，列字段与规范对齐（含死机原因、运行时长、收集状态） | 浏览器实测：触发 1 BOOT 后列表自动出现新行 |
| G4 | 后端提供 List / Detail / Delete 三个端点（手动收集留下期） | E2E 脚本断言覆盖 |

### 2.2 显式不在本期（务实简化，避免范围飘移）

- **手动收集 / 自动文件下发**（规范 §6.4 / §7.2）：MVP 只做「识别+落库」，文件上传链路已经存在（`stationlog` 服务），但「下发 SetParameterValues 让设备上传」的指令编排留到下期。
- **5G 软重启（"4 VALUE_CHANGE"+soft_reboot）**：规范 §6.2.3，下期再做（事件路由分支不同）。
- **收集模式切换（自动/受控）**：规范 §3 的 `errorLogCollectionMode` 配置。本期默认走「识别即落库」，不引入模式切换 UI。
- **磁盘空间保护 / 单设备文件保留上限 / 全局文件总数限制**：规范 §9。`stationlog.FaultLogMaxCount` 已有全局 20 条配额，本期不动；其他清理策略留下期。
- **多平台/多产品类型私有路径映射**（规范 §4.2 / §4.3）：本期只用标准路径 `Device.HaltReason.MainReason` / `DetailReason`。私有路径（如 `Device.LogMgmt.FaultLogURL`）的翻译走现有 ParamModel Translator（T-0098 P2-02），不在本计划新增。
- **导出 CSV / 批量下载**（规范 §7.4 / §7.5）：UI 留按钮但 disabled，下期补。

### 2.3 下一期路线（任务速览）

T-0158 闭环后剩余 5 项独立可发布的能力。每项详细规范见 §11；这里仅速览：

| 候选 ID | 标题 | 规模 | 依赖 | 风险 |
|---------|------|------|------|------|
| T-0159 | 手动收集（受控模式触发设备上传） | M | task / transfer 模块；本期 DB 已备字段 `manual_collection_status` | Redis 超时检测需要稳定的 worker 调度 |
| T-0160 | 5G 软重启识别（"4 VALUE_CHANGE" + `soft_reboot`） | S | 本期 `RecordAbnormalReboot` 接口已通用 | 仅 5G NR 平台，需 CPE 模拟器扩展 |
| T-0161 | 收集模式开关（自动 / 受控） | S | sys_configs 已支持；前端需新增 UI | 模式切换影响所有 1 BOOT 路径 |
| T-0162 | 文件配额 + 磁盘保护清理任务 | M | worker 进程；现有 `enforceFaultLogQuota` 是起点 | 磁盘检查方式跨平台兼容性 |
| T-0163 | CSV 导出 + ZIP 打包下载 | S | 现有 ListAbnormalReboots + MinIO 预签名 | 大数据量分页流式输出 |

完整接入点、验收标准、预埋字段对照见 §11。

---

## 3. 关键设计决策

### D1 异常重启判断规则切换

**当前**（`device_service.go:1045`）：
```go
abnormal := hasEventCode(events, tr069.EventBoot) && !hasEventCode(events, tr069.EventMReboot)
```

**改为**：
```go
abnormal := hasEventCode(events, tr069.EventBoot) && haltMainReason != ""
```

其中 `haltMainReason` 从 Inform 的 `ParameterList` 中按标准路径 `Device.HaltReason.MainReason` 提取（不命中视为空字符串）。

**影响面**：
- `RecordBootFromInform` 签名扩展 — 增加 `params []tr069.ParameterValueStruct` 入参。
- `alarm/reboot_monitor.go` 订阅的 payload 需要补 `halt_main_reason` / `halt_detail_reason` 字段。频繁重启告警的累计逻辑不变（仍按异常重启次数计数）。
- 旧测试用例 `RecordBootFromInform` 需要更新——之前用纯事件码组合的预期，现在要带 HaltReason 参数才算异常。

**回退保险**：保留旧规则作为 fallback，但仅在 ParameterList 完全为空（Inform 异常）时启用，避免新逻辑在边界情况下漏判。

### D2 数据库改造

`station_fault_logs` 表既要继续承载「文件上传后的完整记录」，又要承载「识别即落库的占位记录」。两条路径写同一张表，靠字段可空 + 状态字段区分。

**migration 000150（DDL）变更点**：

```sql
-- 1. 三个文件相关字段从 NOT NULL 改为 NULLABLE（识别时还没有文件）
ALTER TABLE station_fault_logs
    ALTER COLUMN file_name   DROP NOT NULL,
    ALTER COLUMN object_path DROP NOT NULL,
    ALTER COLUMN bucket      DROP NOT NULL;

-- 2. 新增字段
ALTER TABLE station_fault_logs
    ADD COLUMN IF NOT EXISTS device_name             TEXT,
    ADD COLUMN IF NOT EXISTS device_type             TEXT,        -- eNB / gNB
    ADD COLUMN IF NOT EXISTS is_gnb                  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS operate_ip              TEXT,
    ADD COLUMN IF NOT EXISTS software_version        TEXT,
    ADD COLUMN IF NOT EXISTS runtime_before_reboot   BIGINT,      -- 秒数；前端按小时/分钟格式化
    ADD COLUMN IF NOT EXISTS record_status           TEXT NOT NULL DEFAULT 'detected'
        CHECK (record_status IN ('detected', 'file_received', 'collection_failed')),
    ADD COLUMN IF NOT EXISTS collection_fail_reason  TEXT,
    ADD COLUMN IF NOT EXISTS manual_collection_status TEXT NOT NULL DEFAULT '0'
        CHECK (manual_collection_status IN ('0', '1', '2'));        -- 0 未收集/已完成, 1 收集中, 2 失败

-- 3. 索引
CREATE INDEX IF NOT EXISTS idx_station_fault_logs_status ON station_fault_logs(record_status, collected_at DESC);
```

**Down 段**：DROP 上述新列；将三个 NULLABLE 列改回 NOT NULL（DROP NOT NULL 的反向）。

**记录生命周期**：
- 识别即落库 → `record_status = 'detected'`，文件字段空。
- 后续文件上传 → 更新为 `record_status = 'file_received'`，填入 file_name/object_path/bucket。
- 收集失败 → `record_status = 'collection_failed'`，填 `collection_fail_reason`。
- 现有 `is_deleted` 字段语义不变（文件已删，记录保留）。

### D3 「识别即落库」的落点

不复用 `stationlog.Service.HandleLogFileReceived`——那是文件流程。新增独立路径：

```
ACS handler → device.inform_handler → DeviceService.RecordBootFromInform
                                          │
                                          ├─ 已有：publish device.reboot.abnormal（payload 补 HaltReason）
                                          └─ 新增：直接调 stationlog.Service.RecordAbnormalReboot(ctx, snapshot)
                                                  └─ INSERT station_fault_logs (record_status='detected', ...)
```

**为什么选择 inline 调用而非走事件**：
- `device.reboot.abnormal` 事件已经被 `alarm/reboot_monitor.go` 订阅做告警累计，引入第二个订阅者会让"是否成功落库"的语义变得隐式（异步失败难追踪）。
- 落库需要**在原事务链路上完成**，便于在 Inform 响应前完成数据库写入（保证「设备一上线立即可见」）。
- 如果将来要解耦，再把这一步抽到 EventBus 也容易（先内联，后抽离，符合 YAGNI）。

### D4 菜单迁移

**静态侧（`webcode/src/components/Layout/Sidebar/navConfig.ts`）**：
- 从 `log` 子菜单移除 `log-exception`。
- 在 `device` 子菜单（当前含「设备列表 / 设备分组 / 插拔记录 / 回收站」）增加 `device-abnormal-reboot`，path 维持 `/log/exception` 不变（避免改路由表）或改为 `/device/abnormal-reboot`（更语义化，但需同步路由表）。
  - **决策**：改为 `/device/abnormal-reboot`，新路由更语义化，老路由保留 redirect 兜底浏览器书签。

**动态侧（数据库 `menus` 表）**：
- 现有种子 ID：`aaaa0007-1000-0000-0000-000000000002`（在「日志管理 aaaa0007-0000-0000-0000-000000000001」下）。
- 新 seed migration 000139：
  ```sql
  UPDATE menus
     SET parent_id   = '<device-management 菜单 ID>',
         name        = '异常重启记录',
         route_path  = '/device/abnormal-reboot',
         permission_code = 'device:abnormal-reboot',
         sort_order  = 50   -- 排在设备列表之后
   WHERE id = 'aaaa0007-1000-0000-0000-000000000002';

  UPDATE menus
     SET permission_code = 'device:abnormal-reboot:query'
   WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND permission_code = 'log:exception:query';
  -- 同理批改 :add / :edit / :delete
  ```
- 同步更新 `name_i18n` JSONB（zh-CN: "异常重启记录" / en-US: "Abnormal Reboot Records"）。

### D5 API 端点

复用 `stationlog` handler 里的现有日志端点但**新增专用前缀**避免和"运行日志"混用：

| 方法 | 路径 | 用途 | 备注 |
|------|------|------|------|
| GET | `/api/v1/device-abnormal-reboots` | 列表分页 + 过滤 | 过滤：device_sn / device_type / record_status / time range |
| GET | `/api/v1/device-abnormal-reboots/:id` | 详情 | 含完整 HaltReason |
| DELETE | `/api/v1/device-abnormal-reboots/:id` | 删除（软删 is_deleted=true，物理文件走 MinIO 清理） | 批量删除走 `?ids=a,b,c` |
| GET | `/api/v1/device-abnormal-reboots/:id/download` | 单文件下载预签名 URL | 仅 `record_status='file_received'` 可用 |

**字段映射**（后端 snake_case → 前端 camelCase，走现有 Axios 拦截器）：

| DB 列 | API 字段 |
|-------|---------|
| id | id |
| device_id, device_sn | deviceId, deviceSn |
| device_name, device_type, is_gnb, operate_ip, software_version | deviceName, deviceType, isGnb, operateIp, softwareVersion |
| fault_reason | haltMainReason |
| fault_detail | haltDetailReason |
| runtime_before_reboot | runtimeBeforeReboot（秒） |
| record_status | recordStatus |
| manual_collection_status | manualCollectionStatus |
| collection_fail_reason | collectionFailReason |
| file_name, file_size, is_deleted | fileName, fileSize, isFileDeleted |
| collected_at | collectedAt |

### D6 前端改造范围

| 文件 | 改动 |
|------|------|
| `omcmb/frontend-core/src/services/api/deviceAbnormalRebootApi.ts` | **新增**——独立 API 模块（不复用 stationLogApi，避免运行日志/异常重启耦合） |
| `omcmb/frontend-core/src/hooks/api/useDeviceAbnormalReboot.ts` | **新增** |
| `omcmb/frontend-core/src/types/deviceAbnormalReboot.ts` | **新增**（含 BackendXxx → Xxx 映射函数） |
| `omcmb/frontend-core/src/mock/deviceAbnormalReboot.ts` | **新增**（保留 Mock，让 `VITE_USE_MOCK=true` 可独立联调） |
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` + `en-US/index.ts` | 替换/新增词条：`nav.device.abnormalReboot` 等 |
| `omcmb/webcode/src/pages/log/ExceptionLog/index.tsx` | 文件**整体移到** `omcmb/webcode/src/pages/device/AbnormalReboot/index.tsx`，移除 Mock 内联数据，改调 useDeviceAbnormalReboot |
| `omcmb/webcode/src/router/index.tsx`（或路由表文件） | 新增 `/device/abnormal-reboot` 路由 + 旧 `/log/exception` redirect |
| `omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts` | 删除 log 子项 + 新增 device 子项 |

**三皮肤影响**（`webcode-v2/`、`webcode-v3/`）：
- 业务层（API/Hook/Type/Mock/i18n）改动后，需在两个候选皮肤跑 `npm run typecheck` 确认不破。
- 这两个皮肤如果还没引用 ExceptionLog 页面（大概率没有），typecheck 即可；如果引用，需要同步迁移。

---

## 4. 任务拆分

> 任务编号沿用现有体系（最新到 T-0157），从 T-0158 起。建议作为 1 个 umbrella + 5 个 sub-task，进 backlog `§5 Triaged`，待 sprint planning 升 planned。

### 4.1 Umbrella

**T-0158 — 异常重启记录功能（识别即落库 + UI 归位）**
- 来源：本计划
- 规模：M（4-6 工作日）
- 依赖：无外部依赖；内部依赖 `stationlog` / `device` / `alarm/reboot_monitor` 三模块已有的接口稳定性
- 风险：
  - R1（中）异常重启判断规则切换可能影响现有「频繁重启告警」累计——需在 sub-1 完成后用 CPE 模拟器双向验证（旧规则 → 新规则覆盖、新规则下漏判检查）
  - R2（低）`station_fault_logs` 表 ALTER NOT NULL 在 Live DB 已有数据时安全（DROP NOT NULL 不锁表，新列均带默认值）
  - R3（低）菜单 reparent 期间需保证两个 seed migration 同号不冲突（最新 seed 是 000138，新增 000139 / 000140）

### 4.2 Sub-tasks（按依赖顺序）

| ID | 标题 | 规模 | 内容 | 接入点 |
|----|------|------|------|--------|
| T-0158-a | DB schema + stationlog 仓库扩展 | S | 1) 新建 `migrations/000150_station_fault_logs_detected_records.sql`（DDL，按 D2）；2) 新建 `migrations/seed/000139_menu_move_abnormal_reboot.sql`（菜单迁移 + 权限码更新）；3) 扩展 `stationlog/model.go` 的 `FaultLog` 结构体加新字段；4) 扩展 `stationlog/pg_repository.go` 的 `faultLogCols` + 新增 `Insert/Update` 方法；5) `goose up→down→up` 三次演练 | `omcgo/migrations/000150_*.sql`、`omcgo/internal/stationlog/{model,pg_repository}.go` |
| T-0158-b | 异常重启识别规则切换 | S | 1) `RecordBootFromInform` 签名加 `params []tr069.ParameterValueStruct`；2) 提取 `Device.HaltReason.MainReason` / `DetailReason`；3) 判断规则改 D1；4) `device.reboot.abnormal` payload 补 `halt_main_reason` / `halt_detail_reason` / `runtime_before_reboot` 三字段；5) 调用方（`inform_handler.go`）同步传参；6) 现有单测更新（覆盖：HaltReason 空=正常、不空=异常、ParameterList nil=fallback 走旧规则）；7) `alarm/reboot_monitor.go` 测试不变（累计逻辑不依赖 HaltReason 字段） | `omcgo/internal/device/device_service.go`、`omcgo/internal/device/inform_handler.go`、`omcgo/internal/alarm/reboot_monitor.go`（仅 payload 字段映射） |
| T-0158-c | 识别即落库（service 层） | S | 1) `stationlog/service.go` 新增 `RecordAbnormalReboot(ctx, snapshot)` 方法（snapshot 含 device 快照 + HaltReason）；2) 写一行 record_status='detected'；3) 不触发文件相关逻辑（保持 file_name 为空）；4) 单测覆盖：HaltReason 完整 / DetailReason 缺失 / 同设备 1 秒内连续上报（按 collected_at 区分） | `omcgo/internal/stationlog/service.go`（新方法）+ `device_service.go`（inline 调用） |
| T-0158-d | API 端点 + handler | S | 1) `stationlog/handler.go` 新增 4 个端点（按 D5）；2) 路由注册（`cmd/app/router/router.go`）；3) 列表过滤 squirrel where 条件；4) 单测 + E2E 用例（在 `scripts/e2e_verify.sh` 加 device-abnormal-reboot 域，覆盖 list/detail/delete 三个端点） | `omcgo/internal/stationlog/handler.go`、`omcgo/cmd/app/router/router.go`、`omcgo/scripts/e2e_verify.sh` |
| T-0158-e | 前端业务层 + 页面迁移 | M | 1) 新建 `deviceAbnormalRebootApi.ts` + `useDeviceAbnormalReboot.ts` + `types/deviceAbnormalReboot.ts` + `mock/deviceAbnormalReboot.ts`；2) 移动页面到 `webcode/src/pages/device/AbnormalReboot/`；3) 路由表 + navConfig 更新；4) i18n 中英对称；5) 三皮肤 `npm run typecheck`；6) 旧路径 `/log/exception` redirect；7) 浏览器实测（dev 启动 + 模拟器触发 1 BOOT + HaltReason → 列表自动出新行） | 见 D6 表格 |

---

## 5. 数据流与时序

### 5.1 识别即落库（本期实现）

```
CPE  ──1 BOOT + HaltReason──▶  ACS Handler
                                     │
                                     │ publish SubjectDeviceRebootComplete
                                     ▼
                              InformHandler.handleRebootComplete
                                     │
                                     │ DeviceService.RecordBootFromInform(events, params)
                                     ▼
                              extract HaltReason.MainReason / DetailReason
                                     │
                                     ├─ MainReason 空 → 正常重启，仅 bootCount++ 返回
                                     │
                                     └─ MainReason 非空 → 异常
                                         │
                                         ├─ stationlog.RecordAbnormalReboot ──▶ INSERT station_fault_logs
                                         │   (record_status='detected', file_name=NULL)
                                         │
                                         └─ publish device.reboot.abnormal
                                             │
                                             └─ alarm/reboot_monitor 累计 → 超阈值发告警
```

### 5.2 文件上传补全（已有逻辑，本期不动）

```
CPE  ──上传日志文件──▶  MinIO ──事件──▶ stationlog.HandleLogFileReceived
                                              │
                                              │ 按 device_sn 找最近的 record_status='detected' 记录
                                              ├─ 找到 → UPDATE 该记录（file_name/object_path/bucket，status→'file_received'）
                                              └─ 找不到 → 走旧路径 INSERT 新记录（防漏）
```

> **设计要点**：本期不动 `HandleLogFileReceived`，但 sub-task d 完成后需要在 sub-task e 联调阶段补一个 TODO 注释，提示「文件到来时优先更新 detected 记录」——下期 T-NNN-a 实现。

---

## 6. 测试策略

### 6.1 单元测试

- `device_service_test.go`：覆盖 RecordBootFromInform 新签名（4 个 case：HaltReason 完整 / Main 空 Detail 有 / Main 有 Detail 空 / ParameterList nil fallback）。
- `stationlog/service_test.go`：覆盖 RecordAbnormalReboot（含 snapshot 字段完整性）。
- `stationlog/pg_repository_test.go`：覆盖新字段 round-trip 读写。

### 6.2 E2E

`scripts/e2e_verify.sh` 新增 `device-abnormal-reboot` 域，约 6-8 个 `check_status` 断言：
1. POST 模拟 1 BOOT + HaltReason → 200
2. GET list → 包含新记录
3. GET detail → 字段完整
4. 重复 POST → 列表第二条出现
5. DELETE → 软删
6. GET list 过滤 record_status='detected' → 命中
7. GET 不存在的 ID → 404
8. DELETE 不存在的 ID → 404

### 6.3 CPE 模拟器

`scripts/cpe_simulator.py` 已经支持发 Inform 携带任意参数列表。本计划无需改模拟器，只需在 sub-task b 完成后写一个 shell 调用样例（带 `Device.HaltReason.MainReason=halt_reboot` + `Device.HaltReason.DetailReason=watchdog_timeout`）放进 sub-task d 的 E2E 流程。

### 6.4 前端

- `npm run typecheck` 三皮肤全过。
- Vitest 单元测试覆盖 `mapBackendDeviceAbnormalReboot` 函数。
- Playwright E2E：dev 启动 + 模拟器触发 → 页面列表行数 +1（可选，时间允许再做）。

---

## 7. 验收清单（DoD）

- [ ] DB migration 000150 / seed 000139 均 `goose up→down→up` 演练通过
- [ ] `go build ./...` + `go test -race ./...` 全过
- [ ] `golangci-lint run` 净 0
- [ ] E2E 脚本新增断言全部 PASS，且现有断言数不下降
- [ ] 前端 `npm run typecheck` 在 webcode / webcode-v2 / webcode-v3 三个包全过
- [ ] 前端 `npm run lint` 净 0
- [ ] 浏览器实测：dev 启动 → CPE 模拟器触发异常 1 BOOT → 在「设备管理 → 异常重启记录」页面看到新记录，HaltReason 字段非空
- [ ] 旧路径 `/log/exception` 仍可访问（redirect 到新路径）
- [ ] 菜单种子在已有数据 DB 上运行后，原「日志管理」下没有「设备异常日志」项，新「设备管理」下有「异常重启记录」项
- [ ] commit footer 五元组完整（PRD 链接本计划 / Sprint / Risk / Backlog T-0158）
- [ ] 本计划文档 §2.3「下一期路线」记入 backlog `§5 Proposed`

---

## 8. 风险与依赖

| ID | 风险 | 等级 | 缓解 |
|----|------|------|------|
| R1 | 判断规则切换导致频繁重启告警漏发或误发 | 中 | sub-b 完成后用 CPE 模拟器双向回归（10 个用例：5 异常 + 5 正常） |
| R2 | `station_fault_logs` 表 ALTER 在 Live DB 大表上耗时 | 低 | 表当前数据量小（功能未上线），DROP NOT NULL 是元数据操作不锁行；ADD COLUMN 都带默认值或 NULL，不重写 |
| R3 | 菜单 seed 在已部署环境上运行失败（权限码已被某角色绑定） | 中 | seed 用 `ON CONFLICT DO NOTHING` + `WHERE permission_code = old` 的精准 UPDATE，不批量重写；role_permissions 关联表保持 ID 不变（菜单 ID 不变，仅父子关系变） |
| R4 | 前端三皮肤改动遗漏 webcode-v2 / v3 | 低 | DoD 强制 typecheck 全包跑过 |
| R5 | 私有路径（部分 CPE 不上报标准 `Device.HaltReason.MainReason`）导致漏判 | 中 | 本期接受漏判风险，下期通过 ParamModel Translator 接入私有路径双向翻译 |

---

## 9. 时间盒预估

| 阶段 | 工作量 |
|------|-------|
| T-0158-a DB + 仓库 | 0.5 d |
| T-0158-b 判断规则切换 | 0.5 d |
| T-0158-c 识别即落库 | 0.5 d |
| T-0158-d API + E2E | 1 d |
| T-0158-e 前端业务层 + 页面迁移 + 浏览器实测 | 1.5 d |
| 联调 + DoD 闭环 | 0.5 d |
| **合计** | **4.5 d**（含 0.5 d buffer 应对边界情况） |

---

## 10. 后续衔接

本计划闭环后，下期任务（T-0159..T-0163，详见 §11）建议按以下顺序推进：

1. **T-0159 手动收集** — 解锁规范 §7.2 全流程，让 UI「开始收集」按钮可用
2. **T-0160 5G 软重启识别** — 解锁规范 §6.2.3
3. **T-0162 文件配额清理任务 + 磁盘保护**（独立部署 worker，不阻塞 P1 流程）
4. **T-0161 收集模式 UI**（自动/受控切换）— 需要 T-0159 已完成
5. **T-0163 CSV 导出 + ZIP 打包下载** — 用户体验项，可与上述并行

每一项均独立可发布，不强耦合。

---

## 11. 下一期任务详细规范

> 任务编号沿用 T-0158 后续，按 §2.3 速览表排序。每项给出：背景、本期已预埋的挂载点、改动范围、验收标准、风险。
> 本期 T-0158 完成时已经按"为下期留好接口"的原则在数据库 / 后端 / 前端预埋了字段、状态枚举、UI 按钮位，便于下期最小改动接入。

### 11.1 T-0159 — 手动收集（受控模式触发设备上传）

**背景**：规范 §6.4.1（受控模式）+ §7.2（手动收集流程）。受控模式下，识别到异常重启后不自动下发收集指令；用户在 UI 上对某条 detected / collection_failed 记录点击「开始收集」按钮，触发后端向设备下发 SetParameterValues，URL 含记录 ID；超时仍未上传则标记为失败。

**本期已预埋**：

| 预埋点 | 位置 | 用途 |
|--------|------|------|
| `station_fault_logs.manual_collection_status` | migration 000150 | '0'/'1'/'2' 三态字段已就位，下期直接 UPDATE |
| `ManualCollectionIdle / Running / Failed` 常量 | `stationlog/model.go` | service 层直接复用 |
| 前端「开始收集」按钮位 | `pages/device/AbnormalReboot/index.tsx` 详情抽屉 | 当前按 recordStatus 切显示，下期接 mutation hook |
| `collection_fail_reason` 列 | migration 000150 | 失败原因落库字段已备 |
| Redis 命令队列 + task 模块 | `internal/task/` | 任务下发 / 超时检测的基础设施现成 |

**改动范围**：

| 层 | 文件 | 主要内容 |
|----|------|---------|
| DB | seed (新版本号) | 无 DDL 改动；可选 `errorLogCollectionMode` 系统配置项默认值 |
| service | `internal/stationlog/service.go` | 新方法 `TriggerManualCollect(ctx, id)`：UPDATE manual_collection_status='1' + 推 task 队列下发 SPV + 设置 Redis 超时 key |
| service | `internal/stationlog/timeout_worker.go` | 新文件：后台 goroutine 消费超时检测队列，超时则 UPDATE status='2' + collection_fail_reason |
| handler | `internal/stationlog/handler.go` | 新端点 `POST /device-abnormal-reboots/:id/collect` |
| 前端 | `frontend-core/src/services/api/deviceAbnormalRebootApi.ts` | 加 `triggerCollect(id)` 方法 |
| 前端 | `frontend-core/src/hooks/api/useDeviceAbnormalReboot.ts` | 加 `useTriggerCollect` mutation hook |
| 前端 | `pages/device/AbnormalReboot/index.tsx` | 列操作菜单加「开始收集 / 重新收集」按钮，按状态 + 设备在线状态切换 disabled |

**验收标准**：
- [ ] 点击「开始收集」→ DB 中该行 `manual_collection_status` 变为 '1'，向设备下发 SPV 任务可见于 task 队列
- [ ] CPE 模拟器在超时窗口内上传文件 → 记录推进到 file_received，UI 状态变 success
- [ ] 不上传文件等待超时 → 记录状态变 '2'，`collection_fail_reason` 含「超时」字样
- [ ] 设备离线时点击按钮 → 前端阻断，不打到后端
- [ ] 同一记录连续点两次 → 第二次返回 409，不重复下发

**风险**：
- R1（中）超时 worker 在多实例部署下需要分布式协调（建议用 NATS JetStream durable consumer 而非 Redis polling）
- R2（低）规范 §3 中提到的 `manualErrLogtimeout` 配置可后置，先用硬编码 300 秒

**规模**：M（2-3 工作日）

---

### 11.2 T-0160 — 5G 软重启识别（"4 VALUE_CHANGE" + soft_reboot）

**背景**：规范 §6.2.3。5G 设备子模块重启不触发完整设备重启，CPE 在 `Device.HaltReason.MainReason = "soft_reboot"` 的同时上报 `"4 VALUE_CHANGE"` 事件（而非 `"1 BOOT"`），需要单独识别。

**本期已预埋**：

| 预埋点 | 位置 | 用途 |
|--------|------|------|
| `stationlog.RecordAbnormalReboot` 接口 | `internal/stationlog/service.go` | 与硬重启共用入口，snapshot 字段已含 `IsGNB` |
| `is_gnb` 字段 | migration 000150 | 5G/4G 区分已落库 |
| `inform_handler.handlePeriodic` | `internal/device/inform_handler.go` | "4 VALUE_CHANGE" 事件已在订阅，下期分支即可 |

**改动范围**：

| 层 | 文件 | 主要内容 |
|----|------|---------|
| device | `internal/device/inform_handler.go` | `handlePeriodic` / `handleValueChange` 增加：若 `IsGNB && HaltReason.MainReason == "soft_reboot"` → 调用 DeviceService 的新方法 |
| device | `internal/device/device_service.go` | 新增 `RecordSoftRebootFromInform(ctx, device, params)`，与硬重启共用 `abnormalRecorder` + 发布 `device.reboot.abnormal` 事件，但 payload 额外标记 `reboot_type: "soft"` |
| alarm | `internal/alarm/reboot_monitor.go` | 滑动窗口计数器是否区分软/硬：建议合并计数，超阈值告警时附带类型 |
| 前端 | `pages/device/AbnormalReboot/index.tsx` | 列表加「重启类型」列（硬/软），从 `haltMainReason` 推导 |
| 测试 | `internal/device/inform_handler_test.go` | 加 `TestHandlePeriodic_SoftReboot_5G` 用例 |

**验收标准**：
- [ ] CPE 模拟器发 "4 VALUE_CHANGE" + `Device.HaltReason.MainReason=soft_reboot` → `station_fault_logs` 新增一行，`is_gnb=true`
- [ ] 同一会话发 "1 BOOT" + `halt_reboot`（硬重启）→ 另外一行，互不影响
- [ ] 非 5G 设备发 "4 VALUE_CHANGE" + `soft_reboot` → 不落库（按规范仅 5G 适用）

**风险**：
- R1（低）"4 VALUE_CHANGE" 事件在心跳路径中频繁触发，需保证只在 HaltReason 非空时才走异常分支，否则浪费 DB 写入

**规模**：S（0.5-1 工作日）

---

### 11.3 T-0161 — 收集模式开关（自动 / 受控）

**背景**：规范 §3 的 `errorLogCollectionMode`（0=自动 / 1=受控）。受控模式下识别即落库但不自动下发收集指令，靠用户手动触发（依赖 T-0159）。

**本期已预埋**：

| 预埋点 | 位置 | 用途 |
|--------|------|------|
| `sys_configs` 表 | migration 000001+ | 全局配置项已有，下期 INSERT 一条 `errorLogCollectionMode` 即可 |
| 「识别即落库」逻辑 | `device_service.go:RecordBootFromInform` | 当前无视模式，永远落库；下期在落库后按模式决定是否走 T-0159 的下发分支 |

**改动范围**：

| 层 | 文件 | 主要内容 |
|----|------|---------|
| DB | seed (新版本号) | INSERT `sys_configs (key='errorLogCollectionMode', value='0', category='log')` |
| device | `internal/device/device_service.go` | RecordBootFromInform 末尾按 mode 决定是否触发 T-0159 的下发 |
| 前端 | `frontend-core/src/services/api/sysConfigApi.ts` | 复用现有 sys_configs 读写接口 |
| 前端 | `pages/system/SystemConfig` 或 `pages/device/AbnormalReboot/index.tsx` 顶部 | 加切换开关 UI |

**验收标准**：
- [ ] 自动模式下：1 BOOT + HaltReason → DB 入库 + 自动下发 SPV，模拟器上传文件后记录变 file_received
- [ ] 受控模式下：1 BOOT + HaltReason → DB 入库（detected），**不**下发 SPV；UI「开始收集」按钮可见
- [ ] 切换模式即时生效（无需重启 app 进程）

**风险**：
- R1（低）模式切换需要 EventBus 广播让所有 app 实例感知，可复用现有 sys_configs 的 reload 机制（如有）

**规模**：S（1 工作日，依赖 T-0159）

---

### 11.4 T-0162 — 文件配额 + 磁盘保护清理任务

**背景**：规范 §9。三级保护：单设备 N 条上限（`RebootLogSaveCount` 默认 2）、全局 M 条上限（`errorLogMaxNum`）、磁盘使用率 ≥ 90% 时拒绝存储。

**本期已预埋**：

| 预埋点 | 位置 | 用途 |
|--------|------|------|
| 全局配额 `FaultLogMaxCount=20` + `enforceFaultLogQuota` | `stationlog/service.go` | 已实现单一全局上限清理 |
| `ListOldest` 仅过滤 file_received | `stationlog/pg_repository.go` | 已确保 detected 占位不参与清理 |
| `is_deleted` 字段 | migration 000126 | 文件已删但记录保留的语义已支持 |

**改动范围**：

| 层 | 文件 | 主要内容 |
|----|------|---------|
| DB | seed | INSERT sys_configs: `RebootLogSaveCount=2` / `errorLogMaxNum=可配置` / `faultLogDiskThreshold=90` |
| service | `internal/stationlog/service.go` | `enforceFaultLogQuota` 拆分为 `enforcePerDeviceQuota(ctx, sn)` + 现有全局 + 新增 `checkDiskSpace(path)` |
| worker | `cmd/worker/main.go` | 注册新 cron job：每小时跑一次三级清理 |
| 配置 | `appconfig` | 新增 `FaultLogPerDeviceMax` / `FaultLogDiskThresholdPct` 字段 |

**验收标准**：
- [ ] 单设备写 3 条 file_received → 最早一条 `is_deleted=true`，MinIO 对象被删
- [ ] 全局写 21 条 → 最早一条被清理
- [ ] 模拟 90% 磁盘占用 → 新文件落 MinIO 时记录 `collection_fail_reason="disk full"`，不存文件
- [ ] cron job 启动期跑一次回填清理

**风险**：
- R1（中）跨平台磁盘使用率检测：建议用 `golang.org/x/sys/unix` 的 `Statfs`，Windows 容器场景留 nil-safe fallback
- R2（中）批量清理可能锁表，配额超额时分批 100 条一组处理

**规模**：M（2 工作日）

---

### 11.5 T-0163 — CSV 导出 + ZIP 打包下载

**背景**：规范 §7.4（多文件 ZIP 下载）+ §7.5（CSV 导出列表）。本期 UI 留了「导出」按钮但点击只弹 toast，需要接后端。

**本期已预埋**：

| 预埋点 | 位置 | 用途 |
|--------|------|------|
| `ListAbnormalReboots` 端点 | `stationlog/handler.go` | 列表过滤条件已完整，导出可直接复用 |
| MinIO 预签名 URL | `stationlog/service.go:DownloadURL` | 单文件下载已支持，ZIP 流式打包是新方法 |
| 「导出」按钮 | `pages/device/AbnormalReboot/index.tsx` | UI 位已留 |

**改动范围**：

| 层 | 文件 | 主要内容 |
|----|------|---------|
| handler | `internal/stationlog/handler.go` | 新端点 `GET /device-abnormal-reboots/export?format=csv`（参考 T-0115 的实现模式：`encoding/csv` 流式输出 + RFC5987 文件名编码） |
| service | `internal/stationlog/service.go` | 新方法 `ExportCSV(ctx, filter, w io.Writer)` 流式分页查询 + 写出 |
| handler | `internal/stationlog/handler.go` | 新端点 `POST /device-abnormal-reboots/batch-download`（body: `{ids: [...]}`），返回 ZIP 流 |
| service | `internal/stationlog/service.go` | 新方法 `BatchZip(ctx, ids, w io.Writer)`：用 `archive/zip` 流式拼接 MinIO 对象 |
| 前端 | `pages/device/AbnormalReboot/index.tsx` | 「导出」按钮调 CSV 端点；批量选中时显示「打包下载」按钮调 ZIP 端点 |

**验收标准**：
- [ ] 导出 1 万条 → CSV 流式写出，不爆内存（监控 RSS 不超过 baseline + 50MB）
- [ ] ZIP 打包 10 个文件 → 内容完整可解压
- [ ] CSV 列与规范 §7.5 一致（设备 SN / 名称 / 类型 / IP / 产品 / 版本 / 异常类型 / 文件名 / 时间 / 运行时长 / 故障原因）

**风险**：
- R1（低）ZIP 打包时若某个 MinIO 对象不存在（is_deleted=true），需 graceful skip 而非整体失败

**规模**：S（1 工作日）

---

### 11.6 任务编号 & Sprint 分配建议

按 backlog.md 现有体系建议登记如下：

```
T-0159  triaged  M   docs/project/abnormal-reboot-log-plan-20260521.md §11.1
T-0160  triaged  S   docs/project/abnormal-reboot-log-plan-20260521.md §11.2
T-0161  triaged  S   docs/project/abnormal-reboot-log-plan-20260521.md §11.3   (deps: T-0159)
T-0162  triaged  M   docs/project/abnormal-reboot-log-plan-20260521.md §11.4
T-0163  triaged  S   docs/project/abnormal-reboot-log-plan-20260521.md §11.5
```

合计：6.5 工作日（含 buffer），建议一个 Sprint 内闭环。

---

## 12. 本期完成情况（2026-05-21 闭环）

T-0158 五个 sub-task 全部完成；本期范围严格按 §2.1 G1-G4 收敛，§2.2 列出的不在本期项均按计划留下期（详见 §11）。

| sub-task | 状态 | 文件清单 |
|----------|------|---------|
| T-0158-a DB schema + 仓库扩展 | ✓ | `migrations/000150_*`、`seed/000139_*`、`stationlog/{model,pg_repository}.go` |
| T-0158-b 判断规则切换为 HaltReason | ✓ | `device/{device_service,inform_handler,inform_handler_test}.go`、`alarm/reboot_monitor.go` |
| T-0158-c 识别即落库 service 层 | ✓ | `device/abnormal_reboot.go`、`stationlog/service.go`、`cmd/app/provider/modules.go`、`stationlog/service_test.go` |
| T-0158-d API 端点 + E2E | ✓ | `stationlog/handler.go`、`stationlog/handler_test.go`、`scripts/e2e_verify.sh`（+7 claim） |
| T-0158-e 前端业务层 + 页面迁移 | ✓ | `frontend-core/services/api/deviceAbnormalRebootApi.ts`、`frontend-core/hooks/api/useDeviceAbnormalReboot.ts`、`webcode/pages/device/AbnormalReboot/`、`webcode/router/{routes,componentRegistry}.ts`、`webcode/components/Layout/Sidebar/navConfig.ts`、`frontend-core/i18n/{zh-CN,en-US}` |

测试：`go build ./...` + `go test ./internal/{device,stationlog,alarm}/...` 全过；webcode `typecheck` + 新增文件 `lint` 净 0；webcode-v2/v3 typecheck 有 pre-existing 错误（已 stash 验证基线相同），与本期改动无关。

---

*文档结束*
