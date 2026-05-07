package admin

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// importMaxFileSize 限制上传文件大小，防止内存被大文件耗尽。
const importMaxFileSize = 5 * 1024 * 1024 // 5 MB

// importMaxRows 限制单次导入行数，避免长事务和长时间阻塞。
const importMaxRows = 5000

// importSheetName 是模板与解析的目标 sheet 名称。
const importSheetName = "Users"

// ImportUserError 描述某一行导入失败的原因。
type ImportUserError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ImportUserResult 是用户导入接口的响应载荷。
type ImportUserResult struct {
	Created int               `json:"created"`
	Failed  int               `json:"failed"`
	Errors  []ImportUserError `json:"errors,omitempty"`
}

// importTemplateHeader 是表头字段名。顺序与 ImportUsers 解析顺序绑定。
var importTemplateHeader = []string{"username", "password", "display_name", "email", "phone", "role_ids"}

// DownloadImportTemplate 返回一份 .xlsx 模板供前端下载。
// 模板包含表头 + 一行示例 + 字段说明 sheet。
func (h *Handler) DownloadImportTemplate(c *gin.Context) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	// 默认 Sheet1 → 重命名为 Users
	if err := f.SetSheetName("Sheet1", importSheetName); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("rename sheet: %w", err))
		return
	}

	// 表头
	for i, h := range importTemplateHeader {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(importSheetName, cell, h)
	}

	// 示例行
	example := []string{"alice", "P@ssw0rd!", "Alice Liu", "alice@op.cn", "13800000000", ""}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(importSheetName, cell, v)
	}

	// 第二个 sheet 写字段说明
	if _, err := f.NewSheet("说明"); err == nil {
		_ = f.SetCellValue("说明", "A1", "字段")
		_ = f.SetCellValue("说明", "B1", "必填")
		_ = f.SetCellValue("说明", "C1", "格式")
		notes := [][]any{
			{"username", "是", "3-32 位字母/数字/下划线/连字符"},
			{"password", "是", "至少 6 位"},
			{"display_name", "否", "用户展示名"},
			{"email", "否", "符合 RFC 5322"},
			{"phone", "否", "中国手机号 1xxxxxxxxxx"},
			{"role_ids", "否", "多个 UUID 用 | 分隔，如 uuid-1|uuid-2"},
		}
		for r, row := range notes {
			for col, v := range row {
				cell, _ := excelize.CoordinatesToCellName(col+1, r+2)
				_ = f.SetCellValue("说明", cell, v)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		commonerrors.AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("write xlsx: %w", err))
		return
	}

	c.Header("Content-Disposition", `attachment; filename="users_import_template.xlsx"`)
	c.Data(http.StatusOK,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		buf.Bytes())
}

// ImportUsers 解析 multipart/form-data 中的 xlsx 文件，逐行调用 service.CreateUser。
// 部分行失败不会阻塞其他行；最终返回汇总。
func (h *Handler) ImportUsers(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("read upload: %w: %w", err, commonerrors.ErrInvalidInput))
		return
	}
	defer file.Close()

	if header.Size > importMaxFileSize {
		commonerrors.AbortWithError(c, http.StatusRequestEntityTooLarge,
			fmt.Errorf("file too large (max %dMB): %w", importMaxFileSize/1024/1024, commonerrors.ErrInvalidInput))
		return
	}

	xf, err := excelize.OpenReader(file)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("open xlsx: %w: %w", err, commonerrors.ErrInvalidInput))
		return
	}
	defer func() { _ = xf.Close() }()

	// 优先用约定的 Users sheet，否则取第一个 sheet。
	sheet := importSheetName
	if idx, _ := xf.GetSheetIndex(sheet); idx < 0 {
		sheets := xf.GetSheetList()
		if len(sheets) == 0 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("no sheets in workbook: %w", commonerrors.ErrInvalidInput))
			return
		}
		sheet = sheets[0]
	}

	rows, err := xf.GetRows(sheet)
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("read rows: %w: %w", err, commonerrors.ErrInvalidInput))
		return
	}
	if len(rows) < 2 {
		// 仅表头或空表
		response.OK(c, ImportUserResult{Errors: []ImportUserError{}})
		return
	}

	result := ImportUserResult{Errors: []ImportUserError{}}
	dataRowCount := 0

	for rowIdx, row := range rows[1:] { // 跳过表头
		rowNum := rowIdx + 2 // Excel 行号从 1 开始且表头占第 1 行

		if isImportRowSkippable(row) {
			continue
		}

		dataRowCount++
		if dataRowCount > importMaxRows {
			result.Errors = append(result.Errors, ImportUserError{
				Row: rowNum, Message: fmt.Sprintf("超过单次导入上限 %d 行", importMaxRows),
			})
			result.Failed++
			break
		}

		req, parseErr := parseImportRow(row)
		if parseErr != nil {
			result.Errors = append(result.Errors, ImportUserError{Row: rowNum, Message: parseErr.Error()})
			result.Failed++
			continue
		}

		if _, err := h.service.CreateUser(c.Request.Context(), req); err != nil {
			result.Errors = append(result.Errors, ImportUserError{Row: rowNum, Message: err.Error()})
			result.Failed++
			continue
		}
		result.Created++
	}

	response.OK(c, result)
}

// isImportRowSkippable 跳过空行与注释行（首字段以 # 开头）。
func isImportRowSkippable(row []string) bool {
	if len(row) == 0 {
		return true
	}
	allEmpty := true
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		return true
	}
	first := strings.TrimSpace(row[0])
	return strings.HasPrefix(first, "#")
}

// parseImportRow 把一行单元格转为 CreateUserRequest。列序与 importTemplateHeader 对齐。
func parseImportRow(row []string) (CreateUserRequest, error) {
	get := func(i int) string {
		if i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	username := get(0)
	password := get(1)
	if username == "" || password == "" {
		return CreateUserRequest{}, fmt.Errorf("username/password 必填")
	}
	if len(password) < 6 {
		return CreateUserRequest{}, fmt.Errorf("password 长度须 >=6")
	}

	req := CreateUserRequest{
		Username:    username,
		Password:    password,
		DisplayName: get(2),
		Email:       get(3),
		Phone:       get(4),
	}

	if rolesStr := get(5); rolesStr != "" {
		ids, err := parseRoleIDs(rolesStr)
		if err != nil {
			return CreateUserRequest{}, fmt.Errorf("role_ids 解析失败: %w", err)
		}
		req.RoleIDs = ids
	}

	return req, nil
}

// parseRoleIDs 解析 "uuid1|uuid2|uuid3" 形式的多角色字段。
func parseRoleIDs(s string) ([]uuid.UUID, error) {
	parts := strings.Split(s, "|")
	ids := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid %q", p)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
