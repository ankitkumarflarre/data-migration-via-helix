# Rater Policy Data → Helix Migrator: Project Plan (Iteration 1)

Status: **plan only, no implementation yet**. Date: 2026-10-07. Revision 3: decisions D1–D12 recorded (§11); demo cut for end of day in §14.

## 1. Goal

A Go service with a Svelte UI that:

1. Accepts an uploaded rater workbook (`.xlsx` / `.xlsm`) and reads the **`Policy Data`** sheet.
2. Applies **deterministic mapping rules** taken from the *Confirmed* section of a schema-validation report.
3. Shows the user every **impacted Helix table (variant) and column (field)**, with the transformed values and any problems found.
4. Lets the user **override** a mapping (change the target table or column, exclude a column, edit a value map), then **approve**.
5. After approval, writes the records through the **Helix records API** (`/api/entities/records/{variant}`).

**Iteration 1 scope:** one rule set only, taken from
`MappingSQLData/Manatee FL Select HO Rater Effective 12.1.25 - Schema Validation Report.html`
(20 confirmed columns out of 81 in the report; the sheet has 1,394 data rows).

Out of scope for iteration 1: the other four raters, the "Review" and "Not Found" columns, premium and rating output columns, user authentication and roles, and multiple tenants.

---

## 2. The 20 confirmed rules (iteration 1)

The report was generated against an **older DDL**, so its schema names are out of date. Each name maps to a current schema as follows:

| Report schema | Current schema in `ddl.sql` | Coordinate |
|---|---|---|
| `core_all` | `lall_jall_sall_call` | all / all / all / all |
| `lob_property_personal` | `property_jall_personal_call` | property / all / personal / all |
| `jur_us_property_personal` | `property_us_personal_call` | property / us / personal / all |

The Helix API addresses **leaf variants**, not tables. Each rule therefore resolves to a `variant.field` (checked against `/describe`). The report lists up to 156 "locations" for a single column. The rule records one **primary target** and keeps the rest as **alternatives**, which become the options offered when the user overrides.

| # | Col | Header | Primary target (`variant` → `field`) | Helix type | Transform | Notes / alternatives |
|---|---|---|---|---|---|---|
| 1 | A | Policy Number | `policy.property.us-fl.personal.safepoint` → `policy_number` | string(255) | trim | **The business key for the whole row (D5).** Unique per carrier. Alternatives: `policy_lifecycle`, `raw_xml_document` |
| 3 | C | County | `location_address.property.us.personal` → `county` | string(255) | trim, upper-case | Alternative: `policy_address.county`. Source has mixed case (`MIAMI-DADE` / `Miami-dade`) |
| 5 | E | Territory | `loss_ratio_analysis` → `territory` | string(255) | int → string | ⚠ **Attention flag (D4):** the names match, but this is a portfolio reporting table, not a rating territory. Kept as-is and shown to the user at approval, who can override it |
| 6 | F | Effective Date | `policy.property.us-fl.personal.safepoint` → `effective_date` | date | datetime → `YYYY-MM-DD` | Over 70 alternative tables; the policy is the obvious one |
| 10 | J | Year Built | `dwelling.property.us.personal` → `year_built` | integer | int | Alternative: `dwelling_incident` |
| 11 | K | Construction | `dwelling.property.us.personal` → `construction` | string(255) | trim | |
| 12 | L | Protection Class | `dwelling_asset.property.personal` → `protection_class` | string(255) | int → string | |
| 13 | M | Number of Stories | `dwelling.property.us.personal` → `number_of_stories` | integer | int | Alternative: `geo_code_address_result` |
| 14 | N | Number of Units | `dwelling.property.us.personal` → `number_of_units` | **integer** | value map `"1 to 4"→4`, `"5+"→5` (D1) | Any other value is a blocking issue. Bare integers are accepted as they are |
| 16 | P | Burglar Alarm | `dwelling.property.us.personal` → `burglar_alarm` | string(255) | trim | Yes/No |
| 19 | S | Building Code Effectiveness Grading | `dwelling.property.us.personal` → `building_code_effectiveness_grading` | string(255) | any → string | `Ungraded` or 2–5 |
| 22 | V | Roof Deck Attachment | `dwelling.property.us.personal` → `roof_deck_attachment` | string(255) | trim | |
| 23 | W | Roof Shape | `wind_mitigation_verification.property.us-fl.personal.safepoint` → `roof_shape` | enum(hip,gable,flat,other) | value map `Gable→gable`, `Hip Roof→hip`, `Flat→flat` | ⚠ **Attention flag (D4):** kept. Writing it requires a `location`, a `dwelling_asset` and inspection fields taken from templates (§6.2), all shown at approval |
| 25 | Y | Secondary Water Resistance | `dwelling.property.us.personal` → `secondary_water_resistance` | boolean | `SWR→true`, `No SWR→false` | |
| 26 | Z | Opening Protection | `dwelling.property.us.personal` → `opening_protection` | string(255) | trim | Alternative: the wind-mitigation enum (`none`/`basic`/`hurricane_rated`/`unknown`), which would need a value map |
| 27 | AA | Wind Speed Design | `dwelling.property.us.personal` → `wind_speed_design` | string(255) | trim | |
| 28 | AB | Wind Speed Location | `dwelling.property.us.personal` → `wind_speed_location` | string(255) | any → string | Mix of numbers and text |
| 45 | AS | Equipment Breakdown | `section_icoverages.property.us.personal` → `equipment_breakdown` | text | trim | |
| 50 | AX | Consent to Rate | `line.property.us.personal` → `consent_to_rate` | boolean | `0→false`, `1→true` | |
| 73 | BU | Policy Number | `policy.property.us-fl.personal.safepoint` → `policy_number` | string(255) | trim | Same target as column A. **Every value is empty in the HO file.** Rule: use it only if non-empty; if both A and BU have values and they differ, raise a blocking conflict |

