# Consolidate OMC migrations into three baselines

## Goal

Consolidate every Goose SQL migration under `omcgo/migrations` into the
`000001` file of its independent migration stream. The resulting tree keeps
exactly these SQL files:

- `omcgo/migrations/000001_init_schema.sql`
- `omcgo/migrations/seed/000001_init_seed.sql`
- `omcgo/migrations/tsdb/000001_tsdb_schema.sql`

The baselines target clean installations. Existing databases that recorded
later Goose versions must be rebuilt or have their migration state handled
manually, consistent with the current repository policy.

## Approach

Use a hybrid final-state rebuild rather than concatenating historical Up
sections.

### Main database schema

1. Start an isolated clean PostgreSQL 16 instance.
2. Apply `omcgo/migrations/000001` through `000024` in order.
3. Export the final public schema with `pg_dump --schema-only`, excluding all
   Goose version tables and deployment-specific ownership or privilege data.
4. Remove unsupported `psql` meta-commands, restore `search_path`, wrap the
   result in one Goose Up section, and retain a complete baseline Down section.

This produces direct final definitions for tables, columns, constraints,
indexes, functions, and triggers instead of retaining historical ALTER chains.

### Main database seed

1. On the same clean main database, apply seed migrations `000001` through
   `000013` using the independent seed Goose version table.
2. Export the final data with `pg_dump --data-only --inserts
   --on-conflict-do-nothing`, excluding Goose version tables.
3. Normalize the dump for Goose and wrap it as the single seed baseline.
4. Keep seed Down as an explicit no-op because the consolidated seed represents
   an installation snapshot and cannot be safely reversed row by row.

No application service may start against the temporary database before the
dump, preventing runtime data from entering the seed baseline.

### TimescaleDB schema

1. Apply `tsdb/000001` through `000005` to an isolated clean TimescaleDB
   instance and verify the final objects.
2. Integrate the incremental changes into `000001_tsdb_schema.sql` as direct
   final definitions: include the hourly continuous aggregate and policies,
   and define the affected metric columns with their final nullability.
3. Update the baseline Down section so every object introduced by Up is
   removed in dependency-safe order.

The TSDB baseline is edited explicitly instead of using a whole-database dump,
which could leak TimescaleDB extension catalog objects into repository SQL.

## File cleanup and documentation

After the three baselines are complete, delete every numbered SQL file above
`000001` in the schema, seed, and TSDB streams. Update
`omcgo/migrations/README.md` so its file counts, consolidation date, current
state, and next-version guidance match the new baseline.

Scripts outside `omcgo/migrations` and non-SQL documentation are out of scope.

## Verification

Verification uses isolated disposable database instances and does not delete or
modify the developer's existing database volumes.

1. Confirm exactly three SQL files remain under `omcgo/migrations`, one per
   independent stream, and each is version `000001`.
2. Check Goose Up, Down, StatementBegin, and StatementEnd directives for valid
   structure.
3. From empty databases, run schema, seed, and TSDB baselines with their
   independent Goose version tables; all must exit successfully.
4. Compare the consolidated clean database with the pre-consolidation golden
   final state: main-schema objects and per-table seed row counts must match;
   TSDB user objects, continuous aggregate policies, and final column
   nullability must match.
5. Run `cd omcgo && go build ./... && go test ./...` because migration loading
   and migration-focused integration code are part of the backend.

If Docker or local-listener tests are blocked by sandbox permissions, rerun the
required verification with the repository-prescribed escalation and report any
remaining environmental failure explicitly.
