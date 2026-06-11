// Command kpiperf 是 KPI(PM) 文件上传压测工具：模拟大量小基站向 ACS 直传
// 3GPP 32.435 性能文件，压测「文件存储（ACS→MinIO）」与「KPI 解析入库（worker→pm_metrics）」
// 两段能力。
//
// 链路（直传路径，不走 AutonomousTransferComplete SOAP）：
//
//	kpiperf ──PUT /smallcell/FileUploadService?fileType=4&sn=..&filename=..──▶ ACS
//	  ACS 流式落 MinIO(pm-files) + 发 pm.file.received(瘦 payload, 仅 device_sn)
//	  worker 订阅 → 按 SN 查 devices 表(必须预先存在) → 解析 XML → 批量写 pm_metrics → 反算 KPI
//
// 因为 worker 的 resolveDevice 对未知 SN 会报错重试进 DLQ，所以「解析入库」这一段
// 必须先注入测试设备（-mode seed 或 run 模式默认 -seed）。文件存储那一段对任意 SN 都成立。
//
// 用法见同目录 README.md。
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type config struct {
	mode          string
	baseURL       string
	templates     string
	concurrency   int
	devices       int
	files         int
	duration      time.Duration
	buckets       int
	snPrefix      string
	oui           string
	carrier       string
	tech          string
	productClass  string
	fileType      string
	username      string
	password      string
	dsn           string
	noDB          bool
	seed          bool
	cleanupAfter  bool
	workerMetrics string
	drainTimeout  time.Duration
	reqTimeout    time.Duration
	jsonOut       bool
	rewriteBodySN bool
	insecure      bool
}

type bucketWindow struct {
	begin time.Time
	end   time.Time
}

func main() {
	cfg := parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigc
		fmt.Fprintln(os.Stderr, "\n收到中断信号，正在停止…（再次 Ctrl-C 强制退出）")
		cancel()
		<-sigc
		os.Exit(130)
	}()

	switch cfg.mode {
	case "seed":
		runSeed(ctx, cfg)
	case "cleanup":
		runCleanup(ctx, cfg)
	case "run":
		runLoad(ctx, cfg)
	default:
		fmt.Fprintf(os.Stderr, "未知 -mode=%q（可选 run|seed|cleanup）\n", cfg.mode)
		os.Exit(2)
	}
}

func parseFlags() config {
	var c config
	flag.StringVar(&c.mode, "mode", "run", "运行模式：run（注入+上传+验证）| seed（仅注入设备）| cleanup（仅清理测试数据）")
	flag.StringVar(&c.baseURL, "url", "http://localhost:7557", "ACS 上传端点 base URL（直连 ACS :7557，或经 nginx :8080）")
	flag.StringVar(&c.templates, "templates",
		"test/pmperf/A20260611.0815+0800-0830+0800_48BF74.120299024119AA05000.xml,test/pmperf/A20260611.0900+0800-0915+0800_48BF74.120200088897AA04259.xml",
		"逗号分隔的模板 XML 路径（多个则按文件序轮转，4G/5G 混压）")
	flag.IntVar(&c.concurrency, "concurrency", 200, "最大并发上传数（在飞请求上限），范围建议 200–10000")
	flag.IntVar(&c.devices, "devices", 10000, "模拟/注入的不同设备 SN 数（run 模式默认会注入这么多设备）")
	flag.IntVar(&c.files, "files", 0, "总上传文件数；0 表示 devices*buckets（每设备每时间窗一份）")
	flag.DurationVar(&c.duration, "duration", 0, "按时长持续压测（>0 时忽略 -files，循环复用 SN×时间窗）")
	flag.IntVar(&c.buckets, "buckets", 1, "不同 15 分钟 KPI 时间窗个数（向最近的已完成窗口回溯铺开，扩大唯一行空间）")
	flag.StringVar(&c.snPrefix, "sn-prefix", "KPILT", "测试设备 SN 前缀（清理按该前缀 LIKE 命中）")
	flag.StringVar(&c.oui, "oui", "48BF74", "设备 OUI（6 位十六进制，进设备表与文件名厂商段）")
	flag.StringVar(&c.carrier, "carrier", "cmcc", "运营商：cmcc|ctcc|cucc|other（devices 分区键）")
	flag.StringVar(&c.tech, "tech", "lte", "制式：lte|nr|gsm（设备表属性，不影响解析入库）")
	flag.StringVar(&c.productClass, "product-class", "", "设备 product_class（留空也能入 counter；填了才便于 KPI 路由）")
	flag.StringVar(&c.fileType, "filetype", "4", "上传 fileType 查询值（PM=4）")
	flag.StringVar(&c.username, "username", "", "上传端点 Basic Auth 用户名（默认无鉴权）")
	flag.StringVar(&c.password, "password", "", "上传端点 Basic Auth 密码")
	flag.StringVar(&c.dsn, "db", "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable", "PostgreSQL DSN（注入/验证/清理用；宿主默认映射 5432）")
	flag.BoolVar(&c.noDB, "no-db", false, "run 模式下跳过注入与入库验证，只压文件存储（纯 HTTP）")
	flag.BoolVar(&c.seed, "seed", true, "run 模式下上传前注入设备")
	flag.BoolVar(&c.cleanupAfter, "cleanup-after", false, "run 模式结束后一键清理测试数据")
	flag.StringVar(&c.workerMetrics, "worker-metrics", "http://localhost:9092/metrics", "worker Prometheus /metrics（抓 PM 解析指标增量）")
	flag.DurationVar(&c.drainTimeout, "drain", 120*time.Second, "上传完成后等待入库收敛的最长时长")
	flag.DurationVar(&c.reqTimeout, "timeout", 30*time.Second, "单次上传 HTTP 超时")
	flag.BoolVar(&c.jsonOut, "json", false, "以 JSON 输出报告")
	flag.BoolVar(&c.rewriteBodySN, "rewrite-body-sn", true, "改写 body 内 managedElement localDn 的 SN（保真，不影响路由）")
	flag.BoolVar(&c.insecure, "insecure", false, "https 时跳过 TLS 证书校验")
	flag.Parse()
	return c
}

