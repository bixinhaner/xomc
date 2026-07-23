# Release 数据库 CPU 配额调整设计

## 背景

32 核目标机在 10000 基站压测中，主库和 TimescaleDB 使用 4 核配额时分别持续接近
406% 和 376% CPU，已触及容器上限。将主库、TimescaleDB、worker 调整为
10/10/3 核并重启后，PM 队列在约一分钟内从 3492 降到 0，期间未出现新的 flush
失败、超时或 DLQ。

现有 release 资源规划器的 medium 档仍输出 6/4/3，Compose 未注入
`resources.env` 时的兜底值则是 6/4/5，均未固化本次验证结果。

## 方案

- 仅调整 32 核 / 32 GiB 目标机对应的 release medium 档。
- `POSTGRES_CPUS`、`TSDB_CPUS` 分别设为 10，约为 32 核的三分之一并向下取整。
- `WORKER_CPUS` 设为实测值 3；本次队列已能清零，不再沿用此前未经本轮验证的 5 核兜底。
- release Compose 兜底默认值同步为 10/10/3，保证未运行规划脚本时行为一致。
- small/large 档和本地开发 Compose 保持不变，避免把 32 核压测结论外推到其他机型。

## 验收

- 32 核、32 GiB 探测覆盖运行规划器后，`resources.env` 输出 10/10/3。
- release Compose 中三个变量的兜底默认值为 10/10/3。
- 现有 release shell 测试通过，Compose 配置能够成功渲染。

