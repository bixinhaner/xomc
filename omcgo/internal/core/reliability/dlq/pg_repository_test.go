package dlq

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/core/model"
)

// TestNewPgRepository_NilPoolStillConstructible 验证构造器对 nil 池不 panic。
// 真实 SQL 路径由 e2e (GET /admin/dead-letters) 兜底，本文件仅做编译/接口契约级
// 校验 — PRD §8.5 明确说 pg_repository_test.go (skipped)。
func TestNewPgRepository_NilPoolStillConstructible(t *testing.T) {
	r := NewPgRepository(nil)
	assert.NotNil(t, r, "constructor should not return nil even with nil pool")
	// 不调用任何方法（无池调用 PG 必然 panic / nil deref）
}

// TestRepositoryInterfaceSatisfied 编译时校验：PgRepository 实现 Repository 接口。
func TestRepositoryInterfaceSatisfied(t *testing.T) {
	var _ Repository = (*PgRepository)(nil)
}

// TestFilterZeroValueDefaultsViaListRequest 校验 ListRequest 默认值在 Filter 中的行为。
func TestFilterZeroValueDefaultsViaListRequest(t *testing.T) {
	f := Filter{ListRequest: model.ListRequest{}}
	// Limit / Offset 在 Limit() / Offset() 中归一化
	assert.Equal(t, 20, f.Limit(), "default page size should be 20 when zero")
	assert.Equal(t, 0, f.Offset(), "page=1 offset should be 0")
}

// TestFilterListRequestPagination 验证非零分页参数的传播。
func TestFilterListRequestPagination(t *testing.T) {
	f := Filter{ListRequest: model.ListRequest{Page: 3, PageSize: 50}}
	assert.Equal(t, 50, f.Limit())
	assert.Equal(t, 100, f.Offset(), "(3-1)*50 = 100")
}
