# 09 — 项目目录结构

> go-zero monorepo 布局、共享包组织、goctl 代码生成工作流

---

## 1. 目录结构总览

```
omcgo/
├── go.mod                              # 单一 go.mod（monorepo）
├── go.sum
├── Makefile                            # 全局构建/生成/测试目标
├── CLAUDE.md                           # 项目指导文件
│
├── common/                             # ========== 跨服务共享代码 ==========
│   ├── carrier/                        # 运营商抽象层
│   │   ├── carrier.go                  #   Carrier 接口定义
│   │   ├── registry.go                 #   CarrierRegistry
│   │   ├── cmcc/                       #   中国移动适配器
│   │   │   └── adapter.go
│   │   ├── ctcc/                       #   中国电信适配器
│   │   │   └── adapter.go
│   │   └── cucc/                       #   中国联通适配器
│   │       └── adapter.go
│   ├── model/                          # 共享领域模型与常量
│   │   ├── constants.go                #   CarrierCode, Technology, DeviceStatus, AlarmSeverity
│   │   ├── device.go                   #   Device 结构体
│   │   ├── alarm.go                    #   Alarm 结构体
│   │   └── pm.go                       #   PMCounter, KPIValue 结构体
│   ├── errorx/                         # 统一错误码
│   │   ├── codes.go                    #   业务错误码定义 (1000-6999)
│   │   └── codeerror.go                #   CodeError 类型
│   ├── middleware/                      # 共享 HTTP 中间件
│   │   ├── jwt.go                      #   JWT 认证
│   │   ├── rbac.go                     #   RBAC 鉴权
│   │   └── apikey.go                   #   API Key 认证（北向）
│   └── result/                         # 统一 API 响应格式
│       └── response.go                 #   Success / Error / PageResp
│
├── pkg/                                # ========== 可复用公共库 ==========
│   ├── tr069/                          # TR069 协议类型
│   │   ├── types.go                    #   DeviceId, InformMessage, ParameterValueStruct
│   │   ├── events.go                   #   EventCode 常量
│   │   ├── faults.go                   #   CWMP Fault 9000-9013
│   │   └── rpc_types.go               #   9 个 RPC 请求/响应类型
│   ├── soap/                           # SOAP 工具
│   │   ├── envelope.go                 #   SOAP Envelope 结构
│   │   ├── templates.go                #   预编译 text/template
│   │   └── parser.go                   #   xml.Decoder 流式解析
│   └── xmlutil/                        # XML 辅助
│       └── helpers.go                  #   XML 编解码工具函数
│
├── service/                            # ========== 微服务目录 ==========
│   │
│   ├── acs/                            # --- ACS 引擎（独立非 go-zero 服务）---
│   │   ├── main.go                     # 入口：启动 HTTP + zRPC Server
│   │   ├── etc/
│   │   │   └── acs.yaml                # 配置文件
│   │   ├── server.go                   # net/http 服务器
│   │   ├── handler.go                  # SOAP/XML 请求处理管线
│   │   ├── session.go                  # 会话状态机
│   │   ├── dispatcher.go              # 请求分发器（Inform/RPC Response/Empty）
│   │   ├── soap/                       # SOAP 编解码
│   │   │   ├── decoder.go
│   │   │   └── encoder.go
│   │   ├── rpc/                        # 9 个 TR069 RPC 方法实现
│   │   │   ├── get_parameter_values.go
│   │   │   ├── set_parameter_values.go
│   │   │   ├── get_parameter_names.go
│   │   │   ├── add_object.go
│   │   │   ├── delete_object.go
│   │   │   ├── download.go
│   │   │   ├── upload.go
│   │   │   ├── reboot.go
│   │   │   └── factory_reset.go
│   │   ├── connreq/                    # Connection Request 发送
│   │   │   └── sender.go
│   │   ├── cmdqueue/                   # Redis 命令队列
│   │   │   └── queue.go
│   │   ├── auth/                       # CPE 认证（Digest/Basic）
│   │   │   └── authenticator.go
│   │   ├── zrpc/                       # zRPC 服务端（供管理面调用）
│   │   │   ├── server.go              # AcsControl gRPC 服务实现
│   │   │   └── pb/                    # 生成的 proto 代码
│   │   └── metrics.go                  # Prometheus 自定义指标
│   │
│   ├── device/                         # --- 设备管理服务 ---
│   │   ├── api/                        # REST API (go-zero rest)
│   │   │   ├── device.api              #   .api 定义文件
│   │   │   ├── device.go              #   main 入口（goctl 生成）
│   │   │   ├── etc/
│   │   │   │   └── device-api.yaml
│   │   │   └── internal/
│   │   │       ├── config/
│   │   │       │   └── config.go      #   配置结构（goctl 生成 + 手动扩展）
│   │   │       ├── handler/           #   路由处理器（goctl 生成，勿手动修改）
│   │   │       │   ├── device/
│   │   │       │   ├── topology/
│   │   │       │   ├── datamodel/
│   │   │       │   └── provisioning/
│   │   │       ├── logic/             #   业务逻辑（手写）
│   │   │       │   ├── device/
│   │   │       │   │   ├── listdeviceslogic.go
│   │   │       │   │   ├── getdevicelogic.go
│   │   │       │   │   ├── rebootdevicelogic.go
│   │   │       │   │   └── ...
│   │   │       │   ├── topology/
│   │   │       │   ├── datamodel/
│   │   │       │   └── provisioning/
│   │   │       ├── svc/               #   ServiceContext（手写）
│   │   │       │   └── servicecontext.go
│   │   │       └── types/             #   请求/响应类型（goctl 生成）
│   │   │           └── types.go
│   │   ├── rpc/                        # gRPC RPC (go-zero zRPC)
│   │   │   ├── device.go             #   main 入口（goctl 生成）
│   │   │   ├── etc/
│   │   │   │   └── device-rpc.yaml
│   │   │   ├── deviceclient/          #   RPC 客户端（goctl 生成）
│   │   │   │   └── device.go
│   │   │   ├── pb/                    #   Protobuf 生成代码
│   │   │   │   ├── device.pb.go
│   │   │   │   └── device_grpc.pb.go
│   │   │   └── internal/
│   │   │       ├── config/
│   │   │       ├── logic/             #   业务逻辑（手写）
│   │   │       │   ├── registerdevicelogic.go
│   │   │       │   ├── listdeviceslogic.go
│   │   │       │   ├── transitionstatuslogic.go
│   │   │       │   ├── startprovisioninglogic.go
│   │   │       │   └── ...
│   │   │       ├── server/            #   gRPC 服务端（goctl 生成）
│   │   │       │   └── deviceserver.go
│   │   │       └── svc/               #   ServiceContext（手写）
│   │   │           └── servicecontext.go
│   │   └── model/                      # 数据访问层
│   │       ├── devicemodel.go          #   goctl model 生成接口
│   │       ├── devicemodel_gen.go      #   goctl model 生成实现（可覆盖）
│   │       ├── devicecustom.go         #   手写扩展查询（分区表、复杂过滤）
│   │       ├── provisioningtaskmodel.go
│   │       ├── devicegroupmodel.go
│   │       └── vars.go
│   │
│   ├── config/                         # --- 数据模型与配置服务 ---
│   │   ├── rpc/                        # (结构同 device/rpc)
│   │   │   ├── config.go
│   │   │   ├── etc/config-rpc.yaml
│   │   │   ├── configclient/
│   │   │   ├── pb/
│   │   │   └── internal/
│   │   │       ├── logic/
│   │   │       │   ├── resolvedatamodellogic.go  # 三级回退解析（核心）
│   │   │       │   ├── activatedatamodellogic.go
│   │   │       │   └── ...
│   │   │       ├── server/
│   │   │       └── svc/
│   │   └── model/
│   │       ├── datamodeldefinitionmodel.go        # 手写（JSONB + 三级查询）
│   │       ├── configtemplatemodel.go             # goctl 生成
│   │       ├── configtemplatemodel_gen.go
│   │       └── ouiregistrymodel.go                # goctl 生成
│   │
│   ├── monitor/                        # --- 监控服务（PM + 告警 + MR + 北向） ---
│   │   ├── api/                        # monitor-api REST 网关
│   │   │   ├── monitor.api
│   │   │   ├── monitor.go
│   │   │   ├── etc/monitor-api.yaml
│   │   │   └── internal/
│   │   │       ├── handler/
│   │   │       │   ├── pm/
│   │   │       │   ├── alarm/
│   │   │       │   ├── mr/
│   │   │       │   └── northbound/
│   │   │       ├── logic/
│   │   │       └── svc/
│   │   ├── rpc/
│   │   │   ├── pm/                     # pm-rpc
│   │   │   │   ├── pm.go
│   │   │   │   ├── etc/pm-rpc.yaml
│   │   │   │   ├── pmclient/
│   │   │   │   ├── pb/
│   │   │   │   └── internal/
│   │   │   └── alarm/                  # alarm-rpc
│   │   │       ├── alarm.go
│   │   │       ├── etc/alarm-rpc.yaml
│   │   │       ├── alarmclient/
│   │   │       ├── pb/
│   │   │       └── internal/
│   │   └── model/
│   │       ├── pmcountermodel.go       # 手写（TimescaleDB hypertable）
│   │       ├── kpivaluemodel.go        # 手写
│   │       ├── alarmactivemodel.go     # 手写
│   │       ├── alarmhistorymodel.go    # 手写
│   │       └── measurementreportmodel.go # 手写
│   │
│   ├── admin/                          # --- 管理服务 ---
│   │   ├── api/                        # admin-api REST 网关
│   │   │   ├── admin.api
│   │   │   ├── admin.go
│   │   │   ├── etc/admin-api.yaml
│   │   │   └── internal/
│   │   ├── rpc/                        # admin-rpc
│   │   │   ├── admin.go
│   │   │   ├── etc/admin-rpc.yaml
│   │   │   ├── adminclient/
│   │   │   ├── pb/
│   │   │   └── internal/
│   │   └── model/
│   │       ├── usermodel.go            # goctl 生成
│   │       ├── usermodel_gen.go
│   │       ├── rolemodel.go            # goctl 生成
│   │       ├── permissionmodel.go      # goctl 生成
│   │       ├── auditlogmodel.go        # 手写（仅追加 + 时间查询）
│   │       └── firmwareversionmodel.go # goctl 生成
│   │
│   └── worker/                         # --- 后台工作进程 ---
│       ├── main.go                     # 入口：ServiceGroup 启动多个 worker
│       ├── etc/
│       │   └── worker.yaml
│       ├── internal/
│       │   ├── config/
│       │   └── svc/
│       ├── pm/                         # PM 文件解析 worker
│       │   ├── consumer.go             #   NATS 消费者
│       │   └── parser.go               #   PM XML 流式解析
│       ├── mr/                         # MR 文件解析 worker
│       │   ├── consumer.go
│       │   └── parser.go
│       ├── kpi/                        # KPI 计算
│       │   ├── engine.go               #   计算引擎
│       │   ├── formulas.go             #   公式定义
│       │   └── scheduler.go            #   定时触发
│       ├── alarm/                      # 告警关联分析
│       │   └── correlator.go
│       └── heartbeat/                  # 设备离线检测
│           └── checker.go
│
├── api/                                # ========== Proto 定义集中管理 ==========
│   └── proto/
│       ├── device.proto                # device-rpc 接口
│       ├── config.proto                # config-rpc 接口
│       ├── pm.proto                    # pm-rpc 接口
│       ├── alarm.proto                 # alarm-rpc 接口
│       ├── admin.proto                 # admin-rpc 接口
│       └── acs_control.proto           # acs-rpc 控制接口
│
├── migrations/                         # ========== 数据库迁移 ==========
│   ├── 000001_create_devices.up.sql
│   ├── 000001_create_devices.down.sql
│   ├── 000002_create_data_models.up.sql
│   ├── 000002_create_data_models.down.sql
│   ├── ...
│   └── 000012_create_users_rbac.up.sql
│
├── datamodels/                         # ========== TR069 数据模型种子数据 ==========
│   ├── seed/
│   │   ├── carrier_defaults/           # 运营商默认数据模型 JSON
│   │   │   ├── cmcc_lte.json
│   │   │   ├── cmcc_nr.json
│   │   │   ├── ctcc_lte.json
│   │   │   └── cucc_nr.json
│   │   └── product_models/             # 产品级数据模型 JSON
│   └── templates/                      # JSON Schema 模板
│
├── deployments/                        # ========== 部署清单 ==========
│   ├── docker/
│   │   ├── Dockerfile.acs              # ACS 引擎
│   │   ├── Dockerfile.device-api
│   │   ├── Dockerfile.device-rpc
│   │   ├── Dockerfile.config-rpc
│   │   ├── Dockerfile.monitor-api
│   │   ├── Dockerfile.pm-rpc
│   │   ├── Dockerfile.alarm-rpc
│   │   ├── Dockerfile.admin-api
│   │   ├── Dockerfile.admin-rpc
│   │   ├── Dockerfile.worker
│   │   ├── docker-compose.yml          # 本地开发环境
│   │   └── prometheus.yml              # Prometheus 配置
│   └── k8s/
│       ├── namespace.yaml
│       ├── acs-engine/
│       │   ├── deployment.yaml
│       │   ├── service.yaml
│       │   └── hpa.yaml
│       ├── device-api/
│       ├── device-rpc/
│       ├── config-rpc/
│       ├── monitor-api/
│       ├── pm-rpc/
│       ├── alarm-rpc/
│       ├── admin-api/
│       ├── admin-rpc/
│       ├── worker/
│       ├── infrastructure/
│       │   ├── postgresql.yaml
│       │   ├── redis.yaml
│       │   ├── nats.yaml
│       │   ├── etcd.yaml
│       │   └── minio.yaml
│       └── ingress.yaml
│
├── configs/                            # ========== 配置文件模板 ==========
│   ├── acs.yaml.template
│   ├── device-api.yaml.template
│   ├── device-rpc.yaml.template
│   ├── config-rpc.yaml.template
│   ├── monitor-api.yaml.template
│   ├── pm-rpc.yaml.template
│   ├── alarm-rpc.yaml.template
│   ├── admin-api.yaml.template
│   ├── admin-rpc.yaml.template
│   └── worker.yaml.template
│
├── scripts/                            # ========== 构建/部署/测试脚本 ==========
│   ├── goctl-gen.sh                    # goctl 批量生成脚本
│   ├── build-all.sh                    # 编译所有二进制
│   └── migrate.sh                      # 数据库迁移脚本
│
├── test/                               # ========== 测试 ==========
│   ├── integration/
│   │   ├── acs_test.go
│   │   ├── device_test.go
│   │   └── provisioning_test.go
│   ├── e2e/
│   │   └── full_flow_test.go
│   └── fixtures/
│       ├── soap/                       # SOAP 报文样本
│       ├── pm/                         # PM XML 文件样本
│       └── mr/                         # MR 文件样本
│
└── doc/                                # ========== 文档 ==========
    ├── README.md
    ├── go-zero-design/                 # 本架构设计目录（当前）
    ├── architecture/
    ├── detailed-design/
    ├── features/
    └── specs-inventory/
```

