# 查看软件问题单

列出 BaiBM 项目中的软件问题单（缺陷/Bug）。

## 使用方法
```
/list-bugs [过滤条件]
```

$ARGUMENTS 可选值（空格分隔，可组合）：
- 无参数（默认）— 指派给自己的开放问题单
- `all` — 所有人的开放问题单
- `high` — 仅 P1-High 优先级
- `oam` / `l3` / `mac` / `bsp` / `lmt` / `bsc` / `pdcp` / `fpga` — 按模块筛选
- `new` — 仅 OPEN 状态
- `progress` — 仅 IN-PROGRESS 状态
- `reopen` — 仅 REOPENED 状态

示例：
```
/list-bugs
/list-bugs all high
/list-bugs oam
/list-bugs all l3 new
```

---

## 执行步骤

### 1. 读取配置

读取 `.claude/commands/issue/issue-tracker.json` 获取 `base_url` 和 `token`。若不存在则停止并提示。

### 2. 构建请求

基础 URL：
```
${BASE_URL}/projects/baibm/issues.json?tracker_id=1&sort=priority:desc,updated_on:desc&limit=25
```

根据 $ARGUMENTS 追加参数：
- 默认 / `my` → `&assigned_to_id=me`
- `all` → 不加 assigned_to_id
- `high` → `&priority_id=3`
- `new` → `&status_id=1`
- `progress` → `&status_id=2`
- `reopen` → `&status_id=13`
- 无特定状态过滤时 → `&status_id=open`（所有非关闭状态）
- 模块筛选（`oam`/`l3`等）→ 获取结果后在本地按 category.name 过滤（Redmine API 不直接支持 category 筛选）

### 3. 执行请求

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" "${API_URL}"
```

### 4. 格式化输出

用 python3 解析 JSON，输出为可读表格：

```
BaiBM 软件问题单 — 指派给: 王勇 | 状态: 开放 | 总计: N 条
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 #   ID       优先级       状态          分类    指派给           标题
───────────────────────────────────────────────────────────────
 1   109561   P2-Middle   OPEN         OAM    wangyong1246   BaiBM_1.0.32.OXM】稳定性环境出现TR069 CRASH
 2   109612   P2-Middle   OPEN         OAM    wangyong1246   【2+4】导入160个邻区，基站必然重启
 ...
───────────────────────────────────────────────────────────────
提示: 使用 /view-issue <ID> 查看详情 | /download-logs <ID> 下载日志
```

**重要**：
- 不要做任何修改操作，这是只读查询命令
- 结果末尾提示用户可使用 `/view-issue` 和 `/download-logs` 进一步操作
