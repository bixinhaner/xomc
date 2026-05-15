package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// admin_service_test.go — T-0123-P0 service 层守护逻辑表驱动测试
//
// 覆盖路径：
//   - catalog_protected=true → DeleteGroup/Command/Param ErrCatalogProtected
//   - catalog_protected=true → UpdateParam 关键字段（access_type / is_object / ...）锁定
//   - DeleteGroup with commands>0 → ErrGroupNotEmpty
//   - DeleteParam with sub_fields>0 → ErrParamInUse
//   - Happy path Create + Update
//   - audit writer 被正确调用
// ============================================================

// ---- mocks ----

type mockGroupRepo struct {
	groups            map[uuid.UUID]*ParamGroup
	createErr         error
	updateErr         error
	deleteErr         error
	commandCountByGrp map[uuid.UUID]int64
}

func newMockGroupRepo() *mockGroupRepo {
	return &mockGroupRepo{
		groups:            map[uuid.UUID]*ParamGroup{},
		commandCountByGrp: map[uuid.UUID]int64{},
	}
}

func (m *mockGroupRepo) Create(ctx context.Context, g *ParamGroup) error {
	if m.createErr != nil {
		return m.createErr
	}
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	cp := *g
	m.groups[g.ID] = &cp
	return nil
}
func (m *mockGroupRepo) Update(ctx context.Context, g *ParamGroup) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.groups[g.ID]; !ok {
		return ErrGroupNotFound
	}
	cp := *g
	m.groups[g.ID] = &cp
	return nil
}
func (m *mockGroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.groups[id]; !ok {
		return ErrGroupNotFound
	}
	delete(m.groups, id)
	return nil
}
func (m *mockGroupRepo) GetByID(ctx context.Context, id uuid.UUID) (*ParamGroup, error) {
	g, ok := m.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	cp := *g
	return &cp, nil
}
func (m *mockGroupRepo) CountCommandsByGroup(ctx context.Context, id uuid.UUID) (int64, error) {
	return m.commandCountByGrp[id], nil
}

type mockSubFieldRepo struct {
	subFields    map[uuid.UUID]*MMLCommandSubField
	countByParam map[uuid.UUID]int64
}

func newMockSubFieldRepo() *mockSubFieldRepo {
	return &mockSubFieldRepo{
		subFields:    map[uuid.UUID]*MMLCommandSubField{},
		countByParam: map[uuid.UUID]int64{},
	}
}

func (m *mockSubFieldRepo) Create(ctx context.Context, sf *MMLCommandSubField) error {
	if sf.ID == uuid.Nil {
		sf.ID = uuid.New()
	}
	cp := *sf
	m.subFields[sf.ID] = &cp
	return nil
}
func (m *mockSubFieldRepo) Update(ctx context.Context, sf *MMLCommandSubField) error {
	if _, ok := m.subFields[sf.ID]; !ok {
		return ErrSubFieldNotFound
	}
	cp := *sf
	m.subFields[sf.ID] = &cp
	return nil
}
func (m *mockSubFieldRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.subFields[id]; !ok {
		return ErrSubFieldNotFound
	}
	delete(m.subFields, id)
	return nil
}
func (m *mockSubFieldRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLCommandSubField, error) {
	sf, ok := m.subFields[id]
	if !ok {
		return nil, ErrSubFieldNotFound
	}
	cp := *sf
	return &cp, nil
}
func (m *mockSubFieldRepo) ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubField, error) {
	var out []MMLCommandSubField
	for _, sf := range m.subFields {
		if sf.CommandID == commandID {
			out = append(out, *sf)
		}
	}
	return out, nil
}
func (m *mockSubFieldRepo) ListEnrichedByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubFieldEnriched, error) {
	return nil, nil
}
func (m *mockSubFieldRepo) CountByParam(ctx context.Context, paramID uuid.UUID) (int64, error) {
	return m.countByParam[paramID], nil
}

type mockAdminCmdRepo struct {
	commands map[uuid.UUID]*MMLCommand
}

