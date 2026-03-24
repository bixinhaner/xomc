package datamodel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ImportValidationResult contains the result of a dry-run import validation.
type ImportValidationResult struct {
	Valid          bool     `json:"valid"`
	Carrier        string   `json:"carrier"`
	Technology     string   `json:"technology"`
	Version        string   `json:"version"`
	OUI            string   `json:"oui,omitempty"`
	ProductClass   string   `json:"product_class,omitempty"`
	Scope          string   `json:"scope"`
	ParameterCount int      `json:"parameter_count"`
	Errors         []string `json:"errors,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

// importPayload represents the JSON structure used for import/export.
type importPayload struct {
	Carrier         string          `json:"carrier"`
	Technology      string          `json:"technology"`
	Version         string          `json:"version"`
	OUI             string          `json:"oui,omitempty"`
	ProductClass    string          `json:"product_class,omitempty"`
	RootObject      string          `json:"root_object,omitempty"`
	ParameterTree   json.RawMessage `json:"parameter_tree"`
	Source          string          `json:"source,omitempty"`
	SpecDocumentRef string          `json:"spec_document_ref,omitempty"`
	Description     string          `json:"description,omitempty"`
}

// DataModelImporter handles import and export of data model definitions.
type DataModelImporter struct {
	repo      DataModelRepository
	logRepo   ImportLogRepository
}

// NewDataModelImporter creates a new DataModelImporter.
func NewDataModelImporter(repo DataModelRepository, logRepo ImportLogRepository) *DataModelImporter {
	return &DataModelImporter{
		repo:    repo,
		logRepo: logRepo,
	}
}

// ImportFromJSON reads a JSON payload, creates a data model in draft status,
// and logs the import action. The scope is auto-detected from the oui and product_class fields.
func (imp *DataModelImporter) ImportFromJSON(ctx context.Context, reader io.Reader, importedBy string) (*DataModel, error) {
	payload, err := parseImportPayload(reader)
	if err != nil {
		return nil, fmt.Errorf("parse import payload: %w", err)
	}

	if errs := validatePayload(payload); len(errs) > 0 {
		return nil, fmt.Errorf("validate import: %s", errs[0])
	}

	scope := determineScope(payload.OUI, payload.ProductClass)
	rootObject := payload.RootObject
	if rootObject == "" {
		rootObject = "Device."
	}

	dm := &DataModel{
		Carrier:         model.CarrierCode(payload.Carrier),
		Technology:      model.Technology(payload.Technology),
		Version:         payload.Version,
		OUI:             payload.OUI,
		ProductClass:    payload.ProductClass,
		Scope:           scope,
		Status:          StatusDraft,
		IsActive:        false,
		RootObject:      rootObject,
		ParameterTree:   payload.ParameterTree,
		Source:          payload.Source,
		ImportedBy:      importedBy,
		SpecDocumentRef: payload.SpecDocumentRef,
		Description:     payload.Description,
	}

	if err := imp.repo.Create(ctx, dm); err != nil {
		return nil, fmt.Errorf("create imported data model: %w", err)
	}

	logEntry := &ImportLogEntry{
		DataModelID: dm.ID,
		Action:      "imported",
		PerformedBy: importedBy,
	}
	if err := imp.logRepo.Create(ctx, logEntry); err != nil {
		// Log error but don't fail the import.
		return dm, nil
	}

	return dm, nil
}

// ValidateImport performs a dry-run validation of a JSON import payload without
// persisting anything to the database.
func (imp *DataModelImporter) ValidateImport(ctx context.Context, reader io.Reader) (*ImportValidationResult, error) {
	payload, err := parseImportPayload(reader)
	if err != nil {
		return &ImportValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("parse error: %v", err)},
		}, nil
	}

	result := &ImportValidationResult{
		Carrier:    payload.Carrier,
		Technology: payload.Technology,
		Version:    payload.Version,
		OUI:        payload.OUI,
		ProductClass: payload.ProductClass,
	}

	result.Errors = validatePayload(payload)
	result.Scope = string(determineScope(payload.OUI, payload.ProductClass))

	// Count parameters in the tree.
	if len(payload.ParameterTree) > 0 {
		var tree interface{}
		if json.Unmarshal(payload.ParameterTree, &tree) == nil {
			result.ParameterCount = countParameters(tree)
		}
	}

	// Warnings for missing optional fields.
	if payload.Description == "" {
		result.Warnings = append(result.Warnings, "description is empty")
	}
	if payload.Source == "" {
		result.Warnings = append(result.Warnings, "source is not specified")
	}

	result.Valid = len(result.Errors) == 0
	return result, nil
}

// ImportFromXML reads a CPE-exported parameter model XML, creates a data model
// in draft status, and logs the import action. The carrier must be provided
// externally since the XML does not contain carrier information.
func (imp *DataModelImporter) ImportFromXML(ctx context.Context, reader io.Reader, carrier model.CarrierCode, importedBy string) (*DataModel, error) {
	parsed, err := ParseParameterModelXML(reader)
	if err != nil {
		return nil, fmt.Errorf("parse XML: %w", err)
	}

	if errs := ValidateXMLModel(parsed); len(errs) > 0 {
		return nil, fmt.Errorf("validate XML: %s", errs[0])
	}

	if !carrier.IsValid() {
		return nil, fmt.Errorf("invalid carrier: %s", carrier)
	}

	dm, err := buildDataModelFromParsed(parsed, carrier)
	if err != nil {
		return nil, fmt.Errorf("build data model from XML: %w", err)
	}
	dm.SourceType = SourceManual
	dm.ImportedBy = importedBy

	if err := imp.repo.Create(ctx, dm); err != nil {
		return nil, fmt.Errorf("create data model from XML: %w", err)
	}

	logEntry := &ImportLogEntry{
		DataModelID: dm.ID,
		Action:      "imported",
		PerformedBy: importedBy,
	}
	if err := imp.logRepo.Create(ctx, logEntry); err != nil {
		return dm, nil
	}

	return dm, nil
}

// ImportFromXMLForCPE creates or updates a data model from a CPE-uploaded XML.
// Unlike ImportFromXML, this auto-activates the model and uses SourceCPEUploaded.
func (imp *DataModelImporter) ImportFromXMLForCPE(ctx context.Context, reader io.Reader, carrier model.CarrierCode, productClass, firmwareVersion string) (*DataModel, error) {
	parsed, err := ParseParameterModelXML(reader)
	if err != nil {
		return nil, fmt.Errorf("parse XML: %w", err)
	}

	if errs := ValidateXMLModel(parsed); len(errs) > 0 {
		return nil, fmt.Errorf("validate XML: %s", errs[0])
	}

	dm, err := buildDataModelFromParsed(parsed, carrier)
	if err != nil {
		return nil, fmt.Errorf("build data model from XML: %w", err)
	}
	dm.SourceType = SourceCPEUploaded
	dm.ProductClass = productClass
	dm.FirmwareVersion = firmwareVersion
	dm.Status = StatusActive
	dm.IsActive = true
	dm.ImportedBy = "cpe_upload"

	if err := imp.repo.Create(ctx, dm); err != nil {
		return nil, fmt.Errorf("create data model from CPE XML: %w", err)
	}

	logEntry := &ImportLogEntry{
		DataModelID: dm.ID,
		Action:      "imported",
		PerformedBy: "cpe_upload",
	}
	_ = imp.logRepo.Create(ctx, logEntry)

	return dm, nil
}

// buildDataModelFromParsed converts a ParsedParameterModel into a DataModel.
func buildDataModelFromParsed(parsed *ParsedParameterModel, carrier model.CarrierCode) (*DataModel, error) {
	// Map network type to technology.
	tech := mapNetworkType(parsed.NetworkType)

	// Build parameter tree JSON.
	paramTree, err := json.Marshal(parsed.Parameters)
	if err != nil {
		return nil, fmt.Errorf("marshal parameters: %w", err)
	}

	// Build object tree JSON.
	var objectTree json.RawMessage
	if len(parsed.Objects) > 0 {
		objData, err := json.Marshal(parsed.Objects)
		if err != nil {
			return nil, fmt.Errorf("marshal objects: %w", err)
		}
		objectTree = objData
	}

	// Build model metadata JSON.
	metadata := map[string]interface{}{
		"generate_time": parsed.GenerateTime,
		"model_version": parsed.ModelVersion,
		"total_entries": parsed.TotalEntries,
		"serial_number": parsed.SerialNumber,
	}
	metadataJSON, _ := json.Marshal(metadata)

	scope := determineScope(parsed.Vendor, "")
	rootObject := detectRootObjectFromParams(parsed.Parameters)

	dm := &DataModel{
		ID:            uuid.New(),
		Carrier:       carrier,
		Technology:    tech,
		Version:       parsed.ModelVersion,
		OUI:           parsed.Vendor,
		Scope:         scope,
		Status:        StatusDraft,
		IsActive:      false,
		RootObject:    rootObject,
		ParameterTree: paramTree,
		ObjectTree:    objectTree,
		ModelMetadata: json.RawMessage(metadataJSON),
		Source:        fmt.Sprintf("xml_upload:%s", parsed.SerialNumber),
		Description:   fmt.Sprintf("参数模型上传: %s %s (SN: %s)", parsed.Vendor, parsed.NetworkType, parsed.SerialNumber),
	}

	return dm, nil
}

// mapNetworkType converts XML networkType to internal Technology code.
func mapNetworkType(networkType string) model.Technology {
	switch model.Technology(strings.ToLower(networkType)) {
	case model.TechNR:
		return model.TechNR
	default:
		return model.TechLTE
	}
}

// detectRootObjectFromParams determines the root object from parameters.
func detectRootObjectFromParams(params []Parameter) string {
	for _, p := range params {
		if strings.HasPrefix(p.Path, "InternetGatewayDevice.") {
			return "InternetGatewayDevice."
		}
	}
	return "Device."
}

// ExportToJSON retrieves a data model by ID and marshals it as JSON.
func (imp *DataModelImporter) ExportToJSON(ctx context.Context, id uuid.UUID) ([]byte, error) {
	dm, err := imp.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get data model for export: %w", err)
	}

	payload := importPayload{
		Carrier:         string(dm.Carrier),
		Technology:      string(dm.Technology),
		Version:         dm.Version,
		OUI:             dm.OUI,
		ProductClass:    dm.ProductClass,
		RootObject:      dm.RootObject,
		ParameterTree:   dm.ParameterTree,
		Source:          dm.Source,
		SpecDocumentRef: dm.SpecDocumentRef,
		Description:     dm.Description,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal data model for export: %w", err)
	}

	return data, nil
}

// parseImportPayload reads and decodes a JSON import payload from the reader.
func parseImportPayload(reader io.Reader) (*importPayload, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	var payload importPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	return &payload, nil
}

// validatePayload checks the import payload for required fields and valid values.
func validatePayload(p *importPayload) []string {
	var errs []string

	if p.Carrier == "" {
		errs = append(errs, "carrier is required")
	} else if !model.CarrierCode(p.Carrier).IsValid() {
		errs = append(errs, fmt.Sprintf("invalid carrier: %s", p.Carrier))
	}

	if p.Technology == "" {
		errs = append(errs, "technology is required")
	} else if !model.Technology(p.Technology).IsValid() {
		errs = append(errs, fmt.Sprintf("invalid technology: %s", p.Technology))
	}

	if p.Version == "" {
		errs = append(errs, "version is required")
	}

	if len(p.ParameterTree) == 0 {
		errs = append(errs, "parameter_tree is required")
	} else if !json.Valid(p.ParameterTree) {
		errs = append(errs, "parameter_tree is not valid JSON")
	}

	// Scope consistency: product scope requires both oui and product_class.
	if p.ProductClass != "" && p.OUI == "" {
		errs = append(errs, "oui is required when product_class is specified")
	}

	return errs
}

// determineScope auto-detects the scope from OUI and ProductClass values.
func determineScope(oui, productClass string) model.DataModelScope {
	if oui != "" && productClass != "" {
		return model.ScopeProduct
	}
	if oui != "" {
		return model.ScopeOUI
	}
	return model.ScopeCarrierDefault
}

// countParameters recursively counts leaf nodes in a parameter tree.
func countParameters(v interface{}) int {
	switch val := v.(type) {
	case map[string]interface{}:
		count := 0
		for _, child := range val {
			count += countParameters(child)
		}
		if count == 0 {
			return 1
		}
		return count
	case []interface{}:
		count := 0
		for _, item := range val {
			count += countParameters(item)
		}
		return count
	default:
		return 1
	}
}
