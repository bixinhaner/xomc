# OMC 邮件通知运维手册

## 1. 适用范围

本版本只包含两条真实邮件链路：

1. 告警规则 `notify_email`；
2. KPI Query 模板的周期 CSV 附件订阅。

两条链路共用 `notification.smtp` 和 `notification_history`，并由 worker 异步投递。本版本使用阿里企业邮箱，不使用 Office365。短信、FTP 和 #98018 自动汇总不在范围内。

## 2. 阿里企业邮箱配置

app 和 worker 必须使用同一组配置。主机、端口、发件地址和授权码以邮件管理员提供的租户信息为准，不在代码中硬编码，也不再使用 Office365 参数。

邮件管理员需要提供并确认以下信息。下面只是格式示例，不是真实阿里企业邮箱地址或凭据：

| 项目 | 模拟值 | 需要管理员确认的内容 |
| --- | --- | --- |
| SMTP 主机 | `smtp.qiye.example.com` | 企业租户实际 SMTP 主机，只填主机名，不带 `https://` |
| SMTP 端口 | `465` | 端口及对应 TLS 模式 |
| TLS 模式 | `implicit` | `465` 常见为 `implicit`；`587` 常见为 `starttls`，以管理员答复为准 |
| SMTP 账号 | `omc-alert@example.com` | 独立通知账号，或留空表示使用内网免认证 Relay |
| SMTP 授权码 | `REPLACE_WITH_AUTH_CODE` | SMTP 专用授权码，不优先使用网页登录密码 |
| 发件地址 | `omc-alert@example.com` | 账号允许使用的 From 地址或别名 |
| 来源限制 | `10.20.0.0/16` | 是否需要把各 OMC 服务器出口 IP 加入白名单 |
| 收件限制 | `omc-test@example.com` | 测试环境收件人白名单、单封/每日限额和反垃圾策略 |

### 2.1 单台服务器配置

发布包已带 `deploy/configure-smtp.sh` 和模拟模板 `deploy/smtp.env.example`。在发布包外复制一份仅 root 可读的配置，不要直接修改或提交示例文件：

```bash
sudo install -m 600 /opt/omc/current/deploy/smtp.env.example /root/omc-smtp.env
sudo vi /root/omc-smtp.env
sudo bash /opt/omc/current/deploy/configure-smtp.sh --config /root/omc-smtp.env
sudo bash /opt/omc/current/deploy/configure-smtp.sh --check
```

模拟配置内容如下：

```dotenv
OMCGO_NOTIFICATION_SMTP_HOST=smtp.qiye.example.com
OMCGO_NOTIFICATION_SMTP_PORT=465
OMCGO_NOTIFICATION_SMTP_USERNAME=omc-alert@example.com
OMCGO_NOTIFICATION_SMTP_PASSWORD=REPLACE_WITH_AUTH_CODE
OMCGO_NOTIFICATION_SMTP_FROM=omc-alert@example.com
OMCGO_NOTIFICATION_SMTP_TLS_MODE=implicit
OMCGO_NOTIFICATION_SMTP_TIMEOUT=10s
OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES=20971520
```

脚本不会 `source` 密钥文件或输出密码；它会校验参数、备份并原子更新 `/opt/omc/current/deploy/.env`，然后用 `svc.sh start app worker` 收敛 app/worker，使新环境变量生效。`docker compose restart` 不会重读环境变量，不能用于本次变更。配置相同的脚本重跑会跳过文件写入，但仍执行 `docker compose up -d` 状态收敛；若上一次因 Docker 或资源契约异常未应用成功，重试不会假成功。

只检查当前格式或紧急关闭 SMTP：

```bash
sudo bash /opt/omc/current/deploy/configure-smtp.sh --check
sudo bash /opt/omc/current/deploy/configure-smtp.sh --disable
```

`--check` 只检查本机配置格式，不验证 DNS、网络、认证或真实投递。每台独立 OMC 服务器首次接入时仍需应用一次配置；以后正常安装升级会继承这些 SMTP 键，不需要每个版本重新配置。

### 2.2 多环境、多服务器策略

不要在服务器之间复制完整的部署 `.env`，其中还包含数据库、JWT、MinIO 等与单机绑定的秘密。应把 SMTP 作为独立秘密配置，按环境分组管理：