func newMockAdminCmdRepo() *mockAdminCmdRepo {
	return &mockAdminCmdRepo{commands: map[uuid.UUID]*MMLCommand{}}
}

func (m *mockAdminCmdRepo) Create(ctx context.Context, c *MMLCommand) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	cp := *c
	m.commands[c.ID] = &cp
	return nil
}
func (m *mockAdminCmdRepo) Update(ctx context.Context, c *MMLCommand) error {
	if _, ok := m.commands[c.ID]; !ok {
		return ErrCommandNotFound
	}
	cp := *c
	m.commands[c.ID] = &cp
	return nil
}
func (m *mockAdminCmdRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.commands[id]; !ok {
		return ErrCommandNotFound
	}
	delete(m.commands, id)
	return nil
}

type mockAdminParamRepo struct {
	params map[uuid.UUID]*Param
}

func newMockAdminParamRepo() *mockAdminParamRepo {
	return &mockAdminParamRepo{params: map[uuid.UUID]*Param{}}
}

func (m *mockAdminParamRepo) Create(ctx context.Context, p *Param) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	cp := *p
	m.params[p.ID] = &cp
	return nil
}
func (m *mockAdminParamRepo) Update(ctx context.Context, p *Param) error {
	if _, ok := m.params[p.ID]; !ok {
		return ErrParamNotFound
	}
	cp := *p
	m.params[p.ID] = &cp
	return nil
}
func (m *mockAdminParamRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.params[id]; !ok {
		return ErrParamNotFound
	}
	delete(m.params, id)
	return nil
}
func (m *mockAdminParamRepo) GetByID(ctx context.Context, id uuid.UUID) (*Param, error) {
	p, ok := m.params[id]
	if !ok {
		return nil, ErrParamNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *mockAdminParamRepo) List(ctx context.Context, f AdminParamFilter) ([]Param, int64, error) {
	var out []Param
	for _, p := range m.params {
		out = append(out, *p)
	}
	return out, int64(len(out)), nil
}

// ListReferences 空 stub（T-0131 interface 引入；admin_service_test.go 不覆盖此场景）。
func (m *mockAdminParamRepo) ListReferences(ctx context.Context, paramID uuid.UUID) ([]ParamReference, error) {
	return nil, nil
}

// ListPathStateByVersion 空 stub（T-0132 interface 引入；admin_service_test.go 不覆盖 XML import 场景，由 xml_import_service_test.go 单独 stubParamRepo 覆盖）。
func (m *mockAdminParamRepo) ListPathStateByVersion(ctx context.Context, paramVersion string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

// BatchUpsertStandardParams 空 stub（T-0132 interface 引入；同上）。
func (m *mockAdminParamRepo) BatchUpsertStandardParams(ctx context.Context, rows []ImportRow, paramVersion string) (int64, error) {
	return 0, nil
}

type mockAuditWriter struct {
	calls []auditCall
}
type auditCall struct {
	Op       string
	Resource string
}

func (m *mockAuditWriter) Write(ctx context.Context, op, resource string, detail map[string]any) {
	m.calls = append(m.calls, auditCall{Op: op, Resource: resource})
}

// ---- helpers ----

func newAdminTestService() (*AdminService, *mockGroupRepo, *mockAdminCmdRepo, *mockSubFieldRepo, *mockAdminParamRepo, *mockAuditWriter) {
	g := newMockGroupRepo()
	c := newMockAdminCmdRepo()
	sf := newMockSubFieldRepo()
	p := newMockAdminParamRepo()
	audit := &mockAuditWriter{}
	svc := NewAdminService(g, c, sf, p, audit, nil)
	return svc, g, c, sf, p, audit
}

// ============================================================
// Group tests
// ============================================================

func TestAdminService_DeleteGroup_CatalogProtected(t *testing.T) {
	svc, gRepo, _, _, _, _ := newAdminTestService()
	id := uuid.New()
	gRepo.groups[id] = &ParamGroup{ID: id, GroupCode: "BSC_CONFIG", CatalogProtected: true}

	err := svc.DeleteGroup(context.Background(), id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogProtected), "want ErrCatalogProtected, got %v", err)
}

func TestAdminService_DeleteGroup_NotEmpty(t *testing.T) {
	svc, gRepo, _, _, _, _ := newAdminTestService()
	id := uuid.New()
	gRepo.groups[id] = &ParamGroup{ID: id, GroupCode: "ADMIN_GRP", CatalogProtected: false}
	gRepo.commandCountByGrp[id] = 3

	err := svc.DeleteGroup(context.Background(), id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGroupNotEmpty), "want ErrGroupNotEmpty, got %v", err)
}

