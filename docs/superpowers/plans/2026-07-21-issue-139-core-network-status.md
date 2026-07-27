# Issue #139 核心网连接状态分制式展示实施计划

> **供代理执行者：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`，按任务逐项执行本计划；步骤使用复选框（`- [ ]`）跟踪。

**目标：** 设备列表仅为 4G/eNB 展示真实 MME 状态，为 5G/gNB 展示真实 AMF 状态，为 GSM/BSC 展示 BSC Link Status，并让不适用的列显示空值。

**架构：** 继续复用 `device_info.mme_status` 作为 LTE/NR 核心网状态的归一化快速查询列，不增加数据库字段。后端按 `technology` 写入 LTE MME 或 NR AMF，GSM 显式写空以清除历史错误值；前端 API mapper 再按制式路由到 `mmeStatus` / `amfStatus`。BSC Link Status 直接复用列表 SQL 已有的 `bsc_link_status` 派生字段。

**技术栈：** Go、PostgreSQL 设备信息投影、React、TypeScript、React Query、Vitest。

## 全局约束

- 后端保持 `handler -> service -> repository/model` 分层，不新增数据库迁移。
- MME 仅适用于 LTE/eNB，AMF 仅适用于 NR/gNB，BSC Link 仅适用于 GSM/BSC。
- 无适用参数时必须返回空值，不得把“未上报”误判为 `disconnected`。
- 同一 MME 实例的新旧路径只能统计一次；高优先级路径为空时必须继续回退。
- BaiBNQ 参数模型已经包含 `Device.Services.FAPService.{i}.AmfsStatus`，不得重复增加 XML 条目。
- 用户可见列名使用现有 i18n key，不新增硬编码文案。
- 前端验收至少运行 `cd omcmb && npm run typecheck`。

---

### 任务 1：后端按制式计算和清理核心网状态

**文件：**
- 修改：`omcgo/internal/device/info_calc_test.go`
- 修改：`omcgo/internal/device/device_info_calc.go`
- 修改：`omcgo/internal/device/device_info_sync_test.go`
- 修改：`omcgo/internal/device/device_info_sync.go`

**接口：**
- 输入：`model.Technology`、`normalizeAMFStatus(string) string`、设备参数 `map[string]string`。
- 输出：`CalcCoreNetworkStatus(params map[string]string, tech model.Technology) string`。

- [x] **步骤 1：写 MME 路径和制式路由失败测试**

在 `TestCalcMMEStatus` 中增加以下断言：

```go
// 无任何 MME 参数时返回空值。
assert.Empty(t, CalcMMEStatus(map[string]string{}))

// BLQ 旧 LTE 路径仍能得到 partial。
assert.Equal(t, "partial", CalcMMEStatus(map[string]string{
    "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status": "1",
}))

// 同一实例 EPC 与旧路径共存时只计数一次。
assert.Equal(t, "partial", CalcMMEStatus(map[string]string{
    "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": "1",
    "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status": "1",
}))

// EPC 空占位不能阻止旧路径回退。
assert.Equal(t, "partial", CalcMMEStatus(map[string]string{
    "Device.Services.FAPService.1.CellConfig.LTE.EPC.MmePoolConfigParam.1.MME1Status": " ",
    "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status": "1",
}))
```

增加 `TestCalcCoreNetworkStatusByTechnology`，断言 LTE 返回 MME、NR 把 `AmfsStatus` 归一化为 AMF、GSM 返回空值。

- [x] **步骤 2：运行测试并确认失败**

运行：

```bash
cd omcgo
go test ./internal/device -run 'TestCalcMMEStatus|TestCalcCoreNetworkStatusByTechnology' -count=1
```

预期：失败，表现为无参数返回 `disconnected`、旧 LTE 路径未识别以及 `CalcCoreNetworkStatus` 尚不存在。

- [x] **步骤 3：实现最小制式感知计算**

`CalcMMEStatus` 先读取精确 Gateway 状态；否则按每个实例的 EPC 路径、旧 LTE 路径顺序选择第一个非空值。没有观察到任何有效状态时返回空字符串。

```go
func CalcCoreNetworkStatus(params map[string]string, tech model.Technology) string {
    switch tech {
    case model.TechLTE:
        return CalcMMEStatus(params)
    case model.TechNR:
        return normalizeAMFStatus(params[amfsStatusPath])
    default:
        return ""
    }
}
```

- [x] **步骤 4：写历史状态清理失败测试**

在 `device_info_sync_test.go` 中构造无 `AmfsStatus` 的 NR 参数，断言传给 `UpdateSyncFields` 的 `fields` 明确包含 `"mme_status": ""`，而不是删除该 key。

- [x] **步骤 5：使用制式感知结果并确认通过**

`InfoSyncer.SyncFromParameters` 无条件执行：

```go
fields["mme_status"] = CalcCoreNetworkStatus(paramValues, tech)
```

运行：

```bash
cd omcgo
gofmt -w internal/device/device_info_calc.go internal/device/device_info_sync.go internal/device/info_calc_test.go internal/device/device_info_sync_test.go
go test ./internal/device -run 'TestCalcMMEStatus|TestCalcCoreNetworkStatusByTechnology|TestInfoSyncer_SyncFromParameters_ClearsStaleCoreNetworkStatus' -count=1
```

预期：通过。

### 任务 2：前端按制式映射、同步和展示三类状态

**文件：**
- 修改：`omcmb/frontend-core/src/services/api/__tests__/deviceApi.test.ts`
- 修改：`omcmb/frontend-core/src/services/api/deviceApi.ts`
- 新建：`omcmb/webcode/src/pages/device/DeviceList/deviceCoreNetworkStatus.test.ts`
- 新建：`omcmb/webcode/src/pages/device/DeviceList/deviceCoreNetworkStatus.ts`
- 修改：`omcmb/webcode/src/pages/device/DeviceList/deviceBatchTask.test.ts`
- 修改：`omcmb/webcode/src/pages/device/DeviceList/deviceListParamSync.ts`
- 修改：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`

