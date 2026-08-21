# Docker DNS 间歇性故障排查指南

> **场景**: 同样的 docker-compose,昨天能成功,今天报 DNS 错误  
> **创建时间**: 2025-04-10

---

## 🔍 **为什么昨天成功今天失败?**

### **常见原因**

| 原因 | 可能性 | 说明 |
|------|--------|------|
| **DNS 服务器波动** | ⭐⭐⭐⭐⭐ | 公共 DNS 可能暂时不稳定 |
| **网络环境变化** | ⭐⭐⭐⭐ | WiFi 切换、VPN、代理变化 |
| **Docker 重启** | ⭐⭐⭐ | Docker Desktop 更新或重启后 DNS 丢失 |
| **ISP DNS 劫持** | ⭐⭐⭐ | 运营商 DNS 间歇性故障 |
| **镜像源故障** | ⭐⭐ | mirrors.aliyun.com 暂时不可用 |
| **防火墙规则变化** | ⭐⭐ | 系统更新后防火墙重置 |
| **Docker 版本更新** | ⭐ | 新版本改变了 DNS 处理方式 |

---

## 🛠️ **快速诊断 (3 分钟)**

### **方法 1: 使用诊断脚本 (推荐)**

```bash
cd deployments/docker
bash diagnose-dns.sh
```

**脚本会自动检查**:
- ✅ 宿主机 DNS 解析
- ✅ Docker daemon 配置
- ✅ 容器内 DNS 解析
- ✅ 网络连通性
- ✅ 构建测试
- ✅ 系统变化

---

### **方法 2: 手动快速检查**

```bash
# 1. 宿主机 DNS 正常吗?
nslookup mirrors.aliyun.com

# 2. Docker 容器 DNS 正常吗?
docker run --network bridge --rm alpine:3.19 nslookup mirrors.aliyun.com

# 3. Docker daemon 有 DNS 配置吗?
cat /etc/docker/daemon.json | grep dns

# 4. 网络通吗?
curl -I https://mirrors.aliyun.com
```

---

## 📊 **排查决策树**

```
昨天成功,今天失败
    │
    ├─ 1. 宿主机 DNS 正常吗?
    │   ├─ 否 → 修复宿主机网络/DNS
    │   └─ 是 ↓
    │
    ├─ 2. 容器 DNS 正常吗?
    │   ├─ 否 → 配置 Docker daemon DNS ⭐
    │   └─ 是 ↓
    │
    ├─ 3. Docker daemon 有 DNS 配置吗?
    │   ├─ 否 → 配置 daemon.json ⭐⭐⭐
    │   └─ 是 ↓
    │
    ├─ 4. Docker 最近重启过吗?
    │   ├─ 是 → 重启后 DNS 配置丢失?
    │   └─ 否 ↓
    │
    ├─ 5. 是临时网络波动吗?
    │   ├─ 是 → 等待 5 分钟重试
    │   └─ 否 → 继续排查 ↓
    │
    └─ 6. 检查防火墙/VPN/代理变化
```

---

## 🎯 **常见场景及解决方案**

### **场景 1: DNS 服务器间歇性故障** ⭐⭐⭐⭐⭐

**症状**:
```
nslookup mirrors.aliyun.com  # 有时成功,有时超时
```

**解决**:

```bash
# 1. 临时切换到备用 DNS
sudo networksetup -setdnsservers Wi-Fi 223.5.5.5 223.6.6.8.8.8.8

# 2. 或配置 Docker daemon (永久)
cd deployments/docker
bash fix-docker-dns.sh
```

---

### **场景 2: Docker Desktop 重启后 DNS 丢失** ⭐⭐⭐⭐

**症状**:
- 昨天构建成功
- 今天 Docker 自动更新或手动重启后失败
- `/etc/docker/daemon.json` 配置还在,但不生效

**原因**: Docker Desktop 重启时可能未正确加载 daemon.json

**解决**:

