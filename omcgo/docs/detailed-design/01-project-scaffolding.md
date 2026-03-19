# DD-01: 项目脚手架与工程化

> 关联功能域：全部
> 关联 backend-design.md 章节：第十章（项目目录结构）
> 实施阶段：Phase 1（基础建设）
> 依赖文档：无（第一个实施的文档）

---

## 1. 概述

### 1.1 模块定位

项目脚手架是整个 OMC Go 系统的工程基础，定义了代码组织方式、构建流程、配置管理和开发规范。所有后续模块的开发均在此骨架上进行。

### 1.2 核心职责

- Go module 初始化与依赖管理
- 标准化目录结构创建
- 三个部署单元的 cmd 入口设计
- 配置文件加载（viper）与 CLI 框架（cobra）
- Makefile 构建系统
- 代码质量工具链（golangci-lint、pre-commit）

### 1.3 与其他模块的交互关系

本模块为所有其他模块提供项目骨架，是唯一无前置依赖的模块。

---

## 2. 详细设计

### 2.1 Go Module 初始化

```
module github.com/omcgo/omcgo

go 1.22
```

**核心依赖版本锁定**（go.mod）：

| 依赖 | 版本 | 用途 |
|------|------|------|
| `github.com/gin-gonic/gin` | v1.9+ | REST API 框架 |
| `google.golang.org/grpc` | v1.62+ | gRPC |
| `github.com/spf13/viper` | v1.18+ | 配置管理 |
| `github.com/spf13/cobra` | v1.8+ | CLI 框架 |
| `go.uber.org/zap` | v1.27+ | 结构化日志 |
| `github.com/jackc/pgx/v5` | v5.5+ | PostgreSQL 驱动 |
| `github.com/redis/go-redis/v9` | v9.5+ | Redis 客户端 |
| `github.com/nats-io/nats.go` | v1.33+ | NATS 客户端 |
| `github.com/minio/minio-go/v7` | v7.0+ | MinIO 客户端 |
| `github.com/golang-migrate/migrate/v4` | v4.17+ | 数据库迁移 |
| `github.com/Masterminds/squirrel` | v1.5+ | SQL 构建器 |
| `github.com/prometheus/client_golang` | v1.19+ | Prometheus |
| `go.opentelemetry.io/otel` | v1.24+ | OpenTelemetry |
| `github.com/beevik/etree` | v1.3+ | XML 动态遍历 |
| `github.com/go-playground/validator/v10` | v10.18+ | 参数校验 |
| `github.com/google/uuid` | v1.6+ | UUID |
| `github.com/robfig/cron/v3` | v3.0+ | 定时任务 |
| `golang.org/x/time` | latest | 限流 |
| `github.com/stretchr/testify` | v1.9+ | 测试 |
| `github.com/swaggo/swag` | v1.16+ | Swagger |

### 2.2 目录结构创建

```
omcgo/
├── cmd/
│   ├── acs/main.go           # ACS 引擎入口
│   ├── app/main.go           # 主应用入口
│   ├── worker/main.go        # Worker 入口
│   ├── migrate/main.go       # 数据库迁移入口
│   └── omcctl/main.go        # CLI 管理工具入口
├── internal/
│   ├── acs/                   # F01
│   ├── config/                # F02
│   ├── pm/                    # F03
│   ├── alarm/                 # F04
│   ├── mr/                    # F05
│   ├── omcr/                  # F06
│   ├── nedirect/              # F07
│   ├── northbound/            # F08
│   ├── provision/             # F09
│   ├── interop/               # F10
│   ├── carrier/               # 运营商抽象层
│   ├── common/                # 公共类型
│   │   ├── model/
│   │   ├── event/
│   │   ├── errors/
│   │   └── middleware/
│   └── infra/                 # 基础设施
│       ├── db/
│       ├── cache/
│       ├── mq/
│       └── storage/
├── pkg/
│   ├── tr069/
│   ├── soap/
│   └── xmlutil/
├── api/
│   ├── openapi/
│   └── proto/
├── migrations/
├── configs/
│   ├── acs.yaml
│   ├── app.yaml
│   └── worker.yaml
├── datamodels/
│   ├── seed/
│   │   ├── carrier_defaults/
│   │   └── product_models/
│   └── templates/
├── deployments/
│   ├── docker/
│   └── k8s/
├── scripts/
├── test/
│   ├── integration/
│   ├── e2e/
│   └── fixtures/
├── go.mod
├── go.sum
├── Makefile
├── CLAUDE.md
└── README.md
```

### 2.3 cmd 入口设计

每个入口使用 cobra 构建命令树，viper 加载配置。

**cmd/acs/main.go** — ACS 引擎：

```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "omcgo-acs",
        Short: "OMC TR069 ACS Engine",
        RunE:  runACS,
    }
    rootCmd.Flags().String("config", "configs/acs.yaml", "配置文件路径")
    rootCmd.Execute()
}

func runACS(cmd *cobra.Command, args []string) error {
    // 1. 加载配置 (viper)
    // 2. 初始化日志 (zap)
    // 3. 连接基础设施 (Redis, NATS, DB)
    // 4. 创建 ACSServer
    // 5. 注册优雅关闭
    // 6. 启动 HTTP 服务器 (端口 7547/7548)
    // 7. 启动 metrics 服务器 (端口 9090)
}
```

**cmd/app/main.go** — 主应用：

```go
func runApp(cmd *cobra.Command, args []string) error {
    // 1. 加载配置
    // 2. 初始化日志
    // 3. 连接基础设施 (PostgreSQL, TimescaleDB, Redis, NATS, MinIO)
    // 4. 创建各模块 Service
    // 5. 创建 Gin router，注册路由
    // 6. 注册优雅关闭
    // 7. 启动 HTTP (8080), HTTPS (8443), metrics (9091), gRPC (50051)
}
```

