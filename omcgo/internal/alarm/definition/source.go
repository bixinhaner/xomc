package definition

import "strings"

// Source 描述一行 alarm_definitions 的物理来源(严格对标 T-0180 indicator/source.go)。
//
// 三态语义:
//   - SourceBuiltin: XML 来自镜像层 data/alarm-definitions/,只读,不可在线删
//   - SourceCustom:  XML 来自 host bind mount data/alarm-definitions-custom/,可读写,可在线删
//   - SourceUnknown: 历史数据或异常(loaded_from 不带目录前缀或为空),按 builtin 对待(拒删)
type Source string

const (
	SourceBuiltin Source = "builtin"
	SourceCustom  Source = "custom"
	SourceUnknown Source = "unknown"
)

// BuiltinDirPrefix / CustomDirPrefix 是 alarm_definitions.loaded_from 列的目录前缀约定。
// Loader 写入时 = filepath.ToSlash(filepath.Join(subdir, basename)),因此一定以
// "alarm-definitions/" 或 "alarm-definitions-custom/" 开头。
//
// 改前缀只动这里;handler / loader / 前端 DTO 均通过 ClassifySource / IsDeletable
// 间接判定,不直接 strings.Contains。
const (
	BuiltinDirPrefix = "alarm-definitions/"
	CustomDirPrefix  = "alarm-definitions-custom/"
)

// BuiltinDirSubdir / CustomDirSubdir 是 XMLBaseDir 下的子目录名(不含末尾 /)。
// Loader 扫描目录、Handler Upload 写入目标路径、deploy.sh 初始化目录均引用这两个常量。
const (
	BuiltinDirSubdir = "alarm-definitions"        // 镜像层只读 builtin XML 目录(每个 ne_type 一个文件,如 ENB.xml)
	CustomDirSubdir  = "alarm-definitions-custom" // host bind mount 持久化 custom XML 目录
)

// ClassifySource 根据 alarm_definitions.loaded_from 列值判定物理来源。
//
// 输入约定:loaded_from 应为相对 XMLBaseDir 的 slash 路径(Loader 用 filepath.ToSlash
// 规范化)。
//
// 行为:
//   - 以 CustomDirPrefix 开头 → SourceCustom
//   - 以 BuiltinDirPrefix 开头 → SourceBuiltin
//   - 其他(空串 / 裸文件名 / 历史数据)→ SourceUnknown
//
// SourceUnknown 包括历史数据(loaded_from 只存 basename 的旧 Loader 写入,migration
// 回填前的过渡期可能见到)。
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

// IsDeletable 是 DELETE /alarm-definitions/files/{path} 端点与前端 deletable 字段的
// 唯一判定函数。
//
// 当前规则(2026-06-04 用户决策:内置数据不可删除):仅 custom 来源可删,
// builtin(当前目录 XML 加载的内置数据)与 unknown 一律不可删。
// ⚠️ 注:2026-06-03 起上传直接写 builtin 目录,故上传文件也判为 builtin →
// 同样不可删(只能重新上传同名覆盖)。若需"上传可删、出厂锁定",应把上传分流到 custom 目录。
func IsDeletable(loadedFrom string) bool {
	return ClassifySource(loadedFrom) == SourceCustom
}
