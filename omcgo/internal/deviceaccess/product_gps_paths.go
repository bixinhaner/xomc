package deviceaccess

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/product"
)

const (
	gpsLatitudeStandardPath  = "Device.FAP.GPS.LockedLatitude"
	gpsLongitudeStandardPath = "Device.FAP.GPS.LockedLongitude"
	servingCellStandardRoot  = "Device.Services.FAPService."
)

type gpsPathSpec struct {
	EvidencePath       string
	ModelStandardPaths []string
}

func (r *ProductGPSPathResolver) ResolveServingCellRoot(
	ctx context.Context,
	productClass string,
	softwareVersion string,
) (string, error) {
	translator, err := r.translator(ctx, productClass, softwareVersion)
	if err != nil {
		return "", fmt.Errorf("resolve serving-cell parameter translator: %w", err)
	}
	candidates := translator.ToPrivateCandidates(servingCellStandardRoot)
	if len(candidates) == 1 && candidates[0].Found && strings.HasSuffix(candidates[0].Translated, ".") {
		return candidates[0].Translated, nil
	}
	// Some current product models contain only concrete FAPService.1/2/... leaf
	// mappings. Infer the object root only when every usable mapping points to
	// the same private prefix; any vendor-specific ambiguity remains fail-closed.
	if root, ok := inferServingCellRoot(translator.Mappings()); ok {
		return root, nil
	}
	return "", fmt.Errorf("serving-cell root is not mapped unambiguously for product %q: %w", productClass, ErrAccessEvidenceUnavailable)
}

func inferServingCellRoot(mappings []parammodel.ParamMapping) (string, bool) {
	roots := make(map[string]struct{})
	for _, mapping := range mappings {
		standard := strings.TrimSpace(mapping.StandardPath)
		if !strings.HasPrefix(standard, servingCellStandardRoot) {
			continue
		}
		remainder := strings.TrimPrefix(standard, servingCellStandardRoot)
		separator := strings.IndexByte(remainder, '.')
		if separator <= 0 {
			continue
		}
		instance := remainder[:separator]
		if value, err := strconv.Atoi(instance); err != nil || value <= 0 || value > maxServingCellInstances {
			continue
		}
		privatePath := strings.TrimSpace(mapping.PrivatePath)
		marker := "." + instance + "."
		markerIndex := strings.Index(privatePath, marker)
		if markerIndex < 0 || strings.Index(privatePath[markerIndex+len(marker):], marker) >= 0 {
			continue
		}
		roots[privatePath[:markerIndex+1]] = struct{}{}
		if len(roots) > 1 {
			return "", false
		}
	}
	for root := range roots {
		return root, root != "" && strings.HasSuffix(root, ".")
	}
	return "", false
}

// gpsPathSpecs keeps the access-evidence contract stable while accepting the
// standard-path aliases already present in product parameter models. In the BM
// model, for example, the canonical model path is Device.DeviceInfo.SAS.FAP.GPS
// while the CPE private path is Device.FAP.GPS.
var gpsPathSpecs = []gpsPathSpec{
	{
		EvidencePath: gpsLatitudeStandardPath,
		ModelStandardPaths: []string{
			gpsLatitudeStandardPath,
			"Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude",
		},
	},
	{
		EvidencePath: gpsLongitudeStandardPath,
		ModelStandardPaths: []string{
			gpsLongitudeStandardPath,
			"Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude",
		},
	},
	{EvidencePath: "Device.FAP.GPS.Height", ModelStandardPaths: []string{"Device.FAP.GPS.Height"}},
	{EvidencePath: "Device.FAP.GPS.NumberOfSatellites", ModelStandardPaths: []string{"Device.FAP.GPS.NumberOfSatellites"}},
	{EvidencePath: "Device.DeviceInfo.GPS_Status", ModelStandardPaths: []string{"Device.DeviceInfo.GPS_Status"}},
	{EvidencePath: "Device.DeviceInfo.GPS.horizontalAccuracy", ModelStandardPaths: []string{"Device.DeviceInfo.GPS.horizontalAccuracy"}},
	{EvidencePath: "Device.DeviceInfo.GPS.verticalAccuracy", ModelStandardPaths: []string{"Device.DeviceInfo.GPS.verticalAccuracy"}},
}

type GPSParameterPath struct {
	StandardPath string `json:"standard_path"`
	PrivatePath  string `json:"private_path"`
}

type ProductClassResolver interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

type ProductTranslatorResolver interface {
	Translator(ctx context.Context, productID uuid.UUID, softwareVersion string) (*parammodel.Translator, error)
}

type ProductGPSPathResolver struct {
	products    ProductClassResolver
	translators ProductTranslatorResolver
}

func NewProductGPSPathResolver(products ProductClassResolver, translators ProductTranslatorResolver) *ProductGPSPathResolver {
	return &ProductGPSPathResolver{products: products, translators: translators}
}

func (r *ProductGPSPathResolver) Resolve(ctx context.Context, productClass, softwareVersion string) ([]string, error) {
	mappings, err := r.ResolveMappings(ctx, productClass, softwareVersion)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		paths = append(paths, mapping.PrivatePath)
	}
	return paths, nil
}

