# OMC E2E 端到端验证

执行 OMC 系统端到端数据流验证，覆盖基础通信层 (CORS/healthz/统一错误格式/前端基础设施)、认证、设备管理、告警管理、配置模板、固件管理、用户管理、角色、审计日志、设备分组、PM 计数器、KPI 查询、MR 文件/数据、审计日志时间过滤、Dashboard 聚合、设备 CRUD、告警规则、KPI 阈值、系统日志、NE 消息日志、密码管理、权限管理、角色 CRUD、错误响应 request_id、CORS 配置化、OpenAPI 文档、Dashboard 趋势/状态/区域统计、Config Sync (push/pull/status)、备份恢复 (任务/计划 CRUD)、文件管理 (上传/下载/过滤)、MML 控制台 (命令/脚本/任务)、前端集成验证、全量回归全链路，以及 Phase A-D 全模块对齐新端点 (System Info/Dashboard Widgets/PM Tasks/Config Baselines/FTP Configs/MR Indicators/License/Topology Sites/Reports/OpsTools) (~407 个测试用例, Sprint 0-10, S0-S74)。

## 环境信息

- **DB DSN**: `postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable`
- **Backend**: `http://localhost:8080`
- **Backend 目录**: `omcgo/`
- **Frontend 目录**: `omcmb/`
- **Seed 数据**: `omcgo/scripts/seed_e2e_testdata.sql`
- **验证脚本**: `omcgo/scripts/e2e_verify.sh`

## 执行步骤

请按以下顺序严格执行，每步完成后报告状态：

### Step 1: 前置检查

并行执行以下检查：

1. **PostgreSQL 连通性**: 通过 python3 psycopg2 或 psql 验证数据库可达
2. **后端编译检查**: 运行 `cd omcgo && go build ./...` 确认代码编译通过
3. **前端类型检查**: 运行 `cd omcmb/webcode && npx tsc --noEmit` 确认类型无误

如果任何检查失败，**立即停止并报告错误**，不要继续后续步骤。

### Step 2: Seed 测试数据

通过 python3 + psycopg2 执行 SQL 文件刷新 E2E 测试数据（脚本幂等，会先清理旧数据再插入）：

```python
python3 -c "
import psycopg2
conn = psycopg2.connect('postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable')
conn.autocommit = True
with open('omcgo/scripts/seed_e2e_testdata.sql', 'r') as f:
    cur = conn.cursor()
    cur.execute(f.read())
    cur.close()
conn.close()
print('Seed data applied successfully')
"
```

如果 psycopg2 不可用，先安装: `pip3 install --user --break-system-packages psycopg2-binary`

确认输出包含正确的数据: 设备(5)、告警(8+3 trend)、配置模板(3)、固件版本(3)、升级任务(2)、设备分组(3)、PM计数器(8)、KPI定义(2)、KPI值(4)、MR文件(2)、MR记录(4)、审计日志(3)、告警规则(3)、KPI阈值(3)、系统日志(3)、NE消息日志(3)、备份任务(2)、备份计划(2)、管理文件(3)、MML脚本(1)、PM任务(2)、配置基线(2)、配置任务(1)、邻区(2)、FTP配置(2)、MR映射(2)、License(3)、站点(2)、拓扑节点(3)、拓扑边(2)、报表定义(1)、报表记录(1)、运维模板(1)、命令记录(1)。

### Step 3: 重建后端

```bash
cd omcgo && go build -o ./bin/omcgo-app ./cmd/app/
```

### Step 4: 确保后端运行

检查后端进程状态：

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz
```

- 如果返回 `200`：后端已运行，检查是否需要重启（如果 Step 3 重建了新的二进制文件）
  - 获取旧进程 PID: `lsof -ti:8080`
  - 终止旧进程: `kill <PID>`
  - 等待 2 秒让端口释放
  - 启动新进程: `cd omcgo && nohup ./bin/omcgo-app --config configs/app.yaml > /tmp/omcgo-app.log 2>&1 &`
  - 等待 3 秒让服务初始化
  - 验证: `curl -s http://localhost:8080/healthz`
