# Issue #101 GPS 展示回归修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 保留经度、纬度、GPS 高度三处黄色警告图标，优化图标与数值排版，并在确认弹窗展示设备坐标快照的采集时间和槽位。

**Architecture:** 后端接口已经返回 `reported.observedAt`、`reported.sourcePath` 和 `reported.version`，本次不增加数据库字段。前端增加纯展示辅助函数，把技术路径转换为用户可理解的槽位；原始参数路径继续保留在接口中供排障使用，但不在确认弹窗常驻展示。弹窗继续使用已有确认接口和版本冲突保护。

**Tech Stack:** React、TypeScript、Ant Design、Vitest、React Testing Library。

## Global Constraints

- 经度、纬度、GPS 高度三列各保留一个黄色 `WarningOutlined`，数量不变。
- 只有 `locationSync.status === 'pending'` 时显示警告图标。
- 不自动触发测试设备参数同步，不修改 239 测试设备状态。
- 用户可见新增文案必须同时提供中文和英文。
- 所有修改仅位于现有 xomc 仓库和当前 issue 分支。

---

### Task 1: GPS 来源展示模型

**Files:**
- Create: `omcmb/webcode/src/pages/device/DeviceList/deviceGpsObservation.ts`
- Create: `omcmb/webcode/src/pages/device/DeviceList/deviceGpsObservation.test.ts`

**Interfaces:**
- Consumes: `ReportedDeviceLocation.sourcePath: string`
- Produces: `parseGpsObservationSource(sourcePath: string): { sourcePath: string; slot: 1 | 2 | 3 }`

- [ ] **Step 1: 写失败测试**

覆盖标准路径、私有路径和 `.2/.3` 槽位，断言路径不变且槽位分别为 1、2、3。

- [ ] **Step 2: 运行测试确认 RED**

Run: `cd omcmb && npm test -- --run webcode/src/pages/device/DeviceList/deviceGpsObservation.test.ts`

Expected: FAIL，原因是 `deviceGpsObservation` 模块不存在。

- [ ] **Step 3: 最小实现**

实现 `parseGpsObservationSource`：仅把末尾 `.2`、`.3` 识别为槽位，其他路径回退槽位 1，完整参数源路径原样返回。

- [ ] **Step 4: 运行测试确认 GREEN**

Run: `cd omcmb && npm test -- --run webcode/src/pages/device/DeviceList/deviceGpsObservation.test.ts`

Expected: PASS。

### Task 2: 弹窗快照信息与三图标样式

**Files:**
- Modify: `omcmb/webcode/src/pages/device/DeviceList/GpsSyncConfirmModal.tsx`
- Modify: `omcmb/webcode/src/pages/device/DeviceList/GpsSyncConfirmModal.test.tsx`
- Modify: `omcmb/webcode/src/pages/device/DeviceList/GpsSyncTrigger.tsx`
- Modify: `omcmb/webcode/src/pages/device/DeviceList/GpsSyncTrigger.test.tsx`
- Modify: `omcmb/webcode/src/pages/device/DeviceList/index.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Consumes: `device.locationSync.reported.observedAt/sourcePath`
- Produces: 中英文采集时间、坐标槽位展示；三列紧凑黄色警告入口。

- [ ] **Step 1: 写失败测试**

弹窗测试断言英文环境显示 `Observed at`、`Coordinate slot`，不显示原始参数路径且无中文；图标测试断言紧凑按钮类名和黄色警告图标仍存在。

- [ ] **Step 2: 运行测试确认 RED**

Run: `cd omcmb && npm test -- --run webcode/src/pages/device/DeviceList/GpsSyncConfirmModal.test.tsx webcode/src/pages/device/DeviceList/GpsSyncTrigger.test.tsx`

Expected: FAIL，缺少快照元数据和紧凑样式标识。

- [ ] **Step 3: 最小实现**

用已有 `formatSystemTime` 格式化 `observedAt`；使用 Task 1 的解析函数只展示用户可理解的槽位，原始参数路径保留在接口中供排障使用；将警告按钮调整为 `20×20`、图标 `13px`，单元格采用紧凑 inline-flex、2px 间距、等宽数字和不换行。

- [ ] **Step 4: 运行定向测试确认 GREEN**

Run: `cd omcmb && npm test -- --run webcode/src/pages/device/DeviceList/deviceGpsObservation.test.ts webcode/src/pages/device/DeviceList/GpsSyncConfirmModal.test.tsx webcode/src/pages/device/DeviceList/GpsSyncTrigger.test.tsx webcode/src/pages/device/DeviceList/deviceGpsSyncIndicator.test.ts`

Expected: PASS。

### Task 3: 前端回归验证

**Files:**
- Verify only.

- [ ] **Step 1: 类型检查**

Run: `cd omcmb && npm run typecheck`

Expected: exit 0。

- [ ] **Step 2: 核对变更范围**

Run: `git diff --check && git status --short`

Expected: 无格式错误；仅包含 Issue 101 文件和用户原有 `AGENTS.md` 修改，提交时排除 `AGENTS.md`。
