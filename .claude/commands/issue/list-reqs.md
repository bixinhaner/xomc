# 查看需求单

列出 BaiBM 项目中的需求单（Feature/Requirement）。

## 使用方法
```
/list-reqs [过滤条件]
```

$ARGUMENTS 可选值（空格分隔，可组合）：
- 无参数（默认）— 指派给自己的开放需求单
- `all` — 所有人的开放需求单
- `high` — 仅 P1-High 优先级
- `oam` / `l3` / `mac` / `bsp` / `lmt` / `bsc` / `pdcp` / `fpga` — 按模块筛选
- `new` — 仅 OPEN 状态
- `progress` — 仅 IN-PROGRESS 状态

示例：
```
/list-reqs
/list-reqs all
/list-reqs oam
```

---

## 执行步骤

### 1. 读取配置

读取 `.claude/commands/issue/issue-tracker.json` 获取 `base_url` 和 `token`。若不存在则停止并提示。

### 2. 构建请求

基础 URL（tracker_id=2 为需求单）：
```
${BASE_URL}/projects/baibm/issues.json?tracker_id=2&sort=priority:desc,updated_on:desc&limit=25
```

根据 $ARGUMENTS 追加参数：
- 默认 → `&assigned_to_id=me`
- `all` → 不加 assigned_to_id
- `high` → `&priority_id=3`
- `new` → `&status_id=1`
- `progress` → `&status_id=2`
- 无特定状态过滤时 → `&status_id=open`
- 模块筛选 → 获取结果后本地按 category.name 过滤

### 3. 执行请求

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" "${API_URL}"
```

### 4. 格式化输出

```
BaiBM 需求单 — 指派给: 王勇 | 状态: 开放 | 总计: N 条
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 #   ID       优先级       状态          分类    指派给           标题
──────────────────────────────────────────────────────────
 1   109940   P2-Middle   OPEN         OAM    wangyong1246   2+4 同步源同步失步后处理方案更新
 2   109755   P2-Middle   OPEN         OAM    wangyong1246   COLI需要支持基站参数模糊查询
 ...
──────────────────────────────────────────────────────────
提示: 使用 /view-issue <ID> 查看详情
```

**重要**：
- 这是只读查询命令，不做任何修改操作
- 结果末尾提示用户可用 `/view-issue` 查看详情