- 如果返回非 200：
  - 启动后端: `cd omcgo && nohup ./bin/omcgo-app --config configs/app.yaml > /tmp/omcgo-app.log 2>&1 &`
  - 等待 3 秒让服务初始化
  - 验证: `curl -s http://localhost:8080/healthz`
  - 如果仍然失败，检查 `/tmp/omcgo-app.log` 输出错误日志

### Step 5: 执行 E2E 验证

```bash
cd omcgo && bash ./scripts/e2e_verify.sh http://localhost:8080
```

脚本包含 ~407 个测试用例 (Sprint 0-10, S0-S74)，覆盖：

**Sprint 0 (12 cases, S1-S4):**
- 基础通信层: 健康检查 (1), CORS preflight 204 (1), CORS 5 项头部验证 (5), 统一错误格式 (2), 前端静态检查 (3)

**Sprint 1 (43 cases, S5-S8):**
- 健康检查 (1)
- 认证/登录/Token刷新 (9)
- 获取当前用户 (5)
- 设备列表/过滤/详情/统计/分页 (11)
- 告警列表/统计/详情/确认 (12)
- CORS 预检 (2)
- 错误处理 (3)

**Sprint 2 (45 cases, S9-S14):**
- 配置模板 CRUD: 列表/详情/创建/更新/删除 (10)
- 固件管理: 列表/详情/删除 (7)
- 升级任务: 列表/详情 (4)
- 用户管理: 列表/创建/详情/更新/删除 (9)
- 角色管理: 列表 (2)
- 审计日志: 列表 (2)
- 设备分组: 列表/详情/创建/更新/设备列表/删除 (10)
- 告警清除 (1)

**Sprint 3 (44 cases, S15-S18):**
- PM 计数器: 列表/设备过滤/组过滤/时间过滤/字段验证 (9)
- KPI 查询: 值列表/名称过滤/定义列表/运营商过滤/字段验证 (8)
- MR 文件与数据: 文件列表/类型过滤/下载/数据列表/设备过滤/文件字段/记录字段 (11)
- 审计日志时间过滤: 范围查询/空范围/字段验证/操作过滤 (6)

**Sprint 4 (59 cases, S19-S24):**
- Dashboard 聚合: summary/device_stats/alarm_stats/timestamp (4)
- 设备 CRUD: create/duplicate-409/update/verify/delete/verify-404/invalid-400 (8)
- 告警规则 CRUD: list/count/get/filter-carrier/filter-enabled/create/update/verify/delete/condition_config (9)
- KPI 阈值 CRUD: list/count/get/create/update/delete/filter-carrier/filter-enabled (8)
- 系统日志: list/count/filter-level/filter-source/field-validation (5)
- NE 消息日志: list/count/filter-device_sn/filter-message_type/field-validation (5)
- 密码管理: reset-password/lock/unlock (3)
- 权限列表: list/count (2)
- 角色 CRUD: list/count/get/create/update/verify/delete/invalid-uuid (8)

**Sprint 5 (7 cases, S25-S30):**
- 错误响应 request_id: 404 响应含 request_id / 400 响应含 request_id (2)
- CORS 配置化: 允许已配置 origin / 拒绝未配置 origin (2)
- 回归: 健康检查 (1)
- 错误码域范围: Go 集成测试验证 35/35 (1)
- OpenAPI 文档: 文件存在且 >3000 行 (1)

**Sprint 6 (~30 cases, S31-S38):**
- S31: Admin 角色 CRUD 扩展 (5) — createRole → getRoleById → updateRole → deleteRole → getPermissions
- S32: Admin 用户操作扩展 (3) — resetPassword → lockUser → unlockUser
- S33: Admin 角色分配 (3) — assignRole → verify → removeRole
- S34: PM 阈值 CRUD 扩展 (5) — listThresholds → createThreshold → updateThreshold → verify → deleteThreshold
- S35: DataModel 扩展操作 (3) — getAggregatedCounters → calculateKPI → verify
- S36: Group CRUD 扩展 (5) — createGroup → addDevice → listDevices → removeDevice → deleteGroup
- S37: Device 扩展操作 (3) — getStats → getParameters → reboot
- S38: Config Sync (3) — pushConfig → pullConfig → getSyncStatus

