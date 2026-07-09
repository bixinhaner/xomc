# 设备注册批量导入功能设计说明

**版本**: 2026-07-09  
**相关 MR**: feat/device-batch-preregister  
**作者**: AI 辅助设计，hezg 评审

---

## 一、背景

在批量开站场景中，运维人员通常需要在设备物理到站、接入网络**之前**，就提前在 OMC 系统中规划好设备的名称、备注等信息，以便开站过程中实时统计站点就绪状态。

原有的「导入」功能仅能对**已上线过**的设备更新名称/备注，无法录入尚未连接 OMC 的设备，不满足批量开站的工程规划需求。

---

## 二、功能变更说明：Import 按钮改为下拉

### 原设计

```
[ 导出 ]   [ 导入 ]
```

导入功能语义单一：**仅能更新已注册设备的名称和备注**。

### 新设计

```
[ 导出 ]   [ 导入 ▼ ]
               ├─ 导入（更新名称/备注）
               └─ 批量预登记
```

将「导入」升级为下拉按钮，提供两个子选项，明确区分两种操作意图。

---

## 三、两个选项的说明

### 3.1 导入（更新名称/备注）

**入口**：导入 → 导入

**适用场景**：设备**已上线接入** OMC，需要批量更新设备名称或备注。例如：
- 设备上线后补填站点名称
- 批量修正设备备注

**行为**：
- 按 SN 匹配已注册设备，更新 `device_name` / `remark`
- **SN 不存在的行会失败**，不会创建新设备
- **不改变设备归属分组**（之前已修复）
- 导入后设备留在原来的分组，Group Source 保持不变

**CSV 模板列**：`SN`、`设备名称`（可选）、`备注`（可选）

---

### 3.2 批量预登记

**入口**：导入 → 批量预登记

**适用场景**：设备**尚未上线**（还没有接入 OMC），需要提前录入 SN 列表和站点名称，用于批量开站规划和进度统计。例如：
- 工程规划阶段，导入采购清单的 SN 和站点名称
- 施工前预配置，让 OMC 在设备到站前已有记录

**行为**：

| 情况 | 处理方式 |
|------|---------|
| SN **已存在**于 OMC | 只更新名称/备注（同「导入」行为） |
| SN **不存在**于 OMC | **新建设备**，`lifecycle_state = 'registered'`，`is_online = false` |
| 运营商无法推断 | 报错 `carrier_required`，提示在 CSV 中补填运营商字段 |

**关于新建设备的归属**：
- 预登记的新设备**不写入任何分组**（无 `device_group_members` 记录）
- 自然落在「Default Group」**未分组视图**中显示
- Group Source 列显示 **Auto**（绿色 Tag）
- 设备接入后，由 GroupMatchEngine 按规则自动归组（source_type → `rule`）

**关于运营商推断**：
- 优先使用 CSV 填写的 `carrier` 列
- 若为空，自动从 SN 前6位推断 OUI，再通过 CarrierRegistry 解析运营商
- 推断失败则该行报错 `carrier_required`，用户需在 CSV 中补填

**CSV 模板列**：`SN`（必填）、`设备名称`（可选）、`备注`（可选）、`运营商`（可选，cmcc/ctcc/cucc）、`制式`（可选，lte/nr）

---

## 四、完整开站工作流

```
① 工程规划阶段
   运维导出采购清单 → 填写站点名称 → 批量预登记
   → 设备出现在「Default Group」未分组视图，显示 Auto

② 设备到站入网
   设备 Bootstrap Inform 到达 ACS
   → RegisterFromInform 按 SN 查到已有记录，UPDATE（不 INSERT）
   → lifecycle_state 从 registered → commissioned
   → 名称已提前写好，无需再操作

③ 归组
   自动：GroupMatchEngine 按 SN/LAC/TAC 规则命中 → source_type = rule
   手工：运维在设备分组页拖拽/Move → source_type = manual
```

---

## 五、为什么用下拉而不是单独按钮

| 方案 | 优点 | 缺点 |
|------|------|------|
| **下拉（当前）** | 节省工具栏空间；两个操作在概念上都属于"导入"，归为一组符合认知 | 需要多一次点击 |
| 两个独立按钮 | 一次点击 | 工具栏已有导出/移动/回收站/删除，再加一个按钮过于拥挤 |
| 合并成一个 Modal | 减少入口数量 | 两种操作的 CSV 格式不同，合并会造成混淆 |

选择下拉的核心理由：**两个操作都属于「从文件导入」的范畴，对外语义一致，对内行为不同**，下拉是业界常见的区分方式（参考 GitHub、GitLab 的 Clone 按钮下拉）。

---

## 六、改动文件

| 文件 | 改动 |
|------|------|
| `omcgo/internal/device/device_handler_batch_preregister.go` | 新增 `BatchPreRegisterDevices` handler |
| `omcgo/internal/device/device_handler_batch_preregister_test.go` | 新增 4 个测试用例 |
| `omcgo/internal/device/device_handler.go` | 注册路由 `/devices/batch-preregister` |
| `omcgo/internal/device/device_service.go` | 新增 `ResolveCarrierByOUI` 方法 |
| `omcmb/frontend-core/src/types/device.ts` | 新增预登记类型定义 |
| `omcmb/frontend-core/src/services/api/deviceApi.ts` | 新增 `batchPreRegisterDevices` API |
| `omcmb/frontend-core/src/i18n/en-US/index.ts` | 新增预登记相关 i18n key |
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | 同上 |
| `omcmb/webcode/src/pages/device/DeviceGrouping/BatchPreRegisterModal.tsx` | 新增预登记 Modal 组件 |
| `omcmb/webcode/src/pages/device/DeviceGrouping/DeviceListPanel.tsx` | Import 升级为下拉，集成 BatchPreRegisterModal |
| `omcmb/webcode/src/pages/device/DeviceGrouping/useImportExportHandlers.ts` | 新增预登记 handler 和模板下载 |
| `omcmb/webcode/src/pages/device/DeviceGrouping/index.tsx` | 接入新 handler |
