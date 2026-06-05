# issue 配置模板说明

这个目录中的 issue 命令默认从 `issue-tracker.json` 读取平台连接信息。

## 推荐用法

1. 先复制模板文件：

```bash
cp /home/wangyong/.claude/commands/issue/issue-tracker.example.json \
   /home/wangyong/.claude/commands/issue/issue-tracker.json
```

2. 再编辑 `issue-tracker.json`，替换成你的实际配置。

## 配置示例

`issue-tracker.example.json` 是脱敏模板，当前内容适配 Redmine：

```json
{
  "platform": "redmine",
  "base_url": "https://<your-redmine-host>/redmine",
  "token": "<your-redmine-api-token>",
  "username": "<your-username>",
  "default_project": "<your-project-key>",
  "filters": {
    "issue_types": ["bug", "defect"],
    "statuses": ["open", "in_progress", "active", "new"],
    "assignee": "me"
  },
  "log_download_dir": "/tmp/issue-logs",
  "max_issues": 20,
  "auto_commit": false
}
```

## 字段说明

- `platform`: 当前问题平台类型，现有命令按 `redmine` 编写。
- `base_url`: Redmine 根地址。
- `token`: Redmine API Token。
- `username`: 当前登录用户名，主要用于显示或辅助过滤。
- `default_project`: 默认项目 key，例如 `BaiBM`。
- `filters.issue_types`: 默认拉取的问题类型。
- `filters.statuses`: 默认拉取的问题状态。
- `filters.assignee`: 默认指派过滤，通常填 `me`。
- `log_download_dir`: 附件日志下载目录。
- `max_issues`: 一次最多拉取多少条问题。
- `auto_commit`: 是否允许自动提交代码，默认建议为 `false`。

## 使用说明

- `/list-all`、`/list-bugs`、`/list-reqs` 会读取 `base_url`、`token`、`default_project`。
- `/view-issue` 会读取 `base_url`、`token`。
- `/download-logs` 会读取 `base_url`、`token`、`log_download_dir`。
- `/analyze-issue`、`/auto-fix-issues`、`/update-issue` 也依赖同一个配置文件。

## 安全注意事项

- `issue-tracker.example.json` 可以保留在仓库或模板目录中。
- `issue-tracker.json` 包含真实 token，不建议提交到仓库。
- 如果需要共享模板，只共享 `issue-tracker.example.json`，不要共享真实 `issue-tracker.json`。