package minio

import (
	"fmt"
	"sync/atomic"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

// PresignClientProvider 提供运行时变更感知的 MinIO 预签名 client（issue #548 切片 2）。
//
// 调用方（trace / mml / license / backup）每次签 URL 前 Get() 拿当前 client，
// 不要把 Get() 的返回缓存——sys_configs 写入 storage.minio_public_endpoint 时
// PresignBridge 会原子替换内部 client，缓存会拿到陈旧 endpoint。
type PresignClientProvider interface {
	// Get 返回当前 effective endpoint 对应的 MinIO client。永不返回 nil
	// （构造失败也会回退到 NewMinIOClient 内部 endpoint，保留现有 Warn 兜底）。
	Get() *minio.Client
}

// PresignBridge 是 PresignClientProvider 的官方实现，按"sys_configs > env/YAML > 内部"
// 优先级 atomic 维护一个 *minio.Client：
//
//   - 启动期初值 = envFallback（cfg.PublicEndpoint，YAML 已 expand env MINIO_PUBLIC_ENDPOINT）
//   - 运行期 SetPublicEndpoint(s) 把 sys_configs 写入的新值切进来：
//     s 非空 → effective=s；s 空 → 回退 envFallback；envFallback 也空 → cfg.Endpoint（内部硬兜底）
//   - effective endpoint 未变（幂等）时不重建 client，避免反复 alloc
//
// 故意只暴露 Get + SetPublicEndpoint：不提供"切换 access_key / use_ssl"等 API——
// 那些静态项保留由 YAML 启动期一次性加载（issue #548 切片 2 不在 scope）。
type PresignBridge struct {
	cfg         appconfig.MinIOConfig // 静态部分：access key / secret / use_ssl / 兜底 Endpoint
	envFallback string                // 启动期一次性快照的 cfg.PublicEndpoint（env MINIO_PUBLIC_ENDPOINT 展开后）
	logger      *zap.Logger

	current         atomic.Pointer[minio.Client] // 当前 client（按 effective endpoint 重建）
	currentEndpoint atomic.Pointer[string]       // 当前 effective endpoint 字符串（幂等比较用）
	internalWarned  atomic.Bool                  // 是否已打过"全空回退内部 endpoint"的 Warn（去重）
}

// 编译期接口断言
var _ PresignClientProvider = (*PresignBridge)(nil)

// NewPresignBridge 构造订阅桥。初值用 cfg.PublicEndpoint 作 envFallback：
//
//   - cfg.PublicEndpoint 非空：用它建 presign client（行为等同既有 NewPresignClient）
//   - cfg.PublicEndpoint 为空：回退用内部 cfg.Endpoint 建 client（同时打一次 Warn，
//     与 NewPresignClient(...,logger) 的兜底逻辑等价）
//
// 返回错误仅在 minio.New 构造失败时发生——access_key / secret_key 非法或
// endpoint 字符串完全无法解析。正常 dev/prod 配置不会触发。
func NewPresignBridge(cfg appconfig.MinIOConfig, logger *zap.Logger) (*PresignBridge, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	b := &PresignBridge{
		cfg:         cfg,
		envFallback: cfg.PublicEndpoint,
		logger:      logger.Named("minio.presign-bridge"),
	}
	// 触发一次初始化：endpoint = "" 走优先级回退（envFallback 或内部）。
	if err := b.applyEndpoint(""); err != nil {
		return nil, err
	}
	return b, nil
}

// Get 返回当前 client。永不 nil（构造失败的退路在 applyEndpoint 内部已兜底）。
func (b *PresignBridge) Get() *minio.Client {
	return b.current.Load()
}

// CurrentEndpoint 返回当前 effective endpoint 字符串（用于运维/测试自检）。
// 可能形态：sys_configs 值 / envFallback / cfg.Endpoint（内部硬兜底）。
func (b *PresignBridge) CurrentEndpoint() string {
	if p := b.currentEndpoint.Load(); p != nil {
		return *p
	}
	return ""
}

// SetPublicEndpoint 用 sys_configs 写入的新值更新桥。
//
//   - s 空字符串：表示运维删了 sys_configs 那条配置 → 回退 envFallback
//   - s 非空：先 ValidatePublicEndpoint 校验，通过后原子替换 client
//   - effective endpoint 与当前相同：幂等返回，不重建（避免 SavedHook 反复触发时浪费）
//
// 校验失败返非 nil error，当前 client 不变。
//
// 并发安全：多 goroutine 同时调用安全（atomic.Pointer 替换）；与 Get() 完全并行。
func (b *PresignBridge) SetPublicEndpoint(s string) error {
	if err := ValidatePublicEndpoint(s); err != nil {
		return err
	}
	return b.applyEndpoint(s)
}

// applyEndpoint 内部核心：算 effective endpoint，幂等比较，必要时 minio.New 重建并原子替换。
// sysConfigValue 是 sys_configs.storage.minio_public_endpoint 当前值（"" = 未设/已删）。
func (b *PresignBridge) applyEndpoint(sysConfigValue string) error {
	effective, fromInternal := b.resolveEffective(sysConfigValue)

	if cur := b.currentEndpoint.Load(); cur != nil && *cur == effective {
		// 幂等：endpoint 没变，不重建
		return nil
	}

	client, err := b.buildClient(effective)
	if err != nil {
		return fmt.Errorf("rebuild minio presign client for endpoint %q: %w", effective, err)
	}
	b.current.Store(client)
	b.currentEndpoint.Store(&effective)

	switch {
	case fromInternal:
		// 全空回退内部，首次打 Warn 后去重——qa-614 #377 兜底语义不丢，但不刷屏。
		if b.internalWarned.CompareAndSwap(false, true) {
			b.logger.Warn("MinIO public_endpoint 全空，预签名 URL 回退使用内部 endpoint；"+
				"浏览器 / 外部 SDK 可能解析失败（ERR_NAME_NOT_RESOLVED）。"+
				"请在「系统配置 → 存储」UI 配置 public_endpoint，或经 .env 注入 MINIO_PUBLIC_ENDPOINT",
				zap.String("internal_endpoint", b.cfg.Endpoint))
		}
	case sysConfigValue != "":
		b.logger.Info("MinIO presign endpoint 切换（sys_configs）",
			zap.String("endpoint", effective))
	default:
		// 仅初始化 / 回退 env 路径
		b.logger.Info("MinIO presign endpoint 应用（env/YAML fallback）",
			zap.String("endpoint", effective))
	}
	return nil
}

// resolveEffective 按优先级返算 effective endpoint：
//
//	sysConfigValue（非空）→ envFallback（非空）→ cfg.Endpoint（内部硬兜底，fromInternal=true）
//
// 返回 fromInternal=true 标记调用方需要打/查 Warn。
func (b *PresignBridge) resolveEffective(sysConfigValue string) (effective string, fromInternal bool) {
	if sysConfigValue != "" {
		return sysConfigValue, false
	}
	if b.envFallback != "" {
		return b.envFallback, false
	}
	return b.cfg.Endpoint, true
}

// buildClient 构造一个 MinIO client 用于预签名。endpoint 必须是 host[:port] 无 scheme
// （由 ValidatePublicEndpoint 或 envFallback 来源保证）。Region 显式设置 "us-east-1"
// 跳过 GetBucketLocation 探测——与 NewPresignClient 同理（见 minio.go 注释）。
func (b *PresignBridge) buildClient(endpoint string) (*minio.Client, error) {
	if endpoint == "" {
		// 防御性：理论上 resolveEffective 已兜底 cfg.Endpoint；若 cfg.Endpoint 也空
		// 直接报错让 wiring 期暴露问题，比构造一个 endpoint="" 的 client 强。
		return nil, fmt.Errorf("minio endpoint 为空（cfg.Endpoint / cfg.PublicEndpoint / sys_configs 全空）")
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(b.cfg.AccessKey, b.cfg.SecretKey, ""),
		Secure: b.cfg.UseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("minio.New(%q): %w", endpoint, err)
	}
	return client, nil
}
