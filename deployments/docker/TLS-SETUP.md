# TLS 部署 — 自签证书 nginx termination

> **关联**：T-0117（登录非 secure context fail-fast）/ T-0118（运维 TLS 部署）
> 解决：Web Crypto API（`crypto.subtle`）在非 secure context 下不可用，
> 部署在内网 IP（如 `http://172.19.1.73:8081`）时登录加密失败。

## 背景

OMC 前端的密码加密走 Web Crypto API（RSA-OAEP），后者**只在 secure context 可用**：

| URL | Secure Context | crypto.subtle |
|-----|---------------|---------------|
| `https://任何主机` | ✅ | ✅ |
| `http://localhost` / `http://127.0.0.1` | ✅（spec 例外）| ✅ |
| `http://172.19.1.73:8081` | ❌ | ❌ |

Web 标准硬约束，非 OMC bug。生产部署必须**通过 HTTPS 访问**。

本文档提供"零依赖快速启用"路径：自签证书 + nginx termination。Let's Encrypt / 内部 CA 路径作为后续运维优化方向，不在本文档覆盖。

---

## 启用步骤（5 分钟）

### 1. 生成自签证书

```bash
# host_ip 替换为实际部署 IP；days 默认 1825（5 年）
bash deployments/docker/scripts/generate-self-signed-tls.sh 172.19.1.73
```

输出到 `deployments/docker/certs/`：
- `server.crt` — 公钥证书（644）
- `server.key` — 私钥（600）
- `openssl.cnf` — 生成时使用的 CSR 配置（可删）

证书 SAN 自动包含 `localhost`、`127.0.0.1` 和指定 IP。

### 2. 启用 nginx TLS server block

```bash
cd deployments/docker
cp default-tls.conf.example default-tls.conf
```

`default-tls.conf` 已 `.gitignore` 忽略，可按需进一步本地化（如改 SAN / cipher）。

### 3. 修改 docker-compose.yml web 服务

在 `deployments/docker/docker-compose.yml` 的 `web` 服务下增加：

```yaml
  web:
    # ...既有内容...
    ports:
      - "8081:8081"       # 既有 plain HTTP（保留向后兼容）
      - "8080:8080"       # 既有 ACS gateway
      - "443:443"         # ← 新增 TLS gateway
    volumes:
      - ../../run/logs/nginx:/var/log/nginx
      - /etc/localtime:/etc/localtime:ro
      - ./certs:/etc/nginx/certs:ro                                # ← 新增
      - ./default-tls.conf:/etc/nginx/conf.d/default-tls.conf:ro   # ← 新增
```

### 4. 重启 web 容器

```bash
cd deployments/docker
docker-compose up -d web
```

健康检查：

```bash
# nginx 启动正常应有 443 listener
docker-compose logs web | tail -20

# 内部健康探测
curl -sk https://localhost/api/v1/auth/public-key | head -c 100
```

### 5. 浏览器访问

```
https://172.19.1.73/login
```

**首次访问会出现自签证书警告**（Chrome 显示"您的连接不是私密连接"）→
点"高级" → "继续前往..."。此后该浏览器会记住信任。

---

## 验证

| 项 | 验证方法 | 期望 |
|----|---------|------|
| Secure Context | DevTools Console 输入 `window.isSecureContext` | `true` |
| crypto.subtle | DevTools Console 输入 `crypto.subtle` | 对象（非 undefined）|
| 登录 | 填用户名密码登录 | 成功（跳 dashboard）|
| Network 面板 | 看 POST `/api/v1/auth/login` | 200 + 返 token pair |

---

## 与现有 :8081 plain HTTP 入口的关系

- **保留 :8081**：向后兼容内网测试 / 老链接 / 不需要 secure context 的接口
- **新增 :443**：客户使用的主入口（登录必走此路径）
- 两个入口共享同一 nginx 容器、同一 `app_backend` upstream、同一 SPA 构建产物

---

## 长期演进方向

### 短期（本部署）
✅ 自签证书 + nginx 443 termination — 本文档覆盖

### 中期（生产环境）
- **内部 CA 签发证书**：避免自签警告，IT 部门统一管理
- **Let's Encrypt 自动续期**：如部署可公网解析的域名，用 certbot + nginx 自动续期
- **强制重定向 8081 → 443**：当 :443 稳定后，default.conf :8081 server 加 `return 301 https://$host$request_uri;`

### 长期（多实例 / 高可用）
- **TLS 卸载到 LB**：nginx 层只做内部 proxy，TLS 终结在 HAProxy / 云 LB
- **mTLS 双向认证**：客户端证书 + 服务端证书，配合 RBAC 做"机器对机器"鉴权

---

## 故障排查

| 症状 | 可能原因 | 解决 |
|------|---------|------|
| `docker-compose up -d web` 后容器 restart loop | nginx 配置错 / cert 路径错 | `docker-compose logs web` 看 nginx error log |
| 浏览器报 `ERR_CERT_AUTHORITY_INVALID` | 自签证书未信任 | "高级" → "继续访问"，或导入 server.crt 到系统受信根证书 |
| 浏览器报 `ERR_CERT_COMMON_NAME_INVALID` | 证书 SAN 不含访问 IP/域名 | 重跑 generate-self-signed-tls.sh 把实际 IP 加进 SAN |
| 登录仍失败 | TLS 通了但 Web Crypto 还是不可用？ | DevTools Console 看 `window.isSecureContext`；应为 true |
| `/api/v1/auth/login` 401 | 用户名/密码错 / 后端 keystore 未初始化 | 检查 admin 用户密码 / 看 backend 日志 |

---

## 相关文档

- 后端 cipher 设计：`omcgo/internal/admin/loginpwd/` (cipher.go / keystore.go / handler.go)
- 前端 password encrypt：`omcmb/frontend-core/src/services/crypto/passwordCipher.ts`
- T-0117 fix commit：`5c1cf45c`（fail-fast 让错误前置 + 信息明确）
- T-0118 本文件 + 脚本 + nginx conf 模板
