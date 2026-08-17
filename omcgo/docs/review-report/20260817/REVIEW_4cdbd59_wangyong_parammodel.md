# 参数模型多实例属性修正审查报告

## 审查结论

PASS

- CRITICAL：0
- WARNING：0
- INFO：1

## 改动摘要

- 依据中国移动 LTE/5G 南向接口规范及 LTE、NR 设备全量参数集，补齐 BLN IPSec、异频载波、LTE 邻区和 BaiBNQ DSCP 列表的可写多实例对象。
- 校准 BLN 邻区、载波参数范围与默认值，以及 BLN/BaiBNQ 快速设置实例上限。
- 同步运行时参数模型与交付副本，并增加设备参考集和快速设置实例上限回归测试。

## 审查结果

### 正确性

- XML `totalEntries` 与对象、参数实际数量一致。
- 新增对象均使用 `{i}` 通配符、尾随点和 `READ_WRITE`，满足 AddObject/DeleteObject 路径校验。
- BaiBNQ DSCP、VLAN 优先级范围与 NR 设备参考集一致。
- BLN 载波、邻区字段范围与 LTE 规范及设备参考集一致。

### 兼容性

- 标准参数路径保持不变，不影响快速设置参数选择、MML 参数编译和参数同步叶子路径。
- 仅补齐对象映射和校准实例上限，不新增接口、数据库迁移或认证逻辑。
- 运行时模型和交付模型同步修改，避免部署数据与交付文件漂移。

### 测试覆盖

- `go build ./...`：通过。
- `go test -count=1 ./internal/config/parammodel ./internal/quicksettings ./internal/acs ./internal/mml ./internal/provision ./internal/paramsync`：通过。
- 快速设置邻区新增、IPSec 提交前端用例：21/21 通过。
- Docker Compose 热部署成功，web/app/acs 正常运行，首页返回 HTTP 200。
- 部署后数据库确认 BLN 三个对象和 BaiBNQ DSCP 对象、范围、默认值已加载。

## INFO

- 全量 `go test ./...` 的未改动告警定义包存在既有计数断言差异（期望 442、实际 443）；与本次参数模型改动无依赖关系，相关改动包均已单独通过。

## 风险评估

风险较低。主要行为变化是快速设置允许的最大实例数与设备能力对齐；实际增删仍由设备侧 ACS 返回结果约束。
