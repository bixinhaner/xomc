# 北向 SNMP Net-SNMP 测试步骤文档

> 更新时间：2026-08-11
> 适用范围：系统管理 -> 北向配置 -> SNMP 告警
> 测试工具：Net-SNMP `snmpget`、`snmpgetnext`、`snmpwalk`、`snmpbulkwalk`、`snmptrapd`

本文档用于使用 Net-SNMP 验证当前系统北向 SNMP 告警功能，覆盖 SNMP v2c/v3 的 MIB 查询、Trap、Inform、鉴权失败、页面配置生效和结果查询。

## 1. 功能边界

当前系统 SNMP 告警能力包含两类方向：

| 方向 | 系统角色 | 页面配置项 | 说明 |
|---|---|---|---|
| MIB 查询 | OMC 作为 SNMP Agent | 允许 MIB 查询、监听地址、监听端口 | 第三方工具通过 `snmpget/snmpwalk` 查询 OMC 当前活动告警表。 |
| 告警上报 | OMC 作为 SNMP Trap/Inform 发送方 | 启用告警上报、对端地址、对端端口、通知方式 | OMC 将测试告警或真实告警发送到对端 Trap Receiver。 |

注意：

- `监听地址/监听端口` 是给 `snmpget/snmpwalk` 查询用的 Agent 端口，默认 `0.0.0.0:161`。
- `对端地址/对端端口` 是给 OMC 发送 Trap/Inform 用的目标地址和端口，默认 v2c 是 `162`，v3 是 `163`。
- `启用告警上报` 只控制 Trap/Inform 主动上报；`允许 MIB 查询` 控制 GET/WALK Agent 查询，两者可以独立配置。
- 本地 Docker 部署时，Mac 上启动的 `snmptrapd` 对容器来说建议填 `host.docker.internal`；生产现场填真实网管服务器 IP。
- SNMP v2c 只需要 community，不涉及用户名、认证算法、加密算法。
- SNMP v3 需要安全名、认证协议/密码、加密协议/密码；iReasoning 免费版通常不能测试 v3，建议用 Net-SNMP。

## 2. 前置检查

### 2.1 确认 Net-SNMP 命令

在测试机执行：

```bash
which snmpget
which snmpwalk
which snmptrapd
snmpwalk --version
```

macOS 自带 Net-SNMP 常见版本是 `5.6.2.1`，支持 `SNMPv1/v2c/v3`，但算法比较老：

- v3 认证一般支持 `MD5`、`SHA`。
- v3 加密一般支持 `DES`、`AES`，其中命令行 `-x AES` 对应页面上的 `AES128`。
- 如果现场需要 `SHA256/SHA512/AES192/AES256`，建议安装新版 Net-SNMP 后再测。

### 2.2 确认 OMC 本地地址

本地 Docker 部署常用地址：

| 用途 | 地址 |
|---|---|
| 页面访问 | `http://127.0.0.1:8081` |
| SNMP Agent 查询地址 | `127.0.0.1:161` |
| 容器访问宿主机 Trap Receiver | 页面里填 `host.docker.internal` |

如果在其他机器上测试，将 `127.0.0.1` 替换成 OMC 服务器 IP。

### 2.3 确认 UDP 端口

```bash
docker ps --format 'table {{.Names}}\t{{.Ports}}' | grep -E 'omc-app|161/udp'
```

预期看到 OMC app 容器暴露 UDP `161`。如果页面把 SNMP Agent 监听端口改成其他端口，后续命令也要同步改端口。

## 3. 页面配置口径

进入页面：

```text
系统管理 -> 北向配置 -> SNMP 告警
```

### 3.1 v2c 推荐配置

