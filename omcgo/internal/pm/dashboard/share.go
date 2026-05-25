package dashboard

import "github.com/google/uuid"

// CanRead 校验 user 是否可读 dashboard：owner or 在 shared_with 列表里。
func CanRead(d *Dashboard, userID uuid.UUID) bool {
	if d.OwnerID == userID {
		return true
	}
	for _, u := range d.SharedWith {
		if u == userID {
			return true
		}
	}
	return false
}

// CanWrite 校验 user 是否可写（修改 / 删除 / 分享）：仅 owner。
//
// 分享出去的 dashboard 是只读视图；接收方要修改请走 Fork 派生自己的版本。
func CanWrite(d *Dashboard, userID uuid.UUID) bool {
	return d.OwnerID == userID
}