// ---------- seed / cleanup 模式 ----------

func runSeed(ctx context.Context, cfg config) {
	pool := mustDB(ctx, cfg)
	defer pool.Close()
	fmt.Printf("注入 %d 个测试设备（前缀 %s，OUI %s，%s/%s）…\n", cfg.devices, cfg.snPrefix, cfg.oui, cfg.carrier, cfg.tech)
	start := time.Now()
	n, err := seedDevices(ctx, pool, cfg.snPrefix, cfg.devices, cfg.oui, cfg.carrier, cfg.tech, cfg.productClass)
	if err != nil {
		fatal("注入设备失败: %v", err)
	}
	fmt.Printf("完成：新增 %d 个设备（已存在的跳过），耗时 %s\n", n, time.Since(start).Round(time.Millisecond))
}

func runCleanup(ctx context.Context, cfg config) {
	pool := mustDB(ctx, cfg)
	defer pool.Close()
	fmt.Printf("清理测试数据（SN 前缀 %s）…\n", cfg.snPrefix)
	start := time.Now()
	r, err := cleanupAll(ctx, pool, cfg.snPrefix)
	if err != nil {
		fatal("清理失败: %v", err)
	}
	fmt.Printf("完成：删除 pm_metrics %d 行、pm_files %d 行、devices %d 行，耗时 %s\n",
		r.metrics, r.files, r.devices, time.Since(start).Round(time.Millisecond))
}

// ---------- run 模式 ----------

