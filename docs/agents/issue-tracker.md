# Issue tracker: GitLab

Issues and PRDs for this repo live as GitLab issues on host `192.168.10.16`, project `netmanager/xomc`. Prefer the `glab` CLI for issue and merge-request operations when it is authenticated for this GitLab host; otherwise use the GitLab Web UI/API for the same actions.

> **任务源决策（2026-07-08 更新）**：**GitLab Issues 是唯一权威的「活」任务源**。所有新需求 / 缺陷 / PRD 走这里（`glab` CLI 或 GitLab Web/API + matt-pocock `to-issues` / `triage` / `to-prd` / `qa`）。
> `docs/project/backlog.md`（旧 T-NNNN 任务清单）已**冻结为历史审计归档**——保留旧流水线的 closing-evidence 追溯链，不再新增 / 更新任务。
> **编排入口 = `/ship`**：全流程由它按阶段调用 matt-pocock 标准套件（`to-prd` / `to-issues` / `triage` / `tdd`...）在 GitLab Issues 上运作。旧 `/dev-pipeline` 与内部 BaiBM `issue:*`（Redmine）已下线删除；新任务一律以 GitLab Issues 为准，历史任务追溯查 backlog.md。

## Conventions

- **Create an issue**: `glab issue create --title "..." --description "..."`. Use a file or heredoc for multi-line descriptions.
- **Read an issue**: `glab issue view <number> --comments`.
- **List issues**: `glab issue list --label ready-for-agent --per-page 30` (defaults to open issues). Use `--closed` for closed issues or `--all` for all issues.
- **Comment on an issue**: `glab issue note <number> --message "..."`
- **Apply / remove labels**: `glab issue update <number> --label "..."` / `glab issue update <number> --unlabel "..."`
- **Close**: `glab issue close <number>`.

Infer the repo from `git remote -v` when possible. The canonical SSH remote is `git@192.168.10.16:netmanager/xomc.git`.

## When a skill says "publish to the issue tracker"

Create a GitLab issue.

## When a skill says "fetch the relevant ticket"

Run `glab issue view <number> --comments`, or open the issue in GitLab Web if `glab` is unavailable.
