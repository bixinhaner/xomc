# 即插即用参数模板导入导出审查报告

- 日期：2026-08-13
- 分支：`fix/plug-and-play-template-validation`
- 结论：PASS
- CRITICAL：0
- WARNING：0

## 审查范围

- 即插即用参数模板导出备注、枚举/布尔/范围下拉与联动选项
- 导入解析的数据类型、必填项、枚举、范围、字符串长度和正则校验
- 快速设置按参数模型查询及参数模型约束字段透传
- 5G Address Type 条件校验和复合规划字段下发保护

## 关键检查

- 导入约束通过所选产品的精确 TRPath 匹配，不以模糊表头推断替代产品模型。
- 快速设置字段的显示提示、枚举值和 SCS/带宽联动规则在导入导出间复用。
- `Address Type` 采用 `DHCP / Static / DHCPv6 / Staticv6`，静态地址必填关系有回归测试。
- 参数模型接口新增字段均为可选字段，对旧客户端保持兼容。
- `Gateway` 等复合规划字段不会因同名参数而错误下发到无关 TRPath。
- 未发现字符串拼接 SQL、裸 panic、认证绕过、敏感信息或公共接口新增 `any`。

## 验证

- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb && npm test --workspace webcode -- src/pages/device/PlugAndPlay`：25 个文件、104 项测试通过。
- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./internal/config/parammodel ./internal/provision ./internal/quicksettings`：通过。
- `npm run build`：通过。
- 本地 Docker Compose 重建部署：web、app、acs 正常运行，`http://localhost:8081/` 返回 HTTP 200。

## 已知限制

- 本地环境当前没有可选的 4G/5G 产品数据，因此产品选择后的真实文件上传交互以单元回归和接口/部署健康检查覆盖。