| 页面项 | 建议值 | 说明 |
|---|---|---|
| 启用告警上报 | 按需开启 | 开启后 OMC 主动向对端发送 Trap/Inform。 |
| SNMP 版本 | v2c | v2c 只显示 community 相关项。 |
| 通知方式 | Trap 或 Inform | 仅启用告警上报时需要配置；现场如无特殊要求，先测 Trap。 |
| 允许 MIB 查询 | 按需开启 | 开启后可用 `snmpget/snmpwalk` 查询活动告警。 |
| 监听地址 | `0.0.0.0` | 仅允许 MIB 查询时需要配置；OMC Agent 监听所有网卡。 |
| 监听端口 | `161` | 仅允许 MIB 查询时需要配置。 |
| Community | `baicells` | 系统默认值，允许用户修改。 |
| 对端地址 | 本机测试填 `host.docker.internal` | 仅启用告警上报时需要配置；OMC 发送 Trap/Inform 的目标主机。 |
| 对端端口 | 本机测试建议 `1162` | 仅启用告警上报时需要配置；避免普通用户绑定 162 权限问题。 |
| 超时/重试 | 默认即可 | 仅 Inform 场景需要关注。 |

### 3.2 v3 推荐配置

| 页面项 | 建议值 | 说明 |
|---|---|---|
| 启用告警上报 | 按需开启 | 开启后 OMC 主动向对端发送 Trap/Inform。 |
| SNMP 版本 | v3 | 页面应显示 v3 专用安全参数。 |
| 通知方式 | Inform 或 Trap | 仅启用告警上报时需要配置；v3 Inform 用于确认回执验证，Trap 用于普通通知验证。 |
| 允许 MIB 查询 | 按需开启 | 开启后可用 v3 查询活动告警。 |
| 监听地址 | `0.0.0.0` | 仅允许 MIB 查询时需要配置；OMC Agent 监听所有网卡。 |
| 监听端口 | `161` | 仅允许 MIB 查询时需要配置。 |
| 安全名 | `northv3` | Net-SNMP 命令中的 `-u northv3`。 |
| 认证协议 | `SHA` | macOS 自带 Net-SNMP 可测。 |
| 认证密码 | `northAuth123` | 建议至少 8 位。 |
| 加密协议 | `DES` 或 `AES128` | macOS 命令行 AES128 用 `-x AES`。 |
| 加密密码 | `northPriv123` | 建议至少 8 位。 |
| 对端地址 | 本机测试填 `host.docker.internal` | 仅启用告警上报时需要配置；OMC 发送 Trap/Inform 的目标主机。 |
| 对端端口 | 本机测试建议 `1163` | 仅启用告警上报时需要配置；避免系统 163 端口权限和占用问题。 |

## 4. MIB 查询测试

MIB 查询验证 OMC 作为 SNMP Agent 是否能被第三方同步查询当前活动告警。

OMC 告警 MIB 根：

```text
1.3.6.1.4.1.53058.1.1
```

OMC 告警表 Entry：

```text
1.3.6.1.4.1.53058.1.1.1.1.1
```

### 4.1 v2c Walk 活动告警表

页面确认：

- v2c 配置的允许 MIB 查询已开启。
- Community 为 `baicells`。
- 监听地址/端口为 `0.0.0.0:161`。

执行：

```bash
snmpwalk -v2c -c baicells -On -r 0 -t 5 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

预期：

- 有活动告警时，返回多个 OID 和字段值。
- 没有活动告警时，可能返回空表或 noSuchObject/noSuchInstance，属于正常数据状态。
- 命令不应超时。

示例输出形态：

```text
.1.3.6.1.4.1.53058.1.1.1.1.1.2.87607277 = STRING: "50101"
.1.3.6.1.4.1.53058.1.1.1.1.1.5.87607277 = STRING: "867294050000001"
```

### 4.2 v2c Get 单个字段

先通过 walk 找到一个带索引的 OID，再执行：

```bash
snmpget -v2c -c baicells -On -r 0 -t 5 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1.2.87607277
```

预期返回对应告警的 `alarmUniqueId`。

### 4.3 v2c 鉴权失败验证

```bash
snmpwalk -v2c -c wrongCommunity -On -r 0 -t 3 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

预期：

- 命令超时或返回失败。
- 说明 community 校验生效。

### 4.4 v3 Walk 活动告警表

页面确认：

- v3 配置的允许 MIB 查询已开启。
- 安全名为 `northv3`。
- 认证协议/密码为 `SHA` / `northAuth123`。
- 加密协议/密码为 `DES` / `northPriv123`，或 `AES128` / `northPriv123`。
- 监听地址/端口为 `0.0.0.0:161`。

DES 命令：

