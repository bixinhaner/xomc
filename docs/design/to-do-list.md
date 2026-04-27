# MML控制台
# 新建 MML 脚本任务页面
1、任务名称，必填，限制为 128 个字符 ✅
2、设备 SN 必填，脚本任务需要执行的设备 ✅
3、选择脚本，本次执行的任务内容 ✅
4、选择执行方式 改为执行方式 ✅
5、执行策略，分别对设备是否在线进行配置，其中离线设备的配置是：设备 60 分钟内上线执行，调整对应页面文案 ✅

以上需求需同时应用于，MML 控制台页面的保存脚本，这两个页面的功能是一样的，保存脚本时自动填充任务名称、设备 SN、任务内容，用户均可手动调整 ✅

---
## 实施记录（2026-04-27）

实现位置：`omcmb/webcode/src/pages/mml/components/ScriptTaskDrawer.tsx`（ScriptTask 与 Console 共用同一 Drawer，一处修改两处生效）。

| # | 需求 | 改动 |
|---|------|------|
| 1 | 任务名 必填 + 128 字符 | `maxLength={128}` + `showCount` + 新校验规则 `{ max: 128 }` |
| 2 | 设备 SN 必填 | `handleSubmit` 改为：`deviceSns.length === 0` 直接 toast 拦截，不再依赖 commands 是否为空 |
| 3 | 选择脚本副说明 | TextArea/Upload 下方加 `mml.scriptDescTip = "本次执行的任务内容"` |
| 4 | 执行方式 | 段落标题 key 由 `mml.selectExecuteMethod` 改为新增的 `mml.executeMethod = "执行方式"` |
| 5 | 离线设备文案 | Checkbox 文案改 `common.enable`（"启用"），InputNumber 后缀改 `mml.minutesOnlineExecute = "分钟内上线则执行"`；min 由 20 放宽到 1 |
| 6 | Console 入口可手动调整命令 | 新增 `scriptContent` state + `handleScriptContentChange`，TextArea 去掉 `readOnly`，onChange 同步重新拆行 `parsedCommands` |

i18n 同步：`omcmb/frontend-core/src/i18n/{zh-CN,en-US}/index.ts` 新增 4 条 key，旧 key（`mml.selectExecuteMethod` / `mml.waitOnlineRetry`）保留以便他处继续引用。