func runLoad(ctx context.Context, cfg config) {
	// 1. 加载模板
	gens := loadTemplates(cfg)
	gran := time.Duration(gens[0].granSeconds) * time.Second
	wins := buildBuckets(cfg.buckets, gran)

	// 2. 解析总文件数
	totalFiles := cfg.files
	if cfg.duration <= 0 {
		if totalFiles == 0 {
			totalFiles = cfg.devices * cfg.buckets
		}
		uniqueSpace := cfg.devices * cfg.buckets
		if totalFiles > uniqueSpace {
			fmt.Fprintf(os.Stderr, "提示：-files=%d 超过唯一空间 devices*buckets=%d，超出部分会重复 (SN,时间窗) → pm_metrics UPSERT（不增行，但仍走解析入库）\n", totalFiles, uniqueSpace)
		}
	}

	printRunHeader(cfg, gens, wins, totalFiles)

	// 3. DB：连接 + 注入 + 基线快照
	var pool *pgxpool.Pool
	var baseCounts dbCounts
	if !cfg.noDB {
		pool = mustDB(ctx, cfg)
		defer pool.Close()
		if cfg.seed {
			fmt.Printf("注入 %d 个测试设备…\n", cfg.devices)
			t0 := time.Now()
			n, err := seedDevices(ctx, pool, cfg.snPrefix, cfg.devices, cfg.oui, cfg.carrier, cfg.tech, cfg.productClass)
			if err != nil {
				fatal("注入设备失败: %v", err)
			}
			fmt.Printf("  新增 %d 个设备，耗时 %s\n", n, time.Since(t0).Round(time.Millisecond))
		}
		var err error
		if baseCounts, err = queryDBCounts(ctx, pool, cfg.snPrefix); err != nil {
			fatal("读取基线计数失败: %v", err)
		}
	} else {
		fmt.Println("-no-db：跳过注入与入库验证，仅压测文件存储（未注入设备时 worker 会因找不到设备丢弃文件，pm_metrics 不增长）")
	}
	basePM := scrapeProm(ctx, cfg.workerMetrics, 5*time.Second)

	// 4. 上传风暴
	agg := runUploadStorm(ctx, cfg, gens, wins, totalFiles)
	printUploadReport(cfg, agg)

	// 5. 排空 + 入库验证
	var ing *ingestReport
	if !cfg.noDB && ctx.Err() == nil {
		ing = drainAndVerify(ctx, cfg, pool, baseCounts, basePM, int64(agg.ok))
		printIngestReport(cfg, ing)
	}

	// 6. JSON（可选）
	if cfg.jsonOut {
		emitJSON(cfg, agg, ing)
	}

	// 7. 清理（可选）
	if cfg.cleanupAfter && pool != nil {
		fmt.Println("\n按 -cleanup-after 清理测试数据…")
		r, err := cleanupAll(context.Background(), pool, cfg.snPrefix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "清理失败: %v\n", err)
		} else {
			fmt.Printf("已删除 pm_metrics %d 行、pm_files %d 行、devices %d 行\n", r.metrics, r.files, r.devices)
		}
	}
}

// runUploadStorm 用 cfg.concurrency 个 worker 并发上传，返回汇总统计。
func runUploadStorm(ctx context.Context, cfg config, gens []*generator, wins []bucketWindow, totalFiles int) *aggStat {
	client := newHTTPClient(cfg)
	endpoint := cfg.baseURL + "/smallcell/FileUploadService"

	var next int64 = -1
	var done int64
	deadline := time.Time{}
	if cfg.duration > 0 {
		deadline = time.Now().Add(cfg.duration)
	}

	statHint := totalFiles/cfg.concurrency + 16
	stats := make([]*workerStat, cfg.concurrency)

	// 进度条
	progStop := make(chan struct{})
	go progress(&done, totalFiles, cfg.duration, deadline, progStop)

	wallStart := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < cfg.concurrency; w++ {
		ws := newWorkerStat(statHint)
		stats[w] = ws
		wg.Add(1)
		go func(ws *workerStat) {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				if cfg.duration > 0 {
					if time.Now().After(deadline) {
						return
					}
				}
				i := atomic.AddInt64(&next, 1)
				if cfg.duration <= 0 && i >= int64(totalFiles) {
					return
				}
				seq := int(i)
				devIdx := seq % cfg.devices
				bkt := wins[(seq/cfg.devices)%len(wins)]
				gen := gens[seq%len(gens)]
				sn := deviceSN(cfg.snPrefix, devIdx+1)

				fname, body := gen.generate(sn, bkt.begin, bkt.end, seq)
				doUpload(ctx, client, endpoint, cfg, sn, fname, body, ws)
				atomic.AddInt64(&done, 1)
			}
		}(ws)
	}
	wg.Wait()
	wall := time.Since(wallStart)
	close(progStop)

	return mergeStats(stats, wall)
}