```bash
snmpwalk -v3 -l authPriv -u northv3 -a SHA -A northAuth123 -x DES -X northPriv123 -On -r 0 -t 5 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

页面选择 `AES128` 时，Net-SNMP 命令通常写 `-x AES`：

```bash
snmpwalk -v3 -l authPriv -u northv3 -a SHA -A northAuth123 -x AES -X northPriv123 -On -r 0 -t 5 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

预期：

- 有活动告警时返回告警表字段。
- 不应出现 `Timeout` 或 `Unknown user name`。

### 4.5 v3 Get/GetNext/GetBulk

Get：

```bash
snmpget -v3 -l authPriv -u northv3 -a SHA -A northAuth123 -x DES -X northPriv123 -On -r 0 -t 5 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1.2.87607277
```

GetNext：

```bash
snmpgetnext -v3 -l authPriv -u northv3 -a SHA -A northAuth123 -x DES -X northPriv123 -On -r 0 -t 5 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

GetBulk：

```bash
snmpbulkwalk -v3 -l authPriv -u northv3 -a SHA -A northAuth123 -x DES -X northPriv123 -On -r 0 -t 5 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

预期：

- `snmpget` 返回指定字段。
- `snmpgetnext` 返回下一个可访问 OID。
- `snmpbulkwalk` 可批量返回告警表，适合较多活动告警时验证分页/批量读取。

### 4.6 v3 鉴权失败验证

错误用户名：

