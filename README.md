# Rater → Helix Migrator (iteration 1 demo)

Upload a rater workbook. The tool reads its **Policy Data** sheet and applies the deterministic mapping rules taken from the
*Confirmed* section of a schema-validation report. You review every impacted Helix table and column, override what you
need, approve, and the records are written through the Helix entity API.

Iteration 1 ships one rule set: **Manatee FL Select HO Rater Effective 12.1.25** (20 confirmed columns).
The design, the decisions (D1–D12) and the schema backlog (S1–S15) are in [PROJECT_PLAN.md](PROJECT_PLAN.md).

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

## Browse data

The **Browse data** page (header) reads Helix back:

- **By policy number** lists every record linked to the policy, grouped by table. Each record shows how it was found:
  - 🔑 matched on `policy_number`;
  - 📒 the migrator's ledger (needed for tables with no reference to the policy, D2);
  - → referenced by a policy-owned record (party, product, issuer…);
  - ← references a policy-owned record (terms, versions, coverages…).
  
  Reverse references are followed up to 4 levels. Shared records such as the product are shown but not expanded, so other policies don't leak in. Reference values link to the record they point at.
- **By table** lists any of the ~1,000 entities, optionally filtered by one column = value. Strings are quoted and numbers and booleans are not, as the Helix `where=` filter expects. Pages use the API cursor, and clicking a row shows every field.

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
| `internal/browse` | Policy traversal and table listing for the Browse page |
| `internal/api` | JSON API; in-memory jobs (single user) |
| `web/` | Svelte 5 + Vite UI, embedded into the binary |

## Things to know

- **Writes go to the shared remote Helix.** Dry run is on by default. A real write must be confirmed by typing `WRITE`. `migrator cleanup -yes` removes everything recorded in the ledger.
- **Generated placeholders** (`GEN-…`) fill required fields the sheet does not provide (D6). They are deterministic, so re-runs update the same records rather than creating new ones.
- **Jobs are held in memory.** Restarting the server forgets uploaded jobs. The ledger persists.

## Working PersonalHome quote application

Open **Quotes** in the navigation rail, or `/#personalhome/newquote`.

Create a quote → Applicant → Risk schedule → Dwelling coverage → Underwriting → Insurance history → Claims history → Coverage summary → Additional interests → optional Billing instructions → Review.

The application assigns a quote number, saves drafts on the server, validates each page before continuing, and resumes saved quotes after refresh/restart. Add and remove co-applicants, prior policies, losses and interests. Earlier edits invalidate downstream completion. Final submission records **Awaiting rating** locally; download the saved application as JSON.

```bash
make quote-run     # builds UI/server; starts locally without Helix credentials
```

For development, `go run ./cmd/migrator serve -offline -web-dir web/dist` starts the API on port 8080; `cd web && npm run dev` serves the UI with API proxying. Remove `-offline` to enable the existing migration/browse APIs with configured Helix access. Quotes persist under `data/quotes` (or `-quote-dir PATH`) with optimistic version checks and atomic file saves. Back up that directory to retain your applications. This remains a single-user, single-server local application.

**Carrier integrations remain pending:** premiums, complete carrier underwriting/lookup rules, CLUE/NCF, e-signature, payment processing and binding. Submission does not send data to a carrier or write candidate UI mappings to Helix. The app does not fabricate a premium or issue coverage.

The source inventory and mapping reference files remain in the repository. The Flow reference tab has been removed; old `#reference` links open the quote application. The [implementation plan](docs/PERSONALHOME_IMPLEMENTATION_PLAN.md) records source discrepancies and the implemented application scope.

The [page-by-page field audit](docs/PERSONALHOME_FIELD_AUDIT.md) accounts for all 170 detailed HTML inventory rows plus narrative-only additions. The shared schema includes 192 scalar definitions and 22 repeated-entry definitions. Calculated/internal fields remain protected; missing carrier outputs and actions are displayed as unavailable. Policy Information is incorporated into New quote, Location Detail into Risk schedule, and Pricing into Coverage summary. Billing preferences and the unavailable installment outputs share Billing instructions.

Regenerate source-derived definitions, the field audit, and run frontend tests:

```bash
python3 scripts/extract_personalhome.py
python3 scripts/build_quote_schema.py
cd web && npm test  # Node 22.18+ for native TypeScript stripping
```

### Demo data

With the local quote server running, seed three fictional applications:

```bash
python3 scripts/seed_demo_quotes.py
```

- **Avery Demo:** HO3, $400,000 dwelling limit, co-applicant, pool, prior insurance and mortgagee escrow; awaiting rating.
- **Morgan Demo:** HO6, $75,000 contents limit, prior insurance, a $2,250 fictional water-damage loss and condominium association; awaiting rating.
- **Riley Demo:** HO3, saved draft at underwriting with a renovation declaration and notes.

Names ending in Demo, `example.com` emails, 555-01xx phone numbers and fictional addresses distinguish the samples. The script uses the local API and its validations. Its seed index is stored in `data/demo-seed-index.json`; rerunning preserves existing demo quotes and user edits. No premiums, payments or bound policies are fabricated.
