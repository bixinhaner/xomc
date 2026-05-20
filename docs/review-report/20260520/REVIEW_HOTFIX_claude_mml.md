# Review — fix(mml,frontend): RightPanel.tsx import 修复

- 范围：`omcmb/webcode/src/pages/mml/Console/components/RightPanel.tsx`
- Backlog：HOTFIX（构建阻塞）
- 结论：**PASS**

## 背景

c2312c84 通过 git mv 把 `ParamPathExpert.tsx` 重命名为 `ParameterPathCommand.tsx`，但 `RightPanel.tsx` 的 import 路径未同步，导致前端 `vite build` 报 `Could not resolve "./ParamPathExpert"` 直接阻塞 docker 构建。

## 变更

```diff
- import ParamPathExpert from './ParamPathExpert';
+ import ParamPathExpert from './ParameterPathCommand';
```

仅修改 import 来源文件名，本地引用名 `ParamPathExpert` 保持不变（`ParameterPathCommand.tsx` 的 `export default function ParamPathExpert` 与之匹配）。

## 审查清单

- [x] 类型安全：`ParameterPathCommand.tsx` 默认导出函数名 `ParamPathExpert`，签名兼容
- [x] 仅改 import，无业务逻辑变化
- [x] 无对其他文件影响（仅 `RightPanel.tsx` 引用该名）

## 风险

无。
