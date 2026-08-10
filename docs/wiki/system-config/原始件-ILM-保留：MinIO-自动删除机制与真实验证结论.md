# 原始件 ILM 保留：MinIO 自动删除机制与真实验证结论

来源：GitLab Issue #275。

本文说明系统配置页「资源保留与背压」里的「原始件 ILM 保留」是否真实生效。该配置用于控制 MinIO 中 PM/MR 原始 XML 文件的保留天数，涉及 `pm-files` 和 `mr-files` 两个桶。

## 结论

配置生效。系统设置保存后，后端会把 MinIO `pm-files` / `mr-files` 的正式 lifecycle 规则重下发；MinIO 会按规则自动删除满足过期条件的真实 PM/MR 原始件。

当前环境已恢复为测试前口径：

- 页面值：原始件保留天数 = 60 天
- 配置库：`minio.retention.raw_object_days = 60`
- 清理模式：`minio.retention.cleanup_mode = shadow`
- `pm-files` lifecycle：`omc-raw-expire-60d`，Enabled，60 天，整桶覆盖
- `mr-files` lifecycle：`omc-raw-expire-60d`，Enabled，60 天，整桶覆盖
- worker：已恢复运行

## 真实验证摘要

验证时通过系统设置页把保留天数从 60 天临时改为 1 天，并临时停止 worker，避免应用层兜底清理和 MinIO ILM 自动删除混淆。验证过程中没有创建临时 lifecycle 规则，也没有执行 `mc rm` 手工删除对象。

关键结果：

- 修改前 `pm-files` 总对象数：204
- 修改前超过 24 小时候选对象数：119
- 修改后 `pm-files` 对象数：132
- 修改后超过 24 小时候选对象数：47
- `mr-files` 当前无对象，无法观察 MR 原始件实际删除，但 lifecycle 已同步重下发

已确认被 MinIO lifecycle 自动清理的真实候选对象：

- `pm-files/2026/08/04/A20260804.1345+0800-1400+0800_48BF74.1202000240194DP0015.xml.gz`
- `pm-files/2026/08/04/A20260804.1400+0800-1415+0800_48BF74.1202000240194DP0015.xml.gz`

未过期对照对象仍保留：

- `pm-files/2026/08/06/A20260806.1900+0800-1915+0800_48BF74.1202000240194DP0015.xml.gz`

## 规则说明

MinIO lifecycle 是按单个对象判断，不是等整个桶都过期后才删除。

- 某个 PM/MR 原始文件满配置天数，就只删除这个过期文件。
- 其他没满配置天数的文件继续保留。
- `pm-files` / `mr-files` 桶本身不会被删除。
- 空前缀表示桶内所有对象都适用同一条规则。

## 参考记录

本地完整排查记录：

`/Users/shangyingbin/project/tmp/原始件ILM保留排查结论.md`

代码依据：

- `xomc/omcgo/internal/core/components/minio/minio.go`
- `xomc/omcgo/cmd/app/provider/minio_ilm.go`
- `xomc/omcgo/cmd/worker/raw_cleanup.go`
- `xomc/omcgo/internal/rawcleanup/runner.go`
- `xomc/omcgo/migrations/seed/000001_init_seed.sql`
