// Package acs provides TR069/CWMP protocol handling.
package acs

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// InformPeriodLookup 是心跳周期配置读取的唯一外部依赖：按 (category, key) 读
// sys_configs 行的 value 列。消费者驱动小接口（不引 admin 包），由 wiring 层
// 提供 admin.PgSysConfigRepository.GetByKey 的适配器。
//
// 返回 (value, true) 表示读到；(_, false) 表示 key 不存在 / 读失败 — 调用方退化到默认。
type InformPeriodLookup func(ctx context.Context, category, key string) (value string, found bool)

// 心跳周期调整配置位置与键名（与前端 DeviceSettings.tsx 表单字段名一致）。
//   - category=device, key=enbInformPeriodAdjustEnable → 基站心跳调整开关（bool string）
//   - category=device, key=enbInformPeriod → 基站目标周期（秒）
//   - category=device, key=cpeInformPeriodAdjustEnable → CPE心跳调整开关（bool string）
//   - category=device, key=cpeInformPeriod → CPE目标周期（秒）
const (
	informPeriodConfigCategory     = "device"
	informPeriodConfigKeyENBAdjust = "enbInformPeriodAdjustEnable"
	informPeriodConfigKeyENBPeriod = "enbInformPeriod"
	informPeriodConfigKeyCPEAdjust = "cpeInformPeriodAdjustEnable"
	informPeriodConfigKeyCPEPeriod = "cpeInformPeriod"
)

// TR069 参数名：心跳周期（不区分 Device. / InternetGatewayDevice. 树，统一用后缀匹配）
const (
	paramPeriodicInformIntervalSuffix = ".ManagementServer.PeriodicInformInterval"
	// SPV 任务始终使用 TR-181 标准路径，ACS PathTranslationService 在发送时会
	// 根据设备的 ParamModel 自动翻译成对应的私有路径（如 TR-098 设备翻译成
	// InternetGatewayDevice.ManagementServer.PeriodicInformInterval）。
	standardPeriodicInformIntervalPath = "Device.ManagementServer.PeriodicInformInterval"
)

// GPVParams 用于构造 GetParameterValues 请求的 JSON 参数。
type GPVParams struct {
	Names []string `json:"names"`
}

// GPV 任务描述标识（用于响应处理时识别）
const informPeriodGPVDescription = "InformPeriodPolicy:GPV"

// InformPeriodConfig 心跳周期调整配置。
type InformPeriodConfig struct {
	ENBAdjustEnable bool // 基站心跳调整开关
	ENBPeriod       int  // 基站目标周期（秒），0 表示未配置
	CPEAdjustEnable bool // CPE心跳调整开关
	CPEPeriod       int  // CPE目标周期（秒），0 表示未配置
}

// InformPeriodPolicy 实现设备启动时的心跳周期自动调整策略。
//
// 流程：
//  1. 设备 BOOTSTRAP/BOOT Inform 上线
//  2. 入队 GPV 查询当前 PeriodicInformInterval
//  3. GPV 响应后比较配置目标值，不一致则入队 SPV 调整
//
// 设计决策：
//   - 只在 BOOTSTRAP/BOOT 事件时触发，PERIODIC 不触发（避免每次周期 Inform 都做 GPV）
//   - 被动读取失败（Inform 报文不携带 PeriodicInformInterval），改为主动 GPV 查询
//   - 区分 ENB/CPE：复用 isCPEClass() 判断逻辑
type InformPeriodPolicy struct {
	lookup      InformPeriodLookup
	taskService task.Enqueuer
	logger      *zap.Logger
}

// NewInformPeriodPolicy 创建心跳周期调整策略实例。
//
// lookup=nil 或 taskService=nil 时退化为禁用状态（Enabled() 返回 false）。
func NewInformPeriodPolicy(lookup InformPeriodLookup, taskService task.Enqueuer, logger *zap.Logger) *InformPeriodPolicy {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &InformPeriodPolicy{
		lookup:      lookup,
		taskService: taskService,
		logger:      logger,
	}
}

// Enabled 返回策略是否启用（依赖项是否齐全）。
func (p *InformPeriodPolicy) Enabled() bool {
	return p != nil && p.lookup != nil && p.taskService != nil
}

// ShouldTrigger 判断是否应该触发心跳周期调整（仅 BOOTSTRAP/BOOT 事件）。
//
// eventCodes 是 Inform 的事件代码列表（如 ["0 BOOTSTRAP", "4 VALUE CHANGE"]）。
func (p *InformPeriodPolicy) ShouldTrigger(eventCodes []string) bool {
	if !p.Enabled() {
		return false
	}
	for _, code := range eventCodes {
		if code == tr069.EventBootstrap || code == tr069.EventBoot {
			return true
		}
	}
	return false
}

