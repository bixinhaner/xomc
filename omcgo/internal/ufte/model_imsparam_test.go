package ufte

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type previewErrorImsParamDispatcher struct {
	err error
}

func (d *previewErrorImsParamDispatcher) DispatchImsParamByFileID(
	context.Context, []string, uuid.UUID, string, uuid.UUID,
) (uuid.UUID, map[string]string, error) {
	return uuid.Nil, nil, nil
}

func (d *previewErrorImsParamDispatcher) PreviewImsParamFile(
	context.Context, uuid.UUID,
) (string, string, error) {
	return "", "", d.err
}

func TestCreateImsParamDistributeTaskValidatesFileBeforePlaceholder(t *testing.T) {
	typeDef, ok := findTaskTypeByCode(builtInTaskTypes(), "IMS_FILE_DISTRIBUTE")
	require.True(t, ok)
	fileID := uuid.New()
	service := &Service{
		imsParamDispatcher: &previewErrorImsParamDispatcher{err: commonerrors.ErrNotFound},
	}

	_, err := service.createImsParamDistributeTask(context.Background(), &typeDef, CreateTaskRequest{
		TaskName:  "missing IMS file",
		TypeCode:  typeDef.TypeCode,
		DeviceIDs: []uuid.UUID{uuid.New()},
		ParamType: "FT_ImsCore_User_Setting_UD",
		FileID:    &fileID,
	}, "tester", false, nil)

	require.ErrorIs(t, err, commonerrors.ErrNotFound)
}

// resolveTaskType 必须能从带文件类型尾段的落库值回找 IMS 模板
// （"Ims File:FT_ImsCore_*" → IMS_FILE_COLLECT，
// "IMS_FILE_DISTRIBUTE:FT_ImsCore_*" → IMS_FILE_DISTRIBUTE），
// 且不影响存量类型的精确匹配行为。
func TestResolveTaskType_ImsParamSuffixStrip(t *testing.T) {
	catalog := builtInTaskTypes()

	cases := []struct {
		name         string
		fileType     string
		wantTypeCode string
	}{
		{"ims collect exact", "ImsCore Parameters File", "IMS_FILE_COLLECT"},
		{"ims collect with param tail", "ImsCore Parameters File:FT_ImsCore_User_Setting_UD", "IMS_FILE_COLLECT"},
		// 历史 "Ims File" 短字面值的存量任务行反查 miss → 走 fallback（dev 已无此类数据；
		// transfer_router 仍认 IMS FILE 前缀保证 resume 不写错表）。
		{"ims distribute exact", "IMS_FILE_DISTRIBUTE", "IMS_FILE_DISTRIBUTE"},
		{"ims distribute with param tail", "IMS_FILE_DISTRIBUTE:FT_ImsCore_Cdr_U", "IMS_FILE_DISTRIBUTE"},
		{"config backup exact unchanged", "10 {OUI} Configuration File", "CONFIG_BACKUP_XML"},
		{"config restore exact unchanged", "10 <OUI> Configuration File", "CONFIG_RESTORE"},
		{"license exact unchanged", "License File", "LICENSE_UPGRADE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolveTaskType(catalog, software.TaskTypeLogCollect, "", tc.fileType)
			if !ok {
				t.Fatalf("resolveTaskType(%q) not found", tc.fileType)
			}
			if got.TypeCode != tc.wantTypeCode {
				t.Fatalf("resolveTaskType(%q) = %s, want %s", tc.fileType, got.TypeCode, tc.wantTypeCode)
			}
		})
	}
}

func TestImsParamTypeFromStoredFileType(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ImsCore Parameters File:FT_ImsCore_User_Setting_UD", "FT_ImsCore_User_Setting_UD"},
		{"IMS_FILE_DISTRIBUTE:FT_ImsCore_Recovery_D", "FT_ImsCore_Recovery_D"},
		{"ImsCore Parameters File:FT_ImsCore_Unknown", ""}, // 非法尾段
		{"ImsCore Parameters File:", ""},
		{"ImsCore Parameters File", ""},
		{"10 {OUI} Configuration File", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := imsParamTypeFromStoredFileType(tc.in); got != tc.want {
			t.Fatalf("imsParamTypeFromStoredFileType(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderImsParamPlaceholders(t *testing.T) {
	in := "/smallcell/FileUploadService?fileType=IMS_FILE&paramType={paramType}&sn={sn}&taskId={taskId}&filename="
	want := "/smallcell/FileUploadService?fileType=IMS_FILE&paramType=FT_ImsCore_User_Apn_Setting_UD&sn={sn}&taskId={taskId}&filename="
	if got := renderImsParamPlaceholders(in, "FT_ImsCore_User_Apn_Setting_UD"); got != want {
		t.Fatalf("renderImsParamPlaceholders = %q, want %q", got, want)
	}
	if got := renderImsParamPlaceholders(in, ""); got != in {
		t.Fatalf("empty paramType must not mutate template, got %q", got)
	}
	if got := renderImsParamPlaceholders("", "FT_ImsCore_Ue_Route_Setting_UD"); got != "" {
		t.Fatalf("empty template must stay empty, got %q", got)
	}
}