---

## 2. 分层说明

```
┌──────────────────────────────────────────────────────────────┐
│                         common/                               │
│  跨服务共享：接口定义、常量、错误码、中间件、响应格式           │
│  规则：不依赖任何 service/ 下的代码                           │
├──────────────────────────────────────────────────────────────┤
│                          pkg/                                 │
│  独立公共库：TR069 类型、SOAP 工具、XML 辅助                  │
│  规则：可独立复用，不依赖 common/ 和 service/                 │
├──────────────────────────────────────────────────────────────┤
│                        service/                               │
│  微服务实现：每个服务包含 api/ + rpc/ + model/                │
│  规则：可依赖 common/ 和 pkg/，服务间仅通过 zRPC 通信         │
├──────────────────────────────────────────────────────────────┤
│                       api/proto/                              │
│  接口契约：集中管理的 .proto 文件                             │
│  规则：所有 RPC 接口定义的唯一来源                            │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. goctl 代码生成工作流

### 3.1 批量生成脚本

```bash
#!/bin/bash
# scripts/goctl-gen.sh

set -e

PROTO_DIR="api/proto"
STYLE="goZero"

echo "=== Generating API services ==="

# device-api
goctl api go \
    -api service/device/api/device.api \
    -dir service/device/api/ \
    --style $STYLE

