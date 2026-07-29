// Command kpiperf 是 KPI(PM) 文件上传压测工具：模拟大量小基站向 ACS 直传 3GPP 32.435
// 性能文件，压测「文件存储（ACS→MinIO）」与「KPI 解析入库（worker→pm_metrics + KPI 反算）」。
//
// 支持 4G/5G/GSM 三制式同时压、并发可在三者间拆分；默认用「内置真机样本」模板法（仅改时间+SN，
// 不动内容，保真上报真机文件的大小与结构）。-synth 可切回「指标库驱动」合成法，按各平台
// （BLQ/BaiBNQ/BSC）内置指标库的全部源 counter 生成文件，覆盖所有内置指标。
//
// 链路（直传，不走 AutonomousTransferComplete）：
//
//	kpiperf ──PUT /smallcell/FileUploadService?fileType=4&sn=&filename=──▶ ACS
//	  ACS 落 MinIO(pm-files) + 发 pm.file.received → worker 按 SN 查 devices(必须预注入且带
//	  productClass) → 解析 → 写 pm_metrics(counter) → 路由平台反算 KPI(metric_type=kpi)
//
// 关键前置：测试设备必须**在上传前**注入并带正确 productClass（run 模式默认 -seed 完成）。
// 否则 worker 路由负缓存会让设备永远算不出 KPI，且白名单为空放过重名 counter 撞自然键。
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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type config struct {
	mode          string
	baseURL       string
	rats          string
	mix           string
	templates     string
	concurrency   int
	devices       int
	files         int
	buckets       int
	snPrefix      string
	oui           string
	carrier       string
	fileType      string
	username      string
	password      string
	dsn           string
	tsdbDSN       string
	natsURL       string
	noDB          bool
	seed          bool
	cleanupAfter  bool
	workerMetrics string
	drainTimeout  time.Duration
	reqTimeout    time.Duration
	jsonOut       bool
	rewriteBodySN bool
	insecure      bool
	synth         bool
}

type bucketWindow struct {
	begin time.Time
	end   time.Time
}

// ratRun 是单个制式的一轮压测计划 + 结果。
type ratRun struct {
	prof    ratProfile
	gen     fileGen
	genKind string
	devices int
	files   int
	conc    int
	stats   *aggStat
	baseCnt int64
	baseKpi int64
	addCnt  int64
	addKpi  int64
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
	case "purge":
		runPurge(cfg)
	case "run":
		runLoad(ctx, cfg)
	default:
		fmt.Fprintf(os.Stderr, "未知 -mode=%q（可选 run|seed|cleanup|purge）\n", cfg.mode)
		os.Exit(2)
	}
}

