# Rater → Helix Migrator (iteration 1 demo)

Upload a rater workbook. The tool reads its **Policy Data** sheet and applies the deterministic mapping rules taken from the
*Confirmed* section of a schema-validation report. You review every impacted Helix table and column, override what you
need, approve, and the records are written through the Helix entity API.

Iteration 1 ships one rule set: **Manatee FL Select HO Rater Effective 12.1.25** (20 confirmed columns).
The design, the decisions (D1–D12) and the schema backlog (S1–S15) are in `../pythonProjects/PROJECT_PLAN.md`.

## Run

```bash
make run
```

This builds the Svelte UI and the Go server, then serves http://127.0.0.1:8080 (local only).
The Helix URL and writer key come from `HARNESS_URL` / `HARNESS_WRITER_KEY` in the project's `.env`
(git-ignored; copy `.env.example` to start). Override the file with `ENV_FILE=… make run` or `-env-file`.
`HELIX_TOKEN` or `mycel token` also work, with the same precedence as the Python client.

```bash
make test        # Go unit tests (the real HO workbook is used when it is present locally)
make check       # tests + Svelte type check
make cleanup     # list records the migrator created; then: go run ./cmd/migrator cleanup -yes …
make rules       # regenerate the rule set from the report + ddl.sql (review the diff)
```

For UI development, run `cd web && npm run dev` beside a running server; Vite proxies `/api` to it.

## Flow

1. **Upload** an `.xlsx`/`.xlsm` file. The `Policy Data` sheet is read by column letter. Reading stops at the first row with no policy number.
2. **Review**:
   - Impacted tables, grouped by Helix variant. Each field is labelled Sheet, Template, Generated, Reference or Overridden, with before → after examples.
   - Issues, and a row-by-row preview of the records.
   - Overrides: retarget a column to any Helix field, exclude it, edit its value map, or set a template or placeholder value. Every change re-plans; the plan is deterministic and hashed.
3. **Approve**:
   - Acknowledge every attention item: questionable matches, generated placeholders, and your overrides.
   - Choose a row limit and dry run. Approval is tied to the plan hash.
4. **Write**:
   - The issuer and product records are found or created once per upload.
   - Each row is then written as one unit: found by business key, or through the ledger for tables without one, else created.
   - When a record already exists, only its **sheet-sourced** fields are updated.
   - If any write in a row fails, that row's writes are undone (deletes verified by 404).
   - Download a CSV of the results.

## Layout

| Path | What |
|---|---|
| `cmd/rulegen` | Report HTML + `ddl.sql` registry + reviewed pins → committed rule set |
| `internal/rules/rulesets` | `*.pins.json` (reviewed targets), `*.json` (generated rules), `*.templates.json` (required-field sources) |
| `internal/excel` | Policy Data reader (excelize, raw values) |
| `internal/transform` | Normalise, value maps, coercion to Helix types, deterministic placeholders |
| `internal/plan` | Deterministic planner: impact, issues, attention, plan hash |
| `internal/execute` | Find-or-create, update, per-row rollback, cleanup |
| `internal/ledger` | Append-only JSONL of created records (`data/ledger.jsonl`) |
| `internal/helix` | Go port of the Python Helix client (+ `where=` key lookup) |
| `internal/api` | JSON API; in-memory jobs (single user) |
| `web/` | Svelte 5 + Vite UI, embedded into the binary |

## Things to know

- **Writes go to the shared remote Helix.** Dry run is on by default. A real write must be confirmed by typing `WRITE`. `migrator cleanup -yes` removes everything recorded in the ledger.
- **Generated placeholders** (`GEN-…`) fill required fields the sheet does not provide (D6). They are deterministic, so re-runs update the same records rather than creating new ones.
- **Jobs are held in memory.** Restarting the server forgets uploaded jobs. The ledger persists.