# monitor-api
goctl api go \
    -api service/monitor/api/monitor.api \
    -dir service/monitor/api/ \
    --style $STYLE

# admin-api
goctl api go \
    -api service/admin/api/admin.api \
    -dir service/admin/api/ \
    --style $STYLE

echo "=== Generating RPC services ==="

for proto in device config pm alarm admin acs_control; do
    SERVICE=$(echo $proto | sed 's/_control//')
    echo "  Generating $proto..."

    # 确定输出目录
    case $proto in
        device)      OUT_DIR="service/device/rpc" ;;
        config)      OUT_DIR="service/config/rpc" ;;
        pm)          OUT_DIR="service/monitor/rpc/pm" ;;
        alarm)       OUT_DIR="service/monitor/rpc/alarm" ;;
        admin)       OUT_DIR="service/admin/rpc" ;;
        acs_control) OUT_DIR="service/acs/zrpc" ;;
    esac

    goctl rpc protoc ${PROTO_DIR}/${proto}.proto \
        --go_out=${OUT_DIR}/pb \
        --go-grpc_out=${OUT_DIR}/pb \
        --zrpc_out=${OUT_DIR}/ \
        --style $STYLE
done

echo "=== Generating models (cacheable tables) ==="

DB_URL="postgres://omcgo:omcgo_dev@localhost:5432/omcgo?sslmode=disable"

