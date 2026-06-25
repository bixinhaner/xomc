# Review: 调整告警列表 SN 列宽

- Issue: #579
- Scope: frontend
- Date: 2026-06-23
- Result: PASS

## Summary

本次变更覆盖三套前端皮肤的告警列表列宽：

- v1 当前告警、历史告警：缩窄操作列，扩大 SN 列。
- v1 设备详情当前告警：缩窄操作列，并补充 SN 列。
- v2 当前告警、历史告警：缩窄选择列，给 SN 列设置更大最小宽度，收紧操作按钮间距。
- v3 当前告警、历史告警：调整 grid 模板，缩窄选择/操作区域，扩大设备/SN 区域。

## Findings

无 CRITICAL / WARNING 发现。

## Checks

- `npm run typecheck --workspace webcode-v2 && npm run typecheck --workspace webcode-v3`：通过。
- `git diff --check`：通过。
- VS Code diagnostics：本次 7 个改动文件无错误。
- `npm run typecheck`：v1 全量检查因仓库既有 Antd 类型问题失败，报错不涉及本次改动文件。

## Residual Risk

未运行浏览器截图冒烟；风险主要在不同屏宽下表格横向空间观感，需要人工在 8081 页面确认 SN 展示效果。
