# xomc 主备部署设计说明

> 目标：参考旧 OMC `one_key_install/master_slave` 的机制，在 xomc 当前 Docker Compose 交付体系上实现两机主备部署。
>
> 本文描述 xomc 两机主备部署的目标架构、数据同步边界、切换流程、安装入口和设计限制；本次变更只新增设计文档，不实现代码。
>
> 当前项目事实：xomc 后端是 `app / acs / worker` 三部署单元；基础设施是 `postgres`、`postgres-tsdb`、`redis-core`、`redis-pm`、`nats`、`minio`；生产部署入口在 `deployments/release/bundle/deploy/`，实例配置在 `/opt/omc/etc/`，有状态数据路径由 `deploy/.env` 的 `POSTGRES_DATA_PATH / TSDB_DATA_PATH / REDIS_DATA_PATH / REDIS_PM_DATA_PATH / NATS_DATA_PATH / MINIO_DATA_PATH` 控制。

## 1. 总体结论

不要逐字照搬旧版 MySQL/Mongo/unison 方案。xomc 应该模仿旧方案的四个核心思想：

1. 用 VIP 作为唯一业务入口，基站、浏览器、北向系统都访问 VIP。
2. 用本地落盘角色 `host_state=MASTER/BACKUP` 作为业务进程判断角色的统一依据。
3. 备机默认不承载业务流量，只保留数据同步、健康检查和切换能力。
4. 切换动作由统一编排器串起来，步骤必须有锁、有日志、可重入、能失败中断。

xomc 的数据层要按现有组件重新设计：

| 旧 OMC 组件 | xomc 对应组件 | 主备策略 |
|---|---|---|
| MySQL 双主 | PostgreSQL 主库 `postgres` | PostgreSQL streaming replication，一主一备，切换时 promote |
| Mongo 副本集 | TimescaleDB `postgres-tsdb` | 独立 streaming replication，一主一备，切换时 promote |
| Redis 6380/6381 切换清空 | `redis-core` / `redis-pm` | AOF + replica；升主时只清理易失前缀，不做全库 flush |
| unison 文件同步 | MinIO + 少量实例文件 | 业务对象文件用 MinIO bucket replication / `mc mirror`；配置和字典类小文件才做单向同步或校验 |
| keepalived notify 脚本 | keepalived + `xomc-ha-manager` | keepalived 负责 VIP，Go 管理器负责编排业务和数据层动作 |

## 2. 推荐目标架构

两台机器：

- Node A：配置 `preferred_master=true`，但初始 `host_state=BACKUP`。
- Node B：配置 `preferred_master=false`，初始 `host_state=BACKUP`。
- VIP：对外业务地址，所有外部访问统一走 VIP。
- 主备固定为两台机器，不引入第三见证节点；自动切换必须依赖多路心跳、gateway/witness、VIP 状态和数据层门禁，策略保持保守。

主机上新增目录：

```text
/opt/omc/ha/
├── omc.properties                 # 主备规划 + 本机运行态角色，禁止跨机同步覆盖
├── keepalived/keepalived.conf     # keepalived 生成配置
├── logs/ha-manager.log            # 切换编排日志，排障第一入口
└── state/                         # 切换锁、重试令牌、最近一次切换记录
```

新增容器/服务：

| 服务 | 运行位置 | 作用 |
|---|---|---|
| `keepalived` | 两台都运行，host network | 持有/释放 VIP，触发 notify 回调 |
| `xomc-ha-manager` | 两台都运行，host network 或 privileged 容器 | 统一健康检查、角色写入、数据层 promote/rebuild、业务容器启停 |
| `postgres` | 主为 primary，备为 standby 只读副本 | 主库业务数据 |
| `postgres-tsdb` | 主为 primary，备为 standby 只读副本 | PM/KPI/告警历史/trace 时序数据 |
| `redis-core` / `redis-pm` | 主为 master，备为 replica | 会话、任务、PM 窗口等 Redis 状态 |
| `minio` | 两台都运行 | 主写入，备通过 bucket replication/mirror 接收对象 |
| `nats` | 随当前 MASTER 运行，BACKUP 冷待命 | JetStream 事件流；两节点强一致有天然限制 |
| `app / acs / worker / web` | 只在 MASTER 运行 | 业务服务；BACKUP 停止 |

## 3. 部署模式定义

主备规划与运行态角色合并到 `/opt/omc/ha/omc.properties`。`deploy/.env` 仍负责镜像版本、数据路径、密钥等现有部署变量，不再新增一套 `ha.env`，避免同一信息分散在两个文件里。

文件格式使用 properties，便于 shell、Go 和运维脚本共同读取：

