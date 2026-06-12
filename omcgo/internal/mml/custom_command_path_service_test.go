package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// issue #115 调整3（A1）：自定义命令 PATH 关联写操作的鉴权单测。
// 鉴权语义与 UpdateCustomCommand 一致——owner / super_admin 才能写。

// mockCustomCommandPathRepo 实现 CustomCommandPathRepository。
type mockCustomCommandPathRepo struct {
	listFn        func(ctx context.Context, commandID uuid.UUID) ([]MMLCustomCommandPathView, error)
	batchCreateFn func(ctx context.Context, commandID uuid.UUID, ids []uuid.UUID) ([]MMLCustomCommandPath, error)
	updateFn      func(ctx context.Context, commandID, pathID uuid.UUID, ds *bool, so *int) (*MMLCustomCommandPath, error)
	deleteFn      func(ctx context.Context, commandID, pathID uuid.UUID) error
}

func (m *mockCustomCommandPathRepo) ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCustomCommandPathView, error) {
	if m.listFn != nil {
		return m.listFn(ctx, commandID)
	}
	return nil, nil
}

func (m *mockCustomCommandPathRepo) BatchCreate(ctx context.Context, commandID uuid.UUID, ids []uuid.UUID) ([]MMLCustomCommandPath, error) {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, commandID, ids)
	}
	return nil, nil
}

func (m *mockCustomCommandPathRepo) Update(ctx context.Context, commandID, pathID uuid.UUID, ds *bool, so *int) (*MMLCustomCommandPath, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, commandID, pathID, ds, so)
	}
	return nil, nil
}

func (m *mockCustomCommandPathRepo) Delete(ctx context.Context, commandID, pathID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, commandID, pathID)
	}
	return nil
}

func privateCommandOwnedBy(ownerID uuid.UUID) *MMLCustomCommand {
	return &MMLCustomCommand{
		ID: uuid.New(), CommandName: "X", CommandScope: "private",
		Creator: "alice", OwnerUserID: &ownerID,
	}
}

func TestService_BatchAddCustomCommandPaths_NonOwner_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	existing := privateCommandOwnedBy(ownerID)
	cmdRepo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
	}
	pathCalled := false
	pathRepo := &mockCustomCommandPathRepo{
		batchCreateFn: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]MMLCustomCommandPath, error) {
			pathCalled = true
			return nil, nil
		},
	}
	svc := newCRUDServiceWithRepo(cmdRepo)
	svc.SetCustomCommandPathRepo(pathRepo)

	_, err := svc.BatchAddCustomCommandPaths(context.Background(), existing.ID,
		[]uuid.UUID{uuid.New()}, otherID, "bob", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrForbidden), "非 owner 批量添加应 403")
	assert.False(t, pathCalled, "鉴权失败时不得触达 path 仓库")
}

func TestService_BatchAddCustomCommandPaths_Owner_OK(t *testing.T) {
	ownerID := uuid.New()
	existing := privateCommandOwnedBy(ownerID)
	cmdRepo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
	}
	want := []MMLCustomCommandPath{{
		ID: uuid.New(), CommandID: existing.ID, StandardPathID: uuid.New(),
		DefaultSelected: true, SortOrder: 0,
	}}
	pathRepo := &mockCustomCommandPathRepo{
		batchCreateFn: func(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]MMLCustomCommandPath, error) {
			require.Len(t, ids, 1)
			return want, nil
		},
	}
	svc := newCRUDServiceWithRepo(cmdRepo)
	svc.SetCustomCommandPathRepo(pathRepo)

	got, err := svc.BatchAddCustomCommandPaths(context.Background(), existing.ID,
		[]uuid.UUID{want[0].StandardPathID}, ownerID, "alice", false)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_BatchAddCustomCommandPaths_SuperAdmin_OK(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	existing := privateCommandOwnedBy(ownerID)
	cmdRepo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
	}
	pathRepo := &mockCustomCommandPathRepo{
		batchCreateFn: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]MMLCustomCommandPath, error) {
			return []MMLCustomCommandPath{{ID: uuid.New()}}, nil
		},
	}
	svc := newCRUDServiceWithRepo(cmdRepo)
	svc.SetCustomCommandPathRepo(pathRepo)

	// super_admin 即便非 owner 也放行。
	_, err := svc.BatchAddCustomCommandPaths(context.Background(), existing.ID,
		[]uuid.UUID{uuid.New()}, otherID, "bob", true)
	require.NoError(t, err)
}

func TestService_BatchAddCustomCommandPaths_EmptyIDs_Invalid(t *testing.T) {
	svc := newCRUDServiceWithRepo(&mockCustomCommandRepo{})
	svc.SetCustomCommandPathRepo(&mockCustomCommandPathRepo{})

	_, err := svc.BatchAddCustomCommandPaths(context.Background(), uuid.New(),
		nil, uuid.New(), "bob", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput), "空 path 列表应 400")
}

func TestService_UpdateCustomCommandPath_NonOwner_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	existing := privateCommandOwnedBy(ownerID)
	cmdRepo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
	}
	updateCalled := false
	pathRepo := &mockCustomCommandPathRepo{
		updateFn: func(_ context.Context, _, _ uuid.UUID, _ *bool, _ *int) (*MMLCustomCommandPath, error) {
			updateCalled = true
			return nil, nil
		},
	}
	svc := newCRUDServiceWithRepo(cmdRepo)
	svc.SetCustomCommandPathRepo(pathRepo)

	sel := true
	_, err := svc.UpdateCustomCommandPath(context.Background(), existing.ID, uuid.New(), &sel, nil, otherID, "bob", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrForbidden), "非 owner 修改应 403")
	assert.False(t, updateCalled, "鉴权失败时不得触达 path 仓库")
}

func TestService_DeleteCustomCommandPath_NonOwner_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	existing := privateCommandOwnedBy(ownerID)
	cmdRepo := &mockCustomCommandRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*MMLCustomCommand, error) { return existing, nil },
	}
	deleteCalled := false
	pathRepo := &mockCustomCommandPathRepo{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error { deleteCalled = true; return nil },
	}
	svc := newCRUDServiceWithRepo(cmdRepo)
	svc.SetCustomCommandPathRepo(pathRepo)

	err := svc.DeleteCustomCommandPath(context.Background(), existing.ID, uuid.New(), otherID, "bob", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrForbidden), "非 owner 删除应 403")
	assert.False(t, deleteCalled, "鉴权失败时不得触达 path 仓库")
}
