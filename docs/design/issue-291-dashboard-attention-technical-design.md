# Issue #291 仪表板关注事项技术方案

## 目标

在仪表板新增一个只读关注事项接口和前端模块，聚合两类已有业务状态：

- 异常：活跃紧急、重要告警。
- 待办：接入控制待审核候选设备。

首页不落库、不创建新处理状态、不承接处理动作，只跳转原模块。

## 后端设计

### 接口

- `GET /api/v1/dashboard/attention`
  - 参数：`abnormal_limit`、`todo_limit`，当前默认都为 `1`。
  - 返回：异常 section、待办 section、生成时间。
- `GET /api/v1/dashboard/attention/abnormalities`
  - 参数：`page`、`page_size`。
  - 返回：异常分页列表。
- `GET /api/v1/dashboard/attention/todos`
  - 参数：`page`、`page_size`。
  - 返回：待办分页列表。

### 数据来源

- `AlarmSource`
  - 使用 `AlarmReader.ListActive`。
  - 分别读取 `critical`、`major`。
  - 带入当前用户可见网元组。
  - 明细路由为 `/alarm/current?alarmId={id}`。

- `CandidateSource`
  - 使用 `deviceaccess.ManagementStore.ListCandidates`。
  - 固定筛选 `status=pending`、`expires_after=now`。
  - 按 `first_seen_at asc` 排序。
  - 带入当前用户可见网元组。
  - 明细路由为 `/device/access-control?tab=candidates&operator={carrier}&candidateId={id}&reviewStatus=pending`。

### 排序

- 异常：紧急优先于重要；同级按告警发生时间倒序。
- 待办：候选设备按首次发现时间正序。

### 权限

- 超级管理员直接可见。
- 普通用户按接口权限和可见网元组过滤。
- 某个来源失败时，section 返回 `partial`；全部来源失败时该 section 返回 `error`。

## 前端设计

### 首页

- 在原首页板块之间插入 `AttentionBar`。
- 每类展示标题、数量、1 条预览和“查看全部”。
- 空态、加载态、失败态都限定在关注事项模块内。

### 侧边抽屉

- 点击异常“查看全部”只显示异常列表。
- 点击待办“查看全部”只显示待办列表。
- 抽屉内只提供查看/跳转，不提供处理按钮。
- 分页固定在抽屉底部，列表区域滚动。

### 跳转

- 告警页支持 `alarmId` 参数，自动加载并打开对应详情。
- 接入控制页支持 `tab=candidates`、`candidateId`、`reviewStatus`，自动切到候选页并定位对应候选设备。

## 保留与裁剪

保留：

- 关注事项聚合接口。
- 告警异常来源。
- 接入候选待办来源。
- 首页紧凑预览、分类抽屉、原模块深链。

裁剪：

- Ops 运维任务审批待办来源。
- Ops 任务列表筛选、深链、审批 UI、四眼审批兼容改造。
- 首页侧自定义处理动作。

## 验证

- 后端：`go test ./internal/attention ./internal/deviceaccess ./internal/alarm`。
- 前端：关注事项 API、`AttentionBar`、当前告警深链、接入控制深链相关单测。
- 浏览器：本地打开 `/dashboard`，验证首页展示、两个抽屉、异常跳转、待办跳转、空态和暗色主题可读性。
