package definition

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Upload XML 处理共享常量与校验器(严格对标 T-0180 indicator/upload.go;告警侧为
// 扁平目录,无 tech 段)。所有 helper 均纯函数无 IO,可独立单测。

// MaxUploadXMLSize 单文件上限 1 MiB。生产 alarm XML 单 ne_type ≤ ~200KB,给余量。
const MaxUploadXMLSize int64 = 1 << 20

// MaxNeTypeLen 与 alarm_definitions.ne_type varchar(16) 对齐(migrations/000001)。
// 超长 neType 不在守门链拒绝的话,Loader 重载该文件事务必失败,形成
// "上传 201 但 0 行入库"的静默失败(#123)。
const MaxNeTypeLen = 16

// validateUploadNeType 校验文件内容里的 neType 长度 ≤ MaxNeTypeLen(varchar 按字符计,
// 用 rune 数)。空值由调用方先行拒绝,这里只管超长。
func validateUploadNeType(neType string) error {
	if n := utf8.RuneCountInString(neType); n > MaxNeTypeLen {
		return fmt.Errorf("neType %q length %d exceeds max %d (alarm_definitions.ne_type is varchar(%d))",
			neType, n, MaxNeTypeLen, MaxNeTypeLen)
	}
	return nil
}

// uploadFilenamePattern 限制文件名为 [A-Za-z0-9_-] + ".xml",长度 1-64。
// 拒绝路径分隔符 / 点开头 / 空白 / 多扩展名 / 超长名(防遍历 / dotfile / 混淆 / DoS)。
var uploadFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}\.xml$`)

// validateUploadFilename 是 Upload handler 入口处的文件名守门。输入应为已 Base 的 basename。
func validateUploadFilename(name string) error {
	if name == "" {
		return fmt.Errorf("filename empty")
	}
	if !uploadFilenamePattern.MatchString(name) {
		return fmt.Errorf("filename %q does not match %s "+
			"(allowed: 1-64 chars of [A-Za-z0-9_-] + \".xml\")",
			name, uploadFilenamePattern.String())
	}
	return nil
}

// validateUploadXML 验证上传字节流是合法的 alarm XML:
//   - encoding/xml.Decoder Strict=true:well-formedness 严格(拒绝半截 / mismatched tag)
//   - Go 标准库默认不处理外部实体,无 XXE 风险
//   - 根元素必须 <alarmModel>(namespace 不限,与 Loader 对齐)
func validateUploadXML(raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("upload body empty")
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = true
	rootSeen := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			if !rootSeen {
				return fmt.Errorf("no root element found in upload")
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid xml: %w", err)
		}
		if rootSeen {
			continue
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "alarmModel" {
			return fmt.Errorf("root element must be <alarmModel>, got <%s>", start.Name.Local)
		}
		rootSeen = true
	}
}

// pathContainedIn 验证 target 物理路径在 base 目录树之下(二次防路径遍历)。
func pathContainedIn(base, target string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// readAllLimited 是 io.ReadAll 的限长版本,防 multipart header 声明小但实际 body 大的攻击。
func readAllLimited(r io.Reader, max int64) ([]byte, error) {
	limited := io.LimitReader(r, max+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > max {
		return nil, fmt.Errorf("upload exceeds max %d bytes", max)
	}
	return buf, nil
}

// uniqueSuffix 生成 tmp 文件唯一后缀(纳秒时间戳)。
func uniqueSuffix() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
