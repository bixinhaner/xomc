# OMC 数据盘规划 Runbook

> **任务编号**：#172（从 #169「容量/规模化前置」拆出：数据盘外置大半是 ops/runbook，非代码）
> **首次落地**：2026-06-12
> **维护方**：运维与可观测性专家 Owner
> **关联文件**：
> - 资源规划方案：[`deployments/release/bundle/deploy/RESOURCE-PLANNING.md`](../../deployments/release/bundle/deploy/RESOURCE-PLANNING.md)
> - Docker 安装脚本（已含 data-root 切盘逻辑）：`deployments/release/bundle/docker/install-docker.sh`
> - dev 栈卷定义：`deployments/docker/docker-compose.yml`（`pgdata` / `miniodata` named volume）

---

## 0. 这篇 runbook 解决什么

`pgdata`（PostgreSQL/TimescaleDB 时序数据）和 `miniodata`（PM/MR 原始文件）是 docker
**named volume**，默认落在 `/var/lib/docker/volumes`。时序数据规模很大——10 万基站 ≈ 500GB、
100 万基站 ≈ 5TB——**绝不能落在根盘 / overlay 文件系统上**：撑爆根盘会拖垮整机（dockerd、
日志、系统全部受影响），且根盘通常不是高 IOPS 介质，PG 时序写入会成为瓶颈。

本文给运维**部署前**的两条落盘路径（二选一）+ 容量估算公式，确保 PG/MinIO 数据落到独立数据盘。

> **本质是运维部署决策 + 文档，不需要按卷改 compose。** named volume 的物理落点由
> **docker data-root** 决定（见 §1），已经能把整个数据盘外置；只有当你想把 PG 和 MinIO
> 拆到**两块不同的盘**时，才需要 §2 的 bind-mount 覆盖写法。两者**不要同时用**（重复且互相打架）。

---

## 1. 方案 A（推荐）：部署前把 docker data-root 指向独立盘

最省事、覆盖面最广的做法：把**整个 docker 数据目录**搬到独立数据盘，`pgdata` / `miniodata`
以及镜像层、容器层全部跟着落到该盘。

### 1.1 选盘原则

| 数据 | 介质要求 | 原因 |
|------|---------|------|
| **PostgreSQL / TimescaleDB**（`pgdata`） | **NVMe SSD**（强烈建议） | 时序高频写入 + 聚合扫描，对 IOPS / 延迟敏感，机械盘会成瓶颈 |
| **MinIO**（`miniodata`） | SATA SSD 或大容量 HDD 均可 | 顺序读写为主，瓶颈在容量与吞吐而非随机 IOPS |

> 若 PG 与 MinIO 共用一块盘，**按 PG 的标准选 NVMe**（让 MinIO 沾光，不要反过来让 PG 迁就 HDD）。

### 1.2 install-docker.sh 已内建的切盘逻辑

`install-docker.sh` 在安装时会检测 `/var` 可用空间：**`/var` 可用 < 15G 时**会打 warn 并交互
询问是否把数据目录切到 `/home`，确认后写入 `/etc/docker/daemon.json` 的 `data-root`
（`/home/docker-data`）+ containerd 的 `--root`（`/home/containerd-data`）。非交互模式默认按 Y 处理。

也就是说，只要把独立数据盘挂在 `/home`（或让 `/var` 本身就在大盘上），安装脚本会自动落对地方。

### 1.3 手动指定 data-root（数据盘挂在非 /home 路径时）

若独立盘挂载点不是 `/home`（例如挂在 `/data`），在**首次 `docker compose up` 之前**手动设
data-root：

```bash
# 1) 确认独立盘已挂载且空间充足（示例挂在 /data）
df -h /data

# 2) 写入 /etc/docker/daemon.json 的 data-root（保留其它已有键）
#    若文件已存在，请用 python3/jq merge，勿整文件覆盖丢掉 bip / mirror 等配置
sudo mkdir -p /data/docker
cat /etc/docker/daemon.json   # 先看现有内容
# 例（仅当原本为空时可直接写）：
echo '{ "data-root": "/data/docker" }' | sudo tee /etc/docker/daemon.json

# 3) 重启 docker 使 data-root 生效（务必在还没有业务卷之前做，否则旧卷不会自动迁移）
sudo systemctl restart docker
docker info | grep "Docker Root Dir"   # 确认指向 /data/docker

# 4) 之后照常部署，pgdata / miniodata 会落到 /data/docker/volumes 下
```

> ⚠️ **务必在创建业务卷之前切**。data-root 改动不会自动搬迁已存在的卷；若已经跑过数据，需先
> 停栈、`rsync` 旧 data-root 到新盘、再改 daemon.json，否则改完会"丢数据"（其实是指到了空目录）。

---

## 2. 方案 B：用 compose override 把 pgdata/miniodata bind-mount 到数据盘

仅在**需要把 PG 和 MinIO 拆到两块不同的盘**（如 PG 走 NVMe、MinIO 走大容量 HDD）时才用。
用 docker compose 的 override 机制把 named volume 改成指向宿主机绝对路径的 bind-mount，
**不改主 compose 文件**。

