# 北向 compact scenario XML

本目录存放 xomc 北向简版场景 XML 历史设计附件。当前推荐方案见 `../page-config-redesign-20260731.md`，运行配置不再以 XML 为入口。

这些文件只保留输出契约和调度契约：

- scenario code and legacy source
- endpoint group reference
- compression and data version defaults
- file domain, format, period, cron, remote path template, filename template
- object list, technology split, and optional output profile override
- explicit log scenario options where old behavior depended on scenario keys

它们不携带旧 SQL、Mongo Bson、数据库连接串或字段长清单。当前页面化方案只把这些 XML 当作内置模板和映射覆盖率报告的来源，不把 XML 写入运行时任务。

Profile resolution rule:

```text
profileSet + domain + object + format + tech/profile override -> output profile
```

旧 profile set 是 `baicells-legacy-v1`。在历史 XML/Profile 方案中，当 `Object` 没有显式指定 `profile` 时，导入器按 `domain`、`code`、`format` 和可选 `tech` 解析 profile。

`local-s0001.xml` 到 `local-s0017.xml` 只覆盖当前项目需要内置成页面模板的 CM/PM/MR 文件场景。

Profile catalog：

- `profiles/baicells-legacy-v1.xml` 保留旧 XML/Profile 方案显式引用或隐式解析出的 profile key。
- `profiles/README.md` 解释 scenario XML、profile、column set、adapter 的历史边界。
- 大字段 alias 清单属于 column-set 资产或代码生成 fixture，不放回场景 XML。

`legacySource` 只用于迁移追踪，不是运行时依赖，也不应进入调度逻辑。