```bash
# 1. 完全退出 Docker Desktop
# macOS: 菜单栏 Docker 图标 → Quit Docker Desktop

# 2. 确认进程已退出
ps aux | grep Docker

# 3. 重新启动 Docker Desktop

# 4. 验证 DNS 配置生效
docker run --network bridge --rm alpine:3.19 cat /etc/resolv.conf

# 5. 重新构建
docker compose build
```

---

### **场景 3: 网络环境变化** ⭐⭐⭐

**症状**:
- 在公司正常,回家失败
- 切换 WiFi 后失败
- 开启/关闭 VPN 后失败

**排查**:

```bash
# 1. 检查当前网络
ifconfig | grep "inet "

# 2. 检查 DNS 服务器
scutil --dns | grep "nameserver"

# 3. 检查代理
echo $http_proxy
echo $https_proxy

# 4. 检查 VPN
networksetup -listallnetworkservices
```

**解决**:

```bash
# 方案 1: 配置 Docker daemon (推荐,不受网络环境影响)
cd deployments/docker
bash fix-docker-dns.sh

# 方案 2: 根据网络环境切换 DNS
# 公司网络
sudo networksetup -setdnsservers Wi-Fi 10.0.0.1 10.0.0.2

# 家庭网络
sudo networksetup -setdnsservers Wi-Fi 223.5.5.5 8.8.8.8
```

---

### **场景 4: 镜像源暂时不可用** ⭐⭐

**症状**:
```
ERROR: fetch https://dl-cdn.alpinelinux.org/...: DNS error
或
ERROR: fetch ...: connection timed out
```

**排查**:

```bash
# 1. 测试镜像源
curl -I https://dl-cdn.alpinelinux.org
curl -I https://mirrors.aliyun.com

# 2. 测试多个 DNS
nslookup dl-cdn.alpinelinux.com 223.5.5.5
nslookup dl-cdn.alpinelinux.com 8.8.8.8
nslookup dl-cdn.alpinelinux.com 114.114.114.114
```

**解决**:

```bash
# 等待几分钟后重试 (通常是暂时的)
sleep 300
docker compose build

# 或切换镜像源
# 修改 Dockerfile,使用其他镜像
FROM registry.cn-hangzhou.aliyuncs.com/library/golang:1.25-alpine
```

---

### **场景 5: 防火墙规则变化** ⭐⭐

**症状**:
```
docker run --network bridge --rm alpine:3.19 wget https://mirrors.aliyun.com
# wget: can't connect to remote host: Connection refused
```

**排查**:

```bash
# macOS
sudo pfctl -s rules | grep 53

# Linux
sudo iptables -L -n | grep 53
sudo ufw status
```

**解决**:

```bash
# 临时关闭防火墙测试 (不推荐生产环境)
sudo ufw disable

# 或添加 DNS 规则
sudo ufw allow out 53/udp
sudo ufw allow out 53/tcp
```

---

## 🔧 **永久解决方案**

### **方案 1: 配置 Docker Daemon DNS** ⭐⭐⭐⭐⭐ (推荐)

**一劳永逸,不受网络环境影响**:

```bash
cd deployments/docker
bash fix-docker-dns.sh
```

**原理**: 
- 配置写入 `/etc/docker/daemon.json`
- 所有新容器自动使用配置的 DNS
- 不依赖宿主机 DNS 设置

---

### **方案 2: Docker Compose DNS 覆盖**

**临时方案,只影响当前项目**:

```yaml
# docker-compose.override.yml
services:
  migrate:
    dns:
      - 223.5.5.5
      - 8.8.8.8
  
  app:
    dns:
      - 223.5.5.5
      - 8.8.8.8
```

**注意**: 这只影响运行时,不影响构建时!

---

### **方案 3: 在 Dockerfile 中添加重试**

**已实施**,可以应对临时网络抖动:

