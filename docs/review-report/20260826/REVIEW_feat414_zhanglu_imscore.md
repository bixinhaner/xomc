# 代码审查报告

- Issue: #414
- 范围: data_0826 参数表同步 IMSCORE 参数模型 + 纳入标准参数树（standard-model.xml）
- 审查结论: PASS

## 审查范围

- `omcgo/data/param-mappings/IMSCORE.xml`（231 → 245 项）
- `omcgo/data/param-mappings/standard-model.xml`（header 计数修正 + 追加 ImsCore 22 object / 223 param）

均为字典 XML 数据变更，无 Go/TS 代码改动。

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

1. **对齐方式**：CSV `trpath.name` 列直接给出标准路径，逐条精准对齐（209/223 命中既有路径），非模糊匹配。
2. **changeApplies 保护**：改动前后逐条比对，reboot 生效标记（#408 修正）0 改动。
3. **CSV 假象过滤**：字符串默认值被导出工具全大写化（cmnet→CMNET、mgw→MGW 等）与约 140 处无信息量 max=65535/4294967295 均不采纳；仅采纳 5 处真实类型变化与 9 处有效约束/默认值修正。
4. **standard-model.xml**：object 条目带尾点（与 loader 语义一致）；param 保留 max/defaultValue 数值型属性；header 计数与实际解析数对齐（原 header 漂移 1，loader 实际以解析为准，header 仅元信息）。
5. **数据库落库路径**：两文件均走 data/ bind-mount + dictloader 启动期幂等 UPSERT（param_mappings 全量重写、standard_params UPSERT），无新增迁移（符合未封版本 000001 基线约束）。
6. 环境侧运维操作（MML catalog 23 分组/65 命令/426 sub_fields、product_class 字典绑定源刷新）走 admin API/SQL 直接生效，带审计，不属于本 MR 范围。

## 验证记录

- XML 解析：两文件均 well-formed（python ElementTree）
- `cd omcgo && go test ./internal/config/parammodel/... ./internal/mml/... ./internal/product/...`：通过
  （quicksettings 的 TestBuiltinIMSCORE_QuickSettingsReferenceParamModel 在 main 上既有失败，与本次无关：#408 reboot 口径冲突）
- 容器栈重启后日志：`standard-model loaded objects=40 params=2214`、products loaded 18/62
- 库验证：param_mappings IMSCORE=245；standard_params 总数 3223（Device.ImsCore% 245 = 22 object + 223 parameter）；TAC=STRING 已落库
- MML 命令树 `GET /api/v1/mml/group-tree?root=IMSCORE_G_ROOT`：两级树完整返回；ImsCore 产品视角 unsupported 516（ImsCore 命令全部可用），BLQ 视角 unsupported 537（ImsCore 命令全部过滤）
