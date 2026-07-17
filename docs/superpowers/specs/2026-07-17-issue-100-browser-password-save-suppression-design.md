# Issue #100 禁止浏览器记录登录密码设计

## 背景与根因

安全配置 `security.isBrowserAutoRecordPass=true` 已能正确到达登录页，但现实现仍渲染原生 `type="password"` 输入框，仅把 `autocomplete` 改为 `new-password`。Chrome 会根据密码框、用户名和登录成功导航识别凭据，因此登录后仍弹出“要保存密码吗”。

旧版 OMC 在该配置开启时使用普通文本输入控件自行显示密码掩码，而不是向浏览器暴露原生密码字段。本次恢复这一行为，仅覆盖 V1 登录/解锁页面。

## 目标

- 配置开启时，登录页 DOM 中不出现原生 `type="password"` 登录输入框。
- 用户输入时仍显示圆点，可通过眼睛按钮临时查看明文。
- 表单拿到的值仍是用户输入的真实密码，现有登录加密和认证链路保持不变。
- 配置关闭时继续使用标准密码框及 `autocomplete="current-password"`。
- 不影响创建用户、重置密码、自助改密及其他密码输入场景。

## 方案

在登录页面目录新增一个专用掩码输入组件：

- 使用 Ant Design `Input`，输入元素始终为 `type="text"`。
- Chromium 环境通过 `-webkit-text-security: disc` 显示掩码；点击眼睛按钮时切换为 `none`。
- 配置开启时设置 `autocomplete="off"`，并避免使用密码类型字段。
- 如果运行环境不支持 `-webkit-text-security`，安全降级为标准 `Input.Password`，避免密码默认明文显示。
- 组件继续由 Ant Design Form 控制值，提交数据结构不变。

登录页按 `publicSettings.preventBrowserAutofill` 二选一渲染：

- `true`：专用非密码类型掩码输入组件。
- `false` 或配置读取失败：现有 `Input.Password`。

## 安全边界

- 该方案是针对受控 Chromium 终端恢复旧版 OMC 行为，不声称网页标准能够关闭所有第三方密码管理器。
- 不把密码写入 localStorage、sessionStorage、日志或隐藏表单字段。
- 密码仍只存在于表单运行时状态，并继续经过现有前端加密流程发送。
- 公开配置接口仍只对白名单中的 `isBrowserAutoRecordPass` 做历史库兜底；其他安全配置不会被公开。

## 测试与验收

自动化测试：

1. 配置开启：密码控件为 `type="text"`、`autocomplete="off"`，掩码样式生效。
2. 配置关闭：密码控件为 `type="password"`、`autocomplete="current-password"`。
3. 配置开启时输入密码并提交，登录流程收到真实密码值。
4. 锁屏解锁模式继续能输入密码并完成导航。
5. 前端 typecheck 与登录页单测通过。

浏览器验收：

1. 删除 localhost 已保存凭据。
2. 开启“禁止浏览器自动记录密码”。
3. 使用 Chrome 登录，确认 DOM 中登录输入不是密码类型。
4. 登录成功后不出现 Google 密码管理工具保存提示。
5. 退出后再次进入登录页，确认密码未自动回填。

## 非目标

- 不修改浏览器或操作系统企业策略。
- 不阻止浏览器扩展读取页面内容；严格终端管控应使用 Chrome 企业策略关闭密码管理器。
- 不重构其他密码输入组件。
