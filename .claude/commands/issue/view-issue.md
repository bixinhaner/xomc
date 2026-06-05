# 查看问题详情

查看 BaiBM 项目中指定问题单/需求单的完整详情，包括描述、评论、附件列表。

## 使用方法
```
/view-issue <问题ID>
```

$ARGUMENTS 为问题 ID（必填），例如：
```
/view-issue 109561
/view-issue 109940
```

---

## 执行步骤

### 1. 读取配置

读取 `.claude/commands/issue/issue-tracker.json` 获取 `base_url` 和 `token`。若不存在则停止并提示。

### 2. 解析参数

从 $ARGUMENTS 中提取问题 ID。如果未提供或不是数字，提示用户输入正确的 ID。

### 3. 获取问题详情

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/issues/${ISSUE_ID}.json?include=attachments,journals,relations" \
  | python3 -m json.tool
```

### 4. 格式化输出

解析 JSON，按以下结构输出：

```
╔══════════════════════════════════════════════════════════╗
║  #${ID} ${SUBJECT}
╠════���═════════════════════════════════════════════════════╣
║  类型: ${TRACKER}  |  状态: ${STATUS}  |  优先级: ${PRIORITY}
║  模块: ${CATEGORY}  |  严重性: ${SEVERITY}
║  创建人: ${AUTHOR}  |  指派给: ${ASSIGNED_TO}
║  创建时间: ${CREATED_ON}  |  更新时间: ${UPDATED_ON}
║  问题版本: ${ISSUE_VERSION}  |  可��现性: ${REPRODUCIBILITY}
║  链接: ${BASE_URL}/issues/${ID}
╚══════════════════════════════════════════════════════════╝

── 描述 ───────────────────────────────────────────────────
${DESCRIPTION}

── 影响分析 ───────────────────────────────────────────────
${IMPACT_ANALYSIS}（来自自定义字段 id:124）

── 附件 (${COUNT}) ────────────────────────────────────────
 #  文件名                    大小       上传时间        上传者
 1  system_log.zip           2.3MB     2026-03-06     xuwenyong
 2  screenshot.png           156KB     2026-03-06     xuwenyong
 ...

── 评论/历史 ──────────────────────────────────────────────
[2026-03-06 10:30] 张三:
  已确认复现，日志见附件。

[2026-03-06 14:15] 李四:
  初步分析是 xxx 模块的问题。
  ...
```

### 5. 提取关键自定义字段

从 `custom_fields` 数组中提取并显示以下重要字段（如果有值）：

| 字段ID | 字段名 | 说明 |
|--------|--------|------|
| 6 | 严重性/Severity | S1~S4 |
| 16 | 可复现性/Reproducibility | 必现/偶现/难复现 |
| 11 | 问题版本/Issue Version | 发现版本 |
| 124 | 影响分析/Impact Analysis | 影响范围 |
| 71 | 缺陷类型/Defect Type | 功能/性能/接口等 |
| 73 | 问题根因/Root Cause | 根本原因（如已填写） |
| 19 | 修改方案/Modification Solution | 修复方案（如已填写） |
| 1 | 修正结论/Correction Conclusion | 修正结论 |
| 3 | SVN/GIT修订号/Revision Number | 提交记录 |

### 6. 操作提示

在输出末尾提示：
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
后续操作:
  /download-logs ${ID}   — 下载此问题的日志附件
  /analyze-issue ${ID}   — 分析日志并定位代码
  /auto-fix-issues id:${ID} — 自动修复此问题
```

**重要**：这是只读查询命令，不做任何修改操作。
