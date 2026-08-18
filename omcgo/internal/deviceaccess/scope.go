package deviceaccess

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type IdentityVisibilityChecker interface {
	CanAccessIdentity(context.Context, string, string, []uuid.UUID) (bool, error)
}

// PgCarrierVisibilityChecker derives an operator/carrier scope from the
// project's canonical device-group visibility model. The request header is a
// filter only; it never grants a carrier by itself.
type PgCarrierVisibilityChecker struct {
	pool *pgxpool.Pool
}

func NewPgCarrierVisibilityChecker(pool *pgxpool.Pool) *PgCarrierVisibilityChecker {
	return &PgCarrierVisibilityChecker{pool: pool}
}

func (s *PgCarrierVisibilityChecker) CanAccessCarrier(
	ctx context.Context,
	carrier string,
	visibleGroups []uuid.UUID,
) (bool, error) {
	if visibleGroups == nil {
		return true, nil // built-in super administrator
	}
	if s == nil || s.pool == nil || len(visibleGroups) == 0 {
		return false, nil
	}
	query, args, err := buildCarrierVisibilityQuery(carrier, visibleGroups)
	if err != nil {
		return false, err
	}
	var one int
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&one); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("query carrier visibility: %w", err)
		}
		query, args, err = buildRegistrationCarrierVisibilityQuery(carrier, visibleGroups)
		if err != nil {
			return false, err
		}
		if err := s.pool.QueryRow(ctx, query, args...).Scan(&one); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, nil
			}
			return false, fmt.Errorf("query registration carrier visibility: %w", err)
		}
	}
	return true, nil
}

func (s *PgCarrierVisibilityChecker) CanAccessIdentity(
	ctx context.Context,
	carrier, serialNumber string,
	visibleGroups []uuid.UUID,
) (bool, error) {
	if visibleGroups == nil {
		return true, nil
	}
	if s == nil || s.pool == nil || len(visibleGroups) == 0 {
		return false, nil
	}
	formalQuery, formalArgs, err := buildFormalDeviceIdentityQuery(carrier, serialNumber)
	if err != nil {
		return false, err
	}
	var one int
	formalDeviceExists := true
	if err := s.pool.QueryRow(ctx, formalQuery, formalArgs...).Scan(&one); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("query formal device identity: %w", err)
		}
		formalDeviceExists = false
	}

	deviceQuery := storage.Psql.Select("1").From("devices scope_device").
		Where(sq.Expr("LOWER(scope_device.carrier) = LOWER(?)", carrier)).
		Where(sq.Eq{"scope_device.serial_number": serialNumber}).Limit(1)
	deviceQuery = authz.ApplyDeviceVisibilityFilter(deviceQuery, "scope_device.id", visibleGroups)
	query, args, err := deviceQuery.ToSql()
	if err != nil {
		return false, fmt.Errorf("build device identity visibility query: %w", err)
	}
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&one); err == nil {
		return true, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("query device identity visibility: %w", err)
	}
	// A formal device's current group membership is authoritative. Historical
	// preregistration rows must not restore access after the device moves out of
	// the actor's visible scope.
	if formalDeviceExists {
		return false, nil
	}

	realGroups, includeUngrouped := authz.SplitVisibleGroups(visibleGroups)
	registrationQuery := storage.Psql.Select("1").From("device_registrations scope_registration").
		Where(sq.Expr("LOWER(scope_registration.carrier) = LOWER(?)", carrier)).
		Where(sq.Eq{"scope_registration.serial_number": serialNumber}).Limit(1)
	var groupScope sq.Or
	if len(realGroups) > 0 {
		groupScope = append(groupScope, sq.Eq{"scope_registration.group_id": realGroups})
	}
	if includeUngrouped {
		groupScope = append(groupScope, sq.Expr("scope_registration.group_id IS NULL"))
	}
	if len(groupScope) == 0 {
		return false, nil
	}
	query, args, err = registrationQuery.Where(groupScope).ToSql()
	if err != nil {
		return false, fmt.Errorf("build registration identity visibility query: %w", err)
	}
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query registration identity visibility: %w", err)
	}
	return true, nil
}

func buildFormalDeviceIdentityQuery(carrier, serialNumber string) (string, []any, error) {
	query, args, err := storage.Psql.Select("1").From("devices scope_device").
		Where(sq.Expr("LOWER(scope_device.carrier) = LOWER(?)", carrier)).
		Where(sq.Eq{"scope_device.serial_number": serialNumber}).Limit(1).ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build formal device identity query: %w", err)
	}
	return query, args, nil
}

func authorizeIdentityScope(
	ctx context.Context,
	checker IdentityVisibilityChecker,
	actor PolicyActor,
	serialNumber string,
) error {
	if actor.VisibleGroups == nil {
		return nil
	}
	if checker == nil {
		return fmt.Errorf("authorize device access identity: %w", ErrAccessGateDependencyMissing)
	}
	allowed, err := checker.CanAccessIdentity(ctx, actor.Carrier, serialNumber, actor.VisibleGroups)
	if err != nil {
		return fmt.Errorf("authorize device access identity: %w", err)
	}
	if !allowed {
		return commonerrors.ErrForbidden
	}
	return nil
}

func buildCarrierVisibilityQuery(carrier string, visibleGroups []uuid.UUID) (string, []any, error) {
	builder := storage.Psql.Select("1").
		From("devices scope_device").
		Where(sq.Expr("LOWER(scope_device.carrier) = LOWER(?)", carrier)).
		Limit(1)
	builder = authz.ApplyDeviceVisibilityFilter(builder, "scope_device.id", visibleGroups)
	query, args, err := builder.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build carrier visibility query: %w", err)
	}
	return query, args, nil
}

func buildRegistrationCarrierVisibilityQuery(carrier string, visibleGroups []uuid.UUID) (string, []any, error) {
	realGroups, includeUngrouped := authz.SplitVisibleGroups(visibleGroups)
	groupScope := make(sq.Or, 0, 2)
	if len(realGroups) > 0 {
		groupScope = append(groupScope, sq.Eq{"scope_registration.group_id": realGroups})
	}
	if includeUngrouped {
		groupScope = append(groupScope, sq.Expr("scope_registration.group_id IS NULL"))
	}
	if len(groupScope) == 0 {
		return "", nil, commonerrors.ErrForbidden
	}
	query, args, err := storage.Psql.Select("1").From("device_registrations scope_registration").
		Where(sq.Expr("LOWER(scope_registration.carrier) = LOWER(?)", carrier)).
		Where(groupScope).Limit(1).ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build registration carrier visibility query: %w", err)
	}
	return query, args, nil
}

var _ CarrierVisibilityChecker = (*PgCarrierVisibilityChecker)(nil)
var _ IdentityVisibilityChecker = (*PgCarrierVisibilityChecker)(nil)
