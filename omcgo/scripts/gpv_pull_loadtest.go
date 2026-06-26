//go:build ignore

// gpv_pull_loadtest.go — GPV pull consumer 吞吐量压测
//
// 使用方式（在 omcgo/ 目录下执行）：
//   go run scripts/gpv_pull_loadtest.go [flags]
//
// 常用示例：
//   # 模拟 10 万设备规模（28 msg/s 持续 60s = 单设备 15min 周期）
//   go run scripts/gpv_pull_loadtest.go -rate 28 -duration 60s
//
//   # 模拟峰值冲击（1000 设备同时开机 inform）
//   go run scripts/gpv_pull_loadtest.go -rate 1000 -duration 30s
//
//   # 全速打满（不限速，看 consumer 最大吞吐）
//   go run scripts/gpv_pull_loadtest.go -rate 0 -duration 20s -total 50000
//
// 工作原理：
//   直接向 NATS JetStream "command.get_parameters.response" 主题注入
//   符合 event.Event 格式的 GPV 响应消息，绕过 ACS 层，精准测量
//   pull consumer（provision-gpv-pull）处理链路的实际吞吐与积压。
//
// 指标输出：
//   - 发布速率、总发布量
//   - NATS consumer num_pending（积压量）
//   - 压测前后 Prometheus omc_eventbus_delivery_total{outcome=ack} 差值（处理量）

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// ----- 事件结构（与 internal/core/event/types.go 保持一致） -----