// LoadConfig 从 sys_configs 读取心跳周期调整配置。
func (p *InformPeriodPolicy) LoadConfig(ctx context.Context) InformPeriodConfig {
	var cfg InformPeriodConfig
	if p == nil || p.lookup == nil {
		return cfg
	}

	// ENB 配置
	if v, ok := p.lookup(ctx, informPeriodConfigCategory, informPeriodConfigKeyENBAdjust); ok {
		cfg.ENBAdjustEnable = parseBoolString(v)
	}
	if v, ok := p.lookup(ctx, informPeriodConfigCategory, informPeriodConfigKeyENBPeriod); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ENBPeriod = n
		}
	}

	// CPE 配置
	if v, ok := p.lookup(ctx, informPeriodConfigCategory, informPeriodConfigKeyCPEAdjust); ok {
		cfg.CPEAdjustEnable = parseBoolString(v)
	}
	if v, ok := p.lookup(ctx, informPeriodConfigCategory, informPeriodConfigKeyCPEPeriod); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.CPEPeriod = n
		}
	}

	return cfg
}

// EnqueueGPVTask 入队 GetParameterValues 任务查询设备当前心跳周期。
//
// 返回 error 仅用于日志记录，不阻塞 Inform 处理。
func (p *InformPeriodPolicy) EnqueueGPVTask(ctx context.Context, deviceSN, productClass string) error {
	if !p.Enabled() {
		return nil
	}

	cfg := p.LoadConfig(ctx)
	isCPE := isCPEClassForInformPeriod(productClass)

	// 检查对应设备类型的调整是否启用
	if isCPE && !cfg.CPEAdjustEnable {
		p.logger.Debug("inform period adjustment disabled for CPE",
			zap.String("device_sn", deviceSN),
			zap.String("product_class", productClass))
		return nil
	}
	if !isCPE && !cfg.ENBAdjustEnable {
		p.logger.Debug("inform period adjustment disabled for ENB",
			zap.String("device_sn", deviceSN),
			zap.String("product_class", productClass))
		return nil
	}

	// 构造 GPV 参数：查询 PeriodicInformInterval
	// 所有 FAP/eNB 小基站和现代 CPE 都使用 TR-181 (Device:2) 数据模型
	// InternetGatewayDevice. 是旧版 TR-098 模型，仅用于旧式 DSL 网关
	// 不能同时发送两个路径：设备对不支持的路径返回 9005 导致整体失败
	params := GPVParams{
		Names: []string{
			"Device.ManagementServer.PeriodicInformInterval",
		},
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		p.logger.Error("marshal GPV params",
			zap.String("device_sn", deviceSN),
			zap.Error(err))
		return err
	}

	// 入队 GPV 任务
	req := &task.CreateTaskRequest{
		DeviceSN:    deviceSN,
		Method:      "GetParameterValues",
		Params:      paramsJSON,
		Priority:    10,                                              // 高优先级（0 最高）
		Source:      task.TaskSourceSystem,                           // 系统自动触发
		Description: informPeriodGPVDescription + ":" + productClass, // 用于响应处理时识别
	}

	_, err = p.taskService.CreateTask(ctx, req)
	if err != nil {
		p.logger.Error("enqueue inform period GPV task",
			zap.String("device_sn", deviceSN),
			zap.String("product_class", productClass),
			zap.Error(err))
		return err
	}

	p.logger.Info("enqueued inform period GPV task",
		zap.String("device_sn", deviceSN),
		zap.String("product_class", productClass),
		zap.Bool("is_cpe", isCPE))

	return nil
}

// isCPEClassForInformPeriod 判定 product_class 是否属于 "CPE 类"。
// 与 device 包的 isCPEClass 逻辑一致（含 cpe/home/residential/indoor 任一关键字）。
func isCPEClassForInformPeriod(productClass string) bool {
	pc := strings.ToLower(productClass)
	return strings.Contains(pc, "cpe") ||
		strings.Contains(pc, "home") ||
		strings.Contains(pc, "residential") ||
		strings.Contains(pc, "indoor")
}

// parseBoolString 解析布尔字符串（"true"/"1"/"yes" 为 true，其余为 false）。
func parseBoolString(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes"
}

// IsInformPeriodGPVTask 判断任务是否是心跳周期 GPV 查询任务（通过 Description 标识）。
func IsInformPeriodGPVTask(taskDescription string) bool {
	return strings.HasPrefix(taskDescription, informPeriodGPVDescription)
}

// ExtractProductClassFromDescription 从 Description 中提取 productClass。
// Description 格式：`InformPeriodPolicy:GPV:productClass`
func ExtractProductClassFromDescription(desc string) string {
	if !strings.HasPrefix(desc, informPeriodGPVDescription+":") {
		return ""
	}
	return strings.TrimPrefix(desc, informPeriodGPVDescription+":")
}

