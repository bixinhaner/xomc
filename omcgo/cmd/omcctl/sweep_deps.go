package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/devsweep"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
)

// sweepDeps 是 omcctl sweep-paths 在 worker 容器内直连执行所需的全部依赖。
// 寿命:CLI 单次调用,Close 释放连接池。
type sweepDeps struct {
	pool       *pgxpool.Pool
	redis      redis.UniversalClient
	logger     *zap.Logger
	xmlBaseDir string
	closers    []func()
}

func (d *sweepDeps) Close() {
	for i := len(d.closers) - 1; i >= 0; i-- {
		d.closers[i]()
	}
}

// loadSweepDeps 加载 worker 配置(/etc/omcgo/worker.<env>.yaml),初始化 pgxpool +
// redis + logger,返 sweepDeps。本地 dev 跑(非容器内)时 env vars 提供 override。
//
// 环境检测:OMCGO_ENV 缺省视 dev。配置文件查找顺序:
//  1. $OMCGO_CONFIG_PATH(显式 override)
//  2. /etc/omcgo/worker.<env>.yaml(worker 容器标准路径)
//  3. omcgo/cmd/worker/etc/config.<env>.yaml(本地源码 cwd 兜底)
func loadSweepDeps(ctx context.Context) (*sweepDeps, error) {
	env := os.Getenv("OMCGO_ENV")
	if env == "" {
		env = "dev"
	}

	dsn, redisAddrs, xmlBaseDir, err := readWorkerConfig(env)
	if err != nil {
		return nil, fmt.Errorf("load worker config: %w", err)
	}

	// pgxpool
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	poolCfg.MaxConns = 8
	poolCfg.MinConns = 1
	poolCfg.MaxConnIdleTime = 30 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect pg: %w", err)
	}

	// redis (univeral — 单机或集群)
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    redisAddrs,
		PoolSize: 8,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	logger, _ := zap.NewProduction()

	return &sweepDeps{
		pool:       pool,
		redis:      rdb,
		logger:     logger,
		xmlBaseDir: xmlBaseDir,
		closers: []func(){
			func() { _ = rdb.Close() },
			func() { pool.Close() },
			func() { _ = logger.Sync() },
		},
	}, nil
}

// readWorkerConfig 用 viper 读 worker.<env>.yaml,只挑 DSN / redis / XML 路径
// 三个字段。env vars 优先级最高(便于 CI 测试覆盖)。
func readWorkerConfig(env string) (dsn string, redisAddrs []string, xmlBaseDir string, err error) {
	// 1. env override
	if v := os.Getenv("OMCGO_DB_DSN"); v != "" {
		dsn = v
	}
	if v := os.Getenv("OMCGO_REDIS_ADDR"); v != "" {
		redisAddrs = []string{v}
	}

	// 2. 配置文件
	v := viper.New()
	v.SetConfigType("yaml")
	candidates := []string{
		os.Getenv("OMCGO_CONFIG_PATH"),
		"/etc/omcgo/worker." + env + ".yaml",
		"omcgo/cmd/worker/etc/config." + env + ".yaml",
		"cmd/worker/etc/config." + env + ".yaml",
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, statErr := os.Stat(p); statErr == nil {
			v.SetConfigFile(p)
			if rErr := v.ReadInConfig(); rErr == nil {
				break
			}
		}
	}

	if dsn == "" {
		dsn = v.GetString("db.dsn")
	}
	if len(redisAddrs) == 0 {
		redisAddrs = v.GetStringSlice("redis.addrs")
	}
	xmlBaseDir = v.GetString("dictloader.xml_base_dir")

	if dsn == "" {
		return "", nil, "", fmt.Errorf("no DB DSN (set OMCGO_DB_DSN env or db.dsn in worker config)")
	}
	if len(redisAddrs) == 0 {
		return "", nil, "", fmt.Errorf("no redis addrs (set OMCGO_REDIS_ADDR env or redis.addrs in worker config)")
	}
	// XML 默认路径:容器内 /etc/omcgo/data + dictloader directory(param-mappings)
	if xmlBaseDir == "" {
		xmlBaseDir = "data"
	}
	xmlBaseDir = xmlBaseDir + "/param-mappings"
	if _, err := os.Stat("/etc/omcgo/" + xmlBaseDir); err == nil {
		xmlBaseDir = "/etc/omcgo/" + xmlBaseDir
	}

	return dsn, redisAddrs, xmlBaseDir, nil
}

// buildSweepService 用已初始化的依赖装配 devsweep.Service + Applier。
func (d *sweepDeps) buildSweepService(ctx context.Context) (*devsweep.Service, *devsweep.Applier, error) {
	// Product Registry — 必须 Refresh 才能 MatchProductClass
	prodRepo := product.NewPgRepository(d.pool)
	prodCache := product.NewRedisCache(d.redis)
	prodReg := product.NewRegistry(prodRepo, prodCache, nil, d.logger)
	if err := prodReg.Refresh(ctx); err != nil {
		return nil, nil, fmt.Errorf("product registry refresh: %w", err)
	}

	// ParamModel Registry (GetByParamModel 不需要 productGetter)
	paramRepo := parammodel.NewPgRepository(d.pool)
	paramCache := parammodel.NewRedisCache(d.redis)
	paramReg := parammodel.NewRegistry(paramRepo, paramCache, nil, nil, d.logger)

	// Device lookup — 直接用 PgDeviceRepository (满足 DeviceLookup 接口)
	deviceRepo := device.NewPgDeviceRepository(d.pool)

	// Task service
	taskQueue := task.NewRedisTaskQueue(d.redis)
	taskRepo := task.NewPgTaskRepository(d.pool)
	taskSvc := task.NewTaskService(taskQueue, taskRepo, d.logger)

	// Prober + Repository + Service
	prober := devsweep.NewTaskProber(taskSvc, 0, 0, d.logger)
	dbRepo := devsweep.NewPgRepository(d.pool)
	svc := devsweep.NewService(deviceRepo, prodReg, paramReg, prober, dbRepo, d.logger)

	// Applier
	xmlWriter := devsweep.NewXMLWriter(d.xmlBaseDir)
	applier := devsweep.NewApplier(d.pool, d.redis, xmlWriter, paramRepo, dbRepo, d.logger)

	return svc, applier, nil
}
