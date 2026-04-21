# Playwright MCP + VNC 环境搭建指南

> 目标：让 Claude Code 通过 Playwright MCP 打开有头浏览器（headed），在 VNC 远程桌面中可视化操作前端页面，用于 UI 实时查看和调试。

## 环境前提

| 项目 | 要求 |
|------|------|
| OS | Ubuntu 24.04 LTS（Server 或 Desktop 均可） |
| Node.js | v18+ |
| npm | v9+ |
| Claude Code | 已安装并可用 |

---

## 一、安装 XFCE 桌面环境

如果服务器没有图形界面（纯 Server），需要安装轻量桌面：

```bash
sudo apt update
sudo apt install -y xfce4 xfce4-goodies
```

> `xfce4` 是轻量桌面，`xfce4-goodies` 包含终端等实用工具。安装约 500MB。

---

## 二、安装并配置 TightVNC

### 2.1 安装

```bash
sudo apt install -y tightvncserver
```

### 2.2 首次启动（设置密码）

```bash
vncserver :1 -geometry 1440x900 -depth 24
```

首次运行会提示设置 VNC 连接密码（建议 6-8 位）。如果问 "view-only password" 选 `n`。

### 2.3 配置启动脚本

首次启动后会生成 `~/.vnc/xstartup`，替换为以下内容：

```bash
cat > ~/.vnc/xstartup << 'EOF'
#!/bin/bash
unset SESSION_MANAGER
unset DBUS_SESSION_BUS_ADDRESS
exec /usr/bin/startxfce4
EOF

chmod +x ~/.vnc/xstartup
```

### 2.4 重启 VNC 使配置生效

```bash
vncserver -kill :1
vncserver :1 -geometry 1440x900 -depth 24
```

### 2.5 验证 VNC 运行

```bash
ps aux | grep Xtightvnc
```

应看到 `Xtightvnc :1 ... -rfbport 5901` 进程。

### 2.6 常用 VNC 命令

```bash
# 启动（显示器 :1，分辨率 1440x900）
vncserver :1 -geometry 1440x900 -depth 24

# 停止
vncserver -kill :1

# 修改分辨率需重启
vncserver -kill :1
vncserver :1 -geometry 1920x1080 -depth 24
```

> VNC 端口规则：显示器 `:N` 对应端口 `5900+N`。例如 `:1` → 端口 `5901`。

---

## 三、VNC 客户端连接

在本地电脑使用 VNC 客户端连接：

| 客户端 | 平台 |
|--------|------|
| RealVNC Viewer | Windows / macOS / Linux |
| TightVNC Viewer | Windows |
| Remmina | Linux |
| Screen Sharing | macOS（内置） |

连接地址：`<服务器IP>:5901`，输入 2.2 步骤设置的密码。

---

## 四、安装 Playwright MCP

### 4.1 全局安装 playwright-mcp

```bash
npm install -g playwright-mcp
```

验证安装：

```bash
playwright-mcp --version
# 应输出类似 Version 0.0.70
```

### 4.2 安装 Chromium 浏览器

```bash
npx playwright install chromium
```

> 这会下载 Chromium 到 `~/.cache/ms-playwright/` 目录，约 200MB。

---

## 五、配置 Claude Code 的 Playwright MCP

编辑 `~/.claude.json`，在根级 `mcpServers` 中添加 playwright 配置（全局生效，所有项目可用）：

```json
{
  "mcpServers": {
    "playwright": {
      "type": "stdio",
      "command": "playwright-mcp",
      "args": [],
      "env": {
        "DISPLAY": ":1"
      }
    }
  }
}
```

**关键点**：
- 放在根级 `mcpServers`（与 `projects` 同级），而非某个项目下 — 这样所有项目都能使用 Playwright
- `"DISPLAY": ":1"` — 必须与 VNC 启动的显示器号一致，否则浏览器窗口不会显示在 VNC 桌面中
- `headed 模式`是默认行为，**不需要**传 `--headed` 参数；如需无头模式才传 `--headless`
- `args` 留空即可，playwright-mcp 默认启动 Chromium 有头模式

### 5.1 重启 Claude Code

配置修改后需重启 Claude Code 才能加载新的 MCP 服务器。

### 5.2 验证 MCP 加载

重启后在 Claude Code 中尝试：

```
请用浏览器打开 https://www.baidu.com
```

如果成功，VNC 桌面中应弹出 Chromium 窗口并显示百度首页。

---

## 六、浏览器窗口最大化

Playwright MCP 的 `browser_resize` 工具只调整页面视口，不改变窗口大小。要最大化窗口，需通过 CDP 协议：

让 AI 执行以下操作即可：

```
把浏览器窗口最大化
```

AI 会通过 `browser_run_code` 调用 CDP 的 `Browser.setWindowBounds` 实现最大化。

如果 VNC 分辨率不是 1920x1080，还需同步调整视口：

```
把浏览器视口调整为 1440x900（与 VNC 分辨率一致）
```

---

## 七、故障排查

### 问题：VNC 中看不到浏览器窗口

- 检查 `~/.claude.json` 中 playwright 的 `env.DISPLAY` 是否与 VNC 显示器号一致
- 检查 VNC 是否正在运行：`ps aux | grep Xtightvnc`
- 修改配置后需重启 Claude Code

### 问题：页面显示不全 / 被截断

浏览器视口大于 VNC 分辨率。调整视口匹配 VNC 分辨率：

```
把浏览器视口调整为 1440x900
```

### 问题：playwright-mcp 命令找不到

```bash
npm install -g playwright-mcp
# 或检查全局路径
which playwright-mcp
```

### 问题：Chromium 未安装

```bash
npx playwright install chromium
```

---

## 八、快速操作参考

| 操作 | 告诉 AI |
|------|---------|
| 打开页面 | "用浏览器访问 http://localhost:3000" |
| 最大化窗口 | "把浏览器窗口最大化" |
| 调整视口 | "把浏览器视口调整为 1440x900" |
| 截图 | "截个图看看当前页面" |
| 点击元素 | "点击右上角用户名" |
| 填写表单 | "用用户名密码登录" |
| 关闭浏览器 | "关闭浏览器" |

---

## 九、配置架构说明

Claude Code 的 MCP 配置有多个层级：

| 配置位置 | 作用域 | 说明 |
|---------|--------|------|
| `~/.claude.json` → 根级 `mcpServers` | 全局 | 所有项目共享的 MCP 服务器（**playwright 放这里**） |
| `~/.claude.json` → `projects[path].mcpServers` | 项目级 | 仅特定项目加载 |
| 项目根目录 `.mcp.json` | 项目级 | 需 Claude Code 识别启用 |

---

*最后验证：2026-04-21，环境 Ubuntu 24.04 + TightVNC + XFCE4 + playwright-mcp 0.0.70 + Chromium 1217，全局 mcpServers 配置已验证通过*