```text
/opt/omc/ha/omc.properties
host_state=BACKUP
omc_deploy_mode=ha
preferred_master=true

vip1=172.21.175.241
vip1_cidr=172.21.175.241/24
vip_dev=bond0
gateway=172.21.175.1

local_ip_addr=172.21.175.101
peer_ip_addr=172.21.175.102

# 可选：两机有直连/独立主备链路时填写；不填则内部通信回退走 local_ip_addr / peer_ip_addr
ha_link_dev=eth1
local_ha_link_ip_addr=172.21.176.101
peer_ha_link_ip_addr=172.21.176.102

ha_ssh_port=2222
ha_auto_failover=true
ha_nopreempt=true
ha_advert_int=10
ha_check_interval=20
ha_check_fall=2
ha_check_rise=2
```

部署模式只使用可读字符串，不支持数字模式值：`standalone` 表示单机，`ha` 表示主备，`ha_upgrade` 表示单机升级主备。

初始状态规则：两台机器安装完成前都写 `host_state=BACKUP`，不能预先写 `MASTER`。`preferred_master=true` 只表示“这台机器优先抢主”，不等于当前运行态主机；真正能变成 `MASTER` 的条件是 keepalived 已持有 VIP，且 `xomc-ha-manager promote` 完成数据层提升、业务进程启动和健康检查后原子写入 `host_state=MASTER`。

命名规则：`local_*` 表示本机，`peer_*` 表示对端节点。不使用 `slave_*` 这类字段名，因为切换后“对端”不一定是备机，容易误导配置和排障。

双网卡规则：`vip_dev` 是业务网卡，负责 VIP、基站接入、浏览器访问、北向接口和 keepalived VRRP；`ha_link_*` 是可选主备链路配置，两机有直连网卡或独立同步网络时填写，负责 PostgreSQL/TimescaleDB 复制、Redis replication、MinIO mirror、SSH 运维通道和辅助心跳。未配置 `ha_link_*` 时，主备内部通信和心跳回退使用 `local_ip_addr / peer_ip_addr`。直连主备链路通常没有网关，不能把 `gateway` 填成这条直连链路上的地址；本设计只保留 `gateway` 一个字段，它的取值应是业务网侧稳定可达的上游网关、核心交换机管理地址或独立 witness IP，用于判断“是不是本机业务网络坏了”。

硬规则：`/opt/omc/ha/omc.properties` 是本机文件，不参与 unison/rsync 双向同步。两机共同配置由安装脚本在两侧分别生成；`host_state` 只允许 `xomc-ha-manager` 原子写入。

业务进程启停以 `/opt/omc/ha/omc.properties` 中的 `host_state` 和 `xomc-ha-manager` 编排结果为准。BACKUP 不启动业务容器；业务代码不应各自散落解析 `omc.properties`。

## 4. VIP 与切换入口

继续使用 keepalived，因为它成熟、简单、现场可排障。

keepalived 配置原则：

- 两台都配置为 `BACKUP`，通过 priority 区分初始主备。
- `nopreempt` 默认开启，避免原主恢复后自动抢回。
- VRRP 单播，避免客户网络禁组播。
- 默认优先避免频繁切换，而不是追求几秒内快速切换：`advert_int=10`，业务健康检查 `interval=20`，连续失败 2 次才判故障，连续恢复 2 次才认为稳定。
- `ha_auto_failover=true` 允许自动切换，但不是探针失败就切；必须由多路心跳、VIP 状态、数据层健康和冷却窗口共同确认主节点确实异常后，才允许升主。
- `track_interface` 监控业务网卡。
- notify 钩子只调用 `xomc-ha-manager`，不要把复杂逻辑写进 shell。

参数含义：

| 参数 | 默认值 | 说明 |
|---|---:|---|
| `ha_nopreempt` | `true` | 旧主恢复后不自动抢回，避免来回切换 |
| `ha_auto_failover` | `true` | 允许确认主节点异常后自动升主；不满足门禁时只告警不切换 |
| `ha_advert_int` | `10` | VRRP 心跳 10 秒一次 |
| `ha_check_interval` | `20` | 业务健康检查 20 秒一次 |
| `ha_check_fall` | `2` | 连续 2 次失败才触发故障判断 |
| `ha_check_rise` | `2` | 连续 2 次恢复才认为稳定 |

notify 映射：

```text
notify_master -> xomc-ha-manager candidate --reason keepalived
notify_backup -> xomc-ha-manager demote --reason keepalived
notify_fault  -> xomc-ha-manager fault --reason keepalived
notify_stop   -> xomc-ha-manager stop --reason keepalived
```

