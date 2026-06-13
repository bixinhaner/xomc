# OMC 业务冒烟测试套件（按业务域拆分）

> 与 `../smoke_test.sh`（RC 冻结 20 用例速检）和 `../e2e_verify.sh`（全量回归）互补：
> 本目录是**按业务域一脚本一文件**的真实业务冒烟，覆盖各域关键路径，可单独跑也可全量跑。

## 用法

```bash
# 单个业务域（默认 http://localhost:8081）
bash smoke_device.sh
bash smoke_alarm.sh http://localhost:18091      # 本机容器栈 app 直连口

# 全量（按依赖顺序执行 + 汇总表）
bash run_all.sh http://localhost:18091
SMOKE_ONLY="device alarm pm" bash run_all.sh http://localhost:18091   # 只跑指定域
SMOKE_SKIP="acs nedirect" bash run_all.sh http://localhost:18091      # 跳过指定域
```

环境变量：`OMC_BASE_URL`（同位置参数）、`OMC_ADMIN_USER`/`OMC_ADMIN_PASS`（默认 admin/admin123）。
ACS 域额外：`OMC_ACS_URL`（默认 http://localhost:7557）。

**退出码**：`0` 全过 / `1` 有 FAIL / `2` 调用错误（栈不可达、登录失败）。

## 业务域清单

| 脚本 | 业务域 | 关键路径 |
|------|--------|---------|
| smoke_auth.sh | 认证与会话 | 健康探针、RSA 加密登录、me/menus、refresh、公开配置、负路径 |
| smoke_acs.sh | F01 南向 TR069 | CPE 模拟器 Inform 全会话、设备自动注册上线、trace 抓包闭环 |
| smoke_device.sh | F06 设备管理 | 列表/详情/枚举/geo、预注册→回收站→恢复→永久删闭环、统一任务队列 |
| smoke_topology.sh | 拓扑与分组 | 分组树/统计/CRUD、设备进出组、topology graph/geo、站点 |
| smoke_config.sh | F02 配置管理 | 模板 CRUD、基线 CRUD、配置任务/邻区、参数树读链路、quicksettings |
| smoke_product.sh | F02 产品/参数字典 | products 匹配路由、param-models translate、builtin 删除 403 |
| smoke_alarm.sh | F04 告警管理 | 活跃/历史/统计、确认↔反确认、过滤规则闭环、告警定义库、事件日志 |
| smoke_pm.sh | F03 性能管理 | counters/metrics/kpi 三层查询、KPI 定义、门限/仪表盘/导出闭环、指标库 |
| smoke_mr.sh | F05 测量报告 | MR 文件/数据/指标、映射 toggle 闭环、测量任务 |
| smoke_software.sh | F06 固件升级 | 固件上传→推荐→删除闭环、升级任务校验负路径 |
| smoke_ufte.sh | F06 统一文件任务引擎 | overview/任务类型治理闭环、任务建(不启动)→列表→删/批删闭环、设备与候选列表、CSV 导出、start 红线不调用 |
| smoke_backup.sh | F06 配置备份 | 任务/计划/FTP/策略、快照、restore 校验负路径、规范别名路由 |
| smoke_dashboard.sh | F06 仪表盘 | summary 及全部图表端点、widgets 读写还原 |
| smoke_ops.sh | F06 运维工具 | 模板闭环、任务/诊断/下载/维护窗口/playbook、RPC 校验负路径 |
| smoke_report.sh | F06 报表 | 定义闭环、生成→记录→下载链路 |
| smoke_mml.sh | F06 MML | 命令检索/组树、render+parse 纯函数、模板闭环、execute 校验负路径 |
| smoke_filemanager.sh | F06 文件管理 | 上传→下载→删除闭环、四类批量下载 |
| smoke_logs.sh | F06 日志类 | 系统/网元报文日志、基站日志、异常重启、统一重启记录 |
| smoke_admin.sh | F06 RBAC | 用户 CRUD+锁定闭环、角色+菜单/API 权限、API Key 闭环、审计 |
| smoke_system.sh | F06 系统配置 | sysConfig 闭环、字典 batch、登录/操作/任务日志、system/info、License |
| smoke_task.sh | 统一任务队列 | 按 SN 查任务/pending/stats、任务详情 |
| smoke_notification.sh | 通知中心 | 消息/未读数、模板闭环、SSE 建连、webhook 入口 |
| smoke_northbound.sh | F08 北向 | 推送目标闭环+熔断、全量/增量同步、三类导出（注意限流） |
| smoke_provision.sh | F09 自动开站 | 任务列表/详情、创建校验负路径 |
| smoke_interop.sh | F10 互操作 | 用例列表、报告导出、run 负路径 |
| smoke_nedirect.sh | F07 网元直连 | 默认未启用整域 SKIP；启用时只测读路径 |
| smoke_retention.sh | 资源保留与上传背压(#318-321) | 系统配置可见可改（10 键）、#318 背压指标+engage/release 503/200、#319 ILM 60天规则、#321 raw_archive 指标、#320 保留双旋钮并存 |

## 设计约定（写新脚本必读）

1. **共享库**：`source lib.sh` → `smoke_init` → `smoke_login` → `req`/断言 → `smoke_summary`。
   信封 `{ret,msg,data}`，**ret=1 成功**；分页 `page/page_size`（admin 域必须显式传）。
2. **自备数据**：脚本不得假设种子数据存在；写路径用 `$SMOKE_TAG` 命名自建实体，结束前清理。
   列表断言区分：builtin 字典类用 `check_list_nonempty`，业务数据用 `check_list_or_empty`。
3. **危险操作禁区**：重启/升级/恢复/MML execute/SPV 等会触达真实基站的端点，
   **只测参数校验负路径**（`check_ret_fail`），绝不真发。
4. **已知 bug** 用 `known_bug` 标注（不计失败），修复后改回正常断言。
5. macOS 自带 bash 3.2：变量后紧跟全角字符必须写 `${var}`。
