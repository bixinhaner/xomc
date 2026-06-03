package provider

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	alarmdef "github.com/omcgo/omcgo/internal/alarm/definition"
	"github.com/omcgo/omcgo/internal/core/dictloader"
)

// defaultAlarmDefRefreshTimeout 控制 P3-04 启动期 alarmdef.Registry.Refresh 的硬上限。
const defaultAlarmDefRefreshTimeout = 30 * time.Second

// alarmdefReloaderUnwiredErr 当 dictloader.Registry 未注入时，upload-xml 端点的重载
// 步骤返回此错误（FileHandler 仅 Warn，不致命）。
var alarmdefReloaderUnwiredErr = errors.New("dictloader registry not wired in app")

// initAlarmDefModule 装配 T-0098 P3-04 告警定义 handler 链路：
//
//	PgRepository → Registry (in-memory) → Service → Handler
//
// 与 dictload 不同：Registry 只读 DB，不读 XML；DB 数据由 dictload 阶段写入。
// 因此本模块声明 Depends=["dictload"]，确保启动期顺序正确（也容忍空 DB）。
//
// Provider 同时构造 dictloader.Registry → Reloader 适配器，供 FileHandler 的
// upload-xml 流程（写文件 → destructive 重载删孤儿 → RefreshCache）调用 ReloadOne。
func initAlarmDefModule(c *Container) error {
	logger := c.Logger.Named("alarmdef")

	repo := alarmdef.NewPgRepository(c.PgPool)
	metrics := alarmdef.NewRegistryMetrics(c.MetricsReg)
	registry := alarmdef.NewRegistry(repo, metrics, logger)

	// 启动期 Refresh：加载 DB 全量到 sync.Map。失败仅 WARN（DB 空 → 后续 lookup miss）。
	ctx, cancel := context.WithTimeout(context.Background(), defaultAlarmDefRefreshTimeout)
	defer cancel()
	if err := registry.Refresh(ctx); err != nil {
		logger.Warn("alarm-definition registry initial refresh failed; lookup will miss until reload")
	}

	service := alarmdef.NewService(repo, registry, c.Redis, logger)
	reloader := &alarmDefReloader{reg: c.DictLoaderRegistry}
	handler := alarmdef.NewHandler(service, logger)

	// 告警库 XML 管理重构:导入/重载/刷新合并进 FileHandler 的 upload-xml 端点
	// (写文件 → destructive 重载删孤儿 → RefreshCache)。FileHandler 复用 service
	// (RefreshCache + DeleteOrphansSince)与 reloader(ReloadOne)。上传统一写进 builtin
	// 目录,启动期确保该目录存在。
	fileRepo := alarmdef.NewPgFileRepository(c.PgPool)
	fileHandler := alarmdef.NewFileHandler(fileRepo, service, reloader, c.Cfg.DictLoader.XMLBaseDir, logger)
	if err := alarmdef.EnsureBaseDir(c.Cfg.DictLoader.XMLBaseDir); err != nil {
		logger.Warn("ensure alarm builtin dir failed; uploads may fail until dir exists", zap.Error(err))
	}

	c.AlarmDefRegistry = registry
	c.AlarmDefHandler = handler
	c.AlarmDefFileHandler = fileHandler
	logger.Info("alarm-definition module initialized",
		// 暴露行数到 startup log，便于排障
	)
	return nil
}

// alarmDefReloader 把 dictloader.Registry.ReloadOne(...)(Report, error) 适配为
// alarmdef.Reloader 期望的 ReloadOne(ctx, name) error。
type alarmDefReloader struct {
	reg *dictloader.Registry
}

func (r *alarmDefReloader) ReloadOne(ctx context.Context, name string) error {
	if r.reg == nil {
		return alarmdefReloaderUnwiredErr
	}
	_, err := r.reg.ReloadOne(ctx, name)
	return err
}
