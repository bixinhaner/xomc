package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/task"
)

// ─────────────────────────────────────────────────────────────────────────────
// TR-069 FileType 映射
//
// 根据 TR-069 Amendment 6 (cwmp-1-4) 规范和 LMT API 指南：
//
// Upload (CPE→ACS):
//   标准:
//     "1 Vendor Configuration File"    — 配置文件
//     "2 Vendor Log File"              — 日志文件
//   扩展:
//     "4 Vendor Log File"              — 日志文件（扩展编号）
//     "4 Vendor PM File"               — 性能管理文件（运营商扩展）
//     "5 Vendor MR File"               — 测量报告（运营商扩展）
//     "9 Vendor PCAP"                  — 抓包文件
//     "10 <OUI> Configuration File"    — 厂商特定配置文件
//     "11 OUI Parameter Model"         — 数据模型文件
//     "Tr069 Ssl Cert File"            — TR069 SSL 证书
//
// Download (ACS→CPE):
//   标准:
//     "1 Firmware Upgrade Image"       — 固件升级
//     "2 Web Content"                  — Web 内容
//     "3 Vendor Configuration File"    — 配置文件
//   扩展:
//     "10 <OUI> Configuration File"    — 厂商特定配置文件（自动解析导入参数）
//     "101 Script File"                — 脚本文件
//     "103 Base Station Startup File"  — 基站启动文件
//     "License File"                   — License 文件
//     "Tr069 Ssl Cert File"            — TR069 SSL 证书
// ─────────────────────────────────────────────────────────────────────────────

// defaultOUI is the Baicells OUI used for OUI-specific file types.
const defaultOUI = "48BF74"

// uploadFileTypeMap maps aliases to TR-069 Upload FileType strings.
var uploadFileTypeMap = map[string]string{
	// TR-069 标准类型
	"config": "1 Vendor Configuration File",
	"log":    "2 Vendor Log File",

	// 运营商/厂商扩展类型
	"running-log":  "2 Vendor Log File",
	"log-ext":      "4 Vendor Log File",      // 日志文件（扩展编号）
	"security-log": "2 Vendor Security Log",  // 安全日志
	"fault-log":    "2 Vendor Fault Log",     // 故障日志
	"pm":           "4 Vendor PM File",       // 性能管理文件
	"mr":           "5 Vendor MR File",       // 测量报告
	"pcap":         "9 Vendor PCAP",          // 抓包文件
	"datamodel":    "11 OUI Parameter Model", // 数据模型文件
	"config-11":    "11 Configuration File",  // 11 号配置文件
	"ssl-cert":     "Tr069 Ssl Cert File",    // TR069 SSL 证书
}

// uploadFileTypeCodeMap maps numeric codes to Upload FileType strings.
var uploadFileTypeCodeMap = map[string]string{
	"1":  "1 Vendor Configuration File",
	"2":  "2 Vendor Log File",
	"4":  "4 Vendor PM File",
	"5":  "5 Vendor MR File",
	"9":  "9 Vendor PCAP",
	"11": "11 Configuration File",
}

// uploadQueryParam maps rpctool aliases to the short fileType code
// used in the upload URL query parameter: ?fileType=<code>
// This code is parsed by the upload handler's normalizeFileType() to determine
// which MinIO bucket to store the uploaded file in.
var uploadQueryParam = map[string]string{
	// normalizeFileType() 使用 Download 编号体系：
	// 1=Firmware, 2=Patch, 3=Config, 4=PM, 5=MR, 6=LOG, 7=SecurityLog,
	// 8=FaultLog, 9=PCAP, 10=Web, 11=DataModel
	"config":       "3",   // FileTypeConfig
	"log":          "LOG", // FileTypeRunningLog
	"running-log":  "LOG", // FileTypeRunningLog
	"log-ext":      "LOG", // FileTypeRunningLog
	"security-log": "7",   // FileTypeSecurityLog
	"fault-log":    "8",   // FileTypeFaultLog
	"pm":           "PM",  // FileTypePM
	"mr":           "MR",  // FileTypeMR
	"pcap":         "9",   // FileTypePCAP
	"oui-config":   "10",  // FileTypeWeb
	"datamodel":    "11",  // FileTypeDataModel
	"config-11":    "11",  // FileTypeDataModel
	"ssl-cert":     "SSL", // SSL 证书（需服务端 normalizeFileType 支持）
}

// buildUploadURL constructs the full upload URL from base URL, path, fileType and device SN.
// baseURL: e.g. "http://localhost:8080"
// path: e.g. "/smallcell/FileUploadService"
// alias: rpctool file type alias (e.g. "pm", "config-11")
// deviceSN: device serial number for filename generation
func buildUploadURL(baseURL, path, alias, deviceSN string) string {
	// Determine query param code
	code := uploadQueryParam[alias]
	if code == "" {
		// Fallback: extract leading digits from the resolved FileType
		code = alias
	}

	// Generate filename: {sn}_{timestamp}.dat
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.xml.gz", deviceSN, timestamp)

	return fmt.Sprintf("%s%s?fileType=%s&filename=%s",
		strings.TrimRight(baseURL, "/"), path, code, filename)
}

// downloadFileTypeMap maps aliases to TR-069 Download FileType strings.
var downloadFileTypeMap = map[string]string{
	// TR-069 标准类型
	"firmware": "1 Firmware Upgrade Image",
	"web":      "2 Web Content",
	"config":   "3 Vendor Configuration File",

	// 厂商/运营商扩展类型
	"script":   "101 Script File",
	"startup":  "103 Base Station Startup File",
	"license":  "License File",
	"ssl-cert": "Tr069 Ssl Cert File",
}

