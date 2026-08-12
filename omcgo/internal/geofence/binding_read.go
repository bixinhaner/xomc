package geofence

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

const (
	DefaultBindingPageSize = 50
	MaxBindingPageSize     = 200
	BindingStatusCurrent   = "current"
)

type BindingDetail struct {
	Binding
	DeviceSN        string                  `json:"device_sn"`
	DeviceName      string                  `json:"device_name"`
	DeviceCarrier   string                  `json:"device_carrier"`
	DeviceGroupID   *uuid.UUID              `json:"device_group_id,omitempty"`
	DeviceGroupName string                  `json:"device_group_name,omitempty"`
	HasLocation     bool                    `json:"has_location"`
	Evaluation      BindingEvaluationDetail `json:"evaluation"`
}

// BindingEvaluationDetail is the latest read-only Observe state for one
// device/geofence binding. It intentionally excludes raw coordinates.
type BindingEvaluationDetail struct {
	ConfirmedState         ConfirmedState `json:"confirmed_state"`
	CandidateState         CandidateState `json:"candidate_state,omitempty"`
	CandidateCount         int            `json:"candidate_count"`
	LastObservationVersion int64          `json:"last_observation_version,omitempty"`
	LastObservedAt         *time.Time     `json:"last_observed_at,omitempty"`
	LastDistanceToBoundary *float64       `json:"last_distance_to_boundary,omitempty"`
	EvaluationHealth       string         `json:"evaluation_health"`
	ErrorCode              string         `json:"error_code,omitempty"`
}

type BindingDetailFilter struct {
	GeofenceID    uuid.UUID
	Status        BindingStatus
	CurrentOnly   bool
	Keyword       string
	Page          int
	PageSize      int
	VisibleGroups []uuid.UUID
}

