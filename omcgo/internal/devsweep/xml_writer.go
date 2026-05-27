package devsweep

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// XMLWriter 把不支持的 standardPath 标注到 paramModel XML 文件 supported="false"。
//
// 设计:
//   - text-based 行级编辑保留原 XML 注释 / 缩进 / 换行(git diff 友好)
//   - 不解析 + 不重写整个 XML(避免 encoding/xml 重排导致 noise diff)
//   - 原子写(tmp + rename),保留原文件 mtime/perm
//   - 幂等:已 supported="false" 跳过,supported="true" 替换,缺省补 supported="false"
//
// 匹配规则(详见 §1):
//   1. 标行必须形如 <param ... /> 或 <param ... > (单行,本仓库 XML 全单行 entry)
//   2. 优先匹配 standardPath="..." 属性;无 standardPath 时 fallback name="..."
//   3. 路径完全等值匹配(不做前缀展开 — sweep-paths 拿到的是 leaf path)
type XMLWriter struct {
	BaseDir string // 如 /etc/omcgo/data/param-mappings
}

// NewXMLWriter 构造 XMLWriter。BaseDir 空串 → "data/param-mappings"。
func NewXMLWriter(baseDir string) *XMLWriter {
	if baseDir == "" {
		baseDir = "data/param-mappings"
	}
	return &XMLWriter{BaseDir: baseDir}
}

// XMLApplyOutcome 是 Apply 的返回 — 给 CLI 渲染。
type XMLApplyOutcome struct {
	FilePath          string   // 实际写入的 XML 文件绝对路径
	LinesModified     int      // 实际改的行数
	PathsRequested    []string // 调用方请求标注的全集
	PathsApplied      []string // 实际改了 XML 行的 path
	PathsAlreadyMarked []string // XML 中已 supported="false",skip
	PathsNotFound     []string // XML 中查无此 standardPath / name,skip
}

// ResolvePath 根据 paramModel 名拼出 XML 文件绝对路径。
// 文件不存在不报错 — Apply 阶段读时再报。
func (w *XMLWriter) ResolvePath(paramModelName string) string {
	return filepath.Join(w.BaseDir, paramModelName+".xml")
}

// Apply 把给定 standardPath 列表在指定 paramModel XML 文件中标 supported="false"。
//
// 返回 outcome 即使部分失败(只要主流程 IO 成功)— PathsNotFound / PathsAlreadyMarked
// 让 CLI 可以精准报告 "X 个标了, Y 个已是 false, Z 个 XML 里找不到"。
func (w *XMLWriter) Apply(paramModelName string, paths []string) (*XMLApplyOutcome, error) {
	if paramModelName == "" {
		return nil, fmt.Errorf("param_model name required")
	}
	xmlPath := w.ResolvePath(paramModelName)

	outcome := &XMLApplyOutcome{
		FilePath:       xmlPath,
		PathsRequested: append([]string(nil), paths...),
	}
	if len(paths) == 0 {
		return outcome, nil
	}

	raw, err := os.ReadFile(xmlPath)
	if err != nil {
		return outcome, fmt.Errorf("read XML %s: %w", xmlPath, err)
	}
	info, err := os.Stat(xmlPath)
	if err != nil {
		return outcome, fmt.Errorf("stat XML %s: %w", xmlPath, err)
	}

	pathSet := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		pathSet[p] = struct{}{}
	}
	appliedSet := make(map[string]struct{})
	alreadySet := make(map[string]struct{})

	lines := strings.Split(string(raw), "\n")
	modified := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "<param ") && !strings.HasPrefix(trimmed, "<object ") {
			continue
		}

		linePath, ok := extractParamPath(line)
		if !ok {
			continue
		}
		if _, want := pathSet[linePath]; !want {
			continue
		}

		newLine, changed, already := setSupportedFalse(line)
		if already {
			alreadySet[linePath] = struct{}{}
			continue
		}
		if !changed {
			continue
		}
		lines[i] = newLine
		modified++
		appliedSet[linePath] = struct{}{}
	}

	outcome.LinesModified = modified
	outcome.PathsApplied = mapKeysSorted(appliedSet)
	outcome.PathsAlreadyMarked = mapKeysSorted(alreadySet)
	for _, p := range paths {
		if _, ok := appliedSet[p]; ok {
			continue
		}
		if _, ok := alreadySet[p]; ok {
			continue
		}
		outcome.PathsNotFound = append(outcome.PathsNotFound, p)
	}

	if modified == 0 {
		return outcome, nil
	}

	newContent := strings.Join(lines, "\n")
	if err := writeAtomic(xmlPath, []byte(newContent), info.Mode().Perm()); err != nil {
		return outcome, fmt.Errorf("atomic write %s: %w", xmlPath, err)
	}
	return outcome, nil
}

// extractParamPath 从一行 <param .../> 里取 standardPath="P" 或回退 name="P"。
func extractParamPath(line string) (string, bool) {
	if p, ok := extractAttr(line, "standardPath"); ok && p != "" {
		return p, true
	}
	if p, ok := extractAttr(line, "name"); ok && p != "" {
		return p, true
	}
	return "", false
}

// extractAttr 取 `attr="value"` 形式的属性值。
func extractAttr(line, attr string) (string, bool) {
	// 匹配 attr="..."(允许属性间任意空白)。属性值不允许内嵌引号
	// (TR-069 XML 标准属性也确实不含引号,简化处理)。
	re := regexp.MustCompile(`(?:^|\s)` + regexp.QuoteMeta(attr) + `="([^"]*)"`)
	m := re.FindStringSubmatch(line)
	if len(m) == 2 {
		return m[1], true
	}
	return "", false
}

// setSupportedFalse 把 <param ... /> 一行的 supported 属性置为 "false"。
// 返回:newLine, changed, alreadyFalse。
//
// 三种情况:
//  1. 已有 supported="false" → already=true, changed=false
//  2. 已有 supported="true" → 替换为 "false",changed=true
//  3. 没 supported 属性 → 在 /> 前插 ` supported="false"`,changed=true
func setSupportedFalse(line string) (string, bool, bool) {
	// 1. 已经 false → skip
	if reSupportedFalse.MatchString(line) {
		return line, false, true
	}
	// 2. 现存 supported="X" → 替换
	if reSupportedAny.MatchString(line) {
		return reSupportedAny.ReplaceAllString(line, `${1}supported="false"`), true, false
	}
	// 3. 在闭合标签前插入
	if idx := strings.LastIndex(line, "/>"); idx > 0 {
		return line[:idx] + ` supported="false"` + line[idx:], true, false
	}
	if idx := strings.LastIndex(line, ">"); idx > 0 {
		return line[:idx] + ` supported="false"` + line[idx:], true, false
	}
	return line, false, false
}

var (
	reSupportedFalse = regexp.MustCompile(`(?:^|\s)supported="false"`)
	// 捕获 leading whitespace + supported="X" 整体;ReplaceAll 保留 leading space
	reSupportedAny = regexp.MustCompile(`((?:^|\s))supported="[^"]*"`)
)

func mapKeysSorted(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// 简单字符串排序保证输出稳定 — 单测/diff 友好
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// writeAtomic 原子写文件 — 同 dir tmp + Sync + Rename。保留指定 perm。
func writeAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
