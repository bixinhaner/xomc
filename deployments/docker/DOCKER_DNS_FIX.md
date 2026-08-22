# Docker DNS 解析问题修复指南

> **问题**: `docker compose build` 时 `go mod download` 报错 DNS 超时  
> **错误**: `dial tcp: lookup mirrors.aliyun.com on 223.5.5.5:53: read udp 172.17.0.2:42280->223.5.5.5:53: i/o timeout`  
> **创建时间**: 2025-04-10
>
> 以上 `172.17.0.2` 仅保留为历史故障样例。当前 Docker 网段以客户规划的 `DOCKER_BIP`
> 为基准派生；新建或重建 Docker 网络不得复用任何 `172.x` 网段，请以规划库和 Compose
> IPAM 校验为准。

---

## 🔍 **问题根因**

### 错误信息分析

**错误 1**: DNS 解析超时
```
go: github.com/Masterminds/squirrel@v1.5.4: 
Get "https://mirrors.aliyun.com/goproxy/...": 
dial tcp: lookup mirrors.aliyun.com on 223.5.5.5:53: 
read udp 172.17.0.2:42280->223.5.5.5:53: i/o timeout
```

**错误 2**: 只读文件系统 (Docker 新版本)
```
/bin/sh: can't create /etc/resolv.conf: Read-only file system
```

**关键点**:
1. ✅ 宿主机网络正常
2. ✅ 已配置多个 GOPROXY
3. ❌ **容器内 DNS 解析失败** (UDP 53 端口超时)
4. ❌ **Docker 新版本限制**: `/etc/resolv.conf` 是只读的

### 为什么宿主机正常但容器失败?

```
宿主机:
  DNS 配置: /etc/resolv.conf → 正常解析
  网络: 正常
  
Docker 容器:
  DNS 配置: 继承 Docker daemon 配置 → 可能有问题
  网络: Docker bridge 网络 → DNS 转发可能失败
```

**常见原因**:
- Docker daemon DNS 配置不正确
- Docker bridge 网络 DNS 转发问题
- 防火墙阻止 UDP 53 端口
- DNS 服务器响应慢导致超时

---

## ✅ **解决方案 (3 选 1)**

### **方案 1: 配置 Docker Daemon DNS (推荐)** ⭐

#### **步骤**:

**1. 创建或修改 `/etc/docker/daemon.json`**

**方案 A: 使用修复脚本 (推荐,会合并配置)**

```bash
cd deployments/docker
# DOCKER_BIP 由客户按实际业务网规划；脚本会自动派生其它 Docker 网段
export DOCKER_BIP=10.240.0.1/16
bash fix-docker-dns.sh
```

**脚本特性**:
- ✅ 自动检测现有配置
- ✅ 使用 `jq` 合并 DNS 配置 (保留其他配置)
- ✅ 自动备份原文件
- ✅ 如果没有 `jq`,提供 3 种选择

**方案 B: 手动配置**

不要直接覆盖 `/etc/docker/daemon.json`，否则会丢失 Docker 网段规划。请设置客户规划的
`DOCKER_BIP` 后运行修复脚本；脚本会合并 DNS、`bip` 和自动地址池：

```bash
export DOCKER_BIP=10.240.0.1/16
bash deployments/docker/fix-docker-dns.sh
```

**2. 重启 Docker**

```bash
# Linux
sudo systemctl restart docker

# macOS (Docker Desktop)
# 点击菜单栏 Docker 图标 → Restart
```

**3. 清理构建缓存**

```bash
docker builder prune -f
```

**4. 测试 DNS 解析**

```bash
docker run --network bridge --rm alpine:3.19 nslookup mirrors.aliyun.com
```

**5. 重新构建**

```bash
cd deployments/docker
docker compose build
```

---

#### **快速修复脚本**

```bash
cd deployments/docker
bash fix-docker-dns.sh
```

---

### **方案 2: 在 docker-compose.yml 中配置 DNS** ⚡

如果不方便修改 Docker daemon 配置,可以在 compose 文件中为特定服务配置 DNS。

#### **修改 `docker-compose.yml`**:

```yaml
services:
  migrate:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile.app
    dns:
      - 223.5.5.5
      - 223.6.6.6
      - 114.114.114.114
    # ... 其他配置
```

**优势**: 
- ✅ 不影响全局 Docker 配置
- ✅ 只影响指定服务

**劣势**:
- ⚠️ 需要在每个 build 服务中配置

---

### **方案 3: 优化 Dockerfile (治本)** 🏆

#### **问题分析**

当前 `Dockerfile.app`:

```dockerfile
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

ARG GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /build

COPY omcgo/go.mod omcgo/go.sum ./
RUN go mod download  # ← 这里失败
```

**问题**: `go mod download` 需要 DNS 解析,但容器 DNS 不稳定。

---

#### **优化方案 A: 添加重试逻辑 (已实施)** ✅

```dockerfile
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

# 配置 GOPROXY
ARG GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /build

COPY omcgo/go.mod omcgo/go.sum ./

# 添加重试逻辑
RUN for i in 1 2 3; do \
      go mod download && break || \
      { echo "Attempt $i failed, retrying in 3s..."; sleep 3; }; \
    done

COPY omcgo/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-app ./cmd/app
```

**注意**: Docker 新版本中 `/etc/resolv.conf` 是只读的,不能修改。
必须在宿主机配置 Docker daemon 的 DNS。

---

#### **优化方案 B: 使用国内基础镜像**