type BindingDetailPage struct {
	Items    []BindingDetail `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

func normalizeBindingDetailFilter(
	filter BindingDetailFilter,
) (BindingDetailFilter, error) {
	if filter.GeofenceID == uuid.Nil {
		return BindingDetailFilter{}, fmt.Errorf(
			"geofence is required: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if filter.VisibleGroups != nil && len(filter.VisibleGroups) == 0 {
		return BindingDetailFilter{}, commonerrors.ErrForbidden
	}
	if filter.CurrentOnly && filter.Status != "" {
		return BindingDetailFilter{}, fmt.Errorf(
			"binding current scope and status are mutually exclusive: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	switch filter.Status {
	case "", BindingStatusActive, BindingStatusSuspended, BindingStatusRemoved:
	default:
		return BindingDetailFilter{}, fmt.Errorf(
			"unsupported binding status %q: %w",
			filter.Status,
			commonerrors.ErrInvalidInput,
		)
	}
	if filter.Page < 0 || filter.PageSize < 0 ||
		filter.PageSize > MaxBindingPageSize {
		return BindingDetailFilter{}, fmt.Errorf(
			"invalid binding pagination: %w",
			commonerrors.ErrInvalidInput,
		)
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = DefaultBindingPageSize
	}
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	return filter, nil
}

func buildListBindingDetailsQuery(
	filter BindingDetailFilter,
) (string, []any, error) {
	builder := storage.Psql.Select(
		"binding.id",
		"binding.device_id",
		"binding.geofence_id",
		"binding.rule_type",
		"binding.status",
		"binding.bind_source",
		"binding.bound_by",
		"binding.bound_at",
		"binding.removed_by",
		"binding.removed_at",
		"COALESCE(binding.remove_reason, '')",
		"d.serial_number",
		"COALESCE(device_info.device_name, d.site_name, '')",
		"d.carrier",
		"device_group.id",
		"COALESCE(device_group.name, '')",
		"(d.latitude IS NOT NULL AND d.longitude IS NOT NULL)",
		"COALESCE(binding_state.confirmed_state, 'unknown')",
		"COALESCE(binding_state.candidate_state, '')",
		"COALESCE(binding_state.candidate_count, 0)",
		"COALESCE(binding_state.last_observation_version, 0)",
		"binding_state.last_observed_at",
		"binding_state.last_distance_to_boundary",
		"COALESCE(effective_state.evaluation_health, 'healthy')",
		"COALESCE(effective_state.last_evaluation_error_code, '')",
		"COUNT(*) OVER()",
	).From("device_geofence_bindings binding").
		Join("devices d ON d.id = binding.device_id").
		LeftJoin("device_info ON device_info.device_id = d.id").
		LeftJoin(
			"device_geofence_states binding_state " +
				"ON binding_state.binding_id = binding.id",
		).
		LeftJoin(
			"device_geofence_effective_states effective_state " +
				"ON effective_state.device_id = binding.device_id",
		).
		LeftJoin(
			"LATERAL (" +
				"SELECT selected_group.id, selected_group.name " +
				"FROM device_group_members membership " +
				"JOIN device_groups selected_group " +
				"ON selected_group.id = membership.group_id " +
				"WHERE membership.device_id = d.id " +
				"ORDER BY selected_group.sort_order, selected_group.id " +
				"LIMIT 1" +
				") device_group ON TRUE",
		).
		Where(sq.Eq{"binding.geofence_id": filter.GeofenceID}).
		Where("d.deleted_at IS NULL")
	if filter.CurrentOnly {
		builder = builder.Where(sq.Eq{"binding.status": []BindingStatus{
			BindingStatusActive,
			BindingStatusSuspended,
		}})
	} else if filter.Status != "" {
		builder = builder.Where(sq.Eq{"binding.status": filter.Status})
	}
	if filter.Keyword != "" {
		pattern := "%" + filter.Keyword + "%"
		builder = builder.Where(sq.Or{
			sq.ILike{"d.serial_number": pattern},
			sq.ILike{"device_info.device_name": pattern},
			sq.ILike{"d.site_name": pattern},
		})
	}
	builder = authz.ApplyDeviceVisibilityFilter(
		builder,
		"d.id",
		filter.VisibleGroups,
	)
	return builder.
		OrderBy("binding.bound_at DESC", "binding.id").
		Limit(uint64(filter.PageSize)).
		Offset(uint64((filter.Page - 1) * filter.PageSize)).
		ToSql()
}

func (r *PgRepository) ListBindingDetails(
	ctx context.Context,
	filter BindingDetailFilter,
) (BindingDetailPage, error) {
	query, args, err := buildListBindingDetailsQuery(filter)
	if err != nil {
		return BindingDetailPage{}, fmt.Errorf(
			"build list geofence binding details: %w",
			err,
		)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return BindingDetailPage{}, fmt.Errorf(
			"list geofence binding details: %w",
			err,
		)
	}
	defer rows.Close()

	page := BindingDetailPage{
		Items:    make([]BindingDetail, 0),
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	for rows.Next() {
		var item BindingDetail
		var total int64
		if err := rows.Scan(
			&item.ID,
			&item.DeviceID,
			&item.GeofenceID,
			&item.RuleType,
			&item.Status,
			&item.BindSource,
			&item.BoundBy,
			&item.BoundAt,
			&item.RemovedBy,
			&item.RemovedAt,
			&item.RemoveReason,
			&item.DeviceSN,
			&item.DeviceName,
			&item.DeviceCarrier,
			&item.DeviceGroupID,
			&item.DeviceGroupName,
			&item.HasLocation,
			&item.Evaluation.ConfirmedState,
			&item.Evaluation.CandidateState,
			&item.Evaluation.CandidateCount,
			&item.Evaluation.LastObservationVersion,
			&item.Evaluation.LastObservedAt,
			&item.Evaluation.LastDistanceToBoundary,
			&item.Evaluation.EvaluationHealth,
			&item.Evaluation.ErrorCode,
			&total,
		); err != nil {
			return BindingDetailPage{}, fmt.Errorf(
				"scan geofence binding detail: %w",
				err,
			)
		}
		page.Total = total
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return BindingDetailPage{}, fmt.Errorf(
			"iterate geofence binding details: %w",
			err,
		)
	}
	return page, nil
}