```bash
snmpwalk -v3 -l authPriv -u wrongUser -a SHA -A northAuth123 -x DES -X northPriv123 -On -r 0 -t 3 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

错误认证密码：

```bash
snmpwalk -v3 -l authPriv -u northv3 -a SHA -A wrongAuth123 -x DES -X northPriv123 -On -r 0 -t 3 127.0.0.1:161 .1.3.6.1.4.1.53058.1.1.1.1.1
```

预期：

- 命令超时或失败。
- 说明 v3 用户和认证参数校验生效。

## 5. Trap/Inform 接收测试

Trap/Inform 验证 OMC 是否能把告警主动上报到对端。

### 5.1 v2c Trap Receiver

在 Mac 或测试机启动接收器：

```bash
snmptrapd -f -Lo -n -C -c /dev/null udp:1162
```

页面配置：

- SNMP 版本：v2c。
- 通知方式：Trap。
- Community：`baicells`。
- 对端地址：本地 Docker 测试填 `host.docker.internal`；异机测试填接收机 IP。
- 对端端口：`1162`。
- 启用告警上报：开启。

在页面点击对应 SNMP 目标的“测试发送”。

预期：

- 页面提示发送成功。
- 上报结果中出现成功记录，能力为 SNMP。
- `snmptrapd` 控制台收到 Trap。
- Trap OID 为 `1.3.6.1.4.1.53058.1.1.0.1`。
- VarBind 包含 18 个告警字段。

### 5.2 v3 Trap/Inform Receiver

创建临时 `snmptrapd` 配置：

```bash
cat >/tmp/omc-snmptrapd-v3.conf <<'EOF'
createUser northv3 SHA northAuth123 DES northPriv123
disableAuthorization yes
EOF
```

启动 v3 接收器：

```bash
SNMP_PERSISTENT_DIR=/tmp/net-snmp-persistent \
snmptrapd -f -Lo -n -C -c /tmp/omc-snmptrapd-v3.conf udp:1163
```

页面配置：

- SNMP 版本：v3。
- 通知方式：先测 Inform，再测 Trap。
- 安全名：`northv3`。
- 认证协议/密码：`SHA` / `northAuth123`。
- 加密协议/密码：`DES` / `northPriv123`。
- 对端地址：本地 Docker 测试填 `host.docker.internal`；异机测试填接收机 IP。
- 对端端口：`1163`。
- 启用告警上报：开启。

在页面点击对应 SNMP 目标的“测试发送”。

预期：

- Inform 场景页面返回成功，说明对端有响应。
- Trap 场景页面返回成功，说明发送成功。
- `snmptrapd` 控制台能解码 OMC 告警字段。

如果页面选择 `AES128`，配置文件改为：

```bash
cat >/tmp/omc-snmptrapd-v3.conf <<'EOF'
createUser northv3 SHA northAuth123 AES northPriv123
disableAuthorization yes
EOF
```

然后重新启动 `snmptrapd` 再点击“测试发送”。

### 5.3 预期 Trap/Inform 字段

收到的通知应包含标准 `snmpTrapOID.0` 和 OMC 18 个字段。

通知 OID：

```text
1.3.6.1.4.1.53058.1.1.0.1
```

字段顺序：

| 顺序 | 字段 | OID |
|---|---|---|
| 1 | notificationID | `1.3.6.1.4.1.53058.1.1.1.1.1.1` |
| 2 | alarmUniqueId | `1.3.6.1.4.1.53058.1.1.1.1.1.2` |
| 3 | notificationType | `1.3.6.1.4.1.53058.1.1.1.1.1.3` |
| 4 | eventTime | `1.3.6.1.4.1.53058.1.1.1.1.1.4` |
| 5 | equipmentSDN | `1.3.6.1.4.1.53058.1.1.1.1.1.5` |
| 6 | equipmentName | `1.3.6.1.4.1.53058.1.1.1.1.1.6` |
| 7 | equipmentClass | `1.3.6.1.4.1.53058.1.1.1.1.1.7` |
| 8 | objectSDN | `1.3.6.1.4.1.53058.1.1.1.1.1.8` |
| 9 | objectInstanceName | `1.3.6.1.4.1.53058.1.1.1.1.1.9` |
| 10 | objectClass | `1.3.6.1.4.1.53058.1.1.1.1.1.10` |
| 11 | additionalText | `1.3.6.1.4.1.53058.1.1.1.1.1.11` |
| 12 | deviceVendorOUI | `1.3.6.1.4.1.53058.1.1.1.1.1.12` |
| 13 | specificProblemID | `1.3.6.1.4.1.53058.1.1.1.1.1.13` |
| 14 | specificProblem | `1.3.6.1.4.1.53058.1.1.1.1.1.14` |
| 15 | alarmType | `1.3.6.1.4.1.53058.1.1.1.1.1.15` |
| 16 | perceivedSeverity | `1.3.6.1.4.1.53058.1.1.1.1.1.16` |
| 17 | probableCause | `1.3.6.1.4.1.53058.1.1.1.1.1.17` |
| 18 | additionalInformation | `1.3.6.1.4.1.53058.1.1.1.1.1.18` |

测试发送的样例告警一般包含：

```text
alarmUniqueId = 40123
equipmentSDN = 867294050000001
equipmentName = 867294050000001
objectInstanceName = Cell-1
additionalText = Northbound SNMP test alarm
perceivedSeverity = major
additionalInformation = page-config-test
```

## 6. 真实告警联动验证

页面“测试发送”只验证当前 SNMP 目标配置、协议和字段映射是否可用。现场还需要验证真实告警链路：

1. 在 SNMP 告警页面完成 v2c 或 v3 配置并开启目标。
2. 保持 `snmptrapd` 接收器运行。
3. 触发一条真实设备告警，或使用系统内已有告警模拟/测试入口产生告警。
4. 观察 `snmptrapd` 是否收到通知。
5. 在页面“上报结果”查看 SNMP 成功/失败记录。
6. 使用 `snmpwalk/snmpget` 查询活动告警表，确认能查到相同告警。
7. 清除该告警后，再确认清除通知是否上报；如果页面“清除告警级别策略”选择“清除置 0”，清除通知里的 `perceivedSeverity` 应按策略显示。

真实告警需要重点核对：

- Trap/Inform 的字段顺序仍是 18 个固定字段。
- `notificationType` 能区分产生和清除。
- `eventTime` 使用告警产生时间或清除时间。
- `equipmentSDN/equipmentName/equipmentClass` 来自当前设备和告警数据。
- 页面“上报结果”的报文查看能看到对应 OID 和字段值。

## 7. 页面操作回归清单

每次修改 SNMP 功能后，建议按以下顺序回归：

| 序号 | 操作 | 预期 |
|---|---|---|
| 1 | 打开 SNMP 告警页 | 列表加载正常，不应长时间卡住。 |
| 2 | 新增或编辑 v2c 目标 | 只显示 v2c 必要项，community 默认 `baicells`，可保存。 |
| 3 | 启用 v2c MIB 查询 | `snmpwalk -v2c` 可查询，错误 community 失败。 |
| 4 | v2c 测试发送 Trap | 页面有等待状态，完成后有成功或明确失败原因。 |
| 5 | 新增或编辑 v3 目标 | 显示安全名、认证、加密配置，保存后密钥不明文泄露。 |
| 6 | 启用 v3 MIB 查询 | `snmpwalk -v3` 可查询，错误用户名或密码失败。 |
| 7 | v3 测试发送 Inform | `snmptrapd` 收到通知，页面记录成功。 |
| 8 | 切换 v3 Trap/Inform | 保存后立即按新通知方式发送。 |
| 9 | 查看上报结果 | 多数据量时分页展示，支持按 SNMP/目标过滤，报文可展开查看。 |
| 10 | 分别关闭开关 | 关闭“启用告警上报”后停止 Trap/Inform；关闭“允许 MIB 查询”后停止 GET/WALK 查询。 |

## 8. 常见问题

### 8.1 iReasoning 能测 v2c，但不能测 v3

iReasoning MIB Browser 免费/个人版本常见限制是只能选择 SNMP v1/v2c，不能选择 v3。这是工具版本能力限制，不代表 OMC 不支持 v3。v3 请用 Net-SNMP 或支持 SNMPv3 的商业版工具验证。

### 8.2 `snmpwalk` 超时

按顺序检查：

1. 页面“允许 MIB 查询”是否开启。
2. 监听地址和监听端口是否配置正确。
3. 命令里的版本、community 或 v3 用户密码是否和页面一致。
4. 查询端口是否和页面监听端口一致。
5. Docker 是否暴露 UDP 161。
6. 本机或服务器防火墙是否拦截 UDP。

### 8.3 `snmptrapd` 收不到 Trap/Inform

按顺序检查：

1. `snmptrapd` 是否正在监听页面配置的对端端口。
2. 本地 Docker 测试时，页面对端地址是否填 `host.docker.internal`。
3. 异机测试时，页面对端地址是否填接收机真实 IP。
4. v3 的 `createUser` 配置是否和页面安全名、认证协议、加密协议、密码一致。
5. 对端机器防火墙是否允许 UDP 1162/1163 或现场使用端口。

### 8.4 v3 Inform 页面失败但 Trap 能收到

Inform 需要对端回响应，Trap 不需要。此时重点检查：

- `snmptrapd` 是否支持并正确响应 Inform。
- 对端 v3 用户是否配置正确。
- 页面超时时间是否太短。
- 网络链路是否允许 OMC 收到对端响应包。

### 8.5 页面保存后命令仍按旧配置生效

页面保存会触发 SNMP Agent/发送配置刷新。正常情况下几秒内生效。若仍按旧配置响应：

1. 等待 3 到 5 秒后重试。
2. 刷新页面确认配置是否保存成功。
3. 查看页面“上报结果”是否有配置校验失败记录。
4. 检查后端日志中是否有 SNMP Agent 端口占用或启动失败。

## 9. 本地已验证基线

2026-08-11 本地环境已用 Net-SNMP 验证以下项目：

| 项目 | 结论 |
|---|---|
| v2c `snmpwalk` 活动告警表 | 通过 |
| v2c 错误 community | 失败符合预期 |
| v3 DES `snmpwalk/snmpget/snmpgetnext/snmpbulkwalk` | 通过 |
| v3 DES 错误用户名/错误认证密码 | 失败符合预期 |
| v3 DES Inform 测试发送 | 通过 |
| v3 DES Trap 测试发送 | 通过 |
| v3 AES128 Inform 测试发送 | 通过，Net-SNMP 命令使用 `-x AES` |
| MIB 字段数量和顺序 | 18 个字段，与 `omcAlarmMIB.mib` 对齐 |
| 测试后配置恢复 | 已恢复 v3 目标为关闭状态，v2c 默认查询保持可用 |

## 10. 测试完成后的恢复建议

测试完成后建议：

1. 停止本地 `snmptrapd`。
2. 将临时打开的 v3 目标关闭，避免无效对端反复失败。
3. 保留 v2c 或现场正式目标配置时，确认对端地址和端口是真实 OSS 接收端。
4. 再执行一次 `snmpwalk`，确认需要保留的 MIB 查询版本仍可用。
5. 在页面“上报结果”中确认没有持续失败记录。
