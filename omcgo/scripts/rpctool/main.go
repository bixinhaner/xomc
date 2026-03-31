package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	pgcomp "github.com/omcgo/omcgo/internal/core/components/postgres"
	rediscomp "github.com/omcgo/omcgo/internal/core/components/redis"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfgPath    string
	deviceSN   string
	priority   int
	ttlSeconds int
	dryRun     bool
	source     string
	creator    string
	desc       string
	redisAddr  string // 命令行覆盖 Redis 地址
	dbDSN      string // 命令行覆盖 PostgreSQL DSN

	redisClient redis.UniversalClient
	pgPool      *pgxpool.Pool
	taskSvc     *task.TaskService
	acsCfg      appconfig.ACSConfig
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "rpctool",
		Short: "TR069 RPC 任务工具 — 创建 RPC 任务并记录到数据库",
		Long: `rpctool 是一个独立的 CLI 工具，用于创建 TR-069 RPC 任务。
任务同时写入 Redis 队列和 PostgreSQL device_tasks 表，保证完整的任务生命周期追踪。

当 CPE 设备下次连接 ACS 时（通过周期性 Inform 或 Connection Request），
ACS 引擎会从队列中取出任务并发送给设备执行。

配置文件默认加载 cmd/acs/etc/config.local.yaml（本地开发环境，Redis/DB 连接 localhost）。
也可通过 --redis 和 --db 参数直接覆盖连接地址，无需修改配置文件。`,
		SilenceUsage: true,
	}

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "cmd/acs/etc/config.local.yaml", "配置文件路径 (ACS 配置)")
	rootCmd.PersistentFlags().StringVar(&redisAddr, "redis", "", "Redis 地址 (覆盖配置文件，如 localhost:6379)")
	rootCmd.PersistentFlags().StringVar(&dbDSN, "db", "", "PostgreSQL DSN (覆盖配置文件)")
	rootCmd.PersistentFlags().StringVar(&deviceSN, "sn", "", "设备序列号 (必���)")
	rootCmd.PersistentFlags().IntVar(&priority, "priority", 10, "任务优先级 (数值越小优先级越高)")
	rootCmd.PersistentFlags().IntVar(&ttlSeconds, "ttl", 0, "任务过期时间(秒)，0 表示不过期")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "仅打印任务 JSON，不写入数据库")
	rootCmd.PersistentFlags().StringVar(&source, "source", "api", "任务来源 (api/scheduler/system)")
	rootCmd.PersistentFlags().StringVar(&creator, "creator", "rpctool", "创建者 ID")
	rootCmd.PersistentFlags().StringVar(&desc, "desc", "", "任务描述")

	rootCmd.AddCommand(
		newUploadCmd(),
		newDownloadCmd(),
		newGetPVCmd(),
		newSetPVCmd(),
		newGetPNCmd(),
		newRebootCmd(),
		newResetCmd(),
		newAddObjectCmd(),
		newDeleteObjectCmd(),
		newQueueCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// initInfra loads config and initializes Redis + PostgreSQL + TaskService.
func initInfra() error {
	if err := appconfig.Load(cfgPath, &acsCfg); err != nil {
		return fmt.Errorf("加载配置文件 %s: %w", cfgPath, err)
	}

	// 命令行参数覆盖配置文件中的连接地址
	if redisAddr != "" {
		acsCfg.Redis.Addrs = []string{redisAddr}
	}
	if dbDSN != "" {
		acsCfg.DB.DSN = dbDSN
	}

	ctx := context.Background()
	logger := zap.NewNop()

	// Connect Redis
	var err error
	redisClient, err = rediscomp.NewRedisClient(acsCfg.Redis)
	if err != nil {
		return fmt.Errorf("连接 Redis: %w", err)
	}

	// Connect PostgreSQL
	pgPool, err = pgcomp.NewPostgresPool(ctx, acsCfg.DB, logger)
	if err != nil {
		return fmt.Errorf("连接 PostgreSQL: %w", err)
	}

	// Create TaskService
	taskQueue := task.NewRedisTaskQueue(redisClient)
	taskRepo := task.NewPgTaskRepository(pgPool)
	taskSvc = task.NewTaskService(taskQueue, taskRepo, logger)

	return nil
}

// requireSN validates --sn is set.
func requireSN(cmd *cobra.Command, args []string) error {
	if deviceSN == "" {
		return fmt.Errorf("必须指定 --sn (设备序列号)")
	}
	return nil
}

// requireSNAndInfra validates --sn and initializes infrastructure (unless dry-run).
func requireSNAndInfra(cmd *cobra.Command, args []string) error {
	if err := requireSN(cmd, args); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return initInfra()
}

// requireInfra initializes infrastructure (unless dry-run), without requiring --sn.
func requireInfra(cmd *cobra.Command, args []string) error {
	if dryRun {
		return nil
	}
	return initInfra()
}
