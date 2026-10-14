# Prompt: build the Duck Creek Billing screens in the Safe Point workspace

> Hand this whole file to the coding agent. Its working directory is the repo root (`dataMigration`).

---

## Role and goal

You are working in the `dataMigration` repo, a Go server (`cmd/migrator`, `internal/…`) with a plain Svelte 5 + Vite UI (`web/`). It already contains a Rater → Helix migrator and a durable local **PersonalHome quote application** that copies the Duck Creek Policy screens.

Your task is to add a **Billing workspace** that copies **every screen** in the Duck Creek Billing walkthrough. It must look and behave the same way: the same navigation, fields, dropdown values, buttons, validation and confirmation messages, and the same business rules. Data is stored locally and durably, the same way quotes are.

## Sources of truth (read these first, in order)

1. **`docs/BILLING_UI_WORKFLOW.md`**, the analysed spec: navigation map (§1), objects (§2), 18 workflows (§3), code lists (§4), reports (§5), business rules (§6), data problems (§7) and gaps (§8). `#NNN` in that file is a screenshot number.
2. **`~/Documents/Work documents/Billing/Safepoint Billing UI Workflow Documentation.docx`**: 117 screenshots, the visual reference for layout, column order, labels and wording. To get them in order:
   ```bash
   mkdir -p /tmp/billing-docx && cd /tmp/billing-docx && cp "$HOME/Documents/Work documents/Billing/Safepoint Billing UI Workflow Documentation.docx" doc.docx && unzip -oq doc.docx -d x
   ```
   Screenshot `#NNN` is the NNN-th `<a:blip r:embed=…>` in `x/word/document.xml`. Resolve the embed id through `x/word/_rels/document.xml.rels` to find the file under `x/media/`. Look at the screenshot for every screen you build. Where the spec and a screenshot disagree, **the screenshot wins**; note the difference in your summary.
3. **Existing code to copy the patterns of**:
   - `internal/quote/` (`quote.go`, `http.go`): file-backed `Store` with a mutex, atomic writes, optimistic `version` checks; `Register(mux, store, hook)`; `ValidationError` → 422 with per-field errors, `ErrNotFound` → 404, conflicts → 409; `decode` with `DisallowUnknownFields` and a 1 MB limit.
   - `cmd/migrator/main.go`: `serve` flags (`-offline`, `-quote-dir`). Billing must work in `-offline` mode.
   - `web/src/App.svelte`: hash routing (`viewOf`, `switchView`, the `quote-before-leave` guard), the app rail and the top bar.
   - `web/src/lib/personalhome/`: `QuoteApplication.svelte`, `QuoteSidebar.svelte`, `QuoteField.svelte`, `WorkspaceIcon.svelte`, and **`InstallmentSchedule.svelte` / `PaymentDetails.svelte`**, which already copy the billing installment grid. Reuse or generalise them; don't fork them.
   - `web/src/app.css` design tokens (`--accent`, `--accent-soft`, `--surface`, `--surface-2`, `--border`, `--text-muted`, `--danger`), plus light/dark theming through `ThemeToggle`.
   - Tests: `go test ./...`, `web/tests/*.test.mjs` (node:test), and `npm run check` (svelte-check).

## How to work (required)

1. **Plan first; don't write code yet.** Reply with:
   - a screen inventory table: screen → spec section → screenshot numbers → route → component → backend endpoint(s);
   - the data model (Go types and JSON shape) and the store layout;
   - the phase breakdown (see below);
   - **numbered open decisions, each with your proposed default**.
2. Wait for explicit approval. Write code only after the user says go. Any decision the user leaves unanswered is recorded as a backlog item, not silently decided.
3. Build one phase at a time. At the end of each phase run `make check` (Go vet and tests, svelte-check, node tests). Then open the app (`make quote-run` or the `migrator` config in `.claude/launch.json`) and compare each new screen with its screenshots. Report results honestly, including failures.
4. Don't commit unless asked.

## Decisions with proposed defaults (confirm or change these in the plan)