// doUpload 发一次直传 PUT，记录耗时与状态。
func doUpload(ctx context.Context, client *http.Client, endpoint string, cfg config, sn, fname string, body []byte, ws *workerStat) {
	q := url.Values{}
	q.Set("fileType", cfg.fileType)
	q.Set("filename", fname)
	q.Set("sn", sn)
	full := endpoint + "?" + q.Encode()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, full, bytes.NewReader(body))
	if err != nil {
		ws.errs++
		return
	}
	req.Header.Set("Content-Type", "application/xml")
	req.ContentLength = int64(len(body))
	if cfg.username != "" {
		req.SetBasicAuth(cfg.username, cfg.password)
	}

	resp, err := client.Do(req)
	lat := time.Since(start)
	ws.latencies = append(ws.latencies, lat)
	if err != nil {
		ws.errs++
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	ws.status[resp.StatusCode]++
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		ws.ok++
		ws.bytes += int64(len(body))
	} else {
		ws.failed++
	}
}

func newHTTPClient(cfg config) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        cfg.concurrency * 2,
		MaxIdleConnsPerHost: cfg.concurrency,
		MaxConnsPerHost:     cfg.concurrency,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}
	if cfg.insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{Transport: tr, Timeout: cfg.reqTimeout}
}

// ---------- 入库排空与验证 ----------

type ingestReport struct {
	drainDur    time.Duration
	uploadedOK  int64
	filesIngest int64 // 本轮新解析文件数（pm_files.parsed 增量）
	rowsIngest  int64 // 本轮新增 pm_metrics 行数
	fileRate    float64
	rowRate     float64
	converged   bool
	prom        pmProm // 增量
	promScraped bool
	finalCounts dbCounts
}

func drainAndVerify(ctx context.Context, cfg config, pool *pgxpool.Pool, base dbCounts, basePM pmProm, uploadedOK int64) *ingestReport {
	fmt.Printf("\n等待入库收敛（最长 %s）…\n", cfg.drainTimeout)
	r := &ingestReport{uploadedOK: uploadedOK}
	start := time.Now()
	deadline := start.Add(cfg.drainTimeout)

	var lastTotal int64 = -1
	stable := 0
	const stableNeed = 4 // 连续 4 次（~4s）无新增即判定收敛
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()

	for {
		cur, err := queryDBCounts(ctx, pool, cfg.snPrefix)
		if err == nil {
			r.finalCounts = cur
			ingested := cur.filesParsed - base.filesParsed
			if ingested >= uploadedOK && uploadedOK > 0 {
				r.converged = true
				break
			}
			if cur.filesTotal == lastTotal {
				stable++
				if stable >= stableNeed {
					break
				}
			} else {
				stable = 0
				lastTotal = cur.filesTotal
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		select {
		case <-tick.C:
		case <-ctx.Done():
		}
	}

	r.drainDur = time.Since(start)
	r.filesIngest = r.finalCounts.filesParsed - base.filesParsed
	r.rowsIngest = r.finalCounts.metricsRows - base.metricsRows
	if s := r.drainDur.Seconds(); s > 0 {
		r.fileRate = float64(r.filesIngest) / s
		r.rowRate = float64(r.rowsIngest) / s
	}

	finalPM := scrapeProm(ctx, cfg.workerMetrics, 5*time.Second)
	r.promScraped = finalPM.scraped && basePM.scraped
	r.prom = finalPM.sub(basePM)
	return r
}

// ---------- 辅助 ----------

func loadTemplates(cfg config) []*generator {
	paths := splitCSV(cfg.templates)
	if len(paths) == 0 {
		fatal("未指定模板（-templates）")
	}
	var gens []*generator
	for _, p := range paths {
		g, err := loadGenerator(p, cfg.oui, cfg.rewriteBodySN)
		if err != nil {
			fatal("加载模板失败: %v", err)
		}
		gens = append(gens, g)
	}
	return gens
}

// buildBuckets 生成 n 个 15min（gran）时间窗，从最近一个已完成窗口向过去回溯，
// 全部落在 [now-n*gran, now] 内，避免落到未来或被 TimescaleDB 压缩的远期 chunk。
func buildBuckets(n int, gran time.Duration) []bucketWindow {
	if n < 1 {
		n = 1
	}
	base := time.Now().Truncate(gran) // 当前窗口起点 = 最近已完成窗口的终点
	wins := make([]bucketWindow, n)
	for b := 0; b < n; b++ {
		end := base.Add(-time.Duration(b) * gran)
		wins[b] = bucketWindow{begin: end.Add(-gran), end: end}
	}
	return wins
}

func mustDB(ctx context.Context, cfg config) *pgxpool.Pool {
	pool, err := connectDB(ctx, cfg.dsn)
	if err != nil {
		fatal("连接数据库失败（%s）: %v", cfg.dsn, err)
	}
	return pool
}

func progress(done *int64, total int, dur time.Duration, deadline time.Time, stop chan struct{}) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	var last int64
	lastTime := time.Now()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			cur := atomic.LoadInt64(done)
			now := time.Now()
			rate := float64(cur-last) / now.Sub(lastTime).Seconds()
			last, lastTime = cur, now
			if dur > 0 {
				remain := time.Until(deadline).Round(time.Second)
				if remain < 0 {
					remain = 0
				}
				fmt.Printf("  进度：已发 %d，瞬时 %.0f/s，剩余 %s\n", cur, rate, remain)
			} else {
				pctDone := 0.0
				if total > 0 {
					pctDone = float64(cur) / float64(total) * 100
				}
				fmt.Printf("  进度：%d/%d (%.1f%%)，瞬时 %.0f/s\n", cur, total, pctDone, rate)
			}
		}
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range bytes.Split([]byte(s), []byte(",")) {
		t := string(bytes.TrimSpace(p))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "错误："+format+"\n", a...)
	os.Exit(1)
}

