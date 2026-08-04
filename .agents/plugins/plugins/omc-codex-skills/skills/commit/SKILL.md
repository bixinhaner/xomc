---
name: commit
description: 为已完成并验证的 OMC 改动创建规范 Git 提交。用户说“提交代码”“帮我 commit”“创建提交”或在 OMC 交付流程进入 P8 时使用；检查暂存范围、分支安全、提交信息和 Issue 可追溯性，但不推送或创建 MR。
---

# OMC 规范提交

只提交本次任务明确相关且已完成验证的改动。可以自动暂存明确属于本次任务的文件，但提交前必须检查暂存范围；不要推送远端或创建 MR。

## 1. 提交前检查

- 在 `xomc/` 仓库根目录检查当前分支、`git status` 和暂存 diff。
- 工作在 `main` 或 `master` 时停止：说明不能直接在主分支提交，等待用户切换或创建功能分支。
- 没有暂存内容时，可以自动 `git add` 本次任务明确相关的文件；暂存后必须列出暂存文件并检查 staged diff。
- 自动暂存时不要使用 `git add .`、`git add -A` 或目录级粗粒度暂存；只能逐个列出文件路径。
- 如无法明确判断某个文件是否属于本次任务，停止并请用户选择。
- 暂存范围混入无关改动、敏感信息、生成物或不应提交的本地配置时停止并说明原因。
- 确认适用测试和审查已完成；没有证据时如实标为未完成，等待用户决定是否继续。

## 2. 生成提交信息

使用中文 Conventional Commit：

```text
<type>(<scope>): <中文简要描述>
```

- type：`feat`、`fix`、`refactor`、`docs`、`test`、`chore`、`perf`、`build`、`ci` 或 `style`。
- scope 从主要模块推断；范围过杂时省略 scope，不编造。
- 如可从分支名、用户输入或已有提交可靠识别 GitLab Issue，正文加入 `Issue: Closes #NN`；否则写 `Issue: N/A（无关联 Issue）`。
- 正文简要说明 What、Why、Impact；不要加入 Claude 协作者署名或 GitHub 专属字段。

## 3. 执行与核对

- 在用户要求提交后，使用正常 `git commit`；绝不使用 `--no-verify`。
- 若 hook 失败，停止并报告根因，不绕过 hook。
- 提交后检查最新提交哈希、首行消息和文件统计。
- 报告提交结果与关联 Issue；不执行 `git push`、MR 创建或合并，除非用户另行明确要求。
