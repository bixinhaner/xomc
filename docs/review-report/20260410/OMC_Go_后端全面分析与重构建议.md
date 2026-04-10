# OMC Go 后端全面分析与重构建议

> **文档版本**: v2.0  
> **创建时间**: 2025-04-10  
> **最后更新**: 2025-04-10  
> **状态**: 待决策  
> **作者**: AI Assistant

---

## 一、当前架构优势分析

### 1.1 整体架构设计优秀

**模块化单体 + 独立 ACS 引擎**的架构选择非常合理:
- 10 万基站规模下单进程足够(333 sessions/s)
- 10 个功能域跨域交互密集,避免分布式事务复杂度
- ACS 独立部署满足 TR069 有状态会话的独立扩展需求

**三个部署单元清晰**:
- `omcgo-acs` — TR069 ACS 引擎(独立进程,水平可扩展)
- `omcgo-app` — 主应用(F02-F10 模块化单体)
- `omcgo-worker` — 后台工作进程(PM/MR 文件处理、KPI 计算)

### 1.2 技术栈选型合理

| 组件 | 选型 | 评价 |
|------|------|------|
| HTTP 框架 | Gin (管理面) + net/http (ACS) | 合适,ACS 需要完全控制请求生命周期 |
| 数据库 | PostgreSQL 16 + TimescaleDB | 优秀,JSONB 支持灵活 Schema |
| ORM | squirrel + pgx/v5 (不使用 ORM) | ⚠️ Squirrel 已停滞,建议规划迁移 |
| 消息队列 | NATS JetStream | 轻量高效,适合 Go 生态 |
| 数据库迁移 | golang-migrate | ⚠️ 建议升级为 Goose |

### 1.3 目录结构规范

```
omcgo/
├── cmd/                    # 入口(app/acs/worker/migrate/omcctl)
├── internal/               # 私有代码(按功能域组织,扁平化)
│   ├── core/              # 核心基础设施(components/event/middleware/model)
│   ├── device/            # F06 设备管理
│   ├── admin/             # F06 用户管理与 RBAC
│   ├── config/            # F02 数据模型与配置
│   └── ...                # 其他功能域
├── pkg/                    # 可复用公共库(tr069/soap/xmlutil)
├── migrations/             # 数据库迁移文件(147 个)
└── datamodels/             # TR069 数据模型种子数据
```

**符合 Go 标准项目布局**,遵循 `internal` 私有代码约定。

### 1.4 依赖注入与容器设计

Container 设计清晰:
- 基础设施层由 bootstrap 初始化(DB、Redis、MinIO 等)
- 共享服务层由各模块 Init 函数设置
- 按依赖顺序执行,先初始化的模块共享服务写入 Container

### 1.5 Repository 模式实现规范

每个模块遵循统一模式:
```
omcgo/internal/{domain}/{module}/
├── model.go           # 领域模型 + 枚举常量 + Filter
├── repository.go      # Repository 接口定义
├── pg_repository.go   # PostgreSQL 实现(squirrel + pgx/v5)
├── service.go         # 业务逻辑(可选,简单 CRUD 可省略)
└── handler.go         # Gin HTTP 处理器 + RegisterRoutes
```

---

## 二、存在的问题与重构建议

### 2.1 SQL Builder 演进建议 ⭐ 新增

> **详细说明**: 完整的 SQL Builder 对比和工具分工方案请参考:  
> [SQL Builder 演进建议与工具分工方案](./sql-builder-evolution-plan.md)

#### 2.1.1 当前 Squirrel 现状

| 维度 | 数据 |
|------|------|
| **版本** | v1.5.4 (2021年发布) |
| **维护状态** | ⚠️ **基本停滞** (3+ 年未发布新版本) |
| **使用范围** | 40 个 repository 文件 |
| **Issues** | 60+ 未关闭 |
| **PR 积压** | 20+ 未合并 |

