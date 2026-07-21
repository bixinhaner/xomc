# MML 小区服务与基站配置参数目录审查

- 审查日期: 2026-07-21
- 审查对象: `feat/mml-cell-service-catalog`
- 结论: PASS

## 范围

- TD-LTE `SF / 小区服务参数管理（总体）` 命令归一化。
- NR `SF` 新增同级目录 `基站配置参数管理`，并按小节保留查询、修改、添加、删除命令。
- MML 配置页树形展示、搜索定位和命令排序。
- QA 文档、删除 TRPath 明细与归一化摘要。

## 发现

- 未发现阻塞问题。
- 种子迁移的 `Down` 保持 no-op，符合当前 seed 迁移惯例；如需回滚目录绑定，需要用历史 catalog 或备份数据恢复。
- `cell-service-deleted-trpaths.csv` 是审计证据文件，不参与运行时加载。

## 验证

- `go test ./internal/mml ./internal/config/parammodel/mmlstandardloader`
- `npm run typecheck`
- `python3 -m json.tool omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json`
- `python3 -m json.tool omcgo/datamodels/mml-catalog/cmcc-tdlte-v2.3.json`
- `git diff --check --cached`
- 本地已部署 `goomc-local-app-1` 与 `goomc-local-web-1`，`http://localhost:8081/` 返回 `HTTP/1.1 200 OK`。

## 重点核对

- `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.Enable` 已保留。
- `Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary` 已保留。
- NR `基站配置参数管理` 为 `小区服务参数管理（总体）` 同级目录。
- NR 命令直接挂载在 `基站配置参数管理` 下，层级保持与 TD-LTE 小区服务目录一致。
- 命令排序按同一小节内 `查询 / 修改 / 添加 / 删除` 排列。
