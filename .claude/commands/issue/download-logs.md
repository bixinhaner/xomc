# 下载问题日志

下载指定问题单的附件日志文件到本地，自动解压，并列出文件清单。

## 使用方法
```
/download-logs <问题ID>
```

$ARGUMENTS 为问题 ID（必填），例如：
```
/download-logs 109561
/download-logs 109940
```

---

## 执行步骤

### 1. 读取配置

读取 `.claude/commands/issue/issue-tracker.json` 获取 `base_url`、`token` 和 `log_download_dir`（默认 `/tmp/issue-logs`）。

### 2. 获取附件列表

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  "${BASE_URL}/issues/${ISSUE_ID}.json?include=attachments"
```

从返回的 `issue.attachments` 中提取附件信息：
- `id` — 附件 ID
- `filename` — 文件名
- `filesize` — 文件大小
- `content_url` — 下载地址
- `created_on` — 上传时间
- `author.name` — 上传者

### 3. 筛选日志相关文件

按文件扩展名筛选，**下载**以下类型：
- 日志：`.log`, `.txt`, `.csv`
- 压缩包：`.zip`, `.tar.gz`, `.tgz`, `.gz`, `.rar`, `.7z`
- 抓包：`.pcap`, `.cap`, `.pcapng`
- 配置/数据：`.xml`, `.json`, `.cfg`, `.ini`, `.dat`
- core文件：文件名包含 `core` 或 `dump`

**跳过**以下类型（仅记录不下载）：
- 图片：`.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`, `.svg`
- 文档：`.doc`, `.docx`, `.ppt`, `.pptx`, `.pdf`（除非文件名含 log/日志）
- 视频：`.mp4`, `.avi`, `.mov`

### 4. 下载文件

```bash
mkdir -p "${LOG_DIR}/${ISSUE_ID}"

# 下载每个符合条件的附件
curl -s -H "X-Redmine-API-Key: ${TOKEN}" \
  -o "${LOG_DIR}/${ISSUE_ID}/${FILENAME}" \
  "${CONTENT_URL}"
```

### 5. 自动解压

```bash
cd "${LOG_DIR}/${ISSUE_ID}"

# 根据扩展名自动解压
# .zip
unzip -o "${FILENAME}" 2>/dev/null

# .tar.gz / .tgz
tar -xzf "${FILENAME}" 2>/dev/null

# .gz (单文件)
gunzip -k "${FILENAME}" 2>/dev/null

# .rar
unrar x "${FILENAME}" 2>/dev/null || echo "[跳过] 需要安装 unrar: sudo apt install unrar"

# .7z
7z x "${FILENAME}" 2>/dev/null || echo "[跳过] 需要安装 p7zip: sudo apt install p7zip-full"
```

### 6. 列出文件清单

解压完成后，列出所有文件：

```bash
find "${LOG_DIR}/${ISSUE_ID}" -type f -exec ls -lh {} \; | sort -k5 -h -r
```

输出格式：
```
问题 #${ISSUE_ID} 日志下载完成
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
保存目录: /tmp/issue-logs/${ISSUE_ID}/

下载的附件:
 [OK] system_log.zip (2.3MB) → 已解压
 [OK] trace.pcap (800KB)
 [跳过] screenshot.png (图片文件)

解压后的文件:
  /tmp/issue-logs/${ISSUE_ID}/
  ├── system_log/
  │   ├── syslog.log (15MB)
  │   ├── oam.log (8MB)
  │   └── l3.log (3MB)
  └── trace.pcap (800KB)

总计: 3 个文件下载, 1 个解压, 1 个跳过
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
后续操作:
  /analyze-issue ${ISSUE_ID}      — 分析日志定位问题
  /auto-fix-issues id:${ISSUE_ID} — 分析并自动修复
```

**重要**：
- 这是只读操作（仅下载文件到 /tmp，不修改代码）
- 如果附件列表为空或没有日志类文件，提示用户该问题没有可下载的日志
- 大文件（>100MB）下载前先提示用户确认