#### 2.1.2 推荐方案: 混合架构

```
┌─────────────────────────────────────────┐
│           查询类型分类                    │
├─────────────────────────────────────────┤
│                                         │
│  固定查询 (80%)     动态查询 (20%)       │
│  ┌─────────────┐    ┌──────────────┐    │
│  │  sqlc ✅    │    │   goqu ✅    │    │
│  │             │    │              │    │
│  │ • CRUD      │    │ • 设备搜索   │    │
│  │ • PM 统计   │    │ • 动态过滤   │    │
│  │ • KPI 计算  │    │ • 复杂条件   │    │
│  │ • 告警查询  │    │ • 多条件组合 │    │
│  │ • Timescale │    │              │    │
│  │   函数调用  │    │              │    │
│  └─────────────┘    └──────────────┘    │
│                                         │
│  DDL/管理语句 (手写)                     │
│  ┌──────────────────────────────┐       │
│  │  Goose 迁移 + pgx 执行        │       │
│  │  • CREATE_HYPERTABLE         │       │
│  │  • 数据保留策略               │       │
│  │  • 压缩策略                   │       │
│  └──────────────────────────────┘       │
└─────────────────────────────────────────┘
```

#### 2.1.3 工具分工

| 工具 | 职责 | 示例 |
|------|------|------|
| **Goose** | Schema 版本管理 (DDL) | CREATE TABLE, CREATE_HYPERTABLE |
| **sqlc** | 查询代码生成 (DML) | SELECT, INSERT, UPDATE, DELETE |
| **pgx 原生** | 运行时管理操作 | add_retention_policy() |
| **goqu** | 动态查询构建 | 设备搜索、复杂过滤 |

#### 2.1.4 实施路线图

| 阶段 | 任务 | 工作量 | 优先级 |
|------|------|--------|--------|
| **阶段 1** (1-2周) | 封装 Squirrel 抽象层 | 1天 | P0 |
| | 引入 Goose 替代 golang-migrate | 2天 | P0 |
| | 配置 sqlc 工具链 | 1天 | P0 |
| **阶段 2** (持续) | 新模块使用 sqlc | - | P0 |
| **阶段 3** (1-2月) | PM/KPI 查询迁移到 sqlc | 3天 | P0 |
| | 动态查询引入 goqu | 2天 | P1 |

**完整示例代码**: 参考 [examples/](./examples/) 目录

---

### 2.2 数据库迁移管理复杂度高

**当前问题**:
- 147 个迁移文件,编号不连续(000036-000045 被跳过)
- 种子数据直接写在迁移文件中
- 缺乏迁移文件分类管理(schema/seed/index/alter)

**重构建议**:

#### 方案 A: 迁移文件分类目录结构(推荐)

```
migrations/
├── schema/             # 表结构创建与修改
│   ├── 001_create_devices.up.sql
│   ├── 002_create_device_parameters.up.sql
│   └── ...
├── seed/               # 种子数据
│   ├── 001_initial_users.up.sql
│   ├── 002_kpi_definitions.up.sql
│   └── ...
├── indexes/            # 索引优化
│   └── ...
└── data/               # 数据迁移(生产数据修复)
    └── ...
```

**优势**:
- 职责清晰,便于查找和维护
- 可以按需执行特定类型的迁移
- 便于代码审查

---

### 2.3 种子数据管理不完善

**当前问题**:
- 种子数据混杂在迁移文件中,难以维护
- 缺乏种子数据的版本管理和回滚机制
- 不同环境(dev/test/prod)的种子数据需求不同

**重构建议**: 创建独立的种子数据管理工具

```
cmd/seed/
├── main.go            # CLI 入口
├── seed_users.go      # 用户种子数据
├── seed_kpi.go        # KPI 定义种子数据
├── seed_alarms.go     # 告警规则种子数据
└── ...
```