**Impacted variants (8):** `policy` (FL SafePoint), `dwelling`, `dwelling_asset`, `wind_mitigation_verification`, `location_address`, `section_icoverages`, `line`, `loss_ratio_analysis`.

---

## 3. Architecture

```
┌──────────────── Browser (Svelte SPA) ────────────────┐
│ Upload → Review & Override → Approve → Progress/Report│
└───────────────▲──────────────────────┬───────────────┘
                │ JSON / SSE           │ multipart upload
┌───────────────┴──────────────────────▼───────────────┐
│ Go server (single binary, SPA embedded via go:embed) │
│  api ─ jobs ─ excel ─ rules ─ transform ─ plan       │
│                          │                           │
│                     execute ── helix client ─────────┼──► Helix REST
│  SQLite: jobs, overrides, plan hash, write ledger    │    (core.vera.cogniworks.io)
└──────────────────────────────────────────────────────┘          │
                                                            Postgres vera_core
```

- **Single deployable binary.** The Go server serves the built Svelte app and the JSON API.
- **Helix credentials stay on the server only**, using the same order of precedence as the Python client: `HELIX_TOKEN`, then `.env` `HARNESS_*_KEY` (only for the host it was issued for), then `mycel token`. They are never sent to the browser.
- **SQLite** stores jobs, overrides, the approved plan and a **write ledger** (every created record ID). This gives auditing, safe re-runs, and links between records that the model can't express itself (see §6).

### Repository layout (proposed, new folder `migrator/`)

```
migrator/
  go.mod
  cmd/
    server/main.go            # HTTP server, config, embed web/build
    rulegen/main.go           # report HTML → rules JSON (developer tool, run once per report)
  internal/
    rules/                    # rule types, embedded rule sets, validation against /describe
      rulesets/manatee_fl_select_ho_12_1_25.json   # generated + reviewed + committed
    excel/                    # open workbook, read "Policy Data", address columns by letter
    transform/                # type coercion + value maps (pure functions)
    plan/                     # build impact summary + per-row records, apply overrides, plan hash
    helix/                    # REST client: auth, CRUD, describe/catalogue cache
    execute/                  # dependency-ordered writes, idempotency, compensation, progress
    jobs/                     # job state machine + SQLite store
    api/                      # HTTP handlers, SSE progress
  web/                        # SvelteKit (adapter-static) + TypeScript
  testdata/                   # small sanitised workbook fixtures (no confidential data)
```

### Key libraries

| Concern | Choice | Why |
|---|---|---|
| Excel | `github.com/xuri/excelize/v2` | Reads `.xlsm` (macros ignored) and the cached values of formula cells |
| HTML (rulegen only) | `golang.org/x/net/html` | Parses the report tables |
| HTTP | standard library `net/http` (Go 1.22+ routing) | No framework needed |
| Storage | `modernc.org/sqlite` (no cgo) | Keeps the build to one portable binary |
| UI | SvelteKit + `adapter-static`, TypeScript, Vite | Builds a static SPA that is embedded in Go |
| UI tests | Vitest + Playwright | Component tests and end-to-end tests |

---

## 4. Deterministic mapping rules

### 4.1 Rule set format

`rulegen` parses the report HTML once. Its output is **reviewed by a person** and **committed**, and the server never reads HTML at runtime. The output records the report file's SHA-256 and the Helix bundle version it was checked against.

