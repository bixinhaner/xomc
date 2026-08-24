package soap

import (
	"strings"
	"testing"
)

// IMS 核心网参数任务在 Upload/Download SOAP 报文里应渲染 <ParameterType> 标签；
// 非 IMS 任务（固件/备份/日志等）不带该标签，保证既有厂商 CPE 兼容。
// 见 docs/design/imscore-file-transfer.md 与 templates.go 的条件渲染。
func TestImsParamTypeTagRendering(t *testing.T) {
	upload, err := RenderResponse(UploadTmpl, UploadData{
		ID: "1", CommandKey: "k", FileType: "ImsCore Parameters File",
		URL: "http://acs/up", ParamType: "FT_ImsCore_Ims_User_Setting_UD",
	})
	if err != nil {
		t.Fatalf("render upload: %v", err)
	}
	if !strings.Contains(string(upload), "<ParameterType>FT_ImsCore_Ims_User_Setting_UD</ParameterType>") {
		t.Fatalf("upload with paramType should render <ParameterType>, got:\n%s", upload)
	}

	uploadNoParam, err := RenderResponse(UploadTmpl, UploadData{
		ID: "1", CommandKey: "k", FileType: "6", URL: "http://acs/up",
	})
	if err != nil {
		t.Fatalf("render upload (no param): %v", err)
	}
	if strings.Contains(string(uploadNoParam), "<ParameterType>") {
		t.Fatalf("upload without paramType must not render <ParameterType>, got:\n%s", uploadNoParam)
	}

	download, err := RenderResponse(DownloadTmpl, DownloadData{
		ID: "1", CommandKey: "k", FileType: "ImsCore Parameters File",
		URL: "http://acs/dl", TargetFileName: "f.dat", ParamType: "FT_ImsCore_Ue_Route_UD",
	})
	if err != nil {
		t.Fatalf("render download: %v", err)
	}
	if !strings.Contains(string(download), "<ParameterType>FT_ImsCore_Ue_Route_UD</ParameterType>") {
		t.Fatalf("download with paramType should render <ParameterType>, got:\n%s", download)
	}

	downloadLog, err := RenderResponse(DownloadTmpl, DownloadData{
		ID: "1", CommandKey: "k", FileType: "Ims Log File",
		URL: "http://acs/dl", TargetFileName: "f.log", LogType: "FT_ImsCore_Operation_Logs_U",
	})
	if err != nil {
		t.Fatalf("render download (log): %v", err)
	}
	if !strings.Contains(string(downloadLog), "<LogType>FT_ImsCore_Operation_Logs_U</LogType>") {
		t.Fatalf("download with logType should render <LogType>, got:\n%s", downloadLog)
	}
	if strings.Contains(string(downloadLog), "<ParameterType>") {
		t.Fatalf("log task must not render <ParameterType>, got:\n%s", downloadLog)
	}

	downloadNoParam, err := RenderResponse(DownloadTmpl, DownloadData{
		ID: "1", CommandKey: "k", FileType: "1 Firmware Upgrade Image",
		URL: "http://acs/dl", TargetFileName: "fw.bin",
	})
	if err != nil {
		t.Fatalf("render download (no param): %v", err)
	}
	if strings.Contains(string(downloadNoParam), "<ParameterType>") {
		t.Fatalf("download without paramType must not render <ParameterType>, got:\n%s", downloadNoParam)
	}
}
