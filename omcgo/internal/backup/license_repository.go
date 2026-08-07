package backup

import "context"

// LicenseRepository 是 device_licenses 表的契约。
// 设计与 SnapshotRepository 完全对齐：一设备一行（serial_number 主键），冲突覆盖。
type LicenseRepository interface {
	Upsert(ctx context.Context, lic *DeviceLicense) error
	GetBySerialNumber(ctx context.Context, sn string) (*DeviceLicense, error)
	BatchGetBySerialNumbers(ctx context.Context, sns []string) (map[string]*DeviceLicense, error)
	ClaimAutoDispatch(ctx context.Context, sn string) (bool, error)
	ReleaseAutoDispatch(ctx context.Context, sn string) error
	List(ctx context.Context, filter LicenseFilter) (items []DeviceLicense, total int64, err error)
	Delete(ctx context.Context, sn string) error
	BatchDelete(ctx context.Context, sns []string) (deleted []string, err error)
}
