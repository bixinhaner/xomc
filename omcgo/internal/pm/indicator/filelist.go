package indicator

import (
	"encoding/xml"
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

// ENBSubdirName 是 indicator-library 下唯一的制式子目录名。
// 目录调整(2026-06-05):仅 ENB 分子目录,GSM/GNB XML(出厂单文件 + 自定义上传)
// 全部落 indicator-library/ 根级,按文件名 / XML deviceType 属性分类制式。
const ENBSubdirName = "enb"

// rootBuiltinTechByName 是根级出厂单文件名 → tech 的固定映射(大小写不敏感比对)。
var rootBuiltinTechByName = map[string]string{
	"gsm.xml": "gsm",
	"gnb.xml": "gnb",
}

// resolveRootTechSources 收集 GSM/GNB 单制式 XML 源:扫描 indicator-library/ 根级
// *.xml,按 classifyRootIndicatorTech(出厂文件名映射 / XML deviceType 属性)过滤出
// 属于 tech 的文件。
//
// 目录调整(2026-06-05):取消 gsm/、gnb/ 子目录,自定义上传与出厂单文件同住根级,
// sidecar 判来源。deviceType 缺失 / 不可识别的根级文件不属于任何制式 → 跳过。
//
// 输入:
//   - base:   XMLBaseDir
//   - dirRel: 根级目录相对路径,如 "indicator-library"
//   - tech:   目标制式(gsm / gnb)
//
// 目录不存在容忍(返空);输出按 LoadedFrom 字典序稳定排序。
func resolveRootTechSources(base, dirRel, tech string) ([]fileSource, error) {
	files, err := scanXMLBasenamesOptional(filepath.Join(base, dirRel))
	if err != nil {
		return nil, fmt.Errorf("scan root dir %s: %w", dirRel, err)
	}
	var out []fileSource
	for _, name := range files {
		abs := filepath.Join(base, dirRel, name)
		if classifyRootIndicatorTech(abs, name) != tech {
			continue
		}
		out = append(out, fileSource{
			AbsPath:    abs,
			LoadedFrom: filepath.ToSlash(filepath.Join(dirRel, name)),
		})
	}
	sortFileSources(out)
	return out, nil
}

// classifyRootIndicatorTech 判定根级 XML 文件归属的制式(gsm / gnb;空 = 不可识别)。
// 优先出厂文件名固定映射(GSM.xml / GNB.xml,大小写不敏感);
// 其余文件读 XML <indicatorModel deviceType="..."> 属性(上传守门已要求 GSM/GNB
// 自定义 XML 必带 deviceType)。读失败 / 属性缺失 / 值不在白名单 → 返空跳过。
func classifyRootIndicatorTech(absPath, basename string) string {
	if tech, ok := rootBuiltinTechByName[strings.ToLower(basename)]; ok {
		return tech
	}
	dt, err := fileDeviceType(absPath)
	if err != nil || dt == "" {
		return ""
	}
	lower := strings.ToLower(dt)
	if _, ok := allowedFileTechs[lower]; !ok {
		return ""
	}
	return lower
}

// fileDeviceType 读取 XML 文件根元素 <indicatorModel> 的 deviceType 属性。
// 流式解析只取第一个 StartElement(typical ≤ 200 KB,且根元素在文件头,~µs 级)。
func fileDeviceType(absPath string) (string, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	dec := xml.NewDecoder(f)
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "indicatorModel" {
			return "", fmt.Errorf("root element is <%s>, not <indicatorModel>", start.Name.Local)
		}
		return strings.TrimSpace(attrValue(start.Attr, "deviceType")), nil
	}
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
