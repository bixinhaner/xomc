# 审查报告 — Issue #559

- 分支: `fix/559-bsc-quicksettings-dedup-tag-feedback`
- 范围: BSC QuickSettings 多实例 / 邻区表新增删除防重复提交 + 行内 Tag 任务状态反馈
- 改动: 7 文件，+505/-77（webcode TSX + frontend-core i18n/types/api + omcgo/data/*.xml）

## 影响面
- 仅触达前端 + 后端运行时加载的 XML 资源；无 Go 业务代码 / SQL / migrations / auth / 新增 endpoint。
- 故不触发 `/security-review`。

## 关键质量检查
- [x] `go build ./...` EXIT=0
- [x] `go test ./...` — 仅 1 项 pre-existing 失败（`internal/device TestParameterTreeHandler_UsesDefaultParamModelForTreeAndChildren/schema_only_includes_actual_parameters`），已切到 main 复测确认同样失败，与本次改动无关。
- [x] `npm run typecheck --workspace webcode` — 130 错误全部 pre-existing（main 同样 130，逐项比对 system/* 与 topology/*，与本次改动文件无交集）。参见 `/memories/repo/ship-preexisting-failures.md` 处置策略。
- [x] `npm run build --workspace webcode` 成功并已部署到 omc-web-1 容器，浏览器自测通过：
  - 三连击只触发 1 次 AddObject mutation
  - 失败 Tag 显示设备侧错误：例如 "Multiple QRXLEVMIN are not supported"
  - 成功 Tag 显示绿色 check-circle

## 代码异味
- 无 `any` / `TODO` / `FIXME` / `console.log`
- 无 `dangerouslySetInnerHTML` / `innerHTML`
- 无 `panic` / SQL 拼接 / `interface{}` 越界（Go 侧未改）
- 无测试被删
- 双层守卫模式 `submittingRef`(同步幂等) + `isSubmitting`(视觉 loading)，try/finally 确保释放
- `waitForTaskTerminal` 含 60s 超时与轮询间隔 1s，Promise resolve 后立即抛弃 timer 句柄
- 无运营商硬编码

## 结论
通过，进入 commit/PR。