**种子数据文件组织**:
```
datamodels/
├── seed/
│   ├── users/
│   │   ├── admin.json
│   │   └── operator.json
│   ├── kpi/
│   │   ├── lte.json
│   │   └── nr.json
│   └── alarms/
│       └── rules.json
└── templates/           # 配置模板
```

---

### 2.4 初始化流程可以优化

**当前流程**:
```
main.go
  └─> runApp()
       └─> initApp()           # 基础设施初始化
       └─> provider.Setup()    # 模块初始化 + 路由注册
```

**问题**:
- 初始化流程较长,错误处理分散
- 缺乏初始化健康检查
- 模块初始化顺序硬编码

**重构建议**:

#### 方案 A: 引入依赖图管理(推荐)

```go
type ModuleInitializer struct {
    Name     string
    Depends  []string  // 依赖的模块名称
    Init     func(*Container) error
}

var moduleGraph = []ModuleInitializer{
    {Name: "config", Depends: nil, Init: initConfigModule},
    {Name: "topology", Depends: nil, Init: initTopologyModule},
    {Name: "admin", Depends: []string{"topology"}, Init: initAdminModule},
    // ...
}
```

**优势**:
- 依赖关系显式声明
- 自动检测循环依赖
- 支持并行初始化无依赖模块

#### 方案 B: 添加初始化健康检查

```go
func healthCheckPostgres(ctx context.Context, pool *pgxpool.Pool) error {
    var version string
    err := pool.QueryRow(ctx, "SELECT version()").Scan(&version)
    if err != nil {
        return err
    }
    zap.L().Info("postgres health check passed", 
        zap.String("version", version))
    return nil
}
```

---

### 2.5 配置管理可以更灵活

**当前问题**:
- 配置文件使用 YAML,但缺乏配置验证
- 环境变量覆盖机制不够完善
- 缺乏配置热重载的动态生效机制

**重构建议**:

#### 引入配置验证

```go
func (c *AppConfig) Validate() error {
    var errs []error
    
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        errs = append(errs, fmt.Errorf("invalid server port: %d", c.Server.Port))
    }
    
    if c.DB.MaxConns < 1 {
        errs = append(errs, fmt.Errorf("DB.MaxConns must be >= 1"))
    }
    
    if c.JWT.Secret == "" {
        errs = append(errs, fmt.Errorf("JWT secret must not be empty"))
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("config validation failed: %v", errs)
    }
    return nil
}
```

---

### 2.6 测试策略需要完善

**当前问题**:
- 单元测试覆盖率不均衡(部分模块 40%,部分 80%+)
- 集成测试依赖外部服务,本地运行复杂
- 缺乏 E2E 测试

**重构建议**:

#### 完善测试金字塔

```
test/
├── unit/              # 单元测试(已有,继续完善)
├── integration/       # 集成测试
│   ├── fixtures/      # 测试数据
│   ├── setup.go       # 测试环境初始化
│   └── ...
├── e2e/               # 端到端测试(新增)
│   └── scenarios/     # 测试场景
└── mocks/             # Mock 实现
    └── mock_repository.go
```

---

## 三、项目初始化最佳实践建议

### 3.1 数据库表结构初始化

**推荐方案**:

1. **使用迁移工具管理 Schema**
   ```bash
   # 创建新表
   make migrate-create name=create_new_table
   
   # 执行迁移
   make migrate-up
   ```

2. **迁移文件编写规范**
   ```sql
   -- 000NNN_create_new_table.up.sql
   BEGIN;
   
   CREATE TABLE IF NOT EXISTS new_table (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       name VARCHAR(255) NOT NULL,
       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
   );
   
   CREATE INDEX idx_new_table_name ON new_table(name);
   
   COMMENT ON TABLE new_table IS '新表说明';
   COMMENT ON COLUMN new_table.name IS '字段说明';
   
   COMMIT;
   ```

