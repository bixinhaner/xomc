package license

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"
)

// ExportContext 是单条 license PDF / JSON 导出所需的聚合数据。
//
// T-0100-P4-A / PRD §5.3.4：导出"含 license 全字段 + 近 30 天审计 + 当前容量使用"。
// Quota 可空（无 enforcer 时 nil），RecentLogs 取最近 30 天前 50 条审计记录。
type ExportContext struct {
	License     *License
	Quota       *Quota
	RecentLogs  []LicenseLog // 与 LicenseLogRepository.ListByLicense 返回值类型一致（值切片）
	GeneratedAt time.Time
	GeneratedBy string // 用户 username，nil 时填 "system"
}

// WriteSingleLicensePDF 渲染单条 license PDF 到 w。
//
// 布局：A4 portrait，标题「License 详情报告」+ 4 段 Descriptions（基本信息 / 容量
// 使用 / 特性列表 / 近期审计）。失败返 error，调用方决定是否回 500。
//
// PDF 字体使用 fpdf 内置 Helvetica（不依赖外部字体文件，避免容器化部署时缺字体）。
// 中文字段名走 ASCII 标签 + 值原样输出（fpdf 内置 Helvetica 不支持非 ASCII 字符，
// 中文 license_name / notes 会丢失字形 → 用 latin1 fallback 兜底，可读但失真）。
// 后续 P4 增强：嵌入 Noto Sans CJK 子集解决中文显示，本期 MVP 不做（受时间盒限制）。
func WriteSingleLicensePDF(w io.Writer, ctx ExportContext) error {
	if ctx.License == nil {
		return fmt.Errorf("license is required for PDF export")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// 标题
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "License Detail Report", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated: %s by %s",
		ctx.GeneratedAt.Format(time.RFC3339), valueOr(ctx.GeneratedBy, "system")),
		"", 1, "C", false, 0, "")
	pdf.Ln(4)

	// Section 1: 基本信息
	writePDFSectionHeader(pdf, "1. Basic Info")
	rows := [][2]string{
		{"License Code", ctx.License.LicenseCode},
		{"License Name", asciiSafe(ctx.License.LicenseName)},
		{"Product", asciiSafe(ctx.License.ProductName)},
		{"Type", string(ctx.License.LicenseType)},
		{"Status", string(ctx.License.Status)},
		{"Issue Date", ctx.License.IssueDate.Format("2006-01-02")},
		{"Expiry Date", formatExpiry(ctx.License.ExpiryDate)},
		{"Licensor", asciiSafe(strPtrOrEmpty(ctx.License.Licensor))},
		{"Device Type", asciiSafe(strPtrOrEmpty(ctx.License.DeviceType))},
		{"Region", asciiSafe(strPtrOrEmpty(ctx.License.Region))},
	}
	writePDFKVRows(pdf, rows)

	// Section 2: 容量使用
	writePDFSectionHeader(pdf, "2. Capacity Usage")
	capRows := [][2]string{
		{"Max Devices", strconv.Itoa(ctx.License.MaxDevices)},
		{"Used Devices", strconv.Itoa(ctx.License.UsedDevices)},
		{"Grace Period (days)", strconv.Itoa(ctx.License.GracePeriodDays)},
	}
	if ctx.Quota != nil {
		capRows = append(capRows,
			[2]string{"Has Active License", strconv.FormatBool(ctx.Quota.HasActiveLicense)},
			[2]string{"Usage Ratio", fmt.Sprintf("%.2f%%", ctx.Quota.UsageRatio*100)},
			[2]string{"Days Remaining", daysRemainingLabel(ctx.Quota.DaysRemaining)},
		)
	}
	writePDFKVRows(pdf, capRows)

	// Section 3: Features
	writePDFSectionHeader(pdf, "3. Features (raw JSON)")
	pdf.SetFont("Courier", "", 8)
	pdf.MultiCell(0, 4, asciiSafe(string(ctx.License.Features)), "1", "L", false)
	pdf.Ln(2)

	// Section 4: 近期审计
	writePDFSectionHeader(pdf, "4. Recent Audit (last 30 days, max 50)")
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(40, 6, "Time", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 6, "Type", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 6, "Result", "1", 0, "C", true, 0, "")
	pdf.CellFormat(80, 6, "Summary", "1", 1, "C", true, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	if len(ctx.RecentLogs) == 0 {
		pdf.CellFormat(180, 6, "(no audit records)", "1", 1, "C", false, 0, "")
	}
	for i := range ctx.RecentLogs {
		log := ctx.RecentLogs[i]
		summary := summarizeDetails(log.Details)
		pdf.CellFormat(40, 6, log.CreatedAt.Format("01-02 15:04:05"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 6, asciiSafe(string(log.LogType)), "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 6, string(log.Result), "1", 0, "C", false, 0, "")
		pdf.CellFormat(80, 6, asciiSafe(summary), "1", 1, "L", false, 0, "")
	}

	return pdf.Output(w)
}

// WriteAllActiveCSV 渲染所有 active license 汇总 CSV 到 w（含 BOM 头方便 Excel UTF-8）。
//
// T-0100-P4-A / PRD §5.3.4 全量导出。列定义稳定，便于下游 ETL；任何字段拓展走
// 新列 append 而非中插，避免破坏既有消费者。
func WriteAllActiveCSV(w io.Writer, licenses []*License) error {
	// UTF-8 BOM 让 Excel 直接识别为 UTF-8（否则中文乱码）。
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("write BOM: %w", err)
	}

	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"id",
		"license_code",
		"license_name",
		"product_name",
		"license_type",
		"status",
		"max_devices",
		"used_devices",
		"grace_period_days",
		"issue_date",
		"expiry_date",
		"licensor",
		"device_type",
		"region",
		"notes",
		"created_at",
		"updated_at",
	}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, lic := range licenses {
		if err := cw.Write([]string{
			lic.ID.String(),
			lic.LicenseCode,
			lic.LicenseName,
			lic.ProductName,
			string(lic.LicenseType),
			string(lic.Status),
			strconv.Itoa(lic.MaxDevices),
			strconv.Itoa(lic.UsedDevices),
			strconv.Itoa(lic.GracePeriodDays),
			lic.IssueDate.Format(time.RFC3339),
			formatExpiry(lic.ExpiryDate),
			strPtrOrEmpty(lic.Licensor),
			strPtrOrEmpty(lic.DeviceType),
			strPtrOrEmpty(lic.Region),
			strPtrOrEmpty(lic.Notes),
			lic.CreatedAt.Format(time.RFC3339),
			lic.UpdatedAt.Format(time.RFC3339),
		}); err != nil {
			return fmt.Errorf("write csv row %s: %w", lic.LicenseCode, err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// ---- internal helpers ----

func writePDFSectionHeader(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetFillColor(40, 70, 120)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 7, title, "", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(1)
}

func writePDFKVRows(pdf *fpdf.Fpdf, rows [][2]string) {
	pdf.SetFont("Helvetica", "", 9)
	for _, row := range rows {
		pdf.CellFormat(50, 6, row[0], "1", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, row[1], "1", 1, "L", false, 0, "")
	}
	pdf.Ln(3)
}

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func formatExpiry(t *time.Time) string {
	if t == nil {
		return "(perpetual)"
	}
	return t.Format("2006-01-02")
}

func daysRemainingLabel(d int) string {
	if d < 0 {
		return "perpetual / unlimited"
	}
	return strconv.Itoa(d)
}

// asciiSafe 把非 ASCII 字符替换为 '?'，让 fpdf 内置 Helvetica 不爆字形。
//
// MVP 妥协：fpdf 默认字体不支持 CJK；中文 license_name / notes 会显示为 ?。
// P4 增强项（不在本期）：嵌入 Noto Sans CJK 子集字体让 PDF 直接渲染中文。
func asciiSafe(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r < 128 {
			out = append(out, byte(r))
		} else {
			out = append(out, '?')
		}
	}
	return string(out)
}

// summarizeDetails 从 license_log.details JSON 提取 summary 字段；缺省退化为
// 截断 80 字符的 raw JSON。仅 PDF 表格使用，CSV 不展开 details。
func summarizeDetails(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	const maxLen = 60
	s := string(raw)
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}
