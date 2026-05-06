# 系统管理 — UI 定制化（System / UI Customization）PRD

> 文档目的：管理前端 UI 主题、Logo、标题、配色、布局等品牌化定制项；支持运营商按需定制部署。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `UICustomization/index.tsx` 抽取（实现度低，多为占位） |

**关联功能域**：F06 OMC-R 核心 / 品牌化

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/UICustomization/index.tsx](../../../../omcmb/webcode/src/pages/system/UICustomization/index.tsx) |
| 前端子组件 | `omcmb/webcode/src/pages/system/UICustomization/UICustomSettings.tsx`（推断） |
| 后端存储 | 复用 `sys_configs` 表，`category = 'ui_custom'` |
| 共享接口 | `/api/v1/admin/configs`（详见 [system-config.md §6](./system-config.md)）|

---

## 1. 业务背景

OMC 是面向运营商的商用产品，三大运营商（CMCC / CTCC / CUCC）和不同集团客户对界面外观有定制需求：

- **品牌化**：替换 Logo、修改产品名称、定制主题色（移动绿/电信蓝/联通红）
- **本地化**：修改默认语言、字号、密度（紧凑/常规/宽松）
- **布局**：导航位置（顶部/侧边）、菜单折叠默认状态

**与系统配置的边界**：
- `system-config.md` 管"参数"（密码长度、SMTP 等运行参数）
- 本 PRD 管"外观"（颜色、Logo、布局），**底层共用 `sys_configs` 表**，仅按 `category = 'ui_custom'` 划分

**生效路径**：
- 后端：保存到 `sys_configs (category='ui_custom', key='*', value='*')`
- 前端：登录前 `GET /admin/public/configs?category=ui_custom` 拉取（含 Logo/产品名等需登录前展示的项）；登录后通过 ThemeProvider 应用

---

## 2. 实体模型

复用 `sys_configs` 表（详见 [system-config.md §2.1](./system-config.md)），`category = 'ui_custom'` 下的 key 集合：

| key | 说明 | value_type | 默认值 | 是否登录前可见 (`is_public`) |
|-----|------|-----------|-------|---------------------------|
| `logo_url` | Logo 图片 URL（或 base64） | string | `/assets/logo.png` | ✅ |
| `logo_dark_url` | 深色模式 Logo | string | `/assets/logo-dark.png` | ✅ |
| `product_name` | 产品名（顶部标题、浏览器 tab）| string | `OMC 网管系统` | ✅ |
| `favicon_url` | 浏览器图标 | string | `/favicon.ico` | ✅ |
| `theme_primary` | 主色调 | string (hex) | `#1677FF` | ✅ |
| `theme_mode` | 默认主题 | string (`light`/`dark`/`auto`) | `light` | — |
| `font_family` | 字体族 | string | 系统默认 | — |
| `font_size_base` | 基准字号 | string (px/rem) | `14px` | — |
| `density` | 组件密度 | string (`compact`/`default`/`comfortable`) | `default` | — |
| `layout_mode` | 布局模式 | string (`side`/`top`/`mix`) | `side` | — |
| `sidebar_collapsed_default` | 侧边栏默认折叠 | bool | `false` | — |
| `footer_text` | 底部版权信息 | string | `© 2026 Baicells` | ✅ |
| `login_bg_url` | 登录页背景图 | string | — | ✅ |
| `login_announcement` | 登录页公告 | string (rich text/markdown) | — | ✅ |

---

## 3. 页面布局

### 3.1 顶部

- 标题：「UI 定制化」
- 右侧按钮：`保存`（验证表单 + 调 `PUT /admin/configs/batch`）

### 3.2 表单分组

按视觉分组（Card / Collapse），所有字段在同一页面：

#### 品牌
- Logo 上传（亮 / 暗）
- 产品名称
- Favicon

#### 主题
- 主色调（ColorPicker）
- 默认主题（亮 / 暗 / 跟随系统）
- 字体族
- 基准字号
- 组件密度

