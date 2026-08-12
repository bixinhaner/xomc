# 发布包 License keystore 外部挂载修复审查

## 结论

PASS

未发现 CRITICAL、WARNING 级问题。

## 审查范围

- `deployments/release/build-release.sh`
- `deployments/release/bundle/deploy/docker-compose.app.yml`
- `deployments/release/bundle/deploy/install.sh`
- `deployments/release/bundle/deploy/install-resource-preflight_test.sh`
- `deployments/release/bundle/deploy/license-keystore-release_test.sh`

## 核心检查

- 发布构建明确携带仓库内 TrueLicense JKS 公钥库。
- 安装预检使用固定 SHA-256 校验发布包，缺失或篡改时 fail closed。
- 首次安装播种到 `/opt/omc/etc/license`；升级保留现场已有 OEM keystore，不覆盖运维配置。
- App 通过只读 bind mount 消费 keystore，并挂载宿主机 `/sys` 支持 MAC/UUID 绑定。
- STOREPWD 默认值仅用于公开 JKS 完整性与旧 License 解码，不包含签发私钥。
- 新增发布契约测试并同步既有安装测试夹具，防止后续打包链路回归。

## 验证

- `bash deployments/release/bundle/deploy/install-resource-preflight_test.sh` — PASS=16, FAIL=0
- `bash deployments/release/build-release.sh --verify-only` — PASS=325, FAIL=0；License、Tempo、归档门禁通过
- `bash deployments/release/build-release-verification_test.sh` — PASS
- `git diff --check` — PASS
- `.251` 临时修复验证 — app 容器稳定运行，keystore 只读挂载，SHA-256 正确，加载错误消失

## 风险与回滚

- 影响范围仅为正式发布包构建与安装；开发 Compose 行为未改变。
- 现场自定义 keystore 在升级时保留，避免覆盖 OEM 配置。
- 回滚可恢复升级前的 `docker-compose.app.yml`，并重建 app 容器。
