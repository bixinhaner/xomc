package kpi

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/carrier"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"go.uber.org/zap"
)

// RegisteredFormula holds a parsed formula alongside its definition metadata.
type RegisteredFormula struct {
	Name        string
	DisplayName string
	Parsed      *Formula
	Counters    []string
	Unit        string
	Category    string
	Carrier     model.CarrierCode
	Technology  model.Technology
}

// KPIEngine calculates KPI values from PM counters using registered formulas.
type KPIEngine struct {
	formulas        []*RegisteredFormula
	counterRepo     counter.CounterRepository
	kpiRepo         KPIRepository
	carrierRegistry *carrier.CarrierRegistry
	logger          *zap.Logger
}

// NewKPIEngine creates a new KPI engine and loads formulas from all registered carriers.
func NewKPIEngine(
	counterRepo counter.CounterRepository,
	kpiRepo KPIRepository,
	carrierRegistry *carrier.CarrierRegistry,
	logger *zap.Logger,
) *KPIEngine {
	e := &KPIEngine{
		counterRepo:     counterRepo,
		kpiRepo:         kpiRepo,
		carrierRegistry: carrierRegistry,
		logger:          logger,
	}
	e.loadFormulas()
	return e
}

func (e *KPIEngine) loadFormulas() {
	for _, c := range e.carrierRegistry.All() {
		for _, tech := range c.SupportedTechnologies() {
			defs := c.KPIDefinitions(tech)
			for _, d := range defs {
				parsed, err := ParseFormula(d.Formula)
				if err != nil {
					e.logger.Warn("skip invalid kpi formula",
						zap.String("name", d.Name),
						zap.String("formula", d.Formula),
						zap.Error(err),
					)
					continue
				}
				e.formulas = append(e.formulas, &RegisteredFormula{
					Name:        d.Name,
					DisplayName: d.DisplayName,
					Parsed:      parsed,
					Counters:    d.Counters,
					Unit:        d.Unit,
					Category:    d.Category,
					Carrier:     c.Code(),
					Technology:  tech,
				})
			}
		}
	}
	e.logger.Info("KPI formulas loaded", zap.Int("count", len(e.formulas)))
}

// Formulas returns all registered formulas.
func (e *KPIEngine) Formulas() []*RegisteredFormula {
	return e.formulas
}

// Calculate computes KPI values for a device/cell in a time range.
func (e *KPIEngine) Calculate(
	ctx context.Context,
	deviceID uuid.UUID,
	cellID string,
	startTime, endTime time.Time,
	carrierCode model.CarrierCode,
	tech model.Technology,
) ([]model.KPIValue, error) {
	// Collect all needed counter names
	var allCounters []string
	applicable := e.applicableFormulas(carrierCode, tech)
	for _, f := range applicable {
		allCounters = append(allCounters, f.Counters...)
	}

	// Query counter values
	counterValues, err := e.counterRepo.QueryForKPI(ctx, deviceID, cellID, allCounters, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query counters for kpi: %w", err)
	}

	// Evaluate each formula
	var results []model.KPIValue
	for _, f := range applicable {
		value, err := f.Parsed.Evaluate(counterValues)
		if err != nil {
			// Skip formulas with missing counters or division by zero
			continue
		}
		results = append(results, model.KPIValue{
			Time:       endTime,
			DeviceID:   deviceID,
			CellID:     cellID,
			KPIName:    f.Name,
			KPIValue:   value,
			Carrier:    carrierCode,
			Technology: tech,
		})
	}

	return results, nil
}

// CalculateAndStore calculates KPIs and persists them.
func (e *KPIEngine) CalculateAndStore(
	ctx context.Context,
	deviceID uuid.UUID,
	cellID string,
	collectTime time.Time,
	carrierCode model.CarrierCode,
	tech model.Technology,
) ([]model.KPIValue, error) {
	// Use a time window around the collection time
	startTime := collectTime.Add(-time.Duration(15) * time.Minute)
	endTime := collectTime

	results, err := e.Calculate(ctx, deviceID, cellID, startTime, endTime, carrierCode, tech)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		if err := e.kpiRepo.BatchInsert(ctx, results); err != nil {
			return nil, fmt.Errorf("store kpi values: %w", err)
		}
	}

	return results, nil
}

func (e *KPIEngine) applicableFormulas(carrierCode model.CarrierCode, tech model.Technology) []*RegisteredFormula {
	var result []*RegisteredFormula
	for _, f := range e.formulas {
		if f.Carrier == carrierCode && f.Technology == tech {
			result = append(result, f)
		}
	}
	return result
}
