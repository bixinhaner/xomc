# bundle/docker — Docker 引擎离线安装件

本目录的内容会被原样拷入每个交付包的 `docker/` 目录，供运维侧在**无公网**的
内网离线安装 Docker。

## 需要放入的文件（构建侧一次性准备）

| 文件 | 说明 | 获取方式 |
|------|------|---------|
| `install-docker.sh` | 离线安装脚本（已随仓库提供） | 仓库内已有 |
| `docker-<版本>.tgz` | Docker 静态二进制包 | 从 https://download.docker.com/linux/static/stable/ 下载对应架构 |

> Docker 静态二进制包**区分架构**：
> - amd64 → `x86_64/docker-<版本>.tgz`
> - arm64 → `aarch64/docker-<版本>.tgz`
>
> 若需双架构交付包，两个架构的 tgz 都要准备；`build-release.sh` 会把本目录整体
> 拷入交付包。也可在 `build-release.sh` 中按架构区分（当前实现为整体拷入，
> 简单起见可在 amd64 / arm64 构建前分别替换本目录内的 tgz）。

## Docker 版本选择（重要）

`install-docker.sh` **不自动下载、不取 latest**——它装的就是本目录内放着的那个
`docker-*.tgz`。版本是**人为选定**的，请遵循：

- 选用经测试的**近期 stable 版本**（满足部署方案要求 ≥ 20.10）。
- **固定一个具体版本号**，不要每批次随手抓最新；同一产品版本的多批次交付保持一致。
- 把所选 Docker 版本记入交付说明 / 版本台账，便于现场排错与复现。
- 本目录**有且仅有一个** `docker-*.tgz`——`install-docker.sh` 发现多个会直接报错
  （版本无法确定），发现 0 个也报错（不会联网兜底）。

## 若目标机已装 Docker

可不放 `docker-*.tgz`；`build-release.sh` 会告警但继续，运维侧跳过"离线安装
Docker"这一步即可。