func parseFlags() config {
	var c config
	flag.StringVar(&c.mode, "mode", "run", "run（注入+上传+验证）| seed（仅注入）| cleanup（清库+清 NATS）| purge（仅清 NATS PM 流）")
	flag.StringVar(&c.baseURL, "url", "http://localhost:7557", "ACS 上传端点 base URL（直连 ACS :7557 或经 nginx :8080）")
	flag.StringVar(&c.rats, "rats", "lte,nr,gsm", "压测制式（逗号分隔）：lte|nr|gsm，并发/设备数在其间均分")
	flag.StringVar(&c.mix, "mix", "", "按制式显式指定设备数（真实占比，GSM 通常很少）：如 \"lte=300,nr=270,gsm=30\"。设置后覆盖 -rats/-devices，制式与顺序取自此处，每制式文件=设备×buckets，并发按设备占比拆分")
	flag.StringVar(&c.templates, "templates", "", "可选：每个 RAT 一个外部真机模板 XML（数量须等于 RAT 数）；留空=用内置真机样本（-synth 可切指标库合成法）")
	flag.IntVar(&c.concurrency, "concurrency", 100, "总并发上传数（在飞请求上限），按 -rats 拆分；建议 200–10000")
	flag.IntVar(&c.devices, "devices", 9999, "注入的设备总数，按 -rats 拆分（默认每制式 3333）")
	flag.IntVar(&c.files, "files", 0, "总上传文件数；0=各制式 devices×buckets（每设备每时间窗一份）")
	flag.IntVar(&c.buckets, "buckets", 1, "不同 15min KPI 时间窗个数（向最近完成窗回溯，扩大唯一行空间）")
	flag.StringVar(&c.snPrefix, "sn-prefix", "KPILT", "测试设备 SN 前缀（清理按该前缀 LIKE）")
	flag.StringVar(&c.oui, "oui", "48BF74", "设备 OUI（6 hex）")
	flag.StringVar(&c.carrier, "carrier", "cmcc", "运营商：cmcc|ctcc|cucc|other（devices 分区键）")
	flag.StringVar(&c.fileType, "filetype", "4", "上传 fileType（PM=4）")
	flag.StringVar(&c.username, "username", "", "上传端点 Basic Auth 用户名（默认无鉴权）")
	flag.StringVar(&c.password, "password", "", "上传端点 Basic Auth 密码")
	flag.StringVar(&c.dsn, "db", "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable", "主库 PostgreSQL DSN（devices 注入 / dead_letters 清理）")
	flag.StringVar(&c.tsdbDSN, "tsdb", "", "时序库 DSN（KPI/时序库物理分离后 pm_metrics/pm_files 在此；留空=与 -db 同库，兼容未分离部署）")
	flag.StringVar(&c.natsURL, "nats", "nats://localhost:4222", "NATS URL（cleanup/purge 清 PM 流用）")
	flag.BoolVar(&c.noDB, "no-db", false, "跳过注入与入库验证，仅压文件存储")
	flag.BoolVar(&c.seed, "seed", true, "run 模式上传前注入设备（带 productClass）")
	flag.BoolVar(&c.cleanupAfter, "cleanup-after", false, "run 模式结束后一键清理")
	flag.StringVar(&c.workerMetrics, "worker-metrics", "http://localhost:9092/metrics", "worker Prometheus /metrics")
	flag.DurationVar(&c.drainTimeout, "drain", 120*time.Second, "上传后等待入库收敛的最长时长")
	flag.DurationVar(&c.reqTimeout, "timeout", 30*time.Second, "单次上传 HTTP 超时")
	flag.BoolVar(&c.jsonOut, "json", false, "JSON 输出")
	flag.BoolVar(&c.rewriteBodySN, "rewrite-body-sn", true, "模板模式下改写 body 内 localDn SN（合成法无需）")
	flag.BoolVar(&c.insecure, "insecure", false, "https 时跳过 TLS 校验")
	flag.BoolVar(&c.synth, "synth", false, "强制用指标库合成法（覆盖全部内置指标）替代内置真机样本；默认用真机样本")
	flag.Parse()
	return c
}

// ---------- seed / cleanup / purge ----------

func runSeed(ctx context.Context, cfg config) {
	pool := mustDB(ctx, cfg)
	defer pool.Close()
	for _, p := range resolveRatPlan(cfg) {
		prof := builtinProfiles[p.rat] // resolveRatPlan/parseMix 已校验 RAT 合法
		fmt.Printf("注入 %s 设备 %d 个（productClass=%s, %s/%s）…\n", p.rat, p.devices, prof.productClass, cfg.carrier, prof.tech)
		ins, err := seedDevices(ctx, pool, cfg.snPrefix, p.rat, p.devices, cfg.oui, cfg.carrier, prof.tech, prof.productClass)
		if err != nil {
			fatal("注入 %s 失败: %v", p.rat, err)
		}
		fmt.Printf("  新增 %d 个\n", ins)
	}
}

func runCleanup(ctx context.Context, cfg config) {
	pool := mustDB(ctx, cfg)
	defer pool.Close()
	tsPool := mustTSDB(ctx, cfg, pool)
	if tsPool != pool {
		defer tsPool.Close()
	}
	fmt.Printf("清理测试数据（SN 前缀 %s）…\n", cfg.snPrefix)
	r, err := cleanupAll(ctx, pool, tsPool, cfg.snPrefix)
	if err != nil {
		fatal("清理失败: %v", err)
	}
	fmt.Printf("  删除 pm_metrics %d、pm_files %d、devices %d、dead_letters %d 行\n", r.metrics, r.files, r.devices, r.dlq)
	if n, perr := purgePMStream(cfg.natsURL); perr != nil {
		fmt.Fprintf(os.Stderr, "  清 NATS PM 流失败（可忽略）: %v\n", perr)
	} else {
		fmt.Printf("  清空 NATS PM 流 %d 条消息\n", n)
	}
}

