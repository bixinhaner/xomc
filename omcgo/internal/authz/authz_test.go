package authz

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// fakeReader 用固定的设备→分组映射模拟 GroupReader。
type fakeReader struct {
	groups []uuid.UUID
	err    error
}

func (f fakeReader) GetDeviceGroupIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return f.groups, f.err
}

func TestAuthorizeDeviceAccess(t *testing.T) {
	g1, g2, g3 := uuid.New(), uuid.New(), uuid.New()
	dev := uuid.New()
	reader := fakeReader{groups: []uuid.UUID{g2}}

	tests := []struct {
		name          string
		reader        GroupReader
		visibleGroups []uuid.UUID
		wantErr       error
	}{
		{"超管 nil 放行", reader, nil, nil},
		{"reader 未注入放行", nil, []uuid.UUID{g1}, nil},
		{"空可见组 fail-closed", reader, []uuid.UUID{}, commonerrors.ErrForbidden},
		{"有交集放行", reader, []uuid.UUID{g1, g2}, nil},
		{"无交集拒绝", reader, []uuid.UUID{g1, g3}, commonerrors.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AuthorizeDeviceAccess(context.Background(), tt.reader, dev, tt.visibleGroups)
			if err != tt.wantErr {
				t.Fatalf("AuthorizeDeviceAccess() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorizeDeviceAccess_UngroupedDeviceDenied(t *testing.T) {
	// 设备未分组（reader 返回空）但调用者有具体可见组 → 越权。
	err := AuthorizeDeviceAccess(context.Background(), fakeReader{groups: nil}, uuid.New(), []uuid.UUID{uuid.New()})
	if err != commonerrors.ErrForbidden {
		t.Fatalf("ungrouped device should be forbidden, got %v", err)
	}
}

func TestApplyDeviceVisibilityFilter(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()

	t.Run("nil 不过滤", func(t *testing.T) {
		sql, args, err := ApplyDeviceVisibilityFilter(
			storage.Psql.Select("*").From("alarms_active"), "alarms_active.device_id", nil).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(sql, "device_group_members") || len(args) != 0 {
			t.Fatalf("nil visibleGroups should not filter: sql=%q args=%v", sql, args)
		}
	})

	t.Run("空集 fail-closed", func(t *testing.T) {
		sql, _, err := ApplyDeviceVisibilityFilter(
			storage.Psql.Select("*").From("alarms_active"), "alarms_active.device_id", []uuid.UUID{}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "FALSE") {
			t.Fatalf("empty visibleGroups should be WHERE FALSE: sql=%q", sql)
		}
	})

	t.Run("子查询过滤 + 占位符重排", func(t *testing.T) {
		sql, args, err := ApplyDeviceVisibilityFilter(
			storage.Psql.Select("*").From("alarms_active"), "alarms_active.device_id", []uuid.UUID{g1, g2}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "alarms_active.device_id IN (SELECT device_id FROM device_group_members WHERE group_id IN ($1,$2))") {
			t.Fatalf("unexpected subquery sql: %q", sql)
		}
		if len(args) != 2 {
			t.Fatalf("expected 2 args (g1,g2), got %v", args)
		}
	})

	t.Run("与其它 WHERE 共存占位符连续", func(t *testing.T) {
		// 先加一个普通 WHERE，再叠加可见性过滤，断言 $N 连续不冲突。
		b := storage.Psql.Select("*").From("alarms_active").Where("alarms_active.carrier = ?", "cmcc")
		sql, args, err := ApplyDeviceVisibilityFilter(b, "alarms_active.device_id", []uuid.UUID{g1}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") {
			t.Fatalf("placeholders should be sequential: %q", sql)
		}
		if len(args) != 2 {
			t.Fatalf("expected 2 args (carrier, g1), got %v", args)
		}
	})
}

func TestApplyDeviceSNVisibilityFilter(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()

	t.Run("nil 不过滤", func(t *testing.T) {
		sql, args, err := ApplyDeviceSNVisibilityFilter(
			storage.Psql.Select("*").From("pm_metrics"), "device_sn", nil).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(sql, "devices") || len(args) != 0 {
			t.Fatalf("nil visibleGroups should not filter: sql=%q args=%v", sql, args)
		}
	})

	t.Run("空集 fail-closed", func(t *testing.T) {
		sql, _, err := ApplyDeviceSNVisibilityFilter(
			storage.Psql.Select("*").From("pm_metrics"), "device_sn", []uuid.UUID{}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "FALSE") {
			t.Fatalf("empty visibleGroups should be WHERE FALSE: sql=%q", sql)
		}
	})

	t.Run("两层子查询 device_sn → devices → 组成员", func(t *testing.T) {
		sql, args, err := ApplyDeviceSNVisibilityFilter(
			storage.Psql.Select("*").From("pm_metrics"), "device_sn", []uuid.UUID{g1, g2}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		want := "device_sn IN (SELECT serial_number FROM devices WHERE id IN (SELECT device_id FROM device_group_members WHERE group_id IN ($1,$2)))"
		if !strings.Contains(sql, want) {
			t.Fatalf("unexpected sn subquery sql: %q", sql)
		}
		if len(args) != 2 {
			t.Fatalf("expected 2 args (g1,g2), got %v", args)
		}
	})
}

func TestApplyGroupVisibilityFilter(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()

	t.Run("nil 不过滤", func(t *testing.T) {
		sql, args, err := ApplyGroupVisibilityFilter(
			storage.Psql.Select("*").From("pm_group_metrics_hourly"), "device_group_id", nil).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(sql, "device_group_id") || len(args) != 0 {
			t.Fatalf("nil visibleGroups should not filter: sql=%q args=%v", sql, args)
		}
	})

	t.Run("空集 fail-closed", func(t *testing.T) {
		sql, _, err := ApplyGroupVisibilityFilter(
			storage.Psql.Select("*").From("pm_group_metrics_hourly"), "device_group_id", []uuid.UUID{}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "FALSE") {
			t.Fatalf("empty visibleGroups should be WHERE FALSE: sql=%q", sql)
		}
	})

	t.Run("组 id 直接取交", func(t *testing.T) {
		sql, args, err := ApplyGroupVisibilityFilter(
			storage.Psql.Select("*").From("pm_group_metrics_hourly"), "device_group_id", []uuid.UUID{g1, g2}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "device_group_id IN ($1,$2)") {
			t.Fatalf("unexpected group filter sql: %q", sql)
		}
		if len(args) != 2 {
			t.Fatalf("expected 2 args (g1,g2), got %v", args)
		}
	})
}

func TestVisibleSNSubquerySQL(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()

	t.Run("nil 不过滤", func(t *testing.T) {
		sql, groups := VisibleSNSubquerySQL("m.device_sn", "$3", nil)
		if sql != "" || groups != nil {
			t.Fatalf("nil should yield empty sql/groups, got sql=%q groups=%v", sql, groups)
		}
	})

	t.Run("空集 fail-closed", func(t *testing.T) {
		sql, groups := VisibleSNSubquerySQL("m.device_sn", "$3", []uuid.UUID{})
		if sql != "FALSE" || groups != nil {
			t.Fatalf("empty should yield FALSE/nil, got sql=%q groups=%v", sql, groups)
		}
	})

	t.Run("带 paramRef 的裸 SQL 片段", func(t *testing.T) {
		sql, groups := VisibleSNSubquerySQL("m.device_sn", "$3", []uuid.UUID{g1, g2})
		want := "m.device_sn IN (SELECT serial_number FROM devices WHERE id IN (SELECT device_id FROM device_group_members WHERE group_id = ANY($3)))"
		if sql != want {
			t.Fatalf("unexpected sql: %q", sql)
		}
		if len(groups) != 2 {
			t.Fatalf("expected 2 groups returned, got %v", groups)
		}
	})
}