| 环境 | 建议账号/Relay 策略 | 收件策略 | 发布策略 |
| --- | --- | --- | --- |
| 个人测试 | 独立测试账号或测试 Relay | 只允许本人/测试邮箱 | 默认关闭，验证时临时启用 |
| 共享测试 | 公共测试账号或测试 Relay | 固定测试白名单，禁止真实客户地址 | 可批量部署，先在一台 canary 验证 |
| 生产 | 生产专用账号；更推荐内网生产 Relay | 正式收件人及发送限额由邮件侧控制 | 加密保管凭据，逐台/分批执行并验收 |

服务器较少时，在每台机器上传对应环境的 `/root/omc-smtp.env`，重复执行同一个 `configure-smtp.sh` 即可。服务器较多时，推荐用现有 Ansible/堡垒机流水线统一执行，而不是再写一个保存密码的 SSH 循环脚本：

1. inventory 分为 `omc_personal_test`、`omc_shared_test`、`omc_prod`；
2. SMTP 文件放在 Ansible Vault 或企业密钥系统，不进入 Git；
3. 临时下发文件权限设为 `0600`，远端执行 `configure-smtp.sh --config <临时文件>`，执行后删除临时文件；
4. 密钥相关任务启用 `no_log: true`，生产使用 `serial: 1` 或小批次滚动；
5. 先 `--check`，再按第 3、4 节完成真实业务验收，失败时停止后续批次。

当前提交只提供每台服务器内可幂等执行的配置入口，不擅自引入 Ansible inventory、服务器地址或企业密钥系统；这些属于部署现场资产，确定现有发布平台后再接入。

### 2.3 推荐的内网 SMTP Relay

多台 OMC 的长期优选方案是在内网建设一个 SMTP Relay：企业邮箱账号/授权码只保存在 Relay，所有 OMC 服务器只连接 Relay。Relay 根据来源 IP、发件地址和环境限制收件人，再统一转发到阿里企业邮箱。这样授权码轮换只改 Relay 一处，不必登录每台 OMC。

OMC 已支持免认证 Relay：`USERNAME` 和 `PASSWORD` 必须同时留空；`HOST` 指向 Relay，端口和 `TLS_MODE` 按 Relay 配置。生产优先使用 `starttls` 或 `implicit`；只有安全团队批准的隔离内网才使用 `none`。测试和生产应使用不同 Relay 策略或至少不同来源网段、From 地址和收件白名单，防止测试告警发给真实用户。

若暂时没有 Relay，先采用每环境一份加密秘密 + 幂等脚本/Ansible 分发。以后切换 Relay 仍只需替换同一组 SMTP 参数，不改业务代码。

### 2.4 应用配置键

发布环境由下列环境变量覆盖 YAML，app 和 worker 都会接收同一份值：

```text
OMCGO_NOTIFICATION_SMTP_ENABLED
OMCGO_NOTIFICATION_SMTP_HOST
OMCGO_NOTIFICATION_SMTP_PORT
OMCGO_NOTIFICATION_SMTP_USERNAME
OMCGO_NOTIFICATION_SMTP_PASSWORD
OMCGO_NOTIFICATION_SMTP_FROM
OMCGO_NOTIFICATION_SMTP_TLS_MODE
OMCGO_NOTIFICATION_SMTP_TIMEOUT
OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES
```

对应的应用 YAML 结构为：

```yaml
notification:
  smtp:
    enabled: true
    host: "<阿里企业邮箱 SMTP 主机>"
    port: 465
    username: "omc-alert@example.com"
    password: "${ALIBABA_ENTERPRISE_MAIL_PASSWORD}"
    from: "omc-alert@example.com"
    tls_mode: "implicit"
    timeout: 10s
    max_attachment_bytes: 20971520
```

TLS 模式：

- `implicit`：连接建立时立即 TLS，常用于 465；
- `starttls`：先建立 SMTP 连接，必须成功升级 TLS 才会认证；
- `none`：默认禁止用于生产；仅限安全团队批准、网络隔离且按来源 IP 授权的内网 Relay。

密码必须从 Secret 注入，不写入 Git、工单、日志或截图。

## 3. 启用步骤