func runPurge(cfg config) {
	n, err := purgePMStream(cfg.natsURL)
	if err != nil {
		fatal("清 NATS PM 流失败: %v", err)
	}
	fmt.Printf("已清空 NATS PM 流 %d 条消息\n", n)
}

// ---------- run ----------

func runLoad(ctx context.Context, cfg config) {
	plans := resolveRatPlan(cfg)
	if len(plans) == 0 {
		fatal("未指定 -rats / -mix")
	}
	runs := buildRatRuns(cfg, plans)
	wins := buildBuckets(cfg.buckets, time.Duration(runs[0].gen.granSecondsVal())*time.Second)

	printRunHeader(cfg, runs, wins)

	var pool, tsPool *pgxpool.Pool
	var baseAll dbCounts
	if !cfg.noDB {
		pool = mustDB(ctx, cfg)
		defer pool.Close()
		tsPool = mustTSDB(ctx, cfg, pool) // pm_metrics/pm_files 走时序库（分离后）
		if tsPool != pool {
			defer tsPool.Close()
		}
		if cfg.seed {
			for _, r := range runs {
				ins, err := seedDevices(ctx, pool, cfg.snPrefix, r.prof.rat, r.devices, cfg.oui, cfg.carrier, r.prof.tech, r.prof.productClass)
				if err != nil {
					fatal("注入 %s 设备失败: %v", r.prof.rat, err)
				}
				fmt.Printf("注入 %s 设备 %d（新增 %d，productClass=%s）\n", r.prof.rat, r.devices, ins, r.prof.productClass)
			}
		}
		var err error
		if baseAll, err = queryDBCounts(ctx, tsPool, cfg.snPrefix); err != nil {
			fatal("读取基线失败: %v", err)
		}
		for _, r := range runs {
			r.baseCnt, r.baseKpi = queryKPIByRat(ctx, tsPool, cfg.snPrefix, r.prof.rat)
		}
	} else {
		fmt.Println("-no-db：跳过注入与入库验证，仅压文件存储")
	}
	basePM := scrapeProm(ctx, cfg.workerMetrics, 5*time.Second)

	wall := runUploadStormMulti(ctx, cfg, runs, wins)
	printUploadReport(cfg, runs, wall)

	var ing *ingestReport
	if !cfg.noDB && ctx.Err() == nil {
		ing = drainAndVerify(ctx, cfg, tsPool, runs, baseAll, basePM)
		printIngestReport(cfg, runs, ing)
	}
	if cfg.jsonOut {
		emitJSON(cfg, runs, wall, ing)
	}
	if cfg.cleanupAfter && pool != nil {
		fmt.Println("\n按 -cleanup-after 清理…")
		if r, err := cleanupAll(context.Background(), pool, tsPool, cfg.snPrefix); err == nil {
			fmt.Printf("  删除 pm_metrics %d、pm_files %d、devices %d 行\n", r.metrics, r.files, r.devices)
		}
		if n, err := purgePMStream(cfg.natsURL); err == nil {
			fmt.Printf("  清空 NATS PM 流 %d 条\n", n)
		}
	}
}

