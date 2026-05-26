// Package storage 提供 MinIO 对象存储路径构建和文件路由工具。
// path.go：根据文件类型生成标准化对象路径
// router.go：根据 TR-069 文件类型路由到对应的 Bucket 和分类目录
package storage

import (
	"fmt"
	"time"
)

// ObjectPath 为设备上传的文件构建标准对象路径，格式：
// {category}/{carrier}/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}
// 其中 category 为空时直接用 {carrier}/date/sn/filename。
// 使用场景：配置备份、日志、PM/MR 文件入库前生成 MinIO key。
func ObjectPath(category, carrier, deviceSN, filename string) string {
	now := time.Now()
	if category == "" {
		return fmt.Sprintf("%s/%s/%s/%s",
			carrier,
			now.Format("2006/01/02"),
			deviceSN,
			filename,
		)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		category,
		carrier,
		now.Format("2006/01/02"),
		deviceSN,
		filename,
	)
}

// ObjectPathAt 与 ObjectPath 相同，但接受显式时间戳展代内部 time.Now()。
// 适用于需要以历史时间容纳文件的场景（如重进厅历史文件）。
func ObjectPathAt(category, carrier, deviceSN, filename string, t time.Time) string {
	if category == "" {
		return fmt.Sprintf("%s/%s/%s/%s",
			carrier,
			t.Format("2006/01/02"),
			deviceSN,
			filename,
		)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		category,
		carrier,
		t.Format("2006/01/02"),
		deviceSN,
		filename,
	)
}

// FirmwarePath 为固件文件构建对象路径，格式：
// {category}/{productClass}/{version}/{filename}
// 使用场景：固件/补丁 入库 MinIO 时生成 key，按产品型号和版本分目录管理。
func FirmwarePath(category, productClass, version, filename string) string {
	return fmt.Sprintf("%s/%s/%s/%s",
		category,
		productClass,
		version,
		filename,
	)
}

// ExchangePath 为数据交换文件构建对象路径，格式：
// {category}/{YYYY}/{MM}/{DD}/{userID}/{filename}
// 使用场景：设备导入/导出任务中临时文件入库 MinIO Exchange Bucket。
func ExchangePath(category, userID, filename string) string {
	now := time.Now()
	return fmt.Sprintf("%s/%s/%s/%s",
		category,
		now.Format("2006/01/02"),
		userID,
		filename,
	)
}

// DataModelPath 为数据模型 XML 文件构建对象路径，格式：
// datamodel/{carrier}/{oui}/{productClass}/{deviceSN}_{timestamp}.xml
// 使用场景： ACS 收到 FileType=11 上传后，将 CPE 上传的参数模型 XML
// 存入 Exchange Bucket 的 datamodel/ 目录，后绪App 服务经 NATS 事件处理。
func DataModelPath(carrier, oui, productClass, deviceSN string) string {
	ts := time.Now().Format("20060102150405")
	return fmt.Sprintf("datamodel/%s/%s/%s/%s_%s.xml",
		carrier,
		oui,
		productClass,
		deviceSN,
		ts,
	)
}

// DatePrefix 返回当前日期字符串作为路径前缀 YYYY/MM/DD，用于构建时序目录结构。
func DatePrefix() string {
	return time.Now().Format("2006/01/02")
}

// MRObjectPath 为 MR 文件构建对象路径，格式：{deviceSN}/{filename}
// 用户要求 MR 桶下扁平按 SN 目录组织（2026-05-26 决策）。
// 与 ObjectPath 的多层 {category}/{carrier}/{date}/{sn}/{filename} 不同 —
// MR 走独立轻量路径，桶名 mr-files 本身即表示模块归属。
func MRObjectPath(deviceSN, filename string) string {
	return fmt.Sprintf("%s/%s", deviceSN, filename)
}
