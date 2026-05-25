# Review Report — PM 上传自动配置 (PM Auto-Setup on Device Online)

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm, worker, appconfig, deploy
- **Backlog**: T-0164-G1（PM 端到端缺口收尾 / 真机 PM 文件上传链路打通）
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)

## Conclusion

**PASS_WITH_WARNINGS** — 可合入。0 CRITICAL，2 WARNING（v2 增强项），3 INFO。

8 unit test 全过；docker rebuild 部署后 worker log 含 `pm online subscriber registered subject=device.online queue=pm-online-pm-upload-setup`；env `OMC_PUBLIC_HOST=172.19.1.132` 注入正确。

## Files Changed

| Path | LOC | Type |
|------|-----|------|
| `omcgo/internal/core/appconfig/config.go` | +30 | mod（加 PMConfig） |
| `omcgo/cmd/worker/etc/config.{dev,test,prod,local}.yaml` | +6×4 | mod |
| `omcgo/internal/pm/online_subscriber.go` | +205 | new |
| `omcgo/internal/pm/online_subscriber_test.go` | +175 | new |
| `omcgo/cmd/worker/main.go` | +15 | mod |
| `/Users/shangyingbin/project/omc-docker/docker-compose.yml` | +3 | mod（worker env OMC_PUBLIC_HOST） |

## Findings

- 无 CRITICAL
- **W1 D6 corner case**：firmware.changed 抢占 device.online（`device_service.go:732-735`）— 升级后首次 inform 不重发 PM 上传配置，等下次自然 offline→online。用户确认接受此 trade-off。
- **W2 单 SPV 包 3 参数**：单 SetParameterValues task 一次下发 Enable/URL/PeriodicUploadInterval。CPE 部分实现可能要求逐个 SPV（厂商行为）— 真机测试时观察。
- 路径标准 `Device.FAP.PerfMgmt.Config.1.*`；BLQ 平台 standardPath=privatePath（与文档对齐）；param_models translator 兜底其它平台。
- env 注入：docker-compose `OMC_PUBLIC_HOST=${OMC_PUBLIC_HOST:-172.19.1.132}`；OnlineSubscriber expandEnv 支持 `${VAR:-default}` shell 语法。
- 失败处理：log only（无 user_id 不进消息中心，与 task_subscriber 现有规则一致）。

## Tests
- `Test_OnlineSubscriber_EnqueuesSingleSPVWith3Params` ✓
- `Test_OnlineSubscriber_EnvSubstitutionDefaultFallback` ✓
- `Test_OnlineSubscriber_EnvSubstitutionFromEnv` ✓
- `Test_OnlineSubscriber_FailureIsLoggedNotPropagated` ✓
- `Test_OnlineSubscriber_EmptySerialIsSkipped` ✓
- `Test_OnlineSubscriber_MalformedURLIsSkipped` ✓
- `Test_expandEnv_DefaultSyntax` ✓
- `Test_OnlineSubscriber_EnqueuesSingleSPVWith3Params` 兼测 enable_value=1 / interval=900 / 全量 url 拼接

## DoD
- [x] go build + go test 全过
- [x] yaml 配置 dev/test/prod/local 4 套覆盖
- [x] docker-compose env 注入
- [x] docker 部署 + 启动 log 验证 wiring（未真机端到端，留明天真机回来后做）