// ---------- 报告 ----------

func printRunHeader(cfg config, gens []*generator, wins []bucketWindow, totalFiles int) {
	fmt.Println("==================== KPI 上传压测 ====================")
	fmt.Printf("端点      : %s/smallcell/FileUploadService?fileType=%s\n", cfg.baseURL, cfg.fileType)
	names := make([]string, len(gens))
	for i, g := range gens {
		names[i] = g.name
	}
	fmt.Printf("模板      : %v（粒度 %ds）\n", names, gens[0].granSeconds)
	fmt.Printf("并发      : %d\n", cfg.concurrency)
	fmt.Printf("设备数    : %d（SN 前缀 %s，%s/%s）\n", cfg.devices, cfg.snPrefix, cfg.carrier, cfg.tech)
	if cfg.duration > 0 {
		fmt.Printf("时长模式  : %s（循环复用 SN×时间窗）\n", cfg.duration)
	} else {
		fmt.Printf("文件数    : %d（时间窗 %d 个 → 唯一空间 %d）\n", totalFiles, len(wins), cfg.devices*len(wins))
	}
	fmt.Printf("时间窗    : %s … %s\n", fmtTime(wins[len(wins)-1].begin), fmtTime(wins[0].end))
	fmt.Println("=====================================================")
}

func printUploadReport(cfg config, a *aggStat) {
	fmt.Println("\n----- 阶段一：文件存储（ACS → MinIO）-----")
	fmt.Printf("总请求    : %d（成功 %d / 失败 %d / 传输错误 %d）\n", a.total, a.ok, a.failed, a.errs)
	if a.total > 0 {
		fmt.Printf("成功率    : %.2f%%\n", float64(a.ok)/float64(a.total)*100)
	}
	fmt.Printf("墙钟      : %s\n", a.wallTime.Round(time.Millisecond))
	fmt.Printf("吞吐      : %.1f 文件/s，%.2f MB/s\n", a.throughput(), a.mbPerSec())
	fmt.Printf("时延(ms)  : p50 %.1f / p90 %.1f / p95 %.1f / p99 %.1f / max %.1f\n",
		msf(a.pct(50)), msf(a.pct(90)), msf(a.pct(95)), msf(a.pct(99)), msf(a.maxLat()))
	if len(a.status) > 0 {
		fmt.Printf("状态码    : %v\n", a.status)
	}
}

