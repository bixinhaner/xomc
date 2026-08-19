# Review: 修复 LTE Admin State 显示映射

## 审查结论

`PASS_WITH_WARNINGS`

## 变更范围

- `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`
  - 将 LTE `true` 映射为未锁定，将 `false` 映射为锁定。
- `omcmb/webcode/src/pages/device/DeviceList/index.tsx`
  - 与设备详情页保持相同的 LTE 布尔状态映射。

5G 的数值映射保持不变：`1` 为未锁定、`0` 为锁定、`2` 为未锁定、`3` 为关闭中。

## 根因

LTE 后端主要读取 `FAPControl.LTE.AdminState`，其布尔语义为 `true=未锁定`、`false=锁定`。此前前端布尔映射与该语义相反，导致 LTE 设备状态显示错误。

## 验证

- `npm run typecheck`：通过。
- `npx eslint src/pages/device/DeviceDetail/index.tsx src/pages/device/DeviceList/index.tsx`：通过；存在文件原有的 i18n、React effect 和 React Compiler 警告，本次修改未新增相关逻辑。
- `git diff --check`：通过。

## 风险

- 仅调整前端状态标签和颜色，不修改接口、后端数据、数据库或参数值。
- 本次没有可直接覆盖该映射函数的前端单测；当前共享浏览器页面为 GitLab issue 页面，未进行真实设备详情页浏览器验证。