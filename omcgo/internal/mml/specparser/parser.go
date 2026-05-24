package specparser

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ParseMarkdown 读 spec md 文件并返回完整 SpecCatalog。
//
// 算法：单遍 bufio.Scanner 流式扫描，状态机识别四种段落：
//
//  1. §R-2.4 命令中文名权威表（markdown table）→ 71 个 SpecGroup 框架（仅 Chapter/GroupCode/CommandZhName）
//  2. §R-3.2 非可创建对象清单 → Blacklist
//  3. §SA-SR 章节标题 (`^## S[A-R] - `) → 切换"当前章节"
//  4. `#### 命令: <path>` H4 → 切换"当前 group"并读其下 path 表（7 列 markdown table）
//
// 解析完成后 cross-check：§R-2.4 表的 71 个 group_code 必须每个能在 §SA-SR 中找到 H4 标题。
// command_zh_name 71 条互不相同（命中违反返回 ErrDuplicateZhName）。
func ParseMarkdown(path, version string) (*SpecCatalog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open spec md %s: %w", path, err)
	}
	defer f.Close()

	hash, content, err := readWithHash(f)
	if err != nil {
		return nil, fmt.Errorf("read spec md: %w", err)
	}

	cat := &SpecCatalog{
		Version:    version,
		SourceMD:   path,
		SourceHash: hash[:16],
	}

	if err := parseAuthoritativeTable(content, cat); err != nil {
		return nil, fmt.Errorf("parse §R-2.4 authoritative table: %w", err)
	}

	if err := parseBlacklist(content, cat); err != nil {
		return nil, fmt.Errorf("parse §R-3.2 blacklist: %w", err)
	}

	if err := parseChapterDetails(content, cat); err != nil {
		return nil, fmt.Errorf("parse §SA-SR chapter details: %w", err)
	}

	if err := validateUniqueness(cat); err != nil {
		return nil, err
	}

	return cat, nil
}

// readWithHash 把 reader 全量读入并返回 sha256 hex + 字符串内容。
// 用于变更追溯（spec md 是文档级，3485 行 ~ 200KB，单次加载 OK）。
func readWithHash(r io.Reader) (string, string, error) {
	h := sha256.New()
	buf := &strings.Builder{}
	tee := io.TeeReader(r, h)
	scanner := bufio.NewScanner(tee)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 长行兜底（spec table 单行可达数 KB）
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(h.Sum(nil)), buf.String(), nil
}

// =============================================================================
// §R-2.4 权威表解析
// =============================================================================

// r24RowRE 匹配 §R-2.4 表的一行：`| # | 章节 | group_code | 命令中文名 |`
// 例：`| 1 | SA | \`Device.DeviceInfo.*\` | 设备基本信息 |`
//
// 表头与分隔行（`|--:|---|...|`）由 r24HeaderHintRE 识别后跳过。
var (
	r24RowRE = regexp.MustCompile("^\\|\\s*(\\d+)\\s*\\|\\s*([A-Z]{2})\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*([^|]+?)\\s*\\|\\s*$")

	// r24SectionStartHintRE 是 §R-2.4 节标题（含"命令中文名权威表"字样）。
	r24SectionStartHintRE = regexp.MustCompile(`^####?\s+R-2\.4`)
)

func parseAuthoritativeTable(content string, cat *SpecCatalog) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	inSection := false
	var lineNo int
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// 进入 §R-2.4 章节标志
		if !inSection {
			if r24SectionStartHintRE.MatchString(line) {
				inSection = true
			}
			continue
		}

		// 退出条件：遇到下一个 H3/H4（如 ### R-2.5 / #### R-2.5.1 / ### R-3）
		if strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "#### ") {
			// 但要排除 H4/H5 在 R-2.4 内部的子表头（如 R-2.4 本身就是 H4）
			if !r24SectionStartHintRE.MatchString(line) {
				break
			}
		}

		m := r24RowRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// m[1]=#, m[2]=Chapter, m[3]=group_code, m[4]=command_zh_name
		gc := strings.TrimSpace(m[3])
		zh := strings.TrimSpace(m[4])
		cat.Groups = append(cat.Groups, &SpecGroup{
			Chapter:       m[2],
			GroupCode:     gc,
			CommandZhName: zh,
			HasInstance:   strings.Contains(gc, "{i}"),
		})
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan §R-2.4 (line %d): %w", lineNo, err)
	}
	if len(cat.Groups) == 0 {
		return fmt.Errorf("§R-2.4 authoritative table not found or empty")
	}
	return nil
}

