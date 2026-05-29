package indicator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileSource 是 Loader 扫描合并后的一条文件来源记录。
//
// AbsPath:  绝对路径,Loader 用其 os.ReadFile 解析 XML
// LoadedFrom: 相对 XMLBaseDir 的 slash 路径(带 builtin/custom 前缀),
//
//	供 ClassifySource / IsDeletable / 前端 deletable bool 三方共用
type fileSource struct {
	AbsPath    string
	LoadedFrom string
}

// resolveENBSources 合并 ENB 子目录双源 XML 文件列表(T-0180 P1.2)。
//
// 输入:
//   - base: 镜像层根目录,etc/omcgo/data
//   - builtinSubdir: 镜像层 ENB 子目录,如 "indicator-library/enb"
//   - customSubdir:  host bind mount ENB 子目录,如 "indicator-library-custom/enb"
//   - customWins:    同名文件冲突时 custom 是否胜出(D1 决策默认 true)
//
// 合并规则:
//  1. 扫描 builtin 与 custom 各自下的 *.xml(忽略 . 隐藏与子目录)
//  2. 以 basename 为合并键
//  3. customWins=true:同名 custom 胜出,builtin 同名被压制
//     customWins=false:同名 builtin 胜出,custom 同名被压制
//  4. 输出按 LoadedFrom 字典序稳定排序(便于测试断言 + 日志可重现)
//
// LoadedFrom 形如:
//
//	"indicator-library/enb/ALL.xml"           (builtin)
//	"indicator-library-custom/enb/MY.xml"    (custom)
//
// 错误处理:builtin 子目录不存在 → 返错(运维操作或镜像问题,需明示);
//
//	custom 子目录不存在 → 静默忽略(运维上传前 host 目录为空很正常)。
func resolveENBSources(base, builtinSubdir, customSubdir string, customWins bool) ([]fileSource, error) {
	builtinFiles, err := scanXMLBasenames(filepath.Join(base, builtinSubdir))
	if err != nil {
		return nil, fmt.Errorf("scan builtin enb dir %s: %w", builtinSubdir, err)
	}

	customFiles, err := scanXMLBasenamesOptional(filepath.Join(base, customSubdir))
	if err != nil {
		return nil, fmt.Errorf("scan custom enb dir %s: %w", customSubdir, err)
	}

	merged := make(map[string]fileSource, len(builtinFiles)+len(customFiles))

	// 1) 先填 builtin 全集
	for _, name := range builtinFiles {
		merged[name] = fileSource{
			AbsPath:    filepath.Join(base, builtinSubdir, name),
			LoadedFrom: filepath.ToSlash(filepath.Join(builtinSubdir, name)),
		}
	}

	// 2) 再叠加 custom(根据 customWins 决定是否覆盖)
	for _, name := range customFiles {
		_, dup := merged[name]
		if dup && !customWins {
			continue // builtin 胜出,跳过 custom 同名
		}
		merged[name] = fileSource{
			AbsPath:    filepath.Join(base, customSubdir, name),
			LoadedFrom: filepath.ToSlash(filepath.Join(customSubdir, name)),
		}
	}

	// 稳定排序输出(按 LoadedFrom)
	out := make([]fileSource, 0, len(merged))
	for _, src := range merged {
		out = append(out, src)
	}
	sortFileSources(out)
	return out, nil
}

// resolveSingleTechSources 合并 GSM/GNB 双源(builtin 单文件 + custom 子目录多文件,T-0180 P1.2)。
//
// 输入:
//   - base:           XMLBaseDir
//   - builtinFile:    builtin 单文件相对路径,如 "indicator-library/GSM.xml"
//   - customSubdir:   custom 子目录相对路径,如 "indicator-library-custom/gsm"
//
// 合并规则:
//  1. 若 builtin 单文件存在 → 加入结果
//  2. 若 custom 子目录存在 → 全部 *.xml 加入结果
//  3. 由于 builtin 是 "X.xml" 而 custom 是 "subdir/Y.xml",rel 路径不同,
//     不可能"同名覆盖" → CustomOverrides 在此函数无意义,故未列入入参
//
// 输出按 LoadedFrom 字典序稳定排序。
func resolveSingleTechSources(base, builtinFile, customSubdir string) ([]fileSource, error) {
	var out []fileSource

	// builtin 单文件(允许不存在,运维场景:用户只想用 custom)
	builtinAbs := filepath.Join(base, builtinFile)
	if _, err := os.Stat(builtinAbs); err == nil {
		out = append(out, fileSource{
			AbsPath:    builtinAbs,
			LoadedFrom: filepath.ToSlash(builtinFile),
		})
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat builtin file %s: %w", builtinFile, err)
	}

	// custom 子目录(允许不存在,运维未上传时常态)
	customFiles, err := scanXMLBasenamesOptional(filepath.Join(base, customSubdir))
	if err != nil {
		return nil, fmt.Errorf("scan custom dir %s: %w", customSubdir, err)
	}
	for _, name := range customFiles {
		out = append(out, fileSource{
			AbsPath:    filepath.Join(base, customSubdir, name),
			LoadedFrom: filepath.ToSlash(filepath.Join(customSubdir, name)),
		})
	}

	sortFileSources(out)
	return out, nil
}

// scanXMLBasenames 扫指定目录下的 *.xml 文件名(basename,不含路径),
// 忽略子目录与 . 开头隐藏文件。目录不存在返错。
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
		if !strings.EqualFold(filepath.Ext(e.Name()), ".xml") {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// scanXMLBasenamesOptional 等同 scanXMLBasenames,但目录不存在静默返空切片。
// 用于 custom 目录的乐观扫描(运维上传前为空很正常)。
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
