# Codex 版 Matt Pocock 技能集

这是 `.claude/skills-marketplace` 中旧 Claude Code marketplace 包装的 Codex 本地替代版本。

原 Claude 包装通过 Claude marketplace 条目暴露 `mattpocock/skills`。Codex 需要本地插件
manifest 和本地 skill 文件，所以这里把本机已经可用的 skill 副本 vendored 到项目目录中。

## 已包含的 Skills

- `diagnose`
- `grill-me`
- `grill-with-docs`
- `handoff`
- `tdd`
- `to-issues`
- `to-prd`
- `triage`
- `zoom-out`

## 尚未纳入

旧 Claude README 还提到了一些上游 skill，例如 `prototype`、`caveman`、`teach`、
`write-a-skill` 和 `setup-matt-pocock-skills`。本次转换使用的本机 Codex skill cache
里没有这些内容，所以暂未包含。

## Codex Marketplace

仓库本地 marketplace 文件是：

```text
.agents/plugins/marketplace.json
```

它通过下面的相对路径指向本插件：

```text
./plugins/mattpocock-skills
```

团队成员在仓库根目录执行下面命令即可把本地 marketplace 加入 Codex：

```bash
codex plugin marketplace add .agents/plugins
```

执行后可用 `codex plugin marketplace list` 确认 Codex 已识别该 marketplace。

## 来源与许可

这些 skill 来自 `mattpocock/skills`，按 MIT License 分发。本目录保留了对应
`LICENSE` 文本，后续同步上游内容时应一并检查许可和来源说明是否仍然准确。