func TestAdminService_DeleteGroup_Happy(t *testing.T) {
	svc, gRepo, _, _, _, audit := newAdminTestService()
	id := uuid.New()
	gRepo.groups[id] = &ParamGroup{ID: id, GroupCode: "ADMIN_GRP", CatalogProtected: false}

	err := svc.DeleteGroup(context.Background(), id)
	require.NoError(t, err)
	assert.NotContains(t, gRepo.groups, id)
	require.Len(t, audit.calls, 1)
	assert.Equal(t, "mml.catalog.group.deleted", audit.calls[0].Op)
}

func TestAdminService_CreateGroup_AuditEmitted(t *testing.T) {
	svc, _, _, _, _, audit := newAdminTestService()
	_, err := svc.CreateGroup(context.Background(), CreateGroupReq{
		GroupCode: "FOO", ParamVersion: "STANDARD",
	})
	require.NoError(t, err)
	require.Len(t, audit.calls, 1)
	assert.Equal(t, "mml.catalog.group.created", audit.calls[0].Op)
}

// ============================================================
// Param tests
// ============================================================

func TestAdminService_UpdateParam_ProtectedLocksAccessType(t *testing.T) {
	svc, _, _, _, pRepo, _ := newAdminTestService()
	id := uuid.New()
	pRepo.params[id] = &Param{
		ID: id, Tr069Path: "Device.X", AccessType: AccessTypeReadOnly,
		CatalogProtected: true, Source: SourceStandard,
	}

	newAccess := AccessTypeReadWrite
	_, err := svc.UpdateParam(context.Background(), id, UpdateParamReq{
		AccessType: &newAccess,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogProtected))
}

func TestAdminService_UpdateParam_ProtectedAllowsLabelChange(t *testing.T) {
	svc, _, _, _, pRepo, _ := newAdminTestService()
	id := uuid.New()
	pRepo.params[id] = &Param{
		ID: id, Tr069Path: "Device.X", AccessType: AccessTypeReadOnly,
		CatalogProtected: true, Source: SourceStandard,
	}

	newCT := map[string]string{"en-US": "test", "zh-CN": "测试"}
	_, err := svc.UpdateParam(context.Background(), id, UpdateParamReq{
		ConstraintTextI18n: &newCT,
	})
	require.NoError(t, err)
	assert.Equal(t, newCT, pRepo.params[id].ConstraintTextI18n)
}

func TestAdminService_DeleteParam_CatalogProtected(t *testing.T) {
	svc, _, _, _, pRepo, _ := newAdminTestService()
	id := uuid.New()
	pRepo.params[id] = &Param{ID: id, CatalogProtected: true, Source: SourceStandard}

	err := svc.DeleteParam(context.Background(), id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogProtected))
}

func TestAdminService_DeleteParam_InUse(t *testing.T) {
	svc, _, _, sfRepo, pRepo, _ := newAdminTestService()
	id := uuid.New()
	pRepo.params[id] = &Param{ID: id, CatalogProtected: false}
	sfRepo.countByParam[id] = 2

	err := svc.DeleteParam(context.Background(), id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrParamInUse))
}

func TestAdminService_DeleteParam_Happy(t *testing.T) {
	svc, _, _, _, pRepo, audit := newAdminTestService()
	id := uuid.New()
	pRepo.params[id] = &Param{ID: id, CatalogProtected: false, ParamCode: "TEST", Tr069Path: "Device.Test"}

	err := svc.DeleteParam(context.Background(), id)
	require.NoError(t, err)
	assert.NotContains(t, pRepo.params, id)
	require.Len(t, audit.calls, 1)
	assert.Equal(t, "mml.catalog.param.deleted", audit.calls[0].Op)
}

