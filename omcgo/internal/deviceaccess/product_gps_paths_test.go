package deviceaccess

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/stretchr/testify/require"
)

type gpsProductStub struct{ product *product.Product }

func (s gpsProductStub) MatchProductClass(context.Context, string) (*product.MatchResult, error) {
	return &product.MatchResult{Product: s.product}, nil
}

type gpsTranslatorStub struct{ translator *parammodel.Translator }

func (s gpsTranslatorStub) Translator(context.Context, uuid.UUID, string) (*parammodel.Translator, error) {
	return s.translator, nil
}

func TestProductGPSPathResolverTranslatesOnlyMappedPaths(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{StandardPath: "Device.FAP.GPS.LockedLatitude", PrivatePath: "Device.X_VENDOR.GPS.Lat"},
		{StandardPath: "Device.FAP.GPS.LockedLongitude", PrivatePath: "Device.X_VENDOR.GPS.Lon"},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}}, gpsTranslatorStub{translator: translator})

	paths, err := resolver.Resolve(context.Background(), "FAP-TEST", "V1")

	require.NoError(t, err)
	require.Equal(t, []string{"Device.X_VENDOR.GPS.Lat", "Device.X_VENDOR.GPS.Lon"}, paths)
}

func TestProductGPSPathResolverAcceptsBMCoordinateStandardAliases(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude",
			PrivatePath:  "Device.FAP.GPS.LockedLatitude",
		},
		{
			StandardPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude",
			PrivatePath:  "Device.FAP.GPS.LockedLongitude",
		},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)

	mappings, err := resolver.ResolveMappings(context.Background(), "FAP/BU1810", "BM_2.0.4")

	require.NoError(t, err)
	require.Equal(t, []GPSParameterPath{
		{StandardPath: gpsLatitudeStandardPath, PrivatePath: "Device.FAP.GPS.LockedLatitude"},
		{StandardPath: gpsLongitudeStandardPath, PrivatePath: "Device.FAP.GPS.LockedLongitude"},
	}, mappings)
}

func TestProductGPSPathResolverFailsWithoutProductParamModel(t *testing.T) {
	resolver := NewProductGPSPathResolver(gpsProductStub{product: &product.Product{ID: uuid.New()}}, gpsTranslatorStub{})
	_, err := resolver.Resolve(context.Background(), "FAP-TEST", "V1")
	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
}

func TestProductGPSPathResolverResolvesServingCellRootFromModel(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC",
			PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC",
		},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)

	root, err := resolver.ResolveServingCellRoot(context.Background(), "FAP/BU1810", "BM_2.0.4")

	require.NoError(t, err)
	require.Equal(t, servingCellStandardRoot, root)
}

func TestProductGPSPathResolverResolvesServingCellRootFromConcreteInstances(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
		},
		{
			StandardPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity",
		},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)

	root, err := resolver.ResolveServingCellRoot(context.Background(), "FAP/BU1810", "BM_2.0.4")

	require.NoError(t, err)
	require.Equal(t, servingCellStandardRoot, root)
}

func TestProductGPSPathResolverRejectsAmbiguousConcreteServingCellRoots(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "Device.X_VENDOR.Cells.1.Identity",
		},
		{
			StandardPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "Device.X_OTHER.Cells.2.Identity",
		},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)

	_, err := resolver.ResolveServingCellRoot(context.Background(), "FAP-AMBIGUOUS", "V1")

	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
}

func TestProductGPSPathResolverUsesServingPLMNListWhenPrimaryMarkerIsUnmapped(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC",
			PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC",
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID",
			PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID",
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList",
			PrivatePath:  "Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList",
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity",
		},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)

	paths, err := resolver.ResolveRadioMappings(context.Background(), "FAP/MLN/SC", "MLN_5.1.12.2", []int{1}, true, true)

	require.NoError(t, err)
	require.Equal(t, []RadioParameterPath{
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
			PrivatePath:  "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
			FAPInstance:  1, Kind: radioPathTAC,
		},
		{
			StandardPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
			PrivatePath:  "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
			FAPInstance:  1, Kind: radioPathServingPLMNs,
		},
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
			FAPInstance:  1, Kind: radioPathECI,
		},
	}, paths)
}

func TestProductGPSPathResolverSkipsDiscoveredInstancesWithoutCompleteLTEMappings(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
			PrivatePath:  "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
		},
		{
			StandardPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
			PrivatePath:  "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
		},
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
		},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)

	paths, err := resolver.ResolveRadioMappings(
		context.Background(), "SmallCell-LTE", "V1", []int{1, 2}, true, true,
	)

	require.NoError(t, err)
	require.Len(t, paths, 3)
	for _, path := range paths {
		require.Equal(t, 1, path.FAPInstance)
	}
}

func TestProductGPSPathResolverFailsWhenNoDiscoveredInstanceHasCompleteMappings(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
			PrivatePath:  "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
		},
	}}, nil, nil)
	resolver := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)

	_, err := resolver.ResolveRadioMappings(
		context.Background(), "SmallCell-LTE", "V1", []int{2}, true, true,
	)

	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
}
