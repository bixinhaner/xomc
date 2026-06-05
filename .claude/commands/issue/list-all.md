# 查看所有待处理项

列出 BaiBM 项目中指派给自己的所有开放问题（问题单 + 需求单 + 其他）。

## 使用方法
```
/list-all [过滤条件]
```

$ARGUMENTS 可选值（空格分隔，可组合）：
- 无参数（默认）— 指派给自己的所有开放项
- `all` — 所有人的开放项
- `high` — 仅 P1-High 优先级
- `oam` / `l3` / `mac` / `bsp` / `lmt` / `bsc` / `pdcp` / `fpga` — 按模块筛选
- `summary` — 只显示统计摘要，不显示详细列表

示例：
```
/list-all
/list-all summary
/list-all all oam
```

---

## 执行步骤

### 1. 读取配置

读取 `.claude/commands/issue/issue-tracker.json` 获取 `base_url` 和 `token`。若不存在则停止并提示。

### 2. 构建请求

基础 URL（不限 tracker，拉取所有类型）：
```
${BASE_URL}/projects/baibm/issues.json?status_id=open&sort=tracker:asc,priority:desc,updated_on:desc&limit=50
```

参数追加规则同其他 list 命令。默认 `&assigned_to_id=me`。

### 3. 执行请求

```bash
curl -s -H "X-Redmine-API-Key: ${TOKEN}" "${API_URL}"
```

### 4. 格式化输出

先输出**统计摘要**，再输出**分类列表**：

```
BaiBM 待处理项概览 — 王勇 (wangyong1246)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

统计:
  总计: 10 项待处理
  ├── 软件问题单: 8 项
  ├── 需求单: 2 项
  └── 其他: 0 项

按优先级:
  ├── P1-High: 0 项
  ├── P2-Middle: 10 项
  └── P3-Low: 0 项

按模块:
  └── OAM: 10 项

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

如果 $ARGUMENTS 中没有 `summary`，则继续输出详细列表，**按类型分组**：

```
── 软件问题单 (8) ─────────────────────────────────────────
 #   ID       优先级       状态    分类    标题
 1   109861   P2-Middle   OPEN   OAM    【BM_1.0.36】【OXM】11184告警优化
 2   109561   P2-Middle   OPEN   OAM    稳定性环境出现TR069 CRASH
 ...

── 需求单 (2) ─────────────────────────────────────────────
 #   ID       优先级       状态    分类    标题
 1   109940   P2-Middle   OPEN   OAM    2+4 同步源同步失步后处理方案更新
 2   109755   P2-Middle   OPEN   OAM    COLI需要支持基站参数模糊查询

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
快捷操作:
  /view-issue <ID>      — 查看问题详情
  /download-logs <ID>   — 下载问题日志
  /analyze-issue <ID>   — 分析问题并定位代码
  /auto-fix-issues      — 批量自动修复
```

**重要**：这是只读查询命令，不做任何修改操作。