// ============================================================
// Command tests
// ============================================================

func TestAdminService_DeleteCommand_CatalogProtected(t *testing.T) {
	svc, _, cRepo, _, _, _ := newAdminTestService()
	id := uuid.New()
	cRepo.commands[id] = &MMLCommand{ID: id, CommandCode: "LST_FOO", CatalogProtected: true}

	getByID := func(_ context.Context, queryID uuid.UUID) (*MMLCommand, error) {
		c, ok := cRepo.commands[queryID]
		if !ok {
			return nil, ErrCommandNotFound
		}
		cp := *c
		return &cp, nil
	}

	err := svc.DeleteCommand(context.Background(), getByID, id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogProtected))
}

func TestAdminService_UpdateCommand_ProtectedLocksLogicalCode(t *testing.T) {
	svc, _, cRepo, _, _, _ := newAdminTestService()
	id := uuid.New()
	cRepo.commands[id] = &MMLCommand{
		ID: id, CommandCode: "LST_FOO", LogicalCode: "FOO",
		CatalogProtected: true, Source: SourceStandard,
	}
	getByID := func(_ context.Context, queryID uuid.UUID) (*MMLCommand, error) {
		c, ok := cRepo.commands[queryID]
		if !ok {
			return nil, ErrCommandNotFound
		}
		cp := *c
		return &cp, nil
	}

	newLogical := "BAR"
	_, err := svc.UpdateCommand(context.Background(), getByID, id, UpdateCommandReq{
		LogicalCode: &newLogical,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogProtected))
}

func TestAdminService_CreateCommand_DefaultsSource(t *testing.T) {
	svc, _, cRepo, _, _, _ := newAdminTestService()
	c, err := svc.CreateCommand(context.Background(), CreateCommandReq{
		CommandName: "test", CommandCode: "T_FOO", OperationType: "LST",
	})
	require.NoError(t, err)
	assert.Equal(t, SourceAdmin, c.Source)
	assert.False(t, c.CatalogProtected)
	require.Contains(t, cRepo.commands, c.ID)
}

// ============================================================
// SubField tests
// ============================================================

func TestAdminService_CreateSubField_DefaultsSelected(t *testing.T) {
	svc, _, _, sfRepo, _, _ := newAdminTestService()
	cmdID := uuid.New()
	paramID := uuid.New()
	sf, err := svc.CreateSubField(context.Background(), cmdID, CreateSubFieldReq{
		ParamID:  paramID,
		MMLCode:  "TEST_FIELD",
		LabelI18n: map[string]string{"en-US": "Test"},
	})
	require.NoError(t, err)
	assert.True(t, sf.DefaultSelected, "default_selected omitted should default to true")
	assert.False(t, sf.IsRequired)
	require.Contains(t, sfRepo.subFields, sf.ID)
}

func TestAdminService_CreateSubField_ExplicitFalse(t *testing.T) {
	svc, _, _, _, _, _ := newAdminTestService()
	cmdID := uuid.New()
	paramID := uuid.New()
	false_ := false
	sf, err := svc.CreateSubField(context.Background(), cmdID, CreateSubFieldReq{
		ParamID:         paramID,
		MMLCode:         "TEST_FIELD",
		DefaultSelected: &false_,
	})
	require.NoError(t, err)
	assert.False(t, sf.DefaultSelected, "explicit false should be honored")
}

// ============================================================
// IsErrNotFound utility
// ============================================================

func TestIsErrNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"sub_field", ErrSubFieldNotFound, true},
		{"group", ErrGroupNotFound, true},
		{"command", ErrCommandNotFound, true},
		{"param", ErrParamNotFound, true},
		{"protected", ErrCatalogProtected, false},
		{"in_use", ErrParamInUse, false},
		{"not_empty", ErrGroupNotEmpty, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsErrNotFound(tc.err))
		})
	}
}
