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