for table in users roles permissions device_groups config_templates oui_registry \
             provisioning_tasks firmware_versions; do

    # 确定输出目录
    case $table in
        users|roles|permissions)      MODEL_DIR="service/admin/model" ;;
        device_groups)                MODEL_DIR="service/device/model" ;;
        config_templates|oui_registry) MODEL_DIR="service/config/model" ;;
        provisioning_tasks)           MODEL_DIR="service/device/model" ;;
        firmware_versions)            MODEL_DIR="service/admin/model" ;;
    esac

    echo "  Generating model for $table..."
    goctl model pg datasource \
        -url="$DB_URL" \
        -table="$table" \
        -dir="$MODEL_DIR" \
        -cache \
        --style $STYLE
done

echo "=== Done! ==="
```

### 3.2 Makefile 目标

```makefile
# Makefile

.PHONY: generate build test lint docker

# 代码生成
generate:
	bash scripts/goctl-gen.sh

generate-api:
	goctl api go -api service/device/api/device.api -dir service/device/api/ --style goZero
	goctl api go -api service/monitor/api/monitor.api -dir service/monitor/api/ --style goZero
	goctl api go -api service/admin/api/admin.api -dir service/admin/api/ --style goZero

generate-rpc:
	bash scripts/goctl-gen.sh  # RPC 部分