```json
{
  "rule_set": "manatee_fl_select_ho_12_1_25",
  "source_report": { "file": "Manatee FL Select HO Rater Effective 12.1.25 - Schema Validation Report.html", "sha256": "…" },
  "sheet": "Policy Data",
  "helix_bundle": "2026.10.02-1",
  "rules": [
    {
      "id": "HO-06",
      "excel_column": "F",
      "header": "Effective Date",
      "report_status": "Confirmed - Exact/Normalized",
      "target": { "variant": "policy.property.us-fl.personal.safepoint", "field": "effective_date" },
      "alternatives": [ { "variant": "policy_lifecycle", "field": "…" } ],
      "report_locations": ["jur_us_property_personal.policy.effective_date [TABLE]", "…"],
      "transform": { "kind": "date", "format": "2006-01-02" },
      "on_empty": "skip_field",
      "severity_if_invalid": "blocking"
    }
  ]
}
```

### 4.2 How `rulegen` works

1. Parse table 1 (Column-Level Results). Keep rows whose status badge is `Confirmed - Exact/Normalized`.
2. Parse table 2 (Confirmed Mapping Locations) to get every location for each column.
3. Translate the old schema names to the current ones (§2 table). Look up which leaf variant owns each table and column by joining `_engine.variant_lineage` and `variant_field` (from `/model/variants`, `/model/variants/{v}/lineage` and `/describe`).
4. Choose the **primary** target with a fixed ordering: (1) a variant whose coordinate matches the rule set's line of business (property / us-fl / personal / safepoint), then (2) the most specific variant that owns the field, then (3) alphabetical order to break ties. A person can override the choice in a `primary_overrides` section, which is how the choices in §2 are pinned.
5. Derive the transform from the Helix field type (`integer`, `boolean`, `date`, `enum(...)`, `string(n)`, `text`). Value maps for enums and booleans are written by hand in the committed file.
6. A **golden test** asserts exactly 20 rules, with the column letters and targets listed in §2.

### 4.3 Determinism guarantees

- Rules are embedded with `go:embed`, so they can't change at runtime.
- `plan(workbook bytes, rule set, overrides) → plan` is a **pure function**. Rows are sorted by sheet row number and fields by the rule's column order. No maps are iterated in random order, and there is no wall-clock time inside the plan.
- **Plan hash** = SHA-256 of (workbook SHA-256, rule-set version, canonical JSON of the overrides, Helix bundle version). Approval is tied to that hash. If anything changes after review, execution refuses to run.
- On startup, the server checks every rule target against the live `/describe`. If a variant or field has disappeared or changed type, or the bundle version differs, the rule set is marked **stale** and approval is disabled until someone reviews it.

---

## 5. Processing pipeline

```
upload → parse → map/transform → validate → review (overrides) → approve → execute → report
```

| Stage | What happens | Output |
|---|---|---|
| **Upload** | Accept `.xlsx`/`.xlsm` up to 25 MB. Compute SHA-256. Store in a temporary job folder with a retention limit. | `job_id` |
| **Parse** | Open the `Policy Data` sheet. Row 1 is the header. Address columns **by letter**, because the header text repeats (two "Policy Number" columns). Stop at the first row where column A is empty, which handles sheets padded out to the maximum row. Check that each rule's header matches the header in its column; a mismatch is blocking. | Raw rows (typed cells) |
| **Map/transform** | For each row and each rule: transform the value, then place it in `records[variant][field]`. | One draft record per (row, variant) |
| **Validate** | Check types, enum membership, string lengths, required fields per variant (from `/describe`), duplicate policy numbers in the file, and the A/BU conflict. Issues are `blocking` or `warning`. | Issue list with row, column and message |
| **Review** | The UI shows the impact summary and a sample of rows. The user can override targets, value maps, or exclude a rule. Each override triggers a re-plan (deterministic). | New plan and hash |
| **Approve** | Allowed only when there are no blocking issues and the plan hash is the latest. Records who approved and when. | Approved, frozen plan |
| **Execute** | Write to Helix in dependency order (§6), with idempotency keys and per-row compensation. Progress is streamed over SSE. | Write ledger |
| **Report** | Totals of rows written, skipped and failed, plus the created record IDs. Downloadable as CSV. | `report.csv` |

---

## 6. Writing to Helix

### 6.1 What we learned from the model (this changes the scope)

- **Required fields not in the sheet.** The SafePoint FL `policy` variant requires `policyholder_reference`, `product_reference`, `issuing_party_reference`, `issue_date`, `personal_policy_form`, `program_code`, `policy_reference` and `sinkhole_coverage_option`. `dwelling_asset` requires `dwelling_asset_reference`, `dwelling_type`, `is_under_renovation` and `location_reference`. `wind_mitigation_verification` requires `dwelling_asset_reference`, `inspection_date`, `oir_b1_1802_form_revision`, `verification_status` and `wind_mitigation_verification_reference`.
- **Some variants can't be linked to a policy.** `dwelling`, `section_icoverages`, `line`, `location_address` and `loss_ratio_analysis` have no reference field pointing to a policy (or to anything else). Records written there would be orphans in Helix.

