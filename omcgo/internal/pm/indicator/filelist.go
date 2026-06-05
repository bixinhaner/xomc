package indicator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileSource 是 Loader 扫描后的一条文件来源记录。
//
// AbsPath:  绝对路径,Loader 用其 os.ReadFile 解析 XML
// LoadedFrom: 相对 XMLBaseDir 的 slash 路径,供 ClassifySource / IsDeletable /
//
//	前端 deletable bool 三方共用。来源(builtin/custom)由同目录 sidecar
//	(X.xml.custom)判定,见 source.go;loaded_from 本身不再带 builtin/custom 目录前缀。
type fileSource struct {
	AbsPath    string
	LoadedFrom string
}

// resolveENBSources 扫描 ENB 子目录单源 XML 文件列表。
//
// 三库 XML 导入重构(2026-06-04 单目录):取消 builtin/custom 双目录合并,
// 只扫 indicator-library/enb/*.xml(builtin + custom 同住,sidecar 文件已跳过)。
//
// 输入:
//   - base:   XMLBaseDir
//   - enbSubdir: ENB 子目录相对路径,如 "indicator-library/enb"
//
// LoadedFrom 形如 "indicator-library/enb/ALL.xml"。
// 子目录不存在 → 返错(运维操作或镜像问题,需明示)。
// 输出按 LoadedFrom 字典序稳定排序。
func resolveENBSources(base, enbSubdir string) ([]fileSource, error) {
	files, err := scanXMLBasenames(filepath.Join(base, enbSubdir))
	if err != nil {
		return nil, fmt.Errorf("scan enb dir %s: %w", enbSubdir, err)
	}
	out := make([]fileSource, 0, len(files))
	for _, name := range files {
		out = append(out, fileSource{
			AbsPath:    filepath.Join(base, enbSubdir, name),
			LoadedFrom: filepath.ToSlash(filepath.Join(enbSubdir, name)),
		})
	}
	sortFileSources(out)
	return out, nil
}

// resolveSingleTechSources 收集 GSM/GNB 单制式 XML 源:
// 根级单文件(indicator-library/GSM.xml | GNB.xml)+ 同制式子目录多文件
// (indicator-library/gsm/*.xml | indicator-library/gnb/*.xml)。
//
// 三库 XML 导入重构(2026-06-04 单目录):取消 *-custom 目录,custom 上传落地
// 同制式子目录(gsm/ / gnb/);根级单文件保持出厂随包。两者并存,sidecar 判来源。
//
// 输入:
//   - base:        XMLBaseDir
//   - builtinFile: 根级单文件相对路径,如 "indicator-library/GSM.xml"
//   - subdir:      同制式子目录相对路径,如 "indicator-library/gsm"
//
// 根级单文件不存在 / 子目录不存在均容忍(运维场景:只用其中一侧)。
// 输出按 LoadedFrom 字典序稳定排序。
func resolveSingleTechSources(base, builtinFile, subdir string) ([]fileSource, error) {
	var out []fileSource

	// 根级单文件(允许不存在)
	builtinAbs := filepath.Join(base, builtinFile)
	if _, err := os.Stat(builtinAbs); err == nil {
		out = append(out, fileSource{
			AbsPath:    builtinAbs,
			LoadedFrom: filepath.ToSlash(builtinFile),
		})
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat builtin file %s: %w", builtinFile, err)
	}

	// 同制式子目录(允许不存在,运维未上传时常态)
	subFiles, err := scanXMLBasenamesOptional(filepath.Join(base, subdir))
	if err != nil {
		return nil, fmt.Errorf("scan dir %s: %w", subdir, err)
	}
	for _, name := range subFiles {
		out = append(out, fileSource{
			AbsPath:    filepath.Join(base, subdir, name),
			LoadedFrom: filepath.ToSlash(filepath.Join(subdir, name)),
		})
	}

	sortFileSources(out)
	return out, nil
}

// scanXMLBasenames 扫指定目录下的 *.xml 文件名(basename,不含路径),
// 忽略子目录、. 开头隐藏文件、sidecar 标记文件(X.xml.custom)。目录不存在返错。
func scanXMLBasenames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// 跳过 sidecar 标记文件(X.xml.custom);其 ext 是 .custom 非 .xml,
		// 下面 EqualFold 已能排除,这里显式 continue 求清晰 + 安全。
		if strings.HasSuffix(e.Name(), CustomMarkerSuffix) {
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), ".xml") {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// scanXMLBasenamesOptional 等同 scanXMLBasenames,但目录不存在静默返空切片。
// 用于同制式子目录的乐观扫描(运维上传前为空很正常)。
func scanXMLBasenamesOptional(dir string) ([]string, error) {
	files, err := scanXMLBasenames(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return files, nil
}

// sortFileSources 按 LoadedFrom 字典序原地排序(稳定可重现日志/测试)。
func sortFileSources(srcs []fileSource) {
	sort.Slice(srcs, func(i, j int) bool {
		return srcs[i].LoadedFrom < srcs[j].LoadedFrom
	})
}
