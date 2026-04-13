package device

// This file demonstrates using goqu for dynamic query building as an alternative
// to squirrel. The goqu approach provides better composability for complex filter
// combinations. This is additive — the original squirrel-based List method remains
// active and can be swapped incrementally.
//
// To migrate, replace the List method in PgDeviceRepository with a version that
// calls buildDeviceFilterGoqu instead of the inline squirrel builder.

import (
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// buildDeviceFilterGoqu constructs dynamic WHERE conditions for device listing
// using goqu's expression builder. Each filter is applied only when non-nil.
func buildDeviceFilterGoqu(filter DeviceFilter) []goqu.Expression {
	var conds []goqu.Expression

	// Soft-delete filter
	conds = append(conds, goqu.C("deleted_at").IsNull())

	if filter.Carrier != nil {
		conds = append(conds, goqu.C("carrier").Eq(string(*filter.Carrier)))
	}
	if filter.Technology != nil {
		conds = append(conds, goqu.C("technology").Eq(string(*filter.Technology)))
	}
	if filter.Status != nil {
		conds = append(conds, goqu.C("status").Eq(string(*filter.Status)))
	}
	if filter.OUI != nil {
		conds = append(conds, goqu.C("oui").Eq(*filter.OUI))
	}
	if filter.SN != nil && *filter.SN != "" {
		conds = append(conds, goqu.C("serial_number").Eq(*filter.SN))
	}
	if filter.Search != nil && *filter.Search != "" {
		pattern := "%" + *filter.Search + "%"
		conds = append(conds, goqu.Or(
			goqu.C("serial_number").ILike(pattern),
			goqu.C("site_name").ILike(pattern),
		))
	}

	// Extended filters (would need JOINs with device_info in a full implementation)
	if filter.Manufacturer != nil {
		conds = append(conds, goqu.C("manufacturer").Eq(*filter.Manufacturer))
	}
	if filter.ProductClass != nil {
		conds = append(conds, goqu.C("product_class").Eq(*filter.ProductClass))
	}
	if filter.OpState != nil {
		if *filter.OpState == "1" {
			conds = append(conds, goqu.C("status").Eq(model.DeviceActive))
		} else {
			conds = append(conds, goqu.C("status").Neq(model.DeviceActive))
		}
	}

	return conds
}

// buildDeviceFromGoqu creates the base goqu SelectDataset for device queries.
// This demonstrates the goqu pattern; a full implementation would handle
// conditional JOINs for group filtering.
func buildDeviceFromGoqu() *goqu.SelectDataset {
	return storage.GoquDialect.From(goqu.T("devices").As("d"))
}

// buildGroupFilterGoqu creates the group-based WHERE conditions.
func buildGroupFilterGoqu(filter DeviceFilter) (exp.Expression, *goqu.SelectDataset) {
	from := buildDeviceFromGoqu()

	if filter.GroupID != nil {
		from = from.Join(
			goqu.T("device_group_members").As("dgm"),
			goqu.On(goqu.C("d.id").Eq(goqu.I("dgm.device_id"))),
		).Where(goqu.C("dgm.group_id").Eq(*filter.GroupID))
	}

	if len(filter.VisibleGroups) > 0 {
		if filter.GroupID == nil {
			from = from.Join(
				goqu.T("device_group_members").As("dgm2"),
				goqu.On(goqu.C("d.id").Eq(goqu.I("dgm2.device_id"))),
			)
			groupIDs := make([]interface{}, len(filter.VisibleGroups))
			for i, id := range filter.VisibleGroups {
				groupIDs[i] = id
			}
			return goqu.C("dgm2.group_id").In(groupIDs...), from
		}
		// When both GroupID and VisibleGroups exist, use EXISTS subquery
		groupIDs := make([]interface{}, len(filter.VisibleGroups))
		for i, id := range filter.VisibleGroups {
			groupIDs[i] = id
		}
		subQuery := storage.GoquDialect.From(goqu.T("device_group_members").As("dgm_vis")).
			Select("device_id").
			Where(goqu.C("dgm_vis.group_id").In(groupIDs...))
		return goqu.C("d.id").In(subQuery), from
	}

	return nil, from
}
