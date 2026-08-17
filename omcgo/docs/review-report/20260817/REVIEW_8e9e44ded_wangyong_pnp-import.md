# Issue #330 即插即用 5G 参数导入修复审查

## 审查结论

PASS

未发现 CRITICAL 或 WARNING 级问题。

## 审查范围

- BaiBNQ NR Band 参数约束
- 5G 参数配置工作簿的 SCS/带宽联动
- BaiBNQ 生成表头与详情、摘要字段映射
- NR 设备支持的载波带宽选项
- 内置告警定义数量基线同步

## 核心检查

- `FreqBandIndicatorNR` 上限由错误的 40 调整为南向模型范围 1024，并有内置资产回归测试保护。
- 生成工作簿使用实际 DL/UL SCS 表头建立带宽联动，不再回退到跨 SCS 合并选项。
- gNB ID、PCI、Band、CarrierBandwidth 可通过 BaiBNQ 的生成表头及 TR Path 正确进入导入摘要和详情表单。
- NR 带宽选项按设备能力移除 5MHz、15MHz、25MHz，DL/UL 页面和工作簿共享同一数据源。
- 映射读取只访问导入配置中的首实例，不引入外部输入执行、HTML 注入或敏感信息处理。
- 告警基线从 442 同步到 443，对应 #316 已合法新增的 OMC License 容量告警；未修改告警数据及加载行为。

## 验证

- `cd omcgo && go build ./...` — 通过
- `cd omcgo && go test ./...` — 通过
- `cd omcmb && npm run typecheck` — 通过
- `cd omcmb && npm exec --workspace webcode vitest run src/pages/device/PlugAndPlay` — 27 个测试文件、148 条测试通过
- Issue 附件导入浏览器验证 — 新增 1、冲突 0，gNB ID/PCI/带宽/NRARFCN/SSB/TAC 正确展示，校验状态通过

## 风险与影响

- 影响范围限于 BaiBNQ 参数模型和 5G 即插即用参数配置流程。
- 已存在且不再受支持的 5MHz、15MHz、25MHz 当前值仍可展示，避免详情页静默丢值；新选择和工作簿校验不再提供这些档位。
- 无数据库迁移、接口协议或鉴权变更。