#### 布局
- 布局模式
- 侧边栏默认折叠

#### 登录页
- 登录背景
- 登录公告（Markdown 编辑器）

#### 底部
- 版权文案

---

## 4. 操作清单

| 操作 | 触发 | 接口 |
|------|------|------|
| 加载当前配置 | 页面进入 | `GET /admin/configs?category=ui_custom` |
| 实时预览 | 表单字段变更 | 前端本地 ThemeProvider 应用，**不立即写后端** |
| 保存 | 顶部"保存" | `PUT /admin/configs/batch`（一次写所有变更项）|
| 重置 | 顶部"恢复默认" | 前端 `form.resetFields()` + 二次确认后调 `PUT /admin/configs/batch` 写入默认值 |

---

## 5. 表单字段定义

> 见 §2 表，每个 key 对应一个 Form.Item。

**关键 UI 组件选择**：
- `logo_url` / `favicon_url` / `login_bg_url` / `logo_dark_url`：`<Upload>` + 上传到 MinIO（路径写回 sys_configs）
- `theme_primary`：`<ColorPicker>`（Ant Design 5 内置）
- `font_family` / `theme_mode` / `density` / `layout_mode`：`<Select>`
- `font_size_base`：`<InputNumber>` + 单位选择
- `sidebar_collapsed_default`：`<Switch>`
- `footer_text` / `login_announcement`：`<Input>` / `<Input.TextArea>`（公告可考虑 Markdown 编辑器）

---

## 6. 接口契约

完全复用 [system-config.md §6](./system-config.md) 的接口，按 `category=ui_custom` 过滤：

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/admin/configs?category=ui_custom` | 加载（admin 视图，含全部）|
| GET | `/admin/public/configs?category=ui_custom` | 登录前可见的子集（Logo/产品名/登录页等）|
| PUT | `/admin/configs/batch` | 批量保存（依赖 [system-config.md §7 P0 #1](./system-config.md) 待补接口）|

### 6.1 文件上传专用

| Method | 路径 | 说明 |
|--------|------|------|
| POST | `/admin/uploads/ui-asset` | 上传 Logo/Favicon/背景图，返回 `{ "url": "https://minio.../...png" }`（**待补**）|

---

## 7. 后端补齐 Backlog

### P0
1. **依赖 [system-config.md](./system-config.md) P0**：批量更新接口 + value_type 校验
2. **`/admin/public/configs`** 端点（登录前可访问，返回 `is_public = true` 的子集）

### P1
3. **文件上传端点**（§6.1）：上传到 MinIO 的 `ui-assets` bucket，返回公网 URL
4. **资源安全检查**：上传 Logo/背景图限制类型（image/*）、大小（< 2MB）、扫描恶意内容
5. **多版本/多租户**：未来支持按 `carrier` 区分 UI 主题（CMCC 用绿色，CTCC 用蓝色）

### P2
6. **预览模式**：保存前生成预览链接 `/preview?theme=<id>`，分享给评审
7. **CSS 变量直出**：`/admin/public/theme.css` 端点直接吐 `:root { --primary: #...; }`，前端 `<link>` 直接引用

---

## 8. 验收清单（DoD）

后端：
- [ ] `sys_configs` 中 `category='ui_custom'` 的 seed 数据已写入（含 §2 表所有 key 的默认值）
- [ ] `/admin/public/configs` 仅返回 `is_public = true` 的项

前端：
- [ ] 实时预览不污染其他用户（仅本页面生效，不写后端）
- [ ] 保存后刷新页面，主题持久化
- [ ] Logo 上传到 MinIO 而非 base64 入库（避免 sys_configs.value 巨大）
- [ ] `npm run typecheck` & `lint` 通过

---

## 9. 非目标

- 完整的 CSS 自定义（用户改 CSS 类名）— 风险大，超出"定制化"范畴
- 主题市场（用户上传共享主题）— 当前不需要
