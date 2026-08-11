# 即插即用参数配置模板

## 模板清单

| 制式 | 产品技术 | 模板文件 | 前端下载路径 |
| --- | --- | --- | --- |
| 4G | LTE / eNB | [selfConfiguration_qa.xlsx](selfConfiguration_qa.xlsx) | `/templates/selfConfiguration_qa.xlsx` |
| 5G | NR / gNB | [5G_Autonomous_Self-Deployment_Template.xlsx](5G_Autonomous_Self-Deployment_Template.xlsx) | `/templates/5G_Autonomous_Self-Deployment_Template.xlsx` |

页面根据所选产品的技术制式选择默认模板。尚未识别产品制式时，不应提供错误的默认模板。

## 维护规则

1. 本目录保存评审和交付使用的模板原件。
2. `omcmb/webcode/public/templates/` 保存前端构建时使用的运行时副本。
3. 修改模板后必须同步两处文件，并比较 SHA-256。
4. 导入解析逻辑应允许任意 Sheet 提供有效参数，不能因某个特定 Sheet 为空而拒绝整个工作簿。
5. 模板字段、Sheet 语义或校验规则变化时，应同时更新本说明及相应解析测试。
