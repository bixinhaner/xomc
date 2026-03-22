# Database Migrations

Migration files follow the format `000NNN_description.{up,down}.sql`.

## Numbering Note

Migrations 000036-000045 were skipped during development consolidation.
This gap does not affect migration tooling (golang-migrate processes
files in lexicographic order regardless of gaps).

Existing sequence: 000001-000035, 000046-000052
