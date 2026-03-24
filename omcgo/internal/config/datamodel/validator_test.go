package datamodel

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDataModel() *DataModel {
	params := []Parameter{
		{Path: "Device.DeviceInfo.Manufacturer", Type: "string", Writable: false},
		{Path: "Device.ManagementServer.URL", Type: "string", Writable: true, Constraints: &Constraints{MaxLength: 256, MinLength: 1}},
		{Path: "Device.Services.FAPService.{12}.CellConfig.LTE.RAN.Common.CellIdentity", Type: "unsignedInt", Writable: true,
			Constraints: &Constraints{MinValue: int64Ptr(0), MaxValue: int64Ptr(268435455)}},
		{Path: "Device.Services.FAPService.{12}.CellConfig.LTE.RAN.RF.PhyCellID", Type: "unsignedInt", Writable: true,
			Constraints: &Constraints{MinValue: int64Ptr(0), MaxValue: int64Ptr(503)}},
		{Path: "Device.Services.FAPService.{12}.CellConfig.LTE.RAN.RF.DLBandwidth", Type: "string", Writable: true,
			Constraints: &Constraints{EnumValues: []string{"n6", "n15", "n25", "n50", "n75", "n100"}}},
		{Path: "Device.Services.FAPService.{12}.Enable", Type: "boolean", Writable: true},
		{Path: "Device.Time.CurrentLocalTime", Type: "dateTime", Writable: false},
	}
	objects := []ObjectInfo{
		{Name: "Device.", Access: "READ_WRITE", MaxInstances: 0},
		{Name: "Device.Services.FAPService.{12}.", Access: "READ_WRITE", MaxInstances: 0},
		{Name: "Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.", Access: "READ_WRITE", MaxInstances: 6, MinInstances: 1},
		{Name: "Device.Services.FAPService.{12}.CellConfig.LTE.RAN.NeighborList.LTECell.{256}.", Access: "READ_WRITE", MaxInstances: 256},
	}

	paramJSON, _ := json.Marshal(params)
	objectJSON, _ := json.Marshal(objects)

	return &DataModel{
		ParameterTree: paramJSON,
		ObjectTree:    objectJSON,
	}
}

func int64Ptr(v int64) *int64 { return &v }

func TestNewParameterValidator(t *testing.T) {
	dm := newTestDataModel()
	v, err := NewParameterValidator(dm)
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Len(t, v.params, 7)
	assert.Len(t, v.objects, 4)
}

func TestValidateValue_ReadOnly(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	ve := v.ValidateValue("Device.DeviceInfo.Manufacturer", "test")
	require.NotNil(t, ve)
	assert.Equal(t, "writable", ve.Rule)
}

func TestValidateValue_NotExists(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	ve := v.ValidateValue("Device.NonExistent.Param", "test")
	require.NotNil(t, ve)
	assert.Equal(t, "exists", ve.Rule)
}

func TestValidateValue_TypeCheck(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	// Valid unsignedInt
	ve := v.ValidateValue("Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", "123")
	assert.Nil(t, ve)

	// Invalid unsignedInt
	ve = v.ValidateValue("Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", "abc")
	require.NotNil(t, ve)
	assert.Equal(t, "type", ve.Rule)

	// Valid boolean
	ve = v.ValidateValue("Device.Services.FAPService.1.Enable", "true")
	assert.Nil(t, ve)

	// Invalid boolean
	ve = v.ValidateValue("Device.Services.FAPService.1.Enable", "maybe")
	require.NotNil(t, ve)
	assert.Equal(t, "type", ve.Rule)
}

func TestValidateValue_Constraints(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	// Within range
	ve := v.ValidateValue("Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", "100")
	assert.Nil(t, ve)

	// Exceeds max
	ve = v.ValidateValue("Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", "999")
	require.NotNil(t, ve)
	assert.Equal(t, "constraint", ve.Rule)

	// Enum valid
	ve = v.ValidateValue("Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth", "n50")
	assert.Nil(t, ve)

	// Enum invalid
	ve = v.ValidateValue("Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth", "n200")
	require.NotNil(t, ve)
	assert.Equal(t, "constraint", ve.Rule)

	// String min length
	ve = v.ValidateValue("Device.ManagementServer.URL", "")
	require.NotNil(t, ve)
	assert.Equal(t, "constraint", ve.Rule)
}

func TestValidateValue_MultiInstancePath(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	// Instance 1
	ve := v.ValidateValue("Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", "123")
	assert.Nil(t, ve)

	// Instance 2
	ve = v.ValidateValue("Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity", "456")
	assert.Nil(t, ve)
}

func TestValidateAddObject(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	// Can add when below max
	ve := v.ValidateAddObject("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.{6}.", 3)
	assert.Nil(t, ve)

	// Cannot add when at max
	ve = v.ValidateAddObject("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.{6}.", 6)
	require.NotNil(t, ve)
	assert.Equal(t, "max_instances", ve.Rule)
}

func TestValidateDeleteObject(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	// Can delete when above min
	ve := v.ValidateDeleteObject("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.{6}.", 3)
	assert.Nil(t, ve)

	// Cannot delete when at min (PLMNList min=1)
	ve = v.ValidateDeleteObject("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.{6}.", 1)
	require.NotNil(t, ve)
	assert.Equal(t, "min_instances", ve.Rule)

	// NeighborList min=0, can delete all
	ve = v.ValidateDeleteObject("Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.{256}.", 1)
	assert.Nil(t, ve)
}

func TestLookupParam(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	// Exact match
	p := v.LookupParam("Device.DeviceInfo.Manufacturer")
	require.NotNil(t, p)
	assert.Equal(t, "string", p.Type)

	// Multi-instance match
	p = v.LookupParam("Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity")
	require.NotNil(t, p)
	assert.Equal(t, "unsignedInt", p.Type)

	// Not found
	p = v.LookupParam("Device.Unknown.Param")
	assert.Nil(t, p)
}

func TestLookupObject(t *testing.T) {
	dm := newTestDataModel()
	v, _ := NewParameterValidator(dm)

	// Exact match
	o := v.LookupObject("Device.")
	require.NotNil(t, o)

	// Multi-instance match
	o = v.LookupObject("Device.Services.FAPService.1.")
	require.NotNil(t, o)
	assert.Equal(t, "READ_WRITE", o.Access)

	// Not found
	o = v.LookupObject("Device.Unknown.")
	assert.Nil(t, o)
}