```dockerfile
# 使用阿里云 Go 镜像 (已预配置国内源)
FROM registry.cn-hangzhou.aliyuncs.com/library/golang:1.25-alpine AS builder

RUN apk add --no-cache git

# 阿里云镜像源
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
ENV GOFLAGS=-insecure

WORKDIR /build

COPY omcgo/go.mod omcgo/go.sum ./

# 添加重试
RUN for i in 1 2 3; do \
      go mod download && break || \
      { echo "Attempt $i failed, retrying..."; sleep 3; }; \
    done

COPY omcgo/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-migrate ./cmd/migrate
```

**优势**:
- ✅ 基础镜像在国内,拉取更快
- ✅ 预配置国内源
- ✅ 减少 DNS 解析依赖

---

## 🎯 **推荐实施步骤**

### **立即修复 (5 分钟)**

```bash
# 1. 先按客户规划设置 Docker 网段，再配置 Docker DNS (方案 1)
export DOCKER_BIP=10.240.0.1/16
bash deployments/docker/fix-docker-dns.sh

# 2. 重启 Docker
# macOS: Docker Desktop → Restart
# Linux: sudo systemctl restart docker

# 3. 清理缓存
docker builder prune -f

# 4. 重新构建
cd deployments/docker
docker compose build
```

---

### **长期优化 (30 分钟)**

**1. 修改 Dockerfile 添加重试逻辑** (方案 3A)

**2. 考虑使用国内基础镜像** (方案 3B)

**3. 在 CI/CD 中配置 DNS**

```yaml
# .github/workflows/build.yml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Configure DNS
        run: |
          echo "nameserver 223.5.5.5" | sudo tee /etc/resolv.conf
          echo "nameserver 8.8.8.8" | sudo tee -a /etc/resolv.conf
```

---

## 🔧 **故障排查**

### **1. 验证 DNS 配置**

```bash
# 查看 Docker daemon 配置
cat /etc/docker/daemon.json

# 查看容器 DNS 配置
docker run --network bridge --rm alpine:3.19 cat /etc/resolv.conf

# 测试 DNS 解析
docker run --network bridge --rm alpine:3.19 nslookup mirrors.aliyun.com
docker run --network bridge --rm alpine:3.19 nslookup goproxy.cn
```

---

### **2. 检查网络连通性**

```bash
# 测试 DNS 端口
docker run --network bridge --rm alpine:3.19 nc -vz -u 223.5.5.5 53

# 测试 HTTPS 连接
docker run --network bridge --rm alpine:3.19 wget --spider https://mirrors.aliyun.com

# 测试 GOPROXY
docker run --network bridge --rm golang:1.25-alpine go env GOPROXY
```

---

### **3. 常见错误及解决**

| 错误 | 原因 | 解决 |
|------|------|------|
| `i/o timeout` | DNS 服务器无响应 | 更换 DNS 服务器 |
| `connection refused` | DNS 服务未运行 | 检查 DNS 服务状态 |
| `no such host` | DNS 配置错误 | 检查 `/etc/resolv.conf` |
| `network unreachable` | 网络不通 | 检查 Docker 网络 |

---

### **4. 高级调试**

```bash
# 查看 Docker 网络
docker network ls
docker network inspect bridge

# 查看容器网络
docker run --network bridge --rm alpine:3.19 ip addr
docker run --network bridge --rm alpine:3.19 route -n

# 抓包分析
sudo tcpdump -i docker0 port 53 -n

# 查看 Docker 日志
sudo journalctl -u docker -f
```

---

## 📊 **方案对比**

| 方案 | 复杂度 | 效果 | 适用场景 | 推荐度 |
|------|--------|------|---------|--------|
| **方案 1: Docker DNS** | ⭐ 简单 | ✅ 最佳 | 开发环境 | ⭐⭐⭐⭐⭐ |
| **方案 2: Compose DNS** | ⭐⭐ 中等 | ✅ 好 | 临时修复 | ⭐⭐⭐⭐ |
| **方案 3A: 重试逻辑** | ⭐⭐ 中等 | ⚠️ 缓解 | 网络不稳定 | ⭐⭐⭐ |
| **方案 3B: 国内镜像** | ⭐⭐⭐ 复杂 | ✅ 最佳 | 生产环境 | ⭐⭐⭐⭐⭐ |

---

## 🎯 **最终推荐**

### **开发环境**: 方案 1 + 方案 3A

```bash
# 1. 按客户规划配置 Docker DNS 和网段
export DOCKER_BIP=10.240.0.1/16
bash deployments/docker/fix-docker-dns.sh

# 2. 修改 Dockerfile 添加重试
# (参考方案 3A)

# 3. 重启 Docker
# macOS: Docker Desktop → Restart

# 4. 重新构建
docker compose build
```

---

### **生产环境**: 方案 1 + 方案 3B

```dockerfile
# 使用国内基础镜像
FROM registry.cn-hangzhou.aliyuncs.com/library/golang:1.25-alpine AS builder

# 配置阿里云 GOPROXY
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 添加重试逻辑
RUN for i in 1 2 3; do \
      go mod download && break || \
      { echo "Attempt $i failed"; sleep 3; }; \
    done
```

---

## 📝 **验证清单**

构建成功后,验证以下内容:

- [ ] 所有服务构建成功
- [ ] `docker compose up` 启动正常
- [ ] 服务可以正常访问
- [ ] DNS 解析稳定 (多次构建测试)
- [ ] 构建时间合理 (< 5 分钟)

---

## 🔗 **参考资料**

- [Docker DNS 配置文档](https://docs.docker.com/config/containers/container-networking/#dns-services)
- [Go 模块代理](https://goproxy.io/)
- [阿里云 Go 模块代理](https://mirrors.aliyun.com/goproxy/)
- [Docker 网络故障排查](https://docs.docker.com/network/network-tutorial-standalone/)

---

**文档维护**: 随着问题解决持续更新  
**最后更新**: 2025-04-10
