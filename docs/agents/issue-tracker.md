# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues on `github.com/569423176-sketch/goomc`. Use the `gh` CLI for all operations.

> **任务源决策（2026-06-09 定）**：**GitHub Issues 是唯一权威的「活」任务源**。所有新需求 / 缺陷 / PRD 走这里（`gh` CLI + matt-pocock `to-issues` / `triage` / `to-prd` / `qa`）。
> `docs/project/backlog.md`（旧 T-NNNN 任务清单）已**冻结为历史审计归档**——保留旧流水线的 closing-evidence 追溯链，不再新增 / 更新任务。
> **编排入口 = `/ship`**：全流程由它按阶段调用 matt-pocock 标准套件（`to-prd` / `to-issues` / `triage` / `tdd`…）在 GitHub Issues 上运作。旧 `/dev-pipeline` 与内部 BaiBM `issue:*`（Redmine）已下线删除；新任务一律以 GitHub Issues 为准，历史任务追溯查 backlog.md。

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for multi-line bodies.
- **Read an issue**: `gh issue view <number> --comments`, filtering comments by `jq` and also fetching labels.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'` with appropriate `--label` and `--state` filters.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v` — `gh` does this automatically when run inside a clone.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.