### 6.2 Proposed handling

1. **Record templates (configuration in `rulesets/…templates.json`, not hard-coded).** Each variant that gets written has a template supplying the required fields the sheet doesn't have. Values come from four sources, and **every template value is shown in the UI as "system-supplied" and can be edited before approval**:
   - `const`: a fixed value;
   - `derive`: built from another value in the same row, e.g. `policy_number`;
   - `lookup`: a value map applied to a sheet column that isn't one of the 20 confirmed columns (used only to fill required fields, and labelled that way in the UI);
   - `ref`: the ID of another record in the same row or job.

   **Rule for fields the sheet doesn't have (D6):**
   - **Required fields:** filled with a *generated placeholder* value.
   - **Optional fields:** left empty, so they are never sent.

   Every generated value is tagged `generated=true` in the ledger and listed in the approval attention panel and in the final report, so it can be found and fixed next iteration (§13, S11).

   Generated values are **deterministic, not truly random**, because re-running the same file must update the same records (D12), and business keys must stay stable for find-or-create to work:

   | Field type | Generated value |
   |---|---|
   | string / text | `GEN-{field}-{first 8 hex chars of sha256(PN or rule set | field)}` |
   | enum | a fixed valid value named in the template (for example `active` or `submitted`) |
   | boolean | `false` |
   | date | the row's Effective Date |

   **Templates** (`PN` = Policy Number, column A, per D5; `GEN` = generated placeholder, ⚠ = flagged at approval):

   | Variant | Scope | Find-or-create key | Required fields filled by the template |
   |---|---|---|---|
   | `organization` (issuer) | once per job | `party_reference = GEN` ⚠ (stable per rule set) | `legal_name = GEN` ⚠, `record_status=active`, `registration_status=active` |
   | `product` | once per job | `product_code = GEN` ⚠ (stable per rule set) | `product_name = GEN` ⚠, `product_category = GEN` ⚠, `product_status=active`, `effective_date` = earliest Effective Date in the file |
   | `party` (policyholder) | per row | `party_reference = PH-{PN}` | `party_type=person`, `record_status=active`. The sheet has no insured identity |
   | `policy` (FL SafePoint) | per row | `policy_number = {PN}` | `policy_reference={PN}`, `issue_date` = Effective Date ⚠, the 3 refs, **`program_code=safepoint`** (D7), `personal_policy_form` from `Form` (`HO3→ho_3`, `HO6→ho_6`), `sinkhole_coverage_option` from `Sinkhole Coverage` (`Yes→sinkhole_loss`, `No→catastrophic_ground_cover_collapse_only`, D8) |
   | `location` | per row | `location_identifier = LOC-{PN}` | `location_type=primary_premises`, `address_text = "{County}, FL"` ⚠, `effective_date` = Effective Date, `party_reference` → policyholder |
   | `dwelling_asset` | per row | `dwelling_asset_reference = DA-{PN}` | `dwelling_type = GEN` ⚠ (D7: the model has no list of allowed dwelling types; HO3 = owner-occupied home and HO6 = condo is recorded in §13, S12), `is_under_renovation=false` ⚠, `location_reference` → location |
   | `wind_mitigation_verification` (FL) | per row | `wind_mitigation_verification_reference = WMV-{PN}` | `dwelling_asset_reference` → dwelling_asset, `verification_status=submitted`, `inspection_date` = Effective Date ⚠, `oir_b1_1802_form_revision = GEN` ⚠ (D9) |
   | `dwelling`, `section_icoverages`, `line`, `location_address`, `loss_ratio_analysis` | per row | none (no business key, D2). Found through the ledger only | none required |

2. **Find-or-create for prerequisites (D3).** For each record with a business key, the tool first checks whether a record with that key exists:
   1. the **ledger** (records this tool created before);
   2. Helix itself (§6.3 spike).
   
   If one exists, its ID is reused. If not, the record is created. At review the UI shows how many of each will be **reused** and how many **created**, per variant.
