# Issue 158 PLMN 与 MME 回读修复设计

## 目标

修复 249/452 基站快速设置中的两个独立问题：

1. 恢复独立的服务 PLMN 配置能力，最多配置 6 个 PLMN。
2. MME IP+PLMN 保存成功后，只有在设备回读值与本次提交值一致时才用服务器值刷新表单，避免旧值覆盖用户刚提交的多行数据。

## 设备模型

### 452 / MLN

MLN 使用单值字符串参数：

```text
Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList
```

快速设置把该字符串渲染为单列 PLMN 表格。设备值和下发值使用逗号分隔，例如：

```text
46000,46001
```

约束：

- 最多 6 条；
- 每条为 5 或 6 位数字；
- 不允许重复；
- 空行不参与下发。

### 249 / MLQ

MLQ 使用标准多实例对象：

```text
Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.
```

快速设置使用现有 `MultiInstanceTable`，暴露 `PLMNID` 字段，最多 6 个实例。参数模型补充该对象的可写对象映射，使 AddObject/DeleteObject 可以通过现有校验。

### 与 MME 绑定关系的边界

服务 PLMN 列表和 `MmeIpPlmnList` 是两个独立配置：

- 服务 PLMN：基站提供服务的 PLMN 集合；
- MME IP+PLMN：每个 MME 地址及其绑定 PLMN。

本次不自动联动或重写二者，避免引入设备侧未确认的约束。

## MME 保存后回读

保存时在现有快速设置反馈状态中记录本次实际下发的 `parameterPath -> parameterValue`。

当 SPV 任务进入终态时：

- `failed/expired/cancelled`：保留草稿和当前表单值，不用设备旧值覆盖；
- `completed`：失效参数缓存并重新读取目标路径；
- 只有目标路径的当前值与本次提交值一致，才回填表单、清除草稿和字段错误；
- 尚未一致时继续轮询，直到匹配、取消或超时；
- 超时或查询失败时保留草稿，并提示回读失败。

该判断以可观察的参数值为准，不依赖固定延时，也不要求新增 SPV 与自动 GPV 的任务关联接口。

## 测试

- 前端纯函数测试：PLMN 字符串解析、规范化、序列化、重复/格式/数量校验。
- 前端回读测试：第一次返回旧值、后续返回目标值时继续等待；超时时抛出明确错误；取消时立即停止。
- 后端快速设置配置测试：MLN 存在独立 `ExistPlmnidList` 分组；MLQ 存在最多 6 实例的 `EPC.PLMNList` 分组。
- 后端参数映射测试：MLQ 的 `EPC.PLMNList` 对象可写。
- 全量前端类型检查和相关 Go 测试。

## 非目标

- 不修改 MME IP+PLMN 的 TR-069 路径或编码格式。
- 不改变设备端 SPV 后自动 GPV 的后端机制。
- 不为其他快速设置字段引入新的回读语义。
- 不在服务 PLMN 与 MME 绑定表之间增加自动同步。