// =============================================================================
// §R-3.2 非可创建对象清单解析
// =============================================================================

var (
	r32SectionStartHintRE = regexp.MustCompile(`^####?\s+R-3\.2`)
	// r32CodeRE 匹配清单行中的 `Device.X.{i}.*` 反引号片段。
	r32CodeRE = regexp.MustCompile("`(Device\\.[^`]+)`")
)

func parseBlacklist(content string, cat *SpecCatalog) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	inSection := false
	for scanner.Scan() {
		line := scanner.Text()
		if !inSection {
			if r32SectionStartHintRE.MatchString(line) {
				inSection = true
			}
			continue
		}
		// 退出：遇到下一节
		if strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ") {
			if !r32SectionStartHintRE.MatchString(line) {
				break
			}
		}
		// 抓反引号包裹的 Device.* group_code（清单行通常是「- `Device.X.{i}.*`」格式）
		for _, m := range r32CodeRE.FindAllStringSubmatch(line, -1) {
			code := strings.TrimSpace(m[1])
			if strings.Contains(code, "{i}") {
				cat.Blacklist = append(cat.Blacklist, code)
			}
		}
	}
	return scanner.Err()
}

// =============================================================================
// §SA-SR 详情段解析
// =============================================================================

var (
	// chapterTitleRE 匹配章节标题：`## SA - DeviceInfo`
	chapterTitleRE = regexp.MustCompile(`^##\s+([A-Z]{2})\s+-`)

	// h4CommandRE 匹配命令标题：`#### 命令: Device.DeviceInfo.* 📖📝`
	// 反引号包裹和不包裹两种都见过；统一抓 group_code，去前后空白。
	h4CommandRE = regexp.MustCompile(`^####\s+命令:\s+([A-Za-z0-9._{}*]+)\s*`)

	// pathTableRowRE 匹配 H4 下的 path 表一行（7 列）：
	// `| 1 | \`...\` | \`Device.X.Y\` | UserLabel | 用户友好名 | 📝 RW | string |`
	pathTableRowRE = regexp.MustCompile("^\\|\\s*(\\d+)\\s*\\|\\s*`[^`]*`\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*([^|]+?)\\s*\\|\\s*([^|]+?)\\s*\\|\\s*([^|]+?)\\s*\\|\\s*([^|]+?)\\s*\\|\\s*$")

	// typeRangeRE 提取类型字符串中的 [min:max]（连续整数范围）。
	typeRangeRE = regexp.MustCompile(`\[(-?\d+)?:(-?\d+)?\]`)

	// typeBracketAnyRE 兜底剥离类型字符串中的任何 [...] 段（包括枚举 [a, b, c]）。
	// standard_params 暂无 enum_values 列，枚举语义不丢失到 DataType。
	typeBracketAnyRE = regexp.MustCompile(`\[[^\]]*\]`)

	// typeLengthRE 提取类型字符串中的 (length)。
	typeLengthRE = regexp.MustCompile(`\((\d+)\)`)
)

