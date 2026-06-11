// Package bundle —— 文件管理 4 Tab 批量下载,**同步流式**。
//
// 流程极简：
//  1. 前端 POST /<module>/batch-download → handler 直接流 zip 到 response writer
//  2. 浏览器看到 Content-Disposition attachment → 一次下载
//
// 不存中间产物,不开后台 goroutine,不需要表。大批量(GB 级)走 nginx
// proxy_read_timeout 调长即可。
package bundle

import "context"

// Module 标识批量打包任务来自哪个文件管理 Tab。
type Module string

const (
	ModuleFirmware       Module = "firmware"
	ModuleConfigSnapshot Module = "config_snapshot"
	ModuleDeviceLicense  Module = "device_license"
	ModuleMR             Module = "mr"
	// ModuleMRFiles: 跟 ModuleMR 同样打 MR 文件,但 targetIDs 是 mr_files.id
	// (用户在 DeviceFilesDrawer 勾选具体文件),粒度比按设备整盘下载更细。
	ModuleMRFiles Module = "mr_files"
	// PM 文件批量下载: ModulePM 按设备 SN 整盘打,ModulePMFiles 按 pm_files.id 勾选打,
	// 跟 MR 双 module 同构。
	ModulePM      Module = "pm"
	ModulePMFiles Module = "pm_files"
)

// BundleFile —— 一个 zip 条目: 从 MinIO {Bucket}/{ObjectPath} 拉,在 zip 里写为 EntryName。
// EntryName 可以含 "/" 表示子目录(MR 走 {SN}/{filename} 嵌套)。
type BundleFile struct {
	Bucket     string
	ObjectPath string
	EntryName  string
	// DeviceSN 是该文件归属设备的序列号；#63 设备组可见性强制层用它对「按文件 ID
	// 勾选下载」（mr_files / pm_files）的结果做归属过滤——这两类 targetIDs 是文件 ID，
	// 入口无法预先按 SN 过滤，只能在文件解析出 SN 后由 WriteZipTo 后置剔除域外设备。
	// 留空表示该文件不归属任何设备（如 firmware 镜像是全局制品，不参与设备组过滤）。
	DeviceSN string
}

// Source —— 模块特定的"按 targetIDs 列出物理文件"。Service 启动时各模块注册,
// WriteZipTo 触发时按 module 路由。
//   - firmware: targetIDs 是 firmware_versions.id (UUID 字符串列表)
//   - config_snapshot / device_license / mr: targetIDs 是 serial_number 列表
type Source func(ctx context.Context, targetIDs []string) ([]BundleFile, error)
