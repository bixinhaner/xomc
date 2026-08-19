package parammodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkSet(mappings []ParamMapping) *MappingSet {
	return &MappingSet{Mappings: mappings, Source: MappingSourceDefault}
}

func TestMappingValidator_LookupParam_Exact(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{StandardPath: "Device.WiFi.SSID.{i}.Enable", PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", EntryType: "parameter", Access: "readWrite", DataType: "boolean"},
	}))
	got := v.LookupParam("Dev.WiFi.SSID.{i}.Enabled")
	require.NotNil(t, got)
	assert.Equal(t, "boolean", got.DataType)
}

func TestMappingValidator_LookupParam_InstanceNormalization(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", EntryType: "parameter", Access: "readWrite", DataType: "boolean"},
	}))
	got := v.LookupParam("Dev.WiFi.SSID.7.Enabled")
	require.NotNil(t, got)
	assert.Equal(t, "boolean", got.DataType)
}

func TestMappingValidator_LookupParam_Miss(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.A", EntryType: "parameter"},
	}))
	assert.Nil(t, v.LookupParam("Dev.B"))
}

func TestMappingValidator_LookupParam_ObjectIsNotParam(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.WiFi.", EntryType: "object", Access: "readOnly"},
	}))
	assert.Nil(t, v.LookupParam("Dev.WiFi."))
}

func TestMappingValidator_LookupObject(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.WiFi.", EntryType: "object", Access: "readWrite"},
		{PrivatePath: "Dev.A", EntryType: "parameter"},
	}))
	require.NotNil(t, v.LookupObject("Dev.WiFi."))
	assert.Nil(t, v.LookupObject("Dev.A"))
}

func TestMappingValidator_ValidateValue_NotFound(t *testing.T) {
	v := NewMappingValidator(mkSet(nil))
	ve := v.ValidateValue("Dev.X", "1")
	require.NotNil(t, ve)
	assert.Equal(t, "not_found", ve.Code)
}

func TestMappingValidator_ValidateValue_NotWritable(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.A", EntryType: "parameter", Access: "readOnly"},
	}))
	ve := v.ValidateValue("Dev.A", "1")
	require.NotNil(t, ve)
	assert.Equal(t, "not_writable", ve.Code)
}

func TestMappingValidator_ValidateValue_Range(t *testing.T) {
	min := int64(1)
	max := int64(10)
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.A", EntryType: "parameter", Access: "readWrite", MinValue: &min, MaxValue: &max},
	}))

	assert.Nil(t, v.ValidateValue("Dev.A", "5"))

	ve := v.ValidateValue("Dev.A", "0")
	require.NotNil(t, ve)
	assert.Equal(t, "out_of_range", ve.Code)

	ve = v.ValidateValue("Dev.A", "11")
	require.NotNil(t, ve)
	assert.Equal(t, "out_of_range", ve.Code)
}

func TestMappingValidator_ValidateValue_TypeMismatch(t *testing.T) {
	min := int64(1)
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.A", EntryType: "parameter", Access: "readWrite", MinValue: &min},
	}))
	ve := v.ValidateValue("Dev.A", "abc")
	require.NotNil(t, ve)
	assert.Equal(t, "type_mismatch", ve.Code)
}

func TestMappingValidator_ValidateValue_NoRange_NoTypeCheck(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.A", EntryType: "parameter", Access: "readWrite", DataType: "string"},
	}))
	assert.Nil(t, v.ValidateValue("Dev.A", "anyfreeform")) // 无 min/max → 跳过类型检查
}

func TestMappingValidator_ValidateValue_StringLengthRange(t *testing.T) {
	min := int64(1)
	max := int64(6)
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.A", EntryType: "parameter", Access: "readWrite", DataType: "string", MinValue: &min, MaxValue: &max},
	}))

	assert.Nil(t, v.ValidateValue("Dev.A", "46000"))

	ve := v.ValidateValue("Dev.A", "")
	require.NotNil(t, ve)
	assert.Equal(t, "out_of_range", ve.Code)
	assert.Contains(t, ve.Message, "length 0 < min 1")

	ve = v.ValidateValue("Dev.A", "1234567")
	require.NotNil(t, ve)
	assert.Equal(t, "out_of_range", ve.Code)
	assert.Contains(t, ve.Message, "length 7 > max 6")
}