```dockerfile
RUN for i in 1 2 3; do \
      apk add --no-cache git && break || \
      { echo "Attempt $i failed, retrying..."; sleep 2; }; \
    done
```

---

## 📋 **完整排查清单**

遇到问题时,按顺序检查:

- [ ] **1. 宿主机 DNS 正常?**
  ```bash
  nslookup mirrors.aliyun.com
  ```

- [ ] **2. 容器 DNS 正常?**
  ```bash
  docker run --network bridge --rm alpine:3.19 nslookup mirrors.aliyun.com
  ```

- [ ] **3. Docker daemon 有 DNS 配置?**
  ```bash
  cat /etc/docker/daemon.json | grep dns
  ```

- [ ] **4. Docker 最近重启过?**
  - 是 → 重启后配置可能丢失
  - 否 → 继续排查

- [ ] **5. 网络环境变化了?**
  - WiFi 切换?
  - VPN 开/关?
  - 代理变化?

- [ ] **6. 防火墙规则变化?**
  ```bash
  sudo ufw status  # Linux
  sudo pfctl -s rules  # macOS
  ```

- [ ] **7. 镜像源可用?**
  ```bash
  curl -I https://mirrors.aliyun.com
  ```

- [ ] **8. 是临时波动?**
  ```bash
  sleep 300  # 等 5 分钟
  docker compose build
  ```

---

## 🎯 **推荐处理流程**

### **立即执行 (5 分钟)**:

```bash
# 1. 运行诊断脚本
cd deployments/docker
bash diagnose-dns.sh

# 2. 根据诊断结果修复
# 如果显示 "容器 DNS 解析失败"
bash fix-docker-dns.sh

# 3. 重启 Docker
# macOS: Docker Desktop → Restart

# 4. 重新构建
docker builder prune -f
docker compose build
```

---

### **长期优化 (30 分钟)**:

1. **配置 Docker daemon DNS** (必做)
   ```bash
   bash fix-docker-dns.sh
   ```

2. **配置多个 GOPROXY** (已完成)
   ```dockerfile
   ARG GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,direct
   ```

3. **添加重试逻辑** (已完成)
   ```dockerfile
   RUN for i in 1 2 3; do ... done
   ```

4. **监控网络质量** (可选)
   ```bash
   # 添加到 cron 定时任务
   */5 * * * * nslookup mirrors.aliyun.com >> /tmp/dns-monitor.log
   ```

---

## 📊 **诊断结果解读**

### **诊断脚本输出示例**:

```
=========================================
  Docker DNS 问题诊断工具
=========================================

1️⃣  宿主机网络检查
-----------------------------------------
  测试 DNS 解析 (mirrors.aliyun.com)... ✅ PASS
  测试 DNS 解析 (goproxy.cn)... ✅ PASS
  测试 HTTPS 连接... ✅ PASS

2️⃣  Docker 环境检查
-----------------------------------------
  Docker 服务状态... ✅ PASS
  Docker daemon DNS 配置... ⚠️  WARN
    /etc/docker/daemon.json 不存在
  Docker 网络状态... ✅ PASS

3️⃣  容器 DNS 检查
-----------------------------------------
  容器内 DNS 解析... ❌ FAIL
  容器内 HTTPS 连接... ❌ FAIL

=========================================
  诊断结果汇总
=========================================
  通过: 5
  失败: 2
  警告: 1

❌ 发现问题,需要修复

推荐修复方案:
  1. 配置 Docker daemon DNS
     cd deployments/docker
     bash fix-docker-dns.sh
```

---

## 🔗 **相关文档**

- [Docker DNS 修复指南](./DOCKER_DNS_FIX.md)
- [Docker DNS 配置模板](./docker-daemon-dns.json)
- [一键修复脚本](./fix-docker-dns.sh)
- [DNS 覆盖配置](./docker-compose.dns.yml)

---

**文档维护**: 随着问题解决持续更新  
**最后更新**: 2025-04-10