`candidate` 不是无条件升主：它只表示 keepalived 认为本机可能接管。`xomc-ha-manager` 必须继续做升主门禁检查；只有 `ha_auto_failover=true` 且所有门禁通过，才继续执行 promote。任一门禁不满足时，只记录候选状态并上报告警，业务服务保持停止。

换句话说，keepalived 只负责“VIP 到本机”这个信号，不直接启动 xomc 业务：

```text
keepalived 拿到 VIP
  -> notify_master
  -> xomc-ha-manager candidate --reason keepalived
  -> xomc-ha-manager 校验主节点异常、脑裂风险和数据层状态
  -> 门禁全过后才 promote 数据层并启动 app/acs/worker/web
```

健康检查：

```text
vrrp_script chk_xomc {
  script "/usr/local/bin/xomc-ha-manager check"
  interval 20
  fall 2
  rise 2
  weight -30
}
```

`check` 必须按本机角色执行不同检查，不能用 MASTER 标准判断 BACKUP 失败。

MASTER 检查：

- 本机是否持有 VIP。
- `app / acs / worker / web` 是否 ready。
- `postgres` 是否 primary，`postgres-tsdb` 是否 primary。
- Redis 是否 master。
- MinIO 是否可写。
- NATS JetStream 是否 ready。
- 业务网对端、可选主备链路对端、业务网 gateway/witness 心跳是否正常。

BACKUP 检查：

- 本机不持有 VIP。
- `app / acs / worker / web` 已停止，且不会被自动拉起。
- `postgres` 和 `postgres-tsdb` 是 standby，复制源指向当前 MASTER。
- Redis 是 replica，复制源指向当前 MASTER。
- MinIO 同步方向为当前 MASTER 到本机。
- 本机磁盘、数据目录、license、主备链路和 gateway/witness 状态满足接管门禁。

双网卡场景的自动切换判定：

| 业务网对端 | 主备链路对端 | gateway/witness | 处理 |
|---|---|---|---|
| 通 | 通 | 通 | 正常，不切换 |
| 不通 | 通 | 通 | 对端业务网或 keepalived 可能异常；只告警，不自动切换，除非对端 ha-manager 明确已降备/停业务 |
| 不通 | 不通 | 通 | 对端整机或双链路异常概率高；`ha_auto_failover=true` 且升主门禁全过时允许自动接管 |
| 通 | 不通 | 通 | 主备链路异常；告警并暂停自动数据重建，不因主备链路单独切换 |
| 不通 | 不通 | 不通 | 本机业务网络可能异常；禁止接管，主动降备/停业务 |

未配置 `ha_link_*` 时，上表“主备链路对端”视为不可用但不单独触发故障；自动切换主要依赖业务网对端、VIP 状态和 `gateway/witness`。

自动升主门禁必须全部满足：

1. `ha_auto_failover=true`。
2. 本机已持有 VIP。
3. 本机能 ping 通 `gateway/witness`。
4. 对端在业务网和主备链路上都不可达，或对端 ha-manager 明确返回已降备/已停业务。
5. PostgreSQL 和 TimescaleDB 备库状态健康，复制延迟在允许阈值内。
6. Redis replica、MinIO replication/mirror 状态满足现场设定的 RPO。
7. 最近一次切换距离当前时间超过冷却窗口，避免抖动反复切换。
8. 本机没有检测到磁盘满、数据目录只读、license 不可用等本地硬故障。

## 5. 数据层方案

### 5.1 PostgreSQL 主库

xomc 当前主库是 `postgres`，承载设备、用户、配置、任务、告警当前态等强一致业务数据。主备方案：

- 主机：PostgreSQL primary。
- 备机：standby 只读副本。
- 复制方式：streaming replication。
- 初始搭建：`pg_basebackup` 从主同步到备。
- 切换：备机执行 `pg_ctl promote` 或 SQL `SELECT pg_promote()`。
- 原主恢复：优先 `pg_rewind`，失败再全量 `pg_basebackup` 重建。

一致性建议：

- 默认使用异步复制，避免备机断链导致主业务写入被卡死；RPO 通常为秒级。
- xomc 业务接受故障窗口内几秒数据丢失，不建议开启同步复制；主备优先保证主业务不被备机断链拖慢或阻塞。
- 切换前 `xomc-ha-manager` 必须检查 `pg_is_in_recovery()`，禁止两个节点同时 primary。

新增运维命令：

```text
xomc-ha-manager pg status
xomc-ha-manager pg promote
xomc-ha-manager pg rewind --from <new-master>
xomc-ha-manager pg rebuild --from <new-master>
```

### 5.2 TimescaleDB 时序库

xomc 已经把 KPI/PM/告警历史/trace 拆到 `postgres-tsdb`。它必须独立主备，不能和主库混在一起。

方案：

