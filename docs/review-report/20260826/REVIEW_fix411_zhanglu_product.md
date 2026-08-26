# 代码审查报告

- Issue: #411
- 范围: ImsCore 产品类型识别放宽为包含匹配（ProductClass 前后可带字符串）
- 审查结论: PASS

## 审查范围

- 产品字典 `products.xml` 的 ImsCore 匹配正则
- 后端设备类型判别（SQL 谓词、Go 判别函数、device_type 派生 CASE）及对应测试
- 前端设备类型兜底判别（deviceApi 映射、设备详情页判别）

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

1. 语义统一为「ProductClass 包含 ImsCore（大小写不敏感）即核心网」，共 6 处对齐：products.xml 正则、SQL 谓词 `LIKE '%IMSCORE%'`、`IsCoreNetworkProductClass`、device_type 派生 CASE、前端两处 `includes`。
2. 正则 `(?i)ImsCore` 无锚定，globalOrder=61 保持排在全部 FAP 基站模式之后，不抢占基站路由；UPS `LIKE 'UPS%'` 前缀判断与核心网包含判断互不重叠。
3. 全库扫描确认无残留的 `= 'IMSCORE'` / `EqualFold("ImsCore")` / `^ImsCore$` 精确比较。
4. products.xml 走 data/ bind-mount + dictloader 启动期幂等 UPSERT 入 `product_class_patterns`，无需新增迁移（符合未封版本只维护 000001 基线的约束）。
5. 路由缓存 TTL 1h + 版本号失效，放宽后无需手工清理。

## 验证记录

- `cd omcgo && go test ./internal/product/... ./internal/device/...`：通过
- `cd omcgo && go vet ./internal/device/... ./internal/product/...`：通过
- 正则/判别行为用例实测：`ImsCore`、`CoreNetwork/ImsCore`、`ImsCore/V2`、`xxIMSCOREyy`、`Baicells/imscore-prod` 均命中；`FAP/BAIBLQ/SC` 不命中
- 前端 eslint（两个改动文件）：0 error（8 个 warning 均在未改动行，为既有问题）
- `cd omcmb && npm run typecheck`：仅剩 `paramConfigWorkbook.ts` 既有 exceljs 报错（本机依赖未装，与本次改动无关）
- 容器栈重建重启（app/acs/worker/web）：启动日志 `products loaded (18/62)`，`product_class_patterns` 已落 `(?i)ImsCore`，无 panic/fatal
