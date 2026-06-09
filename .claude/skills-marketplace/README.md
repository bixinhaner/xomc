# omc-skills 本地包装市场

把 [`mattpocock/skills`](https://github.com/mattpocock/skills) 暴露为可安装的 Claude Code 插件。

## 为什么需要这个包装

`mattpocock/skills` 仓库只含 `.claude-plugin/plugin.json`，**没有 `marketplace.json`**，
所以不能用 `claude plugin marketplace add mattpocock/skills` 直接添加。
本目录的 `.claude-plugin/marketplace.json` 就是一个最小市场清单，它的插件条目通过
HTTPS（`source: url` → `https://github.com/mattpocock/skills.git`）指向上游仓库。

## 团队约定（shared config + per-machine path）

- **已随仓库提交**：本目录（市场清单）+ 根 `.claude/settings.json` 里的 `enabledPlugins`
  （`"mattpocock-skills@omc-skills": true`，即“启用意图”是共享的）。
- **每台机器各自配置**：市场注册（`extraKnownMarketplaces`，含本机绝对路径）写在
  **`.claude/settings.local.json`**（gitignore，不提交）。

## 新成员一次性接入（在仓库根目录执行）

```bash
claude plugin marketplace add "$(pwd)/.claude/skills-marketplace/.claude-plugin/marketplace.json" --scope local
```

因为 `enabledPlugins` 已在共享 `settings.json` 中，注册市场后插件会自动安装并启用。
下次启动 Claude Code 即可使用 `/mattpocock-skills:<skill>`（如 `/mattpocock-skills:grill-me`、
`/mattpocock-skills:tdd`、`/mattpocock-skills:diagnose`）。

首次建议运行 `/mattpocock-skills:setup-matt-pocock-skills` 完成 issue 跟踪器、triage 标签、
文档存放位置等个性化配置。

## 升级到上游最新

```bash
claude plugin update mattpocock-skills@omc-skills   # 重启 Claude Code 生效
```
