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

> ⚠️ **已装 Docker 的机器也必须经过 173.x 网段断言**。`install-docker.sh
> --skip-if-installed` 会校验并在必要时更新 `/etc/docker/daemon.json`、重启 Docker；发布
> `install.sh` 也会在 Compose 启动前执行同一门禁。已有 172.x bridge 网络必须先停止并删除。

## Docker 网段规划（重要：避开公司 172.x 内网）

公司内网大量使用 `172.x` 段（如 `172.17`、`172.24`，且仍在扩张）。Docker **默认**
把 `docker0` 放在 `172.17.0.0/16`、自动创建的网络放在 `172.18+`，会与公司内网**撞段**：
宿主把 `172.17.0.0/16` 路由进 docker0，导致**公司 172.17 网段的电脑访问不了本机服务**
（回程被 docker0 劫持）。

为此 `install-docker.sh` 会把 Docker 全部网络迁到公司约定的 **`173.x`** 段（避开 172），
三段互不重叠：

| 用途 | 网段 | 由谁配置 |
|------|------|---------|
| `docker0` 默认网桥（`bip`） | `173.17.0.0/16` | `install-docker.sh` 写 `daemon.json` |
| `omcgo-net`（compose 业务网，固定） | `173.18.0.0/16` | `docker-compose.infra.yml` 锁定 |
| 其它/未来自动创建的网络（池） | `173.19.0.0/16`（每网络 /24） | `install-docker.sh` 写 `daemon.json` |

> 说明：`173.x` 是公网地址族，这里**沿用公司既有内部系统的约定**（内网"借用"）。
> 前提是本机与内网都不会去访问真实的 `173.17~173.19` 公网目的地。

### 调整网段

- **改 `install-docker.sh`**：顶部三个变量 `DOCKER_BIP` / `DOCKER_ADDR_POOL_BASE` /
  `DOCKER_ADDR_POOL_SIZE`，或部署前用同名环境变量覆盖：
  ```bash
  sudo DOCKER_BIP=173.17.0.1/16 DOCKER_ADDR_POOL_BASE=173.19.0.0/16 bash install-docker.sh
  ```
- **改 `omcgo-net`**：编辑 `deploy/docker-compose.infra.yml` 的 `networks.omcgo-net.ipam`
  （app/web/monitoring 三个 compose 不重复声明 subnet，合并时以 infra 为准，只改这一处）。
- 三段务必互不重叠，且都避开公司在用的网段。

### 已装机器手动套用（含当前测试机）

```bash
sudo cp /etc/docker/daemon.json /etc/docker/daemon.json.bak 2>/dev/null
# 用 python3 合并(保留 data-root / registry-mirrors 等键)
sudo python3 - <<'PY'
import json,os
p="/etc/docker/daemon.json"; d={}
if os.path.exists(p) and os.path.getsize(p):
    try: d=json.load(open(p))
    except: d={}
d["bip"]="173.17.0.1/16"
d["default-address-pools"]=[{"base":"173.19.0.0/16","size":24}]
json.dump(d,open(p,"w"),indent=2,ensure_ascii=False); open(p,"a").write("\n")
PY
cd <部署目录> && docker compose down          # 停栈(释放旧 172.x 网络)
sudo systemctl restart docker                 # 重建 docker0 到 173.17
docker network prune -f                        # 清掉残留的 172.x 旧网桥
docker compose up -d                           # omcgo-net 按 173.18 重建
# 校验:Docker 路由和网络均应为 173.x；若仍列出 172.x，先删除对应旧网络
ip -4 route
docker network ls -q | xargs -r docker network inspect --format '{{.Name}} {{range .IPAM.Config}}{{.Subnet}} {{end}}'
```
回滚：`sudo cp /etc/docker/daemon.json.bak /etc/docker/daemon.json && sudo systemctl restart docker`。