- 和主库一样使用 streaming replication。
- 默认异步复制，因为 PM 原始文件保存在 MinIO，部分时序数据可以从对象文件重建。
- 切换顺序必须在业务服务启动前完成：先 promote 主库，再 promote TSDB，再启动 app/acs/worker。
- 如果 TSDB promote 失败，判定本次升主失败，不启动 `app / acs / worker / web`；保持 BACKUP 或 FAULT 状态并告警，等待人工恢复。

### 5.3 Redis

旧 OMC 升主时直接 `flushall`。xomc 不能照搬，因为 Redis 中有：

- ACS 会话状态。
- 设备任务队列和 CWMP 映射。
- PM 聚合窗口。
- 分布式锁、节流、临时缓存。

方案：

- `redis-core` 和 `redis-pm` 都开启 AOF，主备之间配置 replica。
- 升主时执行 `REPLICAOF NO ONE`，备机 Redis 变为 master。
- 原主恢复后执行 `REPLICAOF <new-master> 6379` 重新挂到新主。
- 升主后只清理“易失状态前缀”，禁止全库 flush。

建议清理策略：

| Redis 数据 | 升主动作 |
|---|---|
| ACS 活跃 session、nonce、短 TTL admission | 清理或等待 TTL，自然让设备重新 Inform |
| 任务队列、task 详情、CWMP task 映射 | 不清理，依靠 TTL/幂等和 redelivery |
| PM 聚合窗口 | 不清理，避免 15 分钟窗口丢失；异常时走窗口重扫/补算 |
| 分布式锁、正在处理标记 | 清理特定 lock 前缀，避免旧主遗留锁阻塞 |
| 缓存类 key | 可清理，恢复后从 PostgreSQL 回填 |

这里需要新增一份 Redis Key 分类文档，并让 `redisx.Keys` 标注 key 的主备切换策略：`volatile_on_failover`、`durable`、`cache`、`lock`。

### 5.4 MinIO 对象存储

MinIO 承载 PM/MR 文件、固件、备份、日志包、导出文件。它是 xomc 里替代旧版 `/home/omc/data/file` 的关键数据面。

方案：

- 两台都运行 MinIO。
- 业务只写当前 MASTER 的 MinIO。
- 用 MinIO bucket replication 或 `mc mirror --watch` 把主机 bucket 同步到备机。
- 不使用 unison/rsync 热同步 MinIO 数据目录；直接同步 `/data` 容易破坏对象元数据和未完成 multipart upload 的一致性。
- `minio.public_endpoint` 使用当前 MASTER 的物理 IP，不使用 `VIP:9000`。切换期间系统整体不可访问，切换完成后用户重新访问页面或重新发起下载，由新主生成新的预签名 URL。
- `xomc-ha-manager` 在启动 `app` 前必须把 `sys_configs.storage.minio_public_endpoint` 更新为新 MASTER 的 MinIO 物理地址，例如 `172.21.175.102:9000`。
- 切换前已经签发的 MinIO 预签名 URL 不保证继续可用；这是可接受的切换窗口影响，不为它额外暴露 MinIO 到 VIP。
- 切换后新主继续使用同名 bucket；旧主恢复后从新主反向补齐。

需要同步的 bucket：

```text
pm-files
mr-files
firmware
config-backup
logs
reports
omc-exchange
ui-assets
```

### 5.5 NATS JetStream

NATS 是本方案最大的风险点。xomc 主备固定只有两台机器，不做 NATS JetStream 集群，也不做 NATS 数据目录热同步。JetStream 的 stream 元数据、consumer 位点和未刷盘消息不能用 rsync/unison 直接搬目录，否则容易损坏一致性。

处理策略：

- NATS 跟随当前 MASTER 运行。
- BACKUP 上的 NATS 只作为冷待命或随业务一起停止，不参与双机同步。
- 切换时新主启动本机 NATS，业务可靠性主要依靠 PostgreSQL 持久状态、Redis durable 任务队列、MinIO 原始文件和应用层恢复/重扫。
- 切换窗口内，仅存在于旧主 NATS、尚未落到 PostgreSQL/Redis/MinIO/业务 outbox 的少量事件允许丢失；新主不承诺 NATS 事件级重放。
- 禁止 live rsync/unison NATS JetStream 存储目录；如需保留旧主 NATS 数据，只能在停服状态下做人工排障快照。

根据当前 xomc 代码，切主后应恢复/重扫的是下面这些持久化状态，而不是旧主 NATS 消息：

