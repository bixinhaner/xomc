# Docker Registry Mirror 配置指南

> 解决在国内服务器上跑 `docker compose -f deployments/docker/docker-compose.yml up -d --build` 时
> 拉镜像超时 / metadata 加载失败的问题。
>
> 不动项目代码，全部在宿主机层面配置。

---

## 1. 谁需要看这份文档

如果你在新服务器跑 `docker compose up --build` 遇到下面任一现象：

```
ERROR [acs internal] load metadata for docker.io/library/alpine:3.19
ERROR [web internal] load metadata for docker.io/library/nginx:alpine
target acs: failed to solve: DeadlineExceeded: alpine:3.19: failed to resolve source metadata
dial tcp 98.159.108.61:443: i/o timeout
```

或：

```
failed to do request: Head "https://registry-1.docker.io/...": dial tcp ... i/o timeout
```

需要按本文档配置 Docker Hub 镜像源（registry mirror）。

> 这与 [`DOCKER_DNS_FIX.md`](./DOCKER_DNS_FIX.md) 不是同一类问题。DOCKER_DNS_FIX 解决的是
> 容器内 DNS 解析失败（错误形如 `dial tcp: lookup xxx on 53: i/o timeout`，端口 53），
> 本文档解决的是 docker.io 网络访问超时（端口 443）。

---

## 2. 项目当前的国内化现状

`deployments/docker/Dockerfile.*` 已经做了**部分**国内优化：

| 层 | 国内化处理 | 失败时报什么错 |
|---|---|---|
| alpine apk 包 | ✅ `ARG APK_MIRROR=mirrors.aliyun.com` 默认 | `apk add: temporary failure ...` |
| Go modules | ✅ `ARG GOPROXY=https://goproxy.cn,...` 默认 | `go: ... i/o timeout` |
| **基础镜像层**（`FROM alpine`/`golang`/`node`/`nginx`） | ❌ **未硬编码 mirror** | `failed to resolve source metadata for docker.io/...` |
| **BuildKit syntax frontend**（`# syntax=docker/dockerfile:1`） | ❌ **未硬编码 mirror** | `resolve image config for docker-image://docker.io/docker/dockerfile:1` |

**设计取舍**：基础镜像层 mirror 配置在**宿主机 docker 上**做，不绑定到代码里。这样：
- 项目代码干净（不绑定特定 mirror，外发友好）
- 不同环境用不同 mirror（开发用阿里、客户机房用内网 harbor）
- 代价：每台新服务器需按本文档配置一次（10 分钟）

---

## 3. 完整配置流程（4 步走通）

> ⚠️ **关键认知**：dockerd 和 BuildKit 是两个独立引擎，配置文件**不互通**。
> 必须**两份**都配，否则 `docker compose build` 仍超时。

### 3.1 配 daemon 的 mirror（影响 `docker pull` + dockerd 内置 builder）

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://docker.1panel.live",
    "https://hub.rat.dev",
    "https://docker.nju.edu.cn"
  ],
  "dns": ["223.5.5.5", "223.6.6.6", "114.114.114.114"],
  "dns-opts": ["timeout:2", "attempts:3"]
}
EOF

sudo systemctl daemon-reload
sudo systemctl restart docker
```

**验证**：

```bash
docker info 2>/dev/null | grep -A 6 "Registry Mirrors"
```

预期看到：

```
 Registry Mirrors:
  https://docker.m.daocloud.io/
  https://docker.1panel.live/
  https://hub.rat.dev/
  https://docker.nju.edu.cn/
```

**没看到**：daemon.json 没生效。检查：

```bash
sudo cat /etc/docker/daemon.json | python3 -m json.tool   # 验 JSON 合法
sudo journalctl -u docker -n 20 --no-pager | grep -iE "error|config"
```

### 3.2 配 BuildKit 的 mirror（影响 `docker compose build` / `docker buildx`）

`docker compose` v2 默认走 buildx → BuildKit。BuildKit 的 **`docker-container` driver** 才会读
`/etc/buildkit/buildkitd.toml`；**默认 `docker` driver 不读这份文件**。

```bash
sudo mkdir -p /etc/buildkit
sudo tee /etc/buildkit/buildkitd.toml >/dev/null <<'EOF'
[registry."docker.io"]
  mirrors = [
    "docker.m.daocloud.io",
    "docker.1panel.live",
    "hub.rat.dev",
    "docker.nju.edu.cn"
  ]
EOF
```

### 3.3 创建容器化 buildx builder 并启用

```bash
# 删旧 builder（如已存在同名）
docker buildx rm omc-builder 2>/dev/null || true

# 用 docker-container driver 建新 builder，加载 buildkitd.toml
docker buildx create --name omc-builder \
    --driver docker-container \
    --config /etc/buildkit/buildkitd.toml \
    --use

# bootstrap：触发拉取 moby/buildkit 镜像并启动 buildkitd 容器
# 这一步要 daemon.json 的 mirror 已生效（拉 moby/buildkit 走 daemon 路径）
docker buildx inspect --bootstrap
```

**验证 builder 已切换**：

```bash
docker buildx ls
```

预期看到 `omc-builder *` 行（星号表示当前活跃）：

```
NAME/NODE       DRIVER/ENDPOINT          STATUS    PLATFORMS
omc-builder *   docker-container
  omc-builder0  unix:///var/run/docker.sock  running  linux/amd64, linux/amd64/v2, ...
