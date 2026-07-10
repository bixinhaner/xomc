# Task 13 report — 全量验证、部署演练与文档收尾

日期：2026-07-10  
工作树：`feat-mml-script-txt-import`

## 已交付

- 新增 `omcgo/internal/mml/testdata/import-valid.txt` 和 `import-invalid.txt`。
- 新增可重复端点脚本 `omcgo/scripts/e2e_mml_script_import.sh`。脚本覆盖非法文件
  422/不发 token、合法文件校验与保存、同 token 幂等重放、服务端快照执行，以及
  `plan_line_no`、`plan_device_sn`、`plan_order`、`script_content_sha256` 结果追溯；缺少
  栈、凭据或设备时输出实际响应并以非零状态退出，不静默跳过。
- 更新 Docker 部署 README、设计稿和实施计划，写明 `000015` 迁移、API、错误码、切换顺序、
  Redis 双确认和不可逆回滚边界。

## TDD / E2E 证据

首轮命令（先观察到脚本断言失败）：

```text
bash omcgo/scripts/e2e_mml_script_import.sh
```

首轮暴露脚本自身 `set -u` 下空 `AUTH_ARGS` 的错误（`unbound variable`），随后修复并
重新运行。修复后的真实结果：

```text
[PASS] health check (HTTP 200)
[FAIL] invalid TXT is rejected: expected HTTP 422, got 404
body: {"data":null,"msg":"resource not found",...}
[FAIL] valid TXT validation: expected HTTP 200, got 404
body: {"data":null,"msg":"resource not found",...}
[FAIL] ... cannot produce a save token
MML script import E2E: FAIL (pass=1 fail=5)
```

`localhost:8081/healthz` 可达，但 Docker Compose 当前无运行容器，端口上的旧进程尚未部署
新导入路由；本轮未设置 `OMC_TOKEN`，也没有可确认的 `MML_E2E_SN`。因此没有声称 E2E PASS、
脚本 ID、任务 ID 或浏览器截图。

随后发现并修复 Redis Lua claim/release 对空数组进行 cjson round-trip 导致保存 500 的缺陷：
`b141b1e2` 将 claim 状态拆到同 hash-tag 的独立 key，`c52bd8f6` 补充临界 TTL 直接过期保护。
重建 app 镜像后使用管理员会话直连 `http://localhost:18081` 复跑，15 项断言全部通过：

```text
[PASS] health check (HTTP 200)
[PASS] invalid TXT is rejected (HTTP 422)
[PASS] invalid TXT cannot produce a save token
[PASS] valid TXT validation (HTTP 200)
[PASS] valid TXT returns one-time token
[PASS] validated TXT can be saved (HTTP 201)
[PASS] save returns script id
[PASS] same token replay is idempotent (HTTP 201)
[PASS] replay returns the original script
[PASS] execution uses server snapshot (HTTP 201)
[PASS] execution returns task id
[PASS] task snapshot is readable (HTTP 200)
[PASS] task stores immutable script hash
[PASS] task results are traceable (HTTP 200)
[PASS] results expose immutable plan trace fields
MML script import E2E: PASS (pass=15 fail=0)
```

## 质量门

| 命令 | 结果 |
|---|---|
| `bash omcgo/scripts/check-migrations.sh --strict` | exit 0；000001–000015 连续。输出既有 `000008` Down 疑似为空及 seed 同号告警，未新增冲突。 |
| `/usr/local/go/bin/go build ./...`（`omcgo`） | exit 0 |
| `/usr/local/go/bin/go test ./...`（`omcgo`） | exit 0；本轮无 PG 集成失败/跳过。 |
| `npm run skin-parity`（`ombcmb`） | exit 0；v1/v2/v3 均 129 routes / 37 visible menus。 |
| `npm run typecheck`（`ombcmb`） | exit 0；三 workspace 均通过。 |
| `npm run test`（`ombcmb/webcode-v2`） | exit 0；2 files / 9 tests。 |
| `npm run test`（`ombcmb/webcode-v3`） | exit 0；2 files / 10 tests。 |
| `npm run test`（`ombcmb/webcode`） | exit 1；既有 frontend-core 两项失败：`ufteCategory` 的 `device_upgrade` 期望 false 实得 true；`messageFormat` 的 `mml.consoleV2.rawPath.hintRmv` 报 `UNCLOSED_TAG`。未修改这些无关代码。 |

## 最新主分支门

```text
git fetch origin main                         # exit 0
git rev-list --left-right --count HEAD...origin/main
64 26
git merge-tree --write-tree --messages HEAD origin/main
07861b8aa48712b899dadfc203678070e6d7bffe     # 无冲突消息
bash omcgo/scripts/check-migrations.sh --strict # exit 0（同上既有告警）
```

## 未执行 / 阻塞项

- 三皮肤真实浏览器 `/mml/script` 验收（URL、脚本/任务 ID、网络响应和截图）：当前只重建并替换
  app，web 镜像构建在 npm ci 阶段受限，未伪造浏览器证据。
- `omcctl mml reset-script-data --dry-run`：已执行，输出 `matched=0 deleted=0 skipped=41 errors=0`；
  按安全边界未执行有写操作的 `--apply --confirm DELETE-MML-RUNTIME`，非 MML 切换前后数量未做破坏性演练。

## 当前进度

实施计划已设置：

```text
Current checkpoint: All tasks complete
Resume from: Final review and delivery
```

“All tasks complete”表示代码与文档收口；最终交付门仍需在具备完整 Docker 栈和凭据的环境补齐
浏览器、Redis 切换与端点 PASS 证据。