新建 `docker-compose.datadisk.yml`（与主 compose 同目录），在 `up` 时用 `-f` 追加：

```yaml
# docker-compose.datadisk.yml —— 数据盘外置覆盖层（不改主 compose）
# 把 named volume 改成 driver_opts bind，指向各自的独立盘挂载点
volumes:
  pgdata:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /mnt/nvme/omc/pgdata        # ← PG 落 NVMe
  miniodata:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /mnt/hdd/omc/miniodata      # ← MinIO 落大容量 HDD
```

使用：

```bash
# 0) 先在两块盘上建好目录（PG 目录属主需匹配容器内 postgres uid）
sudo mkdir -p /mnt/nvme/omc/pgdata /mnt/hdd/omc/miniodata

# 1) up 时把覆盖层追加在主 compose 之后（后者覆盖前者的同名 volume 定义）
docker compose \
  -f deployments/docker/docker-compose.yml \
  -f deployments/docker/docker-compose.datadisk.yml \
  up -d

# 2) 验证卷确实指向了 bind 设备
docker volume inspect <栈前缀>_pgdata | grep -A3 Options
```

> ⚠️ **与方案 A 二选一**。若已经用方案 A 把整个 data-root 外置，再叠方案 B 就是把 bind 指向
> 「外置盘上的 docker volume」之外的另一处，路径重复、语义混乱。**先决定走 A 还是 B，不要叠加。**

---

## 3. 容量估算公式

部署前据此估算两块盘各需多大，再回到 §1/§2 选盘。规模口径与
[`RESOURCE-PLANNING.md`](../../deployments/release/bundle/deploy/RESOURCE-PLANNING.md) §4「档位与最低配置」
对齐（small=100k / medium=300-500k / large=1M）。

### 3.1 PostgreSQL / TimescaleDB（`pgdata`）

时序表是 pgdata 的大头，量级正比于「**每周期落多少行 × 保留多少周期**」：

```
PG 时序容量 ≈ 设备数 × 单设备计数器数 × (保留期 / 采集周期) × 每行字节
```

- **设备数**：规划规模（100k / 300k / 1M）。
- **单设备计数器数**：一个基站每个采集周期上报的 KPI/计数器条目数（按制式与启用指标集而定，
  量级通常数百到上千）。
- **保留期 / 采集周期**：保留多少个采集周期的明细。PM 采集周期常见 15 分钟（每天 96 个周期）；
  原始明细保留期由 PM 模块保留策略控制（见 `omcgo/internal/pm` 的 retention 配置）。
- **每行字节**：时序行 + 索引的平均落盘字节（含 TimescaleDB chunk 开销，估算时给 1.5–2× 的余量）。

> 经验校准：本仓 §0 给出的量级是 **100k ≈ 500GB、1M ≈ 5TB**。先用上式按你的实际计数器数 /
> 保留期算，再与这个量级对照取较大者，并预留 30% 增长空间。

### 3.2 MinIO（`miniodata`）—— PM/MR 原始文件

MinIO 存 PM/MR 上报的**原始文件**，受 **14 天生命周期**封顶（原始文件桶 14 天过期已由 #169
主 PR 落地，超期自动清理）：

```
MinIO 容量 ≈ 设备数 × 每设备每天文件数 × 单文件平均大小 × 14d
```

- **每设备每天文件数**：每个基站每天产生的 PM/MR 文件数（按采集周期与文件粒度，常见每天数十个）。
- **单文件平均大小**：压缩后的 XML/原始文件均值。
- **× 14d**：由生命周期过期上界封顶——这正是 MinIO 不会无界增长的原因，估算时**只需算 14 天的稳态量**，
  不必按"永久累积"算。

> 若调整了生命周期天数，公式里的 `14d` 同步换成新值；保持公式与实际 ILM 策略一致。

### 3.3 估完之后

把 §3.1 / §3.2 的结果对照
[`RESOURCE-PLANNING.md`](../../deployments/release/bundle/deploy/RESOURCE-PLANNING.md) §4 的档位选盘容量，
再回 §1（整盘外置）或 §2（PG/MinIO 分盘）落地，并各预留 ≥ 30% 增长 buffer。

---

## 4. 与 RESOURCE-PLANNING.md 的关系

| 关注点 | 本 runbook（数据盘规划） | [RESOURCE-PLANNING.md](../../deployments/release/bundle/deploy/RESOURCE-PLANNING.md) |
|--------|------------------------|----------------------------------|
| **盘**（容量 / 介质 / 落点） | ✅ 本文主题：data-root / bind-mount / 容量公式 | 仅在 §4 档位里附带最低配磁盘建议 |
| **CPU / 内存限额** | 不涉及 | ✅ 主题：floor-first 分配、GOMEMLIMIT、PG 调优 |
| **最低配置门禁** | 引用其档位口径 | ✅ `空闲预算 < Σfloor` 即 die + 建议最低配 |

**一句话**：先用 RESOURCE-PLANNING.md 定**机器档位与内存/CPU 限额**，再用本 runbook 定
**数据盘容量与落点**；两者配套，缺一不可。