```

### 3.4 重跑 compose build

```bash
cd /path/to/goomc                    # 项目根目录
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

镜像层（alpine / golang / node / nginx / moby/buildkit）全部走 mirror 拉取。

---

## 4. Mirror 连通性检测

万一某个 mirror 当前服务器访问不了（mirror 偶尔会限速 / 维护 / 区域屏蔽），先检测：

```bash
for m in docker.m.daocloud.io docker.1panel.live hub.rat.dev docker.nju.edu.cn; do
    printf "%-30s " "$m"
    curl -fsS -o /dev/null -w "TCP=%{time_connect}s HTTP=%{http_code}\n" \
        --max-time 5 "https://$m/v2/" 2>&1 || echo "TIMEOUT/FAIL"
done
```

输出示例（健康状态）：

```
docker.m.daocloud.io        TCP=0.012s HTTP=401
docker.1panel.live          TCP=0.018s HTTP=401
hub.rat.dev                 TCP=0.024s HTTP=401
docker.nju.edu.cn           TCP=0.031s HTTP=401
```

HTTP 401 是**正常的**（mirror 要求鉴权才允许列表查询，但镜像拉取不需要鉴权）。
关键看 `TCP=` 是否能连上。`TIMEOUT/FAIL` 的 mirror 从 daemon.json + buildkitd.toml 都删掉。

### 备用 mirror 列表

主推上面 4 个之外，可备用：

```
https://dockerhub.icu
https://docker.unsee.tech
https://docker-0.unsee.tech
https://dockerproxy.com
https://docker.fxxk.dedyn.io
https://hub.uuuadc.top
```

> Mirror 服务变化频繁。本文档列出的 mirror 在 2026-05 时点可用，长期建议在生产部署前
> 用 §4 的连通性检测脚本验证。

---

## 5. 错误信息对照表

| 错误信息 | 根因 | 解法 |
|---|---|---|
| `failed to resolve source metadata for docker.io/library/alpine:3.19` `dial tcp ...:443: i/o timeout` | BuildKit 拉 base image 直接走 docker.io | §3.2 + §3.3 |
| `resolve image config for docker-image://docker.io/docker/dockerfile:1` | BuildKit 拉 syntax frontend 走 docker.io | §3.2 + §3.3 |
| `the --mount option requires BuildKit` | 关了 BuildKit（设了 `DOCKER_BUILDKIT=0`） | `unset DOCKER_BUILDKIT COMPOSE_DOCKER_CLI_BUILD` 恢复 |
| `dial tcp: lookup mirrors.aliyun.com on 53: i/o timeout` | 容器内 DNS 失败（端口 53） | 参 [DOCKER_DNS_FIX.md](./DOCKER_DNS_FIX.md)，不是 mirror 问题 |
| `go: ... mirrors.aliyun.com... no such host` | 同上，DNS 问题 | 同上 |
| `apk add: temporary failure in name resolution` | 同上，DNS 问题 | 同上 |
| `docker buildx inspect --bootstrap` 卡住 | moby/buildkit 镜像没走 daemon mirror | 验 §3.1，必要时 `docker pull moby/buildkit:latest` 手动拉一次 |

---

## 6. 常见错误模式

### 6.1 「我配了 daemon.json 但 compose build 还是超时」

最常见：daemon mirror 生效了（`docker pull alpine:3.19` 能用），但 `docker compose build` 走
BuildKit。**两边都要配**，参 §3.2 + §3.3。

### 6.2 「我配了 buildkitd.toml 但没生效」

`docker` driver 不读 buildkitd.toml。必须用 `docker-container` driver（§3.3 `docker buildx create`）。

验证当前 driver：

```bash
docker buildx ls
```

如果当前活跃 builder 的 `DRIVER` 列是 `docker` 而不是 `docker-container`，buildkitd.toml 没生效。

### 6.3 「mirror 配错或某个 mirror 挂了导致拖累整体」

BuildKit 的 mirror 列表是**顺序尝试**，第一个 mirror 慢/挂会拖慢整体。先 §4 检测，把不通的删掉。

### 6.4 「断网 / 完全离线机房」

部分客户机房完全无外网。两条路：
1. 客户提供内网 harbor（你这边把 daemon.json 的 mirror 改成内网 harbor URL）
2. 在外网机器 `docker save` 镜像成 tar，离线 `docker load` 到目标机

本文档不展开离线部署流程。

---

## 7. 清理 / 还原

如果想撤回所有改动：

```bash
# 删 daemon.json（注意先备份）
sudo rm /etc/docker/daemon.json
sudo systemctl restart docker

# 删 buildkitd.toml + builder
sudo rm -rf /etc/buildkit
docker buildx rm omc-builder
docker buildx use default
```

---

## 8. 相关文档

- [README.md](./README.md) — Docker Compose 部署总指南
- [DOCKER_DNS_FIX.md](./DOCKER_DNS_FIX.md) — 容器内 DNS 解析失败（端口 53 问题）
- [TROUBLESHOOTING-INTERMITTENT-DNS.md](./TROUBLESHOOTING-INTERMITTENT-DNS.md) — 间歇性 DNS 故障排查
