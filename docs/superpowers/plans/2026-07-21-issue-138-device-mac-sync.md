# Issue #138：设备列表所有基站 MAC 地址为空

## 问题现象

基站已经接入 OMC，设备列表中的 MAC 地址列仍显示为空。问题在多种产品上同时出现。

## 根因

后端从参数表读取 MAC、写入 `device_info.mac`、通过列表接口返回以及前端字段映射的链路均已存在。实际缺口位于设备列表发起的局部参数同步：

- 原同步任务请求了带实例号的标准路径 `Device.Ethernet.Interface.{i}.MACAddress`。
- 原同步任务也请求了设备厂商兼容路径 `Device.DeviceInfo.X_COM_MACAddress`。
- MLN、MLQ、BLQ、BM、BSC、BTS 等产品参数模型使用的是不带实例号的固定标准路径 `Device.Ethernet.Interface.MACAddress`，原任务没有请求该路径。

局部同步规划器按照产品参数模型中的标准路径进行匹配。缺少固定标准路径后，同步任务可能正常结束，但没有取得能够写入 `device_info.mac` 的 MAC 参数。

## 修复方案

设备列表局部同步同时请求三类路径：

1. 固定标准路径 `Device.Ethernet.Interface.MACAddress`。
2. 带实例号的标准路径 `Device.Ethernet.Interface.{i}.MACAddress`。
3. 设备厂商兼容路径 `Device.DeviceInfo.X_COM_MACAddress`。

后端现有 `InfoSyncer` 已能识别这三类路径，因此不增加产品型号判断，也不改变设备列表接口结构。

## 为什么这样修复

- 在参数同步源头补齐缺失路径，不从其他字段推导或伪造 MAC。
- 同时兼容固定路径、实例路径和厂商路径，不影响现有产品。
- Bootstrap 全量同步仍从产品参数模型生成路径，不与设备列表局部同步逻辑冲突。

## 验收标准

- 部署修复版本后，对不同产品各执行一次设备信息同步，设备列表显示真实 MAC。
- 刷新页面后 MAC 仍存在，证明数据已写入后端投影而不是临时前端值。
- 新接入设备完成 Bootstrap 同步后也能显示 MAC。

## 自动化验证

- 前端测试校验局部同步请求同时包含三类 MAC 路径。
- 后端既有测试校验 `InfoSyncer` 对固定标准路径和厂商兼容路径的识别。
- 测试服务器仍需部署后使用真实设备完成端到端验收。
