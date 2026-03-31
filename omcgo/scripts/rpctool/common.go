package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/omcgo/omcgo/internal/task"
)

// ─────────────────────────────────────────────────────────────────────────────
// TR-069 FileType 映射
//
// 根据 TR-069 Amendment 6 (cwmp-1-4) 规范：
//
// Upload (CPE→ACS):
//   "1 Vendor Configuration File"    — 配置文件
//   "2 Vendor Log File"              — 日志文件
//   其他为厂商/运营商私有扩展
//
// Download (ACS→CPE):
//   "1 Firmware Upgrade Image"       — 固件升级
//   "2 Web Content"                  — Web 内容
//   "3 Vendor Configuration File"    — 配置文件
//   其他为厂商/运营商私有扩展
// ─────────────────────────────────────────────────────────────────────────────

// uploadFileTypeMap maps aliases to TR-069 Upload FileType strings.
var uploadFileTypeMap = map[string]string{
	// TR-069 标准类型
	"config":  "1 Vendor Configuration File",
	"log":     "2 Vendor Log File",

	// 运营商扩展类型（中国运营商规范）
	"running-log":  "2 Vendor Log File",
	"security-log": "2 Vendor Security Log",
	"fault-log":    "2 Vendor Fault Log",
	"pm":           "4 Vendor PM File",
	"mr":           "5 Vendor MR File",
	"pcap":         "9 Vendor PCAP",
	"datamodel":    "11 OUI Parameter Model",
}

// uploadFileTypeCodeMap maps numeric codes to Upload FileType strings.
var uploadFileTypeCodeMap = map[string]string{
	"1":  "1 Vendor Configuration File",
	"2":  "2 Vendor Log File",
	"4":  "4 Vendor PM File",
	"5":  "5 Vendor MR File",
	"9":  "9 Vendor PCAP",
	"11": "11 OUI Parameter Model",
}

// downloadFileTypeMap maps aliases to TR-069 Download FileType strings.
var downloadFileTypeMap = map[string]string{
	// TR-069 标准类型
	"firmware": "1 Firmware Upgrade Image",
	"web":      "2 Web Content",
	"config":   "3 Vendor Configuration File",

	// 运营商扩展类型
	"patch": "2 Vendor Patch File",
}

// downloadFileTypeCodeMap maps numeric codes to Download FileType strings.
var downloadFileTypeCodeMap = map[string]string{
	"1": "1 Firmware Upgrade Image",
	"2": "2 Web Content",
	"3": "3 Vendor Configuration File",
}

// resolveUploadFileType converts an alias or numeric code to a TR-069 Upload FileType string.
func resolveUploadFileType(input string) (string, error) {
	if _, err := strconv.Atoi(input); err == nil {
		if ft, ok := uploadFileTypeCodeMap[input]; ok {
			return ft, nil
		}
		return "", fmt.Errorf("未知 Upload 文件类型代码: %s\n可用代码: 1(配置), 2(日志), 4(PM), 5(MR), 9(PCAP), 11(数据模型)", input)
	}
	if ft, ok := uploadFileTypeMap[input]; ok {
		return ft, nil
	}
	return "", fmt.Errorf("未知 Upload 文件类型: %q\n可用别名: config, log, running-log, security-log, fault-log, pm, mr, pcap, datamodel", input)
}

// resolveDownloadFileType converts an alias or numeric code to a TR-069 Download FileType string.
func resolveDownloadFileType(input string) (string, error) {
	if _, err := strconv.Atoi(input); err == nil {
		if ft, ok := downloadFileTypeCodeMap[input]; ok {
			return ft, nil
		}
		return "", fmt.Errorf("未知 Download 文件类型代码: %s\n可用代码: 1(固件), 2(Web), 3(配置)", input)
	}
	if ft, ok := downloadFileTypeMap[input]; ok {
		return ft, nil
	}
	return "", fmt.Errorf("未知 Download 文件类型: %q\n可用别名: firmware, web, config, patch", input)
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