type Event struct {
	ID        string            `json:"id"`
	Subject   string            `json:"subject"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// GPV response payload（与 acs/handler.go:publishRPCResponseEvent 对齐）
type GPVPayload struct {
	DeviceSN        string           `json:"device_sn"`
	Method          string           `json:"method"`
	CommandKey      string           `json:"command_key,omitempty"`
	ParameterValues []ParameterValue `json:"parameter_values"`
}

type ParameterValue struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
	Type  string `json:"Type"`
}

// ----- 模拟参数包（仿真真实设备 48 条参数） -----

func buildFakeParams(sn string) []ParameterValue {
	base := []struct{ name, typ string }{
		{"Device.DeviceInfo.SoftwareVersion", "string"},
		{"Device.DeviceInfo.HardwareVersion", "string"},
		{"Device.DeviceInfo.UpTime", "unsignedInt"},
		{"Device.DeviceInfo.X_COM_CellStatus", "string"},
		{"Device.FAP.PerfMgmt.Config.1.Enable", "boolean"},
		{"Device.FAP.PerfMgmt.Config.1.PeriodicUploadInterval", "unsignedInt"},
		{"Device.Services.FAPService.1.FAPControl.LTE.OpState", "boolean"},
		{"Device.Services.FAPService.1.FAPControl.LTE.AdminState", "boolean"},
		{"Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus", "boolean"},
		{"Device.Services.FAPService.1.FAPControl.LTE.Gateway.SecGWServer1", "string"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLEARFCNList", "string"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ULEARFCNList", "string"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth", "string"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ULBandwidth", "string"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", "unsignedInt"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", "unsignedInt"},
		{"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID", "string"},
		{"Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.IsPrimary", "boolean"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.IdleMode.Common.T3402", "unsignedInt"},
		{"Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.IdleMode.Common.T3412", "unsignedInt"},
		{"Device.ManagementServer.URL", "string"},
		{"Device.ManagementServer.PeriodicInformEnable", "boolean"},
		{"Device.ManagementServer.PeriodicInformInterval", "unsignedInt"},
		{"Device.ManagementServer.Username", "string"},
		{"Device.IP.Interface.1.IPv4Address.1.IPAddress", "string"},
		{"Device.IP.Interface.1.IPv4Address.1.SubnetMask", "string"},
		{"Device.IP.Interface.1.IPv4Address.1.AddressingType", "string"},
		{"Device.Services.FAPService.1.FAPControl.LTE.Gateway.SecGWServer2", "string"},
		{"Device.Services.FAPService.1.FAPControl.LTE.Gateway.SecGWServer3", "string"},
		{"Device.Services.FAPService.1.FAPControl.NR.OpState", "boolean"},
		{"Device.Services.FAPService.1.FAPControl.NR.AdminState", "boolean"},
		{"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.ARFCNUL", "unsignedInt"},
		{"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.ARFCNDL", "unsignedInt"},
		{"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.PhyCellID", "unsignedInt"},
		{"Device.Services.FAPService.1.CellConfig.NR.RAN.RF.TxPower", "int"},
		{"Device.Services.FAPService.1.CellConfig.NR.RAN.Common.NRCellIdentity", "unsignedInt"},
		{"Device.Services.FAPService.1.CellConfig.NR.EPC.PLMNList.1.PLMNID", "string"},
		{"Device.X_COM_GPS.GPSLongitude", "string"},
		{"Device.X_COM_GPS.GPSLatitude", "string"},
		{"Device.X_COM_GPS.GPSHeight", "string"},
		{"Device.X_COM_GPS.SatelliteNumber", "unsignedInt"},
		{"Device.X_COM_GPS.GPSState", "string"},
		{"Device.Services.FAPService.1.X_COM_AlarmStatus", "string"},
		{"Device.Services.FAPService.1.X_COM_CellLoadLevel", "unsignedInt"},
		{"Device.Services.FAPService.1.X_COM_ConnectedUECount", "unsignedInt"},
		{"Device.Services.FAPService.1.X_COM_TxBandwidth", "unsignedInt"},
		{"Device.X_COM_NetworkInfo.WanIPAddress", "string"},
		{"Device.X_COM_NetworkInfo.WanIPMask", "string"},
	}
	params := make([]ParameterValue, len(base))
	for i, b := range base {
		params[i] = ParameterValue{
			Name:  b.name,
			Value: fmt.Sprintf("testval-%s-%d", sn[len(sn)-4:], i),
			Type:  b.typ,
		}
	}
	return params
}

// ----- 主函数 -----

func main() {
	natsURL := flag.String("nats", "nats://localhost:4222", "NATS server URL")
	metricsURL := flag.String("metrics", "http://localhost:9091/metrics", "app Prometheus metrics URL (port 9091)")
	natsMonURL := flag.String("nats-mon", "http://localhost:8222", "NATS monitoring HTTP URL")
	rateFlag := flag.Float64("rate", 100, "publish rate msgs/s (0 = unlimited)")
	durationStr := flag.String("duration", "30s", "test duration")
	totalFlag := flag.Int64("total", 0, "stop after N messages (0 = use duration)")
	snPrefix := flag.String("sn-prefix", "LOADTEST-GPV", "device SN prefix (real SNs need DB records)")
	fixedSN := flag.String("fixed-sn", "", "use one real device SN for all messages (recommended for full-chain test)")
	deviceCount := flag.Int("devices", 1000, "number of simulated device SNs (messages cycle through these)")
	subject := flag.String("subject", "command.get_parameters.response", "NATS subject to publish to")
	consumerName := flag.String("consumer", "provision-gpv-pull", "durable consumer name to monitor")
	streamName := flag.String("stream", "COMMAND", "NATS stream name")
	flag.Parse()

	dur, err := time.ParseDuration(*durationStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid duration: %v\n", err)
		os.Exit(1)
	}

	// 连接 NATS
	nc, err := nats.Connect(*natsURL,
		nats.Name("gpv-pull-loadtest"),
		nats.Timeout(5*time.Second),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NATS connect failed: %v\n", err)
		os.Exit(1)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		fmt.Fprintf(os.Stderr, "JetStream init failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[loadtest] 连接 NATS OK: %s\n", *natsURL)
	fmt.Printf("[loadtest] 目标: subject=%s  consumer=%s\n", *subject, *consumerName)
	fmt.Printf("[loadtest] 参数: rate=%.0f msg/s  duration=%s  devices=%d\n\n",
		*rateFlag, dur, *deviceCount)

	// 采集 Prometheus 基线
	deliveredBefore := fetchPrometheusCounter(*metricsURL, "omc_eventbus_delivery_total",
		map[string]string{"outcome": "ack", "subject": *subject})
	pendingBefore := fetchNATSConsumerPending(*natsMonURL, *streamName, *consumerName)
	fmt.Printf("[baseline] omc_eventbus_delivery_total{outcome=ack,subject=%s} = %d\n", *subject, deliveredBefore)
	fmt.Printf("[baseline] consumer num_pending = %d\n\n", pendingBefore)

	// 预生成 SN 列表。若提供 fixed-sn，则所有消息使用同一真实 SN，确保命中
	// device_lookup + translator + BatchUpsert 的完整链路。
	sns := make([]string, 0, *deviceCount)
	if strings.TrimSpace(*fixedSN) != "" {
		sns = append(sns, strings.TrimSpace(*fixedSN))
	} else {
		sns = make([]string, *deviceCount)
		for i := range sns {
			sns[i] = fmt.Sprintf("%s-%06d", *snPrefix, i)
		}
	}

	// 信号处理
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// 压测主循环
	var published atomic.Int64
	var errors atomic.Int64
	startTime := time.Now()

	var ticker *time.Ticker
	if *rateFlag > 0 {
		interval := time.Duration(float64(time.Second) / *rateFlag)
		ticker = time.NewTicker(interval)
		defer ticker.Stop()
	}

	fmt.Println("[loadtest] 开始发布消息...")

	idx := 0
	for {
		select {
		case <-ctx.Done():
			goto done
		default:
		}

		if *totalFlag > 0 && published.Load() >= *totalFlag {
			goto done
		}

		if ticker != nil {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				goto done
			}
		}

		sn := sns[idx%len(sns)]
		idx++

		payload := GPVPayload{
			DeviceSN:        sn,
			Method:          "GetParameterValuesResponse",
			CommandKey:      fmt.Sprintf("sync-gpv-%s", uuid.New().String()[:8]),
			ParameterValues: buildFakeParams(sn),
		}
		payloadBytes, _ := json.Marshal(payload)
		evt := Event{
			ID:        uuid.New().String(),
			Subject:   *subject,
			Payload:   payloadBytes,
			Timestamp: time.Now(),
		}
		evtBytes, _ := json.Marshal(evt)

		if _, err := js.Publish(*subject, evtBytes); err != nil {
			errors.Add(1)
		} else {
			published.Add(1)
		}
	}

done:
	elapsed := time.Since(startTime)
	totalPub := published.Load()
	totalErr := errors.Load()
	actualRate := float64(totalPub) / elapsed.Seconds()

	fmt.Printf("\n[loadtest] 发布完成\n")
	fmt.Printf("  发布总量:  %d msg\n", totalPub)
	fmt.Printf("  错误总量:  %d\n", totalErr)
	fmt.Printf("  实际速率:  %.1f msg/s\n", actualRate)
	fmt.Printf("  耗时:      %s\n\n", elapsed.Round(time.Millisecond))

	// 等待 consumer 处理（最多 10s）
	fmt.Println("[loadtest] 等待 consumer 消化积压（最多 10s）...")
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		p := fetchNATSConsumerPending(*natsMonURL, *streamName, *consumerName)
		fmt.Printf("\r  consumer num_pending = %-8d", p)
		if p == 0 {
			fmt.Println()
			break
		}
	}
	fmt.Println()

	// 采集 Prometheus 结束值
	deliveredAfter := fetchPrometheusCounter(*metricsURL, "omc_eventbus_delivery_total",
		map[string]string{"outcome": "ack", "subject": *subject})
	pendingAfter := fetchNATSConsumerPending(*natsMonURL, *streamName, *consumerName)

	processed := deliveredAfter - deliveredBefore
	backlogDelta := pendingAfter - pendingBefore
	consumeEstimate := totalPub + (pendingBefore - pendingAfter)
	consumeRate := float64(consumeEstimate) / elapsed.Seconds()
	fmt.Printf("[result] omc_eventbus_delivery_total ack delta = %d (已处理)\n", processed)
	fmt.Printf("[result] consumer num_pending (最终) = %d (剩余积压)\n", pendingAfter)
	fmt.Printf("[result] backlog delta = %+d (正数=积压增加, 负数=积压回落)\n", backlogDelta)
	fmt.Printf("[result] estimated consume rate = %.1f msg/s\n", consumeRate)
	if totalPub > 0 {
		fmt.Printf("[result] 处理率 = %.1f%%\n", float64(processed)*100/float64(totalPub))
	}

	// 规模参考换算
	fmt.Printf("\n[规模参考]\n")
	fmt.Printf("  本次实际速率 %.1f msg/s\n", actualRate)
	fmt.Printf("  10 万设备 @ 15min 周期 → 稳态 %.0f msg/s  (本机倍率 ×%.1f)\n",
		100000.0/900, actualRate/(100000.0/900))
	fmt.Printf("  10 万设备冲击峰值（全部同时 Inform）→ ~5000 msg/s\n")
}

// ----- 辅助函数 -----

// fetchPrometheusCounter 从 /metrics 读取指定标签的 counter 值（简单文本解析）。
func fetchPrometheusCounter(metricsURL, metricName string, labels map[string]string) int64 {
	resp, err := http.Get(metricsURL)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	for _, line := range strings.Split(string(body), "\n") {
		if !strings.HasPrefix(line, metricName+"{") {
			continue
		}
		// 检查所有标签是否匹配
		match := true
		for k, v := range labels {
			if !strings.Contains(line, k+`="`+v+`"`) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		// 取最后的数值
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			v, err := strconv.ParseFloat(parts[len(parts)-1], 64)
			if err == nil {
				return int64(v)
			}
		}
	}
	return 0
}

// fetchNATSConsumerPending 通过 NATS HTTP 监控接口查询消费者积压量。
// /jsz?consumers=true 返回结构：{streams: [{config:{name}, consumer_detail:[{name, num_pending}]}]}
func fetchNATSConsumerPending(monURL, stream, consumer string) int64 {
	type consumerInfo struct {
		Name       string `json:"name"`
		NumPending int64  `json:"num_pending"`
	}
	type streamInfo struct {
		Name           string         `json:"name"`
		ConsumerDetail []consumerInfo `json:"consumer_detail"`
	}
	type accountDetail struct {
		StreamDetail []streamInfo `json:"stream_detail"`
	}
	type jszResp struct {
		AccountDetails []accountDetail `json:"account_details"`
	}

	url := fmt.Sprintf("%s/jsz?consumers=true&accounts=true", monURL)
	resp, err := http.Get(url)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var jsz jszResp
	if err := json.Unmarshal(body, &jsz); err != nil {
		return -1
	}
	for _, acc := range jsz.AccountDetails {
		for _, s := range acc.StreamDetail {
			if s.Name != stream {
				continue
			}
			for _, c := range s.ConsumerDetail {
				if c.Name == consumer {
					return c.NumPending
				}
			}
			return -1
		}
	}
	return -1

	// 旧结构兼容逻辑（保留在此注释供未来回滚）：
	// for _, s := range jsz.Streams {
	// 	if s.Config.Name != stream {
	// 		continue
	// 	}
	// 	for _, c := range s.ConsumerDetail {
	// 		name := c.Name
	// 		if name == "" {
	// 			name = c.Config.Name
	// 		}
	// 		if name == consumer {
	// 			return c.NumPending
	// 		}
	// 	}
	// }
	// return -1
}
