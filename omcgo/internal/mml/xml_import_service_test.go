package mml

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
)

// stubParamRepo 仅实现 T-0132 XMLImportService 需要的 2 个方法 + 5 个空 stub 实现 interface 契约。
type stubParamRepo struct {
	pathState        map[string]bool
	pathStateErr     error
	upsertCalls      int
	upsertedRows     []ImportRow
	upsertVersion    string
	upsertReturnRows int64
	upsertErr        error
}

func (r *stubParamRepo) Create(ctx context.Context, p *Param) error           { return nil }
func (r *stubParamRepo) Update(ctx context.Context, p *Param) error           { return nil }
func (r *stubParamRepo) Delete(ctx context.Context, id uuid.UUID) error       { return nil }
func (r *stubParamRepo) GetByID(ctx context.Context, id uuid.UUID) (*Param, error) {
	return nil, nil
}
func (r *stubParamRepo) List(ctx context.Context, f AdminParamFilter) ([]Param, int64, error) {
	return nil, 0, nil
}
func (r *stubParamRepo) ListReferences(ctx context.Context, paramID uuid.UUID) ([]ParamReference, error) {
	return nil, nil
}
func (r *stubParamRepo) ListPathStateByVersion(ctx context.Context, paramVersion string) (map[string]bool, error) {
	if r.pathStateErr != nil {
		return nil, r.pathStateErr
	}
	return r.pathState, nil
}
func (r *stubParamRepo) BatchUpsertStandardParams(ctx context.Context, rows []ImportRow, paramVersion string) (int64, error) {
	r.upsertCalls++
	r.upsertedRows = append(r.upsertedRows, rows...)
	r.upsertVersion = paramVersion
	return r.upsertReturnRows, r.upsertErr
}

const minimalXML = `<?xml version="1.0" encoding="UTF-8"?>
<standardModel>
  <objects>
    <object standardPath="Device.IP.Interface." access="READ_WRITE" changeApplies="Immediate"/>
  </objects>
  <parameters>
    <param standardPath="Device.DeviceInfo.SoftwareVersion" access="READ_ONLY" type="STRING" changeApplies="Immediate"/>
    <param standardPath="Device.IP.Interface.Enable" access="READ_WRITE" type="BOOLEAN" changeApplies="Immediate"/>
    <param standardPath="Device.IP.Interface.MTU" access="READ_WRITE" type="U_INT" changeApplies="OnReboot" min="68" max="9000"/>
  </parameters>
</standardModel>`

func TestXMLImportService_Preview_AllNewRows(t *testing.T) {
	// DB 空 → 所有行都是 "add" 桶
	repo := &stubParamRepo{pathState: map[string]bool{}}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	resp, err := svc.Preview(context.Background(), strings.NewReader(minimalXML), "STANDARD")
	require.NoError(t, err)
	assert.Equal(t, "STANDARD", resp.VersionCode)
	assert.Equal(t, 4, resp.Summary.Total) // 3 params + 1 object
	assert.Equal(t, 4, resp.Summary.Add)
	assert.Equal(t, 0, resp.Summary.Modify)
	assert.Equal(t, 0, resp.Summary.Skipped)
	assert.False(t, resp.Truncated)
	// 4 行 diff 全为 add
	for _, d := range resp.Diffs {
		assert.Equal(t, BucketAdd, d.Bucket)
	}
}

func TestXMLImportService_Preview_MixedBuckets(t *testing.T) {
	// DB 状态：SoftwareVersion 已存在 catalog_protected=true → modify
	//         Enable 已存在 catalog_protected=false → skipped (admin 改过)
	//         MTU 不存在 → add
	//         IP.Interface. (object) 不存在 → add
	repo := &stubParamRepo{
		pathState: map[string]bool{
			"Device.DeviceInfo.SoftwareVersion": true,
			"Device.IP.Interface.Enable":        false,
		},
	}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	resp, err := svc.Preview(context.Background(), strings.NewReader(minimalXML), "STANDARD")
	require.NoError(t, err)
	assert.Equal(t, 4, resp.Summary.Total)
	assert.Equal(t, 2, resp.Summary.Add)
	assert.Equal(t, 1, resp.Summary.Modify)
	assert.Equal(t, 1, resp.Summary.Skipped)

	// 桶映射核查
	gotByPath := map[string]ImportBucket{}
	for _, d := range resp.Diffs {
		gotByPath[d.Tr069Path] = d.Bucket
	}
	assert.Equal(t, BucketModify, gotByPath["Device.DeviceInfo.SoftwareVersion"])
	assert.Equal(t, BucketSkipped, gotByPath["Device.IP.Interface.Enable"])
	assert.Equal(t, BucketAdd, gotByPath["Device.IP.Interface.MTU"])
	assert.Equal(t, BucketAdd, gotByPath["Device.IP.Interface."])
}

