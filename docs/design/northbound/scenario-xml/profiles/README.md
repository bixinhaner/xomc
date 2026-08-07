# 北向 OutputProfile catalog

`OutputProfile` was the contract between compact scenario XML and xomc data
adapters in the historical XML/Profile design. The current recommended design
is page-based configuration in `../../page-config-redesign-20260731.md`.

Scenario XML answers these questions:

- which file should be generated
- when it should be generated
- what remote path and file name template should be used
- which endpoint group should receive the file

OutputProfile answers these questions:

- which columns or XML elements are emitted
- what the column order is
- how legacy alias names map to xomc sources
- which adapter reads the rows
- how timestamps, empty values, CSV quoting, XML root names, and PM metric values
  are formatted

The compact scenario files use `profileSet="baicells-legacy-v1"`. If an
`Object` does not explicitly set `profile`, the importer resolves it with:

```text
<domain>.<object>[.<tech>].<format>.v1
```

Examples:

```text
CM + CP + XML      -> cm.cp.xml.v1
CM + CP + GNB XML  -> cm.cp.gnb.xml.v1
PM + PC + CSV      -> pm.pc.csv.v1
INVENTORY + STATION + CSV -> inventory.station.csv.v1
INVENTORY + STATION + GNB CSV -> inventory.station.gnb.csv.v1
INVENTORY + OMC + CSV -> inventory.omc.csv.v1
```

The catalog in `baicells-legacy-v1.xml` must cover every profile key referenced
or implied by compact XML files and built-in northbound file tasks. The
`ColumnSet` references are intentional: large legacy alias lists should live in
column-set assets or code-generated fixtures, not in scenario XML.

Inventory profiles are catalog entries for `enbMonitorExport.md`, not extra
`local-s000x.xml` scene files. In the current page-based design they become
built-in template seeds for Station/OMC Inventory, not runtime XML references.

Special Inventory renderer rules:

- `inventory.station*.csv.v1` uses GBK, quote-all CSV, and `PreserveAsText`
  columns for ECI/PCI/Earfcn/PLMN/time-like fields.
- `inventory.omc.csv.v1` emits one UTF-8 aggregate row.
Implementation placement:

- Design draft: `docs/design/northbound/scenario-xml/profiles/`
- Runtime candidate: `omcgo/data/northbound/profiles/`
- Go registry package: `omcgo/internal/northbound/profile/`

Runtime must reject a scenario when either its profile key or adapter key is
missing.