func TestMappingValidator_ValidateAddObject_OK(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.WiFi.SSID.", EntryType: "object", Access: "readWrite"},
	}))
	assert.Nil(t, v.ValidateAddObject("Dev.WiFi.SSID.", 5))
}

func TestMappingValidator_ValidateAddObject_MatchesChildObjectTemplate(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Device.Ethernet.Interface.{i}.VlanInterface.{i}.", EntryType: "object", Access: "READ_WRITE"},
	}))
	assert.Nil(t, v.ValidateAddObject("Device.Ethernet.Interface.2.VlanInterface.", 0))
}

func TestMappingValidator_BTSCollectionUsesIndexedObjectCapability(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "DeviceGSM.Bts.", EntryType: "object", Access: "READ_ONLY"},
		{PrivatePath: "DeviceGSM.Bts.{i}.", EntryType: "object", Access: "READ_WRITE"},
	}))

	single := v.LookupObject("DeviceGSM.Bts.")
	require.NotNil(t, single)
	assert.Equal(t, "READ_ONLY", single.Access)

	indexed := v.LookupObject("DeviceGSM.Bts.7.")
	require.NotNil(t, indexed)
	assert.Equal(t, "DeviceGSM.Bts.{i}.", indexed.PrivatePath)
	assert.Equal(t, "READ_WRITE", indexed.Access)

	assert.Nil(t, v.ValidateAddObject("DeviceGSM.Bts.", 0),
		"AddObject on the collection must be authorized by DeviceGSM.Bts.{i}., not the single object")
	assert.Nil(t, v.ValidateDeleteObject("DeviceGSM.Bts.7.", 0))
}

func TestMappingValidator_ValidateAddObject_NotWritable(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.WiFi.", EntryType: "object", Access: "readOnly"},
	}))
	ve := v.ValidateAddObject("Dev.WiFi.", 0)
	require.NotNil(t, ve)
	assert.Equal(t, "not_writable", ve.Code)
}

func TestMappingValidator_ValidateAddObject_NotFound(t *testing.T) {
	v := NewMappingValidator(mkSet(nil))
	ve := v.ValidateAddObject("Dev.X.", 0)
	require.NotNil(t, ve)
	assert.Equal(t, "not_found", ve.Code)
}

func TestMappingValidator_ValidateDeleteObject_OK(t *testing.T) {
	v := NewMappingValidator(mkSet([]ParamMapping{
		{PrivatePath: "Dev.WiFi.SSID.", EntryType: "object", Access: "readWrite"},
	}))
	assert.Nil(t, v.ValidateDeleteObject("Dev.WiFi.SSID.", 3))
}

func TestMappingValidator_NilSet(t *testing.T) {
	v := NewMappingValidator(nil)
	assert.Nil(t, v)
	assert.Nil(t, v.LookupParam("any"))
	assert.Nil(t, v.LookupObject("any"))
	assert.Nil(t, v.ValidateValue("any", "1"))
	assert.Nil(t, v.ValidateAddObject("any.", 0))
	assert.Nil(t, v.ValidateDeleteObject("any.", 0))
	assert.Empty(t, v.Mappings())
	assert.Equal(t, MappingSource(""), v.Source())
}

func TestNormalizeInstancePath(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"Foo.1.Bar", "Foo.{i}.Bar"},
		{"Foo.1.Bar.2.Baz", "Foo.{i}.Bar.{i}.Baz"},
		{"Foo.{i}.Bar", "Foo.{i}.Bar"},
		{"Foo.Bar", "Foo.Bar"},
		{"Foo.1.", "Foo.{i}."},
		{"", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.out, normalizeInstancePath(c.in), "in=%q", c.in)
	}
}

func TestIsWritable(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"readWrite", true},
		{"read_write", true},
		{"RW", true},
		{"rw", true},
		{"writeOnly", true},
		{"WriteOnly", true},
		{"writeable", true},
		{"readOnly", false},
		{"RO", false},
		{"", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, isWritable(c.in), "in=%q", c.in)
	}
}

func TestMappingValidator_SourceAndMappings(t *testing.T) {
	set := &MappingSet{Source: MappingSourceDiscovered, Mappings: []ParamMapping{
		{PrivatePath: "A"}, {PrivatePath: "B"},
	}}
	v := NewMappingValidator(set)
	assert.Equal(t, MappingSourceDiscovered, v.Source())
	assert.Len(t, v.Mappings(), 2)
}
