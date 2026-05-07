# Backlog 子目录导航

> 主入口：[`../backlog.md`](../backlog.md) — 当前活跃任务、仪表盘、Wave 3 进行中队列、近 7 天 Done 速览、近 5 条 changelog 速览。
> 本目录承载历史归档与超大 sub-task 表，使主文件保持可读体量（~30 KB）。

## 目录结构

```
docs/project/
├── backlog.md                          # 当前活跃（精简，每日打开）
└── backlog/                            # 本目录
    ├── README.md                       # 本文件
    ├── bootstrap.md                    # Bootstrap 阶段遗留事项（2026-04-20 起）
    ├── changelog.md                    # 完整 backlog 变更日志（按日追加）
    ├── done/
    │   └── 2026Q2.md                   # 本季度（4-6 月）已完成任务的 Closing Evidence 全量
    ├── waves/
    │   ├── wave-1.md                   # Wave 1 · 止血（已完结 8/8 ✅）
    │   └── wave-2.md                   # Wave 2 · 收尾冲刺（已完结 12/13 ✅）
    └── subtasks/
        └── T-0098-data-dict.md         # T-0098 umbrella 的 36 条 sub-task
```

## 维护规则

| 频率 | 触发动作 |
|---|---|
| **每日** | 仅改 `backlog.md`：State / Updated / 新增 Proposed |
| **每次 S7 关闭** | 任务行从 `backlog.md` §3/§4 迁到 `done/<当季>.md`；`backlog.md` §6 速览滚动保最近 5-10 条 |
| **每周一 Triage** | `changelog.md` 追加；`backlog.md` §10 速览滚动保近 5 条 |
| **每 2 周 Sprint Planning** | 从 `backlog.md` §4 Triaged + 必要时本目录 sub-task 表挑选进 Sprint |
| **每季度首日** | 新建 `done/YYYYQN.md`；旧季文件不再写，仅查 |
| **changelog 满 6 个月** | 整文件改名 `changelog-YYYYHN.md`，新文件接续 |

## dev-pipeline skill 兼容性

- `/dev-pipeline status` 数仪表盘 → 主 `backlog.md` §2；本季 Done 数从 `done/<当季>.md` 拼接
- `/dev-pipeline backlog add "<title>"` → 写主 `backlog.md` §5 Proposed（next-id 扫描需覆盖本目录全部 `*.md`）
- `/dev-pipeline pick T-NNNN` → 先在主 `backlog.md` 中查；找不到再 grep `backlog/**/*.md`（兼容已归档 / sub-task 任务）
- `/dev-pipeline done T-NNNN`（S7 收尾）→ 主表删行 + 追加到 `done/<当季>.md` + `changelog.md` 加一行 + 主 §6 速览刷新

## 拆分原由（2026-05-07）

`backlog.md` 原本为 269 KB / 672 行单文件，单 cell 字符达 8K+，Read 工具无法一次读完，pipeline skill 在 status 命令中扫全文低效。本次按"该归档的归档、活跃的留主表"原则拆分为 1 主 + 6 副，主文件压缩到 ~30 KB / ~430 行，单 cell 控制在 200 字符内。详见 commit message `docs(project): backlog 拆分 — 主文件精简至活跃部分，归档移至 backlog/`。
