# bundle/docker — Docker 引擎离线安装

本目录只放**静态模板** `install-docker.sh`（随仓库入库），它会被 `build-release.sh`
拷入每个交付包的 `docker/` 目录，供运维侧在**无公网**的内网离线安装 Docker。

## Docker 引擎二进制包从哪来

Docker 静态二进制包（`docker-<版本>.tgz`）**不放在本目录**，由下载工具按架构
下载到 `deployments/release/docker-cache/<arch>/`：

```bash
cd deployments/release
./download-docker.sh           # 按 release.conf 的 DOCKER_VERSION 下载 amd64 + arm64
```

- 下载版本、地址由 `release.conf` 的 `DOCKER_VERSION` / `DOCKER_URL_TEMPLATE` 控制。
- `build-release.sh` 打包时，按目标架构从 `docker-cache/<arch>/` 取对应的 tgz，
  连同本目录的 `install-docker.sh` 一起放进交付包 `docker/`。

## 版本选择

- `release.conf` 的 `DOCKER_VERSION` 是**人为选定**的具体版本，不取 latest。
- 选近期 stable 版本（满足部署要求 ≥ 20.10），固定下来，多批次交付保持一致。

## 若目标机已装 Docker

可不下载；`build-release.sh` 会告警但继续，运维侧跳过"离线安装 Docker"即可。

## Docker 网段规划（唯一输入：DOCKER_BIP）

`DOCKER_BIP` 是客户侧唯一需要规划的 Docker 网桥地址。生产安装不会替客户选择
默认网段；如果没有配置 `DOCKER_BIP`，安装会停止并要求先完成网络规划。客户应根据
实际业务网、路由和安全策略选择不冲突的 IPv4 网段；当前实现仍拒绝 `172.0.0.0/8`，
以避免再次产生公司内网冲突。

规划库会按 `DOCKER_BIP` 所在网络块连续派生：

| 用途 | 派生规则 | 生效位置 |
|------|---------|---------|
| `docker0` | `DOCKER_BIP` 所在网络块，网关为 `DOCKER_BIP` 地址 | `/etc/docker/daemon.json` 的 `bip` |
| Compose 业务网 | 下一个同等大小的网络块 | `DOCKER_COMPOSE_SUBNET` |
| Docker 自动地址池 | 再下一个同等大小的网络块；每个自动网络至少 `/24` | `default-address-pools` |
| 测试网络 | 再下一个同等大小的网络块 | `DOCKER_TEST_SUBNET` |
| 迁移基线网络 | 再下一个同等大小的网络块 | `DOCKER_MIGRATION_SUBNET` |

例如客户规划 `10.240.0.1/16`，系统自动得到：

```text
docker0       10.240.0.0/16
Compose 业务网 10.241.0.0/16
自动地址池     10.242.0.0/16（每个网络 /24）
测试网络       10.243.0.0/16
迁移基线网络   10.244.0.0/16
```

不要手工填写 `DOCKER_COMPOSE_SUBNET`、`DOCKER_ADDR_POOL_BASE` 或
`DOCKER_TEST_SUBNET`、`DOCKER_MIGRATION_SUBNET`；这些值必须由 `DOCKER_BIP` 计算，防止
网段漂移或重叠。

### 新部署规划

在交付包的 `deploy/.env` 中只填写：

```bash
DOCKER_BIP=10.240.0.1/16
```

然后执行安装脚本。安装脚本会计算派生值并写回 `.env`，所有 Compose 和后续
`svc.sh`、`healthcheck.sh` 操作都会复用该规划。

### 已装 Docker 修改规划

```bash
export DOCKER_BIP=10.240.0.1/16
cd /opt/omc/infra/docker
sudo -E bash install-docker.sh --skip-if-installed --no-mirror
```

修改 Docker 网桥前必须停止使用旧网段的容器。发布安装在执行网络门禁前会自动删除
无容器的规划外网络；有容器依赖的规划外网络不会被删除，必须先人工处理。

GPV NATS 验证脚本在 Linux Docker Engine 上默认使用 `--network host`，让宿主 Go
测试直接访问临时 NATS 端口；在 macOS 或 Docker Desktop 上自动切换为
`--network bridge` 并只发布 `127.0.0.1` 临时端口。平台探测异常时可显式指定：

```bash
# 在 OMC 交付包根目录执行（已安装环境可使用 /opt/omc/current）
NATS_DOCKER_NETWORK=bridge bash deploy/verify-gpv-nats.sh
```

仅支持 `host` 和 `bridge`，不会使用 `none` 启动 NATS 验证容器。

### 校验

```bash
cat /etc/docker/daemon.json
ip -4 route
docker network ls -q | xargs -r docker network inspect --format '{{.Name}} {{range .IPAM.Config}}{{.Subnet}} {{end}}'
```

所有 Docker bridge 都应属于当前 `DOCKER_BIP` 派生计划；如果修改了 `DOCKER_BIP`，
必须重建旧 Compose 网络，单独修改 `docker-compose.yml` 不会改变已存在的 Docker 网络。
