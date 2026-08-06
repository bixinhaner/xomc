package authz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

// fakeReader 用固定的设备→分组映射模拟 GroupReader。
type fakeReader struct {
	groups []uuid.UUID
	err    error
}

func (f fakeReader) GetDeviceGroupIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return f.groups, f.err
}

type resolverPermission struct {
	err error
}

func (p resolverPermission) GetUserVisibleGroupIDs(
	context.Context,
	uuid.UUID,
	bool,
) ([]uuid.UUID, error) {
	return nil, p.err
}

func TestResolverPermissionBackendErrorsAreAlwaysInternal(t *testing.T) {
	tests := []struct {
		name         string
		backendError error
		rawText      string
	}{
		{
			name:         "plain error",
			backendError: errors.New("database password=do-not-leak"),
			rawText:      "database password=do-not-leak",
		},
		{
			name:         "forbidden sentinel",
			backendError: commonerrors.ErrForbidden,
			rawText:      commonerrors.ErrForbidden.Error(),
		},
		{
			name: "wrapped invalid input sentinel",
			backendError: fmt.Errorf(
				"permission query rejected: %w",
				commonerrors.ErrInvalidInput,
			),
			rawText: "permission query rejected",
		},
		{
			name: "wrapped unavailable sentinel",
			backendError: fmt.Errorf(
				"permission cache unavailable: %w",
				commonerrors.ErrUnavailable,
			),
			rawText: "permission cache unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Set(admin.CtxKeyUserID, uuid.New())
			c.Set(admin.CtxKeyIsSuperAdmin, false)
			resolver := NewResolver(resolverPermission{err: tt.backendError})

			_, err := resolver.ResolveFromContext(c)

			require.Error(t, err)
			require.ErrorIs(t, err, commonerrors.ErrInternal)
			require.NotErrorIs(t, err, commonerrors.ErrForbidden)
			require.NotErrorIs(t, err, commonerrors.ErrInvalidInput)
			require.NotErrorIs(t, err, commonerrors.ErrUnavailable)
			require.Contains(t, err.Error(), tt.rawText)

			okGroups, ok := resolver.FromContext(c)
			require.False(t, ok)
			require.Nil(t, okGroups)
			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			var envelope struct {
				Ret  int    `json:"ret"`
				Msg  string `json:"msg"`
				Data any    `json:"data"`
			}
			require.NoError(
				t,
				json.Unmarshal(recorder.Body.Bytes(), &envelope),
			)
			require.Zero(t, envelope.Ret)
			require.Nil(t, envelope.Data)
			require.Equal(t, commonerrors.ErrInternal.Error(), envelope.Msg)
			require.NotContains(t, recorder.Body.String(), tt.rawText)
		})
	}
}

func TestResolverMissingUserIDRemainsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	resolver := NewResolver(resolverPermission{
		err: errors.New("backend must not be reached"),
	})

	groups, ok := resolver.FromContext(c)

	require.False(t, ok)
	require.Nil(t, groups)
	require.Equal(t, http.StatusForbidden, recorder.Code)
	var envelope struct {
		Msg string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, commonerrors.ErrForbidden.Error(), envelope.Msg)
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

func TestAuthorizeDeviceAccess_DefaultGroupAllowsLegacyUngroupedDevice(t *testing.T) {
	defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
	err := AuthorizeDeviceAccess(context.Background(), fakeReader{groups: nil}, uuid.New(), []uuid.UUID{defaultGroup})
	if err != nil {
		t.Fatalf("default group should allow legacy ungrouped device, got %v", err)
	}
}

func TestAuthorizeDeviceAccess_DefaultGroupAllowsPhysicalMember(t *testing.T) {
	defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
	err := AuthorizeDeviceAccess(
		context.Background(),
		fakeReader{groups: []uuid.UUID{defaultGroup}},
		uuid.New(),
		[]uuid.UUID{defaultGroup},
	)
	if err != nil {
		t.Fatalf("default group should allow physical default-group member, got %v", err)
	}
}

