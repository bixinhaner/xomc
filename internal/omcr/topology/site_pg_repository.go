package topology

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
)

// ======================================================================
// Column lists
// ======================================================================

var siteColumns = []string{
	"id", "name", "domain_id", "address", "longitude", "latitude",
	"device_count", "status", "created_at", "updated_at",
}

var topoNodeColumns = []string{
	"id", "label", "node_type", "x", "y", "status",
	"device_sn", "site_id", "domain_id", "created_at", "updated_at",
}

var topoEdgeColumns = []string{
	"id", "source_id", "target_id", "label", "status", "created_at",
}

// ======================================================================
// PgSiteRepository
// ======================================================================

var _ SiteRepository = (*PgSiteRepository)(nil)

// PgSiteRepository is a PostgreSQL implementation of SiteRepository.
type PgSiteRepository struct {
	pool *pgxpool.Pool
}

// NewPgSiteRepository creates a new PgSiteRepository.
func NewPgSiteRepository(pool *pgxpool.Pool) *PgSiteRepository {
	return &PgSiteRepository{pool: pool}
}

func (r *PgSiteRepository) Create(ctx context.Context, site *Site) error {
	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	now := time.Now()
	site.CreatedAt = now
	site.UpdatedAt = now
	if site.Status == "" {
		site.Status = SiteActive
	}

	query, args, err := psql.Insert("sites").
		Columns("id", "name", "domain_id", "address", "longitude", "latitude",
			"device_count", "status", "created_at", "updated_at").
		Values(
			site.ID, site.Name, nullableUUID(site.DomainID),
			nullableString(site.Address),
			site.Longitude, site.Latitude,
			site.DeviceCount, site.Status, site.CreatedAt, site.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert site SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert site: %w", err)
	}
	return nil
}

func (r *PgSiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*Site, error) {
	query, args, err := psql.Select(siteColumns...).
		From("sites").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select site SQL: %w", err)
	}

	site, err := scanSite(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get site: %w", err)
	}
	return site, nil
}

func (r *PgSiteRepository) List(ctx context.Context, filter SiteFilter) (*model.ListResponse[Site], error) {
	base := psql.Select(siteColumns...).From("sites")
	countBase := psql.Select("COUNT(*)").From("sites")

	if filter.DomainID != nil {
		base = base.Where(sq.Eq{"domain_id": *filter.DomainID})
		countBase = countBase.Where(sq.Eq{"domain_id": *filter.DomainID})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": string(*filter.Status)})
		countBase = countBase.Where(sq.Eq{"status": string(*filter.Status)})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count site SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count sites: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list site SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()

	var items []Site
	for rows.Next() {
		site, err := scanSiteRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan site row: %w", err)
		}
		items = append(items, *site)
	}

	if items == nil {
		items = []Site{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgSiteRepository) ListWithCoordinates(ctx context.Context) ([]Site, error) {
	query, args, err := psql.Select(siteColumns...).
		From("sites").
		Where("longitude IS NOT NULL AND latitude IS NOT NULL").
		OrderBy("name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list sites with coords SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sites with coordinates: %w", err)
	}
	defer rows.Close()

	var items []Site
	for rows.Next() {
		site, err := scanSiteRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan site row: %w", err)
		}
		items = append(items, *site)
	}

	if items == nil {
		items = []Site{}
	}
	return items, rows.Err()
}

// ---- Site scanning helpers ----

func scanSite(row pgx.Row) (*Site, error) {
	var s Site
	var (
		domainID  sql.NullString
		address   sql.NullString
		longitude sql.NullFloat64
		latitude  sql.NullFloat64
	)

	err := row.Scan(
		&s.ID, &s.Name, &domainID, &address, &longitude, &latitude,
		&s.DeviceCount, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan site row: %w", err)
	}

	if domainID.Valid {
		id, _ := uuid.Parse(domainID.String)
		s.DomainID = &id
	}
	if address.Valid {
		s.Address = address.String
	}
	if longitude.Valid {
		s.Longitude = &longitude.Float64
	}
	if latitude.Valid {
		s.Latitude = &latitude.Float64
	}

	return &s, nil
}

func scanSiteRow(rows pgx.Rows) (*Site, error) {
	var s Site
	var (
		domainID  sql.NullString
		address   sql.NullString
		longitude sql.NullFloat64
		latitude  sql.NullFloat64
	)

	err := rows.Scan(
		&s.ID, &s.Name, &domainID, &address, &longitude, &latitude,
		&s.DeviceCount, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan site row: %w", err)
	}

	if domainID.Valid {
		id, _ := uuid.Parse(domainID.String)
		s.DomainID = &id
	}
	if address.Valid {
		s.Address = address.String
	}
	if longitude.Valid {
		s.Longitude = &longitude.Float64
	}
	if latitude.Valid {
		s.Latitude = &latitude.Float64
	}

	return &s, nil
}

// ======================================================================
// PgTopoNodeRepository
// ======================================================================

var _ TopoNodeRepository = (*PgTopoNodeRepository)(nil)

