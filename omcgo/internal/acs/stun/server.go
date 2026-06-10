package stun

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Config holds STUN UDP server settings.
type Config struct {
	Enabled         bool          `mapstructure:"enabled"`
	ListenAddr      string        `mapstructure:"listen_addr"`       // e.g. ":3478"
	WorkerSize      int           `mapstructure:"worker_size"`       // number of reader goroutines
	BufferSize      int           `mapstructure:"buffer_size"`       // UDP read buffer size in bytes
	MaxStoreEntries int           `mapstructure:"max_store_entries"` // L1 address-cache entry cap (0 = default)
	CacheTTL        time.Duration `mapstructure:"cache_ttl"`         // STUN address cache TTL
	SharedSecret    string        `mapstructure:"shared_secret"`     // HMAC-SHA1 secret for CPE UDP CR
}

// Metrics holds Prometheus metrics for the STUN server.
type Metrics struct {
	PacketsReceived *prometheus.CounterVec
	PacketsInvalid  prometheus.Counter
	StoreSize       prometheus.Gauge
}

// NewMetrics registers STUN metrics with the given Prometheus registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		PacketsReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_stun_packets_received_total",
			Help: "Total STUN packets received by type",
		}, []string{"type"}),
		PacketsInvalid: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "acs_stun_packets_invalid_total",
			Help: "Total invalid/unparseable STUN packets",
		}),
		StoreSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_stun_store_size",
			Help: "Number of device addresses in the STUN store",
		}),
	}
	reg.MustRegister(m.PacketsReceived, m.PacketsInvalid, m.StoreSize)
	return m
}

// Server is the STUN UDP server that listens for device address discovery packets.
type Server struct {
	connMu    sync.Mutex // guards conn: Start 写 / Stop 与 worker goroutine 读跨 goroutine
	conn      *net.UDPConn
	processor *Processor
	store     *Store
	config    Config
	metrics   *Metrics
	logger    *zap.Logger
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewServer creates a new STUN UDP server.
func NewServer(cfg Config, store *Store, logger *zap.Logger) *Server {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":3478"
	}
	if cfg.WorkerSize <= 0 {
		cfg.WorkerSize = runtime.NumCPU()
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 2048
	}
	if cfg.CacheTTL > 0 {
		store.SetTTL(cfg.CacheTTL)
	}
	if cfg.MaxStoreEntries > 0 {
		store.SetMaxEntries(cfg.MaxStoreEntries)
	}

	return &Server{
		processor: NewProcessor(store, logger),
		store:     store,
		config:    cfg,
		logger:    logger,
		done:      make(chan struct{}),
	}
}

// SetMetrics attaches Prometheus metrics to the server.
func (s *Server) SetMetrics(m *Metrics) {
	s.metrics = m
}

// Start begins listening for UDP packets on the configured address.
// This method blocks until Stop is called or a fatal error occurs.
func (s *Server) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp4", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("resolve stun listen addr: %w", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("listen stun udp: %w", err)
	}
	s.setConn(conn)

	s.logger.Info("STUN UDP server started",
		zap.String("addr", s.config.ListenAddr),
		zap.Int("workers", s.config.WorkerSize))

	// Start cache cleanup goroutine
	s.wg.Add(1)
	go s.cleanupLoop()

	// Start worker goroutines
	for i := 0; i < s.config.WorkerSize; i++ {
		s.wg.Add(1)
		go s.readLoop(i)
	}

	// Wait for context cancellation or done signal
	select {
	case <-ctx.Done():
	case <-s.done:
	}

	return nil
}

// Stop gracefully shuts down the STUN server.
func (s *Server) Stop() error {
	s.logger.Info("STUN UDP server stopping")
	close(s.done)

	if conn := s.getConn(); conn != nil {
		conn.Close() // this will unblock ReadFromUDP calls
	}

	s.wg.Wait()
	s.logger.Info("STUN UDP server stopped")
	return nil
}

// setConn stores the UDP connection under the conn mutex.
func (s *Server) setConn(conn *net.UDPConn) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	s.conn = conn
}

// getConn returns the UDP connection under the conn mutex.
func (s *Server) getConn() *net.UDPConn {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	return s.conn
}

// LocalAddr returns the local address the server is listening on,
// or nil if the server has not started listening yet.
func (s *Server) LocalAddr() net.Addr {
	conn := s.getConn()
	if conn == nil {
		return nil
	}
	return conn.LocalAddr()
}

// Store returns the server's address store for external use.
func (s *Server) Store() *Store {
	return s.store
}

// readLoop is a worker goroutine that reads and processes UDP packets.
func (s *Server) readLoop(workerID int) {
	defer s.wg.Done()

	// 经锁取一次 conn 快照：worker 在 Start 设置 conn 之后创建，
	// 此处必非 nil；后续读写走本地变量，与 Stop 的读取共用同一把锁同步。
	conn := s.getConn()

	buf := make([]byte, s.config.BufferSize)
	for {
		select {
		case <-s.done:
			return
		default:
		}

		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-s.done:
				return // expected: conn closed during shutdown
			default:
				s.logger.Warn("stun read error",
					zap.Int("worker", workerID),
					zap.Error(err))
				continue
			}
		}

		if n == 0 {
			continue
		}

		// Copy data to avoid buffer reuse issues
		data := make([]byte, n)
		copy(data, buf[:n])

		// Track metrics
		if s.metrics != nil {
			if IsSTUN(data) {
				s.metrics.PacketsReceived.WithLabelValues("standard").Inc()
			} else {
				s.metrics.PacketsReceived.WithLabelValues("nonstandard").Inc()
			}
		}

		// Process the packet
		resp := s.processor.HandlePacket(context.Background(), data, addr)

		// Send response if there is one
		if resp != nil {
			if _, err := conn.WriteToUDP(resp, addr); err != nil {
				s.logger.Warn("stun write error",
					zap.String("dst", addr.String()),
					zap.Error(err))
			}
		}
	}
}

// cleanupLoop periodically removes expired entries from the L1 cache
// and updates the store size metric.
func (s *Server) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			cleaned := s.store.CleanExpired()
			size := s.store.Size()

			if s.metrics != nil {
				s.metrics.StoreSize.Set(float64(size))
			}

			if cleaned > 0 {
				s.logger.Info("stun store cleanup",
					zap.Int("cleaned", cleaned),
					zap.Int("remaining", size))
			}
		}
	}
}
