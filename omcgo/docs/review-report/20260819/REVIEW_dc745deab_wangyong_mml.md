# 代码审查报告：MML 大结果展示与 BSC 命令操作补全

- 日期：2026-08-19
- 基线：`dc745deab`
- 范围：MML 控制台、参数模型、命令目录与基线种子
- 结论：PASS

## 变更概述

- MML 执行结果表只渲染前 80 个动态参数列，并提示用户下载完整结果；详情与 CSV 导出仍保留全量参数。
- 缓存对象路径模板正则并按叶子参数预分组，降低大对象查询结果解析与列展开开销。
- 区分 `DeviceGSM.Bts.` 单实例对象与 `DeviceGSM.Bts.{i}.` 多实例对象，避免单实例声明误授权或阻断 ADD/RMV。
- 为 BSC 多实例 GSM 对象补齐 ADD/RMV/MOD/LST，为 GSM 全局参数、handover2、DNS 与 IPv4 地址配置补齐 MOD/LST。
- 参数详情按产品参数映射读取访问权限，使 BSC 的 `READ_WRITE` 能覆盖标准路径默认权限。

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

- `npm run lint` 返回 0 error；仓库当前仍有 3756 条既有 warning，本次相关文件未发现阻断问题。
- 参数目录变更折回既有 `000001_init_seed.sql`，未新增 `000002+` 迁移，符合当前未封版基线规则。
- 浏览器验证仅选择设备与命令并检查参数列表，没有向设备执行或下发命令。

## 检查项

- 权限口径：MOD 参数由产品级 `param_mappings.access` 决定，未把其他产品的只读映射错误提升为可写。
- 对象语义：ADD/RMV 必须由显式 `{i}.` 对象模板授权；无索引单实例对象不再代表集合能力。
- 数据完整性：列表列数限制只影响渲染，详情弹窗与后端/客户端 CSV 导出继续使用完整 `displayColumns`。
- 性能：路径模板正则按模板缓存，解析时先按精确路径、叶子名和对象规则分组，避免对每个返回参数重复构造正则。
- SQL：种子更新使用固定条件和参数表关联，没有运行时字符串拼接 SQL；重复执行保持幂等。
- 测试：覆盖单/多实例对象能力、产品权限覆盖、GSM 命令种子及 80 列展示与完整下载提示。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：通过，包含 `test/e2e` 与 `test/integration`。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb && npm run test --workspace webcode -- --run src/pages/mml/Console/components/ResultTable.test.tsx`：5 项通过。
- `cd omcmb && npm run lint`：0 error，3756 warning。
- `xmllint --noout` 校验 4 个参数模型 XML：通过。
- `git diff --staged --check`：通过。
- 本地 Docker App 重启后 `http://localhost:8081/`：HTTP 200。
- 可见浏览器验证：BSC 多实例对象显示增删改查；单实例对象显示查改；DNS 与 IPv4 修改命令及参数列表正确，无 page error。
