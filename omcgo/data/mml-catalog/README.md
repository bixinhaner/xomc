# mml-catalog 目录说明

## 角色与边界(CRITICAL)

本目录下的 `cmcc-tdlte-v2.3.json` 等 catalog JSON 文件 **只用于提取命令分组结构**,**不是 path 的真值源**。

| 维度 | 是 / 不是 |
|------|---------|
| ✅ 提取分组结构(chapter / group_code / command_zh_name / has_instance) | **是** |
| ✅ 提取命令身份(command_code / operation_type / rpc_method) | **是** |
| ✅ catalog 来源追溯(generated_from / spec_md_hash) | **是** |
| ❌ 命令实际包含哪些 path | **不是** — 真值源是 `mml_command_sub_fields` 表 |
| ❌ 影响 MML executor 拼 GPV 的 path 列表 | **不是** — executor 走 sub_field JOIN `standard_params` |
| ❌ 影响 T-0170 path 校验 | **不是** — 校验基于 `param_mappings` 字典 |

## 数据流(简化)

```
catalog JSON ──┬─► mml_commands           (command_code / target_paths jsonb,冗余)
               ├─► mml_command_groups     (chapter / group_code)
               └─► standard_params        (从 Paths[] 抽取 standardPath 元属性)
                       ▲
                       │ standard_path_id FK
                       │
               mml_command_sub_fields  ◄──── 真正决定命令字段集的表
                       ▲
                       │ ListByCommand (WHERE is_unsupported = false)
                       │
               MML executor (console_structured.go) ──► GPV path list
```

## 常见误判与正确做法

| 想做什么 | 错误做法(已多次踩坑) | 正确做法 |
|---------|---|---|
| 删除某 LST 命令的某个 path | 改 catalog JSON 的 `target_paths` | 改 `mml_command_sub_fields` 表(DELETE 或标 `is_unsupported=true`) |
| 让某 path 对 BLQ 设备不发出 | 改 BLQ.xml | 同上,改 sub_field 表;BLQ.xml 是 paramModel 字典(another layer) |
| 增加某命令的字段 | 改 catalog 然后 restart app | 写 seed migration INSERT mml_command_sub_fields |

## 涉及该文件的代码注释入口

- `omcgo/internal/mml/catalogloader/loader.go` — Loader 头注释(详尽角色与边界)
- `omcgo/internal/mml/specparser/json_gen.go` — CatalogJSONFile / Group / Cmd 字段注释
- `omcgo/internal/config/parammodel/mmlstandardloader/command_derivator.go` — seed_importer 入口(历史 path,产物已落库)

## 历史教训

- 2026-05-25 T-0173 fault 排查:删 catalog target_paths 5 个 path 没用,真正 take effect 的是 DELETE sub_field 行
- 2026-05-25 T-0174 sweep 自动遍历:加 `is_unsupported` 列、改 repo 过滤,**绕过 catalog 完全不动**
