package definition

import (
	"os"
	"path/filepath"
)

// Source 描述一行 alarm_definitions 的物理来源。
//
// 三态语义(2026-06-04 改 sidecar 判定):
//   - SourceBuiltin: 出厂随包 XML(无 .custom sidecar),不可在线删
//   - SourceCustom:  用户经 UI 上传的 XML(同目录存在 X.xml.custom sidecar 标记),可在线删
//   - SourceUnknown: loaded_from 为空等异常,按 builtin 对待(拒删)
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
// CustomMarkerSuffix 是自定义 XML 的 sidecar 标记后缀。
// 文件 X.xml 若同目录存在 X.xml.custom(空标记文件)⇒ 该 XML 为用户经 UI 上传的自定义文件。
// 标记随文件走 → 扛过 data 反向合并升级 + DB 重建,不依赖目录前缀、不占 DB 列(2026-06-04 设计 D6)。
const CustomMarkerSuffix = ".custom"

// IsCustom 判定 loadedFrom 对应的物理文件是否自定义(同目录 sidecar 存在)。
// baseDir = XMLBaseDir(Loader / Handler 持有);loadedFrom = 相对 baseDir 的 slash 路径。
func IsCustom(baseDir, loadedFrom string) bool {
	if loadedFrom == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(baseDir, filepath.FromSlash(loadedFrom)) + CustomMarkerSuffix)
	return err == nil
}

// ClassifySource 根据 sidecar 判定物理来源(custom / builtin / unknown)。
func ClassifySource(baseDir, loadedFrom string) Source {
	if loadedFrom == "" {
		return SourceUnknown
	}
	if IsCustom(baseDir, loadedFrom) {
		return SourceCustom
	}
	return SourceBuiltin
}

// IsDeletable 是 DELETE /alarm-definitions/files/{path} 端点与前端 deletable 字段的唯一判定函数。
// 仅 custom(有 sidecar)可删;builtin / unknown 一律不可删。
func IsDeletable(baseDir, loadedFrom string) bool {
	return IsCustom(baseDir, loadedFrom)
}
