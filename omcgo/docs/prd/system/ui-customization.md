# 系统管理 — UI 定制化（System / UI Customization）PRD

> 文档目的：管理前端 UI 的产品名称、主题色、登录背景图、菜单 Logo 等品牌化定制项，支持运营商按需定制部署。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `UICustomization/index.tsx` 抽取（实现度低，多为占位）|
| 0.2  | 2026-05-08 | Frontend Team | **重写**：与 `UICustomSettings.tsx` 实际代码 1:1 对齐；只保留 5 个已落地字段（产品名/主题色/登录背景/小 Logo/大 Logo）；v0.1 中其余字段移至 §11 未来扩展 |

**关联功能域**：F06 OMC-R 核心 / 品牌化

**页面路由**：`/system/ui-custom`

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面壳 | [omcmb/webcode/src/pages/system/UICustomization/index.tsx](../../../../omcmb/webcode/src/pages/system/UICustomization/index.tsx) |
| 前端表单子组件 | [omcmb/webcode/src/pages/system/SystemConfig/UICustomSettings.tsx](../../../../omcmb/webcode/src/pages/system/SystemConfig/UICustomSettings.tsx) |
| 路由注册 | [omcmb/webcode/src/router/routes.tsx](../../../../omcmb/webcode/src/router/routes.tsx) `system/ui-custom` |
| 后端存储（规划）| 复用 `sys_configs` 表，`category = 'ui_custom'` |
| 共享接口（规划）| `/api/v1/admin/configs`（详见 [config.md §6](./config.md)）|

---

## 1. 业务背景

OMC 是面向运营商的商用产品，三大运营商（CMCC / CTCC / CUCC）和不同集团客户对界面外观有定制需求：

- **品牌化**：替换登录背景图、菜单 Logo、修改产品名称、定制主题色（移动绿/电信蓝/联通红）
- **快速回退**：定制翻车时一键恢复出厂默认外观

**与系统配置（[config.md](./config.md)）的边界**：
- `config.md` 管"运行参数"（密码长度、SMTP 等业务参数）
- 本 PRD 管"外观资产"（产品名、主题色、Logo / 背景图）
- **底层共用 `sys_configs` 表**，仅按 `category = 'ui_custom'` 划分

**生效路径**（规划，当前为前端 mock）：
- 后端：保存到 `sys_configs (category='ui_custom', key='*', value='*')`，图片资产入 MinIO，URL 写回 `value`
- 前端：登录前 `GET /admin/public/configs?category=ui_custom` 拉取 Logo/产品名等需登录前展示的项；登录后通过 ThemeProvider 应用主题色

---

## 2. 页面入口与权限

| 项 | 说明 |
|----|------|
| 路由 | `/system/ui-custom` |
| 导航 | 系统管理 → UI 定制化（i18n key：`nav.system.uiCustom`）|
| 权限 | 仅超级管理员 / 系统配置管理员可见、可保存（暂未接 RBAC，待 [users.md](./users.md) 收口后补）|
| 页面布局 | `ListPageLayout` + 标题"UI 定制化" |

---

## 3. 字段清单（v0.2 已落地）

> 字段名与代码 `defaultUIConfig` 完全一致；下表是当前页面**全部**字段。`category='ui_custom'` 下的扩展字段见 §11。

| key | 中文标签 | 控件 | 必填 | 默认值 | 校验 | 登录前可见 |
|-----|---------|------|------|--------|------|----------|
| `ui_omc_name` | 产品名称 | `<Input>` | ✅ | `BaiOMC` | 非空；长度 ≤ 32（建议）| ✅ |
| `ui_color` | 主题色 | `<ColorPicker format="hex">` + 预览按钮 | ✅ | `#FF4614` | 合法 HEX 颜色 | — |
| `ui_login_background` | 登录背景 | `<Upload.Dragger>` | ❌ | `./images/login/login_bg.png` | 仅 JPG/PNG；**< 1 MB** | ✅ |
| `ui_menu_logo_up` | 菜单收起 Logo（小）| `<Upload.Dragger>` | ❌ | `./images/login/nav_logo_collapse.png` | 仅 JPG/PNG；**< 400 KB** | ✅ |
| `ui_menu_logo_down` | 菜单展开 Logo（大）| `<Upload.Dragger>` | ❌ | `./images/login/logo_big.png` | 仅 JPG/PNG；**< 400 KB** | ✅ |

