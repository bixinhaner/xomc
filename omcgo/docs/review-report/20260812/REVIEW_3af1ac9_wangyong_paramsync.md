# 参数同步 durable 化与重部署同步审查报告

## 结论

PASS

本次改动未发现 CRITICAL 问题。durable 参数同步已成为唯一数据链路，旧开关和回退逻辑已删除；OMC 重部署会为在线设备提交带独立 `redeploy_id` 的全量同步。

## 审查范围

- durable 参数同步配置、初始化和各触发入口
- OMC 重部署在线设备扫描、重试、幂等和生命周期日志
- partial object coverage、staging 与权威范围删除
- 手动、License、device online、registered、firmware、周期及发布同步
- ACS durable readback 与 result consumer 常开行为

## 关键结论

- 未发现参数同步回退到旧 `sync-gpv` 链路的执行分支。
- OMC 重部署请求使用同一轮次内稳定的幂等键，查询和提交错误支持重试。
- 新建设备仅由 `device.registered` durable consumer 触发首次全量同步，避免 Bootstrap 重复提交。
- partial object 仅对明确的完整对象范围执行缺失值清理，并通过 staging 与 `last_updated_at` 防止误删并发新值。
- SQL 使用 Squirrel + pgx，批量对象范围通过数组参数控制 SQL 和 bind 数量。
- 用户确认：重部署提交返回 durable 结果即视为正常流程，不额外改变其状态分类逻辑；多副本重复批次暂不纳入本次范围。

## 验证

- `go test ./cmd/app/provider ./internal/core/appconfig ./internal/device ./internal/paramsync` — 通过。
- `go test ./internal/core/appconfig ./cmd/app/provider ./internal/provision -run 'TestStartupSyncer|TestPeriodicSync|TestParamSync|TestConfig' -count=1` — 通过。
- `git diff --check` — 通过。
- Docker Compose 重新构建并启动 App、ACS、Worker — 成功。
- Web `http://localhost:8081/` — HTTP 200。
- OMC 重部署产生 4 条 `origin_event_type=omc.redeploy` 全量请求 — 全部 `succeeded`。

## 已知非阻塞项

- `internal/provision` 全包测试存在既有 `INTERFACE.Gateway` 模板定义失败，与本次修改无关。
- 本地部署环境缺少 `system_license_usage` 表，Worker 字典同步存在既有主键冲突，与本次参数同步改动无关。