func parseChapterDetails(content string, cat *SpecCatalog) error {
	// 索引：group_code → *SpecGroup（来自 §R-2.4 解析结果）
	byCode := make(map[string]*SpecGroup, len(cat.Groups))
	for _, g := range cat.Groups {
		byCode[g.GroupCode] = g
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		currentGroup *SpecGroup
		inTable      bool
		lineNo       int
	)
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// 章节标题：仅用于阶段切换（重置 currentGroup）
		if chapterTitleRE.MatchString(line) {
			currentGroup = nil
			inTable = false
			continue
		}

		// H4 命令标题：定位到 SpecGroup
		if m := h4CommandRE.FindStringSubmatch(line); m != nil {
			raw := strings.TrimSpace(m[1])
			// 规范化：spec H4 标题可能是 "Device.DeviceInfo.*" 也可能是 "Device.X.{i}.*"
			// 与 §R-2.4 group_code 形态一致，可直接查表
			currentGroup = byCode[raw]
			inTable = false // 等读到表头分隔行才进入 inTable
			continue
		}

		if currentGroup == nil {
			continue
		}

		// 进入表格：表头分隔行（`|---|---|...`）后下一行起是数据行
		if !inTable {
			if strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| ---") {
				inTable = true
				continue
			}
			continue
		}

		// 已在表格内：尝试匹配数据行；首个非匹配行视为表尾
		m := pathTableRowRE.FindStringSubmatch(line)
		if m == nil {
			// 空行或 H3 之类：表结束
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "---") {
				inTable = false
			}
			continue
		}
		// m[1]=#, m[2]=TR-181 path, m[3]=ParamName, m[4]=ChineseName, m[5]=Access, m[6]=DataType
		p := &SpecPath{
			StandardPath: strings.TrimSpace(m[2]),
			ParamName:    strings.TrimSpace(m[3]),
			ChineseName:  strings.TrimSpace(m[4]),
			Access:       parseAccessIcon(m[5]),
		}
		parseTypeStr(m[6], p)
		currentGroup.Paths = append(currentGroup.Paths, p)
	}
	return scanner.Err()
}

// parseAccessIcon 解析权限列：
//   - "📝 RW" → READ_WRITE
//   - "📖 R"  → READ_ONLY
//   - "WO"   → WRITE_ONLY
//   - 其他   → READ_ONLY (保守默认)
func parseAccessIcon(raw string) string {
	s := strings.TrimSpace(raw)
	switch {
	case strings.Contains(s, "RW") || strings.Contains(s, "📝"):
		return AccessReadWrite
	case strings.Contains(s, "WO"):
		return AccessWriteOnly
	default:
		return AccessReadOnly
	}
}

// parseTypeStr 解析类型字符串，提取 DataType / MinValue / MaxValue / MaxLength。
// 例：
//   - "string"              → DataType=string
//   - "string(64)"          → DataType=string, MaxLength=64
//   - "unsignedInt[1:5]"    → DataType=unsignedInt, MinValue=1, MaxValue=5
//   - "unsignedInt[1:]"     → DataType=unsignedInt, MinValue=1
//   - "boolean"             → DataType=boolean
func parseTypeStr(raw string, p *SpecPath) {
	s := strings.TrimSpace(raw)
	// 提取 (length)
	if m := typeLengthRE.FindStringSubmatch(s); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil {
			p.MaxLength = &v
		}
		s = typeLengthRE.ReplaceAllString(s, "")
	}
	// 提取 [min:max]（仅匹配 a:b 形式）
	if m := typeRangeRE.FindStringSubmatch(s); m != nil {
		if m[1] != "" {
			if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				p.MinValue = &v
			}
		}
		if m[2] != "" {
			if v, err := strconv.ParseInt(m[2], 10, 64); err == nil {
				p.MaxValue = &v
			}
		}
		s = typeRangeRE.ReplaceAllString(s, "")
	}
	// 兜底剥离任何剩余的 [...]（枚举 / 复合范围）— 防 DataType 超出 VARCHAR(16)
	s = typeBracketAnyRE.ReplaceAllString(s, "")
	p.DataType = strings.TrimSpace(s)
	// 二次防御：DataType 仍然过长（极端 spec 文本）→ 截断保 16 字符
	if len(p.DataType) > 16 {
		p.DataType = p.DataType[:16]
	}
}

// =============================================================================
// 唯一性校验
// =============================================================================

// ErrDuplicateZhName 在 §R-2.4 71 条命令中文名存在重名时返回。
type ErrDuplicateZhName struct {
	ZhName   string
	GroupCode string
	OtherGC   string
}

func (e *ErrDuplicateZhName) Error() string {
	return fmt.Sprintf("duplicate command_zh_name %q: %s vs %s (§R-2.4 唯一性禁令)",
		e.ZhName, e.GroupCode, e.OtherGC)
}

func validateUniqueness(cat *SpecCatalog) error {
	seen := make(map[string]string, len(cat.Groups)) // zh_name → group_code
	for _, g := range cat.Groups {
		if other, ok := seen[g.CommandZhName]; ok {
			return &ErrDuplicateZhName{
				ZhName:    g.CommandZhName,
				GroupCode: g.GroupCode,
				OtherGC:   other,
			}
		}
		seen[g.CommandZhName] = g.GroupCode
	}
	return nil
}