1. 保持 app 和 worker 的 `enabled=false` 完成发布和基线建库。
2. 配置阿里企业邮箱测试账号和测试收件人，通过 `configure-smtp.sh` 重建 app 与 worker。
3. 在“告警规则”中新建一条低风险规则：选择告警标识和设备/设备组，将执行动作设为“邮件通知”，填写测试收件人并启用；验证新增与清除邮件的主题、时间、原因和处理建议。
4. 在 KPI Query 列表中只对测试模板开启邮件订阅，检查 CSV 附件与页面查询窗口一致。
5. 观察至少一个完整发送周期后，再逐步扩大收件人。

KPI 订阅时间按系统 PM 时区计算；报表窗口只取已闭合的 15 分钟、小时或日周期，不包含尚未结束的当前窗口。

告警邮件规则最多允许 50 个收件地址，前后端都会校验并按大小写不敏感去重。把执行动作改成非邮件动作时，原收件人会被清空。老 OMC 的邮件总开关、默认收件人叠加和周期汇总参数不在本版本中使用。

规则可选绝对生效时间范围。开始和结束必须同时填写且结束晚于开始；规则只在左闭右开区间 `[开始, 结束)` 内匹配，清空时间范围后恢复长期有效。

### 3.1 与老 OMC 13.0.5 操作的对应关系

2026-08-12 已在 189 测试环境只读走查，所有编辑页均使用 `Cancel` 退出，未保存、未发送邮件：

1. 老系统“告警模板 → 告警源/设备或设备组/告警项”对应新系统告警规则的匹配条件；
2. 老系统模板级 `Email Notification` 和模板收件人，对应新系统规则动作 `notify_email` 及规则收件人；
3. 老系统模板启停对应新系统规则启停；
4. 老系统 KPI 模板 `Regular Report → Day/Hour/15Min → Email` 对应新系统 KPI Query 定时报表订阅和 CSV 附件投递；
5. 老系统邮件总开关、默认收件人合并、`Interval/Tolerance` 汇总，以及 KPI 的 FTP 通道均不迁移。部署级 `notification.smtp.enabled` 是基础设施开关，不是业务级邮件总开关。

## 4. 排查顺序

1. DNS：app/worker 容器内能否解析 SMTP 主机。
2. 网络：出站策略和防火墙是否允许目标端口。
3. TLS：检查 `tls_mode`、证书链、证书主机名和系统时间；STARTTLS 未被服务器广告时系统会直接失败，不会降级明文。
4. AUTH：检查账号、授权码、账号锁定和 SMTP 服务开关。
5. MAIL FROM：`from` 必须是账号允许的发件地址或别名。
6. 附件：检查导出任务、MinIO 对象和 `max_attachment_bytes`；超限邮件不会进入 SMTP DATA。

日志只记录收件人数量和错误分类。不应出现密码、完整收件人列表或邮件正文。

## 5. 状态查询

```sql
-- 最近邮件投递及尝试次数
SELECT id, business_type, business_id, status, retry_count,
       attempted_at, sent_at, error_message, created_at
FROM notification_history
WHERE channel = 'email'
ORDER BY created_at DESC
LIMIT 100;

-- 告警邮件异步任务
SELECT id, status, attempt, max_attempts, scheduled_at,
       heartbeat_at, error_message, updated_at
FROM async_jobs
WHERE job_type = 'alarm_email_notification'
ORDER BY created_at DESC
LIMIT 100;

-- KPI 订阅最近运行结果
SELECT id, query_template_name, period, window_start, window_end,
       status, export_task_id, attachment_name, notification_history_id,
       error_message, created_at
FROM pm_query_report_runs
ORDER BY created_at DESC
LIMIT 100;
```

KPI run 的 `export_failed` 表示 CSV 生成失败，`delivery_failed` 表示对象读取、SMTP 投递或成功状态回填失败。异步任务最多重试三次；已有成功 `export_task_id` 时只重投邮件，不重复生成 CSV。

每个收件地址使用独立 SMTP envelope、独立幂等键和独立历史记录。单个地址拒收不会阻断其他地址；异步任务重试时跳过已成功地址，只重试失败地址。

