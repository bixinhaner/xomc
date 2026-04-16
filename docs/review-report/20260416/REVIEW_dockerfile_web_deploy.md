# Code Review: Dockerfile.web 多阶段构建修复

**Date**: 2026-04-16
**Reviewer**: Claude (automated)
**Scope**: deploy
**Conclusion**: PASS

## Summary

Dockerfile.web 从两阶段改为三阶段构建，deps 阶段安装依赖后通过 COPY --from=deps 将正确平台的 node_modules 覆盖到 builder 阶段，彻底解决不同构建环境 node_modules 平台不匹配导致 vite not found 的问题。

## Findings

| # | Severity | File | Description |
|---|----------|------|-------------|
| 1 | INFO | Dockerfile.web | 三阶段构建：deps(npm install) → builder(COPY source + COPY --from=deps node_modules + build) → nginx |
| 2 | INFO | Dockerfile.web | deps 阶段保留 Docker 层缓存，package.json/lock 不变时跳过 npm install |

## Checklist

- [x] Dockerfile 结构正确，阶段命名清晰
- [x] 保留 npm 层缓存优化
- [x] 无安全配置变更
