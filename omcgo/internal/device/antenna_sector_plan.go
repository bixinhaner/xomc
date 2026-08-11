package device

import (
	"context"
	"fmt"
	"sort"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// AntennaSectorPlan stores OMC-local planning values. Nil fields inherit the
// corresponding device-reported value.
type AntennaSectorPlan struct {
	DeviceID            uuid.UUID
	SectorNumber        int
	Azimuth             *float64
	AntennaHeight       *float64
	MechanicalDowntilt  *float64
	HorizontalBeamwidth *float64
	VerticalBeamwidth   *float64
}

type UpdateAntennaSectorPlanRequest struct {
	Azimuth             *float64 `json:"azimuth"`
	AntennaHeight       *float64 `json:"antenna_height"`
	MechanicalDowntilt  *float64 `json:"mechanical_downtilt"`
	HorizontalBeamwidth *float64 `json:"horizontal_beamwidth"`
	VerticalBeamwidth   *float64 `json:"vertical_beamwidth"`
}

type AntennaSectorPlanRepository interface {
	ListByDevice(ctx context.Context, deviceID uuid.UUID) ([]AntennaSectorPlan, error)
	Upsert(ctx context.Context, plan AntennaSectorPlan) error
}

type PgAntennaSectorPlanRepository struct {
	pool *pgxpool.Pool
}

func NewPgAntennaSectorPlanRepository(pool *pgxpool.Pool) *PgAntennaSectorPlanRepository {
	return &PgAntennaSectorPlanRepository{pool: pool}
}

func (r *PgAntennaSectorPlanRepository) ListByDevice(
	ctx context.Context,
	deviceID uuid.UUID,
) ([]AntennaSectorPlan, error) {
	query, args, err := storage.Psql.
		Select(
			"device_id", "sector_no", "azimuth_deg", "antenna_height_m",
			"mechanical_downtilt_deg", "horizontal_beamwidth_deg",
			"vertical_beamwidth_deg",
		).
		From("device_antenna_sector_plans").
		Where(sq.Eq{"device_id": deviceID}).
		OrderBy("sector_no ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list antenna sector plans: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list antenna sector plans: %w", err)
	}
	defer rows.Close()

	plans := make([]AntennaSectorPlan, 0)
	for rows.Next() {
		var plan AntennaSectorPlan
		if err := rows.Scan(
			&plan.DeviceID,
			&plan.SectorNumber,
			&plan.Azimuth,
			&plan.AntennaHeight,
			&plan.MechanicalDowntilt,
			&plan.HorizontalBeamwidth,
			&plan.VerticalBeamwidth,
		); err != nil {
			return nil, fmt.Errorf("scan antenna sector plan: %w", err)
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate antenna sector plans: %w", err)
	}
	return plans, nil
}

func (r *PgAntennaSectorPlanRepository) Upsert(ctx context.Context, plan AntennaSectorPlan) error {
	query, args, err := storage.Psql.
		Insert("device_antenna_sector_plans").
		Columns(
			"device_id", "sector_no", "azimuth_deg", "antenna_height_m",
			"mechanical_downtilt_deg", "horizontal_beamwidth_deg",
			"vertical_beamwidth_deg",
		).
		Values(
			plan.DeviceID,
			plan.SectorNumber,
			plan.Azimuth,
			plan.AntennaHeight,
			plan.MechanicalDowntilt,
			plan.HorizontalBeamwidth,
			plan.VerticalBeamwidth,
		).
		Suffix(`ON CONFLICT (device_id, sector_no) DO UPDATE SET
    azimuth_deg = EXCLUDED.azimuth_deg,
    antenna_height_m = EXCLUDED.antenna_height_m,
    mechanical_downtilt_deg = EXCLUDED.mechanical_downtilt_deg,
    horizontal_beamwidth_deg = EXCLUDED.horizontal_beamwidth_deg,
    vertical_beamwidth_deg = EXCLUDED.vertical_beamwidth_deg,
    updated_at = NOW()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert antenna sector plan: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert antenna sector plan: %w", err)
	}
	return nil
}

func mergeAntennaSectorPlans(sectors []AntennaSector, plans []AntennaSectorPlan) []AntennaSector {
	byNumber := make(map[int]AntennaSector, len(sectors)+len(plans))
	for _, sector := range sectors {
		byNumber[sector.Number] = sector
	}
	for _, plan := range plans {
		sector, ok := byNumber[plan.SectorNumber]
		if !ok {
			sector = AntennaSector{Number: plan.SectorNumber, FieldSources: map[string]string{}}
		}
		applyPlannedValue(&sector.Azimuth, plan.Azimuth, sector.FieldSources, "azimuth")
		applyPlannedValue(&sector.AntennaHeight, plan.AntennaHeight, sector.FieldSources, "height")
		applyPlannedValue(&sector.MechanicalDowntilt, plan.MechanicalDowntilt, sector.FieldSources, "downtilt")
		applyPlannedValue(&sector.HorizontalBeamwidth, plan.HorizontalBeamwidth, sector.FieldSources, "beamwidth")
		applyPlannedValue(&sector.VerticalBeamwidth, plan.VerticalBeamwidth, sector.FieldSources, "verticalbeamwidth")
		validateAntennaSector(&sector)
		byNumber[plan.SectorNumber] = sector
	}

	numbers := make([]int, 0, len(byNumber))
	for number := range byNumber {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	result := make([]AntennaSector, 0, len(numbers))
	for _, number := range numbers {
		result = append(result, byNumber[number])
	}
	return result
}

func applyPlannedValue(target **float64, planned *float64, sources map[string]string, field string) {
	if planned == nil {
		return
	}
	*target = planned
	sources[field] = "planned"
}