// PgTopoNodeRepository is a PostgreSQL implementation of TopoNodeRepository.
type PgTopoNodeRepository struct {
	pool *pgxpool.Pool
}

// NewPgTopoNodeRepository creates a new PgTopoNodeRepository.
func NewPgTopoNodeRepository(pool *pgxpool.Pool) *PgTopoNodeRepository {
	return &PgTopoNodeRepository{pool: pool}
}

func (r *PgTopoNodeRepository) List(ctx context.Context, filter TopoNodeFilter) (*model.ListResponse[TopoNode], error) {
	base := psql.Select(topoNodeColumns...).From("topo_nodes")
	countBase := psql.Select("COUNT(*)").From("topo_nodes")

	if filter.DomainID != nil {
		base = base.Where(sq.Eq{"domain_id": *filter.DomainID})
		countBase = countBase.Where(sq.Eq{"domain_id": *filter.DomainID})
	}
	if filter.NodeType != nil {
		base = base.Where(sq.Eq{"node_type": *filter.NodeType})
		countBase = countBase.Where(sq.Eq{"node_type": *filter.NodeType})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count topo_node SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count topo_nodes: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list topo_node SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list topo_nodes: %w", err)
	}
	defer rows.Close()

	var items []TopoNode
	for rows.Next() {
		node, err := scanTopoNodeRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan topo_node row: %w", err)
		}
		items = append(items, *node)
	}

	if items == nil {
		items = []TopoNode{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgTopoNodeRepository) ListAll(ctx context.Context, domainID *uuid.UUID) ([]TopoNode, error) {
	base := psql.Select(topoNodeColumns...).From("topo_nodes")
	if domainID != nil {
		base = base.Where(sq.Eq{"domain_id": *domainID})
	}
	base = base.OrderBy("label ASC")

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list all topo_nodes SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all topo_nodes: %w", err)
	}
	defer rows.Close()

	var items []TopoNode
	for rows.Next() {
		node, err := scanTopoNodeRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan topo_node row: %w", err)
		}
		items = append(items, *node)
	}

	if items == nil {
		items = []TopoNode{}
	}
	return items, rows.Err()
}

// ---- TopoNode scanning helpers ----

func scanTopoNodeRow(rows pgx.Rows) (*TopoNode, error) {
	var n TopoNode
	var (
		deviceSN sql.NullString
		siteID   sql.NullString
		domainID sql.NullString
	)

	err := rows.Scan(
		&n.ID, &n.Label, &n.NodeType, &n.X, &n.Y, &n.Status,
		&deviceSN, &siteID, &domainID, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan topo_node row: %w", err)
	}

	if deviceSN.Valid {
		n.DeviceSN = deviceSN.String
	}
	if siteID.Valid {
		id, _ := uuid.Parse(siteID.String)
		n.SiteID = &id
	}
	if domainID.Valid {
		id, _ := uuid.Parse(domainID.String)
		n.DomainID = &id
	}

	return &n, nil
}

// ======================================================================
// PgTopoEdgeRepository
// ======================================================================

var _ TopoEdgeRepository = (*PgTopoEdgeRepository)(nil)

// PgTopoEdgeRepository is a PostgreSQL implementation of TopoEdgeRepository.
type PgTopoEdgeRepository struct {
	pool *pgxpool.Pool
}

// NewPgTopoEdgeRepository creates a new PgTopoEdgeRepository.
func NewPgTopoEdgeRepository(pool *pgxpool.Pool) *PgTopoEdgeRepository {
	return &PgTopoEdgeRepository{pool: pool}
}

func (r *PgTopoEdgeRepository) List(ctx context.Context, filter TopoEdgeFilter) (*model.ListResponse[TopoEdge], error) {
	base := psql.Select(topoEdgeColumns...).From("topo_edges")
	countBase := psql.Select("COUNT(*)").From("topo_edges")

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count topo_edge SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count topo_edges: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list topo_edge SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list topo_edges: %w", err)
	}
	defer rows.Close()

	var items []TopoEdge
	for rows.Next() {
		edge, err := scanTopoEdgeRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan topo_edge row: %w", err)
		}
		items = append(items, *edge)
	}

	if items == nil {
		items = []TopoEdge{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgTopoEdgeRepository) ListAll(ctx context.Context) ([]TopoEdge, error) {
	query, args, err := psql.Select(topoEdgeColumns...).
		From("topo_edges").
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list all topo_edges SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all topo_edges: %w", err)
	}
	defer rows.Close()

	var items []TopoEdge
	for rows.Next() {
		edge, err := scanTopoEdgeRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan topo_edge row: %w", err)
		}
		items = append(items, *edge)
	}

	if items == nil {
		items = []TopoEdge{}
	}
	return items, rows.Err()
}

// ---- TopoEdge scanning helpers ----

func scanTopoEdgeRow(rows pgx.Rows) (*TopoEdge, error) {
	var e TopoEdge
	var label sql.NullString

	err := rows.Scan(
		&e.ID, &e.SourceID, &e.TargetID, &label, &e.Status, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan topo_edge row: %w", err)
	}

	if label.Valid {
		e.Label = label.String
	}

	return &e, nil
}