func buildRatRuns(cfg config, plans []ratPlan) []*ratRun {
	n := len(plans)
	// -files（总数，可选）按各制式设备占比拆分；未设则每制式 = 设备×buckets。
	var fileSplit []int
	if cfg.files > 0 {
		totalDev := 0
		for _, p := range plans {
			totalDev += p.devices
		}
		fileSplit = splitByWeight(cfg.files, plans, totalDev)
	}
	var tmpls []string
	if cfg.templates != "" {
		tmpls = splitCSV(cfg.templates)
		if len(tmpls) != n {
			fatal("模板模式：-templates 数量(%d) 必须等于制式数量(%d)", len(tmpls), n)
		}
	}
	runs := make([]*ratRun, n)
	for i, p := range plans {
		prof := builtinProfiles[p.rat] // resolveRatPlan/parseMix 已校验 RAT 合法
		var g fileGen
		var err error
		switch {
		case len(tmpls) > 0: // 显式 -templates 覆盖全部 RAT
			g, err = loadGenerator(tmpls[i], cfg.oui, cfg.rewriteBodySN)
		case cfg.synth: // -synth 强制指标库合成（覆盖全部内置指标）
			g, err = loadLibGen(prof, prof.indicatorRel, cfg.oui, 900)
		case prof.templateRel != "": // 默认：该 RAT 内置真机样本（仅改时间+SN，不动内容）
			g, err = loadGenerator(prof.templateRel, cfg.oui, cfg.rewriteBodySN)
		default: // 无内置样本时回退合成
			g, err = loadLibGen(prof, prof.indicatorRel, cfg.oui, 900)
		}
		if err != nil {
			fatal("构建 %s 生成器失败: %v", p.rat, err)
		}
		genKind := "指标库合成"
		if _, isTmpl := g.(*generator); isTmpl {
			genKind = "真机样本:" + g.label()
		}
		dev := p.devices
		if dev < 1 {
			dev = 1
		}
		conc := p.conc
		if conc < 1 {
			conc = 1
		}
		files := dev * cfg.buckets
		if fileSplit != nil {
			files = fileSplit[i]
		}
		runs[i] = &ratRun{prof: prof, gen: g, genKind: genKind, devices: dev, conc: conc, files: files}
	}
	return runs
}

// runUploadStormMulti 并发跑所有 RAT 的 worker 池（总在飞 = Σ各 RAT 并发 = cfg.concurrency）。
func runUploadStormMulti(ctx context.Context, cfg config, runs []*ratRun, wins []bucketWindow) time.Duration {
	client := newHTTPClient(cfg)
	endpoint := cfg.baseURL + "/smallcell/FileUploadService"
	total := 0
	for _, r := range runs {
		total += r.files
	}
	var done int64
	progStop := make(chan struct{})
	go progress(&done, total, progStop)

	start := time.Now()
	var outer sync.WaitGroup
	for _, run := range runs {
		outer.Add(1)
		go func(run *ratRun) {
			defer outer.Done()
			run.stats = runRatPool(ctx, cfg, client, endpoint, run, wins, &done)
		}(run)
	}
	outer.Wait()
	close(progStop)
	return time.Since(start)
}

func runRatPool(ctx context.Context, cfg config, client *http.Client, endpoint string, run *ratRun, wins []bucketWindow, done *int64) *aggStat {
	var next int64 = -1
	stats := make([]*workerStat, run.conc)
	poolStart := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < run.conc; w++ {
		ws := newWorkerStat(run.files/max(run.conc, 1) + 16)
		stats[w] = ws
		wg.Add(1)
		go func(ws *workerStat) {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				i := atomic.AddInt64(&next, 1)
				if i >= int64(run.files) {
					return
				}
				seq := int(i)
				devIdx := seq % run.devices
				bkt := wins[(seq/run.devices)%len(wins)]
				sn := deviceSN(cfg.snPrefix, run.prof.rat, devIdx+1)
				fname, body := run.gen.generate(sn, bkt.begin, bkt.end, seq)
				doUpload(ctx, client, endpoint, cfg, sn, fname, body, ws)
				atomic.AddInt64(done, 1)
			}
		}(ws)
	}
	wg.Wait()
	return mergeStats(stats, time.Since(poolStart))
}

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
	filesIngest int64
	rowsIngest  int64
	kpiIngest   int64
	fileRate    float64
	rowRate     float64
	converged   bool
	prom        pmProm
	promScraped bool
}

