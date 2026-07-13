package collector

import (
	"context"
	"fmt"
	"math"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/resultnorm"
)

// NumberProcessLookup reads sys_configs indicator.process.number.
// Returning an empty value is allowed and lets resultnorm use its default policy.
type NumberProcessLookup func(ctx context.Context) (string, error)

func (c *PMCollector) normalizeResults(ctx context.Context, counters []model.PMCounter, kpis []model.KPIValue) error {
	numberProcess := ""
	if c.numberProcessLookup != nil {
		value, err := c.numberProcessLookup(ctx)
		if err != nil {
			return fmt.Errorf("read %s: %w", resultnorm.ConfigName, err)
		}
		numberProcess = value
	}

	for i := range counters {
		if math.IsNaN(counters[i].CounterValue) {
			continue
		}
		normalized, err := resultnorm.Normalize(counters[i].CounterValue, &resultnorm.Metadata{
			Unit:       counters[i].Unit,
			StatisType: counters[i].StatisType,
		}, numberProcess)
		if err != nil {
			return fmt.Errorf("normalize PM counter result %s: %w", counters[i].CounterName, err)
		}
		counters[i].CounterValue = normalized
	}

	for i := range kpis {
		normalized, err := resultnorm.Normalize(kpis[i].KPIValue, &resultnorm.Metadata{
			Unit:       kpis[i].Unit,
			StatisType: kpis[i].StatisType,
		}, numberProcess)
		if err != nil {
			return fmt.Errorf("normalize PM KPI result %s: %w", kpis[i].IndicatorID, err)
		}
		kpis[i].KPIValue = normalized
	}
	return nil
}