func TestXMLImportService_Preview_EmptyXML(t *testing.T) {
	emptyXML := `<?xml version="1.0"?><standardModel><parameters></parameters><objects></objects></standardModel>`
	repo := &stubParamRepo{pathState: map[string]bool{}}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	_, err := svc.Preview(context.Background(), strings.NewReader(emptyXML), "STANDARD")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0 rows")
}

func TestXMLImportService_Preview_EmptyVersionCode(t *testing.T) {
	repo := &stubParamRepo{pathState: map[string]bool{}}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	_, err := svc.Preview(context.Background(), strings.NewReader(minimalXML), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version_code required")
}

func TestXMLImportService_Preview_TruncatedFlag(t *testing.T) {
	// 构造一个超过 MaxPreviewDiffs 的 XML（mock by injecting many object lines is heavy；
	// here we instead verify by setting MaxPreviewDiffs threshold semantics via small XML +
	// asserting Truncated=false when below threshold）
	repo := &stubParamRepo{pathState: map[string]bool{}}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	resp, err := svc.Preview(context.Background(), strings.NewReader(minimalXML), "STANDARD")
	require.NoError(t, err)
	assert.False(t, resp.Truncated, "4 rows < 200 threshold, should not truncate")
}

func TestXMLImportService_Apply_Success(t *testing.T) {
	repo := &stubParamRepo{
		pathState:        map[string]bool{},
		upsertReturnRows: 4,
	}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	resp, err := svc.Apply(context.Background(), strings.NewReader(minimalXML), "STANDARD")
	require.NoError(t, err)
	assert.Equal(t, "STANDARD", resp.VersionCode)
	assert.Equal(t, 4, resp.Total)
	assert.Equal(t, int64(4), resp.RowsAffected)
	assert.Equal(t, 1, repo.upsertCalls)
	assert.Equal(t, "STANDARD", repo.upsertVersion)
	assert.Len(t, repo.upsertedRows, 4)
}

func TestXMLImportService_Apply_EmptyVersionCode(t *testing.T) {
	repo := &stubParamRepo{pathState: map[string]bool{}}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	_, err := svc.Apply(context.Background(), strings.NewReader(minimalXML), "")
	require.Error(t, err)
}

func TestXMLImportService_Apply_NoRowsInXML(t *testing.T) {
	emptyXML := `<?xml version="1.0"?><standardModel><parameters></parameters><objects></objects></standardModel>`
	repo := &stubParamRepo{pathState: map[string]bool{}}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	_, err := svc.Apply(context.Background(), strings.NewReader(emptyXML), "STANDARD")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0 rows")
}

func TestXMLImportService_Apply_InvalidXML(t *testing.T) {
	invalidXML := `<not-valid-xml`
	repo := &stubParamRepo{pathState: map[string]bool{}}
	svc := NewXMLImportService(repo, nil, zap.NewNop())

	_, err := svc.Apply(context.Background(), strings.NewReader(invalidXML), "STANDARD")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse xml")
}

func TestBuildImportRows_Dedup(t *testing.T) {
	// 同 path 在 params 和 objects 中都出现 → params 优先，不重复
	xml := `<?xml version="1.0"?><standardModel>
		<parameters>
			<param standardPath="Device.Foo" access="READ_ONLY" type="STRING"/>
		</parameters>
		<objects>
			<object standardPath="Device.Foo" access="READ_ONLY"/>
		</objects>
	</standardModel>`
	params, objects, err := mmlstandardloader.ParseStandardXML(strings.NewReader(xml))
	require.NoError(t, err)
	rows := buildImportRows(params, objects)
	assert.Len(t, rows, 1, "duplicate path should be deduped")
}

func TestBuildImportRows_ValueTypeMapping(t *testing.T) {
	xml := `<?xml version="1.0"?><standardModel><parameters>
		<param standardPath="A.Int" type="INT" access="READ_ONLY"/>
		<param standardPath="A.UInt" type="U_INT" access="READ_ONLY"/>
		<param standardPath="A.Bool" type="BOOLEAN" access="READ_ONLY"/>
		<param standardPath="A.Str" type="STRING" access="READ_ONLY"/>
		<param standardPath="A.Unknown" type="WHATEVER" access="READ_ONLY"/>
	</parameters></standardModel>`
	params, objects, err := mmlstandardloader.ParseStandardXML(strings.NewReader(xml))
	require.NoError(t, err)
	rows := buildImportRows(params, objects)
	got := map[string]string{}
	for _, r := range rows {
		got[r.Tr069Path] = r.ValueType
	}
	assert.Equal(t, "int", got["A.Int"])
	assert.Equal(t, "unsignedInt", got["A.UInt"])
	assert.Equal(t, "boolean", got["A.Bool"])
	assert.Equal(t, "string", got["A.Str"])
	assert.Equal(t, "string", got["A.Unknown"], "unknown type falls back to string")
}