func drainAndVerify(ctx context.Context, cfg config, pool *pgxpool.Pool, runs []*ratRun, base dbCounts, basePM pmProm) *ingestReport {
	var uploadedOK int64
	for _, r := range runs {
		if r.stats != nil {
			uploadedOK += int64(r.stats.ok)
		}
	}
	fmt.Printf("\n等待入库收敛（最长 %s）…\n", cfg.drainTimeout)
	r := &ingestReport{uploadedOK: uploadedOK}
	start := time.Now()
	deadline := start.Add(cfg.drainTimeout)
	var lastTotal int64 = -1
	stable := 0
	const stableNeed = 4
	var final dbCounts
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for {
		cur, err := queryDBCounts(ctx, pool, cfg.snPrefix)
		if err == nil {
			final = cur
			if cur.filesParsed-base.filesParsed >= uploadedOK && uploadedOK > 0 {
				r.converged = true
				break
			}
			if cur.filesTotal == lastTotal {
				if stable++; stable >= stableNeed {
					break
				}
			} else {
				stable, lastTotal = 0, cur.filesTotal
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
	r.filesIngest = final.filesParsed - base.filesParsed
	r.rowsIngest = final.metricsRows - base.metricsRows
	r.kpiIngest = final.kpiRows - base.kpiRows
	if s := r.drainDur.Seconds(); s > 0 {
		r.fileRate = float64(r.filesIngest) / s
		r.rowRate = float64(r.rowsIngest) / s
	}
	for _, rr := range runs {
		c, k := queryKPIByRat(ctx, pool, cfg.snPrefix, rr.prof.rat)
		rr.addCnt, rr.addKpi = c-rr.baseCnt, k-rr.baseKpi
	}
	finalPM := scrapeProm(ctx, cfg.workerMetrics, 5*time.Second)
	r.promScraped = finalPM.scraped && basePM.scraped
	r.prom = finalPM.sub(basePM)
	return r
}

// ---------- 辅助 ----------

func buildBuckets(n int, gran time.Duration) []bucketWindow {
	if n < 1 {
		n = 1
	}
	if gran <= 0 {
		gran = 15 * time.Minute
	}
	base := time.Now().Truncate(gran)
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

// mustTSDB 返回承载 pm_metrics/pm_files 的时序库连接池。KPI/时序库物理分离后这些表在
// 独立实例（-tsdb）；未设 -tsdb（或与 -db 同 DSN）则复用主库连接（兼容未分离部署）。
// 返回值若 != mainPool，调用方需负责 Close。
func mustTSDB(ctx context.Context, cfg config, mainPool *pgxpool.Pool) *pgxpool.Pool {
	if cfg.tsdbDSN == "" || cfg.tsdbDSN == cfg.dsn {
		return mainPool
	}
	pool, err := connectDB(ctx, cfg.tsdbDSN)
	if err != nil {
		fatal("连接时序库失败（%s）: %v", cfg.tsdbDSN, err)
	}
	return pool
}

func progress(done *int64, total int, stop chan struct{}) {
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
			pct := 0.0
			if total > 0 {
				pct = float64(cur) / float64(total) * 100
			}
			fmt.Printf("  进度：%d/%d (%.1f%%)，瞬时 %.0f/s\n", cur, total, pct, rate)
		}
	}
}

// splitN 把 total 平均拆成 n 份，余数依次加到前面的份。
func splitN(total, n int) []int {
	if n <= 0 {
		return nil
	}
	out := make([]int, n)
	base, rem := total/n, total%n
	for i := range out {
		out[i] = base
		if i < rem {
			out[i]++
		}
	}
	return out
}

// ratPlan 是一个制式的设备/并发计划，由 resolveRatPlan 统一产出，供 run / seed 共用，
// 保证两条路径对设备数的理解一致。
type ratPlan struct {
	rat     string
	devices int
	conc    int
}

// resolveRatPlan 解析压测的制式计划：
//   - 设了 -mix（如 "lte=300,nr=270,gsm=30"）：制式与顺序取自 mix，设备数按 mix 显式给定
//     （真实占比，GSM 通常远少于 4G/5G）；总并发 cfg.concurrency 按各制式设备占比拆分。
//   - 未设 -mix：回退旧行为——制式取 -rats，设备数/并发数在制式间均分（splitN）。
func resolveRatPlan(cfg config) []ratPlan {
	if mix := parseMix(cfg.mix); len(mix) > 0 {
		totalDev := 0
		for _, m := range mix {
			totalDev += m.devices
		}
		concs := splitByWeight(cfg.concurrency, mix, totalDev)
		out := make([]ratPlan, len(mix))
		for i, m := range mix {
			out[i] = ratPlan{rat: m.rat, devices: m.devices, conc: concs[i]}
		}
		return out
	}
	rats := splitCSV(cfg.rats)
	devs := splitN(cfg.devices, len(rats))
	concs := splitN(cfg.concurrency, len(rats))
	out := make([]ratPlan, len(rats))
	for i, r := range rats {
		out[i] = ratPlan{rat: r, devices: devs[i], conc: concs[i]}
	}
	return out
}

// parseMix 解析 -mix "lte=300,nr=270,gsm=30" → 有序 [(rat,devices)]。空串 → nil（未启用）。
func parseMix(s string) []ratPlan {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []ratPlan
	for _, p := range splitCSV(s) {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			fatal("非法 -mix 项 %q，应形如 lte=300", p)
		}
		rat := strings.TrimSpace(kv[0])
		if _, ok := builtinProfiles[rat]; !ok {
			fatal("-mix 未知 RAT %q（可选 lte|nr|gsm）", rat)
		}
		n, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil || n < 0 {
			fatal("-mix 项 %q 设备数非法", p)
		}
		out = append(out, ratPlan{rat: rat, devices: n})
	}
	return out
}

// splitByWeight 把 total 按各项 devices 占比拆分（余数给最大权重项），用于按设备占比分配并发。
func splitByWeight(total int, items []ratPlan, weightSum int) []int {
	out := make([]int, len(items))
	if weightSum <= 0 {
		return splitN(total, len(items))
	}
	assigned, maxIdx, maxW := 0, 0, -1
	for i, m := range items {
		out[i] = total * m.devices / weightSum
		assigned += out[i]
		if m.devices > maxW {
			maxW, maxIdx = m.devices, i
		}
	}
	if rem := total - assigned; rem > 0 {
		out[maxIdx] += rem
	}
	return out
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range bytes.Split([]byte(s), []byte(",")) {
		if t := string(bytes.TrimSpace(p)); t != "" {
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

func printRunHeader(cfg config, runs []*ratRun, wins []bucketWindow) {
	fmt.Println("==================== KPI 上传压测 ====================")
	fmt.Printf("端点    : %s/smallcell/FileUploadService?fileType=%s\n", cfg.baseURL, cfg.fileType)
	mode := "内置真机样本（仅改时间+SN，不动内容）"
	switch {
	case cfg.templates != "":
		mode = "外部真机模板（-templates）"
	case cfg.synth:
		mode = "指标库合成（-synth，覆盖全部内置指标）"
	}
	fmt.Printf("生成方式: %s\n", mode)
	fmt.Printf("总并发  : %d   设备总数: %d   时间窗: %d 个\n", cfg.concurrency, cfg.devices, cfg.buckets)
	for _, r := range runs {
		fmt.Printf("  [%-3s] 平台 %-8s 并发 %-5d 设备 %-6d 文件 %-7d 行/文件 %-6d %s\n",
			r.prof.rat, r.prof.platform, r.conc, r.devices, r.files, r.gen.counterCount(), r.genKind)
	}
	fmt.Printf("时间窗  : %s … %s\n", fmtTime(wins[len(wins)-1].begin), fmtTime(wins[0].end))
	fmt.Println("=====================================================")
}

func printUploadReport(cfg config, runs []*ratRun, wall time.Duration) {
	fmt.Println("\n----- 阶段一：文件存储（ACS → MinIO）-----")
	var tot, ok, fail, errs int
	var bytesUp int64
	for _, r := range runs {
		a := r.stats
		if a == nil {
			continue
		}
		tot += a.total
		ok += a.ok
		fail += a.failed
		errs += a.errs
		bytesUp += a.bytes
		fmt.Printf("  [%-3s] %d 文件 成功%d/失败%d/错误%d  %.0f/s  p50 %.0fms p99 %.0fms\n",
			r.prof.rat, a.total, a.ok, a.failed, a.errs, a.throughput(), msf(a.pct(50)), msf(a.pct(99)))
	}
	fmt.Printf("  合计  : %d 文件（成功 %d / 失败 %d / 错误 %d）墙钟 %s\n", tot, ok, fail, errs, wall.Round(time.Millisecond))
	if s := wall.Seconds(); s > 0 {
		fmt.Printf("  吞吐  : %.1f 文件/s，%.2f MB/s\n", float64(tot)/s, float64(bytesUp)/1024/1024/s)
	}
}

func printIngestReport(cfg config, runs []*ratRun, r *ingestReport) {
	fmt.Println("\n----- 阶段二：KPI 解析入库（worker → pm_metrics + KPI 反算）-----")
	conv := "收敛"
	if !r.converged {
		conv = "未完全收敛（达 -drain 超时/被中断，可调大 -drain）"
	}
	fmt.Printf("排空      : %s（%s）\n", r.drainDur.Round(time.Millisecond), conv)
	fmt.Printf("上传成功  : %d 文件 → 解析入库 %d 文件\n", r.uploadedOK, r.filesIngest)
	fmt.Printf("新增行    : %d 行 pm_metrics（counter %d + KPI %d）\n", r.rowsIngest, r.rowsIngest-r.kpiIngest, r.kpiIngest)
	fmt.Printf("入库吞吐  : %.1f 文件/s，%.0f 行/s\n", r.fileRate, r.rowRate)
	for _, rr := range runs {
		fmt.Printf("  [%-3s] 平台 %-8s 新增 counter %d，KPI %d\n", rr.prof.rat, rr.prof.platform, rr.addCnt, rr.addKpi)
	}
	if r.promScraped {
		fmt.Printf("worker 指标增量: 成功 %.0f / 失败 %.0f / 迟到 %.0f 文件；白名单未命中值 %.0f；已知但禁用值 %.0f；单文件均 %.1fms；上报延迟均 %.1fs\n",
			r.prom.filesSuccess, r.prom.filesFailed, r.prom.filesLate, r.prom.whitelistMiss, r.prom.knownDisabled, r.prom.avgProcMillis(), r.prom.avgDelaySec())
	} else {
		fmt.Printf("worker 指标   : %s\n", promHint(cfg.workerMetrics))
	}
	if r.filesIngest < r.uploadedOK {
		fmt.Println("注意      : 入库文件数 < 上传成功数 → 查 worker 日志/指标（设备未注入 / 迟到压缩 chunk / 落后 / DLQ）")
	}
	if r.kpiIngest == 0 && r.rowsIngest > 0 {
		fmt.Println("注意      : counter 入库但 KPI=0 → 设备 productClass 未路由到平台（检查 -seed 是否在上传前注入并带 productClass）")
	}
}

func emitJSON(cfg config, runs []*ratRun, wall time.Duration, r *ingestReport) {
	type ratJSON struct {
		RAT      string  `json:"rat"`
		Platform string  `json:"platform"`
		Conc     int     `json:"concurrency"`
		Devices  int     `json:"devices"`
		Files    int     `json:"files"`
		OK       int     `json:"upload_ok"`
		Failed   int     `json:"upload_failed"`
		TPS      float64 `json:"upload_files_per_sec"`
		P99ms    float64 `json:"upload_p99_ms"`
		AddCnt   int64   `json:"ingest_counter_rows"`
		AddKpi   int64   `json:"ingest_kpi_rows"`
	}
	out := map[string]any{
		"concurrency": cfg.concurrency, "devices": cfg.devices, "buckets": cfg.buckets,
		"upload_wall_seconds": wall.Seconds(),
	}
	var rats []ratJSON
	for _, rr := range runs {
		a := rr.stats
		rj := ratJSON{RAT: rr.prof.rat, Platform: rr.prof.platform, Conc: rr.conc, Devices: rr.devices, Files: rr.files, AddCnt: rr.addCnt, AddKpi: rr.addKpi}
		if a != nil {
			rj.OK, rj.Failed, rj.TPS, rj.P99ms = a.ok, a.failed, a.throughput(), msf(a.pct(99))
		}
		rats = append(rats, rj)
	}
	out["rats"] = rats
	if r != nil {
		out["ingest"] = map[string]any{
			"drain_seconds": r.drainDur.Seconds(), "files_ingested": r.filesIngest,
			"rows_ingested": r.rowsIngest, "kpi_rows": r.kpiIngest,
			"files_per_sec": r.fileRate, "rows_per_sec": r.rowRate, "converged": r.converged,
		}
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println("\n" + string(b))
}
