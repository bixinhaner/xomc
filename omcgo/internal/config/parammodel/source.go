package parammodel

import (
	"os"
	"path/filepath"
)

// Source 描述一行 param_model 的物理来源。
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

// BuiltinDirSubdir 是 XMLBaseDir 下唯一的 paramModel XML 子目录名(不含末尾 /)。
// 三库 XML 导入重构(2026-06-04 D3/D5/D6):取消 builtin/custom 双目录,所有 XML
// (出厂 + 用户上传)同住此目录;来源由同目录 sidecar(X.xml.custom)判定,见
// CustomMarkerSuffix。Loader 扫描、Handler Upload 写入、deploy.sh 初始化目录均引用此常量。
const BuiltinDirSubdir = "param-mappings" // 唯一 paramModel XML 目录(builtin + custom 同住)

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

// IsDeletable 是 DELETE /param-models/:name 端点与前端 deletable 字段的唯一判定函数。
// 仅 custom(有 sidecar)可删;builtin / unknown 一律不可删。
func IsDeletable(baseDir, loadedFrom string) bool {
	return IsCustom(baseDir, loadedFrom)
}