3. **Write ledger** (SQLite): `(job_id, sheet_row, policy_number, variant, record_id, version, idempotency_key, action=created|reused, status)`. For the five variants without a reference to the policy, the ledger is the **only** link back to the policy. This is accepted for iteration 1 (D2) and listed in §13.
4. **Dependency order per row:** `organization`, `product` (once per job) → `party` → `policy` → `location` → `dwelling_asset` → `wind_mitigation_verification` → `dwelling`, `section_icoverages`, `line`, `location_address`, `loss_ratio_analysis`. The order is checked against `/export/list` (load order), not just hard-coded.
5. **Idempotency:** every create sends `Idempotency-Key = sha256(plan_hash | sheet_row | variant)`. Retrying a failed or interrupted job therefore doesn't create duplicates. The ledger is checked before each write.
6. **Failure handling:** each row is treated as one unit. If any create in a row fails, the records **created** (never *reused*) for that row are deleted in reverse order and each deletion is checked with a GET that must return 404 (the same pattern as `run.py`). The row is marked failed and the job continues with the next row. Unexpected HTTP errors (401/403/5xx) pause the job so the user can choose to resume or abort.
7. **Concurrency:** a small worker pool (default 4, configurable; the Helix database connection pool is 4) with rate limiting. About 1,394 rows × 11 per-row variants ≈ 15k creates, plus 2 records per job.
8. **Re-runs update what comes from the sheet (D12).** When a record with the same key already exists (found in the ledger or Helix):
   - It is **updated with only the sheet-sourced fields**, meaning the confirmed rules that target that variant. Template and generated values are never overwritten.
   - The update is a `PATCH` against the record's current `version`. The tool reads the record first, skips the write if nothing changed, and on a version conflict re-reads and retries once.
   - For variants with no business key (`dwelling`, `section_icoverages`, `line`, `location_address`, `loss_ratio_analysis`), the record is found through the ledger. If the ledger has no entry, a new record is created.
   - The review and approval screens show **create / update / unchanged** counts per variant.
   - Before each `PATCH`, the record's prior values are saved in the ledger. If a later write in the same row fails, those values are restored on a best-effort basis.
9. **Go Helix client:** a port of `storage_engine_client.py`. It covers base-URL validation (no credentials or query string, no loopback for writes), role-based tokens, refusing to follow a redirect to a different host, wrapping `{data}` / `{error}` responses, and `create/get/patch/delete`, plus read-only `describe`, `catalogue`, `variants` and `export/list` with caching.

### 6.3 Spike: looking up a record by business key (start of M4)

The only list API known so far is `GET /api/entities/list/{entity}?limit=&after=`, which pages through everything with no filter. Before building find-or-create, the spike checks for (a) a filter or business-key lookup parameter on `/list` or `/records`, and (b) whether a create with a duplicate key returns a conflict error that includes the existing ID.

- **If a lookup exists:** use it.
- **If not:** the fallback is (1) the ledger, then (2) a paged scan of the entity, cached per job. That is acceptable while the database is nearly empty, but it is a scaling risk recorded in §13.


---

## 7. Backend HTTP API

| Method & path | Purpose |
|---|---|
| `GET /api/health` | Server status, Helix reachability, rule-set freshness |
| `GET /api/rulesets` | Available rule sets (iteration 1: one) with stale/fresh status |
| `POST /api/jobs` | Multipart upload plus `ruleset` → creates a job; parses and plans it |
| `GET /api/jobs/{id}` | Job status, counts, plan hash, issues summary |
| `GET /api/jobs/{id}/impact` | **Impacted variants → fields**, each with source column/header, transform, Helix type, non-empty count, conversion-error count, sample values, alternatives |
| `GET /api/jobs/{id}/rows?page=&variant=` | Paged preview of transformed records |
| `GET /api/jobs/{id}/issues?severity=` | Validation issues (row, column, rule, message) |
| `PUT /api/jobs/{id}/overrides` | Replaces the overrides (retarget, exclude, value-map edit, template value); re-plans; returns the new hash |
| `GET /api/jobs/{id}/context` / `PUT …` | Context references (policyholder / product / issuer) and template values |
| `GET /api/schema/variants?entity=` and `/api/schema/variants/{v}/fields` | Options for overrides, from cached `/describe` |
| `POST /api/jobs/{id}/approve` | Body `{plan_hash}`. Rejected if the hash is stale or blocking issues exist |
| `GET /api/jobs/{id}/events` | SSE progress (rows done/failed, current variant) |
| `POST /api/jobs/{id}/pause` / `resume` / `abort` | Execution control |
| `GET /api/jobs/{id}/report.csv` | Final ledger and result per row |

---

## 8. UI (Svelte)

### 8.1 Screens