| 数据类型 | 持久化来源 | 切主后动作 | 说明 |
| --- | --- | --- | --- |
| 设备任务队列 | PostgreSQL `device_tasks` + Redis task queue | worker 启动执行 `RestorePendingQueues`；设备下一次 Inform 执行 `RecoverPendingTasks` | 这是最重要的恢复项。`pending` 任务可从 PG 补灌回 Redis；已发出但未回包的 `sent` 任务在设备重连时按重试预算恢复。 |
| 通用业务 outbox | PostgreSQL `event_outbox` | worker 的 `event-outbox-relay` 继续认领并发布未完成事件 | 对已经进入 PG outbox 的事件，切主后可以继续投递。没有进入 outbox、只停在旧主 NATS 的事件不能恢复。 |
| PM 聚合窗口 | TimescaleDB/Redis 聚合状态 + `pm_aggregation_outbox`/`pm_aggregation_replay_sources` | worker 启动执行 `RestoreActiveWindows`，并由 PM outbox relay 继续发布/重投 | PM 聚合已有专门恢复逻辑，是重点保障对象。 |
| PM/MR/DataModel/Backup/Log 文件处理 | MinIO 对象 + PostgreSQL 处理记录 | 需要补“文件已落 MinIO 但处理事件丢失”的巡检/重扫任务 | 文件本体已在 MinIO，理论上可重扫；但不能依赖旧主 NATS 自动重放。开发时要明确哪些文件类型已有入库标记，哪些需要新增重扫器。 |
| 权限和系统配置热更新 | PostgreSQL 配置/RBAC 表 | app/worker/acs 启动全量加载；运行中靠周期刷新兜底 | `sys.casbin.policy.reload`、`sys.config.saved` 丢失只影响热刷新及时性，不应作为切主必须重放的数据。 |
| 设备在线、心跳、属性变化、告警上报 | PostgreSQL 中已提交的设备/告警状态 + 设备下一次 Inform | 已落库的状态随 PG 复制恢复；切换窗口内纯 NATS 事件允许丢失 | 下一次设备上报会自然修正在线、心跳、部分属性状态。切换窗口内新增告警如只在旧主 NATS 且未入库，本方案不承诺恢复。 |
| 北向推送、SSE、Trace 导出等通知类事件 | 部分有 PG outbox/DLQ，部分是瞬态事件 | 已入 outbox/DLQ 的继续重试；纯瞬态通知不恢复 | 切主期间系统对外不可用，通知类事件按最终状态可查询优先，不追求窗口内零丢失。 |

### 5.6 普通文件与配置同步

老 OMC 没有 MinIO，业务文件直接落宿主文件系统，所以依赖 unison 做文件主备。xomc 已经把大部分业务文件对象化，主备同步边界要收窄：

- PM/MR 文件、固件、配置备份、日志包、报表、导出文件：走 MinIO replication / `mc mirror`，不走 unison。
- MinIO 数据目录：禁止用 unison/rsync 热同步。
- 配置、证书、密钥、少量字典实例文件：优先由安装脚本在两侧生成；运行过程中可能变化的实例态小文件，可做受控单向同步或一致性校验。

仍需保持一致的宿主路径：

```text
/opt/omc/etc/                 # prod yaml、secrets.env、证书、license 公钥库
/opt/omc/data/                # 三库 XML、自定义字典等实例态；短期同步，长期建议迁入 MinIO 或数据库
```

建议：

- `/opt/omc/etc` 优先由安装脚本在两侧生成；升级后用 `xomc-ha-manager verify` 校验关键字段一致。
- `/opt/omc/data` 短期可用 rsync 单向同步或 lsyncd；不建议双向 unison，避免两侧同时修改后产生冲突。
- 长期把自定义字典、导入文件等实例态迁到 MinIO 或 PostgreSQL，让主备复制回到数据层自身机制。
- secrets 必须两机一致，尤其 `OMCGO_JWT_SECRET`、`OMC_SHARED_SECRET`、MinIO 账号、PG 复制账号。
- `/opt/omc/ha/omc.properties` 同时包含主备规划和本机状态，不能被对端同步覆盖；如需对比两侧规划一致性，由 `xomc-ha-manager verify` 读取两端文件后只做校验，不做自动覆盖。
- License 需要单独确认：当前 app 会读取宿主 `/sys` 做 MAC/UUID 绑定。主备两台机器硬件不同，license 必须支持双机绑定或 VIP/业务证书维度授权，否则切到备机会因 license 失效导致业务起不来。

## 6. 业务进程角色控制

旧版备机停 Java 业务。xomc 也应该这样做：

| 进程 | MASTER | BACKUP |
|---|---|---|
| web | 运行 | 停止 |
| app | 运行 | 停止 |
| acs | 运行 | 停止 |
| worker | 运行 | 停止，避免重复消费 NATS/Redis/PG 任务 |
| migrate | 只在安装/升级主侧执行 | 不自动执行，等数据同步 |
| monitoring | 两侧可运行 | 指标区分 node label |