// downloadFileTypeCodeMap maps numeric codes to Download FileType strings.
var downloadFileTypeCodeMap = map[string]string{
	"1":   "1 Firmware Upgrade Image",
	"2":   "2 Web Content",
	"3":   "3 Vendor Configuration File",
	"101": "101 Script File",
	"103": "103 Base Station Startup File",
}

// resolveUploadFileType converts an alias or numeric code to a TR-069 Upload FileType string.
// Special cases:
//   - "oui-config" or "10" returns "10 <OUI> Configuration File" (OUI filled by caller)
//   - If input contains spaces, it is treated as a raw FileType string (passthrough)
func resolveUploadFileType(input string) (string, error) {
	// Raw passthrough: if input looks like a full FileType string (contains spaces), use as-is
	if strings.Contains(input, " ") {
		return input, nil
	}

	// Special: oui-config handled by caller (needs --oui flag)
	if input == "oui-config" || input == "10" {
		return "", fmt.Errorf("__OUI_CONFIG__")
	}

	if _, err := strconv.Atoi(input); err == nil {
		if ft, ok := uploadFileTypeCodeMap[input]; ok {
			return ft, nil
		}
		return "", fmt.Errorf("未知 Upload 文件类型代码: %s\n可用代码: 1(配置), 2(日志), 4(PM), 5(MR), 9(PCAP), 11(数据模型)", input)
	}
	if ft, ok := uploadFileTypeMap[input]; ok {
		return ft, nil
	}
	return "", fmt.Errorf("未知 Upload 文件类型: %q\n可用别名: config, log, log-ext, running-log, security-log, fault-log, pm, mr, pcap, datamodel, oui-config, ssl-cert", input)
}

// resolveDownloadFileType converts an alias or numeric code to a TR-069 Download FileType string.
// Special cases:
//   - "oui-config" or "10" returns "10 <OUI> Configuration File" (OUI filled by caller)
//   - If input contains spaces, it is treated as a raw FileType string (passthrough)
func resolveDownloadFileType(input string) (string, error) {
	// Raw passthrough: if input looks like a full FileType string (contains spaces), use as-is
	if strings.Contains(input, " ") {
		return input, nil
	}

	// Special: oui-config handled by caller (needs --oui flag)
	if input == "oui-config" || input == "10" {
		return "", fmt.Errorf("__OUI_CONFIG__")
	}

	if _, err := strconv.Atoi(input); err == nil {
		if ft, ok := downloadFileTypeCodeMap[input]; ok {
			return ft, nil
		}
		return "", fmt.Errorf("未知 Download 文件类型代码: %s\n可用代码: 1(固件), 2(Web), 3(配置), 101(脚本), 103(启动文件)", input)
	}
	if ft, ok := downloadFileTypeMap[input]; ok {
		return ft, nil
	}
	return "", fmt.Errorf("未知 Download 文件类型: %q\n可用别名: firmware, web, config, script, startup, license, ssl-cert, oui-config", input)
}

// resolveOUIConfigFileType returns the OUI-specific configuration file type string.
func resolveOUIConfigFileType(oui string) string {
	if oui == "" {
		oui = defaultOUI
	}
	return fmt.Sprintf("10 %s Configuration File", oui)
}

// ─────────────────────────────────────────────────────────────────────────────
// 任务创建
// ─────────────────────────────────────────────────────────────────────────────

// createAndPrint creates a task via TaskService (Redis + PostgreSQL) and prints the result.
func createAndPrint(ctx context.Context, sn, method string, params interface{}, description string) error {
	var rawParams json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("序列化参数: %w", err)
		}
		rawParams = data
	}

	// Use --desc flag if provided, otherwise use auto-generated description.
	taskDesc := desc
	if taskDesc == "" {
		taskDesc = description
	}

	req := &task.CreateTaskRequest{
		DeviceSN:    sn,
		Method:      method,
		Params:      rawParams,
		Priority:    priority,
		ExpiresIn:   ttlSeconds,
		Source:      task.TaskSource(source),
		CreatorID:   creator,
		Description: taskDesc,
	}

	if dryRun {
		t := task.NewTask(req)
		prettyJSON, _ := json.MarshalIndent(t, "", "  ")
		fmt.Println("=== Dry Run (未写入数据库) ===")
		fmt.Printf("设备 SN:  %s\n", t.DeviceSN)
		fmt.Printf("方法:     %s\n", t.Method)
		fmt.Printf("优先级:   %d\n", t.Priority)
		fmt.Printf("来源:     %s\n", t.Source)
		fmt.Printf("创建者:   %s\n", t.CreatorID)
		if t.ExpiresAt != nil {
			fmt.Printf("过期时间: %s\n", t.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println("任务 JSON:")
		fmt.Println(string(prettyJSON))
		return nil
	}

	t, err := taskSvc.CreateTask(ctx, req)
	if err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}

	prettyJSON, _ := json.MarshalIndent(t, "", "  ")
	fmt.Println("=== 任务已创建 (Redis + PostgreSQL) ===")
	fmt.Printf("任务 ID:  %s\n", t.ID)
	fmt.Printf("设备 SN:  %s\n", t.DeviceSN)
	fmt.Printf("方法:     %s\n", t.Method)
	fmt.Printf("状态:     %s\n", t.Status)
	fmt.Printf("优先级:   %d\n", t.Priority)
	fmt.Printf("来源:     %s\n", t.Source)
	fmt.Printf("创建者:   %s\n", t.CreatorID)
	if t.ExpiresAt != nil {
		fmt.Printf("过期时间: %s\n", t.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("任务 JSON:")
	fmt.Println(string(prettyJSON))

	return nil
}