func TestAuthorizeDeviceAccessByGrants(t *testing.T) {
	groupA, groupB := uuid.New(), uuid.New()
	grants := []model.DeviceVisibilityGrant{{GroupIDs: []uuid.UUID{groupA}, Technologies: []model.Technology{model.TechLTE}}}

	t.Run("匹配分组且制式命中时放行", func(t *testing.T) {
		err := AuthorizeDeviceAccessByGrants([]uuid.UUID{groupA, groupB}, model.TechLTE, grants)
		if err != nil {
			t.Fatalf("AuthorizeDeviceAccessByGrants() = %v, want nil", err)
		}
	})

	t.Run("分组命中但制式不符时拒绝", func(t *testing.T) {
		err := AuthorizeDeviceAccessByGrants([]uuid.UUID{groupA}, model.TechNR, grants)
		if err != commonerrors.ErrForbidden {
			t.Fatalf("AuthorizeDeviceAccessByGrants() = %v, want forbidden", err)
		}
	})

	t.Run("空技术列表视为不限制", func(t *testing.T) {
		err := AuthorizeDeviceAccessByGrants([]uuid.UUID{groupA}, model.TechNR, []model.DeviceVisibilityGrant{{GroupIDs: []uuid.UUID{groupA}, Technologies: []model.Technology{}}})
		if err != nil {
			t.Fatalf("AuthorizeDeviceAccessByGrants() = %v, want nil", err)
		}
	})

	t.Run("默认组真实成员且制式命中时放行", func(t *testing.T) {
		defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
		err := AuthorizeDeviceAccessByGrants(
			[]uuid.UUID{defaultGroup},
			model.TechLTE,
			[]model.DeviceVisibilityGrant{{
				GroupIDs:     []uuid.UUID{defaultGroup},
				Technologies: []model.Technology{model.TechLTE},
			}},
		)
		if err != nil {
			t.Fatalf("AuthorizeDeviceAccessByGrants() = %v, want nil", err)
		}
	})

	t.Run("默认组继续兼容历史未分组设备", func(t *testing.T) {
		defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
		err := AuthorizeDeviceAccessByGrants(
			nil,
			model.TechLTE,
			[]model.DeviceVisibilityGrant{{
				GroupIDs:     []uuid.UUID{defaultGroup},
				Technologies: []model.Technology{model.TechLTE},
			}},
		)
		if err != nil {
			t.Fatalf("AuthorizeDeviceAccessByGrants() = %v, want nil", err)
		}
	})

	t.Run("空 grant fail-closed", func(t *testing.T) {
		err := AuthorizeDeviceAccessByGrants([]uuid.UUID{groupA}, model.TechLTE, []model.DeviceVisibilityGrant{})
		if err != commonerrors.ErrForbidden {
			t.Fatalf("AuthorizeDeviceAccessByGrants() = %v, want forbidden", err)
		}
	})
}