BACKUP 上 `web/app/acs/worker` 均停止，不接业务流量、不接设备会话、不消费队列、不执行后台任务。切换时由 `xomc-ha-manager promote` 在新 MASTER 上完成数据角色确认后再启动这些业务进程。

BACKUP 停止的是业务进程，不是数据层服务。PostgreSQL/TimescaleDB、Redis、MinIO 等数据层服务在 BACKUP 上保持运行，用于持续接收当前 MASTER 的复制数据。NATS 不做双机同步，可随业务冷待命或由切换流程启动。

## 7. 切换流程

### 7.1 BACKUP 升 MASTER

`xomc-ha-manager promote` 步骤：

1. 获取本机切换锁 `/opt/omc/ha/state/switch.lock`，防止多个 promote 同时执行。
2. 做幂等判断：如果本机已经是 MASTER，且 VIP、PostgreSQL/TimescaleDB 主库状态、业务进程状态都正常，本次 promote 直接返回成功，不重复执行升主动作。
3. 判断来源：人工 `switchover/promote` 可继续；自动来源必须满足 `ha_auto_failover=true`。
4. 做主节点异常确认：同时检查业务网对端、主备链路对端、业务网 gateway/witness、对端 ha-manager 状态和最近心跳。若证据不足，只告警，不升主。
5. 脑裂检查：若对端不通且 gateway/witness 也不通，判定本机业务网络可能故障，本机主动降备，不接管。
6. 确认本机持有 VIP；如果没有 VIP，不启动业务。
7. 检查本机数据层可接管：PG/TSDB 备库健康、复制延迟在阈值内，Redis/MinIO 同步状态满足现场 RPO，license/磁盘/数据目录正常。
8. promote `postgres`，确认 `pg_is_in_recovery() = false`。
9. promote `postgres-tsdb`，确认 `pg_is_in_recovery() = false`。
10. promote `redis-core` 和 `redis-pm`。
11. 启动/切换 MinIO 为主写端，暂停反向覆盖任务。
12. 启动 NATS，确认 JetStream ready。
13. 清理 Redis 易失锁和会话类前缀，不清 durable task/PM 窗口。
14. 确保 `OMC_PUBLIC_HOST`、ACS upload/download base URL 指向 VIP；同时把 `sys_configs.storage.minio_public_endpoint` 更新为新 MASTER 的物理 IP。
15. 启动 `app / acs / worker / web`。
16. 调用 `/readyz`、Nginx、ACS upload endpoint、MinIO health、NATS health 做本机闭环检查。
17. 闭环检查通过后，原子写 `/opt/omc/ha/omc.properties`：`host_state=MASTER`。
18. 插入系统告警/审计：主备切换完成。
19. 释放切换锁。

失败处理：任一步失败都不继续启动业务。记录失败原因，保持 BACKUP 或 FAULT 状态，由人工恢复。

### 7.2 MASTER 降 BACKUP

`xomc-ha-manager demote` 步骤：

1. 获取切换锁。
2. 撤掉 VIP 或确认本机已经不持有 VIP。
3. 写 `host_state=BACKUP`。
4. 停止业务容器 `web / app / acs / worker`，先停入口 `web/acs`，再停后台 `worker/app`，并阻止这些业务容器被自动拉起。
5. 数据层容器不要直接停止：PostgreSQL/TimescaleDB、Redis、MinIO 需要保留现场，用于后续按新主方向重新加入复制。
6. 停止 NATS 写入或切到待命；NATS 不做双机同步，不能继续作为业务 NATS 使用。
7. 将 Redis 改为 replica，指向当前新主。
8. PostgreSQL/TSDB 不直接“降级成 standby”；PostgreSQL 没有安全在线 demote。原主恢复后必须走 `pg_rewind` 或全量重建。
9. 恢复 MinIO 从新主同步。
10. 释放锁。

### 7.3 原主恢复

原主恢复不能自动抢回主。流程：

1. 确认当前新主稳定。
2. 停原主业务服务。
3. 对 `postgres` 执行 `pg_rewind --source-server=<new-master>`；失败则 `pg_basebackup`。
4. 对 `postgres-tsdb` 执行同样流程。
5. Redis 改为 replica。
6. MinIO 从新主 mirror 补齐。
7. 启动 keepalived/ha-manager，但保持 BACKUP。
8. 如需计划回切，走人工 `xomc-ha-manager switchover --candidate <node>`。

### 7.4 切换后的复制方向调整

主备切换完成后，复制方向必须以当前 MASTER 为源重新确认：

```text
切换前：A(MASTER) -> B(BACKUP)
切换后：B(MASTER) -> A(BACKUP)
```