func printIngestReport(cfg config, r *ingestReport) {
	fmt.Println("\n----- 阶段二：KPI 解析入库（worker → pm_metrics）-----")
	conv := "收敛"
	if !r.converged {
		conv = "未完全收敛（达到 -drain 超时或被中断；可调大 -drain）"
	}
	fmt.Printf("排空      : %s（%s）\n", r.drainDur.Round(time.Millisecond), conv)
	fmt.Printf("已上传成功: %d 文件\n", r.uploadedOK)
	fmt.Printf("解析入库  : %d 文件（pm_files.parsed 增量）\n", r.filesIngest)
	fmt.Printf("新增行    : %d 行 pm_metrics（含 counter + KPI）\n", r.rowsIngest)
	fmt.Printf("入库吞吐  : %.1f 文件/s，%.0f 行/s\n", r.fileRate, r.rowRate)
	if r.promScraped {
		fmt.Printf("worker 指标增量: 成功 %.0f / 失败 %.0f / 迟到 %.0f 文件；丢弃 counter %.0f\n",
			r.prom.filesSuccess, r.prom.filesFailed, r.prom.filesLate, r.prom.droppedCounters)
		fmt.Printf("            单文件处理均值 %.1f ms；上报延迟均值 %.1f s\n",
			r.prom.avgProcMillis(), r.prom.avgDelaySec())
	} else {
		fmt.Printf("worker 指标   : %s\n", promHint(cfg.workerMetrics))
	}
	if r.filesIngest < r.uploadedOK {
		fmt.Printf("注意      : 入库文件数 < 上传成功数，可能原因：设备未注入(SN 未命中) / 迟到数据落压缩 chunk / worker 落后或 DLQ。查 worker 日志与上面 worker 指标。\n")
	}
}

type jsonReport struct {
	Mode        string `json:"mode"`
	Concurrency int    `json:"concurrency"`
	Devices     int    `json:"devices"`
	Buckets     int    `json:"buckets"`
	// 上传
	Total      int     `json:"total"`
	OK         int     `json:"ok"`
	Failed     int     `json:"failed"`
	Errors     int     `json:"errors"`
	UploadSec  float64 `json:"upload_seconds"`
	UploadTPS  float64 `json:"upload_files_per_sec"`
	UploadMBps float64 `json:"upload_mb_per_sec"`
	P50ms      float64 `json:"p50_ms"`
	P90ms      float64 `json:"p90_ms"`
	P95ms      float64 `json:"p95_ms"`
	P99ms      float64 `json:"p99_ms"`
	MaxMs      float64 `json:"max_ms"`
	// 入库
	IngestDrainSec  float64 `json:"ingest_drain_seconds,omitempty"`
	IngestFiles     int64   `json:"ingest_files,omitempty"`
	IngestRows      int64   `json:"ingest_rows,omitempty"`
	IngestFileTPS   float64 `json:"ingest_files_per_sec,omitempty"`
	IngestRowTPS    float64 `json:"ingest_rows_per_sec,omitempty"`
	IngestConverged bool    `json:"ingest_converged,omitempty"`
	PromSuccess     float64 `json:"prom_files_success,omitempty"`
	PromFailed      float64 `json:"prom_files_failed,omitempty"`
	PromLate        float64 `json:"prom_files_late,omitempty"`
	PromAvgProcMs   float64 `json:"prom_avg_proc_ms,omitempty"`
	PromAvgDelaySec float64 `json:"prom_avg_delay_sec,omitempty"`
}

func emitJSON(cfg config, a *aggStat, r *ingestReport) {
	jr := jsonReport{
		Mode: cfg.mode, Concurrency: cfg.concurrency, Devices: cfg.devices, Buckets: cfg.buckets,
		Total: a.total, OK: a.ok, Failed: a.failed, Errors: a.errs,
		UploadSec: a.wallTime.Seconds(), UploadTPS: a.throughput(), UploadMBps: a.mbPerSec(),
		P50ms: msf(a.pct(50)), P90ms: msf(a.pct(90)), P95ms: msf(a.pct(95)), P99ms: msf(a.pct(99)), MaxMs: msf(a.maxLat()),
	}
	if r != nil {
		jr.IngestDrainSec = r.drainDur.Seconds()
		jr.IngestFiles = r.filesIngest
		jr.IngestRows = r.rowsIngest
		jr.IngestFileTPS = r.fileRate
		jr.IngestRowTPS = r.rowRate
		jr.IngestConverged = r.converged
		if r.promScraped {
			jr.PromSuccess = r.prom.filesSuccess
			jr.PromFailed = r.prom.filesFailed
			jr.PromLate = r.prom.filesLate
			jr.PromAvgProcMs = r.prom.avgProcMillis()
			jr.PromAvgDelaySec = r.prom.avgDelaySec()
		}
	}
	b, _ := json.MarshalIndent(jr, "", "  ")
	fmt.Println("\n" + string(b))
}