**Sprint 7 (~20 cases, S39-S46):**
- S39: Dashboard 告警趋势 (3) — alarm-trend endpoint with day range, date field validation
- S40: Dashboard 设备状态 (2) — device-status endpoint with status counts
- S41: Dashboard KPI 趋势 (3) — kpi-trend endpoint with time/value fields, required param validation
- S42: Dashboard 区域统计 (2) — region-stats endpoint with array response
- S43: Config Sync 集成 (3) — push/pull/status with Sprint 7 specific payloads
- S44: Device 参数查询 (2) — device parameter retrieval and items validation
- S45: DataModel resolve (2) — datamodel list with items check
- S46: Sprint 7 回归 (3) — dashboard/summary, healthz, CORS headers

**Sprint 8 (~25 cases, S47-S52):**
- S47: Backup 任务 CRUD (6) — list → create → get → cancel → verify cancelled → delete
- S48: Backup 计划 CRUD (4) — list → create → update → delete
- S49: 文件管理 (6) — list → multipart upload → get detail → download with Content-Disposition → type filter → delete
- S50: MML 命令 (4) — list commands (3 seed) → get command detail → execute command → get task status
- S51: MML 脚本 (3) — create script → list scripts → delete script
- S52: MML 任务历史 (2) — list tasks → verify task from S50 in list

**Sprint 9 (~15 cases, S53-S56):**
- S53: Backup 前端集成 (3) — paginated task list with total, create+delete cleanup, paginated schedule list
- S54: 文件管理前端集成 (3) — paginated file list with total, file_type filter, file detail by id
- S55: MML 前端集成 (3) — paginated command list, execute command with task_id, task detail with status
- S56: 全量回归 (6) — healthz, dashboard summary, devices list, alarms list, alarm-trend, CORS headers

**Sprint 10 (~76 cases, S57-S74) — Phase A-D 全模块对齐:**
- S57: System Info (2) — GET /system/info → version/db_status
- S58: Dashboard Widgets + KPI + Alarm Pie (4) — GET/PUT widgets, alarm-type-pie, kpi-time-series
- S59: PM Tasks (4) — GET list, POST create, status filter, field check
- S60: Phase A Regression (2) — system/info + dashboard/widgets smoke
- S61: Config Baselines CRUD (7) — list → create → get → update → verify → delete → 404
- S62: Config Tasks & Neighbors (4) — tasks list/create, neighbors list/field check
- S63: FTP Config CRUD (6) — list → create → update → test → delete → filter
- S64: MR Indicators & Mappings (5) — indicators paginated/all, mappings list/update/toggle
- S65: License CRUD (8) — list → get → summary → activate → revoke → import → get new → filter
- S66: Topology Sites (3) — list → create → get
- S67: Topology Graph (4) — nodes → edges → graph → geo
- S68: Reports CRUD (8) — definitions list/create/get/update, generate, records, sample-data, delete
- S69: OpsTools Templates CRUD (5) — list → create → get → update → delete
- S70: OpsTools Command Records (3) — list → create → field check
- S71: OpsTools Tasks Lifecycle (6) — create → list → get → cancel → create → pause
- S72: MR Export (1) — POST export placeholder
- S73: Error Code Spot Check (2) — baselines 404, licenses 404
- S74: Phase A-D Full Regression (5) — 5 key endpoints smoke test

### Step 6: 结果汇总

根据脚本输出汇总结果：

- **全部通过**: 报告 "E2E 验证通过 (N/N PASS)"，确认所有数据流正常
- **部分失败**:
  1. 列出所有失败的用例及错误信息
  2. 分析失败原因（网络/数据/代码 bug）
  3. 如果是代码 bug，定位问题文件并建议修复方案
  4. 如果是数据问题（如告警已被确认导致重复确认失败），建议重新 seed 后重跑

## 注意事项

- E2E 脚本会修改数据状态（如确认告警、创建/删除资源），重复运行前需重新执行 Step 2 刷新数据
- 后端依赖 PostgreSQL + TimescaleDB + Redis + NATS + MinIO，确保基础设施正在运行
- 脚本依赖 `python3` (JSON 解析) 和 `curl`，不依赖 `jq`
- Sprint 4/6/8/9 测试包含 CRUD 操作（创建→更新→删除），确保 seed 数据不影响后续测试
- Sprint 8 测试依赖 backup/filemanager/mml 三个新模块的 migration（000022-000024）
