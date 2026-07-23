# Release 热点服务 CPU 比例调整设计

## 背景

32 核目标机在 10000 基站压测中，主库和 TimescaleDB 使用 4 核配额时分别持续接近
406% 和 376% CPU，已触及容器上限。后续分阶段将主库、TimescaleDB、worker 调整为
10/10/3、10/10/5，最终验证到 10/16/8；各阶段均未出现新的 flush 失败、超时、
重投、OOM 或 DLQ。

现有 release 资源规划器的 medium 档仍输出 6/4/3，Compose 未注入
`resources.env` 时的兜底值则是 6/4/5，均未固化本次验证结果。

## 方案

- release medium/large 档按宿主总核数动态计算：主库三分之一、TimescaleDB 二分之一、
  worker 四分之一，向下取整且最低 2 核。
- release Compose 的 32 核兜底默认值同步为 10/16/8，保证未运行规划脚本时行为一致。
- worker 内存维持 1 GiB（`GOMEMLIMIT=921MiB`）；8 核实测峰值约 325 MiB，无需扩到 2 GiB。
- small 档和本地开发 Compose 保持不变。

## 验收

- 32 核、32 GiB 探测覆盖运行规划器后，主库/TimescaleDB/worker 输出 10/16/8。
- 64 核 large 档输出 21/32/16，验证三种比例规则而非固定字面量。
- release Compose 中三个变量的兜底默认值为 10/16/8。
- 现有 release shell 测试通过，Compose 配置能够成功渲染。