- **D-B1 Scope:** a local functional copy (create, read, update, state changes) with no real money movement and no Helix writes. *Default: yes.* Helix sync of billing is a later phase, modelled on `internal/quotesync`.
- **D-B2 Card payments:** copy the Global Payments/Heartland modal (#067–#070) as a **simulated** gateway. Never store a full card number or CVV. Keep only the last 4 digits and the name. The 2.7% processing fee is calculated and shown, and only the base amount posts to billing. Accept the published test card numbers only; reject anything else with a clear "simulated gateway" message. *Default: yes.*
- **D-B3 System date:** a workspace-wide **Set System Date** control (footer, as in the screenshots), stored server-side. Every "today" default (entry dates, hold start, effective dates) and every time-based rule uses it. Provide a "run due processes" action that generates invoices and moves installments to past due, so time-based screens can be shown. *Default: yes.*
- **D-B4 Seed data:** a reproducible seed command or script that loads the accounts, policies, payments, suspense items, error-queue rows, batches, hold events, tasks and parties from the screenshots (CLAD3057412, CLAD3057414, STXW3081544, SFLH3074412, STXH3074413, the unidentified payments, and so on). Use the screenshots' values. Replace real-looking personal data (emails, street addresses) with obvious test values. *Default: yes.*
- **D-B5 Link to quotes:** when a quote is submitted or bound, create its billing account and term (account number = policy number, §6 rule 1) using `Billing.BilledTo`, `Billing.PaymentPlan` and `Billing.InvoiceDueDate` from `internal/quote/schema.json`. *Default: build this behind an explicit "Create billing account" action in phase 2, not automatically.*
- **D-B6 Reports:** render the 9 reports from live store data, with CSV and PDF/print export. Show the other export formats (Word, Excel, …) in the dropdown but disable them. *Default: yes.*
- **D-B7 Data problems in §7 (O1–O11):** seed enough rows to show the Error Queue, Batches Awaiting Deposit, Pending Authorization and Cash In Suspense. Don't copy defects such as duplicate rows or null-reference errors as behaviour. *Default: yes.*
- **D-B8 Screens the doc doesn't show (§8):** build only a placeholder screen with a "not yet specified" note, and list each in the summary. *Default: yes.*

## Architecture constraints

- **Backend:** a new package `internal/billing/` (store, domain logic, `http.go` with `Register(mux, store)`), wired in `cmd/migrator/main.go` with a `-billing-dir` flag (default `data/billing`). Same error mapping and decode rules as `internal/quote`. Money is integer cents; never use float64. Dates are `YYYY-MM-DD` strings in JSON and shown as `M/D/YYYY`.
- **Routes:** REST under `/api/billing/…` (accounts, terms, payments, unidentified payments, suspense, disbursements, holds, hold events, tasks, parties, batches, errors, reports, system date). Every state-changing call takes the record `version` and rejects stale versions with 409.
- **Frontend:** `web/src/lib/billing/`, Svelte 5 runes (`$state`, `$derived`, `$props`, `$effect`), the existing CSS tokens, `WorkspaceIcon`. Add a **Billing** item to the app rail and a `#billing/...` hash route family in `App.svelte`. Keep a top nav of **Home · Administration · Batch Cash Management**, a left menu on account and term pages, and collapsible side panels (Account Ledger, Account Details, Payee, Payor, Term Details, Term Ledger, Agent). Use deep links such as `#billing/account/CLAD3057414/suspense`.
- **Accessibility and responsiveness** match the quote app: labelled inputs, `aria-expanded` on tree rows and panels, keyboard reachable, usable at 375 px width (grids scroll sideways inside their container, not the page).
- **No new runtime dependencies** unless you justify one in the plan.

## Business rules to enforce (from spec §6, plus what the screenshots show)

1. One billing account per policy; account number = policy number.
2. Unallocated or overpaid amounts go to account-level suspense. Show the note "Any remaining balance will stay on hold in suspense at the account level."
3. A disbursement needs a Disbursement Reason before **Refund** is enabled. It is created as *Pending Authorization*. Amounts at or under a configurable threshold are auto-authorized by `System`; above it they wait for an approver. Ledger › Pending Disbursements updates.
4. The refund method is always Check, whatever the source payment method was.
5. Card payments: base + 2.7% fee shown in the gateway; billing posts the base only.
6. Transfer To Collections is blocked unless the term outstanding balance is > 0. Use the exact message from #057.
7. Holds start on the system date and cannot be back-dated. A blank end date means "Indefinite". Resume asks *Adjust due dates / Do not adjust due dates*, and Adjust really moves the open installment due dates.
8. Party data is effective-dated. **Correct** edits the current row in place and keeps its effective date. **Replace** end-dates the current row and creates a new one effective from the system date. Other Names also have **Remove**.
9. Unidentified payment entry needs at least one identifier. It can be applied to an account, re-referenced, or reversed (`RP-Reverse Payment`).
10. Payment reversal sets status `Reversed`, writes a `Reverse Payment` activity, and re-opens what the payment had paid.
11. Primary buttons stay disabled until their required inputs are valid (Create batch, Place Hold, Refund, Import Statements, Search).
12. Every state change writes an Account Activity entry with the wording from the screenshots, e.g. "Check payment for $981.00 received. Check Number: Receipt date: 10/12/2026", "Disbursement for $10.00 was submitted for authorization. Payee: TEST", "Account hold set. Hold type: … Reason: … Beginning: … Ending: Indefinite".
13. Paid In Full Discount shows the "configuration has not been established" message until a PIF configuration exists. Add a simple admin setting for it.
14. Error-queue items keep their payload. **Resubmit** shows editable payload fields and re-runs the job against the store. Disbursement Interface **Start Service** refuses to queue a second request while one is pending (the duplicate risk in #047).

## Screen inventory to deliver (all 117 screenshots must map to one of these)

**Home and payments**
- Home dashboard: Activities, Recently Accessed, Reports, Find a Billing Account, Notifications, Tasks (#001, #002, #030)
- Header quick search (#001)
- Search For A Payment → Payment Details with Allocations (#003–#005)
- Enter A Single Payment: known account → Search Results → Make A Payment (Pay To Account; Pay Specific Policies grid) → confirmation (#006–#012)
- Enter A Single Payment: unidentified (#013)
- Unidentified Payments list with filters → Unidentified Payment detail: apply, Update Reference, Reverse (#014–#017)

**New account and party search**
- Create New Account (#018–#019)
- Search for a Person or Place: Agency/Person × Name and Address / License / Phone / Membership ID / Other (#020–#025, #116)
- Party search results: View / Select / 360, Revise Search / Add New (#026–#027, #117)
- Import Agent Statement (#028–#029)

**Reports**
- Reporting Home: Report History, Export, Reports dropdowns (#032–#034)
- The 9 reports with exact columns from spec §5 (#032–#041)

**Administration**
- Admin landing (#042)
- Error Queue with filters, Error Details modal, Resubmission modal (#043–#045)
- Manage Hold Events, plus the Create Hold Event form (not in the doc; placeholder per D-B8) (#046)
- Disbursement Interface Service (#047)
- Billing Transactions search (#048)

**Batch Cash Management**
- Batches Awaiting Deposit (#049)
- Start A New Payment Batch (#050)
- Processed Batches with Reverse (#051)

**Account**
- Account page: summary tiles, Recent Account Activity, Policies/Arrangements, side panels (#031, #053–#054, #063, #065, #073, #082, #089, #115)
- Installment Schedule, with Schedule History (#062)
- Select Billing Account Options (#064)
- Payment Processing submenu: Make A Payment (check and card), Suspense Allocation, Payments List (#010, #066, #071–#072)
- Card gateway modal: card details, summary with fee, terms Accept/Decline, Payment Successful (#067–#070)
- Items In Suspense with action menu → Disburse Funds → submitted for authorization (#074–#077)
- EFT Maintenance: Scheduled EFT Transactions (#078)
- Hold Activity: individual or area event → banner → Edit Hold / Resume (#079–#083)
- Account Tasks → New Task → Tasks Available for Pickup, More Options filters (#084–#087)
- Billing Transactions → Add Miscellaneous Billing Item (#088; the form itself is a placeholder per D-B8)

**Policy term**
- Policy Term Summary with Policy Term Actions and term side panels (#055)
- Transfer Policy Term (#056)
- Transfer To Collections, including the blocked state (#057)
- Term Plan Setup (#058)
- Renewal Plan Setup (#059)
- Hold Billing for Policy Term (#060)
- Paid In Full Discount (#061)
- Send To Internal Collections (placeholder per D-B8)

**Party (People & Places)**
- People & Places Details with all collapsible sections (#090, #093, #098, #101, #104, #107, #110)
- Correct / Replace Name & Contact (#091–#092)
- Addresses: list, add, correct, replace, associations (#094–#097)
- Phones: list, add (#099–#100)
- Email Addresses: list, add, edit (#102–#103)
- Locations: list, add (#105–#106)
- Demographics (#108)
- Other Names: correct, replace, remove (#109)
- Licenses: list, add (#111–#112)
- Memberships: list, add (#113–#114)
- Change Payee / Change Payor (#115–#117)

Use the dropdown values from spec §4 **exactly**. Put them in one shared code-list module (Go and TS) so they don't drift apart.

## Suggested phases

1. **Foundation:** `internal/billing` store and model, system date, seed, Billing rail entry and routing, Home, account search, Account page (read-only), Policy Term Summary (read-only), Installment Schedule.
2. **Cash in:** single payment (check and pay-specific allocation), simulated card gateway, Payments List, Search For A Payment, Payment Details, reversal, unidentified payments, create account and party search, create account from a quote (D-B5).
3. **Cash out and control:** suspense, suspense allocation, disbursement with authorization, holds (account, term, area events, resume), tasks, account settings, term plan/renewal plan, transfers, collections, PIF.
4. **Party:** People & Places, Correct/Replace effective-dating, all sub-entities, Change Payee/Payor.
5. **Back office:** Batch Cash Management, Administration (error queue, resubmit, hold events, disbursement service, billing transactions), Import Statements, the 9 reports and export.

## Tests required

- **Go:** table-driven tests for every rule in "Business rules" above. Cover at least: allocation and suspense math in cents, reversal re-opening items, disbursement authorization threshold, hold resume adjusting due dates, Correct vs Replace effective dating, collections block, version conflicts (409), validation (422 with field keys), and store persistence across reopen.
- **Web (`web/tests/*.test.mjs`):** code-list parity between Go and TS. Every route in the inventory resolves to a component. Every screenshot `#001–#117` is mapped by exactly one screen (keep the mapping in a JSON file the test reads).
- `make check` must pass at the end of each phase.

## Definition of done

- Every screen in the inventory exists and is reachable through the same navigation path as in the screenshots. Labels, column order and messages match the screenshots.
- All business rules are enforced on the server, not only in the UI.
- Seeded data reproduces the screenshots' scenarios.
- `make check` passes, and screens are checked visually against the screenshots in light and dark themes and at 375 px.
- Final summary lists: screens built; placeholders (D-B8); places where you deviated from the spec or screenshots, and why; decisions still open.
