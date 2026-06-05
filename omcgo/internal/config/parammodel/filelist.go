package parammodel

import (
	"fmt"
	"path/filepath"
	"sort"
)

// resolveLoadedFrom 计算写入 param_models.loaded_from 的字段值。
//
// 返回相对 base 的 slash 分隔路径(Windows 反斜杠规范化);
// filepath.Rel 失败时降级为裸 basename(防御性,避免 Loader 崩溃)。
//
// 期望输出形如 "param-mappings/BTS.xml"。来源(builtin/custom)由同目录 sidecar
// (X.xml.custom)判定,见 source.go::ClassifySource / IsDeletable。
func resolveLoadedFrom(base, absPath string) string {
	rel, err := filepath.Rel(base, absPath)
	if err != nil {
		return filepath.Base(absPath)
	}
	return filepath.ToSlash(rel)
}

// resolveLoaderFiles 是 Loader.run 文件枚举步骤的纯函数封装。
//
// 三库 XML 导入重构(2026-06-04):单目录扫描,不再合并 custom 目录。
//   - 显式白名单(whitelist != nil):builtin 目录内按列表取文件
//   - 自动扫描(whitelist 空):扫 builtin 目录全部 *.xml(剔除 reserved + sidecar)
//
// builtin 目录错误 → 直接 error(builtin 应永远可读),输出按 basename 字典序排序。
func resolveLoaderFiles(builtinDir string, whitelist, reserved []string) ([]string, error) {
	if len(whitelist) > 0 {
		files := make([]string, 0, len(whitelist))
		for _, name := range whitelist {
			files = append(files, filepath.Join(builtinDir, name))
		}
		return files, nil
	}

	names, err := scanXMLFiles(builtinDir)
	if err != nil {
		return nil, fmt.Errorf("scan builtin %s: %w", builtinDir, err)
	}
	names = pruneReserved(names, reserved)
	sort.Strings(names)

	files := make([]string, 0, len(names))
	for _, name := range names {
		files = append(files, filepath.Join(builtinDir, name))
	}
	return files, nil
}
