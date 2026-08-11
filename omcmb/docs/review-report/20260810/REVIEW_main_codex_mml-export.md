# Issue #273 MML 导出修复审查报告

## 结论

PASS

未发现 CRITICAL 或 WARNING 问题。

## 审查范围

- MML 后端汇总 CSV 的命令排序、命令名称与操作类型输出
- MML 控制台“下载全部”的跨任务导出范围
- 多任务 CSV 合并、表头去重与序号连续编号
- 后端与前端回归测试

## 关键检查

- 后端按 `command_index` 稳定排序，同序号按创建时间排序，符合实际执行链顺序。
- 每个 device task 的首个参数行写入命令和操作类型，后续参数行保持空白，避免误归属和重复展示。
- 前端只合并当前命令记录列表中的任务，不扩大到其他用户或全局任务。
- CSV 合并使用引号感知解析，能保留逗号、双引号和单元格内换行，不使用不安全的行级字符串切割。
- 合并结果只保留一个表头，并对每个设备/命令块连续编号。
- 未引入新路由、数据库迁移、SQL、认证逻辑或运营商硬编码。

## 验证

- `cd omcgo && go build ./...` — PASS
- `cd omcgo && go test ./...` — PASS
- `cd omcmb && npm run typecheck` — PASS
- `npm run test --workspace webcode -- --run ../frontend-core/src/services/api/__tests__/mmlCsvMerge.test.ts src/pages/mml/Console/components/ResultTable.test.tsx` — PASS（2 files，4 tests）
- `npm run build` — PASS
- Docker Compose Web/App/ACS/Worker 重建 — PASS
- `curl -I http://localhost:8081/` — HTTP 200
- MML 控制台浏览器冒烟 — PASS

## 风险与影响

- 影响范围仅为 MML 控制台结果 CSV 导出。
- “下载全部”会按命令记录列表逐任务请求并在浏览器合并；列表上限为 50 条。
- 单设备下载语义保持不变。