`pending` 投递若超过 4 分钟仍未结束，下次异步任务重试可接管；该时间短于通用异步任务 5 分钟 zombie 接管阈值，确保第一次恢复重试即可重新投递。接管通过数据库条件更新原子抢占，多个 worker 同时重试时只有一个能进入 SMTP。同一逐地址幂等键已 `sent` 或尚在新鲜 `pending` 时不重复发送。

SMTP 的成功边界是服务器对 `DATA` 内容返回最终成功响应；之后 `QUIT` 失败只表示连接清理异常，不会触发重发。通用异步任务最后一次投递仍失败时，通知历史进入 `dead_letter`，该状态不会被当作成功，也不会自动重发。

SMTP 本身没有跨系统幂等协议，因此无法承诺严格的 exactly-once：如果服务器已接受 `DATA`，而进程在通知历史写入 `sent` 前崩溃，超时接管仍可能再次投递。出现收件人反馈重复、任务终态与邮箱实际收件不一致时，应以 `dedup_key`、业务 ID、`attempted_at` 和 SMTP 服务端日志联合核对；确认邮件已被服务器接受后，不要直接重新创建同一业务任务。

通知历史包含完整收件地址和渲染后的邮件正文，HTTP 查询接口仅超级管理员可访问。普通用户即使拥有 `Alarm.View` 也不能读取通知历史。

KPI 调度器恢复时不会逐个追发全部过期窗口；每个订阅只折叠到最近一个已经闭合的自然窗口，并把 `next_run_at` 一次推进到未来，避免长时间停机后形成邮件风暴。

## 6. 回退

1. 先禁用 KPI 邮件订阅和告警 `notify_email` 规则。
2. 再执行 `configure-smtp.sh --disable`，由脚本关闭基础设施开关并重建 app/worker。
3. 不停止告警入库、PM 采集、KPI 查询或通用 worker。
4. 未封版阶段数据结构已折入 `000001_init_schema.sql`；测试环境按整套基线重建，不新增 `000002+` 回退迁移。

## 7. 发布门禁

- 后端单测和异步重试测试通过；
- 基于 `000001` 的空库重建通过；
- 前端类型检查、相关测试和真实浏览器验证通过；
- 通知历史接口使用普通 `Alarm.View` 账号验证为拒绝访问；
- 两个收件人中一个模拟拒收时，另一个仍成功且重试只触达失败地址；
- 测试 SMTP 完成中文告警邮件、CSV 附件、错误密码和附件超限场景；
- 安全日志检查无凭据、完整收件人或正文泄露。

### 7.1 当前验证状态（2026-08-13）

本地门禁已完成：空库三个 `000001` 基线和旧库 MainReconcile 通过，app/worker/web 构建及健康检查通过，前端 293 个测试文件、1973 个用例串行全绿。真实 worker 已完成告警产生后的异步邮件、逐收件人部分失败重试、中文处理建议、KPI 自然窗口 CSV 附件、附件超限、认证配置异常和普通用户 403 验证。

2026-08-13 补充门禁：邮件并发重试改为原子抢占后，`internal/notification` 竞态检查、相关后端包测试与 `go vet`、整个 Go 模块跨包编译检查均通过；前端类型检查、改动文件 ESLint、生产构建和 293/1973 全量测试再次通过。

本地生产页面当前被测试库缺少 License 限制在 License 恢复页；V1 Mock 已完成告警邮件规则无保存交互并确认控制台无错误或警告。此环境限制不影响已完成的后端真实链路验证，但正式环境仍必须按本节门禁补做受 License 保护页面的真实 HTTP 回显检查。

本次 UI 状态收口修改后，V1 Mock 已重新启动，但应用内浏览器的 URL 安全策略拒绝自动重载 `127.0.0.1:3300`；因此本次抽屉关闭重开和 KPI 订阅的最终页面回归仍需人工刷新已保留的本地页面补验，不将自动化阻断冒充为通过。

尚未完成且不能用本地模拟替代的唯一外部项，是使用真实阿里企业邮箱测试账号、授权码、白名单收件人和生产出站网络进行 SMTP 认证与投递。拿到凭据前保持 `notification.smtp.enabled=false`；不得改回 Office365，也不得把测试密码写入 Git、普通应用配置、工单或日志。真实凭据只能进入权限为 `0600` 的临时 Secret 文件或企业密钥系统。
