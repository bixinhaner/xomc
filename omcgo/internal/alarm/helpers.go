package alarm

// strPtr 把字符串转为 *string；空字符串返回 nil（区分"未设置"与"空值"）。
//
// 多个文件复用（filter_engine / reboot_monitor / receiver / tr069_parser），
// T-0098-P5-06 把旧 library_service.go 删除后从那里抽出本助手。
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
