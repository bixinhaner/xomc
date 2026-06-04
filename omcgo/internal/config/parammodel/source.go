package parammodel

import "strings"

// Source 描述一行 param_model 的物理来源(T-0178)。
//
// 三态语义:
//   - SourceBuiltin: XML 来自镜像层 data/param-mappings/,只读,不可在线删
//   - SourceCustom:  XML 来自 host bind mount data/param-mappings-custom/,可读写,可在线删
//   - SourceUnknown: 历史数据或异常(loaded_from 不带目录前缀),按 builtin 对待(拒删)
type Source string

const (
	SourceBuiltin Source = "builtin"
	SourceCustom  Source = "custom"
	SourceUnknown Source = "unknown"
)

// BuiltinDirPrefix / CustomDirPrefix 是 loaded_from 列的目录前缀约定。
// Loader 写入时 = filepath.Rel(XMLBaseDir, absPath) | ToSlash,
// 因此一定以 "param-mappings/" 或 "param-mappings-custom/" 开头。
//
// 改前缀只动这里;handler / loader / 前端 DTO 均通过 ClassifySource / IsDeletable
// 间接判定,不直接 strings.Contains。
const (
	BuiltinDirPrefix = "param-mappings/"
	CustomDirPrefix  = "param-mappings-custom/"
)

// BuiltinDirSubdir / CustomDirSubdir 是 XMLBaseDir 下的子目录名(不含末尾 /)。
// Loader 扫描目录、Handler Upload 写入目标路径、deploy.sh 初始化目录均引用这两个常量;
// 与 *DirPrefix 保持一致(后者多一个 / 用于字符串前缀匹配)。
const (
	BuiltinDirSubdir = "param-mappings"        // 镜像层只读 builtin XML 目录
	CustomDirSubdir  = "param-mappings-custom" // host bind mount 持久化 custom XML 目录
)

// ClassifySource 根据 param_models.loaded_from 列值判定物理来源。
//
// 输入约定:loaded_from 应为相对 XMLBaseDir 的 slash 路径
// (Loader 用 filepath.ToSlash 规范化,Windows 反斜杠也兼容)。
//
// 行为:
//   - 以 CustomDirPrefix 开头 → SourceCustom
//   - 以 BuiltinDirPrefix 开头 → SourceBuiltin
//   - 其他(空串 / 裸文件名 / 路径遍历模式)→ SourceUnknown
//
// SourceUnknown 包括历史数据(T-0098 P1-06 入库时未带前缀,
// 由 T-0178 数据迁移回填 builtin/ 前缀;迁移前 / 跨重启过渡期可能见到)。
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

// IsDeletable 是 DELETE /param-models/:name 端点与前端 deletable 字段的
// 唯一判定函数。
//
// 当前规则(2026-06-04 用户决策:内置数据不可删除):仅 custom 来源可删,
// builtin(当前目录 XML 加载的内置数据)与 unknown 一律不可删。
// ⚠️ 注:2026-06-03 起上传直接写 builtin 目录,故上传文件也判为 builtin →
// 同样不可删(只能重新上传同名覆盖)。若需"上传可删、出厂锁定",应把上传分流到 custom 目录。
//
// 调用方:
//   - DeleteModel 入口守门(builtin/unknown → 403 ErrCodeParamModelBuiltinNotDeletable)
//   - List/Detail DTO 的 deletable 字段填值
//   - 前端不重新推导,直接渲染 deletable bool
func IsDeletable(loadedFrom string) bool {
	return ClassifySource(loadedFrom) == SourceCustom
}
