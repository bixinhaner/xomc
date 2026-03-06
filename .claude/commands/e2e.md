# OMC E2E 端到端验证

执行 OMC 系统端到端数据流验证，覆盖认证、设备管理、告警管理、配置模板、固件管理、用户管理、角色、审计日志、设备分组、PM 计数器、KPI 查询、MR 文件/数据、审计日志时间过滤全链路 (132 个测试用例)。

## 环境信息

- **PSQL**: `/usr/local/Cellar/postgresql@16/16.13/bin/psql`
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

1. **PostgreSQL 连通性**: 运行 `/usr/local/Cellar/postgresql@16/16.13/bin/psql "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable" -c "SELECT 1;"` 验证数据库可达
2. **后端编译检查**: 运行 `cd omcgo && go build ./...` 确认代码编译通过
3. **后端单元测试**: 运行 `cd omcgo && go test ./...` 确认测试通过
4. **前端类型检查**: 运行 `cd omcmb/webcode && npx tsc --noEmit` 确认类型无误

如果任何检查失败，**立即停止并报告错误**，不要继续后续步骤。

### Step 2: Seed 测试数据

运行以下命令刷新 E2E 测试数据（脚本幂等，会先清理旧数据再插入）：

```bash
/usr/local/Cellar/postgresql@16/16.13/bin/psql "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable" -f omcgo/scripts/seed_e2e_testdata.sql
```

确认输出包含正确的计数: 设备(5)、告警(8)、配置模板(3)、固件版本(3)、升级任务(2)、设备分组(3)、PM计数器(8)、KPI定义(2)、KPI值(4)、MR文件(2)、MR记录(4)、审计日志(3)。

### Step 3: 重建后端

```bash
cd omcgo && make build-app
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
  - 启动新进程: `cd omcgo && nohup ./bin/omcgo-app > /tmp/omcgo-app.log 2>&1 &`
  - 等待 3 秒让服务初始化
  - 验证: `curl -s http://localhost:8080/healthz`
- 如果返回非 200：
  - 启动后端: `cd omcgo && nohup ./bin/omcgo-app > /tmp/omcgo-app.log 2>&1 &`
  - 等待 3 秒让服务初始化
  - 验证: `curl -s http://localhost:8080/healthz`
  - 如果仍然失败，检查 `/tmp/omcgo-app.log` 输出错误日志

### Step 5: 执行 E2E 验证

```bash
cd omcgo && ./scripts/e2e_verify.sh http://localhost:8080
```

脚本包含 132 个测试用例，覆盖：

**Sprint 1:**
- 健康检查 (1)
- 认证/登录/Token刷新 (9)
- 获取当前用户 (5)
- 设备列表/过滤/详情/统计/分页 (11)
- 告警列表/统计/详情/确认 (12)
- CORS 预检 (2)
- 错误处理 (3)

**Sprint 2:**
- 配置模板 CRUD: 列表/详情/创建/更新/删除 (10)
- 固件管理: 列表/详情/删除 (7)
- 升级任务: 列表/详情 (4)
- 用户管理: 列表/创建/详情/更新/删除 (9)
- 角色管理: 列表 (2)
- 审计日志: 列表 (2)
- 设备分组: 列表/详情/创建/更新/设备列表/删除 (10)
- 告警清除 (1)

**Sprint 3:**
- PM 计数器: 列表/设备过滤/组过滤/时间过滤/字段验证 (9)
- KPI 查询: 值列表/名称过滤/定义列表/运营商过滤/字段验证 (8)
- MR 文件与数据: 文件列表/类型过滤/下载/数据列表/设备过滤/文件字段/记录字段 (11)
- 审计日志时间过滤: 范围查询/空范围/字段验证/操作过滤 (6)

### Step 6: 结果汇总

根据脚本输出汇总结果：

- **全部通过**: 报告 "E2E 验证通过 (N/N PASS)"，确认所有数据流正常
- **部分失败**:
  1. 列出所有失败的用例及错误信息
  2. 分析失败原因（网络/数据/代码 bug）
  3. 如果是代码 bug，定位问题文件并建议修复方案
  4. 如果是数据问题（如告警已被确认导致重复确认失败），建议重新 seed 后重跑

## 注意事项

- E2E 脚本会修改数据状态（如确认告警），重复运行��需重新执行 Step 2 刷新数据
- 后端依赖 PostgreSQL + Redis，确保 Docker 基础设施正在运行
- 如果 `psql` 路径不可用，尝试 `which psql` 或 `find /usr/local -name psql -type f` 查找
