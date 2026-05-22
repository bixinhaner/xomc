package mmlstandardloader

import (
	"strings"
)

// MaxParamsPerGroup 单个 group 包含的最大叶子 param 数（设计阈值参考）。
// 当前 leaf-based 算法天然按 TR-069 path 自然层级聚合，
// 通常 5-50 个 param/group；超 50 的 flat 大子树视为接受边界情况。
const MaxParamsPerGroup = 50

// GroupSpec 一个逻辑命令分组 — 最终成为 mml_command_groups 一行。
//
// Path 形如 "Device.DeviceInfo.AntennaInfo" 或
// "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell"。
// Path 末段已去掉末尾 `{i}`（如有），保留中间 `{i}` 段。
type GroupSpec struct {
	Path        string      // group 路径（保留中间 {i}，去末尾 {i}）
	Level       int         // path 段数（监控用）
	Code        string      // path 去全部 {i} 后转 UPPER_SNAKE_CASE = group_code
	Params      []ParamSpec // 归入本 group 的全部叶子 param
	HasInstance bool        // path 含任意 {i}（决定能否生成 ADD/RMV 命令）
}

// BuildGroups 把一批 ParamSpec 按"父路径"自然聚合为 group。
//
// 算法：
//  1. 对每个 param，取其 path 去掉**最后一段**（参数 leaf 名）作为父路径
//  2. 父路径末尾若是 `{i}`，去掉（因为父路径的实例号属于参数 leaf 的容器，不是 group 标识）
//  3. 按"去末尾 {i} 后的父路径"分组 → 每组形成一个 GroupSpec
//
// 例：
//
//	"Device.DeviceInfo.AntennaInfo.Azimuth"
//	  segs without last = "Device.DeviceInfo.AntennaInfo"
//	  no trailing {i}   → group path = "Device.DeviceInfo.AntennaInfo"
//
//	"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI"
//	  segs without last = "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}"
//	  strip trailing {i} → "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell"
//	  → group path 保留中间 {i}
//	  → HasInstance=true（可生成 ADD/RMV）
//	  → group_code 去全部 {i} → "DEVICE_SERVICES_FAPSERVICE_CELLCONFIG_LTE_RAN_NEIGHBORLIST_LTECELL"
//
// 不变量：每个 param 都归属**且只属于**一个 group。
func BuildGroups(params []ParamSpec) []GroupSpec {
	if len(params) == 0 {
		return nil
	}
	// 按父路径 group key 聚合
	byKey := make(map[string][]ParamSpec)
	keyOrder := make([]string, 0)
	for _, p := range params {
		key := parentGroupPath(p.StandardPath)
		if _, exists := byKey[key]; !exists {
			keyOrder = append(keyOrder, key)
		}
		byKey[key] = append(byKey[key], p)
	}
	// 按首次出现顺序输出，保持稳定性
	groups := make([]GroupSpec, 0, len(byKey))
	for _, k := range keyOrder {
		if k == "" {
			// fallback 极端情况：param path 只有一段，分到 "_ORPHAN" group
			groups = append(groups, makeGroup("_ORPHAN", byKey[k]))
			continue
		}
		groups = append(groups, makeGroup(k, byKey[k]))
	}
	return groups
}

// parentGroupPath 计算 param standardPath 的 group key：
//   - 去掉最后一段（leaf 参数名）
//   - 去掉**末尾的连续 {i} 段**（参数所在的实例号属于 leaf 容器，不是 group 标识）
//   - 中间的 {i} 段保留
func parentGroupPath(path string) string {
	segs := strings.Split(path, ".")
	if len(segs) <= 1 {
		return "" // ORPHAN
	}
	parent := segs[:len(segs)-1]
	// strip trailing {i}
	for len(parent) > 0 && parent[len(parent)-1] == "{i}" {
		parent = parent[:len(parent)-1]
	}
	return strings.Join(parent, ".")
}

// makeGroup 构造 GroupSpec：计算 Code / Level / HasInstance。
func makeGroup(path string, params []ParamSpec) GroupSpec {
	return GroupSpec{
		Path:        path,
		Level:       strings.Count(path, ".") + 1,
		Code:        toGroupCode(path),
		Params:      params,
		HasInstance: strings.Contains(path, "{i}"),
	}
}

// toGroupCode 把 path 转 group_code（UPPER_SNAKE_CASE，去全部 `{i}`）。
//
//	"Device.DeviceInfo.AntennaInfo"          → "DEVICE_DEVICEINFO_ANTENNAINFO"
//	"Device.Services.FAPService.{i}.CellConfig.LTE" → "DEVICE_SERVICES_FAPSERVICE_CELLCONFIG_LTE"
//	"DeviceGSM.Bts"                          → "DEVICEGSM_BTS"
func toGroupCode(path string) string {
	s := strings.ReplaceAll(path, ".{i}", "")
	s = strings.ReplaceAll(s, "{i}.", "")
	s = strings.ReplaceAll(s, "{i}", "")
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ToUpper(s)
}