# 编译
build: build-acs build-device-api build-device-rpc build-config-rpc \
       build-monitor-api build-pm-rpc build-alarm-rpc \
       build-admin-api build-admin-rpc build-worker

build-acs:
	go build -o bin/acs-engine service/acs/main.go

build-device-api:
	go build -o bin/device-api service/device/api/device.go

build-device-rpc:
	go build -o bin/device-rpc service/device/rpc/device.go

build-config-rpc:
	go build -o bin/config-rpc service/config/rpc/config.go

build-monitor-api:
	go build -o bin/monitor-api service/monitor/api/monitor.go

build-pm-rpc:
	go build -o bin/pm-rpc service/monitor/rpc/pm/pm.go

build-alarm-rpc:
	go build -o bin/alarm-rpc service/monitor/rpc/alarm/alarm.go

build-admin-api:
	go build -o bin/admin-api service/admin/api/admin.go

build-admin-rpc:
	go build -o bin/admin-rpc service/admin/rpc/admin.go

build-worker:
	go build -o bin/worker service/worker/main.go

# 测试
test:
	go test ./... -v -count=1

test-integration:
	go test ./test/integration/... -v -tags=integration

# 代码检查
lint:
	golangci-lint run ./...

# 数据库迁移
migrate-up:
	migrate -database "$(DATABASE_URL)" -path migrations/ up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path migrations/ down 1

# Docker
docker-build:
	docker-compose -f deployments/docker/docker-compose.yml build

docker-up:
	docker-compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	docker-compose -f deployments/docker/docker-compose.yml down
```

---

## 4. 文件命名与代码风格

### 4.1 go-zero 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| goctl --style | `goZero` | 驼峰命名：`listDevicesLogic.go` |
| Handler 文件 | 按 group 分目录 | `handler/device/listdeviceshandler.go` |
| Logic 文件 | 按 group 分目录 | `logic/device/listdeviceslogic.go` |
| Model 文件 | 表名 + model | `devicemodel.go`, `devicemodel_gen.go` |
| 手写扩展 | *custom.go | `devicecustom.go` |
| Proto 文件 | 小写下划线 | `acs_control.proto` |
| .api 文件 | 服务名 | `device.api`, `monitor.api` |

### 4.2 goctl 生成 vs 手写的边界

```
goctl 生成（不要手动修改）:
  handler/*.go          → 路由处理器骨架
  types/types.go        → 请求/响应类型
  server/*server.go     → gRPC 服务端注册
  *client/*.go          → gRPC 客户端封装
  *model_gen.go         → Model CRUD 实现

手写（业务逻辑核心）:
  logic/*.go            → 所有业务逻辑
  svc/servicecontext.go → 依赖注入
  config/config.go      → 自定义配置项
  *custom.go            → 数据库扩展查询
  middleware/*.go        → 自定义中间件
```
