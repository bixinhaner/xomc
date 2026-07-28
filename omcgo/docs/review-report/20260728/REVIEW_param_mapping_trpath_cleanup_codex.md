# REVIEW: param mapping trpath cleanup

## 结论

PASS

## 范围

- 清理 `BLQ.xml`、`MLN.xml`、`MLQ.xml`、`BSC.xml`、`BTS.xml` 中按对应全量参数集无法匹配的独有叶子参数。
- 新增 `GSM全量参数集.csv`、`LTE全量参数集.csv` 作为本轮清理基准。
- 将 `*_trpath_review.md` 统一移动到 `omcgo/data/param-mappings/trpath-reviews/`。
- 同步 `BaiBNQ` 参数模型测试，移除已清理参数 `Device.FAP.NguIpBind1.BindInterface` 的旧期望，仅保留 fallback 可写性验证。

## 审查要点

- 未发现 CRITICAL。
- XML 清理遵循归一化口径：数字实例段归一为 `{i}` 后对比 CSV `trpath.name`。
- 删除范围限制在 XML 独有且无子路径的叶子参数；对象/表节点保留。
- BSC 保留 `DeviceGSM.` 产品私有前缀参数，不按 GSM CSV 缺失删除。
- LTE 三个站型的 `RRCTimers` 均按 `LTE全量参数集.csv` 更新 `access/type/min/max/defaultValue`。
- Review 文档已迁入统一目录，BM/BaiBNQ 旧 review 以重命名形式保留历史内容。

## 验证

- `python3` XML/CSV 口径校验：通过。5 个目标 XML 均可解析；独有叶子参数为 0；BSC 排除 `DeviceGSM.` 产品私有前缀；LTE `RRCTimers` 无缺失。
- `git diff --check --cached`：通过。
- `go build ./...`：通过。
- `go test ./internal/config/parammodel`：通过。
- `go test ./...`：通过。
- `npm run build`（`omcmb/webcode`）：通过。
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web`：web/app/acs 镜像构建完成；compose 依赖的 `migrate-tsdb-schema` 因本地 TSDB 缺少 `public.pm_measurement_anchors` 退出 1。
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --no-deps acs app web`：通过，`acs/app/web` 已启动。
- `curl -I --max-time 10 http://localhost:8081/`：`HTTP/1.1 200 OK`。

## 风险与说明

- `migrate-tsdb-schema` 失败属于本地 TSDB 迁移状态问题，不阻塞本次参数映射 XML/CSV 清理提交；服务已用 `--no-deps` 恢复并验证 UI 入口可访问。
- 两份新增 CSV 为参考基准文件，已统一 LF 行尾并清理 tab/行尾空格以通过 `diff --check`。
