# AT-001 104 部署入场门禁记录

> 环境：`http://172.17.3.104:8081`
>
> 目标提交：MR !670 / `b36f9bb457a06ef69dd43a942f1cf5d7a2b15be2`
>
> 当前结果：数据库门禁已修复，等待同版本全制品重部署

## 修复前

- App、ACS、Worker、Web 健康端点均可访问；
- 接入名单接口 200，策略和导入接口 500；
- 运行库缺少 `device_access_actions.candidate_id` 等当前基线结构；
- 页面存在大量 `deviceAccess.*` 裸 key，中英文均缺失；
- 未创建策略、名单或候选，未触发重评、任务和 RF 动作。

## 数据库修复证据

使用当前分支 `omcgo-migrate` 对 104 主库执行幂等基线重放，命令退出 0。修复后只读查询结果：

```text
candidate_id=true
import_batches=true
failure_mode=true
policies_query=ok rows=1
imports_query=ok rows=0
```

## Web 构建证据

当前分支生产构建通过，生成主 bundle `index-DV6HYkHv.js`；该 bundle 同时包含接入控制新增的中文和英文消息。104 必须替换完整 Web 制品，不能只替换 AccessControl chunk。

## 重部署后检查

- 记录 App、Worker、ACS、Web 镜像摘要和启动时间；
- 记录迁移任务退出码与日志；
- 实测 `policies`、`imports`、`access-list` 状态码与响应；
- 中文、英文各走一遍五页签并保存控制台证据；
- 确认不再有数据库字段缺失和 `MISSING_TRANSLATION`。
