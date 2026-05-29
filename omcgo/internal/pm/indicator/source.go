package indicator

import "strings"

// Source 描述一行 perf_indicators_* 的物理来源(T-0180,对标 T-0178 parammodel/source.go)。
//
// 三态语义:
//   - SourceBuiltin: XML 来自镜像层 data/indicator-library/,只读,不可在线删
//   - SourceCustom:  XML 来自 host bind mount data/indicator-library-custom/,可读写,可在线删
//   - SourceUnknown: 历史数据或异常(loaded_from 不带目录前缀或为空),按 builtin 对待(拒删)
type Source string

const (
	SourceBuiltin Source = "builtin"
	SourceCustom  Source = "custom"
	SourceUnknown Source = "unknown"
)

// BuiltinDirPrefix / CustomDirPrefix 是 perf_indicators_*.loaded_from 列的目录前缀约定。
// Loader 写入时 = filepath.Rel(XMLBaseDir, absPath) | ToSlash,
// 因此一定以 "indicator-library/" 或 "indicator-library-custom/" 开头(builtin 路径)
// 或 "indicator-library/enb/..." 这类带子目录的形式(builtin LTE 多文件)。
//
// 改前缀只动这里;handler / loader / 前端 DTO 均通过 ClassifySource / IsDeletable
// 间接判定,不直接 strings.Contains。
const (
	BuiltinDirPrefix = "indicator-library/"
	CustomDirPrefix  = "indicator-library-custom/"
)

// BuiltinDirSubdir / CustomDirSubdir 是 XMLBaseDir 下的子目录名(不含末尾 /)。
// Loader 扫描目录、Handler Upload 写入目标路径、deploy.sh 初始化目录均引用这两个常量;
// 与 *DirPrefix 保持一致(后者多一个 / 用于字符串前缀匹配)。
const (
	BuiltinDirSubdir = "indicator-library"        // 镜像层只读 builtin XML 目录(含 enb/ 子目录 + GSM.xml / GNB.xml 单文件)
	CustomDirSubdir  = "indicator-library-custom" // host bind mount 持久化 custom XML 目录(enb/gsm/gnb 全部子目录化)
)

// ClassifySource 根据 perf_indicators_*.loaded_from 列值判定物理来源。
//
// 输入约定:loaded_from 应为相对 XMLBaseDir 的 slash 路径
// (Loader 用 filepath.ToSlash 规范化,Windows 反斜杠也兼容)。
//
// 行为:
//   - 以 CustomDirPrefix 开头 → SourceCustom
//   - 以 BuiltinDirPrefix 开头 → SourceBuiltin
//   - 其他(空串 / 裸文件名 / 路径遍历模式)→ SourceUnknown
//
// SourceUnknown 包括历史数据(migration 000217 之前的 Loader 入库无 loaded_from 列;
// 迁移后到 Reload 之间的过渡期可能见到)。
func ClassifySource(loadedFrom string) Source {
	switch {
	case strings.HasPrefix(loadedFrom, CustomDirPrefix):
		return SourceCustom
	case strings.HasPrefix(loadedFrom, BuiltinDirPrefix):
		return SourceBuiltin
	default:
		return SourceUnknown
	}
}

// IsDeletable 是 DELETE /indicators/files/{path} 端点与前端 deletable 字段的
// 唯一判定函数(T-0180 PRD §3 GWT-5 守门规则)。
//
// 当前规则:仅 SourceCustom 可删。SourceBuiltin / SourceUnknown 一律不可删
// (后者从安全侧考虑:无法确认来源时按保守语义拒绝,避免误删用户/历史数据)。
//
// 调用方:
//   - 后端 handler.DeleteIndicatorFile 入口守门(返 403 ErrCodeIndicatorBuiltinNotDeletable)
//   - List/Files DTO 的 deletable 字段填值
//   - 前端不重新推导,直接渲染 deletable bool
func IsDeletable(loadedFrom string) bool {
	return ClassifySource(loadedFrom) == SourceCustom
}