func (r *ProductGPSPathResolver) ResolveMappings(
	ctx context.Context,
	productClass string,
	softwareVersion string,
) ([]GPSParameterPath, error) {
	translator, err := r.translator(ctx, productClass, softwareVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve GPS parameter translator: %w", err)
	}
	paths := make([]GPSParameterPath, 0, len(gpsPathSpecs))
	for _, spec := range gpsPathSpecs {
		for _, modelStandardPath := range spec.ModelStandardPaths {
			translated := translator.ToPrivate(modelStandardPath)
			if !translated.Found {
				continue
			}
			paths = append(paths, GPSParameterPath{
				StandardPath: spec.EvidencePath,
				PrivatePath:  translated.Translated,
			})
			break
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("GPS paths are not mapped for product %q: %w", productClass, ErrAccessEvidenceUnavailable)
	}
	return paths, nil
}

func (r *ProductGPSPathResolver) ResolveRadioMappings(
	ctx context.Context,
	productClass string,
	softwareVersion string,
	fapInstances []int,
	needTAC bool,
	needECGI bool,
) ([]RadioParameterPath, error) {
	translator, err := r.translator(ctx, productClass, softwareVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve radio parameter translator: %w", err)
	}
	instances := sortedPositiveInstances(fapInstances)
	if len(instances) == 0 || len(instances) > maxServingCellInstances {
		return nil, fmt.Errorf("invalid serving-cell instance set: %w", ErrAccessEvidenceUnavailable)
	}

	paths := make([]RadioParameterPath, 0, len(instances)*4)
	for _, instance := range instances {
		instancePaths := make([]RadioParameterPath, 0, 4)
		complete := true
		if needTAC {
			standard := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.EPC.TAC", instance)
			translated := translator.ToPrivate(standard)
			if !translated.Found {
				complete = false
			} else {
				instancePaths = append(instancePaths, RadioParameterPath{
					StandardPath: standard, PrivatePath: translated.Translated,
					FAPInstance: instance, Kind: radioPathTAC,
				})
			}
		}
		if needECGI {
			eciStandard := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.RAN.Common.CellIdentity", instance)
			eci := translator.ToPrivate(eciStandard)
			if !eci.Found {
				complete = false
			}

			plmnStandard := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.EPC.PLMNList.1.PLMNID", instance)
			primaryStandard := fmt.Sprintf("Device.Services.FAPService.%d.CellConfig.LTE.EPC.PLMNList.1.IsPrimary", instance)
			plmn := translator.ToPrivate(plmnStandard)
			primary := translator.ToPrivate(primaryStandard)
			if plmn.Found && primary.Found {
				instancePaths = append(instancePaths,
					RadioParameterPath{StandardPath: plmnStandard, PrivatePath: plmn.Translated, FAPInstance: instance, Kind: radioPathPLMN},
					RadioParameterPath{StandardPath: primaryStandard, PrivatePath: primary.Translated, FAPInstance: instance, Kind: radioPathPLMNPrimary},
				)
			} else {
				servingPLMNsStandard := fmt.Sprintf("Device.Services.FAPService.%d.FAPControl.LTE.Gateway.ExistPlmnidList", instance)
				servingPLMNs := translator.ToPrivate(servingPLMNsStandard)
				if !servingPLMNs.Found {
					complete = false
				} else {
					instancePaths = append(instancePaths, RadioParameterPath{
						StandardPath: servingPLMNsStandard, PrivatePath: servingPLMNs.Translated,
						FAPInstance: instance, Kind: radioPathServingPLMNs,
					})
				}
			}
			if eci.Found {
				instancePaths = append(instancePaths, RadioParameterPath{
					StandardPath: eciStandard, PrivatePath: eci.Translated,
					FAPInstance: instance, Kind: radioPathECI,
				})
			}
		}
		if complete {
			paths = append(paths, instancePaths...)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no discovered FAPService instance has complete radio evidence mappings: %w", ErrAccessEvidenceUnavailable)
	}
	return paths, nil
}

func (r *ProductGPSPathResolver) translator(
	ctx context.Context,
	productClass string,
	softwareVersion string,
) (*parammodel.Translator, error) {
	if r == nil || r.products == nil || r.translators == nil {
		return nil, ErrAccessEvidenceUnavailable
	}
	match, err := r.products.MatchProductClass(ctx, strings.TrimSpace(productClass))
	if err != nil {
		return nil, fmt.Errorf("match product class: %w", err)
	}
	if match == nil || match.Product == nil || match.Product.ParamModelID == nil {
		return nil, fmt.Errorf("product has no parameter model: %w", ErrAccessEvidenceUnavailable)
	}
	translator, err := r.translators.Translator(ctx, match.Product.ID, softwareVersion)
	if err != nil {
		return nil, fmt.Errorf("load product parameter translator: %w", err)
	}
	if translator == nil {
		return nil, fmt.Errorf("product parameter translator is nil: %w", ErrAccessEvidenceUnavailable)
	}
	return translator, nil
}

func sortedPositiveInstances(instances []int) []int {
	seen := make(map[int]struct{}, len(instances))
	for _, instance := range instances {
		if instance > 0 && instance <= maxServingCellInstances {
			seen[instance] = struct{}{}
		}
	}
	result := make([]int, 0, len(seen))
	for instance := range seen {
		result = append(result, instance)
	}
	slices.Sort(result)
	return result
}
