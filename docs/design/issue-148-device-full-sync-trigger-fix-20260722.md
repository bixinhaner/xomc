# Issue #148：设备重新接入与 OMC 发布全量同步

> 状态：已实施并验证
>
> 更新日期：2026-07-23
>
> 代码基线：`main@c7c0ed9bc`

## 目标

只增加两个全量参数同步触发：

1. 设备在设备分组页面永久删除后重新接入，创建新设备 UUID，并同步一次当前参数。
2. OMC 正式交付包发生切换后，分批同步在线、已入网设备；同一交付包重启或多副本启动不重复同步。

全量同步只读取设备当前参数，不恢复已删除设备的告警、KPI、监控或配置历史。

## 业务规则

### 设备重新接入

- 设备分组红色“删除”调用永久删除接口；重新接入按新设备处理，不恢复旧 UUID。
- 回收站设备继续保持软删除状态，Inform 不自动恢复，也不创建重复设备。
- 永久删除后没有遗留记录可区分“第一次接入”和“删除后接入”，因此所有真正创建新 UUID 的设备首次注册时同步一次。
- 已存在设备重复 Bootstrap 只更新设备，不触发本次全量同步。

`RegisterFromInform` 返回 `Device + Created`，`device.registered` 事件透传 `created`。只有以下条件同时满足才提交：

```text
created=true
routing_mode=durable
trigger_reason=device_registered
scope=full
idempotency_key=device_registered:<device_uuid>
```

该入口直接进入现有 `parameter_sync` durable 数据面，不使用 legacy `sync-gpv` fallback。

为避免设备已经入库、但 `device.registered` 发布失败后永久丢失本次触发，首次注册还要把原始
Inform Event ID 写入现有 `devices.extension_data` 的系统保留键。派生的
`device.registered` 复用该 Event ID；发布失败向上返回，让原 Inform 消息重投。
同一原始事件重投时仍识别为本次新建设备并保持 `created=true`，而参数同步请求继续由
`device_registered:<device_uuid>` 保证幂等。该方案不新增表或字段。

`device_registered` 是强制业务触发：只受 `run_enabled` 和 `routing_mode=durable`
控制，不受设备级 `canary_percent` 漏选。已有设备的普通重复 Bootstrap 仍为
`created=false`，不会借此补触发。

### OMC 发布切换

正式构建把 `ReleaseVersion` 和 `GitCommit` 编译进 App。后端不解析版本格式，只计算：

```text
SHA256(ReleaseVersion + "\x00" + GitCommit) -> deterministic campaign UUID
```

因此版本号改成 `V100R003C20`、`2027Q1-GA` 等格式也不影响流程：

- 相同交付包得到相同 campaign，同版本重启不重复；
- 版本或提交变化得到新 campaign；
- 开发构建缺少正式元数据时不启动发布同步。

构建信息只由 `build-release.sh` 传给 `Dockerfile.app`，不修改 Compose、容器运行时环境变量、端口、网络或 Volume。

发布扫描复用现有 `PeriodicSyncer` 的 PG leader、BatchSize、MaxConcurrent、StaggerWindow 和轮询周期，但不受 `periodicSyncEnabled` 开关控制。候选设备必须：

- 未删除、`commissioned`、在线；
- 自动退避已到期；
- 本 campaign 没有 `accepted/queued/running/succeeded` 的 `omc_upgrade` 请求。

每次重试使用新的 attempt idempotency key；单台失败不阻断其他设备。候选排序优先选择
从未尝试过的设备，再选择退避到期的失败设备，避免固定 `device_id + LIMIT` 被持续失败设备
占满。`omc_upgrade` 与 `device_registered` 一样是强制业务触发，不受设备级
`canary_percent` 漏选。

Stagger 等待发生在获取并发槽之前；`MaxConcurrent` 只限制真正提交中的设备，不能让等待
Stagger 的设备占住并发槽并把一个窗口成倍放大。

## 数据库

不新增表和列。迁移 `000004` 只扩展现有两个 CHECK 约束，允许：

```text
device_registered
omc_upgrade
```

独立 trigger reason 用于正确区分业务审计、候选判断和自动退避，不能用 `bootstrap/periodic` 冒充。

## 修改边界

本 Issue 只修改：

- 新设备注册的 `Created` 语义和 durable 全量同步；
- 正式构建身份与确定性发布 campaign；
- 基于现有 `parameter_sync_requests` 的发布候选扫描；
- 对应测试和约束迁移。

明确不修改：

- 永久删除清理历史数据的既有实现；
- 回收站恢复规则；
- Compose、运行时 Docker 配置和前端；
- 新的发布批次表、管理页面或独立调度器；
- 周期同步和参数同步数据面的既有业务语义。

## 验收

- 永久删除后重新接入生成新 UUID，只提交一次 `device_registered` 全量同步。
- 普通首次注册同样只提交一次；已有设备重复 Bootstrap 不提交。
- 回收站设备不恢复、不提交。
- 新正式发布生成新 campaign，分批提交 `omc_upgrade`。
- 同一交付包重启、多副本启动和已成功设备不重复提交。
- 离线设备不创建南向任务，后续在线后可成为候选。
- 单台持续失败不阻塞排序靠后的设备，发布 campaign 可继续向后收敛。
- `device.registered` 发布失败后，原 Bootstrap 重投仍能补交同一个幂等全量同步请求。
- 非零 StaggerWindow 不因 MaxConcurrent 分批等待而成倍放大。
- `routing_mode=closed` 不执行；只有 `durable` 执行。
- 不新增业务表，不修改 Compose。

验证结果（2026-07-23）：

```text
go build ./...                                                    PASS
go test ./... -count=1                                            PASS
bash -n deployments/release/build-release.sh                      PASS
docker compose ... build migrate-schema                           PASS
docker compose ... up -d --force-recreate postgres migrate-schema PASS（Exited 0）
```