3. **使用触发器自动维护 updated_at**
   ```sql
   CREATE OR REPLACE FUNCTION update_updated_at_column()
   RETURNS TRIGGER AS $$
   BEGIN
       NEW.updated_at = NOW();
       RETURN NEW;
   END;
   $$ language 'plpgsql';
   
   CREATE TRIGGER update_new_table_updated_at
       BEFORE UPDATE ON new_table
       FOR EACH ROW
       EXECUTE FUNCTION update_updated_at_column();
   ```

---

### 3.2 种子数据初始化

**推荐方案**:

1. **系统必需数据放在迁移文件中**
   ```sql
   INSERT INTO users (id, username, password_hash, is_system)
   VALUES ('00000000-0000-0000-0000-000000000001', 'admin', '$2a$10$...', true)
   ON CONFLICT DO NOTHING;
   ```

2. **业务数据通过管理界面导入**
   - KPI 定义、告警规则等通过 Web 界面配置
   - 提供导入导出功能(JSON/Excel)
   - 支持环境差异配置

3. **TR069 数据模型使用专用导入工具**
   ```bash
   omcgo-app --import-datamodels ./datamodels/seed/
   ```

---

### 3.3 环境差异化配置

```
cmd/app/etc/
├── config.dev.yaml      # 开发环境
├── config.test.yaml     # 测试环境
├── config.staging.yaml  # 预发布环境
└── config.prod.yaml     # 生产环境
```

**环境变量覆盖优先级**:
```
环境变量 > 配置文件 > 默认值
```

---

### 3.4 一键初始化脚本

```bash
#!/bin/bash
# scripts/init-dev.sh

set -e

echo "=== OMC Go Development Environment Setup ==="

# 1. 启动基础设施
echo "1. Starting infrastructure..."
docker-compose -f deployments/docker/docker-compose.yml up -d postgres redis nats minio

# 2. 等待服务就绪
echo "2. Waiting for services..."
sleep 10

# 3. 执行数据库迁移
echo "3. Running database migrations..."
go run ./cmd/migrate up --dsn "postgres://omcgo:omcgo@localhost:5432/omcgo?sslmode=disable"

# 4. 构建二进制
echo "4. Building binaries..."
make build

# 5. 启动服务
echo "5. Starting services..."
air -c .air.toml

echo "=== Setup Complete ==="
```

---

## 四、实施优先级与路线图

### 阶段一: 基础优化 (1-2 周) ⭐ 包含 SQL Builder 基础设施

| 任务 | 优先级 | 工作量 | 状态 |
|------|--------|--------|------|
| **SQL Builder 抽象层封装** | P0 | 1天 | 🆕 |
| **引入 Goose 迁移工具** | P0 | 2天 | 🆕 |
| **配置 sqlc 工具链** | P0 | 1天 | 🆕 |
| 迁移文件分类整理 | P0 | 2天 | 待开始 |
| 种子数据独立管理工具 | P0 | 3天 | 待开始 |
| 配置验证机制 | P1 | 1天 | 待开始 |
| 初始化健康检查 | P1 | 1天 | 待开始 |

### 阶段二: 架构增强 (2-3 周)

| 任务 | 优先级 | 工作量 | 状态 |
|------|--------|--------|------|
| **新模块使用 sqlc** | P0 | 持续 | 🆕 |
| **PM/KPI 查询迁移到 sqlc** | P0 | 3天 | 🆕 |
| 依赖图管理模块初始化 | P1 | 3天 | 待开始 |
| 动态查询引入 goqu | P1 | 2天 | 🆕 |
| 单元测试覆盖率提升 | P1 | 5天 | 待开始 |
| 集成测试环境优化 | P2 | 3天 | 待开始 |

### 阶段三: 生产加固 (持续)

