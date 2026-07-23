# MML Seed 分组修复审查报告

- 日期：2026-07-23
- 作者：wangyong
- 范围：migration
- 结论：PASS

## 变更范围

- 修复 MML350 新环境 seed 中部分配置组显示名仍为技术路径的问题。
- 停用旧的 `ADD/RMV SI_SUB_05` 空字段命令壳，避免新环境插入 NR_CELL 对象命令时出现活跃命令名冲突。
- 为 `Device.Ethernet.Interface.{i}.PortType` 和 `Device.Ethernet.Interface.{i}.interfaceType` 指定不同 MML code，避免同命令字段编码唯一约束冲突。
- 删除 NR 日志字段 seed 中残留的无效 CTE 片段。
- 从活跃 MML 目录中移除 `MML350_G_DEVICE_SERVICES_FAPSERVICE` 分组、其 5 个命令和 41 个字段绑定，保留 `standard_params` 标准参数。
- 在 `mml_apply_config_updates_20260721.sql` 中同步追加同样清理，避免手工重跑脚本后恢复该分组。

## 审查项

- CRITICAL：无。
- WARNING：无。
- INFO：`000001_init_seed.sql` 是聚合 seed 基线，改动会影响新环境初始化；已通过清空本地 Docker volumes 后完整重建验证。

## 验证

- `npm run build`（`omcmb/webcode`）：通过；仅有既有 Vite 弃用和 chunk size 警告。
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh down -v`：已清空本地环境和数据卷。
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build`：通过；schema 迁移到 version 3，seed 迁移到 version 2。
- `curl -I --max-time 10 http://localhost:8081/`：返回 `HTTP/1.1 200 OK`。
- 数据库校验：
  - `MML350_G_DEVICE_SERVICES_FAPSERVICE` 分组数为 0。
  - 相关 `LST MML350_DEVICE_SERVICES_FAPSERVICE__%` 活跃命令数为 0。
  - 相关活跃字段绑定数为 0。
  - MML350 技术路径显示名异常数为 0。

## 风险与影响

- 影响范围集中在 MML 配置目录初始化和 20260721 MML 配置追加脚本。
- 删除的是 MML 活跃分组/命令/字段绑定，不删除标准参数定义。
- 现有环境如已执行过旧 seed，需要配合本次清理 SQL 或重新初始化才能消除旧分组；本地已按新环境重建验证。
