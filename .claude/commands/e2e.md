# OMC E2E 端到端验证

执行 OMC 系统端到端数据流验证，覆盖认证、设备管理、告警管理、配置模板、固件管理、用户管理、角色、审计日志、设备分组、PM 计数器、KPI 查询、MR 文件/数据、审计日志时间过滤、Dashboard 聚合、设备 CRUD、告警规则、KPI 阈值、系统日志、NE 消息日志、密码管理、权限管理、角色 CRUD 全链路 (191 个测试用例)。

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

确认输出包含正确的数据: 设备(5)、告警(8)、配置模板(3)、固件版本(3)、升级任务(2)、设备分组(3)、PM计数器(8)、KPI定义(2)、KPI值(4)、MR文件(2)、MR记录(4)、审计日志(3)、告警规则(3)、KPI阈值(3)、系统日志(3)、NE消息日志(3)。

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

脚本包含 191 个测试用例，覆盖：

**Sprint 1 (43 cases):**
- 健康检查 (1)
- 认证/登录/Token刷新 (9)
- 获取当前用户 (5)
- 设备列表/过滤/详情/统计/分页 (11)
- 告警列表/统计/详情/确认 (12)
- CORS 预检 (2)
- 错误处理 (3)

**Sprint 2 (45 cases):**
- 配置模板 CRUD: 列表/详情/创建/更新/删除 (10)
- 固件管理: 列表/详情/删除 (7)
- 升级任务: 列表/详情 (4)
- 用户管理: 列表/创建/详情/更新/删除 (9)
- 角色管理: 列表 (2)
- 审计日志: 列表 (2)
- 设备分组: 列表/详情/创建/更新/设备列表/删除 (10)
- 告警清除 (1)

**Sprint 3 (44 cases):**
- PM 计数器: 列表/设备过滤/组过滤/时间过滤/字段验证 (9)
- KPI 查询: 值列表/名称过滤/定义列表/运营商过滤/字段验证 (8)
- MR 文件与数据: 文件列表/类型过滤/下载/数据列表/设备过滤/文件字段/记录字段 (11)
- 审计日志时间过滤: 范围查询/空范围/字段验证/操作过滤 (6)

**Sprint 4 (59 cases):**
- Dashboard 聚合: summary/device_stats/alarm_stats/timestamp (4)
- 设备 CRUD: create/duplicate-409/update/verify/delete/verify-404/invalid-400 (8)
- 告警规则 CRUD: list/count/get/filter-carrier/filter-enabled/create/update/verify/delete/condition_config (9)
- KPI 阈值 CRUD: list/count/get/create/update/delete/filter-carrier/filter-enabled (8)
- 系统日志: list/count/filter-level/filter-source/field-validation (5)
- NE 消息日志: list/count/filter-device_sn/filter-message_type/field-validation (5)
- 密码管理: reset-password/lock/unlock (3)
- 权限列表: list/count (2)
- 角色 CRUD: list/count/get/create/update/verify/delete/invalid-uuid (8)

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
- Sprint 4 测试包含 CRUD 操作（创建→更新→删除），确保 seed 数据不影响后续测试