| 任务 | 优先级 | 工作量 | 状态 |
|------|--------|--------|------|
| 核心 CRUD 逐步迁移到 sqlc | P1 | 持续 | 🆕 |
| 配置热重载 | P2 | 2天 | 待开始 |
| E2E 测试框架 | P2 | 5天 | 待开始 |
| 性能基准测试 | P1 | 3天 | 待开始 |
| 监控告警完善 | P1 | 3天 | 待开始 |

---

## 五、总结

### 5.1 当前架构评分

| 维度 | 评分 | 说明 |
|------|------|------|
| 整体架构 | 9/10 | 模块化单体选择正确,部署单元清晰 |
| 技术栈 | 8/10 | 选型合理,但 Squirrel 需规划迁移 |
| 目录结构 | 8/10 | 规范清晰,但迁移文件需整理 |
| 代码质量 | 8/10 | Repository 模式规范,测试覆盖需提升 |
| 可维护性 | 7/10 | 依赖管理可优化,种子数据需改进 |
| 可扩展性 | 8/10 | 支持 10 万级,预留 100 万扩展路径 |

### 5.2 核心建议

1. **立即执行** (1-2周):
   - ✅ 封装 SQL Builder 抽象层 (零风险)
   - ✅ 引入 Goose + sqlc 工具链
   - ✅ 迁移文件分类整理
   - ✅ 种子数据独立管理

2. **短期优化** (1个月):
   - 🔄 新模块使用 sqlc
   - 🔄 PM/KPI 查询迁移到 sqlc
   - 🔄 配置验证 + 依赖图管理

3. **长期规划** (3个月):
   - 🔄 核心 CRUD 逐步迁移到 sqlc
   - 🔄 E2E 测试 + 性能基准
   - 🔄 完全弃用 Squirrel

### 5.3 风险与注意事项

1. **SQL Builder 迁移风险**: 通过抽象层隔离,可随时回退
2. **迁移文件重构风险**: 需确保生产环境迁移兼容性
3. **依赖图引入风险**: 初期增加复杂度,需充分测试
4. **测试环境依赖**: Testcontainers 需要 Docker 环境

---

## 六、参考资料

### SQL Builder 与工具链
- [SQL Builder 演进建议与工具分工方案](./sql-builder-evolution-plan.md) ⭐
- [Goose + sqlc + pgx 完整示例](./examples/README.md) ⭐
- [sqlc 官方文档](https://docs.sqlc.dev/)
- [goqu 官方文档](https://github.com/doug-martin/goqu)
- [Goose 迁移工具](https://github.com/pressly/goose)

### 项目架构
- [Go 项目布局标准](https://github.com/golang-standards/project-layout)
- [PostgreSQL JSONB 类型](https://www.postgresql.org/docs/current/datatype-json.html)
- [TimescaleDB hypertable](https://docs.timescale.com/use-timescale/latest/hypertables/)
- [依赖注入模式](https://en.wikipedia.org/wiki/Dependency_injection)

---

## 七、决策清单

### SQL Builder 相关
- [ ] 是否立即封装 Squirrel 抽象层?
- [ ] 是否引入 Goose 替代 golang-migrate?
- [ ] 是否在新模块中使用 sqlc?
- [ ] 是否开始迁移 PM/KPI 查询到 sqlc?
- [ ] 动态查询是否使用 goqu?

### 其他重构
- [ ] 迁移文件分类整理方案确认?
- [ ] 种子数据独立管理工具确认?
- [ ] 配置验证机制确认?
- [ ] 依赖图管理确认?
- [ ] 实施优先级确认?

**决策人**: _____________  
**决策日期**: _____________

---

## 八、文档版本历史

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|---------|------|
| v2.0 | 2025-04-10 | 新增 SQL Builder 演进建议、工具分工方案、完整示例 | AI |
| v1.0 | 2025-04-10 | 初始版本,架构分析与重构建议 | AI |

---

**文档维护**: 随着项目实施持续更新  
**最后更新**: 2025-04-10  
**下次审查**: 团队决策后
