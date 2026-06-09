# Claude AI 的 git 操作边界（完整矩阵）

> 从根 `CLAUDE.md §8.1` 抽出的完整软/硬约束矩阵。CLAUDE.md 只保留 4 条核心铁律，细节在此。

## 设计哲学

- **软约束**（本文件行为准则）——约束 Claude 的"主动行为"：不自动 push、不自动 pull、不打搅用户。
- **硬约束**（`.claude/settings.json` permissions）——约束技术层面：真正破坏性的命令（`--force` 推送、`--no-verify` 提交、`rm -rf`）无论谁触发都拦住。
- **原则**：用户明确指令 → 软约束解除 → 硬约束仍在。不给硬约束留 ask 缓冲，因为 ask 规则在"用户已明确要求"时反而是冗余噪声。

## 操作边界矩阵

| 操作 | Claude 行为（软约束） | settings.json（硬约束） |
|------|---------------------|----------------------|
| `git status` / `diff` / `log` / `show` / `branch`（只读） | 自动执行 | allow |
| `git add` / `git commit` | 任务完成时自动执行 | allow |
| `git fetch` / `git pull` / `git push`（普通） | **严禁自主触发**。仅在用户明确说"推送/push/上传/更新/同步远端"等指令时执行 | allow（不打搅用户） |
| `git push -f` / `--force` / `--force-with-lease` | **永远不执行**，即使用户要求也先追问理由 | **deny**（硬禁） |
| `git commit --no-verify` | 永远不执行 | **deny** |
| `git rebase -i` / `git reset --hard` / `git branch -D` / `git clean -f` | 需用户明确指令 | ask（二次确认） |
| `rm -rf` / `git config --global` | 永远不执行 | **deny** |

## 行为准则

1. Claude 完成任务后默认**只 commit 不 push**，并在回复中说明"本地已提交（hash xxx），未推送远端，如需推送请告知"。
2. `git pull` 和 `git push` **严禁同一条命令**，必须分步。
3. 推送前若本地有与远端分叉的提交，必须 `pull --rebase` 处理干净再 push。
4. `--force` 推送需要用户**显式且重复确认**——即使用户说了"强推"，也追问一句"确认覆盖远端 X commit 吗？"后再执行。

## 提交示例

```
feat(device): 实现设备列表分页查询与批量操作

What: 新增 device handler 的 List/BatchDelete 接口，支持按 carrier/status 过滤
Why: 设备管理基础功能需求
Impact: 新增 GET /api/v1/devices 和 DELETE /api/v1/devices/batch 端点

Refs: #<issue-number>
Related: F06
```