1. **Upload:** drag-and-drop or file picker (`.xlsx`, `.xlsm`), a rule-set selector (one option for now), a parse progress indicator, and parse errors shown inline (missing sheet, header mismatch).
2. **Review impacted tables and columns** (the main screen):
   - A summary bar: rows read, records to create per variant, blocking issues, warnings, and the plan hash.
   - **Grouped by target table (variant)**, which can be expanded. Each mapped column shows: source column letter and header → target field, Helix type, transform, sample values before and after, and counts of non-empty values and conversion errors.
   - **Override controls** for each column:
     - change the target variant and field (a searchable combobox that only lists fields with a compatible type);
     - exclude the column;
     - edit the value map (for enums and booleans);
     - restore the default.
     
     Overridden cells are highlighted, and a "changes" drawer lists every override against its default.
   - **System-supplied values** panel: template values and context references, which can be edited.
   - **Issues** tab: a filterable table, where clicking a row jumps to the column it concerns.
   - **Rows** tab: a paged preview of the records that will be written.
3. **Approve:** a confirmation screen showing:
   - totals per variant, split into **created** and **reused** (D3);
   - the target Helix host and the plan hash;
   - an **"Needs your attention"** list that must be acknowledged item by item before Approve is enabled (D4). It contains:
     - questionable matches (`Territory → loss_ratio_analysis`);
     - prerequisite chains (`Roof Shape` → location / dwelling_asset / wind mitigation);
     - placeholder template values (marked ⚠ in §6.2);
     - every override the user made.
     
     Each item has an "Override…" link back to the review screen.
   
   The user then confirms to submit.
4. **Progress & results:** a live progress bar per variant, a stream of failed rows with the reason, pause/resume/abort controls, and a CSV download when finished.

### 8.2 Theming and quality bar

- Design tokens as CSS custom properties: colour, surface, border, text, accent, success/warning/danger, spacing, radius, and type scale.
- **Light and dark themes.** The default follows `prefers-color-scheme`, with a manual toggle remembered in `localStorage`.
- Accessible: WCAG AA contrast in both themes, full keyboard support (combobox, dialogs, tables), visible focus rings, and status that never relies on colour alone (icon + text).
- Responsive down to tablet width. Wide tables scroll inside their own container.
- No component library is needed for iteration 1. A small set of in-house components (Button, Table, Combobox, Dialog, Badge, Toast, Tabs, FileDrop) keeps the bundle small and the theme consistent.

---

## 9. Testing strategy

| Layer | Tests |
|---|---|
| `rulegen` | Golden test against the real report: exactly 20 rules, the column letters and targets in §2, a stable SHA-256 |
| `rules` | Validation against a recorded `/describe` fixture: missing variant or field → marked stale; type mismatch → blocking |
| `excel` | A sanitised fixture workbook: duplicate headers, padded rows, `NULL` strings, mixed int/str cells, dates. Smoke test against the real HO workbook (1,394 rows) kept local and not committed |
| `transform` | Table-driven tests for each transform: `"1 to 4"`→4, `"5+"`→5, any other unit text → blocking issue, `SWR`→true, `Hip Roof`→hip, `HO6`→ho_6 |
| templates | Find-or-create: an existing key is reused (no create), a missing key is created, a reused record is never deleted during compensation, an existing policy → row skipped |
| `plan` | **Determinism:** same input gives the same hash byte-for-byte across runs and orderings. Overrides change the hash. A/BU conflict detection |
| `helix` | `httptest` fake server, mirroring the Python tests: token precedence, keys only sent to their own host, refusal of cross-host redirects, error wrapping |
| `execute` | Fake Helix with injected failures: partial row failure → compensating deletes verified by 404, idempotent retry creates no duplicates, pause and resume, an auth failure pauses the job |
| API | Handler tests for the job lifecycle, approval with a stale hash rejected |
| UI | Vitest component tests (combobox, value-map editor, theme toggle). Playwright end-to-end run: upload → override → approve → progress, against the Go server with a fake Helix |
| Live | An opt-in smoke command: about 3 rows against the remote Helix that cleans up after itself, like `run.py` |

---

## 10. Milestones

| # | Milestone | Deliverable | Exit criteria |
|---|---|---|---|
| M0 | Decisions | D1–D12 ✅ | Done |
| M1 | Skeleton | `migrator/` Go module, SvelteKit app, embed build, `make dev/build/test`, CI lint and tests | Binary serves an empty themed SPA plus `/api/health` |
| M2 | Rules | `rulegen`, committed HO rule set, validation against `/describe` | Golden test passes; startup reports the rule set as fresh |
| M3 | Excel + transform + plan | Parser, transforms, plan builder, plan hash, issues | Real HO file produces the expected impact summary deterministically |
| M4 | Helix client (Go) | Business-key lookup spike (§6.3), port of the Python client, describe/catalogue cache, find-or-create | All client tests pass; read-only calls work live; lookup strategy chosen |
| M5 | API + jobs | SQLite store, job state machine, all §7 endpoints except execution | API end-to-end with fixtures |
| M6 | UI | Upload, Review/Override, Approve, Progress screens, both themes | Playwright end-to-end passes |
| M7 | Execution | Ordered writes, idempotency, compensation, SSE, report | Fault-injection tests pass; live smoke test writes and cleans up 3 rows |
| M8 | Hardening | Size limits, retention, logs without cell values, README / runbook | Ready for an internal pilot |