不能让旧 MASTER 恢复后继续以 primary/master 身份启动，否则会形成双主。各组件规则如下：

| 组件 | 新主接管时 | 旧主恢复后 |
|---|---|---|
| PostgreSQL `postgres` | 新主执行 promote，变成 writable primary | 旧主必须 `pg_rewind` 或 `pg_basebackup`，作为新主 standby 加回 |
| TimescaleDB `postgres-tsdb` | 同 PostgreSQL | 同 PostgreSQL |
| Redis `redis-core` / `redis-pm` | 新主执行 `REPLICAOF NO ONE` | 旧主执行 `REPLICAOF <new-master> 6379`，作为 replica 加回 |
| MinIO | 新主成为主写端，暂停旧方向同步 | 旧主恢复后从新主 mirror/replication 补齐，方向改为新主到旧主 |
| NATS | 不做双机同步，随当前 MASTER 启动 | 旧主恢复后不反向同步 JetStream 数据目录 |

`xomc-ha-manager status` 必须展示当前复制方向和落后量；`xomc-ha-manager verify` 必须检查复制源是否为当前 MASTER，发现方向反了要告警并禁止自动切换。

## 8. 安装与发布包改造

在 `deployments/release/bundle/deploy/` 增加：

```text
ha/
├── docker-compose.ha.yml
├── keepalived.conf.tmpl
├── omc.properties.example
├── ha-manager-entrypoint.sh
├── pg-replication-lib.sh
├── redis-replication-lib.sh
├── minio-replication-lib.sh
├── nats-standby-lib.sh
└── healthcheck-ha.sh
```

离线部署包必须提供一份可直接复制修改的示例配置：

```text
deploy/ha/omc.properties.example
```

安装人员先复制到运行目录，再按现场 IP 和网卡修改：

```text
cp deploy/ha/omc.properties.example /opt/omc/ha/omc.properties
vi /opt/omc/ha/omc.properties
```

`omc.properties.example` 里要保留注释，明确哪些字段必填、哪些字段可选、两台机器哪些字段要相反。`host_state` 示例固定为 `BACKUP`，禁止示例里出现 `MASTER`。

统一使用现有主入口 `install.sh`，同时支持单机和主备部署模式。`install.sh` 不再重复接收 VIP、网卡、对端 IP 等 HA 参数，主备参数统一从 `/opt/omc/ha/omc.properties` 读取；脚本只保留一个可选配置路径参数，用于非标准路径调试：

```text
install.sh --mode standalone
install.sh --mode ha --step prepare [--ha-config /opt/omc/ha/omc.properties]
install.sh --mode ha --step init-primary [--ha-config /opt/omc/ha/omc.properties]
install.sh --mode ha --step init-backup [--ha-config /opt/omc/ha/omc.properties]
```

`--mode standalone` 走现有单机安装流程；`--mode ha` 进入主备安装流程。未传 `--ha-config` 时，默认读取 `/opt/omc/ha/omc.properties`。安装脚本负责校验必填字段、渲染 keepalived/systemd/compose 配置，并拒绝使用命令行参数覆盖 HA 规划字段，避免出现两个配置源。

安装顺序采用阶段式执行。大步骤按顺序推进，上一阶段未完成不要进入下一阶段；其中“两台机器本机准备”类步骤可以并行，但必须两台都成功后再继续。

1. 两台都安装相同版本发布包和相同镜像。
2. 两台都执行 `cp deploy/ha/omc.properties.example /opt/omc/ha/omc.properties`，再按现场规划修改。
3. 两台都执行 `install.sh --mode ha --step prepare`。此阶段只做本机准备：创建目录、渲染配置、校验 `omc.properties`、加载镜像、生成 systemd/compose/keepalived 配置；两台可以并行执行。
4. 优先主机执行 `install.sh --mode ha --step init-primary`，初始化 PG/TSDB/Redis/MinIO/NATS 数据层。
5. 确认主机数据层初始化完成且可被备机访问。
6. 备机执行 `install.sh --mode ha --step init-backup`，通过 `pg_basebackup`、Redis replica、MinIO replication 初始化同步数据。
7. 确认备机数据层已进入同步状态。
8. 两侧启动 keepalived/ha-manager。
9. VIP 只在 MASTER 持有。
10. 执行主备状态检查和切换演练。

## 9. 运维命令设计

统一提供 `xomc-ha-manager`：

```text
xomc-ha-manager status
xomc-ha-manager check
xomc-ha-manager promote --reason manual|keepalived|healthcheck
xomc-ha-manager demote --reason manual|keepalived|healthcheck
xomc-ha-manager switchover --candidate <node>
xomc-ha-manager rebuild-standby --from <node>
xomc-ha-manager verify
xomc-ha-manager redis cleanup-volatile --dry-run
xomc-ha-manager minio reconcile --from <node>
```

