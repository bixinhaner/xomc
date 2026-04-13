package storage

import (
	"github.com/doug-martin/goqu/v9"
)

// GoquDialect is the pre-configured PostgreSQL dialect for goqu queries.
// All dynamic query builders should use this dialect to ensure
// dollar-sign placeholder format ($1, $2, ...) is used.
//
// Usage:
//
//	ds := storage.GoquDialect.From("devices").Where(goqu.C("status").Eq("active"))
//	sql, args, _ := ds.ToSQL()
var GoquDialect = goqu.Dialect("postgres")