---

## 11. Decisions and open questions

### Decided (2026-10-07)

| # | Decision |
|---|---|
| D1 | `Number of Units`: `"1 to 4"` → 4, `"5+"` → 5 |
| D2 | Work with the Helix schema **as it is**. Variants without a policy link rely on the ledger. Every schema gap is recorded in §13 for the next iteration |
| D3 | Prerequisite records (issuer, product, policyholder, location, dwelling asset, wind mitigation) are **found by business key, or created if they don't exist** (§6.2) |
| D4 | `Territory` and `Roof Shape` stay. They and all placeholder values are **flagged for the user at approval**, who can override them |
| D5 | Excel column **A** is the policy number and the business key for each row |

| D6 | Required fields the sheet doesn't have get **generated placeholder values** (deterministic, tagged, flagged at approval). Optional fields are left empty. Fixed properly next iteration (S11) |
| D7 | `program_code = safepoint`. `HO3` = owner-occupied home and `HO6` = condo. The model has no list of allowed dwelling types, so `dwelling_type` gets a placeholder this iteration (S12) |
| D8 | Sinkhole: `Yes → sinkhole_loss`, `No → catastrophic_ground_cover_collapse_only` |
| D9 | Wind-mitigation `inspection_date` and `oir_b1_1802_form_revision` get placeholders this iteration (S13) |
| D10 | Write `policy` only. The `policy_term` → `policy_revision` → `policy_version` chain is deferred (S14) |
| D11 | **Single user, local, demo-only.** No authentication. The server listens on `127.0.0.1` only. Target: a working demo **by end of day 2026-10-07** (§14) |
| D12 | Re-runs **update** existing records with the sheet-sourced fields only (§6.2, item 8) |

### Still open (non-blocking)

1. **Column BU (second Policy Number).** It is empty in the HO file. Default: ignore it unless it has a value; if it differs from A, raise a blocking conflict.

---

## 12. Risks

| Risk | Mitigation |
|---|---|
| The Helix model changes (bundle version moves) | Startup check against `/describe`; rule set marked stale; approval blocked |
| No cross-record transactions in Helix | Per-row compensation, idempotency keys, ledger, resumable jobs |
| Generated placeholder values look like real data in Helix | `GEN-` prefix, `generated=true` in the ledger, listed in the report and the attention panel; cleanup command (§14) |
| Demo writes land on the shared remote Helix | Row limit and dry run by default; a cleanup command deletes everything the ledger says this tool created (§14) |
| Orphan records in variants that can't link to a policy | Ledger link (D2); schema changes listed in §13 |
| No business-key lookup API | §6.3 spike; fallback is ledger plus a cached paged scan, which slows down as Helix fills |
| Placeholder values written as if they were real (inspection date, form revision) | Attention flag at approval; the values come from the template file, so they can be fixed in one place |
| Report target choices are only name matches | Primary targets pinned by a person in the committed rule set; user overrides at review |
| Confidential workbooks | Temporary storage with retention, no cell values in logs, credentials only on the server |
| Throughput (~15k creates) | Worker pool, rate limiting, SSE progress; use a batch endpoint if Helix exposes one (`batch` appears in `/metrics`, but there is no documented route yet) |

---

## 13. Schema change backlog (for the next iteration)

D2 means iteration 1 works with the Helix schema as it is. These are the gaps found so far, to raise with the model owners. The tool also keeps this list in a running state: each time a rule needs a workaround, an entry is added here.