> 所有 i18n 文本通过 `useT()` 获取，命名空间 `system.ui.*`（详见 [omcmb/frontend-core/src/i18n/zh-CN/index.ts](../../../../omcmb/frontend-core/src/i18n/zh-CN/index.ts)）。

---

## 4. 页面布局（与代码 1:1 对应）

```
┌─ ListPageLayout / 标题：UI 定制化 ──────────────────────────────────┐
│                                                                     │
│  ┌─ 第一行（Row gutter=24） ────────────────────────────────────┐   │
│  │  Col 12：产品名称 [Input]                                    │   │
│  │  Col 12：主题色  [ColorPicker hex] [预览按钮 EyeOutlined]    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ── Divider「图片上传」──                                           │
│                                                                     │
│  ┌─ 第二行（Row gutter=24） ────────────────────────────────────┐   │
│  │  Col 8：登录背景  Card                                       │   │
│  │         extra：Max 1 MB                                      │   │
│  │         Body：Dragger（点击或拖拽上传，支持 jpg/png）         │   │
│  │  Col 8：菜单收起 Logo Card  extra：Max 400 KB                │   │
│  │         Body：Dragger（菜单收起时显示）                       │   │
│  │  Col 8：菜单展开 Logo Card  extra：Max 400 KB                │   │
│  │         Body：Dragger（菜单展开时显示）                       │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ── Divider ──                                                      │
│                                                                     │
│  [恢复默认 UndoOutlined（Popconfirm 二次确认）]                     │
│                                                                     │
│  ── 页面底部居中 ──                                                  │
│  [保存 SaveOutlined]                                                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. 操作清单

| # | 操作 | 触发 | 当前实现 | 目标实现 |
|---|------|------|---------|----------|
| 1 | 加载当前配置 | 页面进入 | **未实现**：表单使用硬编码 `defaultUIConfig` | `GET /admin/configs?category=ui_custom` 后 `form.setFieldsValue` |
| 2 | 主题色实时预览 | 点击「预览」按钮 | **未实现**：仅 `message.info('previewThemeColor')` | 动态修改 CSS 变量 `--ant-color-primary`，仅本页面会话生效，刷新还原 |
| 3 | 上传登录背景 | Dragger 选文件 | 前端校验 < 1 MB；`beforeUpload` 返回 `false` 阻止自动上传，文件留在 `loginBgFile` state | 上传到 MinIO `ui-assets/` bucket，写回 URL 到 `ui_login_background` |
| 4 | 上传 Logo（大/小）| Dragger 选文件 | 前端校验 < 400 KB；同上仅入 state | 同 #3，分别写回 `ui_menu_logo_up` / `ui_menu_logo_down` |
| 5 | 保存 | 底部「保存」按钮 | **mock**：`form.validateFields()` 通过后 `setTimeout(800ms)` + `message.success` | `PUT /admin/configs/batch`（一次提交所有变更项），同时把已上传的文件 URL 写回 |
| 6 | 恢复默认 | 「恢复默认」按钮 + Popconfirm | 前端 `form.setFieldsValue(defaultUIConfig)` + 清空 3 个文件 state；**不写后端** | 二次确认后调 `PUT /admin/configs/batch` 把 5 项 key 回写为出厂默认 |

### 5.1 表单校验规则

- `ui_omc_name`：`required: true`，message 取 `system.ui.pleaseInputOmcName`
- `ui_color`：`required: true`
- 上传类（3 项）：`accept="image/jpeg,image/jpg,image/png"`，`maxCount={1}`，体积上限见 §3
- 体积超限：阻止入 fileList 并 `message.error`

### 5.2 上传组件特殊行为

所有 3 个 `<Dragger>` 当前**不会自动上传**（`beforeUpload` 总是 return `false`）。这是有意行为——文件只暂存到 React state，等待"保存"时统一处理。后端补齐 §7-P0-3 的上传端点后，"保存"按钮内部应：
1. 遍历 3 个 file state，对非空项 POST 上传得到 URL
2. 把 URL 合并进 `form.getFieldsValue()`
3. 调 `PUT /admin/configs/batch`

---

## 6. 接口契约

完全复用 [config.md §6](./config.md) 的 `sys_configs` 接口，按 `category=ui_custom` 过滤。

| Method | 路径 | 用途 |
|--------|------|------|
| GET | `/api/v1/admin/configs?category=ui_custom` | 加载（admin 视图，含全部 5 个 key）|
| GET | `/api/v1/admin/public/configs?category=ui_custom` | 登录前可见子集（产品名 + Logo + 登录背景）|
| PUT | `/api/v1/admin/configs/batch` | 批量保存（依赖 [config.md §7 P0 #1](./config.md) 待补接口）|

### 6.1 文件上传端点（待补）

| Method | 路径 | 入参 | 出参 |
|--------|------|------|------|
| POST | `/api/v1/admin/uploads/ui-asset` | `multipart/form-data { file, kind?: 'login_bg'|'logo_small'|'logo_large' }` | `{ "url": "https://minio.../ui-assets/<uuid>.png" }` |

约束：
- 仅接收 `image/jpeg` / `image/png`
- 体积上限按 `kind` 区分：`login_bg < 1 MB`、其余 `< 400 KB`
- 服务端二次校验（不信任前端）
- 写入 MinIO `ui-assets` bucket（公网可读）
- 返回 URL 直接落入 `sys_configs.value`

---

## 7. 后端补齐 Backlog

### P0（阻塞页面真实生效）
1. **依赖 [config.md](./config.md) P0**：`PUT /admin/configs/batch` + `value_type` 校验
2. **`/admin/public/configs` 端点**：登录前可访问，返回 `is_public = true` 的子集（产品名、3 张图片资产）
3. **`/admin/uploads/ui-asset` 文件上传端点**（§6.1）

### P1（生产可用）
4. **登录前页面消费**：登录页 SPA bootstrap 时拉取 `public/configs`，把产品名注入 `<title>`、Logo / 背景图注入登录组件
5. **ThemeProvider 接入主题色**：登录后 `appStore` 持久化 `ui_color`，全局 ConfigProvider `theme.token.colorPrimary` 读取
6. **资源安全检查**：上传时 magic number 校验（防伪装为 png 的 EXE）、生成新文件名（防路径遍历）、可选 ClamAV 扫描

### P2（增强）
7. **多版本 / 多租户**：未来支持按部署级 `carrier` 区分主题（CMCC 绿/CTCC 蓝/CUCC 红）—— 此处 `carrier` 是部署级标识（来自 `sys_configs` 或环境变量），与 [users.md v1.0 已删除的 `users.carrier`](./users.md) 是不同维度
8. **预览隔离**：保存前生成短链 `/preview?theme=<id>`，分享给评审或客户确认
9. **CSS 变量直出**：`/admin/public/theme.css` 端点直接吐 `:root { --primary: #FF4614; }`，前端 `<link>` 引用，免去 JS bootstrap 闪屏

---

## 8. 验收清单（DoD）

### 前端（v0.2 当前实现）
- [x] 页面渲染 5 个字段 + 预览 / 恢复默认 / 保存按钮
- [x] 表单校验规则按 §3 落地
- [x] 上传体积超限有 `message.error` 提示
- [x] 恢复默认有 Popconfirm 二次确认
- [x] `npm run typecheck` & `lint` 通过

### 后端（v0.2 待补）
- [ ] `sys_configs` 中 `category='ui_custom'` 的 5 条 seed 已写入（key + 默认值与 §3 对齐）
- [ ] 上述 5 个 key 标 `is_public = true`
- [ ] `/admin/public/configs` 端点上线，返回过滤后的子集
- [ ] `/admin/uploads/ui-asset` 端点上线，约束按 §6.1
- [ ] 文件 URL 入 `sys_configs.value`，原始资产入 MinIO（**禁止 base64 入库**，避免 `value` 巨大）

### 联调（v0.2 收口）
- [ ] "保存"成功后刷新页面，5 项配置持久化
- [ ] 登录页能正确显示自定义产品名 + 背景 + Logo
- [ ] 主题色保存后，登录后的 Ant Design 组件主色随之变化

---

## 9. 非目标

- **完整 CSS 自定义**（用户改 CSS 类名 / 注入任意 CSS）：风险大，超出"定制化"范畴，不做
- **主题市场**（用户上传分享主题）：当前业务不需要
- **多套并存的命名主题**（如同时保存"夏季"/"冬季"两套）：v0.x 仅维护一组活动配置
- **A/B 主题灰度**：超出 OMC 业务范围

---

## 10. 风险与依赖

| 类型 | 内容 | 缓解 |
|------|------|------|
| 依赖 | 整页面真正可用前必须等 [config.md](./config.md) P0 收口（`PUT /admin/configs/batch`）| 如先于 config 收口，可临时给 `category='ui_custom'` 单独写一条 PUT 端点 |
| 依赖 | 文件上传端点 `/admin/uploads/ui-asset` 未实现 | P0-3 提单跟进；前端 fileList 当前已为"暂存 + 不自动上传"打好底子，对接后改 `handleSave` 即可 |
| 安全 | 用户上传的图片资产存在恶意文件风险 | §7 P1-6 magic number + 重命名 + 可选 ClamAV |
| 体积 | base64 直接入 `sys_configs.value` 会导致表行膨胀（PG TOAST 触发，索引扫描变慢）| 强制走 MinIO，`value` 仅存 URL |
| 默认值漂移 | 当前 `defaultUIConfig` 是相对路径 `./images/login/...`，依赖前端 bundle 自带的图 | 后端 seed 写真实路径或 MinIO URL；恢复默认操作要与 seed 一致 |

---

## 11. 未来扩展（v0.1 设想，v0.2 暂不做）

> v0.1 PRD 曾设想过下表中的字段，但现网代码未实现。为保持 PRD 与实现一致，移到此章节作为**未来路线图**。如需启用，新建独立 PRD 版本号并在 §3 表中显式登记。

| key | 中文 | 控件 | 备注 |
|-----|------|------|------|
| `logo_dark_url` | 深色模式 Logo | `<Upload>` | 配合 `theme_mode='dark'` |
| `favicon_url` | 浏览器图标 | `<Upload>` | 替换 `/favicon.ico` |
| `theme_mode` | 默认主题模式 | `<Select>` (`light`/`dark`/`auto`) | 接入 ThemeProvider |
| `font_family` | 字体族 | `<Select>` | 系统默认 / Inter / 思源黑体 |
| `font_size_base` | 基准字号 | `<InputNumber>` | 12 / 14 / 16 |
| `density` | 组件密度 | `<Select>` (`compact`/`default`/`comfortable`) | 全局 ConfigProvider `componentSize` |
| `layout_mode` | 布局模式 | `<Select>` (`side`/`top`/`mix`) | 影响菜单位置 |
| `sidebar_collapsed_default` | 侧边栏默认折叠 | `<Switch>` | 影响首次访问的初始状态 |
| `footer_text` | 底部版权文案 | `<Input>` | 显示在 ListPageLayout 底部 |
| `login_announcement` | 登录页公告 | Markdown 编辑器 | 维护期通告等场景 |

---

## 12. 验证步骤（手工 QA）

| 步骤 | 期望结果 |
|------|---------|
| 1. 进入 `/system/ui-custom` | 页面渲染，5 项默认值正确填充 |
| 2. 修改"产品名称"为 `MyOMC`，点保存 | 出现 `保存` 成功提示（v0.2 mock，无后端写入）|
| 3. 选择主题色 `#1677FF`，点"预览" | 出现"预览主题色"提示（v0.2 mock，未真正应用）|
| 4. 上传 1.5 MB 的 jpg 到"登录背景" | 出现"登录背景超出 1MB"错误提示，文件不入列表 |
| 5. 上传 500 KB 的 png 到"小 Logo" | 出现"Logo 超出 400KB"错误提示 |
| 6. 上传 300 KB 的合法 png 到"大 Logo" | 文件出现在拖拽区列表，可"X"移除 |
| 7. 点"恢复默认" → 确认 | 5 项字段回到 §3 默认值，3 个上传列表清空 |
| 8. 清空"产品名称"后点保存 | 出现"请输入产品名"校验错误，不进入保存流程 |

---

## 13. 变更日志

| 版本 | 日期 | 变更 |
|------|------|------|
| 0.1 | 2026-05-06 | 初稿；设想 14 个字段（含 logo_dark/favicon/theme_mode/font/density/layout 等），与实现严重分歧 |
| 0.2 | 2026-05-08 | **重写对齐实现**：精简到 5 个真实字段，逐字段标注校验上限与默认值；§4 给出与代码 1:1 的页面布局；§5 区分"当前实现"与"目标实现"；v0.1 设想字段移至 §11 未来扩展；新增 §10 风险与依赖、§12 手工 QA 步骤 |
