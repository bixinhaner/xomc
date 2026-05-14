package main

import "strings"

// zhDictionary TR-069 高频词英中翻译词典（Q4=A 决议，~200 词）。
//
// 使用：renderNameI18n → splitPascalCase → translateZh 逐词查表。
// 命中策略：逐词查；任一词不命中即返回空（前端 "🟡 需翻译" badge 兜底）。
// 后续 P3 admin UI 上线后迁出到 omcgo/data/mml-zh-dictionary.json 文件加载。
//
// 词典覆盖原则：
//   - TR-069 spec §A.2 高频 noun（Device / Service / Interface / Cell / Channel ...）
//   - TR-098 / TR-181 标准化术语
//   - 状态 / 操作动词（Status / Enable / Reset / Reboot / Update ...）
//   - 缩写不译（OUI / MAC / IP / DNS / UDP / TCP / API / SDK 等）
var zhDictionary = map[string]string{
	// 设备
	"Device": "设备", "Devices": "设备", "Equipment": "设备",
	"Module": "模块", "Component": "组件", "Hardware": "硬件",
	"Software": "软件", "Firmware": "固件", "System": "系统",
	"Product": "产品", "Manufacturer": "厂商", "Vendor": "厂商",
	"Model": "型号", "Name": "名称", "Serial": "序列号",
	"Number": "编号", "Code": "编码", "Id": "标识",
	"Identifier": "标识", "Version": "版本", "Revision": "修订",
	"Class": "类别", "Type": "类型", "Category": "分类",
	"Description": "描述", "Comment": "备注", "Note": "注释",
	// 网络
	"Interface": "接口", "Port": "端口", "Address": "地址",
	"Subnet": "子网", "Gateway": "网关", "Route": "路由",
	"Routing": "路由", "Network": "网络", "Connection": "连接",
	"Link": "链路", "Tunnel": "隧道", "Channel": "信道",
	"Frequency": "频率", "Band": "频段", "Bandwidth": "带宽",
	"Spectrum": "频谱",
	// 性能
	"Throughput": "吞吐", "Bitrate": "比特率", "Rate": "速率",
	"Speed": "速度", "Latency": "延迟", "Delay": "延迟",
	"Jitter": "抖动", "Drop": "丢失",
	"Error": "错误", "Errors": "错误", "Fault": "故障",
	"Faults": "故障", "Failure": "故障", "Performance": "性能",
	"Counter": "计数器", "Counters": "计数器", "Stats": "统计",
	"Statistic": "统计", "Statistics": "统计",
	// 蜂窝/小区
	"Cell": "小区", "Cells": "小区", "Sector": "扇区",
	"Carrier": "载波", "Carriers": "载波", "Beam": "波束",
	"Antenna": "天线", "Power": "功率", "Tx": "发射",
	"Rx": "接收", "Gain": "增益", "Loss": "损耗",
	"Azimuth": "方位角", "Tilt": "倾角", "Downtilt": "下倾角",
	"Height": "高度", "Beamwidth": "波束宽度",
	"Pci": "PCI", "Pdcp": "PDCP", "Rrc": "RRC", "Rlc": "RLC",
	"Mac": "MAC", "Phy": "PHY",
	"Eci": "ECI", "Lac": "LAC", "Tac": "TAC", "Plmn": "PLMN",
	"Mcc": "MCC", "Mnc": "MNC",
	// 状态
	"Status": "状态", "State": "状态", "Mode": "模式",
	"Enable": "启用", "Enabled": "启用", "Disable": "禁用",
	"Disabled": "禁用", "Active": "活动", "Inactive": "非活动",
	"Available": "可用", "Online": "在线", "Offline": "离线",
	"Up": "上行", "Down": "下行", "Uplink": "上行",
	"Downlink": "下行", "Running": "运行中", "Stopped": "已停止",
	"Idle": "空闲", "Busy": "繁忙",
	// 时间
	"Time": "时间", "Date": "日期", "Datetime": "日期时间",
	"Timestamp": "时间戳", "Uptime": "运行时间", "Duration": "时长",
	"Period": "周期", "Interval": "间隔", "Timeout": "超时",
	"Schedule": "调度", "Trigger": "触发",
	// 操作
	"Reset": "重置", "Reboot": "重启", "Restart": "重启",
	"Update": "更新", "Upgrade": "升级", "Downgrade": "降级",
	"Install": "安装", "Uninstall": "卸载", "Add": "添加",
	"Remove": "移除", "Delete": "删除", "Create": "创建",
	"Edit": "编辑", "View": "查看", "List": "列表",
	"Set": "设置", "Get": "获取", "Read": "读取",
	"Write": "写入", "Configure": "配置", "Apply": "应用",
	"Save": "保存", "Cancel": "取消", "Confirm": "确认",
	// 配置
	"Config": "配置", "Configuration": "配置", "Param": "参数",
	"Params": "参数", "Parameter": "参数", "Parameters": "参数",
	"Setting": "设置", "Settings": "设置", "Property": "属性",
	"Properties": "属性", "Attribute": "属性", "Value": "值",
	"Default": "默认", "Threshold": "阈值", "Limit": "限制",
	"Min": "最小", "Max": "最大", "Range": "范围",
	"Count": "数量", "Total": "总数", "Index": "索引",
	"Info": "信息", "Information": "信息", "Detail": "详情",
	"Details": "详情",
	// 告警
	"Alarm": "告警", "Alarms": "告警", "Warning": "警告",
	"Critical": "严重", "Major": "主要", "Minor": "次要",
	"Severity": "严重程度", "Event": "事件", "Events": "事件",
	"Notify": "通知", "Notification": "通知",
	// 性能 KPI
	"Kpi": "KPI", "Indicator": "指标", "Indicators": "指标",
	"Metric": "指标", "Metrics": "指标", "Measurement": "测量",
	"Sample": "采样", "Samples": "采样",
	// 安全
	"Auth": "认证", "Authentication": "认证", "Authorization": "鉴权",
	"User": "用户", "Users": "用户", "Password": "密码",
	"Token": "令牌", "Key": "密钥", "Secret": "密钥",
	"Cert": "证书", "Certificate": "证书", "Tls": "TLS",
	"Ssl": "SSL", "Encrypt": "加密", "Encryption": "加密",
	"Decrypt": "解密", "Hash": "哈希",
	// 文件
	"File": "文件", "Files": "文件", "Path": "路径",
	"Url": "URL", "Uri": "URI", "Download": "下载",
	"Upload": "上传", "Storage": "存储",
	// 通用助词
	"Ip": "IP", "Lte": "LTE", "Gsm": "GSM",
	"Nr": "NR", "Umts": "UMTS", "Wcdma": "WCDMA",
	"Server": "服务器", "Client": "客户端", "Source": "源",
	"Destination": "目的", "Target": "目标", "Origin": "来源",
	"Local": "本地", "Remote": "远程", "External": "外部",
	"Internal": "内部",
	// 数据类型
	"Boolean": "布尔", "String": "字符串", "Integer": "整数",
	"Float": "浮点", "Json": "JSON", "Xml": "XML",
}

// translateZh 把英文 phrase（如 "Software Version"）逐词查表翻为中文。
// 任一词不命中 → 返回空字符串（"🟡 需翻译" 兜底）。
//
// 已知缩写（OUI/MAC/IP/...）走 dictionary 命中相同字面值的特殊处理。
func translateZh(en string) string {
	en = strings.TrimSpace(en)
	if en == "" {
		return ""
	}
	words := strings.Fields(en)
	if len(words) == 0 {
		return ""
	}

	out := make([]string, 0, len(words))
	for _, w := range words {
		zh, ok := zhDictionary[w]
		if !ok {
			// 大写首字母兜底（"Software" 命中但 "software" 没存）
			zh, ok = zhDictionary[capitalize(w)]
		}
		if !ok {
			// 全大写缩写自我直通（OUI / MAC / IP / DNS / etc.）
			if isAcronym(w) {
				zh = w
				ok = true
			}
		}
		if !ok {
			return "" // 任一词不命中 → 留空，前端 badge 兜底
		}
		out = append(out, zh)
	}
	// 中文之间不加空格
	return strings.Join(out, "")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}

// isAcronym 判定整词是否全大写（且 ≥2 字符）— 视作缩写不译。
func isAcronym(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
