# 系统管理 — 系统仪表板（System / Dashboard）PRD

> 文档目的：以"运维一屏"视角展示系统健康指标 — 资源占用、设备在线、告警概况、微服务状态、最近操作。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `SystemDashboard/index.tsx` + `internal/core/components/sysinfo.go` 抽取 |

**关联功能域**：F06 OMC-R 核心 / 运维可观测性

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/SystemDashboard/index.tsx](../../../../omcmb/webcode/src/pages/system/SystemDashboard/index.tsx) |
| 前端 Hook | `useSystemInfo()` from `@core/hooks/api/useSystem` |
| 后端 Handler | `sysInfoHandler.GetSystemInfo`（推断在 `internal/core/components/sysinfo.go`）|
| 路由注册 | [cmd/app/provider/router.go:385](../../../cmd/app/provider/router.go#L385) `sysInfoGroup.GET("/system/info", ...)` |

---

## 1. 业务背景

系统仪表板是运维入口的"瞭望塔"：

- **用户**：管理员 / 运维工程师
- **使用频次**：登录后默认首页；每分钟自动刷新
- **数据特点**：实时聚合，无写操作，纯只读 GET

**不在范围**（其它专项页面承担）：
- 告警明细 → F04 告警管理
- 设备列表 → F06 设备管理
- KPI 趋势 → F03 性能管理

---

## 2. 实体模型

无独立 DB 表。数据来源：

| 区块 | 数据源 |
|------|--------|
| CPU/内存/磁盘 | `gopsutil` 实时采集（`internal/core/components/sysinfo.go`）|
| 在线设备数 | `SELECT COUNT(*) FROM devices WHERE conn_status = 'online'` |
| 活跃告警数 | `SELECT COUNT(*) FROM alarms WHERE status = 'active'` |
| 微服务状态 | 进程内 health check + 外部依赖（PG/Redis/NATS/MinIO）ping |
| 最近操作日志 | `SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 6` |

---

## 3. 页面布局（仪表板，无表格）

### 3.1 顶部 4 张统计卡片

| 卡片 | 指标 | 数据字段 | 颜色 |
|------|------|---------|------|
| 在线设备 | `online_count / total_count` | `device.online`, `device.total` | 蓝 |
| 活跃告警 | 告警总数 | `alarm.active_count` | 红/橙（按数量分级）|
| 活跃会话 | 当前在线用户数 | `session.active_count` | 绿 |
| 微服务正常数 | `up / total` | `services.up`, `services.total` | 绿 |

### 3.2 中部 3 个 Gauge

| Gauge | 指标 | 字段 | 阈值 |
|-------|------|------|------|
| CPU 使用率 | 0-100% | `cpu_usage` | 80% 黄 / 90% 红 |
| 内存使用率 | 0-100% | `mem_usage` | 同上 |
| 磁盘使用率 | 0-100% | `disk_usage` | 85% 黄 / 95% 红 |

### 3.3 微服务状态网格（10 个）

| 列 | 说明 |
|----|------|
| 服务名 | `omcgo-app` / `omcgo-acs` / `omcgo-worker` / `postgres` / `redis` / `nats` / `minio` / 等 |
| 状态 | `<Tag>` 在线 绿 / 离线 红 / 告警 黄 |
| 端口 | 数字 |
| 运行时间 | `2d 3h 14m` |

### 3.4 最近操作日志时间线（6 条）

```
2026-05-06 12:34:56  admin     创建用户 alice
2026-05-06 12:30:00  admin     重置密码 bob
...
```

数据源：`GET /admin/audit-logs?pageSize=6&sort=created_at:desc`

---

## 4. 操作清单

| 操作 | 行为 |
|------|------|
| 自动刷新 | 60 秒 polling（React Query `refetchInterval`）|
| 手动刷新 | 顶部右上角刷新按钮 |
| 卡片点击跳转 | 在线设备 → 设备列表；活跃告警 → 告警列表；操作日志 → 操作日志页 |

---

## 5. 表单字段定义

无（纯展示）。

---

## 6. 接口契约

### 6.1 已实现

| Method | 路径 | 响应字段 |
|--------|------|---------|
| GET | `/api/v1/system/info` | 见下 |

**Response 200**（推断结构，需要看 `sysInfoHandler` 实际返回）：

```json
{
  "cpu_usage": 45.2,
  "mem_usage": 62.8,
  "disk_usage": 38.1,
  "device": { "online": 1234, "total": 1500 },
  "alarm": { "active_count": 23 },
  "session": { "active_count": 5 },
  "services": [
    { "name": "omcgo-app", "status": "up", "port": 8081, "uptime_sec": 192600 },
    { "name": "postgres", "status": "up", "port": 5432, "uptime_sec": 1234567 },
    ...
  ]
}
```

### 6.2 待补

| Method | 路径 | 说明 | 优先级 |
|--------|------|------|--------|
| GET | `/api/v1/system/info/recent-logs?limit=6` | 复用现有 `/admin/audit-logs?pageSize=6`，或专门聚合接口 | P2 |
| WebSocket | `/api/v1/system/info/stream` | 实时推送指标变化（替代 polling）| P3 |

---

## 7. 后端补齐 Backlog

### P0
1. **响应字段确认**：当前 `sysInfoHandler.GetSystemInfo` 实际返回字段需与前端期望对齐（前端代码访问 `systemInfo?.cpuUsage` 等），可能存在 snake↔camel 自动转换或缺字段
2. **设备/告警/会话计数**：`/system/info` 是否包含这些聚合数字？若仅有 CPU/内存/磁盘等系统资源，需要 service 层加入 `device_repository.CountByStatus` / `alarm_repository.CountActive` / Redis SCAN 会话数

### P1
3. **微服务健康检查**：扩展 `health.go`，针对每个外部依赖（PG/Redis/NATS/MinIO）单独 ping，返回结构化数组
4. **微服务运行时间**：进程启动时记录 `start_time`，响应中返回 `uptime_sec`

### P2
5. **指标缓存** —— 短期缓存（5-10 秒）避免每分钟 polling 都打实时聚合查询

---

## 8. 验收清单（DoD）

后端：
- [ ] `/system/info` 响应字段与 §6.1 对齐
- [ ] 聚合查询响应时间 < 200ms（10 万设备规模）

前端：
- [ ] 60s 自动刷新生效
- [ ] 离线状态友好降级（如 CPU 数据缺失显示 `--`，不报错）
- [ ] `npm run typecheck` & `lint` 通过

---

## 9. 非目标

- 自定义看板布局（拖拽组件）— 当前固定布局
- 历史趋势图 — 由 F03 性能管理 PRD 承担