`status` 输出必须适合人读，也必须支持 JSON：

```text
xomc-ha-manager status --json
```

## 10. 验证方案

### 10.1 单项验证

- keepalived：拔掉 MASTER 网卡或停 keepalived，VIP 漂移到 BACKUP。
- PostgreSQL：主库写入一条设备/告警数据，确认备库可读；切换后新主可写。
- TSDB：写入一条 PM/trace 数据，确认备库复制；切换后新主可写。
- Redis：写入 task/session/PM window 测试 key，确认 replica 同步；升主后 durable key 不丢。
- MinIO：上传 PM/MR/firmware/config-backup 对象，确认备机 bucket 可见。
- NATS：停主 NATS 后确认业务降级路径和恢复路径。
- License：在备机 MASTER 状态启动 app，确认 license 校验通过。

### 10.2 切换演练

至少覆盖：

1. 手工 switchover：主备正常时人工切换。
2. MASTER 断电：模拟主机完全不可达。
3. MASTER 业务异常：停 app/acs/worker/web，连续健康检查失败并确认主节点不可服务后，才触发自动倒换。
4. 主库异常：停 postgres，触发倒换。
5. 数据面断链：业务网正常、数据同步网异常，不应盲目切换。
6. 心跳脑裂：对端不通且网关不通，本机必须降备，不得接管。
7. 原主恢复：确认不会自动抢回，必须以 BACKUP 加入。

### 10.3 验收指标

| 指标 | 目标 |
|---|---|
| VIP 漂移时间 | 30 秒内 |
| 业务恢复 RTO | 目标 2 分钟内，最终以真实演练结果确认 |
| PostgreSQL RPO | 异步复制秒级，目标按现场配置控制在几秒内 |
| TSDB RPO | 异步复制秒级到分钟级，允许用 MinIO 原始文件补算 |
| Redis durable key 丢失 | 目标不丢；异步复制窗口内仍可能丢失少量最近写入，关键任务以 PostgreSQL 恢复为准 |
| MinIO 对象复制延迟 | 正常小于 60 秒，按现场带宽调整 |
| 脑裂保护 | 必须通过，不通过不得上线自动切换 |

## 11. 主要风险与决策点

1. 两节点没有真正仲裁，自动切换有脑裂风险。本方案必须保守：网关不可达时宁可停服务，不抢主。
2. NATS JetStream 在两机主备下不做强一致同步；切换窗口内尚未落到 DB/Redis/MinIO/业务 outbox 的少量事件允许丢失，已持久化的数据通过恢复/重扫处理。
3. Redis 不能照旧版全库 flush。必须先完成 key 分类，否则会丢任务或 PM 聚合窗口。
4. License 可能绑定物理机。必须确认双机授权，否则主备功能技术上完成也无法切到备机运行。
5. `OMC_PUBLIC_HOST`、ACS upload/download URL 必须统一指向 VIP；MinIO `public_endpoint` 使用当前 MASTER 物理 IP，并由 `xomc-ha-manager` 在业务启动前切到新主 IP。
6. PostgreSQL/TimescaleDB 不采用同步复制作为默认方案；业务接受几秒 RPO，优先避免备机断链拖慢或阻塞主业务。
7. TSDB 数据量大，`pg_basebackup`/`pg_rewind` 时间可能很长，需要压测恢复耗时。
8. MinIO 对象复制延迟会影响 PM/MR 补算完整性，需要在页面/告警中显示复制滞后。
9. 北向 Socket/SNMP 端口直接由 app 暴露，必须只在 MASTER 可用，否则上游 OSS 可能连到双端。
10. 监控和运维命令必须区分“服务活着”和“服务可作为 MASTER 写入”，不能只看容器 running。

## 12. 设计边界

本方案包含：

- 支持两机主备安装。
- 支持 VIP 漂移。
- 支持 PG/TSDB/Redis/MinIO 数据同步。
- 支持 BACKUP 升 MASTER。
- 支持原主恢复为 BACKUP。
- 支持业务容器按角色启停，备机不跑业务。
- 支持手工切换和保守自动切换。

本方案不包含：

- 具体代码、脚本、配置模板和部署包实现细节。
- NATS 在两机主备下 0 丢失自动切换。
- 原主恢复后自动抢回。
- 多活写入。
- 两机主备下的强一致自动仲裁。

这个边界最接近旧 OMC 的主备机制，也最适合 xomc 当前 Docker Compose 交付形态：主备关系清楚，现场可理解、可排障、可恢复。