**接口：**
- 输入：`Device.networkType`、后端 `mme_status` 和 `bsc_link_status`。
- 输出：`mmeStatusForDevice`、`amfStatusForDevice`、`bscLinkStatusForDevice` 三个纯函数，以及三个制式专属列表列。

- [x] **步骤 1：写 API 制式映射失败测试**

在 `deviceApi.test.ts` 中分别返回 LTE、NR、GSM 设备：

```ts
expect(lte.mmeStatus).toBe('connected');
expect(lte.amfStatus).toBe('');
expect(nr.mmeStatus).toBe('');
expect(nr.amfStatus).toBe('connected');
expect(gsm.mmeStatus).toBe('');
expect(gsm.amfStatus).toBe('');
expect(gsm.bscLinkStatus).toBe('connected');
```

- [x] **步骤 2：写列适用性和同步路径失败测试**

`deviceCoreNetworkStatus.test.ts` 断言三个 helper 只为适用制式返回值。`deviceBatchTask.test.ts` 断言 eNB 只请求 MME 路径，gNB 只请求 `Device.Services.FAPService.1.AmfsStatus`。

- [x] **步骤 3：运行前端测试并确认用例失败**

运行：

```bash
cd omcmb
npm run test --workspace webcode -- ../frontend-core/src/services/api/__tests__/deviceApi.test.ts
npm run test --workspace webcode -- src/pages/device/DeviceList/deviceCoreNetworkStatus.test.ts src/pages/device/DeviceList/deviceBatchTask.test.ts
```

预期：用例失败，表现为 NR 仍把 `mme_status` 映射到 MME、辅助函数不存在、AMF 同步路径缺失。

- [x] **步骤 4：实现 API 制式路由和三个纯函数**

`mapBackendDevice` 先计算一次 `networkType`。LTE 把 `bd.mme_status` 映射到 MME；NR 把 `bd.amf_status || bd.mme_status` 映射到 AMF；其他制式两者均为空。BSC Link 继续使用 `bd.bsc_link_status`。

- [x] **步骤 5：增加制式专属同步路径和列表列**

`deviceListParamSync.ts` 增加：

```ts
{ key: 'mmeStatus', scope: 'eNB', paths: ['Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MmeStatus'] },
{ key: 'amfStatus', scope: 'gNB', paths: ['Device.Services.FAPService.1.AmfsStatus'] },
```

`DeviceList/index.tsx` 中 MME、AMF、BSC Link 三列分别调用对应 helper，不适用制式显示 `-`。

- [x] **步骤 6：运行前端测试并确认用例通过**

运行：

```bash
cd omcmb
npm run typecheck
npm run test --workspace webcode -- ../frontend-core/src/services/api/__tests__/deviceApi.test.ts
npm run test --workspace webcode -- src/pages/device/DeviceList/deviceCoreNetworkStatus.test.ts src/pages/device/DeviceList/deviceBatchTask.test.ts
```

预期：通过。

### 任务 3：完成文档、全量验证和提交

**文件：**
- 修改：`docs/superpowers/plans/2026-07-21-issue-139-core-network-status.md`

**接口：**
- 输入：任务 1、任务 2 的测试结果和最终文件清单。
- 输出：可用于 GitLab MR 的根因、修复方案、验证记录。

- [x] **步骤 1：自审范围**

确认不修改 BaiBNQ/standard-model XML，不新增数据库迁移，不把 #101 的 GPS 改动带入分支。

- [x] **步骤 2：运行最终验证**

运行：

```bash
cd omcgo && go build ./... && go test ./...
cd omcmb && npm run typecheck
cd omcmb && npm run test --workspace webcode -- ../frontend-core/src/services/api/__tests__/deviceApi.test.ts
cd omcmb && npm run test --workspace webcode -- src/pages/device/DeviceList/deviceCoreNetworkStatus.test.ts src/pages/device/DeviceList/deviceBatchTask.test.ts
git diff --check
```

预期：所有命令退出码为 0。

- [x] **步骤 3：创建独立提交**

```bash
git add docs/superpowers/plans/2026-07-21-issue-139-core-network-status.md \
  omcgo/internal/device/device_info_calc.go \
  omcgo/internal/device/device_info_sync.go \
  omcgo/internal/device/info_calc_test.go \
  omcgo/internal/device/device_info_sync_test.go \
  omcmb/frontend-core/src/services/api/deviceApi.ts \
  omcmb/frontend-core/src/services/api/__tests__/deviceApi.test.ts \
  omcmb/webcode/src/pages/device/DeviceList/deviceCoreNetworkStatus.ts \
  omcmb/webcode/src/pages/device/DeviceList/deviceCoreNetworkStatus.test.ts \
  omcmb/webcode/src/pages/device/DeviceList/deviceListParamSync.ts \
  omcmb/webcode/src/pages/device/DeviceList/deviceBatchTask.test.ts \
  omcmb/webcode/src/pages/device/DeviceList/index.tsx
git commit -m "fix(device): 按制式展示核心网连接状态 (#139)"
```