**cmd/worker/main.go** — 后台工作进程：

```go
func runWorker(cmd *cobra.Command, args []string) error {
    // 1. 加载配置
    // 2. 连接基础设施
    // 3. 注册 NATS 消费者 (pm.file.received, mr.file.received 等)
    // 4. 启动 WorkerPool
    // 5. 启动 metrics (9092)
}
```

**cmd/migrate/main.go** — 数据库迁移：

```go
// 子命令: up, down, version, force
```

**cmd/omcctl/main.go** — CLI 工具：

```go
// 子命令: device list, device get, alarm list, pm query, datamodel import 等
```

### 2.4 配置文件设计

**configs/acs.yaml**：

```yaml
server:
  host: "0.0.0.0"
  port: 7547
  tls_port: 7548
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

session:
  timeout: 5m
  max_concurrent: 10000

rate_limit:
  per_device: 10       # 每设备每分钟最大 Inform 数
  global_burst: 5000   # 全局突发上限

redis:
  addrs: ["localhost:6379"]
  password: ""
  db: 0
  pool_size: 100

nats:
  url: "nats://localhost:4222"
  cluster_id: "omcgo"

db:
  dsn: "postgres://omcgo:password@localhost:5432/omcgo?sslmode=disable"
  max_conns: 20

metrics:
  port: 9090

log:
  level: "info"
  format: "json"
```

**configs/app.yaml**：

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  tls_port: 8443
  grpc_port: 50051

db:
  dsn: "postgres://omcgo:password@localhost:5432/omcgo?sslmode=disable"
  max_conns: 50
  min_conns: 10

tsdb:
  dsn: "postgres://omcgo:password@localhost:5432/omcgo_ts?sslmode=disable"
  max_conns: 30

redis:
  addrs: ["localhost:6379"]
  password: ""
  pool_size: 200

nats:
  url: "nats://localhost:4222"

minio:
  endpoint: "localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  buckets:
    pm_files: "pm-files"
    mr_files: "mr-files"
    firmware: "firmware"
    config_backup: "config-backup"
    logs: "logs"

metrics:
  port: 9091

log:
  level: "info"
  format: "json"
```

### 2.5 Makefile 设计

```makefile
.PHONY: build build-acs build-app build-worker test lint generate

# 构建
build: build-acs build-app build-worker

build-acs:
	go build -o bin/omcgo-acs ./cmd/acs

build-app:
	go build -o bin/omcgo-app ./cmd/app

build-worker:
	go build -o bin/omcgo-worker ./cmd/worker

build-migrate:
	go build -o bin/omcgo-migrate ./cmd/migrate

build-omcctl:
	go build -o bin/omcctl ./cmd/omcctl

# 测试
test:
	go test ./... -v -race -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

test-integration:
	go test ./test/integration/... -v -tags=integration

# 代码质量
lint:
	golangci-lint run ./...

vet:
	go vet ./...

# 代码生成
generate:
	protoc --go_out=. --go-grpc_out=. api/proto/*.proto
	swag init -g cmd/app/main.go -o api/openapi

# 数据库
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# Docker
docker-build:
	docker build -f deployments/docker/Dockerfile.acs -t omcgo-acs:latest .
	docker build -f deployments/docker/Dockerfile.app -t omcgo-app:latest .
	docker build -f deployments/docker/Dockerfile.worker -t omcgo-worker:latest .

docker-up:
	docker-compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	docker-compose -f deployments/docker/docker-compose.yml down

# 清理
clean:
	rm -rf bin/ coverage.out
```

### 2.6 golangci-lint 配置

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - misspell
    - gofmt
    - goimports
    - gocritic

linters-settings:
  govet:
    check-shadowing: true
  errcheck:
    check-type-assertions: true

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

---

## 3. 实施子阶段

### 阶段 1a：Go Module + 目录骨架 + Makefile

**交付物**：
- `go.mod` 包含所有核心依赖
- 完整目录结构（空目录放 `.gitkeep`）
- `Makefile` 包含 build/test/lint target
- `.golangci.yml` 配置

**验证**：`make build` 编译通过（入口文件为空 main）

### 阶段 1b：cmd 入口 + 配置加载

**交付物**：
- 5 个 cmd 入口（acs/app/worker/migrate/omcctl）
- 3 个配置文件模板（acs.yaml/app.yaml/worker.yaml）
- 配置结构体定义（使用 viper 加载）
- cobra 命令注册

**验证**：`./bin/omcgo-acs --config configs/acs.yaml` 启动并打印配置

### 阶段 1c：CI/CD + Linter

**交付物**：
- GitHub Actions workflow（build + test + lint）
- pre-commit hook 配置
- `.gitignore` 完善

**验证**：`make lint` 通过，CI pipeline 绿色

---

## 4. 文件清单

```
cmd/acs/main.go
cmd/app/main.go
cmd/worker/main.go
cmd/migrate/main.go
cmd/omcctl/main.go
configs/acs.yaml
configs/app.yaml
configs/worker.yaml
internal/config.go              # 配置结构体定义
Makefile
.golangci.yml
.gitignore
go.mod
go.sum
```

---

## 5. 测试策略

- 配置加载测试：验证 YAML 解析、环境变量覆盖、默认值
- cmd 入口测试：验证 cobra 命令注册、flag 解析
- 集成测试骨架：`test/` 目录下放置示例测试

---

## 6. 参考

- backend-design.md 第十章：项目目录结构
- CLAUDE.md 第 4 节：项目目录结构
- CLAUDE.md 第 9 节：常用命令