func TestApplyDeviceVisibilityGrantsFilter(t *testing.T) {
	groupA, groupB := uuid.New(), uuid.New()

	t.Run("nil 不过滤", func(t *testing.T) {
		sql, args, err := ApplyDeviceVisibilityGrantsFilter(
			storage.Psql.Select("*").From("devices"), "d.id", "d.technology", nil).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(sql, "device_group_members") || len(args) != 0 {
			t.Fatalf("nil grants should not filter: sql=%q args=%v", sql, args)
		}
	})

	t.Run("空 grants fail-closed", func(t *testing.T) {
		sql, _, err := ApplyDeviceVisibilityGrantsFilter(
			storage.Psql.Select("*").From("devices"), "d.id", "d.technology", []model.DeviceVisibilityGrant{}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "FALSE") {
			t.Fatalf("empty grants should be WHERE FALSE: sql=%q", sql)
		}
	})

	t.Run("分组+制式子查询", func(t *testing.T) {
		sql, args, err := ApplyDeviceVisibilityGrantsFilter(
			storage.Psql.Select("*").From("devices"), "d.id", "d.technology",
			[]model.DeviceVisibilityGrant{{GroupIDs: []uuid.UUID{groupA}, Technologies: []model.Technology{model.TechLTE}}, {GroupIDs: []uuid.UUID{groupB}}}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "d.id IN (SELECT device_id FROM device_group_members WHERE group_id IN ($1))") {
			t.Fatalf("unexpected grant filter sql: %q", sql)
		}
		if !strings.Contains(sql, "d.technology IN ($2)") {
			t.Fatalf("expected technology predicate in sql: %q", sql)
		}
		if len(args) != 3 {
			t.Fatalf("expected 3 args, got %v", args)
		}
	})

	t.Run("空技术列表不加制式过滤", func(t *testing.T) {
		sql, args, err := ApplyDeviceVisibilityGrantsFilter(
			storage.Psql.Select("*").From("devices"), "d.id", "d.technology",
			[]model.DeviceVisibilityGrant{{GroupIDs: []uuid.UUID{groupA}, Technologies: []model.Technology{}}}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(sql, "FALSE") {
			t.Fatalf("empty technologies should not fail-closed: sql=%q", sql)
		}
		if strings.Contains(sql, "d.technology IN") {
			t.Fatalf("empty technologies should not add technology predicate: sql=%q", sql)
		}
		if len(args) != 1 {
			t.Fatalf("expected 1 arg, got %v", args)
		}
	})

	t.Run("默认组同时匹配真实成员和历史未分组设备", func(t *testing.T) {
		defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
		sql, args, err := ApplyDeviceVisibilityGrantsFilter(
			storage.Psql.Select("*").From("devices"), "d.id", "d.technology",
			[]model.DeviceVisibilityGrant{{
				GroupIDs:     []uuid.UUID{defaultGroup},
				Technologies: []model.Technology{model.TechLTE},
			}},
		).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "d.id IN (SELECT device_id FROM device_group_members WHERE group_id IN ($1))") {
			t.Fatalf("expected physical default-group predicate in sql: %q", sql)
		}
		if !strings.Contains(sql, "NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id)") {
			t.Fatalf("expected legacy ungrouped predicate in sql: %q", sql)
		}
		if len(args) != 2 || args[0] != defaultGroup || args[1] != model.TechLTE {
			t.Fatalf("expected default group and LTE args, got %v", args)
		}
	})
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

	t.Run("默认组同时匹配真实成员和历史未分组设备", func(t *testing.T) {
		defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
		sql, args, err := ApplyDeviceVisibilityFilter(
			storage.Psql.Select("*").From("alarms_active"), "alarms_active.device_id", []uuid.UUID{g1, defaultGroup}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "alarms_active.device_id IN (SELECT device_id FROM device_group_members WHERE group_id IN ($1,$2))") {
			t.Fatalf("expected physical default-group predicate in sql: %q", sql)
		}
		if !strings.Contains(sql, "NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = alarms_active.device_id)") {
			t.Fatalf("expected legacy ungrouped predicate in sql: %q", sql)
		}
		if len(args) != 2 || args[1] != defaultGroup {
			t.Fatalf("expected default group in query args, got %v", args)
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

	t.Run("默认组序列号过滤同时匹配真实成员和历史未分组设备", func(t *testing.T) {
		defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
		sql, args, err := ApplyDeviceSNVisibilityFilter(
			storage.Psql.Select("*").From("pm_metrics"), "device_sn", []uuid.UUID{defaultGroup}).ToSql()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, "device_sn IN (SELECT serial_number FROM devices WHERE id IN (SELECT device_id FROM device_group_members WHERE group_id IN ($1)))") {
			t.Fatalf("expected physical default-group device-sn predicate in sql: %q", sql)
		}
		if !strings.Contains(sql, "NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id)") {
			t.Fatalf("expected legacy ungrouped device-sn predicate in sql: %q", sql)
		}
		if len(args) != 1 || args[0] != defaultGroup {
			t.Fatalf("expected default group in query args, got %v", args)
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

	t.Run("默认组时返回真实成员和历史未分组 SQL", func(t *testing.T) {
		defaultGroup := uuid.MustParse(global.DefaultLevel2GroupID)
		sql, groups := VisibleSNSubquerySQL("m.device_sn", "$3", []uuid.UUID{defaultGroup})
		want := "(m.device_sn IN (SELECT serial_number FROM devices WHERE id IN (SELECT device_id FROM device_group_members WHERE group_id = ANY($3))) OR m.device_sn IN (SELECT serial_number FROM devices d WHERE NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id)))"
		if sql != want {
			t.Fatalf("unexpected sql: %q", sql)
		}
		if len(groups) != 1 || groups[0] != defaultGroup {
			t.Fatalf("expected default group in real groups, got %v", groups)
		}
	})
}