| # | Gap | Effect in iteration 1 | Proposed change |
|---|---|---|---|
| S1 | `dwelling.property.us.personal` has no reference to a policy or policy version | Linked through the ledger only | Add `policy_version_reference` (or `insured_object_reference`) |
| S2 | `section_icoverages.property.us.personal` has no reference | Ledger only | Add `policy_version_reference` |
| S3 | `line.property.us.personal` has no reference | Ledger only | Add `policy_version_reference` |
| S4 | `location_address.property.us.personal` has no reference | Ledger only | Add `location_reference` |
| S5 | `loss_ratio_analysis.territory` is a portfolio metric, not a rating territory | Flagged at approval | Add a rating-territory field on policy, location or dwelling, then move the rule |
| S6 | `dwelling.number_of_units` is an integer, but raters use bands (`1 to 4`, `5+`) | Value map D1 (4 / 5) loses the band's meaning | Add a band enum, or keep the integer and add `number_of_units_band` |
| S7 | `wind_mitigation_verification` requires inspection fields the rater doesn't have | Placeholder values, flagged | Make inspection fields optional, or give rater imports a separate home for rating characteristics |
| S8 | No lookup by business key in the API (to be confirmed in §6.3) | Ledger plus paged scan | Add `GET /records/{variant}?key=…` |
| S9 | `dwelling` and `dwelling_asset` overlap (`number_of_units`, construction year/type) | Primary target picked by hand in the rule set | Settle which entity owns rating characteristics |
| S10 | Rater mapping reports were generated against an old DDL (`core_all`, `jur_*`, `lob_*`) and, for Mobile HO and WO, a foreign schema | `rulegen` translates the schema names | Regenerate the reports against the current `/model/ddl` |
| S11 | Required fields without a source in the rater (issuer and product identity, `issue_date`, `address_text`, `is_under_renovation`, inspection fields, …) | Generated placeholders (D6) | Agree real sources or defaults for each; make some optional in the model |
| S12 | `dwelling_asset.dwelling_type` is free text with no list of allowed values | Placeholder (D7) | Add an enum (e.g. `owner_occupied_home`, `condominium_unit`, …) and map `HO3`/`HO6` to it |
| S13 | Wind-mitigation `inspection_date` and `oir_b1_1802_form_revision` are not in the rater | Placeholders (D9) | Source them from inspection data, or make them optional |
| S14 | No `policy_term` / `policy_revision` / `policy_version` written, although `Risk Term` and `Effective Date` imply them | Deferred (D10) | Build the full chain; attach coverages and dwelling to `policy_version` |
| S15 | No authentication or audit of who approved | Demo-only (D11) | SSO, per-user approval audit, roles |

---

## 14. Demo cut — end of day 2026-10-07 (D11)

The full plan above is the target. For today's demo, the scope is cut to the shortest path that still shows the whole flow end to end.

### In scope today

| Area | Demo version |
|---|---|
| Rules | `rulegen` parses the HO report and writes `rulesets/manatee_fl_select_ho_12_1_25.json` (20 rules, with the primary targets pinned as in §2). Golden test: 20 rules |
| Templates | `templates.json` as in §6.2, with generated placeholders (D6) |
| Excel | excelize; `Policy Data` sheet; columns addressed by letter; stop at the first empty column A |
| Transform + validate | All transforms in §2; type, enum and length checks; issues are blocking or warning |
| Plan | Deterministic plan and plan hash; approval tied to the hash |
| Review UI | Upload → impacted variants and fields (before/after samples, counts) → overrides (retarget variant/field, exclude, edit value map, edit template value) → issues |
| Approval UI | Totals (create / update / unchanged, per variant), the "Needs your attention" list with acknowledgements, a **row limit** (default 5) and a **dry run** toggle |
| Execution | Go Helix client (port of the Python client), find-or-create, update on re-run, idempotency keys, per-row compensation, worker pool of 4. Progress by polling `GET /api/jobs/{id}` every second |
| Ledger | One JSON file per environment (`data/ledger.json`, protected by a mutex) instead of SQLite |
| Cleanup | `migrator cleanup` deletes, children first, every record the ledger marks as `created`, then checks for 404 |
| Theme | Light and dark tokens, a toggle, and readable tables |
| Tests | Go unit tests for rulegen, transforms, plan determinism, and the executor against an `httptest` fake Helix |

### Deferred until after the demo

SSE progress, pause/resume, SQLite, business-key lookup in Helix beyond the ledger, startup staleness blocking (the demo only shows a warning), Vitest/Playwright, upload retention, and authentication.

**Demo fallback for finding records:** the ledger first. If a create fails because the key already exists, the tool runs one paged `/list` scan for that entity to find the ID, then switches to an update.

### Order of work (≈ 8 hours)

1. **Skeleton, rules, transforms and their tests:** about 1.5 h. Exit: 20-rule golden test is green.
2. **Excel parsing, plan and JSON API:** about 1.5 h. Exit: the real HO file gives an impact summary through `curl`.
3. **Helix client, executor, ledger and cleanup:** about 2 h. Exit: the fake-Helix tests pass, and a live dry run of 5 rows works.
4. **Svelte UI (four screens) and theme:** about 2.5 h. Exit: the full flow works in the browser.
5. **Live run of 5 rows, re-run to show updates, cleanup, demo script:** about 0.5 h.

**Main schedule risk:** step 4. If time runs short, the review table will offer override by editing only, without searchable comboboxes.