// SPVParams 用于构造 SetParameterValues 请求的 JSON 参数。
// 字段名 "values" 与 rpc.SetParameterValuesHandler.BuildRequest 期望的结构一致。
type SPVParams struct {
	Values []SPVParameter `json:"values"`
}

// SPVParameter SetParameterValues 参数。
type SPVParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"` // 可选，默认 xsd:string
}

// GPV 任务 SPV 后续任务描述标识
const informPeriodSPVDescription = "InformPeriodPolicy:SPV"

// ProcessGPVResponse 处理心跳周期 GPV 响应，比较当前值与配置目标值，不一致则入队 SPV。
//
// paramValues 是 GPV 响应中的参数列表（Name-Value 对）。
// productClass 从任务 Description 中提取。
// 返回 true 表示已入队 SPV，false 表示无需调整或处理失败。
func (p *InformPeriodPolicy) ProcessGPVResponse(
	ctx context.Context,
	deviceSN string,
	productClass string,
	paramValues []ParameterValue,
) bool {
	if !p.Enabled() {
		return false
	}

	// 从响应中提取 PeriodicInformInterval 当前值
	var currentInterval int
	var foundPath string
	for _, pv := range paramValues {
		if strings.HasSuffix(pv.Name, paramPeriodicInformIntervalSuffix) {
			if v, err := strconv.Atoi(pv.Value); err == nil && v > 0 {
				currentInterval = v
				foundPath = pv.Name
				break
			}
		}
	}

	if currentInterval == 0 {
		p.logger.Debug("PeriodicInformInterval not found or invalid in GPV response",
			zap.String("device_sn", deviceSN),
			zap.String("product_class", productClass))
		return false
	}

	// 加载配置，获取目标值
	cfg := p.LoadConfig(ctx)
	isCPE := isCPEClassForInformPeriod(productClass)

	var targetInterval int
	if isCPE {
		if !cfg.CPEAdjustEnable || cfg.CPEPeriod <= 0 {
			p.logger.Debug("inform period adjustment disabled or no target for CPE",
				zap.String("device_sn", deviceSN))
			return false
		}
		targetInterval = cfg.CPEPeriod
	} else {
		if !cfg.ENBAdjustEnable || cfg.ENBPeriod <= 0 {
			p.logger.Debug("inform period adjustment disabled or no target for ENB",
				zap.String("device_sn", deviceSN))
			return false
		}
		targetInterval = cfg.ENBPeriod
	}

	// 比较当前值与目标值
	if currentInterval == targetInterval {
		p.logger.Info("inform period already matches target, no adjustment needed",
			zap.String("device_sn", deviceSN),
			zap.Int("current", currentInterval),
			zap.Int("target", targetInterval))
		return false
	}

	// 入队 SPV 任务调整心跳周期
	// 日志记录设备返回的私有路径（用于调试），但 SPV 任务始终用标准路径
	p.logger.Info("inform period mismatch, enqueuing SPV to adjust",
		zap.String("device_sn", deviceSN),
		zap.Int("current", currentInterval),
		zap.Int("target", targetInterval),
		zap.String("device_path", foundPath)) // 设备返回的私有路径（仅日志）

	return p.enqueueSPVTask(ctx, deviceSN, targetInterval)
}

// enqueueSPVTask 入队 SetParameterValues 任务设置心跳周期。
//
// 任务使用 TR-181 标准路径 Device.ManagementServer.PeriodicInformInterval，
// ACS PathTranslationService 在发送时会根据设备 ParamModel 翻译成私有路径。
func (p *InformPeriodPolicy) enqueueSPVTask(ctx context.Context, deviceSN string, targetValue int) bool {
	params := SPVParams{
		Values: []SPVParameter{
			{
				Name:  standardPeriodicInformIntervalPath, // 使用标准路径
				Value: strconv.Itoa(targetValue),
				Type:  "xsd:unsignedInt",
			},
		},
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		p.logger.Error("marshal SPV params",
			zap.String("device_sn", deviceSN),
			zap.Error(err))
		return false
	}

	req := &task.CreateTaskRequest{
		DeviceSN:    deviceSN,
		Method:      "SetParameterValues",
		Params:      paramsJSON,
		Priority:    10, // 高优先级
		Source:      task.TaskSourceSystem,
		Description: informPeriodSPVDescription,
	}

	_, err = p.taskService.CreateTask(ctx, req)
	if err != nil {
		p.logger.Error("enqueue inform period SPV task",
			zap.String("device_sn", deviceSN),
			zap.Int("target", targetValue),
			zap.Error(err))
		return false
	}

	p.logger.Info("enqueued inform period SPV task",
		zap.String("device_sn", deviceSN),
		zap.String("path", standardPeriodicInformIntervalPath),
		zap.Int("target", targetValue))

	return true
}

// ParameterValue 表示 GPV 响应中的参数名-值对。
type ParameterValue struct {
	Name  string
	Value string
}
